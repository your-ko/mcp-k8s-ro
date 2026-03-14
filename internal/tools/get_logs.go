package tools

import (
	"encoding/json"

	"github.com/your-ko/mcp-k8s-ro/internal/k8s"
	"github.com/your-ko/mcp-k8s-ro/internal/mcp"
)

type LogGetter struct {
	client *k8s.Client
}

func NewLogGetter(client *k8s.Client) LogGetter {
	return LogGetter{client: client}
}

func (tool LogGetter) Name() string {
	return "k8s_get_logs"
}

func (tool LogGetter) Description() string {
	return "Returns logs for a given pod. " +
		"This server is pinned to context '" + tool.client.Header() + "'. " +
		"Restart the server to switch clusters."
}

func (tool LogGetter) InputSchema() mcp.InputSchema {
	return mcp.InputSchema{
		Type: "object",
		Properties: json.RawMessage(`{
             "name":  {"type":"string","description":"Pod name"},
             "namespace": {"type":"string","description":"Namespace"},
			  "tailLines": {"type":"integer","description":"Number of lines to tail"}
         }`),
		Required: []string{"name"},
	}

}

func (tool LogGetter) Execute(params json.RawMessage) (string, error) {
	var p struct {
		Name      string `json:"name"`
		Namespace string `json:"namespace"`
		TailLines int64  `json:"tailLines"`
	}
	err := json.Unmarshal(params, &p)
	if err != nil {
		return "", err
	}

	return tool.client.GetLogs(p.Name, p.Namespace, p.TailLines)
}
