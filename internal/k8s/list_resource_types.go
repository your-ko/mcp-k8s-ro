package k8s

import (
	"encoding/json"

	"github.com/your-ko/mcp-k8s-ro/internal/mcp"
)

type ListResourceTypes struct {
	client *Client
}

func NewResourceTypesLister(client *Client) ListResourceTypes {
	return ListResourceTypes{client: client}
}

func (tool ListResourceTypes) Name() string {
	return "k8s_list_resource_types"
}

func (tool ListResourceTypes) Description() string {
	return "Lists all available resource types via discovery API. " +
		"This server is pinned to context '" + tool.client.contextName + "'. " +
		"Restart the server to switch clusters."
}

func (tool ListResourceTypes) InputSchema() mcp.InputSchema {
	return mcp.InputSchema{
		Type: "object",
		Properties: json.RawMessage(`{
             "group":  {"type":"string","description":"API group filter parameter"}
         }`),
		Required: []string{},
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
	return tool.client.getApiResources(p.ApiGroup)
}

type apiResourcesOutput struct {
	Name       string `yaml:"name"`
	Kind       string `yaml:"kind"`
	Group      string `yaml:"group"`
	Namespaced bool   `yaml:"namespaced"`
}
