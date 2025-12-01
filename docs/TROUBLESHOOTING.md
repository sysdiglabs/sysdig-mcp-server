# Troubleshooting

**Problem**: Tool not appearing in MCP client
- **Solution**: Check API token permissions match tool's `WithRequiredPermissions()`. The token must have **all** permissions listed.

**Problem**: "unable to authenticate with any method"
- **Solution**: For `stdio`, verify `SYSDIG_MCP_API_HOST` and `SYSDIG_MCP_API_TOKEN` env vars are set correctly. For remote transports, check `Authorization: Bearer <token>` header format.

**Problem**: Tests failing with "command not found"
- **Solution**: Enter Nix shell with `nix develop` or `direnv allow`. All dev tools are provided by the flake.

**Problem**: `generate_sysql` returning 500 error
- **Solution**: This tool requires a regular user API token, not a Service Account token. Switch to a user-based token.

**Problem**: Pre-commit hooks not running
- **Solution**: Run `pre-commit install` to install git hooks, then `pre-commit run -a` to test all files.
