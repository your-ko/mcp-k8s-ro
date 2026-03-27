package mcp

import (
	"encoding/json"
	"errors"
	"reflect"
	"testing"
)

func TestServer_Register(t *testing.T) {
	type fields struct {
		name        string
		version     string
		contextName string
		clusterName string
	}
	tests := []struct {
		name   string
		fields fields
	}{
		{
			name: "tool registered",
			fields: fields{
				name:        "test",
				version:     "test",
				contextName: "test",
				clusterName: "test",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Server{
				name:        tt.fields.name,
				version:     tt.fields.version,
				contextName: tt.fields.contextName,
				clusterName: tt.fields.clusterName,
			}
			s.Register(NewMockTool(t))

			if len(s.tools) != 1 {
				t.Errorf("expected %d tool registered, got %d\n", 1, len(s.tools))
			}
		})
	}
}

func TestServer_process(t *testing.T) {

	type ServerData struct {
		name        string
		version     string
		contextName string
		clusterName string
	}
	serverData := ServerData{
		name:        "test",
		version:     "test",
		contextName: "test",
		clusterName: "test",
	}
	type args struct {
		request JSONRPCRequest
	}
	tests := []struct {
		name         string
		args         args
		tools        func() []Tool
		wantResponse any
		wantErr      *rpcError
	}{
		{
			name: "notification_no_id",
			args: args{request: JSONRPCRequest{
				ID:     nil,
				Method: "initialize",
			}},
		},
		{
			name: "initialize",
			args: args{request: JSONRPCRequest{
				ID:     1,
				Method: "initialize",
			}},
			wantResponse: InitialisationResult{
				ProtocolVersion: "2024-11-05",
				ServerInfo:      ServerInfo{serverData.name, serverData.version, serverData.clusterName, serverData.contextName},
				Capabilities:    Capabilities{},
				Instructions:    getInstructions(),
			},
		},
		{
			name: "tools_list_empty",
			args: args{request: JSONRPCRequest{
				ID:     1,
				Method: "tools/list",
			}},
			wantResponse: ListResult{Tools: make([]ToolDefinition, 0)},
		},
		{
			name: "tools_list_returns_registered_tools",
			args: args{request: JSONRPCRequest{
				ID:     1,
				Method: "tools/list",
			}},
			tools: func() []Tool {
				m := NewMockTool(t)
				m.EXPECT().Name().Return("my_tool")
				m.EXPECT().Description().Return("tooling")
				m.EXPECT().InputSchema().Return(InputSchema{Type: "object"})
				return []Tool{m}
			},
			wantResponse: ListResult{Tools: []ToolDefinition{{"my_tool", "tooling", InputSchema{Type: "object"}}}},
		},
		{
			name: "tools_call_valid",
			args: args{request: JSONRPCRequest{
				ID:     1,
				Method: "tools/call",
				Params: json.RawMessage(`{"name":"test","arguments":{}}`),
			}},
			tools: func() []Tool {
				m := NewMockTool(t)
				m.EXPECT().Execute(json.RawMessage(`{}`)).Return("result", nil)
				m.EXPECT().Name().Return("test")
				return []Tool{m}
			},
			wantResponse: ExecutionResult{[]ContentItem{{Type: "text", Text: "result"}}},
		},
		{
			name: "tools_call_invalid_json_params",
			args: args{request: JSONRPCRequest{
				ID:     1,
				Method: "tools/call",
				Params: json.RawMessage(`{"invalid_json`),
			}},
			wantErr: &rpcError{code: -32602},
		},
		{
			name: "tools_call_unknown_tool",
			args: args{request: JSONRPCRequest{
				ID:     1,
				Method: "tools/call",
				Params: json.RawMessage(`{"name":"unknown","arguments":{}}`),
			}},
			tools: func() []Tool {
				m := NewMockTool(t)
				m.EXPECT().Name().Return("test")
				return []Tool{m}
			},
			wantErr: &rpcError{code: -32603, err: errors.New("unknown tool: unknown")},
		},
		{
			name: "tools_call_tool_execute_error",
			args: args{request: JSONRPCRequest{
				ID:     1,
				Method: "tools/call",
				Params: json.RawMessage(`{"name":"test","arguments":{}}`),
			}},
			tools: func() []Tool {
				m := NewMockTool(t)
				m.EXPECT().Name().Return("test")
				m.EXPECT().Execute(json.RawMessage(`{}`)).Return("result", errors.New("expected error"))
				return []Tool{m}
			},
			wantErr: &rpcError{code: -32603, err: errors.New("expected error")},
		},
		{
			name: "unknown_method",
			args: args{request: JSONRPCRequest{
				ID:     1,
				Method: "foo/bar",
			}},
			wantErr: &rpcError{code: -32601, err: errors.New("unknown method: foo/bar")},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Server{
				name:        serverData.name,
				version:     serverData.version,
				contextName: serverData.contextName,
				clusterName: serverData.clusterName,
			}
			if tt.tools != nil {
				for _, tool := range tt.tools() {
					s.Register(tool)
				}
			}

			response, err := s.process(tt.args.request)

			if tt.wantErr != nil && err == nil {
				t.Errorf("expected error, got nil")
			}
			if err != nil && tt.wantErr == nil {
				t.Errorf("expected error %s, got nil", tt.wantErr.err)
			}
			if err != nil && tt.wantErr != nil {
				if err.code != tt.wantErr.code {
					t.Errorf("process() got error code = %d, want %d", err.code, tt.wantErr.code)
				}
			}

			if !reflect.DeepEqual(response, tt.wantResponse) {
				t.Errorf("process() got = %v, want %v", response, tt.wantResponse)
			}
		})
	}
}
