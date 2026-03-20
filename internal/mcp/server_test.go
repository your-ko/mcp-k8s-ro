package mcp

import "testing"

func TestServer_Register(t *testing.T) {
	type fields struct {
		name        string
		version     string
		contextName string
		clusterName string
		tools       []Tool
	}
	type args struct {
		tool Tool
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Server{
				name:        tt.fields.name,
				version:     tt.fields.version,
				contextName: tt.fields.contextName,
				clusterName: tt.fields.clusterName,
				tools:       tt.fields.tools,
			}
			s.Register(tt.args.tool)
		})
	}
}
