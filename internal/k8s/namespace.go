package k8s

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/your-ko/mcp-k8s-ro/internal/mcp"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// TODO: also add "params":{"name":"k8s_describe_namespace","arguments":{"name":"kube-system"}}

type GetNamespace struct {
	Client *kubernetes.Clientset
}

func (GetNamespace) Name() string        { return "k8s_get_namespaces" }
func (GetNamespace) Description() string { return "List all namespaces in the cluster" }
func (GetNamespace) InputSchema() mcp.InputSchema {
	return mcp.InputSchema{
		Type: "object",
	}
}

func (n GetNamespace) Execute(params json.RawMessage) (string, error) {
	// TODO: improve context
	// TODO: improve listOptions
	namespaceList, err := n.Client.CoreV1().Namespaces().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return "", err
	}
	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("%-40s %-10s %s\n", "NAME", "STATUS", "AGE"))
	for _, ns := range namespaceList.Items {
		builder.WriteString(fmt.Sprintf("%-40s %-10s %s\n", ns.Name, ns.Status.Phase, ns.CreationTimestamp.UTC().Format("2006-01-02")))
	}
	return builder.String(), nil
}
