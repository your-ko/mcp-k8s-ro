package tools

import (
	"encoding/json"

	"github.com/your-ko/mcp-k8s-ro/internal/k8s"
	"github.com/your-ko/mcp-k8s-ro/internal/mcp"
)

type PodTopper struct {
	client *k8s.Client
}

func NewPodTopper(client *k8s.Client) PodTopper {
	return PodTopper{client: client}
}

func (tool PodTopper) Name() string {
	return "k8s_top_pods"
}

func (tool PodTopper) Description() string {
	return "CPU/memory usage per pod. Requires metrics-server, but invaluable for performance debugging. Uses clientSet.MetricsV1beta1(). " +
		"This server is pinned to context '" + tool.client.ContextSummary() + "'. " +
		"Restart the server to switch clusters."

}

func (tool PodTopper) InputSchema() mcp.InputSchema {
	return mcp.InputSchema{
		Type: "object",
		Properties: json.RawMessage(`{
             "namespace": {"type":"string","description":"Namespace (omit for cluster-scoped resources)"}
         }`),
	}
}

func (tool PodTopper) Execute(params json.RawMessage) (string, error) {
	var p struct {
		Namespace string `json:"namespace"`
	}
	err := json.Unmarshal(params, &p)
	if err != nil {
		return "", err
	}
	return tool.client.TopPods(p.Namespace)
}
