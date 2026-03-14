package main

import (
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/your-ko/mcp-k8s-ro/internal/k8s"
	"github.com/your-ko/mcp-k8s-ro/internal/mcp"
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
	slog.Info(fmt.Sprintf("Version: %s, BuildDate: %s, GitCommit: %s", Version, BuildDate, GitCommit))

	config, err := getConfig()

	if err != nil {
		slog.With("error", err)
		os.Exit(1)
	}
	rules := clientcmd.NewDefaultClientConfigLoadingRules()
	apiConfig, err := rules.Load()
	if err != nil {
		os.Exit(1)
	}
	contextName := apiConfig.CurrentContext
	clusterName := apiConfig.Contexts[contextName].Cluster

	k8sClient, err := k8s.NewClient(config, contextName, clusterName)
	if err != nil {
		slog.With("error", err).Error("failed to create k8s client")
		os.Exit(1)
	}

	server := mcp.New("mcp-k8s-ro", Version, contextName, clusterName)
	server.Register(k8s.NewResourcesLister(k8sClient))
	server.Register(k8s.NewResourceDescriber(k8sClient))
	server.Register(k8s.NewResourceTypesLister(k8sClient))
	server.Register(k8s.NewLogGetter(k8sClient))
	server.Register(k8s.NewEventGetter(k8sClient))
	server.Start()
}

func updateMetadata() {
	if Version == "" {
		Version = "dev"
	}
	if BuildDate == "" {
		BuildDate = time.Now().Format(time.RFC822)
	}
}
