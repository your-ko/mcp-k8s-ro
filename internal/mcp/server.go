package mcp

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
)

type Server struct {
	name        string
	version     string
	contextName string
	clusterName string
	tools       []Tool
}

func New(name string, version string, contextName string, clusterName string) *Server {
	return &Server{
		name:        name,
		version:     version,
		contextName: contextName,
		clusterName: clusterName,
	}
}

func (s *Server) process(request JSONRPCRequest) (*JSONRPCResponse, *rpcError) {
	if request.ID == nil {
		return nil, nil
	}
	switch request.Method {
	case "initialize":
		return &JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      request.ID,
			Result: InitialisationResult{
				ProtocolVersion: "2024-11-05",
				ServerInfo:      ServerInfo{Name: s.name, Version: s.version, ClusterName: s.clusterName, ContextName: s.contextName},
				Instructions:    "This is a READ-ONLY server. For any operation that would create, update, delete, scale, restart, exec into, or otherwise mutate Kubernetes resources: do NOT even attempt it. Instead, print  the equivalent kubectl command and tell the user to run it manually.",
				Capabilities:    Capabilities{},
			},
		}, nil
	case "tools/list":
		toolDefs := make([]ToolDefinition, 0, len(s.tools))
		for _, tool := range s.tools {
			toolDefs = append(toolDefs, ToolDefinition{
				Name:        tool.Name(),
				Description: tool.Description(),
				InputSchema: tool.InputSchema(),
			})
		}
		return &JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      request.ID,
			Result:  map[string]any{"tools": toolDefs},
		}, nil
	case "tools/call":
		var p struct {
			Name      string          `json:"name"`
			Arguments json.RawMessage `json:"arguments"`
		}
		err := json.Unmarshal(request.Params, &p)
		if err != nil {
			return nil, &rpcError{-32602, err} // invalid params
		}
		for _, tool := range s.tools {
			if tool.Name() == p.Name {
				result, err := tool.Execute(p.Arguments)
				if err != nil {
					return nil, &rpcError{-32603, err}
				}
				return &JSONRPCResponse{
					JSONRPC: "2.0",
					ID:      request.ID,
					Result:  map[string]any{"content": []map[string]string{{"type": "text", "text": result}}},
				}, nil
			}
		}
		return nil, &rpcError{-32603, fmt.Errorf("unknown tool: %s", p.Name)}
	}
	return nil, &rpcError{-32601, fmt.Errorf("unknown method: %s", request.Method)}
}

func (s *Server) Start(input io.Reader, output io.Writer) {
	scanner := bufio.NewScanner(input)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024) // A large tools/call payload will silently fail with token too long.
	for scanner.Scan() {
		line := scanner.Bytes()

		request := JSONRPCRequest{}
		err := json.Unmarshal(line, &request)
		if err != nil {
			handleError(output, request.ID, -32700, err)
			continue
		}
		response, rpcErr := s.process(request)
		if rpcErr != nil {
			handleError(output, request.ID, rpcErr.code, rpcErr.err)
			continue
		}
		if response == nil {
			continue
		}
		err = json.NewEncoder(output).Encode(response)
		if err != nil {
			handleError(output, request.ID, -32603, err)
		}
	}
}

func handleError(output io.Writer, requestId any, errorCode int, err error) {
	response := JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      requestId,
		Error: &JSONRPCError{
			Code:    errorCode,
			Message: err.Error(),
		}}
	_ = json.NewEncoder(output).Encode(response)
}

func (s *Server) Register(tool Tool) {
	s.tools = append(s.tools, tool)
}
