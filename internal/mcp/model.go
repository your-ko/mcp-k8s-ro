package mcp

import (
	"encoding/json"
)

type Tool interface {
	Name() string
	Description() string
	InputSchema() InputSchema
	Execute(params json.RawMessage) (string, error)
}

type JSONRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      any             `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type JSONRPCResponse struct {
	JSONRPC string        `json:"jsonrpc"`
	ID      any           `json:"id"`
	Result  any           `json:"result,omitempty"`
	Error   *JSONRPCError `json:"error,omitempty"`
}

type JSONRPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type InitialisationResult struct {
	ProtocolVersion string       `json:"protocolVersion"`
	ServerInfo      ServerInfo   `json:"serverInfo"`
	Capabilities    Capabilities `json:"capabilities"`
	Instructions    string       `json:"instructions"`
}

type ExecutionResult struct {
	Content []ContentItem `json:"content"`
}

type ListResult struct {
	Tools []ToolDefinition `json:"tools"`
}

type ContentItem struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type ServerInfo struct {
	Name        string `json:"name"`
	Version     string `json:"version"`
	ClusterName string `json:"clusterName"`
	ContextName string `json:"contextName"`
}

type Capabilities struct {
	Tools struct{} `json:"tools"`
}

type InputSchema struct {
	Type       string          `json:"type"`
	Properties json.RawMessage `json:"properties"`
	Required   []string        `json:"required,omitempty"`
}

type Property struct {
	Type        string `json:"type"`
	Description string `json:"description"`
}

type ToolDefinition struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	InputSchema InputSchema `json:"inputSchema"`
}

type rpcError struct {
	code int
	err  error
}

func (e *rpcError) Error() string { return e.err.Error() }
