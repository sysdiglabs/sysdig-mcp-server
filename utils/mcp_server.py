"""
This module provides the FastMCP server for Sysdig Secure tools.
It includes endpoints for Sysdig Secure Events Feed, Inventory, Vulnerability Management, and Sysdig Sage tools.
"""

import logging
import os
from typing import Optional
import uvicorn
from starlette.requests import Request
from starlette.responses import JSONResponse, Response
from fastapi import FastAPI
from fastmcp import FastMCP
from fastmcp.resources import HttpResource, TextResource
from utils.middleware.auth import CustomAuthMiddleware
from starlette.middleware import Middleware
from tools.events_feed.tool import EventsFeedTools
from tools.inventory.tool import InventoryTools
from tools.vulnerability_management.tool import VulnerabilityManagementTools
from tools.sysdig_sage.tool import SageTools

# Application config loader
from utils.app_config import get_app_config

# Set up logging
log = logging.getLogger(__name__)
logging.basicConfig(
    format="%(asctime)s-%(process)d-%(levelname)s- %(message)s",
    level=os.environ.get("LOGLEVEL", "ERROR"),
)

# Load app config (expects keys: mcp.host, mcp.port, mcp.transport)
app_config = get_app_config()

_mcp_instance: Optional[FastMCP] = None

middlewares = [Middleware(CustomAuthMiddleware)]


def create_simple_mcp_server() -> FastMCP:
    """
    Instantiate and configure the FastMCP server.

    Returns:
        FastMCP: An instance of the FastMCP server configured with Sysdig Secure tools and resources.
    """
    return FastMCP(
        name="Sysdig MCP Server",
        instructions="Provides Sysdig Secure tools and resources.",
        host=app_config["mcp"]["host"],
        port=app_config["mcp"]["port"],
        debug=True,
        tags=["sysdig", "mcp", os.environ.get("MCP_TRANSPORT", app_config["mcp"]["transport"]).lower()],
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


async def run_stdio():
    """
    Run the MCP server using STDIO transport.
    """
    mcp = get_mcp()
    # Add tools to the MCP server
    add_tools(mcp)
    # Add resources to the MCP server
    add_resources(mcp)
    await mcp.run_stdio_async()


def run_http():
    """Run the MCP server over HTTP/SSE transport via Uvicorn."""
    mcp = get_mcp()
    # Add tools to the MCP server
    add_tools(mcp)
    # Add resources to the MCP server
    add_resources(mcp)
    # Mount the MCP HTTP/SSE app at '/sysdig-mcp-server'
    mcp_app = mcp.http_app(
        path="/mcp", transport=os.environ.get("MCP_TRANSPORT", app_config["mcp"]["transport"]).lower(), middleware=middlewares
    )
    app = FastAPI(lifespan=mcp_app.lifespan)
    app.mount("/sysdig-mcp-server", mcp_app)

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

    print(f"Starting {mcp.name} at http://{app_config['app']['host']}:{app_config['app']['port']}/sysdig-mcp-server/mcp")
    uvicorn.run(
        app,
        port=app_config["app"]["port"],
        host=app_config["app"]["host"],
        log_level=os.environ.get("LOGLEVEL", app_config["app"]["log_level"]).lower(),
    )


def add_tools(mcp: FastMCP) -> None:
    """Add tools to the MCP server."""

    # Register the events feed tools
    events_feed_tools = EventsFeedTools()
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
    inventory_tools = InventoryTools()
    mcp.add_tool(
        inventory_tools.tool_list_resources,
        name="list_resources",
        description=(
            """
            List inventory resources based on a Sysdig Secure query filter expression with optional pagination.'
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
    vulnerability_tools = VulnerabilityManagementTools()
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
    sysdig_sage_tools = SageTools()
    mcp.add_tool(
        sysdig_sage_tools.tool_sysdig_sage,
        name="sysdig_sysql_sage_query",
        description=(
            """
            Query Sysdig Sage to generate a SysQL query based on a natural language question,
            execute it against the Sysdig API, and return the results.
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

            Examples:
                - zone in ("zone1") and machineImage = "ami-0b22b359fdfabe1b5"
                - (projectId = "1235495521" or projectId = "987654321") and vuln.severity in ("Critical")
                - vuln.name in ("CVE-2023-0049")
                - vuln.cvssScore >= "3"
                - asset.type = "host"
                - cloudProvider = "gcp" and gcp.project.id = "my-project"
            Note:
                The supported fields are going to depend on the API endpoint you are querying.
                Chek the description of each tool for the supported fields.
            """
        ),
        tags=["query-language", "documentation"],
    )
    mcp.add_resource(vm_docs)
    mcp.add_resource(filter_query_language)
