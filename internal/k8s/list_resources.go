package k8s

import (
	"context"
	"encoding/json"

	"github.com/your-ko/mcp-k8s-ro/internal/mcp"
	"gopkg.in/yaml.v3"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// ListResources Takes a resource type (e.g. pods, deployments, certificates) and optional namespace, returns a formatted list.
// Works for any resource — core or CRD.
//
//	 Input:
//	{"resource": "pods", "namespace": "kube-system"}
//
//	Output:
//	- name: kube-proxy-5ppkb
//    namespace: kube-system
//    status: Running
//    ready: 1/1
//    created: "2026-03-09"

type ListResources struct {
	client      *Client
	contextName string
	clusterName string
}

func NewResourcesLister(client *Client) ListResources {
	return ListResources{client: client}
}

func (tool ListResources) Name() string {
	return "k8s_list_resources"
}

func (tool ListResources) Description() string {
	return "List any Kubernetes resource by type"
}

func (tool ListResources) InputSchema() mcp.InputSchema {
	return mcp.InputSchema{
		Type: "object",
		Properties: json.RawMessage(`{
              "resource":  {"type":"string","description":"Resource type, e.g. pods, deployments, certificates"},
              "namespace": {"type":"string","description":"Namespace (omit for cluster-scoped resources)"}
          }`),
		Required: []string{"resource"},
	}
}

func (tool ListResources) Execute(params json.RawMessage) (string, error) {
	var p struct {
		Resource  string `json:"resource"`
		Namespace string `json:"namespace"`
	}
	err := json.Unmarshal(params, &p)
	if err != nil {
		return "", err
	}
	// TODO: improve context
	list, err := tool.client.ListResources(context.TODO(), p.Resource, p.Namespace)
	if err != nil {
		return "", err
	}
	return formatResourcesList(tool.normaliseList(list))
}

type listResourcesOutput struct {
	Name      string `yaml:"name"`
	Namespace string `yaml:"namespace,omitempty"`
	Status    string `yaml:"status,omitempty"`
	Ready     string `yaml:"ready,omitempty"`
	Created   string `yaml:"created,omitempty"`
	Context   string `yaml:"context"`
}

func formatResourcesList(list []listResourcesOutput) (string, error) {
	yamlBytes, err := yaml.Marshal(list)
	if err != nil {
		return "", err
	}
	//fmt.Fprintln(os.Stderr, string(yamlBytes))
	return string(yamlBytes), nil
}

func (tool ListResources) normaliseList(list *unstructured.UnstructuredList) []listResourcesOutput {
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
			Context:   tool.contextName,
		})
	}
	return result
}
