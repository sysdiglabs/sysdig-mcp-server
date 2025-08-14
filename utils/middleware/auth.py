"""
Custom middleware for authorization and access control
"""

import json
import logging
import os
from starlette.requests import Request
from fastmcp.server.middleware import Middleware, MiddlewareContext, CallNext
from utils.sysdig.helpers import TOOL_PERMISSIONS
from fastmcp.tools import Tool
from fastmcp.server.dependencies import get_http_request
from utils.sysdig.api import initialize_api_client, get_sysdig_api_instances
from utils.sysdig.client_config import get_configuration
from utils.sysdig.legacy_sysdig_api import LegacySysdigApi
from utils.app_config import get_app_config

# Set up logging
logging.basicConfig(format="%(asctime)s-%(process)d-%(levelname)s- %(message)s", level=os.environ.get("LOGLEVEL", "ERROR"))
log = logging.getLogger(__name__)

# Load app config (expects keys: mcp.host, mcp.port, mcp.transport)
app_config = get_app_config()


async def _get_permissions(context: MiddlewareContext) -> None:
    """
    Get the permissions for the current user and set them in the context.
    Args:
        context (MiddlewareContext): The middleware context.
    """
    try:
        legacy_sysdig_api = _init_legacy_sysdig_api(context)
        permissions = legacy_sysdig_api.get_me_permissions()
        context.fastmcp_context.set_state("permissions", permissions.json().get("permissions", []))
    except Exception as e:
        log.error(f"Error fetching permissions: {e}")
        raise


def _init_legacy_sysdig_api(context: MiddlewareContext) -> LegacySysdigApi:
    """
    Initialize the legacy Sysdig API client either from the HTTP request state or from environment variables.
    Args:
        context (MiddlewareContext): The middleware context.

    Returns:
        LegacySysdigApi: The initialized legacy Sysdig API client.
    """
    try:
        legacy_sysdig_api: LegacySysdigApi = None
        transport = context.fastmcp_context.get_state("transport_method")
        if transport in ["streamable-http", "sse"]:
            # Try to get the HTTP request
            log.debug("Attempting to get the HTTP request to initialize the Sysdig API client.")
            request: Request = get_http_request()
            legacy_sysdig_api = request.state.api_instances["legacy_sysdig_api"]
        else:
            # If running in STDIO mode, we need to initialize the API client from environment variables
            log.debug("Trying to init the Sysdig API client from environment variables.")
            cfg = get_configuration(old_api=True)
            api_client = initialize_api_client(cfg)
            legacy_sysdig_api = LegacySysdigApi(api_client)
        return legacy_sysdig_api
    except Exception as e:
        log.error(f"Error initializing legacy Sysdig API: {e}")
        raise


async def allowed_tool(context: MiddlewareContext, tool: Tool) -> bool:
    """
    Check if the user has permission to access a specific tool.

    Args:
        context (MiddlewareContext): The middleware context.
        tool_id (str): The ID of the tool to check permissions for.

    Returns:
        bool: True if the user has permission to access the tool, False otherwise.
    """
    permissions = context.fastmcp_context.get_state("permissions")
    if permissions is None:
        # Try to fetch permissions
        await _get_permissions(context)
        permissions = context.fastmcp_context.get_state("permissions")
    for tag in tool.tags:
        if tag in TOOL_PERMISSIONS:
            tool_permissions = TOOL_PERMISSIONS[tag]
            if all(permission in permissions for permission in tool_permissions):
                return True
    log.warning(f"User does not have permission to access tool: {tool.name}")
    return False


async def _save_api_instance() -> None:
    """
    Save the API client instance to the request state.

    Raises:
        Exception: If the Authorization header is missing or invalid.
    """
    request = get_http_request()
    auth_header = request.headers.get("Authorization")
    if not auth_header or not auth_header.startswith("Bearer "):
        raise Exception("Missing or invalid Authorization header")
    # set header to be used by the API client

    # Extract releavant information from the request headers
    token = auth_header.removeprefix("Bearer ").strip()
    base_url = request.headers.get("X-Sysdig-Host", app_config["sysdig"]["host"]) or str(request.base_url)
    log.info(f"Using Sysdig API base URL: {base_url}")

    # Initialize the API client with the token and base URL
    # TODO: Implement a more elegant solution for API client initialization, we will end up having multiple API instances
    cfg = get_configuration(token, base_url)
    legacy_cfg = get_configuration(token, base_url, old_api=True)
    api_client = initialize_api_client(cfg)
    legacy_sysdig_api = initialize_api_client(legacy_cfg)
    api_instances = get_sysdig_api_instances(api_client)
    _legacy_sysdig_api = LegacySysdigApi(legacy_sysdig_api)
    api_instances["legacy_sysdig_api"] = _legacy_sysdig_api
    # Having access to the Sysdig API instances per request to be used by the MCP tools
    request.state.api_instances = api_instances


class CustomMiddleware(Middleware):
    """
    Custom middleware for filtering tool listings and performing authentication.
    """

    # TODO: Evaluate if init the clients and perform auth only on the `notifications/initialized` event
    async def on_message(self, context: MiddlewareContext, call_next: CallNext) -> CallNext:
        """
        Handle incoming messages and initialize the Sysdig API client if needed.
        Returns:
            CallNext: The next middleware or route handler to call.
        Raises:
            Exception: If a problem occurs while initializing the API clients.
        """
        # FIXME: Currently not able to get the notifications/initialized that should be the one initializing the API instances
        # for the whole session
        allowed_notifications = ["notifications/initialized", "tools/list", "tools/call"]
        # Save transport method in context
        if not context.fastmcp_context.get_state("transport_method"):
            transport_method = os.environ.get("MCP_TRANSPORT", app_config["mcp"]["transport"]).lower()
            context.fastmcp_context.set_state("transport_method", transport_method)
        try:
            if (
                context.fastmcp_context.get_state("transport_method") in ["streamable-http", "sse"]
                and context.method in allowed_notifications
            ):
                await _save_api_instance()

            return await call_next(context)
        except Exception as error:
            log.error(f"Error in {context.method}: {type(error).__name__}: {error}")
            raise Exception(f"Error in {context.method}: {type(error).__name__}: {error}")

    async def on_list_tools(self, context: MiddlewareContext, call_next: CallNext) -> list[Tool]:
        """
        Handle listing of tools by checking permissions for the current user.

        Returns:
            list[Tool]: A list of tools that the user is allowed to access.

        Raises:
            Exception: If a problem occurs while checking tool permissions.
        """
        result = await call_next(context)

        filtered_tools = [tool for tool in result if await allowed_tool(context, tool)]

        if not filtered_tools:
            log.warning(f"No allowed tools found for session: {context.fastmcp_context.session_id}")
            raise Exception("No allowed tools found for the user.")
        # Return modified list
        return filtered_tools
