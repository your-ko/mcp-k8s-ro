package mcp

type Tool interface {
	Name() string
	Description() string
	InputSchema() string
}
