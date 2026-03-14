package tools

import (
	"encoding/json"

	"github.com/your-ko/mcp-k8s-ro/internal/k8s"
	"github.com/your-ko/mcp-k8s-ro/internal/mcp"
)

type ListResourceTypes struct {
	client *k8s.Client
}

func NewResourceTypesLister(client *k8s.Client) ListResourceTypes {
	return ListResourceTypes{client: client}
}

func (tool ListResourceTypes) Name() string {
	return "k8s_list_resource_types"
}

func (tool ListResourceTypes) Description() string {
	return "Lists all available resource types via discovery API. " +
		"This server is pinned to context '" + tool.client.ContextSummary() + "'. " +
		"Restart the server to switch clusters."
}

func (tool ListResourceTypes) InputSchema() mcp.InputSchema {
	return mcp.InputSchema{
		Type: "object",
		Properties: json.RawMessage(`{
             "group":  {"type":"string","description":"API group filter parameter"}
         }`),
	}
}

func (tool ListResourceTypes) Execute(params json.RawMessage) (string, error) {
	var p struct {
		ApiGroup string `json:"group"`
	}
	err := json.Unmarshal(params, &p)
	if err != nil {
		return "", err
	}
	return tool.client.GetApiResources(p.ApiGroup)
}
