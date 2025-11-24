# Sysdig MCP Server – Agent Handbook

This document is optimized for MCP coding agents. It highlights what matters most when you need to explore the repository, extend a tool, or run validation quickly.

## Quick Facts

| Topic | Details |
| --- | --- |
| Purpose | Expose vetted Sysdig Secure workflows to LLMs through MCP tools |
| Entry point | `cmd/server/main.go` (Cobra CLI that wires config, Sysdig client, handler, transports) |
| Runtime | Go 1.25+, uses `mcp-go`, `cobra`, `ginkgo/gomega`, `golangci-lint` |
| Dev shell | `nix develop` (check `IN_NIX_SHELL=1` before hacking or running commands) |
| Key commands | `just fmt`, `just lint`, `just test`, `just check`, `just test-coverage` |

## Repository Layout

| Path | Ownership Notes |
| --- | --- |
| `cmd/server` | Cobra CLI + transport bootstrap; `setupHandler` registers every MCP tool. |
| `internal/config` | Loads environment variables (`SYSDIG_MCP_*`) and enforces validation (stdio requires host/token). |
| `internal/infra/mcp` | Generic MCP handler, HTTP/SSE middlewares, permission filtering logic. |
| `internal/infra/mcp/tools` | One file per tool + `_test.go`. Helpers live in `utils.go`. |
| `internal/infra/sysdig` | Typed Sysdig Secure client plus auth helpers (`WrapContextWithHost/Token`). |
| `docs/` | Assets referenced from the README (diagrams, screenshots). |
| `justfile` | Canonical dev tasks (format, lint, generate, test, dependency bump). |

## Day-to-Day Workflow

1. Assume you are in a Nix shell and you have all the available tools. Otherwise edit `flake.nix` to add any required tool you don't have in the PATH.
2. Make focused changes (new MCP tool, bugfix, docs, etc.).
3. Run the default quality gates:
   ```bash
   just fmt        # gofumpt -w .
   just lint       # golangci-lint run
   just test       # ginkgo -r -p (auto-runs `go generate ./...`)
   ```
4. Use `just check` to chain fmt+lint+test, and `just test-coverage` when you need coverage artifacts.
5. Follow Conventional Commits when preparing PRs.
6. In case you need to update or add more dependencies run `just bump`.

## MCP Tools & Permissions

The handler filters tools dynamically based on `GetMyPermissions` from Sysdig Secure. Each tool declares mandatory permissions via `WithRequiredPermissions`. Current tools (`internal/infra/mcp/tools`):

| Tool | File | Capability | Required Permissions | Useful Prompts |
| --- | --- | --- | --- | --- |
| `list_runtime_events` | `tool_list_runtime_events.go` | Query runtime events with filters, cursor, scope. | `policy-events.read` | “Show high severity runtime events from last 2h.” |
| `get_event_info` | `tool_get_event_info.go` | Pull full payload for a single policy event. | `policy-events.read` | “Fetch event `abc123` details.” |
| `get_event_process_tree` | `tool_get_event_process_tree.go` | Retrieve the process tree for an event when available. | `policy-events.read` | “Show the process tree behind event `abc123`.” |
| `run_sysql` | `tool_run_sysql.go` | Execute caller-supplied Sysdig SysQL queries safely. | `sage.exec`, `risks.read` | “Run the following SysQL…”. |
| `generate_sysql` | `tool_generate_sysql.go` | Convert natural language to SysQL via Sysdig Sage. | `sage.exec` (does not work with Service Accounts) | “Create a SysQL to list S3 buckets.” |
| `kubernetes_list_clusters` | `tool_kubernetes_list_clusters.go` | Lists Kubernetes cluster information. | `promql.exec` | "List all Kubernetes clusters" |
| `kubernetes_list_nodes` | `tool_kubernetes_list_nodes.go` | Lists Kubernetes node information. | `promql.exec` | "List all Kubernetes nodes in the cluster 'production-gke'" |
| `kubernetes_list_workloads` | `tool_kubernetes_list_workloads.go` | Lists Kubernetes workload information. | `promql.exec` | "List all desired workloads in the cluster 'production-gke' and namespace 'default'" |
| `kubernetes_list_pod_containers` | `tool_kubernetes_list_pod_containers.go` | Retrieves information from a particular pod and container. | `promql.exec` | "Show me info for pod 'my-pod' in cluster 'production-gke'" |
| `kubernetes_list_cronjobs` | `tool_kubernetes_list_cronjobs.go` | Retrieves information from the cronjobs in the cluster. | `promql.exec` | "List all cronjobs in cluster 'prod' and namespace 'default'" |
| `troubleshoot_kubernetes_list_top_unavailable_pods` | `tool_troubleshoot_kubernetes_list_top_unavailable_pods.go` | Shows the top N pods with the highest number of unavailable or unready replicas. | `promql.exec` | "Show the top 20 unavailable pods in cluster 'production'" |
| `troubleshoot_kubernetes_list_top_restarted_pods` | `tool_troubleshoot_kubernetes_list_top_restarted_pods.go` | Lists the pods with the highest number of container restarts. | `promql.exec` | "Show the top 10 pods with the most container restarts in cluster 'production'" |
| `troubleshoot_kubernetes_list_top_400_500_http_errors_in_pods` | `tool_troubleshoot_kubernetes_list_top_400_500_http_errors_in_pods.go` | Lists the pods with the highest rate of HTTP 4xx and 5xx errors over a specified time interval. | `promql.exec` | "Show the top 20 pods with the most HTTP errors in cluster 'production'" |
| `troubleshoot_kubernetes_list_top_network_errors_in_pods` | `tool_troubleshoot_kubernetes_list_top_network_errors_in_pods.go` | Shows the top network errors by pod over a given interval. | `promql.exec` | "Show the top 10 pods with the most network errors in cluster 'production'" |

Every tool has a companion `_test.go` file that exercises request validation, permission metadata, and Sysdig client calls through mocks.
Note that if you add more tools you need to also update this file to reflect that.

## Adding or Updating Tools

1. Create a new file in `internal/infra/mcp/tools/tool_<name>.go` plus `_test.go`.
2. Implement a struct with a `handle` method and `RegisterInServer`; reuse helpers from `utils.go` (`Examples`, `WithRequiredPermissions`, `toPtr`, etc.).
3. Cover all branches with Ginkgo/Gomega tests. Use the `tools_suite_test.go` suite for shared setup.
4. Register the tool in `cmd/server/main.go` inside `setupHandler`.
5. Document required permissions and sample prompts in both the README and MCP metadata.

## Testing & Quality Gates

- `just test` runs `go generate ./...` first, then executes the whole suite via Ginkgo (`-r -p` to parallelize). Avoid leaving focused specs (`FDescribe`, `FIt`) in committed code.
- `just lint` runs `golangci-lint run` using the repo’s configuration (see `.golangci.yml` if adjustments are necessary).
- `just test-coverage` emits `coverage.out`; open it with `go tool cover -func=coverage.out`.
- For manual checks, `go test ./...` and `ginkgo ./path/to/package` work inside the Nix shell.

## Troubleshooting & Tips

- **Missing config:** `SYSDIG_MCP_API_HOST` and `SYSDIG_MCP_API_TOKEN` are mandatory in `stdio`. Validation fails early in `internal/config/config.go`.
- **Token scope:** If a tool does not appear, verify the token’s permissions under **Settings > Users & Teams > Roles**. `generate_sysql` currently requires a regular user token, not a Service Account.
- **Remote auth:** When using `streamable-http` or `sse`, pass `Authorization: Bearer <token>` and optionally `X-Sysdig-Host`. These values override env vars via the request context middleware.
- **Environment drift:** Always run inside `nix develop`; lint/test expect binaries like `gofumpt`, `golangci-lint`, and `ginkgo` provided by the flake.
- **Dependency refresh:** Use `just bump` (updates flake inputs, runs `go get -u`, `go mod tidy`, and rebuilds `package.nix`) when you truly need to refresh dependencies.

## Reference Links

- `README.md` – comprehensive product docs, quickstart, and client configuration samples.
- `pkg.go.dev/github.com/sysdiglabs/sysdig-mcp-server` – use when checking published module versions.
- [Model Context Protocol](https://modelcontextprotocol.io/) – protocol reference for tool/transport behavior.
