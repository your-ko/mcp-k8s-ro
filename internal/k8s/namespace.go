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

type GetNamespace struct{}

func (GetNamespace) Name() string        { return "k8s_get_namespaces" }
func (GetNamespace) Description() string { return "List all namespaces in the cluster" }
func (GetNamespace) InputSchema() mcp.InputSchema {
	return mcp.InputSchema{
		Type: "object",
	}
}

func (n GetNamespace) Execute(params json.RawMessage) (string, error) {
	config, err := getConfig()
	if err != nil {
		return "", err
	}
	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err)
	}
	// TODO: improve context
	// TODO: improve listOptions
	namespaceList, err := client.CoreV1().Namespaces().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return "", err
	}
	var builder strings.Builder
	for _, ns := range namespaceList.Items {
		builder.WriteString(fmt.Sprintf("%s\n", ns.Name))
	}
	return builder.String(), nil
}
