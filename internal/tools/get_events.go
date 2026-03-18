package tools

import (
	"context"
	"encoding/json"
	"time"

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
	return "Returns list of K8s events. " +
		"This server is pinned to context '" + tool.client.ContextSummary() + "'. " +
		"Restart the server to switch clusters."
}

func (tool EventGetter) InputSchema() mcp.InputSchema {
	return mcp.InputSchema{
		Type: "object",
		Properties: json.RawMessage(`{
             "namespace": {"type":"string","description":"Namespace (omit for cluster-scoped resources)"},
             "limit": {"type":"integer","description":"Number of lines to fetch. 0 or none is to fetch everything"}
         }`),
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
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return tool.client.GetEvents(ctx, p.Namespace, p.Limit)
}
