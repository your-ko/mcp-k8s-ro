package k8s

import (
	"encoding/json"

	"github.com/your-ko/mcp-k8s-ro/internal/mcp"
)

type ListResourceTypes struct {
	client *Client
}

func NewListResourceTypes(client *Client) ListResourceTypes {
	return ListResourceTypes{client: client}
}

func (tool ListResourceTypes) Name() string {
	return "k8s_list_resource_types"
}

func (tool ListResourceTypes) Description() string {
	return "Lists all available resource types via discovery API"
}

func (tool ListResourceTypes) InputSchema() mcp.InputSchema {
	return mcp.InputSchema{
		Type:       "object",
		Properties: json.RawMessage(`{}`),
		Required:   []string{},
	}
}

func (tool ListResourceTypes) Execute(params json.RawMessage) (string, error) {
	//TODO implement me
	panic("implement me")
}
