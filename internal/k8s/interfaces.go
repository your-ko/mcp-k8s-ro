package k8s

import (
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
)

// what ListResources / GetResource need
type dynamicClient interface {
	Resource(gvr schema.GroupVersionResource) dynamic.NamespaceableResourceInterface
}

// what GetApiResources needs
type discoveryClient interface {
	ServerPreferredResources() ([]*metav1.APIResourceList, error)
}

// what resolveGVR needs
type restMapper interface {
	ResourceFor(input schema.GroupVersionResource) (schema.GroupVersionResource, error)
	KindsFor(input schema.GroupVersionResource) ([]schema.GroupVersionKind, error)
	RESTMapping(gk schema.GroupKind, versions ...string) (*meta.RESTMapping, error)
}
