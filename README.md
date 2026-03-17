# mcp-k8s-ro

A read-only MCP server that gives Claude access to Kubernetes clusters. Built in Go, communicates over stdio using the MCP protocol.

## Constraints

- **Context-pinned** — the server locks to the active kubeconfig context at startup. Switching contexts in another terminal has no effect on the running server.
- **Read-only** — only `get`, `describe`, `logs`, and `top` style operations. No create, update, or delete.
- **Mutation suggestions** — if a mutating operation is needed, the server prints the equivalent `kubectl` command for the user to run manually.
- **Secret masking** — secret values are redacted by default.

## Tools

| Tool                      | Description                                                                                                                                                                                 |
|---------------------------|---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `k8s_list_resources`      | List any resource type by name — pods, deployments, CRDs, etc. Accepts optional namespace filter. Returns name, status, readiness, restarts, node, IP, and more depending on resource kind. |
| `k8s_describe_resource`   | Return the full YAML of a single resource. Secret data is masked.                                                                                                                           |
| `k8s_list_resource_types` | List all available resource types via the discovery API. Accepts optional API group filter.                                                                                                 |
| `k8s_get_logs`            | Fetch pod logs. Supports container selector, tail lines, and `--previous` for crashed containers.                                                                                           |
| `k8s_get_events`          | List Kubernetes events for a namespace or the whole cluster, sorted by most recent.                                                                                                         |
| `k8s_top_pods`            | CPU and memory usage per pod, with per-container breakdown. Requires metrics-server.                                                                                                        |
| `k8s_top_nodes`           | CPU and memory usage per node, with percentage of allocatable capacity. Requires metrics-server.                                                                                            |

## Configuration

| Environment variable | Default          | Description             |
|----------------------|------------------|-------------------------|
| `KUBECONFIG`         | `~/.kube/config` | Path to kubeconfig file |

## Build

```bash
make build
# binary is written to bin/mcp-k8s-ro
```

## Usage with Claude

Add the server to your Claude Desktop or `claude` CLI configuration:

```json
{
  "mcpServers": {
    "k8s": {
      "command": "/path/to/bin/mcp-k8s-ro",
      "env": {
        "KUBECONFIG": "/Users/you/.kube/config"
      }
    }
  }
}
```

The server locks to the current kubeconfig context at startup. The active context and cluster name are included in 
every tool response so you always know which cluster Claude is talking to.

