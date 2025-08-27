"""
Auth configuration for the MCP server.
"""

from typing import Optional

from fastmcp.server.auth import OAuthProvider
from fastmcp.server.auth.providers.jwt import JWTVerifier
from pydantic import AnyHttpUrl
from utils.app_config import AppConfig


# FIXME: Need to implement OAuthProxy in v 2.12.0 not yet available in Pip repo
# https://gofastmcp.com/servers/auth/oauth-proxy
def obtain_remote_auth_provider(app_config: AppConfig) -> OAuthProvider:
    # Configure token validation for your identity provider

    # Create the remote auth provider
    remote_auth_provider: Optional[OAuthProvider] = None

    if app_config.oauth_enabled():
        token_verifier = JWTVerifier(
            jwks_uri=app_config.oauth_jwks_uri(),
            issuer=app_config.oauth_auth_endpoint(),
            audience=app_config.oauth_audience(),
            required_scopes=app_config.oauth_required_scopes(),
        )

        remote_auth_provider = OAuthProvider(
            token_verifier=token_verifier,
            upstream_authorization_endpoint=AnyHttpUrl(app_config.oauth_auth_endpoint()),
            upstream_token_endpoint=AnyHttpUrl(app_config.oauth_token_endpoint()),
            upstream_client_id=app_config.oauth_client_id(),
            upstream_client_secret=app_config.oauth_client_secret(),
            base_url=app_config.mcp_base_url(),
            redirect_path=app_config.oauth_redirect_path(),
            allowed_client_redirect_uris=app_config.oauth_allowed_client_redirect_uris(),
        )

    return remote_auth_provider
