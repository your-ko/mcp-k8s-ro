# mcp-k8s

This is an MCP project to allow Claude to execute commands on K8s. 

Limitations:
* MCP can work *ONLY* with the current context. So it won't happen accidentally that user changed context and didn't realize it.
* MCP provide *ONLY* read-only operations, such as `kubectl get pods`. No create/update/delete operations are allowed.
* If there is a need to create/update/delete operation, then this operation needs to be printed, 
  so user will have to explicitly enter it themselves
* Secret outputs are masked by default
* Should be able to run as a go binary and as a docker container

Language - Go

## List of tools
Complete:
- k8s_list_resources — list any resource type including CRDs
- k8s_describe_resource — get full YAML of any resource
- k8s_list_resource_types — discovery API, lists all available kinds
- k8s_get_logs — pod logs with tail support
- k8s_get_events — cluster/namespace events, sorted by time
- k8s_top_pods — CPU/memory per pod + per container
- k8s_top_nodes — CPU/memory per node with % of allocatable
