[![Main](https://github.com/your-ko/mcp-k8s-ro/actions/workflows/main.yaml/badge.svg)](https://github.com/your-ko/mcp-k8s-ro/actions/workflows/main.yaml)
[![golangci-lint](https://github.com/your-ko/mcp-k8s-ro/actions/workflows/golangci-lint.yaml/badge.svg)](https://github.com/your-ko/mcp-k8s-ro/actions/workflows/golangci-lint.yaml)
[![Link validation](https://github.com/your-ko/mcp-k8s-ro/actions/workflows/link-validator.yaml/badge.svg)](https://github.com/your-ko/mcp-k8s-ro/actions/workflows/link-validator.yaml)
# mcp-k8s-ro

A read-only MCP server that gives Claude access to Kubernetes clusters. Built in Go, communicates over stdio using the MCP protocol.

## Why

- **Safe by design** — read-only, so Claude can never accidentally mutate your cluster.
- **Token-efficient** — responses include only relevant fields (name, status, restarts, etc.) rather than raw Kubernetes API objects, which saves significant context space.
- **Cluster-aware** — every response includes the active context and cluster name, so Claude always knows which cluster it's looking at.
- **Secret-safe** — secret values are masked before being sent to the model.
- **No extra infra** — runs as a local binary or Docker container, connects to whatever kubeconfig context is active at startup.

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

## Usage with Claude

### Binary

Build the binary and add it to your Claude Desktop or `claude` CLI configuration:

```bash
make build
# binary is written to bin/mcp-k8s-ro
```

```json
{
  "mcpServers": {
    "k8s": {
      "type" : "stdio",
      "command": "/path/to/bin/mcp-k8s-ro",
      "env": {
        "KUBECONFIG": "/path/to/.kube/config"
      }
    }
  }
}
```
or execute `claude mcp add --transport stdio --scope user mcp-k8s-ro [path to binary]`
### Docker

Pull the image from GitHub Container Registry:

```bash
docker pull ghcr.io/your-ko/mcp-k8s-ro:latest 
```
or pin a particular version (recommended)

Add it to your Claude Desktop or `claude` CLI configuration. The kubeconfig directory is mounted read-only into the container:

```json
{
  "mcpServers": {
    "k8s": {
      "command": "docker",
      "args": [
        "run", "--rm", "-i",
        "-v", "/path/to/.kube:/home/nonroot/.kube:ro",
        "ghcr.io/your-ko/mcp-k8s-ro:latest"
      ]
    }
  }
}
```

If your kubeconfig is in a non-standard location, pass it via `KUBECONFIG`:

```json
{
  "mcpServers": {
    "k8s": {
      "command": "docker",
      "args": [
        "run", "--rm", "-i",
        "-e", "KUBECONFIG=/config/my-kubeconfig",
        "-v", "/path/to/my-kubeconfig:/config/my-kubeconfig:ro",
        "ghcr.io/your-ko/mcp-k8s-ro:latest"
      ]
    }
  }
}
```

The server locks to the current kubeconfig context at startup. 
The active context and cluster name are included in every tool response so you always know which 
cluster Claude is talking to.

