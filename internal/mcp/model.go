package mcp

import (
	"encoding/json"
)

// Tool is implemented by every k8s tool registered with the MCP server.
type Tool interface {
	// Name returns the tool identifier used in tools/list and tools/call.
	Name() string
	// Description is shown to the model to explain what the tool does.
	Description() string
	// InputSchema returns the JSON Schema for the tool's arguments.
	InputSchema() InputSchema
	// Execute runs the tool with the given JSON-encoded arguments.
	Execute(params json.RawMessage) (string, error)
}

// JSONRPCRequest is an incoming JSON-RPC 2.0 message from the client.
// Notifications (no ID) are handled by ignoring them without a response.
type JSONRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      any             `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// JSONRPCResponse is the outgoing JSON-RPC 2.0 response written to stdout.
type JSONRPCResponse struct {
	JSONRPC string        `json:"jsonrpc"`
	ID      any           `json:"id"`
	Result  any           `json:"result,omitempty"`
	Error   *JSONRPCError `json:"error,omitempty"`
}

// JSONRPCError carries a standard JSON-RPC error code and message.
type JSONRPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// InitialisationResult is the response body for the "initialize" method.
type InitialisationResult struct {
	ProtocolVersion string       `json:"protocolVersion"`
	ServerInfo      ServerInfo   `json:"serverInfo"`
	Capabilities    Capabilities `json:"capabilities"`
	Instructions    string       `json:"instructions"`
}

// ExecutionResult is the response body for a successful "tools/call".
type ExecutionResult struct {
	Content []ContentItem `json:"content"`
}

// ListResult is the response body for "tools/list".
type ListResult struct {
	Tools []ToolDefinition `json:"tools"`
}

// ContentItem is a single piece of content returned by a tool execution.
// Type is always "text" for this server.
type ContentItem struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// ServerInfo is included in the initialize response so the client knows
// which server, version, and cluster it is talking to.
type ServerInfo struct {
	Name        string `json:"name"`
	Version     string `json:"version"`
	ClusterName string `json:"clusterName"`
	ContextName string `json:"contextName"`
}

// Capabilities advertises which MCP features this server supports.
// Only tools are supported; resources and prompts are not.
type Capabilities struct {
	Tools struct{} `json:"tools"`
}

// InputSchema is the JSON Schema object describing a tool's arguments.
type InputSchema struct {
	Type       string          `json:"type"`
	Properties json.RawMessage `json:"properties"`
	Required   []string        `json:"required,omitempty"`
}

// Property describes a single field within an InputSchema.
type Property struct {
	Type        string `json:"type"`
	Description string `json:"description"`
}

// ToolDefinition is the wire representation of a tool in the tools/list response.
type ToolDefinition struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	InputSchema InputSchema `json:"inputSchema"`
}

// rpcError wraps a JSON-RPC error code and a Go error for internal use.
type rpcError struct {
	code int
	err  error
}

func (e *rpcError) Error() string { return e.err.Error() }
