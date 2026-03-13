package k8s

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"gopkg.in/yaml.v3"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	memory "k8s.io/client-go/discovery/cached"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
)

type Client struct {
	dynamic   dynamic.Interface
	discovery *discovery.DiscoveryClient
	mapper    meta.RESTMapper // resolves "pods" → GVR
	clientSet *kubernetes.Clientset
}

func NewClient(config *rest.Config) (*Client, error) {
	dyn, err := dynamic.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	disc, err := discovery.NewDiscoveryClientForConfig(config)
	if err != nil {
		return nil, err
	}
	mapper := restmapper.NewDeferredDiscoveryRESTMapper(
		memory.NewMemCacheClient(disc),
	)
	clientSet, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	return &Client{dynamic: dyn, discovery: disc, mapper: mapper, clientSet: clientSet}, nil
}

type listResourcesOutput struct {
	Name      string `yaml:"name"`
	Namespace string `yaml:"namespace,omitempty"`
	Status    string `yaml:"status,omitempty"`
	Ready     string `yaml:"ready,omitempty"`
	Created   string `yaml:"created,omitempty"`
}

type apiResourcesOutput struct {
	Name       string `yaml:"name"`
	Kind       string `yaml:"kind"`
	Group      string `yaml:"group"`
	Namespaced bool   `yaml:"namespaced"`
}

func (c *Client) resolveGVR(resource string) (schema.GroupVersionResource, bool, error) {
	partialResource := schema.GroupVersionResource{Resource: resource}
	gvr, err := c.mapper.ResourceFor(partialResource)
	if err != nil {
		return schema.GroupVersionResource{}, false, err
	}
	gvks, err := c.mapper.KindsFor(gvr)
	if err != nil || len(gvks) == 0 {
		return gvr, true, nil
	}
	mapping, err := c.mapper.RESTMapping(gvks[0].GroupKind(), gvks[0].Version)
	if err != nil {
		return gvr, true, nil
	}
	return gvr, mapping.Scope.Name() == meta.RESTScopeNameNamespace, nil
}

func (c *Client) ListResources(ctx context.Context, resource, namespace string) (*unstructured.UnstructuredList, error) {
	gvr, namespaced, err := c.resolveGVR(resource)
	if err != nil {
		return nil, err
	}
	if namespaced && namespace != "" {
		return c.dynamic.Resource(gvr).Namespace(namespace).List(ctx, metav1.ListOptions{})
	}
	return c.dynamic.Resource(gvr).List(ctx, metav1.ListOptions{})
}

func (c *Client) GetResource(ctx context.Context, name string, resource string, namespace string) (*unstructured.Unstructured, error) {
	gvr, namespaced, err := c.resolveGVR(resource)
	if err != nil {
		return nil, err
	}

	if namespaced && namespace != "" {
		return c.dynamic.Resource(gvr).Namespace(namespace).Get(ctx, name, metav1.GetOptions{})
	}
	return c.dynamic.Resource(gvr).Get(ctx, name, metav1.GetOptions{})
}

func formatResourcesList(list []listResourcesOutput) (string, error) {
	yamlBytes, err := yaml.Marshal(list)
	if err != nil {
		return "", err
	}
	//fmt.Fprintf(os.Stderr, string(yamlBytes))
	return string(yamlBytes), nil
}

func normaliseList(list *unstructured.UnstructuredList) []listResourcesOutput {
	result := make([]listResourcesOutput, 0)
	for _, item := range list.Items {
		status, _, _ := unstructured.NestedString(item.Object, "status", "phase")
		containersStatus, _ := getContainerInfo(item)
		result = append(result, listResourcesOutput{
			Name:      item.GetName(),
			Namespace: item.GetNamespace(),
			Status:    status,
			Ready:     containersStatus,
			Created:   item.GetCreationTimestamp().UTC().Format("2006-01-02"),
		})
	}
	return result
}

func getContainerInfo(item unstructured.Unstructured) (string, error) {
	if item.GetKind() != "Pod" {
		return "", nil
	}

	containerStatuses, found, err := unstructured.NestedSlice(item.Object, "status", "containerStatuses")
	if err != nil {
		// TODO: Do I really need to return err?
		return "", err
	}
	if !found {
		return "", nil
	}
	total := len(containerStatuses)
	ready := 0
	for _, cs := range containerStatuses {
		container, ok := cs.(map[string]any)
		if !ok {
			continue
		}
		if isReady, ok := container["ready"].(bool); ok && isReady {
			ready++
		}
	}

	return fmt.Sprintf("%v/%v", ready, total), nil
}

func (c *Client) getLogs(podName string, namespace string, tailLines int64) (string, error) {
	request := c.clientSet.CoreV1().Pods(namespace).GetLogs(podName, &v1.PodLogOptions{TailLines: &tailLines})
	podLogs, err := request.Stream(context.TODO())
	if err != nil {
		return "", err
	}
	defer podLogs.Close()

	buf := new(bytes.Buffer)
	_, err = io.Copy(buf, podLogs)
	if err != nil {
		return "", errors.New("error in copy from podLogs to buf")
	}
	str := buf.String()

	return str, nil
}

var skipGroups = map[string]bool{
	"authentication.k8s.io":  true,
	"authorization.k8s.io":   true,
	"apiregistration.k8s.io": true, // apiservices
	"coordination.k8s.io":    true, // leases (internal leader election)
}

func (c *Client) get() (string, error) {
	resources, err := c.discovery.ServerPreferredResources()
	if err != nil {
		return "", err
	}
	result := make([]apiResourcesOutput, 0)
	for _, apiGroup := range resources {
		for _, r := range apiGroup.APIResources {
			if strings.Contains(r.Name, "/") {
				// Any resource name containing / is a subresource — pods/log, pods/exec, pods/status, deployments/scale etc.
				// We don't need them
				continue
			}
			if skipGroups[apiGroup.GroupVersion] {
				// no need to return them as well
				continue
			}
			group := r.Group
			if group == "" {
				group = apiGroup.GroupVersion
			}
			result = append(result, apiResourcesOutput{
				Name:       r.Name,
				Kind:       r.Kind,
				Group:      r.Group,
				Namespaced: r.Namespaced,
			})
		}
	}

	return formatApiList(result)
}

func formatApiList(list []apiResourcesOutput) (string, error) {
	yamlBytes, err := yaml.Marshal(list)
	if err != nil {
		return "", err
	}
	//fmt.Fprintf(os.Stderr, string(yamlBytes))
	return string(yamlBytes), nil
}
