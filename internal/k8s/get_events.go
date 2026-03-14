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

func (tool EventGetter) Name() string {
	return "k8s_get_events"
}

func (tool EventGetter) Description() string {
	return "Returns K8s events"
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
	// TODO: add print header
	return tool.client.getEvents(p.Namespace, p.Limit)
}

type eventOutput struct {
	Name      string   `yaml:"name"`
	Namespace string   `yaml:"namespace"`
	Kind      string   `yaml:"kind"`
	Reason    string   `yaml:"reason"`
	Message   string   `yaml:"message"`
	Type      string   `yaml:"type"`
	Count     int32    `yaml:"count"`
	FirstTime yamlTime `yaml:"firstTime"`
	LastTime  yamlTime `yaml:"lastTime"`
}
