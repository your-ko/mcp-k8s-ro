package k8s

import (
	"context"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/dynamic"
)

// fakeResourceClient is a minimal dynamic.NamespaceableResourceInterface that
// returns a fixed list and supports Namespace() chaining.
type fakeResourceClient struct {
	list *unstructured.UnstructuredList
	err  error
}

func (f *fakeResourceClient) Namespace(_ string) dynamic.ResourceInterface { return f }

func (f *fakeResourceClient) List(_ context.Context, _ metav1.ListOptions) (*unstructured.UnstructuredList, error) {
	return f.list, f.err
}

func (f *fakeResourceClient) Create(_ context.Context, _ *unstructured.Unstructured, _ metav1.CreateOptions, _ ...string) (*unstructured.Unstructured, error) {
	return nil, nil
}
func (f *fakeResourceClient) Update(_ context.Context, _ *unstructured.Unstructured, _ metav1.UpdateOptions, _ ...string) (*unstructured.Unstructured, error) {
	return nil, nil
}
func (f *fakeResourceClient) UpdateStatus(_ context.Context, _ *unstructured.Unstructured, _ metav1.UpdateOptions) (*unstructured.Unstructured, error) {
	return nil, nil
}
func (f *fakeResourceClient) Delete(_ context.Context, _ string, _ metav1.DeleteOptions, _ ...string) error {
	return nil
}
func (f *fakeResourceClient) DeleteCollection(_ context.Context, _ metav1.DeleteOptions, _ metav1.ListOptions) error {
	return nil
}
func (f *fakeResourceClient) Get(_ context.Context, _ string, _ metav1.GetOptions, _ ...string) (*unstructured.Unstructured, error) {
	return nil, nil
}
func (f *fakeResourceClient) Watch(_ context.Context, _ metav1.ListOptions) (watch.Interface, error) {
	return nil, nil
}
func (f *fakeResourceClient) Patch(_ context.Context, _ string, _ types.PatchType, _ []byte, _ metav1.PatchOptions, _ ...string) (*unstructured.Unstructured, error) {
	return nil, nil
}
func (f *fakeResourceClient) Apply(_ context.Context, _ string, _ *unstructured.Unstructured, _ metav1.ApplyOptions, _ ...string) (*unstructured.Unstructured, error) {
	return nil, nil
}
func (f *fakeResourceClient) ApplyStatus(_ context.Context, _ string, _ *unstructured.Unstructured, _ metav1.ApplyOptions) (*unstructured.Unstructured, error) {
	return nil, nil
}

// namespacedScope and clusterScope implement meta.RESTScope.
type namespacedScope struct{}

func (namespacedScope) Name() meta.RESTScopeName { return meta.RESTScopeNameNamespace }

type clusterScope struct{}

func (clusterScope) Name() meta.RESTScopeName { return meta.RESTScopeNameRoot }

func restMappingFor(gvk schema.GroupVersionKind, scope meta.RESTScope) *meta.RESTMapping {
	return &meta.RESTMapping{
		Resource:         schema.GroupVersionResource{Group: gvk.Group, Version: gvk.Version},
		GroupVersionKind: gvk,
		Scope:            scope,
	}
}
