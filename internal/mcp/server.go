package mcp

import (
	"bufio"
	"encoding/json"
	"errors"
	"os"
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

func (s *Server) Process(request JSONRPCRequest) (*JSONRPCResponse, error) {
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
			return nil, err
		}
		for _, tool := range s.tools {
			if tool.Name() == p.Name {
				result, err := tool.Execute(p.Arguments)
				if err != nil {
					return nil, err
				}
				return &JSONRPCResponse{
					JSONRPC: "2.0",
					ID:      request.ID,
					Result:  map[string]any{"content": []map[string]string{{"type": "text", "text": result}}},
				}, nil
			}
		}
	}
	return nil, errors.New("not implemented yet")
}

func (s *Server) Start() {
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		line := scanner.Bytes()

		request := JSONRPCRequest{}
		err := json.Unmarshal(line, &request)
		if err != nil {
			handleError(request, err)
			continue
		}
		response, err := s.Process(request)
		if err != nil {
			handleError(request, err)
			continue
		}
		if response == nil {
			continue
		}
		err = json.NewEncoder(os.Stdout).Encode(response)
		if err != nil {
			handleError(request, err)
		}
	}
}

func handleError(request JSONRPCRequest, err error) {
	response := JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      request.ID,
		Error: &JSONRPCError{
			Code:    -32603, // TODO: improve in the future, check JSON-RPC 2.0 standard error codes
			Message: err.Error(),
		}}
	_ = json.NewEncoder(os.Stdout).Encode(response)
}

func (s *Server) Register(tool Tool) {
	s.tools = append(s.tools, tool)
}
