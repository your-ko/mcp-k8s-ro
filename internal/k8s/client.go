package k8s

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	memory "k8s.io/client-go/discovery/cached"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
	metrics "k8s.io/metrics/pkg/client/clientset/versioned"
)

type Client struct {
	dynamic      dynamic.Interface
	discovery    *discovery.DiscoveryClient
	mapper       meta.RESTMapper // resolves "pods" → GVR
	clientSet    *kubernetes.Clientset
	contextName  string
	clusterName  string
	metricClient *metrics.Clientset
}

func NewClient(config *rest.Config, contextName string, clusterName string) (*Client, error) {
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
	metricClient, err := metrics.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	return &Client{
		dynamic:      dyn,
		discovery:    disc,
		mapper:       mapper,
		clientSet:    clientSet,
		contextName:  contextName,
		clusterName:  clusterName,
		metricClient: metricClient,
	}, nil
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

func (c *Client) ListResources(ctx context.Context, resource, namespace string) (string, error) {
	gvr, namespaced, err := c.resolveGVR(resource)
	if err != nil {
		return "", err
	}
	var list *unstructured.UnstructuredList
	if namespaced && namespace != "" {
		list, err = c.dynamic.Resource(gvr).Namespace(namespace).List(ctx, metav1.ListOptions{})
	} else {
		list, err = c.dynamic.Resource(gvr).List(ctx, metav1.ListOptions{})
	}
	if err != nil {
		return "", err
	}
	return formatResourcesList(normaliseList(list), c.Header())
}

func (c *Client) Header() string {
	return fmt.Sprintf("# context: %s | cluster: %s\n", c.contextName, c.clusterName)
}

func (c *Client) ContextSummary() string {
	return fmt.Sprintf("%s (%s)", c.contextName, c.clusterName)
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
			Created:   item.GetCreationTimestamp().UTC().Format(time.DateTime),
		})
	}
	return result
}

func (c *Client) GetResource(ctx context.Context, name string, resource string, namespace string) (string, error) {
	gvr, namespaced, err := c.resolveGVR(resource)
	if err != nil {
		return "", err
	}

	var res *unstructured.Unstructured
	if namespaced && namespace != "" {
		res, err = c.dynamic.Resource(gvr).Namespace(namespace).Get(ctx, name, metav1.GetOptions{})
	} else {
		res, err = c.dynamic.Resource(gvr).Get(ctx, name, metav1.GetOptions{})
	}
	if err != nil {
		return "", err
	}
	yamlBytes, err := yaml.Marshal(res.Object)
	if err != nil {
		return "", err
	}
	return c.Header() + string(yamlBytes), nil
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

func (c *Client) GetLogs(podName string, namespace string, tailLines int64) (string, error) {
	opts := &v1.PodLogOptions{}
	if tailLines > 0 {
		opts.TailLines = &tailLines
	}
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

	return c.Header() + str, nil
}

var skipGroups = map[string]bool{
	"authentication.k8s.io":  true,
	"authorization.k8s.io":   true,
	"apiregistration.k8s.io": true, // apiservices
	"coordination.k8s.io":    true, // leases (internal leader election)
}

func (c *Client) ListApiResources(groupFilter string) (string, error) {
	resources, err := c.discovery.ServerPreferredResources()
	if err != nil {
		return "", err
	}
	result := make([]apiResourcesOutput, 0)
	for _, apiGroup := range resources {
		for _, r := range apiGroup.APIResources {
			group := r.Group
			if groupFilter != "" && group != groupFilter {
				// use group filter
				continue
			}
			if strings.Contains(r.Name, "/") {
				// Any resource name containing / is a subresource — pods/log, pods/exec, pods/status, deployments/scale etc.
				// We don't need them
				continue
			}
			if group == "" {
				group = apiGroup.GroupVersion
			}
			groupVersion := strings.Split(apiGroup.GroupVersion, "/")[0]
			if skipGroups[groupVersion] {
				// no need to return them as well
				continue
			}
			//fmt.Fprintln(os.Stderr, apiGroup.GroupVersion)

			result = append(result, apiResourcesOutput{
				Name:       r.Name,
				Kind:       r.Kind,
				Group:      group,
				Namespaced: r.Namespaced,
			})
		}
	}

	return formatApiList(result, c.Header())
}

func (c *Client) GetEvents(namespace string, limit int64) (string, error) {
	opts := metav1.ListOptions{}
	if limit > 0 {
		opts.Limit = limit
	}
	list, err := c.clientSet.EventsV1().Events(namespace).List(context.TODO(), opts)
	if err != nil {
		return "", err
	}

	result := make([]eventOutput, 0)
	for _, event := range list.Items {
		eventCount := int32(1)
		if event.Series != nil {
			eventCount = event.Series.Count
		} else if event.DeprecatedCount > 0 {
			eventCount = event.DeprecatedCount
		}
		lastTime := event.EventTime.Time
		if event.Series != nil {
			lastTime = event.Series.LastObservedTime.Time
		} else if !event.DeprecatedLastTimestamp.IsZero() {
			lastTime = event.DeprecatedLastTimestamp.Time
		}
		firstTime := event.EventTime.Time
		if firstTime.IsZero() {
			firstTime = event.DeprecatedFirstTimestamp.Time
		}
		result = append(result, eventOutput{
			Name:      event.Regarding.Name,
			Namespace: event.Regarding.Namespace,
			Kind:      event.Regarding.Kind,
			Reason:    event.Reason,
			Message:   event.Note,
			Type:      event.Type,
			Count:     eventCount,
			FirstTime: yamlTime{MicroTime: metav1.MicroTime{Time: firstTime}},
			LastTime:  yamlTime{MicroTime: metav1.MicroTime{Time: lastTime}},
		})
	}

	sort.SliceStable(result, func(left, right int) bool {
		return !result[left].LastTime.Before(&result[right].LastTime.MicroTime)
	})

	return formatEventList(result, c.Header())
}

func (c *Client) TopPods(namespace string) (string, error) {
	podMetricsList, err := c.metricClient.MetricsV1beta1().PodMetricses(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return "", err
	}
	result := make([]podTopOutput, 0, len(podMetricsList.Items))
	for _, podMetrics := range podMetricsList.Items {
		totalCPU := resource.NewQuantity(0, resource.DecimalSI)
		totalMemory := resource.NewQuantity(0, resource.BinarySI)
		containers := make([]containerTopOutput, 0, len(podMetrics.Containers))
		for _, cMetrics := range podMetrics.Containers {
			cpu := cMetrics.Usage[v1.ResourceCPU]
			mem := cMetrics.Usage[v1.ResourceMemory]
			totalCPU.Add(cpu)
			totalMemory.Add(mem)
			containers = append(containers, containerTopOutput{
				Name:   cMetrics.Name,
				CPU:    fmt.Sprintf("%dm", cpu.MilliValue()),
				Memory: fmt.Sprintf("%dMi", mem.Value()/1024/1024),
			})
		}
		result = append(result, podTopOutput{
			Name:       podMetrics.Name,
			Namespace:  podMetrics.Namespace,
			CPU:        fmt.Sprintf("%dm", totalCPU.MilliValue()),
			Memory:     fmt.Sprintf("%dMi", totalMemory.Value()/1024/1024),
			Containers: containers,
		})
	}
	yamlBytes, err := yaml.Marshal(result)
	if err != nil {
		return "", err
	}
	return c.Header() + string(yamlBytes), nil
}

func formatEventList(list []eventOutput, header string) (string, error) {
	yamlBytes, err := yaml.Marshal(list)
	if err != nil {
		return "", err
	}
	return header + string(yamlBytes), nil
}

func formatApiList(list []apiResourcesOutput, header string) (string, error) {
	yamlBytes, err := yaml.Marshal(list)
	if err != nil {
		return "", err
	}
	return header + string(yamlBytes), nil
}

func formatResourcesList(list []listResourcesOutput, header string) (string, error) {
	yamlBytes, err := yaml.Marshal(list)
	if err != nil {
		return "", err
	}
	return header + string(yamlBytes), nil
}
