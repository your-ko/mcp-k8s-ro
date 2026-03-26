package k8s

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
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
	dynamic      dynamicClient
	discovery    discoveryClient
	mapper       restMapper
	clientSet    *kubernetes.Clientset
	metricClient *metrics.Clientset
	contextName  string
	clusterName  string
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
		contextName:  contextName,
		clusterName:  clusterName,
		clientSet:    clientSet,
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
		// can't determine scope — assume namespaced so namespace filter still applies if provided
		return gvr, true, nil
	}
	mapping, err := c.mapper.RESTMapping(gvks[0].GroupKind(), gvks[0].Version)
	if err != nil {
		// can't determine scope — assume namespaced so namespace filter still applies if provided
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
	return serializeList(normaliseList(list), c.Header())
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
		res := listResourcesOutput{
			Name:      item.GetName(),
			Namespace: item.GetNamespace(),
			Status:    status,
			Created:   item.GetCreationTimestamp().UTC().Format(time.DateTime),
		}
		setResourceSpecificFields(item, &res)
		result = append(result, res)
	}
	return result
}

func setResourceSpecificFields(item unstructured.Unstructured, res *listResourcesOutput) {
	switch item.GetKind() {
	case "Secret":
		if resourceType, ok, err := unstructured.NestedString(item.Object, "type"); ok && err == nil {
			res.Type = resourceType
		}
	case "Service":
		if spec, ok, err := unstructured.NestedMap(item.Object, "spec"); ok && err == nil {
			if itemType, ok := spec["type"]; ok {
				if _, ok := itemType.(string); ok {
					res.Type = itemType.(string)
				}
			}
			if clusterIp, ok := spec["clusterIP"]; ok {
				if ip, ok := clusterIp.(string); ok {
					res.ClusterIP = ip
				}
			}
		}
		portsInfo := make([]string, 0)
		if ports, ok, err := unstructured.NestedSlice(item.Object, "spec", "ports"); ok && err == nil {
			for _, port := range ports {
				if portMap, ok := port.(map[string]interface{}); ok {
					portsInfo = append(portsInfo, fmt.Sprintf("%d/%s", portMap["port"], portMap["protocol"]))
				}
			}
		}
		res.Ports = strings.Join(portsInfo, ",")
	case "Pod":
		if nodeName, ok, err := unstructured.NestedString(item.Object, "spec", "nodeName"); ok && err == nil {
			res.Node = nodeName
		}
		if podIP, ok, err := unstructured.NestedString(item.Object, "status", "podIP"); ok && err == nil {
			res.PodIP = podIP
		}
		restarts := 0
		readyCount := 0
		waitingReasons := make([]string, 0)
		lastTermReasons := make([]string, 0)
		if containerStatuses, ok, err := unstructured.NestedSlice(item.Object, "status", "containerStatuses"); ok && err == nil {
			for _, cStatus := range containerStatuses {
				if cStatusMap, ok := cStatus.(map[string]interface{}); ok {
					if rc, ok := cStatusMap["restartCount"].(int64); ok {
						restarts += int(rc)
					}
					if isReady, ok := cStatusMap["ready"].(bool); ok && isReady {
						readyCount++
					}
					if stateMap, ok := cStatusMap["state"].(map[string]interface{}); ok {
						if waiting, ok := stateMap["waiting"].(map[string]interface{}); ok {
							if reason, ok := waiting["reason"].(string); ok && reason != "" {
								waitingReasons = append(waitingReasons, reason)
							}
						}
					}
					if lastState, ok := cStatusMap["lastState"].(map[string]interface{}); ok {
						if terminated, ok := lastState["terminated"].(map[string]interface{}); ok {
							if reason, ok := terminated["reason"].(string); ok && reason != "" {
								lastTermReasons = append(lastTermReasons, reason)
							}
						}
					}
				}
			}
			res.Ready = fmt.Sprintf("%d/%d", readyCount, len(containerStatuses))
		}
		res.Restarts = restarts
		if len(waitingReasons) > 0 {
			res.StateReason = strings.Join(waitingReasons, ",")
		}
		if len(lastTermReasons) > 0 {
			res.LastTerminationReason = strings.Join(lastTermReasons, ",")
		}
	case "Deployment", "StatefulSet", "DaemonSet":
		desired, _, _ := unstructured.NestedInt64(item.Object, "spec", "replicas")
		ready, _, _ := unstructured.NestedInt64(item.Object, "status", "readyReplicas")
		res.Ready = fmt.Sprintf("%d/%d", ready, desired)
	case "Node":
		if addresses, ok, err := unstructured.NestedSlice(item.Object, "status", "addresses"); ok && err == nil {
			for _, addr := range addresses {
				addrMap, ok := addr.(map[string]interface{})
				if !ok {
					continue
				}
				switch addrMap["type"] {
				case "InternalIP":
					ip, ok := addrMap["address"].(string)
					if ok {
						res.InternalIP = ip
						continue
					}
				case "ExternalIP":
					ip, ok := addrMap["address"].(string)
					if ok {
						res.ExternalIP = ip
						continue
					}
				}
			}
		}
		if conditions, ok, err := unstructured.NestedSlice(item.Object, "status", "conditions"); ok && err == nil {
			ready := "NotReady"
			problems := make([]string, 0)
			for _, condition := range conditions {
				conditionMap, ok := condition.(map[string]interface{})
				if !ok {
					// in case if condition is not a map
					continue
				}
				typ, _ := conditionMap["type"].(string)
				status, _ := conditionMap["status"].(string)
				if typ == "Ready" && status == "True" {
					ready = "Ready"
				} else if typ != "Ready" && status == "True" {
					// pressure/unavailable conditions — True means problem
					problems = append(problems, typ)
				}
			}
			if len(problems) > 0 {
				res.Status = strings.Join(problems, ",")
			}
			res.Ready = ready
		}
	}
}

func (c *Client) GetResource(ctx context.Context, name string, resource string, namespace string) (string, error) {
	gvr, namespaced, err := c.resolveGVR(resource)
	if err != nil {
		return "", err
	}

	var item *unstructured.Unstructured
	if namespaced && namespace != "" {
		item, err = c.dynamic.Resource(gvr).Namespace(namespace).Get(ctx, name, metav1.GetOptions{})
	} else {
		item, err = c.dynamic.Resource(gvr).Get(ctx, name, metav1.GetOptions{})
	}
	if err != nil {
		return "", err
	}

	if err = sanitize(item); err != nil {
		return "", err
	}

	yamlBytes, err := yaml.Marshal(item.Object)
	if err != nil {
		return "", err
	}
	return c.Header() + string(yamlBytes), nil
}

func (c *Client) GetLogs(ctx context.Context, podName string, namespace string, tailLines int64, previous bool, container string) (string, error) {
	opts := &v1.PodLogOptions{}
	if tailLines > 0 {
		opts.TailLines = &tailLines
	}
	if container != "" {
		opts.Container = container
	}
	opts.Previous = previous
	request := c.clientSet.CoreV1().Pods(namespace).GetLogs(podName, opts)
	podLogs, err := request.Stream(ctx)
	if err != nil {
		return "", err
	}
	defer func(podLogs io.ReadCloser) {
		err := podLogs.Close()
		if err != nil {
			slog.Error("can't close pod log reader", "error", err)
		}
	}(podLogs)

	buf := new(bytes.Buffer)
	_, err = io.Copy(buf, podLogs)
	if err != nil {
		return "", fmt.Errorf("error in copy from podLogs to buf: %w", err)
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
		if resources == nil {
			return "", err
		}
		// partial result — some API groups unreachable (e.g. broken CRDs), log and continue with what we have
		slog.Warn("ServerPreferredResources returned partial results", "error", err)
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
			result = append(result, apiResourcesOutput{
				Name:       r.Name,
				Kind:       r.Kind,
				Group:      group,
				Namespaced: r.Namespaced,
			})
		}
	}

	return serializeList(result, c.Header())
}

func (c *Client) GetEvents(ctx context.Context, namespace string, limit int64) (string, error) {
	opts := metav1.ListOptions{}
	if limit > 0 {
		opts.Limit = limit
	}
	list, err := c.clientSet.EventsV1().Events(namespace).List(ctx, opts)
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

	return serializeList(result, c.Header())
}

func (c *Client) TopPods(ctx context.Context, namespace string) (string, error) {
	podMetricsList, err := c.metricClient.MetricsV1beta1().PodMetricses(namespace).List(ctx, metav1.ListOptions{})
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
	return serializeList(result, c.Header())
}

func (c *Client) TopNodes(ctx context.Context) (string, error) {
	nodeMetrics, err := c.metricClient.MetricsV1beta1().NodeMetricses().List(ctx, metav1.ListOptions{})
	if err != nil {
		return "", err
	}
	nodes, err := c.clientSet.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return "", err
	}
	capacities := make(map[string]v1.ResourceList, len(nodes.Items))
	for _, node := range nodes.Items {
		capacities[node.Name] = node.Status.Allocatable
	}

	result := make([]nodeTopOutput, 0, len(nodeMetrics.Items))
	for _, nodeMetric := range nodeMetrics.Items {
		cpu := nodeMetric.Usage[v1.ResourceCPU]
		mem := nodeMetric.Usage[v1.ResourceMemory]

		cpuPct := ""
		memPct := ""
		if capacity, ok := capacities[nodeMetric.Name]; ok {
			if capCPU := capacity.Cpu(); capCPU.MilliValue() > 0 {
				cpuPct = fmt.Sprintf("%d%%", cpu.MilliValue()*100/capCPU.MilliValue())
			}
			if capMem := capacity.Memory(); capMem.Value() > 0 {
				memPct = fmt.Sprintf("%d%%", mem.Value()*100/capMem.Value())
			}
		}

		result = append(result, nodeTopOutput{
			Name:      nodeMetric.Name,
			CPU:       fmt.Sprintf("%dm", cpu.MilliValue()),
			CPUPct:    cpuPct,
			Memory:    fmt.Sprintf("%dMi", mem.Value()/1024/1024),
			MemoryPct: memPct,
		})
	}
	return serializeList(result, c.Header())
}

// sanitize removes or redacts fields before the object is serialized and returned.
// It mutates the object in place.
func sanitize(item *unstructured.Unstructured) error {
	// Always strip managedFields — no diagnostic value, saves tokens.
	unstructured.RemoveNestedField(item.Object, "metadata", "managedFields")

	switch item.GetKind() {
	case "Secret":
		for _, field := range []string{"data", "stringData"} {
			if raw, ok, err := unstructured.NestedFieldNoCopy(item.Object, field); ok && err == nil {
				if m, ok := raw.(map[string]interface{}); ok {
					redacted := make(map[string]interface{}, len(m))
					for k := range m {
						redacted[k] = "*****"
					}
					if err := unstructured.SetNestedMap(item.Object, redacted, field); err != nil {
						return err
					}
				}
			}
		}
	case "CertificateSigningRequest":
		if spec, ok, err := unstructured.NestedMap(item.Object, "spec"); ok && err == nil {
			if _, ok := spec["request"]; ok {
				if err := unstructured.SetNestedField(item.Object, "*****", "spec", "request"); err != nil {
					return err
				}
			}
		}

	case "Certificate":
		if spec, ok, err := unstructured.NestedFieldNoCopy(item.Object, "spec"); ok && err == nil {
			if specMap, ok := spec.(map[string]interface{}); ok {
				if _, ok := specMap["keystores"]; ok {
					if err := unstructured.SetNestedField(item.Object, "*****", "spec", "keystores"); err != nil {
						return err
					}
				}
			}
		}
		if conditions, ok, err := unstructured.NestedSlice(item.Object, "status", "conditions"); ok && err == nil {
			redacted := false
			for _, condition := range conditions {
				if condMap, ok := condition.(map[string]interface{}); ok {
					if message, ok := condMap["message"]; ok {
						if messageStr, ok := message.(string); ok {
							if strings.HasPrefix(messageStr, "-----BEGIN CERTIFICATE-----") {
								condMap["message"] = "*****"
								redacted = true
							}
						}
					}
				}
			}
			if redacted {
				if err := unstructured.SetNestedSlice(item.Object, conditions, "status", "conditions"); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func serializeList[T any](list []T, header string) (string, error) {
	yamlBytes, err := yaml.Marshal(list)
	if err != nil {
		return "", err
	}
	return header + string(yamlBytes), nil
}
