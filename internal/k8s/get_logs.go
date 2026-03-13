package k8s

import (
	"encoding/json"

	"github.com/your-ko/mcp-k8s-ro/internal/mcp"
)

type LogGetter struct {
	client *Client
}

func NewLogGetter(client *Client) LogGetter {
	return LogGetter{client: client}
}

func (tool LogGetter) Name() string {
	return "k8s_get_logs"
}

func (tool LogGetter) Description() string {
	return "Returns logs for a given pod"
}

func (tool LogGetter) InputSchema() mcp.InputSchema {
	return mcp.InputSchema{
		Type: "object",
		Properties: json.RawMessage(`{
              "name":  {"type":"string","description":"Pod name"},
              "namespace": {"type":"string","description":"Namespace"}
          }`),
		Required: []string{"resource"},
	}

}

func (tool LogGetter) Execute(params json.RawMessage) (string, error) {
	var p struct {
		Resource  string `json:"resource"`
		Namespace string `json:"namespace"`
	}
	err := json.Unmarshal(params, &p)
	if err != nil {
		return "", err
	}

	// TODO: implement me
	return "", err
}
