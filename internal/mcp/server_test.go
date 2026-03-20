package mcp

import (
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
				Instructions:    "This is a READ-ONLY server. For any operation that would create, update, delete, scale, restart, exec into, or otherwise mutate Kubernetes resources: do NOT even attempt it. Instead, print  the equivalent kubectl command and tell the user to run it manually.",
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
				m.On("Name").Return("my_tool")
				m.On("Description").Return("tooling")
				m.On("InputSchema").Return(InputSchema{Type: "object"})
				return []Tool{m}
			},
			wantResponse: ListResult{Tools: []ToolDefinition{{"my_tool", "tooling", InputSchema{Type: "object"}}}},
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
			for _, tool := range tt.tools() {
				s.Register(tool)
			}

			response, err := s.process(tt.args.request)

			if tt.wantErr != nil && err == nil {
				t.Errorf("expected error, got nil")
			}
			if err != nil && tt.wantErr == nil {
				t.Errorf("expected error %s, got nil", tt.wantErr.err)
			}
			if err != nil && tt.wantErr != nil {
				if !reflect.DeepEqual(err, tt.wantErr) {
					t.Errorf("process() got error = %v, want %v", err, tt.wantErr)
				}
			}

			if !reflect.DeepEqual(response, tt.wantResponse) {
				t.Errorf("process() got = %v, want %v", response, tt.wantResponse)
			}
		})
	}
}
