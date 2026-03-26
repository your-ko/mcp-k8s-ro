package k8s

import (
	"context"
	"errors"
	"strings"
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
		name         string
		args         args
		setupMapper  func(*mockrestMapper)
		setupDyn     func(*mockdynamicClient)
		wantContains []string
		wantErr      error
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
			wantContains: []string{"my-pod", "test-context", "test-cluster"},
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
			wantContains: []string{"my-node", "test-context", "test-cluster"},
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
			wantContains: []string{"test-context", "test-cluster"},
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

			result, err := c.ListResources(context.Background(), tt.args.resource, tt.args.namespace)

			if tt.wantErr == nil {
				if err != nil {
					t.Fatalf("expected no error, got %s", err)
				}
				for _, s := range tt.wantContains {
					if !strings.Contains(result, s) {
						t.Errorf("expected result to contain %q, got:\n%s", s, result)
					}
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

func TestClient_GetResource(t *testing.T) {
	type args struct {
		name      string
		resource  string
		namespace string
	}
	tests := []struct {
		name         string
		args         args
		setupMapper  func(*mockrestMapper)
		setupDyn     func(*mockdynamicClient)
		wantContains []string
		wantErr      error
	}{
		{
			name: "get pod returns yaml with name and header",
			args: args{name: "my-pod", resource: "pods", namespace: "default"},
			setupMapper: func(m *mockrestMapper) {
				gvr := schema.GroupVersionResource{Group: "", Version: "v1", Resource: "pods"}
				gvk := schema.GroupVersionKind{Group: "", Version: "v1", Kind: "Pod"}
				m.EXPECT().ResourceFor(schema.GroupVersionResource{Resource: "pods"}).Return(gvr, nil)
				m.EXPECT().KindsFor(gvr).Return([]schema.GroupVersionKind{gvk}, nil)
				m.EXPECT().RESTMapping(gvk.GroupKind(), mock.Anything).
					Return(restMappingFor(gvk, namespacedScope{}), nil)
			},
			setupDyn: func(m *mockdynamicClient) {
				item := unstructured.Unstructured{}
				item.SetName("my-pod")
				item.SetNamespace("default")
				item.SetKind("Pod")
				gvr := schema.GroupVersionResource{Group: "", Version: "v1", Resource: "pods"}
				m.EXPECT().Resource(gvr).Return(&fakeResourceClient{item: &item})
			},
			wantContains: []string{"my-pod", "test-context", "test-cluster"},
		},
		{
			name: "get secret masks data values",
			args: args{name: "my-secret", resource: "secrets", namespace: "default"},
			setupMapper: func(m *mockrestMapper) {
				gvr := schema.GroupVersionResource{Group: "", Version: "v1", Resource: "secrets"}
				gvk := schema.GroupVersionKind{Group: "", Version: "v1", Kind: "Secret"}
				m.EXPECT().ResourceFor(schema.GroupVersionResource{Resource: "secrets"}).Return(gvr, nil)
				m.EXPECT().KindsFor(gvr).Return([]schema.GroupVersionKind{gvk}, nil)
				m.EXPECT().RESTMapping(gvk.GroupKind(), mock.Anything).
					Return(restMappingFor(gvk, namespacedScope{}), nil)
			},
			setupDyn: func(m *mockdynamicClient) {
				item := unstructured.Unstructured{}
				item.SetName("my-secret")
				item.SetKind("Secret")
				_ = unstructured.SetNestedField(item.Object, map[string]interface{}{
					"password": "super-secret",
				}, "data")
				gvr := schema.GroupVersionResource{Group: "", Version: "v1", Resource: "secrets"}
				m.EXPECT().Resource(gvr).Return(&fakeResourceClient{item: &item})
			},
			wantContains: []string{"my-secret", "*****"},
		},
		{
			name: "resolveGVR returns error",
			args: args{name: "my-pod", resource: "unknown", namespace: "default"},
			setupMapper: func(m *mockrestMapper) {
				m.EXPECT().ResourceFor(mock.Anything).
					Return(schema.GroupVersionResource{}, errors.New("no matches for kind"))
			},
			setupDyn: func(m *mockdynamicClient) {},
			wantErr:  errors.New("no matches for kind"),
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

			result, err := c.GetResource(context.Background(), tt.args.name, tt.args.resource, tt.args.namespace)

			if tt.wantErr == nil {
				if err != nil {
					t.Fatalf("expected no error, got %s", err)
				}
				for _, s := range tt.wantContains {
					if !strings.Contains(result, s) {
						t.Errorf("expected result to contain %q, got:\n%s", s, result)
					}
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

func Test_setResourceSpecificFields(t *testing.T) {
	tests := []struct {
		name string
		item unstructured.Unstructured
		want listResourcesOutput
	}{
		{
			name: "secret sets type",
			item: func() unstructured.Unstructured {
				u := unstructured.Unstructured{}
				u.SetKind("Secret")
				_ = unstructured.SetNestedField(u.Object, "kubernetes.io/tls", "type")
				return u
			}(),
			want: listResourcesOutput{Type: "kubernetes.io/tls"},
		},
		{
			name: "service sets type, clusterIP and ports",
			item: func() unstructured.Unstructured {
				u := unstructured.Unstructured{}
				u.SetKind("Service")
				_ = unstructured.SetNestedField(u.Object, "ClusterIP", "spec", "type")
				_ = unstructured.SetNestedField(u.Object, "10.0.0.1", "spec", "clusterIP")
				_ = unstructured.SetNestedSlice(u.Object, []interface{}{
					map[string]interface{}{"port": int64(80), "protocol": "TCP"},
					map[string]interface{}{"port": int64(443), "protocol": "TCP"},
				}, "spec", "ports")
				return u
			}(),
			want: listResourcesOutput{Type: "ClusterIP", ClusterIP: "10.0.0.1", Ports: "80/TCP,443/TCP"},
		},
		{
			name: "pod sets node, podIP, ready count and restarts",
			item: func() unstructured.Unstructured {
				u := unstructured.Unstructured{}
				u.SetKind("Pod")
				_ = unstructured.SetNestedField(u.Object, "node-1", "spec", "nodeName")
				_ = unstructured.SetNestedField(u.Object, "10.1.2.3", "status", "podIP")
				_ = unstructured.SetNestedSlice(u.Object, []interface{}{
					map[string]interface{}{"ready": true, "restartCount": int64(2), "state": map[string]interface{}{}, "lastState": map[string]interface{}{}},
					map[string]interface{}{"ready": false, "restartCount": int64(1), "state": map[string]interface{}{}, "lastState": map[string]interface{}{}},
				}, "status", "containerStatuses")
				return u
			}(),
			want: listResourcesOutput{Node: "node-1", PodIP: "10.1.2.3", Ready: "1/2", Restarts: 3},
		},
		{
			name: "pod sets waiting state reason and last termination reason",
			item: func() unstructured.Unstructured {
				u := unstructured.Unstructured{}
				u.SetKind("Pod")
				_ = unstructured.SetNestedSlice(u.Object, []interface{}{
					map[string]interface{}{
						"ready":        false,
						"restartCount": int64(5),
						"state": map[string]interface{}{
							"waiting": map[string]interface{}{"reason": "CrashLoopBackOff"},
						},
						"lastState": map[string]interface{}{
							"terminated": map[string]interface{}{"reason": "OOMKilled"},
						},
					},
				}, "status", "containerStatuses")
				return u
			}(),
			want: listResourcesOutput{Ready: "0/1", Restarts: 5, StateReason: "CrashLoopBackOff", LastTerminationReason: "OOMKilled"},
		},
		{
			name: "deployment sets ready replicas",
			item: func() unstructured.Unstructured {
				u := unstructured.Unstructured{}
				u.SetKind("Deployment")
				_ = unstructured.SetNestedField(u.Object, int64(3), "spec", "replicas")
				_ = unstructured.SetNestedField(u.Object, int64(2), "status", "readyReplicas")
				return u
			}(),
			want: listResourcesOutput{Ready: "2/3"},
		},
		{
			name: "statefulset sets ready replicas",
			item: func() unstructured.Unstructured {
				u := unstructured.Unstructured{}
				u.SetKind("StatefulSet")
				_ = unstructured.SetNestedField(u.Object, int64(3), "spec", "replicas")
				_ = unstructured.SetNestedField(u.Object, int64(3), "status", "readyReplicas")
				return u
			}(),
			want: listResourcesOutput{Ready: "3/3"},
		},
		{
			name: "daemonset sets ready replicas",
			item: func() unstructured.Unstructured {
				u := unstructured.Unstructured{}
				u.SetKind("DaemonSet")
				_ = unstructured.SetNestedField(u.Object, int64(5), "spec", "replicas")
				_ = unstructured.SetNestedField(u.Object, int64(4), "status", "readyReplicas")
				return u
			}(),
			want: listResourcesOutput{Ready: "4/5"},
		},
		{
			name: "node sets IPs and ready status",
			item: func() unstructured.Unstructured {
				u := unstructured.Unstructured{}
				u.SetKind("Node")
				_ = unstructured.SetNestedSlice(u.Object, []interface{}{
					map[string]interface{}{"type": "InternalIP", "address": "192.168.1.10"},
					map[string]interface{}{"type": "ExternalIP", "address": "1.2.3.4"},
				}, "status", "addresses")
				_ = unstructured.SetNestedSlice(u.Object, []interface{}{
					map[string]interface{}{"type": "Ready", "status": "True"},
				}, "status", "conditions")
				return u
			}(),
			want: listResourcesOutput{InternalIP: "192.168.1.10", ExternalIP: "1.2.3.4", Ready: "Ready"},
		},
		{
			name: "node with disk pressure shows problem in status",
			item: func() unstructured.Unstructured {
				u := unstructured.Unstructured{}
				u.SetKind("Node")
				_ = unstructured.SetNestedSlice(u.Object, []interface{}{
					map[string]interface{}{"type": "Ready", "status": "True"},
					map[string]interface{}{"type": "DiskPressure", "status": "True"},
				}, "status", "conditions")
				return u
			}(),
			want: listResourcesOutput{Ready: "Ready", Status: "DiskPressure"},
		},
		{
			name: "node not ready",
			item: func() unstructured.Unstructured {
				u := unstructured.Unstructured{}
				u.SetKind("Node")
				_ = unstructured.SetNestedSlice(u.Object, []interface{}{
					map[string]interface{}{"type": "Ready", "status": "False"},
				}, "status", "conditions")
				return u
			}(),
			want: listResourcesOutput{Ready: "NotReady"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := listResourcesOutput{}
			setResourceSpecificFields(tt.item, &got)

			if got != tt.want {
				t.Errorf("setResourceSpecificFields() =\n%+v\nwant\n%+v", got, tt.want)
			}
		})
	}
}
