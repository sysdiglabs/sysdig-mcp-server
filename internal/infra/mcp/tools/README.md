# MCP Tools & Permissions

The handler filters tools dynamically based on the Sysdig user's permissions. Each tool declares mandatory permissions via `WithRequiredPermissions`.

| Tool | File | Capability | Required Permissions | Useful Prompts |
| --- | --- | --- | --- | --- |
| `list_runtime_events` | `tool_list_runtime_events.go` | Query runtime events with filters, cursor, scope. | `policy-events.read` | “Show high severity runtime events from last 2h.” |
| `get_event_info` | `tool_get_event_info.go` | Pull full payload for a single policy event. | `policy-events.read` | “Fetch event `abc123` details.” |
| `get_event_process_tree` | `tool_get_event_process_tree.go` | Retrieve the process tree for an event when available. | `policy-events.read` | “Show the process tree behind event `abc123`.” |
| `run_sysql` | `tool_run_sysql.go` | Execute caller-supplied Sysdig SysQL queries safely. | `sage.exec`, `risks.read` | “Run the following SysQL…”. |
| `generate_sysql` | `tool_generate_sysql.go` | Convert natural language to SysQL via Sysdig Sage. | `sage.exec` (does not work with Service Accounts) | “Create a SysQL to list S3 buckets.” |
| `kubernetes_list_clusters` | `tool_kubernetes_list_clusters.go` | Lists Kubernetes cluster information. | `metrics-data.read` | "List all Kubernetes clusters" |
| `kubernetes_list_nodes` | `tool_kubernetes_list_nodes.go` | Lists Kubernetes node information. | `metrics-data.read` | "List all Kubernetes nodes in the cluster 'production-gke'" |
| `kubernetes_list_workloads` | `tool_kubernetes_list_workloads.go` | Lists Kubernetes workload information. | `metrics-data.read` | "List all desired workloads in the cluster 'production-gke' and namespace 'default'" |
| `kubernetes_list_pod_containers` | `tool_kubernetes_list_pod_containers.go` | Retrieves information from a particular pod and container. | `metrics-data.read` | "Show me info for pod 'my-pod' in cluster 'production-gke'" |
| `kubernetes_list_cronjobs` | `tool_kubernetes_list_cronjobs.go` | Retrieves information from the cronjobs in the cluster. | `metrics-data.read` | "List all cronjobs in cluster 'prod' and namespace 'default'" |
| `troubleshoot_kubernetes_list_top_unavailable_pods` | `tool_troubleshoot_kubernetes_list_top_unavailable_pods.go` | Shows the top N pods with the highest number of unavailable or unready replicas. | `metrics-data.read` | "Show the top 20 unavailable pods in cluster 'production'" |
| `troubleshoot_kubernetes_list_top_restarted_pods` | `tool_troubleshoot_kubernetes_list_top_restarted_pods.go` | Lists the pods with the highest number of container restarts. | `metrics-data.read` | "Show the top 10 pods with the most container restarts in cluster 'production'" |
| `troubleshoot_kubernetes_list_top_400_500_http_errors_in_pods` | `tool_troubleshoot_kubernetes_list_top_400_500_http_errors_in_pods.go` | Lists the pods with the highest rate of HTTP 4xx and 5xx errors over a specified time interval. | `metrics-data.read` | "Show the top 20 pods with the most HTTP errors in cluster 'production'" |
| `troubleshoot_kubernetes_list_top_network_errors_in_pods` | `tool_troubleshoot_kubernetes_list_top_network_errors_in_pods.go` | Shows the top network errors by pod over a given interval. | `metrics-data.read` | "Show the top 10 pods with the most network errors in cluster 'production'" |
| `troubleshoot_kubernetes_list_count_pods_per_cluster` | `tool_troubleshoot_kubernetes_list_count_pods_per_cluster.go` | List the count of running Kubernetes Pods grouped by cluster and namespace. | `metrics-data.read` | "List the count of running Kubernetes Pods in cluster 'production'" |
| `troubleshoot_kubernetes_list_underutilized_pods_by_cpu_quota` | `tool_troubleshoot_kubernetes_list_underutilized_pods_by_cpu_quota.go` | List Kubernetes pods with CPU usage below 25% of the quota limit. | `metrics-data.read` | "Show the top 10 underutilized pods by CPU quota in cluster 'production'" |
| `troubleshoot_kubernetes_list_underutilized_pods_by_memory_quota` | `tool_troubleshoot_kubernetes_list_underutilized_pods_by_memory_quota.go` | List Kubernetes pods with memory usage below 25% of the limit. | `metrics-data.read` | "Show the top 10 underutilized pods by memory quota in cluster 'production'" |
| `troubleshoot_kubernetes_list_top_cpu_consumed_by_workload` | `tool_troubleshoot_kubernetes_list_top_cpu_consumed_by_workload.go` | Identifies the Kubernetes workloads (all containers) consuming the most CPU (in cores). | `metrics-data.read` | "Show the top 10 workloads consuming the most CPU in cluster 'production'" |
| `troubleshoot_kubernetes_list_top_cpu_consumed_by_container` | `tool_troubleshoot_kubernetes_list_top_cpu_consumed_by_container.go` | Identifies the Kubernetes containers consuming the most CPU (in cores). | `metrics-data.read` | "Show the top 10 containers consuming the most CPU in cluster 'production'" |
| `troubleshoot_kubernetes_list_top_memory_consumed_by_workload` | `tool_troubleshoot_kubernetes_list_top_memory_consumed_by_workload.go` | Lists memory-intensive workloads (all containers). | `metrics-data.read` | "Show the top 10 workloads consuming the most memory in cluster 'production'" |
| `troubleshoot_kubernetes_list_top_memory_consumed_by_container` | `tool_troubleshoot_kubernetes_list_top_memory_consumed_by_container.go` | Lists memory-intensive containers. | `metrics-data.read` | "Show the top 10 containers consuming the most memory in cluster 'production'" |

# Adding a New Tool

1.  **See other tools:** Check how other tools are implemented so you can have the context on how they should look like.

2.  **Create Files:** Add `tool_<name>.go` and `tool_<name>_test.go` in `internal/infra/mcp/tools/`.

3.  **Implement the Tool:**
    *   Define a struct that holds the Sysdig client, or any required collaborator.
    *   Implement the `handle` method, which contains the tool's core logic.
    *   Implement the `RegisterInServer` method to define the tool's MCP schema, including its name, description, parameters, and required permissions. Use helpers from `utils.go`.
    *   If a tool does not have any required permission, just specify `WithRequiredPermissions()`.
        If the tool requires one or multiple permissions, specify them like `WithRequiredPermissions("a.permission", "another.permission")`.

4.  **Write Tests:** Use Ginkgo/Gomega to write BDD-style tests. Mock the Sysdig client to cover:
    - Parameter validation
    - Permission metadata
    - Sysdig API client interactions (mocked)
    - Error handling

5.  **Register the Tool:** Add the new tool to `setupHandler()` in `cmd/server/main.go`.

6.  **Document:** Add the new tool to the README.md and the table in this document.


## Testing Philosophy

- Use BDD-style tests with Ginkgo/Gomega
- Each tool requires comprehensive test coverage for:
  - Parameter validation (all possible combinations need to be tested)
  - Permission metadata
  - Sysdig API client interactions (mocked using go-mock)
  - Error handling
- No focused specs (`FDescribe`, `FIt`) should be committed
