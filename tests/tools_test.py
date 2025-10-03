"""
This is a simple test client to verify the MCP server is running and responding to requests.
"""

from fastmcp import Client
from fastmcp.client.transports import StreamableHttpTransport
import os
import pytest


@pytest.mark.asyncio
async def test_list_runtime_events():
    """
    Test the list_runtime_events tool of the MCP server.
    This function initializes a client, calls the list_runtime_events tool,
    and prints the status code and total number of runtime events retrieved.
    """

    async with Client(
        transport=StreamableHttpTransport(
            url="http://localhost:8080/sysdig-mcp-server/mcp",
            headers={"X-Sysdig-Token": f"Bearer {os.getenv('SYSDIG_MCP_API_SECURE_TOKEN')}"},
        ),
        auth="oauth",  # Use "oauth" if the server is configured with OAuth authentication, otherwise use None
    ) as client:
        tool_name = "list_runtime_events"
        result = await client.call_tool(tool_name)
        assert result.structured_content.get("status_code") == 200
        print(
            f"Tool {tool_name} completed with status code: {result.structured_content.get('status_code')}"
            f" with a total of: {result.data.get('results', {}).get('page', {}).get('total', 0)} runtime events."
        )
