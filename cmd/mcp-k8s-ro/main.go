package main

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/your-ko/mcp-k8s-ro/internal/mcp"
)

var GitCommit string
var Version string
var BuildDate string

func main() {

	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	slog.SetDefault(logger)
	slog.Info("MCP K8S RO starting...")
	slog.Info(fmt.Sprintf("Version: %s, BuildDate: %s, GitCommit: %s", Version, BuildDate, GitCommit))

	server := mcp.New("mcp-k8s-ro", Version)
	server.Start()
}
