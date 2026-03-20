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
		tools        func(t *testing.T) []Tool
		wantResponse any
		wantErr      *rpcError
	}{
		{
			name:         "initialise",
			args:         args{},
			wantResponse: nil,
			wantErr:      nil,
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
			if !reflect.DeepEqual(err, tt.wantResponse) {
				t.Errorf("process() got1 = %v, want %v", err, tt.wantResponse)
			}
		})
	}
}
