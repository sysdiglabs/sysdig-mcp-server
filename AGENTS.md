# Sysdig MCP Server – Agent Developer Handbook

This document is a comprehensive guide for an AI agent tasked with developing and maintaining the Sysdig MCP Server.

## 1. Project Overview

**Sysdig MCP Server** is a Go-based Model Context Protocol (MCP) server that exposes Sysdig Secure platform capabilities to LLMs. It provides tools for querying runtime security events, Kubernetes metrics, and executing SysQL queries through multiple transport protocols (stdio, streamable-http, SSE).

### 1.1. Quick Facts

| Topic | Details |
| --- | --- |
| **Purpose** | Expose vetted Sysdig Secure workflows to LLMs through MCP tools. |
| **Tech Stack** | Go 1.25+, `mcp-go`, Cobra CLI, Ginkgo/Gomega, `golangci-lint`, Nix. |
| **Entry Point** | `cmd/server/main.go` (Cobra CLI that wires config, Sysdig client, etc.). |
| **Dev Shell** | `nix develop` provides a consistent development environment. |
| **Key Commands** | `just fmt`, `just lint`, `just test`, `just check`, `just update`. |

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
.github/workflows        - CI Workflows
cmd/server/              - CLI entry point, tool registration
internal/
  config/                - Environment variable loading and validation
  infra/
    clock/               - System clock abstraction (for testing)
    mcp/                 - MCP server handler, transport setup, middleware
      tools/             - Individual MCP tool implementations
    sysdig/              - Sysdig API client (generated + extensions)
docs/                    - Documentation assets
justfile                 - Canonical development tasks (format, lint, test, generate, update)
flake.nix                - Defines the Nix development environment and its dependencies
package.nix              - Defines how the package is going to be built with Nix
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
    - `client.gen.go`: Generated OpenAPI client (**DO NOT EDIT**, manually regenerated via oapi-codegen, not with `go generate`)
    - `client.go`: Authentication strategies with fallback support
    - Context-based auth: `WrapContextWithToken()` and `WrapContextWithHost()` for remote transports
    - Fixed auth: `WithFixedHostAndToken()` for stdio mode and remote transports
    - Custom extensions in `client_extension.go` and `client_*.go` files

5.  **Tools (`internal/infra/mcp/tools/`):**
    - Each tool has its own file: `tool_<name>.go` + `tool_<name>_test.go`
    - Tools implement `RegisterInServer(server *server.MCPServer)`
    - Use `WithRequiredPermissions()` from `utils.go` to declare Sysdig API permissions
    - Permission filtering happens automatically in handler

## 4. Day-to-Day Workflow

1.  **Enter the Dev Shell:** Always work inside the Nix shell (`nix develop` or `direnv allow`). You can assume the developer already did that.
2.  **Make Focused Changes:** Implement a new tool, fix a bug, or improve documentation.
3.  **Run Quality Gates:** Use `just` to run formatters, linters, and tests.
4.  **Commit:** Follow the Conventional Commits specification (see section 4.4).

### 4.1. Testing & Quality Gates

The project enforces quality through a series of checks.

```bash
just fmt        # Format Go code with gofumpt.
just lint       # Run the golangci-lint linter.
just test       # Run the unit test suite (auto-runs `go generate` first).
just check      # A convenient alias for fmt + lint + test.
```

### 4.2. Pre-commit Hooks

This repository uses **pre-commit** to automate quality checks before each commit.
The hooks are configured in `.pre-commit-config.yaml` to run `just fmt`, `just lint`, and `just test`.
If any of the hooks fail, the commit will not be created.

### 4.3 Updating all dependencies

Automated with `just update`. Requires `nix` installed.

### 4.4 Commit Conventions

Follow the [Conventional Commits](https://www.conventionalcommits.org/) specification with these guidelines:

- **Title only:** Commits should have only a title, no description body.
- **Large changes:** If the change is significant, add a description explaining the **why**, not what changed.
- **Format:** `<type>(<scope>): <subject>` (scope is optional).
- **Types:** `feat`, `fix`, `docs`, `refactor`, `test`, `chore`, `build`, `ci`.

Examples:
```
feat(tools): add new runtime events tool
fix: correct API endpoint URL
chore: update dependencies
```

## 5. Guides & Reference

*   **Tools & New Tool Creation:** See `internal/infra/mcp/tools/README.md`
*   **Releasing:** See `docs/RELEASING.md`
*   **Troubleshooting:** See `docs/TROUBLESHOOTING.md`
*   **Conventional Commits:** [Specification](https://www.conventionalcommits.org/)
*   **Protocol:** [Model Context Protocol](https://modelcontextprotocol.io/)
