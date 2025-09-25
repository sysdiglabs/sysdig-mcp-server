"""
This is a simple test client to verify the MCP server is running and responding to requests.
"""

from fastmcp import Client
from fastmcp.client.transports import StreamableHttpTransport
import asyncio
import os


async def main():
    async with Client(
        transport=StreamableHttpTransport(
            url="http://localhost:8080/sysdig-mcp-server/mcp",
            headers={"X-Sysdig-Token": f"Bearer {os.getenv('SYSDIG_MCP_API_SECURE_TOKEN')}"},
        ),
        auth="oauth",
    ) as client:
        print("✓ Authenticated with Oauth!")
        tool_name = "list_runtime_events"
        result = await client.call_tool(tool_name)
        print(
            f"{tool_name} tool completed with status code: "
            f"{result.structured_content.get('status_code')} with a total of: "
            f"{result.data.get('results', {}).get('page', {}).get('total', 0)} runtime events."
        )


if __name__ == "__main__":
    asyncio.run(main())
