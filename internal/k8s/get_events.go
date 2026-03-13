package k8s

import (
	"encoding/json"

	"github.com/your-ko/mcp-k8s-ro/internal/mcp"
)

type EventGetter struct {
	client *Client
}

func NewEventGetter(client *Client) EventGetter {
	return EventGetter{client: client}
}

func (e EventGetter) Name() string {
	return "k8s_get_events"
}

func (e EventGetter) Description() string {
	return "Returns K8s events"
}

func (e EventGetter) InputSchema() mcp.InputSchema {
	return mcp.InputSchema{
		Type: "object",
		Properties: json.RawMessage(`{
              "namespace": {"type":"string","description":"Namespace (omit for cluster-scoped resources)"}
          }`),
		Required: []string{},
	}
}

func (e EventGetter) Execute(params json.RawMessage) (string, error) {
	//TODO implement me
	panic("implement me")
}
