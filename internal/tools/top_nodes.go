package tools

import (
	"context"
	"encoding/json"
	"time"

	"github.com/your-ko/mcp-k8s-ro/internal/k8s"
	"github.com/your-ko/mcp-k8s-ro/internal/mcp"
)

type NodeTopper struct {
	client *k8s.Client
}

func NewNodeTopper(client *k8s.Client) NodeTopper {
	return NodeTopper{client: client}
}

func (tool NodeTopper) Name() string {
	return "k8s_top_nodes"
}

func (tool NodeTopper) Description() string {
	return "CPU/memory usage per K8s node. Requires metrics-server, but invaluable for performance debugging. Uses clientSet.MetricsV1beta1(). " +
		"This server is pinned to context '" + tool.client.ContextSummary() + "'. " +
		"Restart the server to switch clusters."

}

func (tool NodeTopper) InputSchema() mcp.InputSchema {
	return mcp.InputSchema{
		Type:       "object",
		Properties: json.RawMessage(`{}`),
	}
}

func (tool NodeTopper) Execute(_ json.RawMessage) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return tool.client.TopNodes(ctx)
}
