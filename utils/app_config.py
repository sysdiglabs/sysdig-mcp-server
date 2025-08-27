"""
Utility functions to load and manage the application configuration.
It will load a single configuration class object that can be accessed throughout the application.
"""

import os
from typing import Optional, List

# app_config singleton
_app_config: Optional[dict] = None

# Use ENV_PREFIX to avoid conflicts with other environment variables
ENV_PREFIX = "SYSDIG_MCP_"


class AppConfig:
    """
    A class to encapsulate the application configuration.
    """

    def sysdig_endpoint(self) -> str:
        """
        Get the Sysdig endpoint.

        Raises:
            RuntimeError: If no SYSDIG_MCP_API_HOST environment variable is set.
        Returns:
            str: The Sysdig API host (e.g., "https://us2.app.sysdig.com").
        """
        if f"{ENV_PREFIX}API_HOST" not in os.environ:
            raise RuntimeError(f"Variable `{ENV_PREFIX}API_HOST` must be defined.")

        return os.environ[f"{ENV_PREFIX}API_HOST"]

    def sysdig_secure_token(self) -> str:
        """
        Get the Sysdig secure token.

        Raises:
             RuntimeError: If no SYSDIG_MCP_API_SECURE_TOKEN environment variable is set.
        Returns:
            str: The Sysdig secure token.
        """
        if f"{ENV_PREFIX}API_SECURE_TOKEN" not in os.environ:
            raise RuntimeError(f"Variable `{ENV_PREFIX}API_SECURE_TOKEN` must be defined.")

        return os.environ[f"{ENV_PREFIX}API_SECURE_TOKEN"]

    # MCP Config Vars
    def transport(self) -> str:
        """
        Get the transport protocol (lower case).
        Valid values are: "stdio", "streamable-http", or "sse".
        Defaults to "stdio".

        Raises:
            ValueError: If no transport protocol environment variable is set.
        Returns:
            str: The transport protocol (e.g., "stdio", "streamable-http", or "sse").
        """
        transport = os.environ.get(f"{ENV_PREFIX}TRANSPORT", "stdio").lower()

        if transport not in ("stdio", "streamable-http", "sse"):
            raise ValueError(
                "Invalid transport protocol. Valid values are: stdio, streamable-http, sse. (sse will be deprecated)"
            )

        return transport

    def log_level(self) -> str:
        """
        Get the log level from the environment or defaults to INFO.

        Returns:
            str: The log level string (e.g., "DEBUG", "INFO", "WARNING", "ERROR").
        """
        return os.environ.get(f"{ENV_PREFIX}LOGLEVEL", "ERROR")

    def port(self) -> int:
        """
        Get the port for the remote MCP Server Deployment ("streamable-http", or "sse" transports).
        Defaults to `8080`.

        Returns:
            int: The MCP server port.
        """
        return int(os.environ.get(f"{ENV_PREFIX}LISTENING_PORT", "8080"))

    #
    def host(self) -> str:
        """
        Get the host for the remote MCP Server deployment ("streamable-http", or "sse" transports).
        Defaults to "localhost".

        Returns:
            str: The host string (e.g., "localhost").
        """
        return os.environ.get(f"{ENV_PREFIX}LISTENING_HOST", "localhost")

    def mcp_mount_path(self) -> str:
        """
        Get the string value for the remote MCP Mount Path.

        Returns:
            str: The MCP mount path.
        """
        return os.environ.get(f"{ENV_PREFIX}MOUNT_PATH", "/sysdig-mcp-server")

    def oauth_jwks_uri(self) -> str:
        """
        Get the string value for the remote OAuth JWKS URI.
        Returns:
            str: The OAuth JWKS URI.
        """
        return os.environ.get(f"{ENV_PREFIX}OAUTH_JWKS_URI", "")

    def oauth_auth_endpoint(self) -> str:
        """
        Get the string value for the remote OAuth Auth Endpoint.
        Returns:
            str: The OAuth Auth Endpoint.
        """
        return os.environ.get(f"{ENV_PREFIX}OAUTH_AUTH_ENDPOINT", "")

    def oauth_token_endpoint(self) -> str:
        """
        Get the string value for the remote OAuth Token Endpoint.
        Returns:
            str: The OAuth Token Endpoint.
        """
        return os.environ.get(f"{ENV_PREFIX}OAUTH_TOKEN_ENDPOINT", "")

    def oauth_required_scopes(self) -> List[str]:
        """
        Get the list of required scopes for the remote OAuth.
        Returns:
            List[str]: The list of scopes.
        """
        raw = os.environ.get(f"{ENV_PREFIX}OAUTH_REQUIRED_SCOPES", "")
        if not raw:
            return []
        # Support comma-separated scopes in env var
        return [s.strip() for s in raw.split(",") if s.strip()]

    def oauth_audience(self) -> str:
        """
        Get the string value for the remote OAuth Audience.
        Returns:
            str: The OAuth Audience.
        """
        return os.environ.get(f"{ENV_PREFIX}OAUTH_AUDIENCE", "")

    def oauth_client_id(self) -> str:
        """
        Get the string value for the remote OAuth Client ID.
        Returns:
            str: The OAuth Client ID.
        """
        return os.environ.get(f"{ENV_PREFIX}OAUTH_CLIENT_ID", "")

    def oauth_client_secret(self) -> str:
        """
        Get the string value for the remote OAuth Client Secret.
        Returns:
            str: The OAuth Client Secret.
        """
        return os.environ.get(f"{ENV_PREFIX}OAUTH_CLIENT_SECRET", "")

    def mcp_base_url(self) -> str:
        """
        Get the string value for the remote MCP Base URL.
        Returns:
            str: The MCP Base URL.
        """
        return os.environ.get(f"{ENV_PREFIX}BASE_URL", "http://localhost:8080")

    def oauth_redirect_path(self) -> str:
        """
        Get the string value for the remote OAuth Redirect Path.
        Returns:
            str: The OAuth Redirect Path.
        """
        return os.environ.get(f"{ENV_PREFIX}OAUTH_REDIRECT_PATH", "/auth/callback")

    def oauth_allowed_client_redirect_uris(self) -> List[str]:
        """
        Get the list of allowed client redirect URIs for the remote OAuth.
        Returns:
            List[str]: The list of allowed client redirect URIs.
        """
        raw = os.environ.get(f"{ENV_PREFIX}OAUTH_ALLOWED_CLIENT_REDIRECT_URIS", "http://localhost:8080")
        if not raw:
            return []
        # Support comma-separated URIs in env var
        return [s.strip() for s in raw.split(",") if s.strip()]

    def oauth_enabled(self) -> bool:
        """
        Check to enable the remote OAuth.
        Returns:
            bool: Whether the remote OAuth should be enabled or not.
        """
        return os.environ.get(f"{ENV_PREFIX}OAUTH_ENABLED", "false").lower() == "true"

    def oauth_resource_server_uri(self) -> str:
        """
        Get the string value for the remote OAuth Server Resource URI.
        Returns:
            str: The OAuth Resource URI.
        """
        return os.environ.get(f"{ENV_PREFIX}OAUTH_RESOURCE_SERVER_URI", "[]")


def get_app_config() -> AppConfig:
    """
    Get the overall app config
    This function uses a singleton pattern to ensure the config is loaded only once.
    If the config is already loaded, it returns the existing config.

    Returns:
        AppConfig: The singleton application configuration wrapper.
    """
    global _app_config
    if _app_config is None:
        _app_config = AppConfig()
    return _app_config
