package tools

import (
	"context"
	"encoding/json"

	"github.com/your-ko/mcp-k8s-ro/internal/k8s"
	"github.com/your-ko/mcp-k8s-ro/internal/mcp"
)

type DescribeResource struct {
	client *k8s.Client
}

func NewResourceDescriber(client *k8s.Client) DescribeResource {
	return DescribeResource{client: client}
}

func (tool DescribeResource) Name() string {
	return "k8s_describe_resource"
}

func (tool DescribeResource) Description() string {
	return "Describe any Kubernetes resource." +
		"This server is pinned to context '" + tool.client.Header() + "'. " +
		"Restart the server to switch clusters."
}

func (tool DescribeResource) InputSchema() mcp.InputSchema {
	return mcp.InputSchema{
		Type: "object",
		Properties: json.RawMessage(`{
             "name":  {"type":"string","description":"Resource name"},
             "resource":  {"type":"string","description":"Resource type, e.g. pods, deployments, certificates"},
             "namespace": {"type":"string","description":"Namespace (omit for cluster-scoped resources)"}
         }`),
		Required: []string{"name", "resource"},
	}
}

func (tool DescribeResource) Execute(params json.RawMessage) (string, error) {
	var p struct {
		Name      string `json:"name"`
		Resource  string `json:"resource"`
		Namespace string `json:"namespace"`
	}
	err := json.Unmarshal(params, &p)
	if err != nil {
		return "", err
	}
	return tool.client.GetResource(context.TODO(), p.Name, p.Resource, p.Namespace)
}
