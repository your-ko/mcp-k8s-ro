package main

import (
	"bufio"
	"encoding/json"
	"log/slog"
	"os"

	"github.com/your-ko/mcp-k8s-ro/internal/mcp"
)

var GitCommit string
var Version string
var BuildDate string

func main() {
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		line := scanner.Bytes()

		request := mcp.JSONRPCRequest{}
		err := json.Unmarshal(line, &request)
		if err != nil {
			slog.With("error", err).Error("error unmarshalling request")
		}
		slog.Debug("request: %v", request)

		response := mcp.process(request)
		slog.Debug("response: %v", response)
		err = json.NewEncoder(os.Stdout).Encode(response)
		if err != nil {
			slog.With("error", err).Error("error unmarshalling request")
		}
	}
}
