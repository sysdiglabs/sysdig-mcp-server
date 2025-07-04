# MCP Server

| App Test | Helm Test |
|------|---------|
| [![App Test](https://github.com/sysdiglabs/sysdig-mcp-server/actions/workflows/publish.yaml/badge.svg?branch=main)](https://github.com/sysdiglabs/sysdig-mcp-server/actions/workflows/publish.yaml) | [![Helm Test](https://github.com/sysdiglabs/sysdig-mcp-server/actions/workflows/helm_test.yaml/badge.svg?branch=main)](https://github.com/sysdiglabs/sysdig-mcp-server/actions/workflows/helm_test.yaml) |

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
    - [`app_config.yaml`](#app_configyaml)
    - [Environment Variables](#environment-variables)
  - [Running the Server](#running-the-server)
    - [Docker](#docker)
    - [K8s Deployment](#k8s-deployment)
    - [UV](#uv)
  - [Client Configuration](#client-configuration)
    - [Authentication](#authentication)
    - [URL](#url)
    - [Claude Desktop App](#claude-desktop-app)
    - [MCP Inspector](#mcp-inspector)

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
                "SYSDIG_HOST",
                "-e",
                "MCP_TRANSPORT",
                "-e",
                "SYSDIG_SECURE_TOKEN",
                "ghcr.io/sysdiglabs/sysdig-mcp-server:latest"
            ],
            "env": {
              "SYSDIG_HOST": "<your_sysdig_host>",
              "SYSDIG_SECURE_TOKEN": "<your_sysdig_secure_api_token>",
              "MCP_TRANSPORT": "stdio"
            }
          }
        }
      }
      ```

## Available Tools

<details>
<summary><strong>Events Feed</strong></summary>

| Tool Name | Description | Sample Prompt |
|-----------|-------------|----------------|
| `get_event_info` | Retrieve detailed information for a specific security event by its ID | "Get full details for event ID 123abc" |
| `list_runtime_events` | List runtime security events with optional filters | "Show me high severity events from the last 2 hours in cluster1" |
| `get_event_process_tree` | Retrieve the process tree for a specific event (if available) | "Get the process tree for event ID abc123" |

</details>

<details>
<summary><strong>Inventory</strong></summary>

| Tool Name | Description | Sample Prompt |
|-----------|-------------|----------------|
| `list_resources` | List inventory resources using filters (e.g., platform or category) | "List all exposed IAM resources in AWS" |
| `get_resource` | Get detailed information about an inventory resource by its hash | "Get inventory details for hash abc123" |

</details>

<details>
<summary><strong>Vulnerability Management</strong></summary>

| Tool Name | Description | Sample Prompt |
|-----------|-------------|----------------|
| `list_runtime_vulnerabilities` | List runtime vulnerability scan results with filtering | "List running vulnerabilities in cluster1 sorted by severity" |
| `list_accepted_risks` | List all accepted vulnerability risks | "Show me all accepted risks related to nginx containers" |
| `get_accepted_risk` | Retrieve a specific accepted risk by ID | "Get details for accepted risk id abc123" |
| `list_registry_scan_results` | List scan results for container registries | "List failed scans from harbor registry" |
| `get_vulnerability_policy_by_id` | Get a specific vulnerability policy by ID | "Show policy ID 42" |
| `list_vulnerability_policies` | List all vulnerability policies | "List all vulnerability policies for pipeline stage" |
| `list_pipeline_scan_results` | List CI pipeline scan results | "Show me pipeline scans that failed for ubuntu images" |
| `get_scan_result` | Retrieve detailed scan results by scan ID | "Get results for scan ID 456def" |

</details>

<details>
<summary><strong>Sysdig Sage</strong></summary>

| Tool Name | Description | Sample Prompt |
|-----------|-------------|----------------|
| `sysdig_sysql_sage_query` | Generate and run a SysQL query using natural language | "List top 10 pods by memory usage in the last hour" |

</details>

### Available Resources

- Sysdig Secure Vulnerability Management Overview:
  - VM documentation based on the following [url](https://docs.sysdig.com/en/sysdig-secure/vulnerability-management/)
- Sysdig Filter Query Language Instructions:
  - Sysdig Filter Query Language for different API endpoint filters

## Requirements

### UV Setup

You can use [uv](https://github.com/astral-sh/uv) as a drop-in replacement for pip to create the virtual environment and install dependencies.

If you don't have `uv` installed, you can install it via (Linux and MacOS users):

```bash
curl -Ls https://astral.sh/uv/install.sh | sh
```

To set up the environment:

```bash
uv venv
source .venv/bin/activate
```

This will create a virtual environment using `uv` and install the required dependencies.

## Configuration

The application can be configured via the `app_config.yaml` file and environment variables.

### `app_config.yaml`

This file contains the main configuration for the application, including:

- **app**: Host, port, and log level for the MCP server.
- **sysdig**: The Sysdig Secure host to connect to.
- **mcp**: Transport protocol (stdio, sse, streamable-http), URL, host, and port for the MCP server.

### Environment Variables

The following environment variables are required for configuring the Sysdig SDK:

- `SYSDIG_HOST`: The URL of your Sysdig Secure instance (e.g., `https://us2.app.sysdig.com`).
- `SYSDIG_SECURE_TOKEN`: Your Sysdig Secure API token.

You can find your API token in the Sysdig Secure UI under **Settings > Sysdig Secure API**. Make sure to copy the token as it will not be shown again.

![API_TOKEN_CONFIG](./docs/assets/settings-config-token.png)
![API_TOKEN_SETTINGS](./docs/assets/api-token-copy.png)

You can set these variables in your shell or in a `.env` file.

You can also use `MCP_TRANSPORT` to override the transport protocol set in `app_config.yaml`.

> All of this env variables have precedence over the fields configured in the app_config.yaml.

## Running the Server

You can run the MCP server using either Docker, `uv` or install it in your K8s cluster with helm.

### Docker

To run the server using Docker, you first need to build the image:

```bash
docker build -t sysdig-mcp-server .
```

Then, you can run the container, making sure to pass the required environment variables:

```bash
docker run -e SYSDIG_HOST=<your_sysdig_host> -e SYSDIG_SECURE_TOKEN=<your_sysdig_secure_api_token> -p 8080:8080 sysdig-mcp-server
```

By default, the server will run using the `stdio` transport. To use the `streamable-http` or `sse` transports, set the `MCP_TRANSPORT` environment variable to `streamable-http` or `sse`:

```bash
docker run -e MCP_TRANSPORT=streamable-http -e SYSDIG_HOST=<your_sysdig_host> -e SYSDIG_SECURE_TOKEN=<your_sysdig_secure_api_token> -p 8080:8080 sysdig-mcp-server
```

### K8s Deployment

If you want to run the Sysdig MCP server in a K8s cluster you can use the helm chart provided in the `charts/sysdig-mcp` path

Modify the `values.yaml`

```yaml
# Example values.yaml
---
sysdig:
  secrets:
    create: true
    # If enabled, the secrets will be mounted as environment variables
    secureAPIToken: "<your_sysdig_secure_api_token>"
  mcp:
    transport: "streamable-http"
  # You can set the Sysdig Tenant URL at this level or below in the app_config configmap
  host: "https://us2.app.sysdig.com" # <your_sysdig_host> "https://eu1.app.sysdig.com"

configMap:
  enabled: true
  app_config: |
    # Sysdig MCP Server Configuration
    # This file is used to configure the Sysdig MCP server.
    # You can add your custom configuration here.
    app:
      host: "0.0.0.0"
      port: 8080
      log_level: "error"

    sysdig:
      host: "https://us2.app.sysdig.com" # <your_sysdig_host> "https://eu1.app.sysdig.com"

    mcp:
      transport: streamable-http
      host: "0.0.0.0"
      port: 8080
```

Install the chart

```bash,copy
helm upgrade --install sysdig-mcp ./charts/sysdig-mcp/ -n sysdig-mcp -f charts/sysdig-mcp/values.yaml
```

### UV

To run the server using `uv`, first set up the environment as described in the [UV Setup](#uv-setup) section. Then, run the `main.py` script:

```bash
uv run main.py
```

By default, the server will run using the `stdio` transport. To use the `streamable-http` or `sse` transports, set the `MCP_TRANSPORT` environment variable to `streamable-http` or `sse`:

```bash
MCP_TRANSPORT=streamable-http uv run main.py
```

## Client Configuration

To use the MCP server with a client like Claude or Cursor, you need to provide the server's URL and authentication details.

### Authentication

When using the `sse` or `streamable-http` transport, the server requires a Bearer token for authentication. The token is passed in the `Authorization` header of the HTTP request.

Additionally, you can specify the Sysdig Secure host by providing the `X-Sysdig-Host` header. If this header is not present, the server will use the value from `app_config.yaml`.

Example headers:

```
Authorization: Bearer <your_sysdig_secure_api_token>
X-Sysdig-Host: <your_sysdig_host>
```

### URL

If you are running the server with the `sse` or `streamable-http` transport, the URL will be `http://<host>:<port>/sysdig-mcp-server/mcp`, where `<host>` and `<port>` are the values configured in `app_config.yaml` or the Docker run command.

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
            "SYSDIG_HOST": "<your_sysdig_host>",
            "SYSDIG_SECURE_TOKEN": "<your_sysdig_secure_api_token>",
            "MCP_TRANSPORT": "stdio"
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
              "SYSDIG_HOST",
              "-e",
              "MCP_TRANSPORT",
              "-e",
              "SYSDIG_SECURE_TOKEN",
              "ghcr.io/sysdiglabs/sysdig-mcp-server"
          ],
          "env": {
            "SYSDIG_HOST": "<your_sysdig_host>",
            "SYSDIG_SECURE_TOKEN": "<your_sysdig_secure_api_token>",
            "MCP_TRANSPORT": "stdio"
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
