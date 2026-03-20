package mcp

import "testing"

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
