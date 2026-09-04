# Troubleshooting

**Problem**: Tool not appearing in MCP client
- **Solution**: Check API token permissions match tool's `WithRequiredPermissions()`. The token must have **all** permissions listed.

**Problem**: Server exits with a missing configuration error
- **Solution**: All transports require absolute `SYSDIG_MCP_API_HOST` and `SYSDIG_MCP_API_TOKEN` values. Remote transports also require `SYSDIG_MCP_RESOURCE_URL`, `SYSDIG_MCP_AUTH_ISSUER`, and `SYSDIG_MCP_AUTH_JWKS_URL`. The resource URL path must exactly match `SYSDIG_MCP_MOUNT_PATH`.

**Problem**: Remote request returns `401 Unauthorized`
- **Solution**: Use an issuer-signed MCP access token, not the Sysdig API token. Verify its `iss`, `aud`, expiry, asymmetric signing algorithm, and required scopes. Follow the `resource_metadata` URL in the `WWW-Authenticate` response header to inspect the server's OAuth metadata.

**Problem**: Browser request returns `403 Forbidden`
- **Solution**: Add the browser's exact origin to `SYSDIG_MCP_ALLOWED_ORIGINS`. Include only `scheme://authority`; wildcard origins and origins with paths are rejected. Hostname matching is case-insensitive.

**Problem**: Sysdig API connection fails with "certificate signed by unknown authority"
- **Solution**: If the Sysdig API uses a self-signed certificate (e.g. on-prem), set `SYSDIG_MCP_API_SKIP_TLS_VERIFICATION=true`.

**Problem**: OAuth JWKS retrieval fails with "certificate signed by unknown authority"
- **Solution**: If the authorization server's JWKS endpoint uses a self-signed certificate, set `SYSDIG_MCP_AUTH_JWKS_SKIP_TLS_VERIFICATION=true`. This is intentionally separate from the Sysdig API TLS setting because the two endpoints are different trust domains.

**Problem**: Tests failing with "command not found"
- **Solution**: Enter Nix shell with `nix develop` or `direnv allow`. All dev tools are provided by the flake.

**Problem**: Prek hooks not running
- **Solution**: Run `prek install` to install git hooks, then `prek run -a` to test all files.
