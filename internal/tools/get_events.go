package tools

import (
	"encoding/json"

	"github.com/your-ko/mcp-k8s-ro/internal/k8s"
	"github.com/your-ko/mcp-k8s-ro/internal/mcp"
)

type EventGetter struct {
	client *k8s.Client
}

func NewEventGetter(client *k8s.Client) EventGetter {
	return EventGetter{client: client}
}

func (tool EventGetter) Name() string {
	return "k8s_get_events"
}

func (tool EventGetter) Description() string {
	return "Returns list of K8s events." +
		"This server is pinned to context '" + tool.client.Header() + "'. " +
		"Restart the server to switch clusters."
}

func (tool EventGetter) InputSchema() mcp.InputSchema {
	return mcp.InputSchema{
		Type: "object",
		Properties: json.RawMessage(`{
             "namespace": {"type":"string","description":"Namespace (omit for cluster-scoped resources)"},
             "limit": {"type":"integer","description":"Number of lines to fetch"}
         }`),
		Required: []string{},
	}
}

func (tool EventGetter) Execute(params json.RawMessage) (string, error) {
	var p struct {
		Namespace string `json:"namespace"`
		Limit     int64  `json:"limit"`
	}
	err := json.Unmarshal(params, &p)
	if err != nil {
		return "", err
	}
	if p.Limit == 0 {
		// if not provided then show at least something
		p.Limit = 100
	}
	return tool.client.GetEvents(p.Namespace, p.Limit)
}
