package k8s

import (
	"encoding/json"

	"github.com/your-ko/mcp-k8s-ro/internal/mcp"
)

type DescribeResource struct {
	client *Client
}

func NewDescribeResource(client *Client) DescribeResource {
	return DescribeResource{client: client}
}

func (d DescribeResource) Name() string {
	return "k8s_describe_resource"
}

func (d DescribeResource) Description() string {
	return "Describe any Kubernetes resource"
}

func (d DescribeResource) InputSchema() mcp.InputSchema {
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

func (d DescribeResource) Execute(params json.RawMessage) (string, error) {
	//TODO implement me
	panic("implement me")
}
