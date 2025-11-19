# MCP Server

| App Test |
|------|
| [![App Test](https://github.com/sysdiglabs/sysdig-mcp-server/actions/workflows/publish.yaml/badge.svg?branch=main)](https://github.com/sysdiglabs/sysdig-mcp-server/actions/workflows/publish.yaml) |

---

## Table of contents

- [MCP Server](#mcp-server)
  - [Table of contents](#table-of-contents)
  - [Description](#description)
  - [Quickstart Guide](#quickstart-guide)
  - [Available Tools](#available-tools)
    - [Available Resources](#available-resources)
  - [Requirements](#requirements)
    - [UV Setup](#uv-setup)
  - [Configuration](#configuration)
    - [API Permissions](#api-permissions)
  - [Running the Server](#running-the-server)
    - [Docker](#docker)
    - [UV](#uv)
  - [Client Configuration](#client-configuration)
    - [Authentication](#authentication)
    - [URL](#url)
    - [Claude Desktop App](#claude-desktop-app)
    - [MCP Inspector](#mcp-inspector)
    - [Goose Agent](#goose-agent)

## Description

This is an implementation of an [MCP (Model Context Protocol) Server](https://modelcontextprotocol.io/quickstart/server) to allow different LLMs to query information from Sysdig Secure platform. **It is still in early development and not yet ready for production use.** New endpoints and functionalities will be added over time. The goal is to provide a simple and easy-to-use interface for querying information from Sysdig Secure platform using LLMs.

## Quickstart Guide

Get up and running with the Sysdig MCP Server quickly using our pre-built Docker image.

1. **Get your API Token**:
    Go to your Sysdig Secure instance and navigate to **Settings > Sysdig Secure API**. Here, you can generate or copy your API token. This token is required to authenticate requests to the Sysdig Secure platform (See the [Configuration](#configuration) section for more details).

2. **Pull the image**:

    Pull the latest Sysdig MCP Server image from the GitHub Container Registry:

    ```bash
    docker pull ghcr.io/sysdiglabs/sysdig-mcp-server:latest
    ```

3. **Configure your client**:

    For example, you can configure Claude Desktop app to use the Sysdig MCP Server by editing the `claude_desktop_config.json` file. This is useful for running the server locally with the `stdio` transport. You can apply this configuration to any other client that supports MCP (For more details, see the [Client Configuration](#client-configuration) section).

    Substitute the following placeholders with your actual values:
    - `<your_sysdig_host>`: The hostname of your Sysdig Secure instance (e.g., `https://us2.app.sysdig.com` or `https://eu1.app.sysdig.com`)
    - `<your_sysdig_secure_api_token>`: Your Sysdig Secure API token

    ```json
      {
        "mcpServers": {
          "sysdig-mcp-server": {
            "command": "docker",
            "args": [
              "run",
                "-i",
                "--rm",
                "-e",
                "SYSDIG_MCP_API_HOST",
                "-e",
                "SYSDIG_MCP_TRANSPORT",
                "-e",
                "SYSDIG_MCP_API_SECURE_TOKEN",
                "ghcr.io/sysdiglabs/sysdig-mcp-server:latest"
            ],
            "env": {
              "SYSDIG_MCP_API_HOST": "<your_sysdig_host>",
              "SYSDIG_MCP_API_SECURE_TOKEN": "<your_sysdig_secure_api_token>",
              "SYSDIG_MCP_TRANSPORT": "stdio"
            }
          }
        }
      }
      ```

## Available Tools

<details>
<summary><strong>Events Feed</strong></summary>

| Tool Name | Description | Required Permission | Sample Prompt |
|-----------|-------------|---------------------|----------------|
| `get_event_info` | Retrieve detailed information for a specific security event by its ID | `policy-events.read` | "Get full details for event ID 123abc" |
| `list_runtime_events` | List runtime security events with optional filters | `policy-events.read` | "Show me high severity events from the last 2 hours in cluster1" |
| `get_event_process_tree` | Retrieve the process tree for a specific event (if available) | `policy-events.read` | "Get the process tree for event ID abc123" |

</details>

<details>
<summary><strong>Sysdig SysQL</strong></summary>

| Tool Name | Description | Required Permission | Sample Prompt |
|-----------|-------------|---------------------|----------------|
| `generate_sysql` | Generates a SysQL query from a natural language question. | `sage.exec` | "List top 10 pods by memory usage in the last hour" |
| `run_sysql` | Execute a pre-written SysQL query directly (use only when user provides explicit query) | `sage.exec`, `risks.read` | "Run this query: MATCH CloudResource WHERE type = 'aws_s3_bucket' LIMIT 10" |

</details>

<details>
<summary><strong>Sysdig CLI scanner</strong></summary>

> **Note:** This tool is **only available when using `stdio` transport**. It is not available for `streamable-http` or `sse` transports.

| Tool Name | Description | Sample Prompt |
|-----------|-------------|----------------|
| `run_sysdig_cli_scanner` | Run the Sysdig CLI Scanner to analyze a container image or IaC files for vulnerabilities and posture and misconfigurations. | "Scan this image ubuntu:latest for vulnerabilities" |

</details>

### Available Resources

- Sysdig Filter Query Language Instructions:
  - Sysdig Filter Query Language for different API endpoint filters

## Requirements

### UV Setup

You can use [uv](https://github.com/astral-sh/uv) as a drop-in replacement for pip to create the virtual environment and install dependencies.

If you don't have `uv` installed, you can install it following the instructions that you can find on the `README` of the project.

If you want to develop, set up the environment using:

```bash
uv venv
source .venv/bin/activate
```

This will create a virtual environment using `uv` and install the required dependencies.

## Configuration

The following environment variables are **required** for configuring the Sysdig SDK:

- `SYSDIG_MCP_API_HOST`: The URL of your Sysdig Secure instance (e.g., `https://us2.app.sysdig.com`).
- `SYSDIG_MCP_API_SECURE_TOKEN`: Your Sysdig Secure API token.

You can also set the following variables to override the default configuration:

- `SYSDIG_MCP_TRANSPORT`: The transport protocol for the MCP Server (`stdio`, `streamable-http`, `sse`). Defaults to: `stdio`.
- `SYSDIG_MCP_MOUNT_PATH`:  The URL prefix for the Streamable-http/sse deployment. Defaults to: `/sysdig-mcp-server`
- `SYSDIG_MCP_LOGLEVEL`: Log Level of the application (`DEBUG`, `INFO`, `WARNING`, `ERROR`). Defaults to: `INFO`
- `SYSDIG_MCP_LISTENING_PORT`: The port for the server when it is deployed using remote protocols (`steamable-http`, `sse`). Defaults to: `8080`
- `SYSDIG_MCP_LISTENING_HOST`: The host for the server when it is deployed using remote protocols (`steamable-http`, `sse`). Defaults to: `localhost`

You can find your API token in the Sysdig Secure UI under **Settings > Sysdig Secure API**. Make sure to copy the token as it will not be shown again.

![API_TOKEN_CONFIG](./docs/assets/settings-config-token.png)
![API_TOKEN_SETTINGS](./docs/assets/api-token-copy.png)

You can set these variables in your shell or in a `.env` file.

**Example `.env` file:**

```bash
# Required Configuration
SYSDIG_MCP_API_HOST=https://us2.app.sysdig.com
SYSDIG_MCP_API_SECURE_TOKEN=your-api-token-here

# Optional Configuration (with defaults)
SYSDIG_MCP_TRANSPORT=stdio
SYSDIG_MCP_LOGLEVEL=INFO
SYSDIG_MCP_LISTENING_PORT=8080
SYSDIG_MCP_LISTENING_HOST=localhost
SYSDIG_MCP_MOUNT_PATH=/sysdig-mcp-server
```

### API Permissions

To use the MCP server tools, your API token needs specific permissions in Sysdig Secure. We recommend creating a dedicated Service Account (SA) with a custom role containing only the required permissions.

**Minimum Required Permissions by Tool:**

| Tool Category | Required Permissions | Sysdig UI Permission Names |
|--------------|---------------------|---------------------------|
| **CLI Scanner** | `secure.vm.cli-scanner.exec` | Vulnerability Management: "CLI Execution" (EXEC) |
| **Threat Detection (Events Feed)** | `policy-events.read` | Threats: "Policy Events" (Read) |
| **SysQL** | `sage.exec`, `risks.read` | SysQL: "AI Query Generation" (EXEC) + Risks: "Access to risk feature" (Read) |

**Additional Permissions:**

- Settings: "API Access Token" - View, Read, Edit (required to generate and manage API tokens)

**Setting up Permissions:**

1. Go to **Settings > Users & Teams > Roles** in your Sysdig Secure instance
2. Create a new role with the permissions listed above
3. Assign this role to a Service Account or user
4. Use the API token from that account with the MCP server

> **Note:** When selecting permissions, some dependent permissions may be automatically added by Sysdig.

For detailed instructions, see the official [Sysdig Roles Administration documentation](https://docs.sysdig.com/en/administration/roles-administration/).

>[!IMPORTANT]
> **Service Account Limitation:** The `generate_sysql` tool currently does not work with Service Account tokens and will return a 500 error. For this tool, use an API token assigned to a regular user account.


## Running the Server

You can run the MCP server using Docker (recommended for production), `uv` (for development), or install it in your K8s cluster with helm.

### Docker (Recommended)

The easiest way to run the server is using the pre-built Docker image from GitHub Container Registry (as shown in the [Quickstart Guide](#quickstart-guide)).

If you need to build the image locally, you can do so with:

```bash
docker build -t sysdig-mcp-server .
```

Then, run the container with the required environment variables:

```bash
docker run -e SYSDIG_MCP_API_HOST=<your_sysdig_host> -e SYSDIG_MCP_API_SECURE_TOKEN=<your_sysdig_secure_api_token> -e SYSDIG_MCP_TRANSPORT=stdio -p 8080:8080 sysdig-mcp-server
```

To use the `streamable-http` or `sse` transports (for remote MCP clients), set the `SYSDIG_MCP_TRANSPORT` environment variable accordingly:

```bash
docker run -e SYSDIG_MCP_TRANSPORT=streamable-http -e SYSDIG_MCP_API_HOST=<your_sysdig_host> -e SYSDIG_MCP_API_SECURE_TOKEN=<your_sysdig_secure_api_token> -p 8080:8080 sysdig-mcp-server
```

### UV (Development)

For local development, you can run the server using `uv`. First set up the environment as described in the [UV Setup](#uv-setup) section, then run:

```bash
uv run main.py
```

By default, the server will run using the `stdio` transport. To use the `streamable-http` or `sse` transports, set the `SYSDIG_MCP_TRANSPORT` environment variable:

```bash
SYSDIG_MCP_TRANSPORT=streamable-http uv run main.py
```

## Client Configuration

To use the MCP server with a client like Claude or Cursor, you need to provide the server's URL and authentication details.

### Authentication

When using the `sse` or `streamable-http` transport, the server requires a Bearer token for authentication. The token is passed in the `X-Sysdig-Token` or default to `Authorization` header of the HTTP request (i.e `Bearer SYSDIG_SECURE_API_TOKEN`).

Additionally, you can specify the Sysdig Secure host by providing the `X-Sysdig-Host` header. If this header is not present, the server will use the value from the env variable `SYSDIG_MCP_API_HOST`.

Example headers:

```
Authorization: Bearer <your_sysdig_secure_api_token>
X-Sysdig-Host: <your_sysdig_host>
```

### URL

If you are running the server with the `sse` or `streamable-http` transport, the URL will be `http://<host>:<port>/sysdig-mcp-server/mcp`.

For example, if you are running the server locally on port 8080, the URL will be `http://localhost:8080/sysdig-mcp-server/mcp`.

### Claude Desktop App

For the Claude Desktop app, you can manually configure the MCP server by editing the `claude_desktop_config.json` file. This is useful for running the server locally with the `stdio` transport.

1. **Open the configuration file**:
    - Go to **Settings > Developer** in the Claude Desktop app.
    - Click on **Edit Config** to open the `claude_desktop_config.json` file.

2. **Add the MCP server configuration**:
    - Add the following JSON object to the `mcpServers` section of the file.

    ```json
    {
      "mcpServers": {
        "sysdig-mcp-server": {
          "command": "uv",
          "args": [
            "--directory",
            "<path_to_your_sysdig_mcp_server_directory>",
            "run",
            "main.py"
            ],
          "env": {
            "SYSDIG_MCP_API_HOST": "<your_sysdig_host>",
            "SYSDIG_MCP_API_SECURE_TOKEN": "<your_sysdig_secure_api_token>",
            "SYSDIG_MCP_TRANSPORT": "stdio"
          }
        }
      }
    }
    ```

    Or, alternatively, if you want to use docker, you can add the following configuration:

    ```json
    {
      "mcpServers": {
        "sysdig-mcp-server": {
          "command": "docker",
          "args": [
             "run",
              "-i",
              "--rm",
              "-e",
              "SYSDIG_MCP_API_HOST",
              "-e",
              "SYSDIG_MCP_TRANSPORT",
              "-e",
              "SYSDIG_MCP_API_SECURE_TOKEN",
              "ghcr.io/sysdiglabs/sysdig-mcp-server"
          ],
          "env": {
            "SYSDIG_MCP_API_HOST": "<your_sysdig_host>",
            "SYSDIG_MCP_API_SECURE_TOKEN": "<your_sysdig_secure_api_token>",
            "SYSDIG_MCP_TRANSPORT": "stdio"
          }
        }
      }
    }
    ```

3. **Replace the placeholders**:
    - Replace `<your_sysdig_host>` with your Sysdig Secure host URL.
    - Replace `<your_sysdig_secure_api_token>` with your Sysdig Secure API token.
    - Replace `<path_to_your_sysdig_mcp_server_directory>` with the absolute path to the `sysdig-mcp-server` directory.

4. **Save the file** and restart the Claude Desktop app for the changes to take effect.

### MCP Inspector

1. Run the [MCP Inspector](https://modelcontextprotocol.io/docs/tools/inspector) locally
2. Select the transport type and have the Sysdig MCP server running accordingly.
3. Pass the Authorization header if using "streamable-http" or the SYSDIG_SECURE_API_TOKEN env var if using "stdio"

![mcp-inspector](./docs/assets/mcp-inspector.png)


### Goose Agent

1. In your terminal run `goose configure` and follow the steps to add the extension (more info on the [goose docs](https://block.github.io/goose/docs/getting-started/using-extensions/)), again could be using docker or uv as shown in the above examples.
2. Your `~/.config/goose/config.yaml` config file should have one config like this one, check out the env vars

  ```yaml
  extensions:
  ...
    sysdig-mcp-server:
      args: []
      bundled: null
      cmd: sysdig-mcp-server
      description: Sysdig MCP server
      enabled: true
      env_keys:
      - SYSDIG_MCP_TRANSPORT
      - SYSDIG_MCP_API_HOST
      - SYSDIG_MCP_API_SECURE_TOKEN
      envs:
        SYSDIG_MCP_TRANSPORT: stdio
      name: sysdig-mcp-server
      timeout: 300
      type: stdio
  ```
3. Have fun

![goose_results](./docs/assets/goose_results.png)


