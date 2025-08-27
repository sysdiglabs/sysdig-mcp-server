"""
Custom middleware for access control and initialization of Sysdig API clients.
"""

import logging
import os
from starlette.requests import Request
from fastmcp.server.middleware import Middleware, MiddlewareContext, CallNext
from utils.sysdig.helpers import TOOL_PERMISSIONS
from fastmcp.tools import Tool
from fastmcp.server.dependencies import get_http_request
from utils.sysdig.client_config import initialize_api_client, get_sysdig_api_instances
from utils.sysdig.client_config import get_configuration
from utils.sysdig.legacy_sysdig_api import LegacySysdigApi
from utils.app_config import AppConfig
from utils.app_config import get_app_config

# Set up logging
logging.basicConfig(
    format="%(asctime)s-%(process)d-%(levelname)s- %(message)s",
    level=get_app_config().log_level(),
)
log = logging.getLogger(__name__)

# TODO: Define the correct message notifications
INIT_NOTIFICATIONS = ["notifications/initialized", "tools/list", "tools/call"]


def _get_permissions(context: MiddlewareContext) -> None:
    """
    Get the permissions for the current user/team based on the Bearer token and set them in the context.
    Args:
        context (MiddlewareContext): The middleware context.
    Raises:
        Exception: If fetching permissions fails.
    """
    try:
        api_instances: dict = context.fastmcp_context.get_state("api_instances")
        legacy_api_client: LegacySysdigApi = api_instances.get("legacy_sysdig_api")
        response = legacy_api_client.get_me_permissions()
        if response.status != 200:
            log.error(f"Error fetching permissions: Status {response.status} {legacy_api_client.api_client.configuration.host}")
            raise Exception("Failed to fetch user permissions. Check your current Token and permissions.")
        context.fastmcp_context.set_state("permissions", response.json().get("permissions", []))
    except Exception as e:
        log.error(f"Error fetching permissions: {e}")
        raise


async def allowed_tool(context: MiddlewareContext, tool: Tool) -> bool:
    """
    Check if the user has permission to access a specific tool.
    Args:
        context (MiddlewareContext): The middleware context.
        tool (str): The tool to check permissions for.
    Returns:
        bool: True if the user has permission to access the tool, False otherwise.
    """
    permissions = context.fastmcp_context.get_state("permissions")
    if permissions is None:
        # Try to fetch permissions once
        _get_permissions(context)
        permissions = context.fastmcp_context.get_state("permissions")
    for tag in tool.tags:
        if tag in TOOL_PERMISSIONS:
            tool_permissions = TOOL_PERMISSIONS[tag]
            if all(permission in permissions for permission in tool_permissions):
                return True
    log.warning(f"User does not have permission to access tool: {tool.name}")
    return False


async def _save_api_instances(context: MiddlewareContext, app_config: AppConfig) -> None:
    """
    This method initializes the Sysdig API client and saves the instances to the FastMCP context per request.
    Based on the transport method, it extracts the Bearer token from the Authorization
    header or from the environment variables.
    Raises:
        Exception: If the Authorization header or required env vars are missing.
    """
    cfg = None
    legacy_cfg = None

    if context.fastmcp_context.get_state("transport_method") in ["streamable-http", "sse"]:
        request: Request = get_http_request()
        # TODO: Check for the custom Authorization header or use the default. Will be relevant with the Oauth provider config.
        auth_header = request.headers.get("X-Sysdig-Token", request.headers.get("Authorization"))
        if not auth_header or not auth_header.startswith("Bearer "):
            err = "Missing or invalid Authorization header"
            log.error(err)
            raise Exception(err)

        # Extract relevant information from the request headers
        token = auth_header.removeprefix("Bearer ").strip()
        base_url = request.headers.get("X-Sysdig-Host", app_config.sysdig_endpoint()) or str(request.base_url)
        log.info(f"Using Sysdig API base URL: {base_url}")

        cfg = get_configuration(app_config, token, base_url)
        legacy_cfg = get_configuration(app_config, token, base_url, old_api=True)
    else:
        # If running in STDIO mode, we initialize the API client from environment variables
        cfg = get_configuration(app_config)
        legacy_cfg = get_configuration(app_config, old_api=True)

    api_client = initialize_api_client(cfg)
    legacy_sysdig_api = initialize_api_client(legacy_cfg)
    api_instances = get_sysdig_api_instances(api_client)
    # api_instances have a dictionary of all the Sysdig API instances needed to be accessed in every request
    _legacy_sysdig_api = LegacySysdigApi(legacy_sysdig_api)
    api_instances["legacy_sysdig_api"] = _legacy_sysdig_api
    # Save the API instances to the context
    log.debug("Saving API instances to the context.")
    context.fastmcp_context.set_state("api_instances", api_instances)


class CustomMiddleware(Middleware):
    """
    Custom middleware for filtering tool listings and performing authentication.
    """

    def __init__(self, app_config: AppConfig):
        self.app_config = app_config

    # TODO: Evaluate if init the clients and perform auth only on the `notifications/initialized` event
    async def on_message(self, context: MiddlewareContext, call_next: CallNext) -> CallNext:
        """
        Handle incoming messages and initialize the Sysdig API client if needed.
        Returns:
            CallNext: The next middleware or route handler to call.
        Raises:
            Exception: If a problem occurs while initializing the API clients.
        """
        # Save transport method in context
        if not context.fastmcp_context.get_state("transport_method"):
            transport_method = os.environ.get("MCP_TRANSPORT", self.app_config.transport()).lower()
            context.fastmcp_context.set_state("transport_method", transport_method)
        try:
            # TODO: Currently not able to get the notifications/initialized only that should be the method that initializes
            # the API instances for the whole session, we need to check if its possible
            if context.method in INIT_NOTIFICATIONS:
                await _save_api_instances(context, self.app_config)

            return await call_next(context)
        except Exception as error:
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
        try:
            filtered_tools = [tool for tool in result if await allowed_tool(context, tool)]
            if not filtered_tools:
                raise Exception("No allowed tools found for the user.")
        except Exception as e:
            log.error(f"Error filtering tools: {e}")
            raise
        return filtered_tools
