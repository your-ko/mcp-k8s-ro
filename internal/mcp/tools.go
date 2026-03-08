package mcp

import "encoding/json"

type Tool interface {
	Name() string
	Description() string
	InputSchema() InputSchema
	Execute(params json.RawMessage) (string, error)
}

type getNS struct{}

func (getNS) Name() string        { return "k8s_get_namespaces" }
func (getNS) Description() string { return "List all namespaces in the cluster" }
func (getNS) InputSchema() InputSchema {
	return InputSchema{
		Type: "object",
	}
}
