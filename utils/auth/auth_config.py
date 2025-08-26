"""
Auth configuration for the MCP server.
"""
from typing import Optional

from fastmcp.server.auth import RemoteAuthProvider
from fastmcp.server.auth.providers.jwt import JWTVerifier
from pydantic import AnyHttpUrl
from utils.app_config import AppConfig


def obtain_remote_auth_provider(app_config: AppConfig) -> RemoteAuthProvider:
    # Configure token validation for your identity provider

    # Create the remote auth provider
    remote_auth_provider: Optional[RemoteAuthProvider] = None

    if app_config.oauth_enabled():
        token_verifier = JWTVerifier(
            jwks_uri=app_config.oauth_jwks_uri(),
            issuer=app_config.oauth_issuer(),
            audience=app_config.oauth_audience(),
            required_scopes=app_config.oauth_required_scopes()
        )

        remote_auth_provider = RemoteAuthProvider(
            token_verifier=token_verifier,
            authorization_servers=[AnyHttpUrl(app_config.oauth_issuer())],
            resource_server_url=AnyHttpUrl(app_config.oauth_resource_server_uri()),
        )

    return remote_auth_provider