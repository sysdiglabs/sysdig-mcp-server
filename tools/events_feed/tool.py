"""
Events Feed MCP Tools.

This module defines MCP tools for interacting with the Sysdig Secure Events Feed API,
including retrieving detailed information for a specific event and listing multiple events.
"""

import logging
import os
import time
from datetime import datetime
from typing import Optional, Annotated, Any, Dict
from pydantic import Field
from fastmcp import Context
from sysdig_client import ApiException
from fastmcp.prompts.prompt import PromptMessage, TextContent
from fastmcp.exceptions import ToolError
from starlette.requests import Request
from sysdig_client.api import SecureEventsApi
from utils.sysdig.old_sysdig_api import OldSysdigApi
from fastmcp.server.dependencies import get_http_request
from utils.query_helpers import create_standard_response
from utils.sysdig.client_config import get_configuration
from utils.app_config import get_app_config
from utils.sysdig.api import initialize_api_client

logging.basicConfig(format="%(asctime)s-%(process)d-%(levelname)s- %(message)s", level=os.environ.get("LOGLEVEL", "ERROR"))

log = logging.getLogger(__name__)

# Load app config (expects keys: mcp.host, mcp.port, mcp.transport)
app_config = get_app_config()


class EventsFeedTools:
    """
    A class to encapsulate the tools for interacting with the Sysdig Secure Events Feed API.
    This class provides methods to retrieve event information and list runtime events.
    """

    def init_client(self, config_tags: set[str], old_api: bool = False) -> SecureEventsApi | OldSysdigApi:
        """
        Initializes the SecureEventsApi client from the request state.
        If the request does not have the API client initialized, it will create a new instance
        using the Sysdig Secure token and host from the environment variables.
        Args:
            config_tags (set[str]): The tags associated with the MCP server configuration, used to determine the transport mode.
        Returns:
            SecureEventsApi | OldSysdigApi: An instance of the SecureEventsApi or OldSysdigApi client.
        Raises:
            ValueError: If the SYSDIG_SECURE_TOKEN environment variable is not set.
        """
        secure_events_api: SecureEventsApi = None
        old_sysdig_api: OldSysdigApi = None
        if "streamable-http" in config_tags:
            # Try to get the HTTP request
            log.debug("Attempting to get the HTTP request to initialize the Sysdig API client.")
            request: Request = get_http_request()
            secure_events_api = request.state.api_instances["secure_events"]
            old_sysdig_api = request.state.api_instances["old_sysdig_api"]
        else:
            # If running in STDIO mode, we need to initialize the API client from environment variables
            log.debug("Running in STDIO mode, initializing the Sysdig API client from environment variables.")
            SYSDIG_SECURE_TOKEN = os.environ.get("SYSDIG_SECURE_TOKEN", "")
            if not SYSDIG_SECURE_TOKEN:
                raise ValueError("Can not initialize client, SYSDIG_SECURE_TOKEN environment variable is not set.")
            SYSDIG_HOST = os.environ.get("SYSDIG_HOST", app_config["sysdig"]["host"])
            cfg = get_configuration(SYSDIG_SECURE_TOKEN, SYSDIG_HOST)
            api_client = initialize_api_client(cfg)
            secure_events_api = SecureEventsApi(api_client)
            # Initialize the old Sysdig API client for process tree requests
            old_cfg = get_configuration(SYSDIG_SECURE_TOKEN, SYSDIG_HOST, old_api=True)
            old_sysdig_api = initialize_api_client(old_cfg)
            old_sysdig_api = OldSysdigApi(old_sysdig_api)

        if old_api:
            return old_sysdig_api
        return secure_events_api

    def tool_get_event_info(self, event_id: str, ctx: Context) -> dict:
        """
        Retrieves detailed information for a specific security event.

        Args:
            event_id (str): The unique identifier of the security event.

        Returns:
            Event: The Event object containing detailed information about the specified event.
        """
        # Init of the sysdig client
        secure_events_api = self.init_client(config_tags=ctx.fastmcp.tags)
        try:
            # Get the HTTP request
            start_time = time.time()
            # Get event
            raw = secure_events_api.get_event_v1_without_preload_content(event_id)

            execution_time = (time.time() - start_time) * 1000

            response = create_standard_response(results=raw, execution_time_ms=execution_time)

            return response
        except ToolError as e:
            logging.error("Exception when calling SecureEventsApi->get_event_v1: %s\n" % e)
            raise e

    def tool_list_runtime_events(
        self,
        ctx: Context,
        cursor: Optional[str] = None,
        scope_hours: int = 1,
        limit: int = 50,
        filter_expr: Annotated[
            Optional[str],
            Field(
                description=(
                    """
                    Logical filter expression to select runtime security events. Supports operators: =, !=, in, contains,
                    startsWith, exists.
                    Combine with and/or/not. Key attributes include: severity (codes "0"-"7"), originator, sourceType,
                    ruleName, rawEventCategory,
                    - `severity in ("0","1","2","3")` ← high-severity events
                    - `severity in ("4","5")` ← medium
                    - `severity in ("6")` ← low
                    - `severity in ("7")` ← info,
                    kubernetes.cluster.name, host.hostName, container.imageName, aws.accountId, azure.subscriptionId,
                    gcp.projectId, policyId, trigger.
                    """
                ),
                examples=[
                    'originator in ("awsCloudConnector","gcp") and not sourceType = "auditTrail"',
                    'ruleName contains "Login"',
                    'severity in ("0","1","2","3")',
                    'kubernetes.cluster.name = "cluster1"',
                    'host.hostName startsWith "web-"',
                    'container.imageName = "nginx:latest" and originator = "hostscanning"',
                    'aws.accountId = "123456789012"',
                    'policyId = "CIS_Docker_Benchmark"',
                ],
            ),
        ] = None,
    ) -> dict:
        """
        Retrieve the runtime security events from the last `scope_hours` hours, optionally filtered by severity level,
        cluster name, or an optional filter expression.

        Args:
            cursor (Optional[str]): Cursor for pagination.
            scope_hours (int): Number of hours back from now to include events. Defaults to 1.
            severity_level (Optional[str]): One of "info", "low", "medium", "high". If provided, filters by that severity.
            If None, includes all severities.
            cluster_name (Optional[str]): Name of the Kubernetes cluster to filter events. If None, includes all clusters.
            limit (int): Maximum number of events to return. Defaults to 50.
            filter_expr (Optional[str]): An optional filter expression to further narrow down events.

        Returns:
            List[Event]: A list of Event objects matching the criteria.
        """
        secure_events_api = self.init_client(config_tags=ctx.fastmcp.tags)
        start_time = time.time()
        # Compute time window
        now_ns = time.time_ns()
        from_ts = now_ns - scope_hours * 3600 * 1_000_000_000
        to_ts = now_ns
        base_filter_expr = (
            'source != "auditTrail" and not originator in ("benchmarks","compliance","cloudsec","scanning","hostscanning")'
        )

        if filter_expr:
            # If a filter expression is provided, combine it with the base filter
            filter_expr = f"{base_filter_expr} and ({filter_expr})"
        # Build severity filter expression
        try:
            api_response = secure_events_api.get_events_v1_without_preload_content(
                to=to_ts, var_from=from_ts, filter=filter_expr, limit=limit, cursor=cursor
            )
            duration_ms = (time.time() - start_time) * 1000
            log.debug(f"Execution time: {duration_ms:.2f} ms")

            response = create_standard_response(
                results=api_response,
                execution_time_ms=duration_ms,
            )
            return response
        except ToolError as e:
            log.error(f"Exception when calling SecureEventsApi->get_events_v1: {e}\n")
            raise e

    # A tool to retrieve all the process-tree information for a specific event.Add commentMore actions

    def tool_get_event_process_tree(self, ctx: Context, event_id: str) -> Dict[str, Any]:
        """
        Retrieves the process tree for a specific security event.
        Not every event has a process tree, so this may return an empty tree.

        Args:
            ctx (Context): The context object containing request-specific information.
            event_id (str): The unique identifier of the security event.

        Returns:
            dict: A dictionary containing the process tree information for the specified event.
        """
        try:
            start_time = time.time()
            # Get process tree branches
            old_api_client = self.init_client(config_tags=ctx.fastmcp.tags, old_api=True)
            branches = old_api_client.request_process_tree_branches(event_id)
            # Get process tree
            tree = old_api_client.request_process_tree_trees(event_id)

            # Parse the response
            branches = create_standard_response(results=branches, execution_time_ms=(time.time() - start_time) * 1000)
            tree = create_standard_response(results=tree, execution_time_ms=(time.time() - start_time) * 1000)

            execution_time = (time.time() - start_time) * 1000

            response = (
                {
                    "branches": branches.get("results", []),
                    "tree": tree.get("results", []),
                    "metadata": {
                        "execution_time_ms": execution_time,
                        "timestamp": datetime.utcnow().isoformat() + "Z",
                    },
                },
            )

            return response
        except ToolError as e:
            log.error(f"Exception when calling Sysdig Sage API to get process tree: {e}")
            raise e

    # Prompts
    # Docs: https://modelcontextprotocol.io/docs/concepts/prompts
    def investigate_event_prompt(self, severity: str, relative_time: str) -> PromptMessage:
        """Generates a prompt message for investigating security events.
        Args:
            severity (str): The severity level of the security event (e.g., "high", "medium", "low").
            relative_time (str): The time range for the events to investigate (e.g., "last 24 hours").
        Returns:
            PromptMessage: A message object containing the prompt for investigation.
        """
        content = (
            f"Please investigate security events with severity '{severity}' of the last {relative_time}. "
            "Provide detailed information about the event and any recommended actions."
            "Extract the process ID and the container information."
        )
        return PromptMessage(role="user", content=TextContent(type="text", text=content))
