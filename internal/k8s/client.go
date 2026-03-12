package k8s

import (
	"context"
	"fmt"

	"gopkg.in/yaml.v3"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	memory "k8s.io/client-go/discovery/cached"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
)

type Client struct {
	dynamic   dynamic.Interface
	discovery *discovery.DiscoveryClient
	mapper    meta.RESTMapper // resolves "pods" → GVR
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
	return &Client{dynamic: dyn, discovery: disc, mapper: mapper}, nil
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

func formatList(list []output) (string, error) {
	yamlBytes, err := yaml.Marshal(list)
	if err != nil {
		return "", err
	}
	//fmt.Fprintf(os.Stderr, string(yamlBytes))
	return string(yamlBytes), nil
}

type output struct {
	Name      string `yaml:"name"`
	Namespace string `yaml:"namespace,omitempty"`
	Status    string `yaml:"status,omitempty"`
	Ready     string `yaml:"ready,omitempty"`
	Created   string `yaml:"created,omitempty"`
}

func normaliseList(list *unstructured.UnstructuredList) []output {
	result := make([]output, 0)
	for _, item := range list.Items {
		status, _, _ := unstructured.NestedString(item.Object, "status", "phase")
		containersStatus, _ := getContainerInfo(item)
		result = append(result, output{
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
