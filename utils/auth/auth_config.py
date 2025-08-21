"""
Auth configuration for the MCP server.
"""

from fastmcp.server.auth import RemoteAuthProvider
from fastmcp.server.auth.providers.jwt import JWTVerifier
from pydantic import AnyHttpUrl
from utils.app_config import get_app_config

# Load app config (expects keys: mcp.host, mcp.port, mcp.transport)
app_config = get_app_config()

# Configure token validation for your identity provider
token_verifier = JWTVerifier(
    jwks_uri=get_app_config().get("oauth", {}).get("jwks_uri", ""),
    issuer=get_app_config().get("oauth", {}).get("issuer", ""),
    audience=get_app_config().get("oauth", {}).get("audience", ""),
    required_scopes=get_app_config().get("oauth", {}).get("required_scopes", []),
)

# Create the remote auth provider
remote_auth_provider: RemoteAuthProvider = None
if get_app_config().get("oauth", {}).get("enabled", False):
    remote_auth_provider = RemoteAuthProvider(
        token_verifier=token_verifier,
        authorization_servers=[AnyHttpUrl(get_app_config().get("oauth", {}).get("issuer", ""))],
        resource_server_url=AnyHttpUrl(get_app_config().get("oauth", {}).get("resource_server_url", "")),
    )
