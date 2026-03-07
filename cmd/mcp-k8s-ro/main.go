package main

import (
	"github.com/your-ko/mcp-k8s-ro/internal/mcp"
)

var GitCommit string
var Version string
var BuildDate string

func main() {
	server := mcp.New("mcp-k8s-ro", Version)
	server.Start()
}
