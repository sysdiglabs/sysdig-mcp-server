"""
Token-based authentication middleware
"""

import json
import logging
import os
from starlette.middleware.base import BaseHTTPMiddleware, RequestResponseEndpoint
from starlette.requests import Request
from starlette.responses import Response
from starlette.types import ASGIApp

from utils.sysdig.api import initialize_api_client, get_sysdig_api_instances
from utils.sysdig.client_config import get_configuration
from utils.sysdig.old_sysdig_api import OldSysdigApi
from utils.app_config import AppConfig


class CustomAuthMiddleware(BaseHTTPMiddleware):
    """
    Custom middleware for handling token-based authentication in the MCP server and initializing Sysdig API clients.
    """

    def __init__(self, app: ASGIApp, app_config: AppConfig):
        super().__init__(app)
        self.app_config = app_config
        # Set up logging
        logging.basicConfig(format="%(asctime)s-%(process)d-%(levelname)s- %(message)s", level=self.app_config.log_level())
        self.log = logging.getLogger(__name__)

    async def dispatch(self, request: Request, call_next: RequestResponseEndpoint) -> Response:
        """
        Dispatch method to handle incoming requests, validate the Authorization header,
        and initialize the Sysdig API client with the provided token and base URL.
        Args:
            request (Request): The incoming HTTP request.
            call_next (RequestResponseEndpoint): The next middleware or endpoint to call.
        Returns:
            Response: The response from the next middleware or endpoint, or an error response if authentication fails.
        """

        auth_header = request.headers.get("Authorization")
        if not auth_header or not auth_header.startswith("Bearer "):
            json_response = {"error": "Unauthorized", "message": "Missing or invalid Authorization header"}
            return Response(json.dumps(json_response), status_code=401)
        # set header to be used by the API client

        # Extract releavant information from the request headers
        token = auth_header.removeprefix("Bearer ").strip()
        session_id = request.headers.get("mcp-session-id", "")
        base_url = request.headers.get("X-Sysdig-Host", self.app_config.sysdig_endpoint()) or str(request.base_url)
        self.log.info(f"Received request with session ID: {session_id}")
        self.log.info(f"Using Sysdig API base URL: {base_url}")

        # Initialize the API client with the token and base URL
        cfg = get_configuration(token, base_url)
        cfg_sage = get_configuration(token, base_url, old_api=True)
        api_client = initialize_api_client(cfg)
        old_sysdig_api = initialize_api_client(cfg_sage)
        api_instances = get_sysdig_api_instances(api_client)
        _old_sysdig_api = OldSysdigApi(old_sysdig_api)
        api_instances["old_sysdig_api"] = _old_sysdig_api
        # Having access to the Sysdig API instances per request to be used by the MCP tools
        request.state.api_instances = api_instances

        try:
            response = await call_next(request)
            return response
        except Exception as e:
            return Response(f"Internal server error: {str(e)}", status_code=500)


def create_auth_middleware(app_config: AppConfig):
    """
    Factory function to create the CustomAuthMiddleware with injected app_config.

    Args:
        app_config (AppConfig): The application configuration object

    Returns:
        A middleware class that can be instantiated by Starlette
    """

    class ConfiguredAuthMiddleware(CustomAuthMiddleware):
        def __init__(self, app: ASGIApp):
            super().__init__(app, app_config)

    return ConfiguredAuthMiddleware