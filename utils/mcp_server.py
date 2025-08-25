"""
This module provides the FastMCP server for Sysdig Secure tools.
It includes endpoints for Sysdig Secure Events Feed, Inventory, Vulnerability Management, and Sysdig Sage tools.
"""

import logging
import os
import asyncio
from typing import Optional
import uvicorn
from starlette.requests import Request
from starlette.responses import JSONResponse, Response
from typing_extensions import Literal
from fastapi import FastAPI
from fastmcp import FastMCP
from fastmcp.resources import HttpResource, TextResource
from utils.middleware.auth import create_auth_middleware
from starlette.middleware import Middleware
from tools.events_feed.tool import EventsFeedTools
from tools.inventory.tool import InventoryTools
from tools.vulnerability_management.tool import VulnerabilityManagementTools
from tools.sysdig_sage.tool import SageTools
from tools.cli_scanner.tool import CLIScannerTool

# Application config loader
from utils.app_config import get_app_config, AppConfig

# Load app config (expects keys: mcp.host, mcp.port, mcp.transport)
app_config = get_app_config()

# Set up logging
logging.basicConfig(
    format="%(asctime)s-%(process)d-%(levelname)s- %(message)s",
    level=app_config.log_level(),
)
log = logging.getLogger(__name__)

_mcp_instance: Optional[FastMCP] = None

middlewares = [Middleware(create_auth_middleware(app_config))]


def create_simple_mcp_server() -> FastMCP:
    """
    Instantiate and configure the FastMCP server.

    Returns:
        FastMCP: An instance of the FastMCP server configured with Sysdig Secure tools and resources.
    """
    return FastMCP(
        name="Sysdig MCP Server",
        instructions="Provides Sysdig Secure tools and resources.",
        host=app_config.host(),
        port=app_config.port(),
        tags=["sysdig", "mcp", app_config.transport()],
    )


def get_mcp() -> FastMCP:
    """
    Return a singleton FastMCP instance.

    Returns:
        FastMCP: The singleton instance of the FastMCP server.
    """
    global _mcp_instance
    if _mcp_instance is None:
        _mcp_instance = create_simple_mcp_server()
    return _mcp_instance


def run_stdio():
    """
    Run the MCP server using STDIO transport.
    """
    mcp = get_mcp()
    # Add tools to the MCP server
    add_tools(mcp=mcp)
    # Add resources to the MCP server
    add_resources(mcp)
    try:
        asyncio.run(mcp.run_stdio_async())
    except KeyboardInterrupt:
        log.info("Keyboard interrupt received, forcing immediate exit")
        os._exit(0)
    except Exception as e:
        log.error(f"Exception received, forcing immediate exit: {str(e)}")
        os._exit(1)


def run_http():
    """Run the MCP server over HTTP/SSE transport via Uvicorn."""
    mcp = get_mcp()
    # Add tools to the MCP server
    transport = app_config.transport()

    add_tools(mcp=mcp)
    # Add resources to the MCP server
    add_resources(mcp)

    # Mount the MCP HTTP/SSE app
    mcp_app = mcp.http_app(transport=transport, middleware=middlewares)
    suffix_path = mcp.settings.streamable_http_path if transport == "streamable-http" else mcp.settings.sse_path
    app = FastAPI(lifespan=mcp_app.lifespan)
    app.mount(app_config.mcp_mount_path(), mcp_app)

    @app.get("/healthz", response_class=Response)
    async def health_check(request: Request) -> Response:
        """
        Health check endpoint.

        Args:
            request (Request): The incoming HTTP request.
        Returns:
            Response: A JSON response indicating the server status.
        """
        return JSONResponse({"status": "ok"})

    log.info(
        f"Starting {mcp.name} at http://{app_config.host()}:{app_config.port()}{app_config.mcp_mount_path()}{suffix_path}"
    )
    # Use Uvicorn's Config and Server classes for more control
    config = uvicorn.Config(
        app,
        host=app_config.sysdig_endpoint(),
        port=app_config.port(),
        timeout_graceful_shutdown=1,
        log_level=app_config.log_level().lower(),
    )
    server = uvicorn.Server(config)

    # Override the default behavior
    server.force_exit = True  # This makes Ctrl+C force exit
    try:
        asyncio.run(server.serve())
    except KeyboardInterrupt:
        log.info("Keyboard interrupt received, forcing immediate exit")
        os._exit(0)
    except Exception as e:
        log.error(f"Exception received, forcing immediate exit: {str(e)}")
        os._exit(1)


def add_tools(mcp: FastMCP) -> None:
    """
    Registers the tools to the MCP Server.

    Args:
        mcp (FastMCP): The FastMCP server instance.
    """
    # Register the events feed tools
    events_feed_tools = EventsFeedTools(app_config)
    log.info("Adding Events Feed Tools...")
    mcp.add_tool(
        events_feed_tools.tool_get_event_info,
        name="get_event_info",
        description="Retrieve detailed information for a specific security event by its ID",
    )
    mcp.add_tool(
        events_feed_tools.tool_list_runtime_events,
        name="list_runtime_events",
        description="List runtime security events from the last given hours, optionally filtered by severity level.",
    )

    mcp.add_prompt(
        events_feed_tools.investigate_event_prompt,
        name="investigate_event",
        description="Prompt to investigate a security event based on its severity and time range.",
        tags={"analysis", "secure_feeds"},
    )
    mcp.add_tool(
        events_feed_tools.tool_get_event_process_tree,
        name="get_event_process_tree",
        description=(
            """
            Retrieve the process tree for a specific security event by its ID. Not every event has a process tree,
            so this may return an empty tree.
        """
        ),
    )

    # Register the Sysdig Inventory tools
    log.info("Adding Sysdig Inventory Tools...")
    inventory_tools = InventoryTools(app_config)
    mcp.add_tool(
        inventory_tools.tool_list_resources,
        name="list_resources",
        description=(
            """
            List inventory resources based on Sysdig Filter Query Language expression with optional pagination.'
            """
        ),
    )
    mcp.add_tool(
        inventory_tools.tool_get_resource,
        name="get_resource",
        description="Retrieve a single inventory resource by its unique hash identifier.",
    )

    # Register the Sysdig Vulnerability Management tools
    log.info("Adding Sysdig Vulnerability Management Tools...")
    vulnerability_tools = VulnerabilityManagementTools(app_config)
    mcp.add_tool(
        vulnerability_tools.tool_list_runtime_vulnerabilities,
        name="list_runtime_vulnerabilities",
        description=(
            """
            List runtime vulnerability assets scan results from Sysdig Vulnerability Management API
            (Supports pagination using cursor).
            """
        ),
    )
    mcp.add_tool(
        vulnerability_tools.tool_list_accepted_risks,
        name="list_accepted_risks",
        description="List all accepted risks. Supports filtering and pagination.",
    )
    mcp.add_tool(
        vulnerability_tools.tool_get_accepted_risk,
        name="get_accepted_risk",
        description="Retrieve a specific accepted risk by its ID.",
    )
    mcp.add_tool(
        vulnerability_tools.tool_list_registry_scan_results,
        name="list_registry_scan_results",
        description="List registry scan results. Supports filtering and pagination.",
    )
    mcp.add_tool(
        vulnerability_tools.tool_get_vulnerability_policy,
        name="get_vulnerability_policy_by_id",
        description="Retrieve a specific vulnerability policy by its ID.",
    )
    mcp.add_tool(
        vulnerability_tools.tool_list_vulnerability_policies,
        name="list_vulnerability_policies",
        description="List all vulnerability policies. Supports filtering, pagination, and sorting.",
    )
    mcp.add_tool(
        vulnerability_tools.tool_list_pipeline_scan_results,
        name="list_pipeline_scan_results",
        description="List pipeline scan results (e.g., built images). Supports pagination and filtering.",
    )
    mcp.add_tool(
        vulnerability_tools.tool_get_scan_result,
        name="get_scan_result",
        description="Retrieve a specific scan result (registry/runtime/pipeline).",
    )
    mcp.add_prompt(
        vulnerability_tools.explore_vulnerabilities_prompt,
        name="explore_vulnerabilities",
        description="Prompt to explore vulnerabilities based on filters",
        tags={"vulnerability", "exploration"},
    )

    # Register the Sysdig Sage tools
    log.info("Adding Sysdig Sage Tools...")
    sysdig_sage_tools = SageTools(app_config)
    mcp.add_tool(
        sysdig_sage_tools.tool_sage_to_sysql,
        name="sysdig_sysql_sage_query",
        description=(
            """
            Query Sysdig Sage to generate a SysQL query based on a natural language question,
            execute it against the Sysdig API, and return the results.
            """
        ),
    )

    if app_config.transport() == "stdio":
        # Register the tools for STDIO transport
        cli_scanner_tool = CLIScannerTool(app_config)
        log.info("Adding Sysdig CLI Scanner Tool...")
        mcp.add_tool(
            cli_scanner_tool.run_sysdig_cli_scanner,
            name="run_sysdig_cli_scanner",
            description=(
                """
                Run the Sysdig CLI Scanner to analyze a container image or IaC files for vulnerabilities
                and posture and misconfigurations.
                """
            ),
        )


def add_resources(mcp: FastMCP) -> None:
    """
    Add resources to the MCP server.
    Args:
        mcp (FastMCP): The FastMCP server instance.
    """
    vm_docs = HttpResource(
        name="Sysdig Secure Vulnerability Management Overview",
        description="Sysdig Secure Vulnerability Management documentation.",
        uri="resource://sysdig-secure-vulnerability-management",
        url="https://docs.sysdig.com/en/sysdig-secure/vulnerability-management/",
        tags=["documentation"],
    )
    filter_query_language = TextResource(
        name="Sysdig Filter Query Language",
        description=(
            "Sysdig Filter Query Language documentation. "
            "Learn how to filter resources in Sysdig using the Filter Query Language for the API calls."
        ),
        uri="resource://filter-query-language",
        text=(
            """
            Query language expressions for filtering results.
            The query language allows you to filter resources based on their attributes.
            You can use the following operators and functions to build your queries:

            Operators:
                - `and` and `not` logical operators
                - `=`, `!=`
                - `in`
                - `contains` and `startsWith` to check partial values of attributes
                - `exists` to check if a field exists and not empty

            Note:
                The supported fields are going to depend on the API endpoint you are querying.
                Check the description of each tool for the supported fields.
            
            Examples:
                - <field1> in ("example") and <field2> = "example2"
                - <field3> >= "3"
            """
        ),
        tags=["query-language", "documentation"],
    )
    mcp.add_resource(vm_docs)
    mcp.add_resource(filter_query_language)
