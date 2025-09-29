"""
This module provides the FastMCP server for Sysdig Secure tools.
It includes endpoints for Sysdig Secure Events Feed and Sysdig SysQL tools.
"""

import logging
import os
import asyncio
from typing import Optional
import uvicorn
from fastmcp.prompts import Prompt
from starlette.requests import Request
from starlette.responses import JSONResponse, Response
from fastapi import FastAPI
from fastmcp import FastMCP, Settings
from fastmcp.resources import HttpResource, TextResource

from utils.auth.auth_config import obtain_remote_auth_provider
from utils.auth.middleware.auth import CustomMiddleware
from tools.events_feed.tool import EventsFeedTools
from tools.sysql.tool import SysQLTools
from tools.cli_scanner.tool import CLIScannerTool

# Application config loader
from utils.app_config import AppConfig


class SysdigMCPServer:
    def __init__(self, app_config: AppConfig):
        self.app_config = app_config
        # Set up logging
        self.log = logging.getLogger(__name__)

        middlewares = [CustomMiddleware(app_config)]

        self.mcp_instance: Optional[FastMCP] = FastMCP(
            name="Sysdig MCP Server",
            instructions="Provides Sysdig Secure tools and resources.",
            include_tags={"sysdig_secure"},
            middleware=middlewares,
            auth=obtain_remote_auth_provider(app_config),
        )
        # Add tools to the MCP server
        self.add_tools()
        # Add resources to the MCP server
        self.add_resources()

    def run_stdio(self):
        """
        Run the MCP server using STDIO transport.
        """
        try:
            asyncio.run(self.mcp_instance.run_stdio_async())
        except KeyboardInterrupt:
            self.log.info("Keyboard interrupt received, forcing immediate exit")
            os._exit(0)
        except Exception as e:
            self.log.error(f"Exception received, forcing immediate exit: {str(e)}")
            os._exit(1)

    def run_http(self):
        """Run the MCP server over HTTP/SSE transport via Uvicorn."""

        # Add tools to the MCP server
        transport = self.app_config.transport()

        settings = Settings()

        # Mount the MCP HTTP/SSE app
        mcp_app = self.mcp_instance.http_app(transport=transport)
        suffix_path = settings.streamable_http_path if transport == "streamable-http" else settings.sse_path
        app = FastAPI(lifespan=mcp_app.lifespan)
        app.mount(self.app_config.mcp_mount_path(), mcp_app)

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

        self.log.info(
            f"Starting {self.mcp_instance.name} at http://{self.app_config.host()}:{self.app_config.port()}{self.app_config.mcp_mount_path()}{suffix_path}"
        )
        # Use Uvicorn's Config and Server classes for more control
        config = uvicorn.Config(
            app,
            host=self.app_config.host(),
            port=self.app_config.port(),
            timeout_graceful_shutdown=1,
            log_level=self.app_config.log_level().lower(),
        )
        server = uvicorn.Server(config)

        # Override the default behavior
        server.force_exit = True  # This makes Ctrl+C force exit
        try:
            asyncio.run(server.serve())
        except KeyboardInterrupt:
            self.log.info("Keyboard interrupt received, forcing immediate exit")
            os._exit(0)
        except Exception as e:
            self.log.error(f"Exception received, forcing immediate exit: {str(e)}")
            os._exit(1)

    def add_tools(self) -> None:
        """
        Registers the tools to the MCP Server.

        Args:
            mcp (FastMCP): The FastMCP server instance.
        """
        # Register the events feed tools
        events_feed_tools = EventsFeedTools(self.app_config)
        self.log.info("Adding Events Feed Tools...")
        self.mcp_instance.tool(
            name_or_fn=events_feed_tools.tool_get_event_info,
            name="get_event_info",
            description="Retrieve detailed information for a specific security event by its ID",
            tags={"threat-detection", "sysdig_secure"},
        )
        self.mcp_instance.tool(
            name_or_fn=events_feed_tools.tool_list_runtime_events,
            name="list_runtime_events",
            description="List runtime security events from the last given hours, optionally filtered by severity level.",
            tags={"threat-detection", "sysdig_secure"},
        )

        self.mcp_instance.add_prompt(
            Prompt.from_function(
                fn=events_feed_tools.investigate_event_prompt,
                name="investigate_event",
                description="Prompt to investigate a security event based on its severity and time range.",
                tags={"analysis", "sysdig_secure", "threat_detection"},
            )
        )
        self.mcp_instance.tool(
            name_or_fn=events_feed_tools.tool_get_event_process_tree,
            name="get_event_process_tree",
            description=(
                """
                Retrieve the process tree for a specific security event by its ID. Not every event has a process tree,
                so this may return an empty tree.
            """
            ),
            tags={"threat-detection", "sysdig_secure"},
        )

        # Register the Sysdig SysQL tools
        self.log.info("Adding Sysdig SysQL Tools...")
        sysdig_sysql_tools = SysQLTools(self.app_config)
        self.mcp_instance.tool(
            name_or_fn=sysdig_sysql_tools.tool_generate_and_run_sysql,
            name="generate_and_run_sysql",
            description=(
                """
                Query Sysdig to generate a SysQL query based on a natural language question,
                execute it against the Sysdig API, and return the results.
                """
            ),
            tags={"sysql", "sysdig_secure"},
        )

        if self.app_config.transport() == "stdio":
            # Register the tools for STDIO transport
            cli_scanner_tool = CLIScannerTool(self.app_config)
            self.log.info("Adding Sysdig CLI Scanner Tool...")
            self.mcp_instance.tool(
                name_or_fn=cli_scanner_tool.run_sysdig_cli_scanner,
                name="run_sysdig_cli_scanner",
                description=(
                    """
                    Run the Sysdig CLI Scanner to analyze a container image or IaC files for vulnerabilities
                    and posture and misconfigurations.
                    """
                ),
                tags={"cli-scanner", "sysdig_secure"},
            )

    def add_resources(self) -> None:
        """
        Add resources to the MCP server.
        """
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
        self.mcp_instance.add_resource(filter_query_language)
