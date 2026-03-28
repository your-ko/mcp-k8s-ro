package k8s

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/mock"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
			name: "cluster-scoped resource with namespace emits warning",
			args: args{resource: "nodes", namespace: "kube-system"},
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
			wantContains: []string{"my-node", "warning", "nodes", "kube-system"},
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
		name            string
		args            args
		setupMapper     func(*mockrestMapper)
		setupDyn        func(*mockdynamicClient)
		wantContains    []string
		wantNotContains []string
		wantErr         error
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
				item.SetManagedFields([]v1.ManagedFieldsEntry{{Manager: "kubectl"}})
				gvr := schema.GroupVersionResource{Group: "", Version: "v1", Resource: "pods"}
				m.EXPECT().Resource(gvr).Return(&fakeResourceClient{item: &item})
			},
			wantContains:    []string{"my-pod", "test-context", "test-cluster"},
			wantNotContains: []string{"managedFields"},
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
			name: "cluster-scoped resource with namespace emits warning",
			args: args{name: "my-node", resource: "nodes", namespace: "kube-system"},
			setupMapper: func(m *mockrestMapper) {
				gvr := schema.GroupVersionResource{Group: "", Version: "v1", Resource: "nodes"}
				gvk := schema.GroupVersionKind{Group: "", Version: "v1", Kind: "Node"}
				m.EXPECT().ResourceFor(schema.GroupVersionResource{Resource: "nodes"}).Return(gvr, nil)
				m.EXPECT().KindsFor(gvr).Return([]schema.GroupVersionKind{gvk}, nil)
				m.EXPECT().RESTMapping(gvk.GroupKind(), mock.Anything).
					Return(restMappingFor(gvk, clusterScope{}), nil)
			},
			setupDyn: func(m *mockdynamicClient) {
				item := unstructured.Unstructured{}
				item.SetName("my-node")
				item.SetKind("Node")
				gvr := schema.GroupVersionResource{Group: "", Version: "v1", Resource: "nodes"}
				m.EXPECT().Resource(gvr).Return(&fakeResourceClient{item: &item})
			},
			wantContains: []string{"my-node", "warning", "nodes", "kube-system"},
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
				for _, s := range tt.wantNotContains {
					if strings.Contains(result, s) {
						t.Errorf("expected result NOT to contain %q, got:\n%s", s, result)
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
			name: "loadbalancer service sets externalIP from ip",
			item: func() unstructured.Unstructured {
				u := unstructured.Unstructured{}
				u.SetKind("Service")
				_ = unstructured.SetNestedField(u.Object, "LoadBalancer", "spec", "type")
				_ = unstructured.SetNestedField(u.Object, "10.0.0.2", "spec", "clusterIP")
				_ = unstructured.SetNestedSlice(u.Object, []interface{}{
					map[string]interface{}{"ip": "1.2.3.4"},
				}, "status", "loadBalancer", "ingress")
				return u
			}(),
			want: listResourcesOutput{Type: "LoadBalancer", ClusterIP: "10.0.0.2", ExternalIP: "1.2.3.4"},
		},
		{
			name: "loadbalancer service sets externalIP from hostname",
			item: func() unstructured.Unstructured {
				u := unstructured.Unstructured{}
				u.SetKind("Service")
				_ = unstructured.SetNestedField(u.Object, "LoadBalancer", "spec", "type")
				_ = unstructured.SetNestedField(u.Object, "10.0.0.3", "spec", "clusterIP")
				_ = unstructured.SetNestedSlice(u.Object, []interface{}{
					map[string]interface{}{"hostname": "abc.elb.amazonaws.com"},
				}, "status", "loadBalancer", "ingress")
				return u
			}(),
			want: listResourcesOutput{Type: "LoadBalancer", ClusterIP: "10.0.0.3", ExternalIP: "abc.elb.amazonaws.com"},
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

func Test_sanitize(t *testing.T) {
	tests := []struct {
		name        string
		input       func() *unstructured.Unstructured
		wantErr     bool
		assertField func(t *testing.T, item *unstructured.Unstructured)
	}{
		{
			name: "managedFields removed",
			input: func() *unstructured.Unstructured {
				u := &unstructured.Unstructured{Object: map[string]interface{}{}}
				_ = unstructured.SetNestedField(u.Object, []interface{}{"managed"}, "metadata", "managedFields")
				return u
			},
			assertField: func(t *testing.T, item *unstructured.Unstructured) {
				_, ok, _ := unstructured.NestedFieldNoCopy(item.Object, "metadata", "managedFields")
				if ok {
					t.Error("expected managedFields to be removed")
				}
			},
		},
		{
			name: "secret data values masked",
			input: func() *unstructured.Unstructured {
				u := &unstructured.Unstructured{Object: map[string]interface{}{}}
				u.SetKind("Secret")
				_ = unstructured.SetNestedField(u.Object, map[string]interface{}{
					"password": "super-secret",
					"token":    "abc123",
				}, "data")
				return u
			},
			assertField: func(t *testing.T, item *unstructured.Unstructured) {
				data, _, _ := unstructured.NestedMap(item.Object, "data")
				for k, v := range data {
					if v != "*****" {
						t.Errorf("expected data[%q] to be masked, got %q", k, v)
					}
				}
			},
		},
		{
			name: "secret stringData values masked",
			input: func() *unstructured.Unstructured {
				u := &unstructured.Unstructured{Object: map[string]interface{}{}}
				u.SetKind("Secret")
				_ = unstructured.SetNestedField(u.Object, map[string]interface{}{
					"api-key": "plaintext-secret",
				}, "stringData")
				return u
			},
			assertField: func(t *testing.T, item *unstructured.Unstructured) {
				sd, _, _ := unstructured.NestedMap(item.Object, "stringData")
				for k, v := range sd {
					if v != "*****" {
						t.Errorf("expected stringData[%q] to be masked, got %q", k, v)
					}
				}
			},
		},
		{
			name: "secret with no data is a no-op",
			input: func() *unstructured.Unstructured {
				u := &unstructured.Unstructured{Object: map[string]interface{}{}}
				u.SetKind("Secret")
				return u
			},
			assertField: func(t *testing.T, item *unstructured.Unstructured) {
				_, dataOk, _ := unstructured.NestedMap(item.Object, "data")
				_, sdOk, _ := unstructured.NestedMap(item.Object, "stringData")
				if dataOk || sdOk {
					t.Error("expected no data or stringData fields")
				}
			},
		},
		{
			name: "CSR spec.request redacted",
			input: func() *unstructured.Unstructured {
				u := &unstructured.Unstructured{Object: map[string]interface{}{}}
				u.SetKind("CertificateSigningRequest")
				_ = unstructured.SetNestedField(u.Object, "LS0tLS1CRUdJTi...", "spec", "request")
				return u
			},
			assertField: func(t *testing.T, item *unstructured.Unstructured) {
				v, _, _ := unstructured.NestedString(item.Object, "spec", "request")
				if v != "*****" {
					t.Errorf("expected spec.request to be masked, got %q", v)
				}
			},
		},
		{
			name: "CSR without spec.request is a no-op",
			input: func() *unstructured.Unstructured {
				u := &unstructured.Unstructured{Object: map[string]interface{}{}}
				u.SetKind("CertificateSigningRequest")
				_ = unstructured.SetNestedField(u.Object, "system:nodes", "spec", "signerName")
				return u
			},
			assertField: func(t *testing.T, item *unstructured.Unstructured) {
				_, ok, _ := unstructured.NestedFieldNoCopy(item.Object, "spec", "request")
				if ok {
					t.Error("expected no spec.request field to be written")
				}
			},
		},
		{
			name: "Certificate spec.keystores redacted",
			input: func() *unstructured.Unstructured {
				u := &unstructured.Unstructured{Object: map[string]interface{}{}}
				u.SetKind("Certificate")
				_ = unstructured.SetNestedField(u.Object, map[string]interface{}{
					"jks": map[string]interface{}{"create": true},
				}, "spec", "keystores")
				return u
			},
			assertField: func(t *testing.T, item *unstructured.Unstructured) {
				v, _, _ := unstructured.NestedString(item.Object, "spec", "keystores")
				if v != "*****" {
					t.Errorf("expected spec.keystores to be masked, got %q", v)
				}
			},
		},
		{
			name: "Certificate without keystores is a no-op",
			input: func() *unstructured.Unstructured {
				u := &unstructured.Unstructured{Object: map[string]interface{}{}}
				u.SetKind("Certificate")
				_ = unstructured.SetNestedField(u.Object, "letsencrypt", "spec", "issuerRef")
				return u
			},
			assertField: func(t *testing.T, item *unstructured.Unstructured) {
				_, ok, _ := unstructured.NestedFieldNoCopy(item.Object, "spec", "keystores")
				if ok {
					t.Error("expected no spec.keystores field to be written")
				}
			},
		},
		{
			name: "Certificate PEM condition message redacted",
			input: func() *unstructured.Unstructured {
				u := &unstructured.Unstructured{Object: map[string]interface{}{}}
				u.SetKind("Certificate")
				_ = unstructured.SetNestedSlice(u.Object, []interface{}{
					map[string]interface{}{
						"type":    "Ready",
						"status":  "True",
						"message": "-----BEGIN CERTIFICATE-----\nMIIBkTCB...\n-----END CERTIFICATE-----",
					},
				}, "status", "conditions")
				return u
			},
			assertField: func(t *testing.T, item *unstructured.Unstructured) {
				conditions, _, _ := unstructured.NestedSlice(item.Object, "status", "conditions")
				for _, c := range conditions {
					cm := c.(map[string]interface{})
					if msg, ok := cm["message"].(string); ok && msg != "*****" {
						t.Errorf("expected PEM condition message to be masked, got %q", msg)
					}
				}
			},
		},
		{
			name: "Certificate non-PEM condition message left intact",
			input: func() *unstructured.Unstructured {
				u := &unstructured.Unstructured{Object: map[string]interface{}{}}
				u.SetKind("Certificate")
				_ = unstructured.SetNestedSlice(u.Object, []interface{}{
					map[string]interface{}{
						"type":    "Ready",
						"status":  "True",
						"message": "Certificate is up to date and has not expired",
					},
				}, "status", "conditions")
				return u
			},
			assertField: func(t *testing.T, item *unstructured.Unstructured) {
				conditions, _, _ := unstructured.NestedSlice(item.Object, "status", "conditions")
				for _, c := range conditions {
					cm := c.(map[string]interface{})
					if msg, ok := cm["message"].(string); ok && msg == "*****" {
						t.Error("expected non-PEM condition message to be left intact")
					}
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			item := tt.input()
			_, err := sanitize(item)
			if tt.wantErr && err == nil {
				t.Fatal("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			tt.assertField(t, item)
		})
	}
}
