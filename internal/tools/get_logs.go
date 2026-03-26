package tools

import (
	"context"
	"encoding/json"
	"time"

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
		"This server is pinned to context '" + tool.client.ContextSummary() + "'. " +
		"Restart the server to switch clusters."
}

func (tool LogGetter) InputSchema() mcp.InputSchema {
	return mcp.InputSchema{
		Type: "object",
		Properties: json.RawMessage(`{
             "name":  {"type":"string","description":"Pod name"},
             "namespace": {"type":"string","description":"Namespace"},
             "container": {"type":"string","description":"Container selector"},
             "previous": {"type":"boolean","description":"Show logs from a crashed/restarted container (kubectl logs --previous). Useful for debugging crashloops."},
             "tailLines": {"type":"integer","description":"Number of lines to tail. 0 or none is to fetch everything"}
         }`),
		Required: []string{"name"},
	}
}

func (tool LogGetter) Execute(params json.RawMessage) (string, error) {
	var p struct {
		Name      string `json:"name"`
		Namespace string `json:"namespace"`
		Container string `json:"container"`
		Previous  bool   `json:"previous"`
		TailLines int64  `json:"tailLines"`
	}
	err := json.Unmarshal(params, &p)
	if err != nil {
		return "", err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	return tool.client.GetLogs(ctx, p.Name, p.Namespace, p.TailLines, p.Previous, p.Container)
}
