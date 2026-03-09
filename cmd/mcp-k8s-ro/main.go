package main

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/your-ko/mcp-k8s-ro/internal/k8s"
	"github.com/your-ko/mcp-k8s-ro/internal/mcp"
	"k8s.io/client-go/kubernetes"
)

var GitCommit string
var Version string
var BuildDate string

func main() {

	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	slog.SetDefault(logger)
	slog.Info("MCP K8S RO starting...")
	slog.Info(fmt.Sprintf("Version: %s, BuildDate: %s, GitCommit: %s", Version, BuildDate, GitCommit))

	config, err := getConfig()
	if err != nil {
		slog.With("error", err)
		os.Exit(1)
	}
	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		slog.With("error", err)
		os.Exit(1)
	}

	server := mcp.New("mcp-k8s-ro", Version)
	server.Register(k8s.GetNamespace{Client: client})
	server.Start()
}
