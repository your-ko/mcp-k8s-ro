package k8s

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/mock"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func TestClient_ListResources(t *testing.T) {
	podsGVR := schema.GroupVersionResource{Group: "", Version: "v1", Resource: "pods"}
	nodesGVR := schema.GroupVersionResource{Group: "", Version: "v1", Resource: "nodes"}
	podGVK := schema.GroupVersionKind{Group: "", Version: "v1", Kind: "Pod"}
	nodeGVK := schema.GroupVersionKind{Group: "", Version: "v1", Kind: "Node"}

	podItem := unstructured.Unstructured{}
	podItem.SetName("my-pod")
	podItem.SetNamespace("default")
	podItem.SetKind("Pod")

	nodeItem := unstructured.Unstructured{}
	nodeItem.SetName("my-node")
	nodeItem.SetKind("Node")

	type args struct {
		resource  string
		namespace string
	}
	tests := []struct {
		name        string
		args        args
		setupMapper func(*mockrestMapper)
		setupDyn    func(*mockdynamicClient)
		wantErr     error
	}{
		{
			name: "list pods in namespace",
			args: args{resource: "pods", namespace: "default"},
			setupMapper: func(m *mockrestMapper) {
				m.EXPECT().ResourceFor(schema.GroupVersionResource{Resource: "pods"}).Return(podsGVR, nil)
				m.EXPECT().KindsFor(podsGVR).Return([]schema.GroupVersionKind{podGVK}, nil)
				m.EXPECT().RESTMapping(podGVK.GroupKind(), mock.Anything).
					Return(restMappingFor(podGVK, namespacedScope{}), nil)
			},
			setupDyn: func(m *mockdynamicClient) {
				m.EXPECT().Resource(podsGVR).Return(&fakeResourceClient{
					list: &unstructured.UnstructuredList{Items: []unstructured.Unstructured{podItem}},
				})
			},
		},
		{
			name: "list nodes (cluster-scoped)",
			args: args{resource: "nodes", namespace: ""},
			setupMapper: func(m *mockrestMapper) {
				m.EXPECT().ResourceFor(schema.GroupVersionResource{Resource: "nodes"}).Return(nodesGVR, nil)
				m.EXPECT().KindsFor(nodesGVR).Return([]schema.GroupVersionKind{nodeGVK}, nil)
				m.EXPECT().RESTMapping(nodeGVK.GroupKind(), mock.Anything).
					Return(restMappingFor(nodeGVK, clusterScope{}), nil)
			},
			setupDyn: func(m *mockdynamicClient) {
				m.EXPECT().Resource(nodesGVR).Return(&fakeResourceClient{
					list: &unstructured.UnstructuredList{Items: []unstructured.Unstructured{nodeItem}},
				})
			},
		},
		{
			name: "namespaced resource with no namespace falls back to cluster-wide list",
			args: args{resource: "pods", namespace: ""},
			setupMapper: func(m *mockrestMapper) {
				m.EXPECT().ResourceFor(schema.GroupVersionResource{Resource: "pods"}).Return(podsGVR, nil)
				m.EXPECT().KindsFor(podsGVR).Return([]schema.GroupVersionKind{podGVK}, nil)
				m.EXPECT().RESTMapping(podGVK.GroupKind(), mock.Anything).
					Return(restMappingFor(podGVK, namespacedScope{}), nil)
			},
			setupDyn: func(m *mockdynamicClient) {
				m.EXPECT().Resource(podsGVR).Return(&fakeResourceClient{
					list: &unstructured.UnstructuredList{},
				})
			},
		},
		{
			name: "resolveGVR returns error",
			args: args{resource: "unknown", namespace: "default"},
			setupMapper: func(m *mockrestMapper) {
				m.EXPECT().ResourceFor(mock.Anything).
					Return(schema.GroupVersionResource{}, errors.New("no matches for kind"))
			},
			setupDyn: func(m *mockdynamicClient) {},
			wantErr:  errors.New("no matches for kind"),
		},
		{
			name: "dynamic List returns error",
			args: args{resource: "pods", namespace: "default"},
			setupMapper: func(m *mockrestMapper) {
				m.EXPECT().ResourceFor(schema.GroupVersionResource{Resource: "pods"}).Return(podsGVR, nil)
				m.EXPECT().KindsFor(podsGVR).Return([]schema.GroupVersionKind{podGVK}, nil)
				m.EXPECT().RESTMapping(podGVK.GroupKind(), mock.Anything).
					Return(restMappingFor(podGVK, namespacedScope{}), nil)
			},
			setupDyn: func(m *mockdynamicClient) {
				m.EXPECT().Resource(podsGVR).Return(&fakeResourceClient{
					err: errors.New("connection refused"),
				})
			},
			wantErr: errors.New("connection refused"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mapper := newMockrestMapper(t)
			dyn := newMockdynamicClient(t)
			tt.setupMapper(mapper)
			tt.setupDyn(dyn)

			c := &Client{
				mapper:      mapper,
				dynamic:     dyn,
				contextName: "test-context",
				clusterName: "test-cluster",
			}

			_, err := c.ListResources(context.Background(), tt.args.resource, tt.args.namespace)

			if tt.wantErr == nil {
				if err != nil {
					t.Fatalf("expected no error, got %s", err)
				}
				return
			}

			if err == nil {
				t.Fatalf("expected error %v, got nil", tt.wantErr)
			}

			if tt.wantErr.Error() != err.Error() {
				t.Fatalf("expected error message:\n%q\ngot:\n%q", tt.wantErr.Error(), err.Error())
			}
		})
	}
}
