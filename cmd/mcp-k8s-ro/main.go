package main

import (
	"log/slog"
	"os"
	"time"

	"github.com/your-ko/mcp-k8s-ro/internal/k8s"
	"github.com/your-ko/mcp-k8s-ro/internal/mcp"
	"github.com/your-ko/mcp-k8s-ro/internal/tools"
	"k8s.io/client-go/tools/clientcmd"
)

var GitCommit string
var Version string
var BuildDate string

func main() {

	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	slog.SetDefault(logger)
	slog.Info("MCP K8S RO starting...")
	updateMetadata()
	slog.Info("Starting", "Version", Version, "BuildDate", BuildDate, "GitCommit", GitCommit)

	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	clientConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, &clientcmd.ConfigOverrides{})

	config, err := clientConfig.ClientConfig()
	if err != nil {
		slog.Error("failed to build rest config", "error", err)
		os.Exit(1)
	}
	apiConfig, err := clientConfig.RawConfig()
	if err != nil {
		slog.Error("failed to load kubeconfig", "error", err)
		os.Exit(1)
	}

	contextName := apiConfig.CurrentContext
	k8sCtx, ok := apiConfig.Contexts[contextName]
	if !ok {
		slog.Error("current context not found in kubeconfig", "context", contextName)
		os.Exit(1)
	}
	clusterName := k8sCtx.Cluster

	k8sClient, err := k8s.NewClient(config, contextName, clusterName)
	if err != nil {
		slog.Error("failed to create k8s client", "error", err)
		os.Exit(1)
	}

	server := mcp.New("mcp-k8s-ro", Version, contextName, clusterName)
	server.Register(tools.NewResourcesLister(k8sClient))
	server.Register(tools.NewResourceDescriber(k8sClient))
	server.Register(tools.NewResourceTypesLister(k8sClient))
	server.Register(tools.NewLogGetter(k8sClient))
	server.Register(tools.NewEventGetter(k8sClient))
	server.Register(tools.NewPodTopper(k8sClient))
	server.Register(tools.NewNodeTopper(k8sClient))

	server.Start(os.Stdin, os.Stdout)
}

func updateMetadata() {
	if Version == "" {
		Version = "dev"
	}
	if BuildDate == "" {
		BuildDate = time.Now().Format(time.RFC822)
	}
}
