# Sysdig MCP Server – Agent Developer Handbook

This document is a comprehensive guide for an AI agent tasked with developing and maintaining the Sysdig MCP Server. It covers everything from project setup and architecture to daily workflows and troubleshooting.

## 1. Project Overview

**Sysdig MCP Server** is a Go-based Model Context Protocol (MCP) server that exposes Sysdig Secure platform capabilities to LLMs. It provides tools for querying runtime security events, Kubernetes metrics, and executing SysQL queries through multiple transport protocols (stdio, streamable-http, SSE).

### 1.1. Quick Facts

| Topic | Details |
| --- | --- |
| **Purpose** | Expose vetted Sysdig Secure workflows to LLMs through MCP tools. |
| **Tech Stack** | Go 1.25+, `mcp-go`, Cobra CLI, Ginkgo/Gomega, `golangci-lint`, Nix. |
| **Entry Point** | `cmd/server/main.go` (Cobra CLI that wires config, Sysdig client, etc.). |
| **Dev Shell** | `nix develop` provides a consistent development environment. |
| **Key Commands** | `just fmt`, `just lint`, `just test`, `just check`, `just bump`. |

## 2. Environment Setup

### 2.1. Using Nix (Recommended)

The repository uses a Nix flake to ensure a consistent development environment with all required tools.

```bash
# Enter the development shell with all tools available
nix develop

# Or, if you use direnv, allow it to load the environment
direnv allow
```

### 2.2. Required Environment Variables

The server requires API credentials to connect to Sysdig Secure.

- `SYSDIG_MCP_API_HOST`: Sysdig Secure instance URL (e.g., `https://us2.app.sysdig.com`).
- `SYSDIG_MCP_API_TOKEN`: Sysdig Secure API token.

For a full list of optional variables (e.g., for transport configuration), see the project's `README.md`.

## 3. Architecture

### 3.1. Repository Layout

```
cmd/server/              - CLI entry point, tool registration
internal/
  config/                - Environment variable loading and validation
  infra/
    clock/               - System clock abstraction (for testing)
    mcp/                 - MCP server handler, transport setup, middleware
      tools/             - Individual MCP tool implementations
    sysdig/              - Sysdig API client (generated + extensions)
docs/                    - Documentation assets
justfile                 - Canonical development tasks (format, lint, test, generate, bump)
flake.nix                - Defines the Nix development environment and its dependencies
```

### 3.2. Key Components & Flow

1.  **Entry Point (`cmd/server/main.go`):**
    - Cobra CLI that loads config, sets up Sysdig client, registers tools, and starts transport
    - `setupHandler()` registers all MCP tools (line 88-114)
    - `startServer()` handles stdio/streamable-http/sse transport switching (line 118-140)

2.  **Configuration (`internal/config/config.go`):**
    - Loads environment variables with `SYSDIG_MCP_*` prefix
    - Validates required fields for stdio transport (API host and token mandatory)
    - Supports remote transports where auth can come via HTTP headers

3.  **MCP Handler (`internal/infra/mcp/mcp_handler.go`):**
    - Wraps mcp-go server with permission filtering (`toolPermissionFiltering`, line 26-64)
    - Dynamically filters tools based on Sysdig API token permissions
    - HTTP middleware extracts `Authorization` and `X-Sysdig-Host` headers for remote transports (line 108-138)

4.  **Sysdig Client (`internal/infra/sysdig/`):**
    - `client.gen.go`: Generated OpenAPI client (**DO NOT EDIT**, regenerated via oapi-codegen)
    - `client.go`: Authentication strategies with fallback support
    - Context-based auth: `WrapContextWithToken()` and `WrapContextWithHost()` for remote transports
    - Fixed auth: `WithFixedHostAndToken()` for stdio mode
    - Custom extensions in `client_extension.go` and `client_*.go` files

5.  **Tools (`internal/infra/mcp/tools/`):**
    - Each tool has its own file: `tool_<name>.go` + `tool_<name>_test.go`
    - Tools implement `RegisterInServer(server *server.MCPServer)`
    - Use `WithRequiredPermissions()` from `utils.go` to declare Sysdig API permissions
    - Permission filtering happens automatically in handler

### 3.3. Authentication Flow

1. **stdio transport**: Fixed host/token from env vars (`SYSDIG_MCP_API_HOST`, `SYSDIG_MCP_API_TOKEN`)
2. **Remote transports**: Extract from HTTP headers (`Authorization: Bearer <token>`, `X-Sysdig-Host`)
3. Fallback chain: Try context auth first, then fall back to env var auth
4. Each request includes Bearer token in Authorization header to Sysdig APIs

### 3.4. Tool Permission System

- Each tool declares its required Sysdig API permissions using `WithRequiredPermissions("permission1", "permission2")`.
- Before exposing tools to the LLM, the handler calls the Sysdig `GetMyPermissions` API.
- The agent will only see tools for which the provided API token has **all** required permissions.
- Common permissions: `policy-events.read`, `sage.exec`, `risks.read`, `promql.exec`

## 4. Day-to-Day Workflow

1.  **Enter the Dev Shell:** Always work inside the Nix shell (`nix develop` or `direnv allow`) to ensure all tools are available. You can assume the developer is already in a Nix shell.
2.  **Make Focused Changes:** Implement a new tool, fix a bug, or improve documentation.
3.  **Run Quality Gates:** Use `just` to run formatters, linters, and tests.
4.  **Commit:** Follow the Conventional Commits specification. Keep the commit messages short, just title, no description. Pre-commit hooks will run quality gates automatically.

### 4.1. Testing & Quality Gates

The project enforces quality through a series of checks.

```bash
just fmt        # Format Go code with gofumpt.
just lint       # Run the golangci-lint linter.
just test       # Run the unit test suite (auto-runs `go generate` first).
just check      # A convenient alias for fmt + lint + test.
```

### 4.2. Pre-commit Hooks

This repository uses **pre-commit** to automate quality checks before each commit. The hooks are configured in `.pre-commit-config.yaml` to run `just fmt`, `just lint`, and `just test`.

This means that every time you run `git commit`, your changes are automatically formatted, linted, and tested. If any of these checks fail, the commit is aborted, allowing you to fix the issues.

If the hooks do not run automatically, you may need to install them first:
```bash
# Install the git hooks defined in the configuration
pre-commit install

# After installation, you can run all checks on all files
pre-commit run -a
```

## 5. MCP Tools & Permissions

The handler filters tools dynamically based on the Sysdig user's permissions. Each tool declares mandatory permissions via `WithRequiredPermissions`.

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
| `troubleshoot_kubernetes_list_count_pods_per_cluster` | `tool_troubleshoot_kubernetes_list_count_pods_per_cluster.go` | List the count of running Kubernetes Pods grouped by cluster and namespace. | `promql.exec` | "List the count of running Kubernetes Pods in cluster 'production'" |
| `troubleshoot_kubernetes_list_underutilized_pods_by_cpu_quota` | `tool_troubleshoot_kubernetes_list_underutilized_pods_by_cpu_quota.go` | List Kubernetes pods with CPU usage below 25% of the quota limit. | `promql.exec` | "Show the top 10 underutilized pods by CPU quota in cluster 'production'" |
| `troubleshoot_kubernetes_list_underutilized_pods_by_memory_quota` | `tool_troubleshoot_kubernetes_list_underutilized_pods_by_memory_quota.go` | List Kubernetes pods with memory usage below 25% of the limit. | `promql.exec` | "Show the top 10 underutilized pods by memory quota in cluster 'production'" |
| `troubleshoot_kubernetes_list_top_cpu_consumed_by_workload` | `tool_troubleshoot_kubernetes_list_top_cpu_consumed_by_workload.go` | Identifies the Kubernetes workloads (all containers) consuming the most CPU (in cores). | `promql.exec` | "Show the top 10 workloads consuming the most CPU in cluster 'production'" |
| `troubleshoot_kubernetes_list_top_cpu_consumed_by_container` | `tool_troubleshoot_kubernetes_list_top_cpu_consumed_by_container.go` | Identifies the Kubernetes containers consuming the most CPU (in cores). | `promql.exec` | "Show the top 10 containers consuming the most CPU in cluster 'production'" |
| `troubleshoot_kubernetes_list_top_memory_consumed_by_workload` | `tool_troubleshoot_kubernetes_list_top_memory_consumed_by_workload.go` | Lists memory-intensive workloads (all containers). | `promql.exec` | "Show the top 10 workloads consuming the most memory in cluster 'production'" |
| `troubleshoot_kubernetes_list_top_memory_consumed_by_container` | `tool_troubleshoot_kubernetes_list_top_memory_consumed_by_container.go` | Lists memory-intensive containers. | `promql.exec` | "Show the top 10 containers consuming the most memory in cluster 'production'" |

## 6. Adding a New Tool

1.  **Create Files:** Add `tool_<name>.go` and `tool_<name>_test.go` in `internal/infra/mcp/tools/`.

2.  **Implement the Tool:**
    *   Define a struct that holds the Sysdig client.
    *   Implement the `handle` method, which contains the tool's core logic.
    *   Implement the `RegisterInServer` method to define the tool's MCP schema, including its name, description, parameters, and required permissions. Use helpers from `utils.go`.

3.  **Write Tests:** Use Ginkgo/Gomega to write BDD-style tests. Mock the Sysdig client to cover:
    - Parameter validation
    - Permission metadata
    - Sysdig API client interactions (mocked)
    - Error handling

4.  **Register the Tool:** Add the new tool to `setupHandler()` in `cmd/server/main.go` (line 88-114).

5.  **Document:** Add the new tool to the README.md and the table in section 5 (MCP Tools & Permissions).

### 6.1. Example Tool Structure

```go
type ToolMyFeature struct {
    sysdigClient sysdig.ExtendedClientWithResponsesInterface
}

func (h *ToolMyFeature) handle(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
    param := request.GetString("param_name", "")
    response, err := h.sysdigClient.SomeAPICall(ctx, param)
    // Handle response...
    return mcp.NewToolResultJSON(response.JSON200)
}

func (h *ToolMyFeature) RegisterInServer(s *server.MCPServer) {
    tool := mcp.NewTool("my_feature",
        mcp.WithDescription("What this tool does"),
        mcp.WithString("param_name",
            mcp.Required(),
            mcp.Description("Parameter description"),
        ),
        mcp.WithReadOnlyHintAnnotation(true),
        mcp.WithDestructiveHintAnnotation(false),
        WithRequiredPermissions("permission.name"),
    )
    s.AddTool(tool, h.handle)
}
```

### 6.2. Testing Philosophy

- Use BDD-style tests with Ginkgo/Gomega
- Each tool requires comprehensive test coverage for:
  - Parameter validation
  - Permission metadata
  - Sysdig API client interactions (mocked using go-mock)
  - Error handling
- Integration tests marked with `_integration_test.go` suffix
- No focused specs (`FDescribe`, `FIt`) should be committed

## 7. Conventional Commits

All commit messages must follow the [Conventional Commits](https://www.conventionalcommits.org/) specification. This is essential for automated versioning and changelog generation.

- **Types**: `feat`, `fix`, `docs`, `style`, `refactor`, `test`, `chore`, `build`, `ci`.
- **Format**: `<type>(<optional scope>): <imperative description>`

## 8. Code Generation

- `internal/infra/sysdig/client.gen.go` is auto-generated from OpenAPI spec via oapi-codegen.
- Run `go generate ./...` (or `just generate`) to regenerate after spec changes.
- Generated code includes all Sysdig Secure API types and client methods.
- **DO NOT** manually edit `client.gen.go`. Extend functionality in separate files (e.g., `client_extension.go`).

## 9. Important Constraints

1. **Generated Code**: Never manually edit `client.gen.go`. Extend functionality in separate files like `client_extension.go`.

2. **Service Account Limitation**: The `generate_sysql` tool does NOT work with Service Account tokens (returns 500). Use regular user API tokens for this tool.

3. **Permission Filtering**: Tools are hidden if the API token lacks required permissions. Check user's Sysdig role if a tool is unexpectedly missing.

4. **stdio Mode Requirements**: When using stdio transport, `SYSDIG_MCP_API_HOST` and `SYSDIG_MCP_API_TOKEN` MUST be set. Remote transports can receive these via HTTP headers instead.

## 10. Troubleshooting

**Problem**: Tool not appearing in MCP client
- **Solution**: Check API token permissions match tool's `WithRequiredPermissions()`. Use Sysdig UI: **Settings > Users & Teams > Roles**. The token must have **all** permissions listed.

**Problem**: "unable to authenticate with any method"
- **Solution**: For `stdio`, verify `SYSDIG_MCP_API_HOST` and `SYSDIG_MCP_API_TOKEN` env vars are set correctly. For remote transports, check `Authorization: Bearer <token>` header format.

**Problem**: Tests failing with "command not found"
- **Solution**: Enter Nix shell with `nix develop` or `direnv allow`. All dev tools are provided by the flake.

**Problem**: `generate_sysql` returning 500 error
- **Solution**: This tool requires a regular user API token, not a Service Account token. Switch to a user-based token.

**Problem**: Pre-commit hooks not running
- **Solution**: Run `pre-commit install` to install git hooks, then `pre-commit run -a` to test all files.

## 11. Reference Links

- `README.md` – Comprehensive product docs, quickstart, and client configuration samples.
- `CLAUDE.md` – Complementary guide with additional examples and command reference.
- [Model Context Protocol](https://modelcontextprotocol.io/) – Protocol reference for tool/transport behavior.
