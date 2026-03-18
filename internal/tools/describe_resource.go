package tools

import (
	"context"
	"encoding/json"
	"time"

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
	return "Describe any Kubernetes resource. " +
		"This server is pinned to context '" + tool.client.ContextSummary() + "'. " +
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
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return tool.client.GetResource(ctx, p.Name, p.Resource, p.Namespace)
}
