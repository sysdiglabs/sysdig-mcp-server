"""
Main entry point for the MCP server application.
"""

import os
import asyncio
from dotenv import load_dotenv

# Application config loader
from utils.app_config import get_app_config

# Register all tools so they attach to the MCP server
from utils.mcp_server import run_stdio, run_http

# Load environment variables from .env
load_dotenv()

app_config = get_app_config()

if __name__ == "__main__":
    # Choose transport: "stdio" or "sse" (HTTP/SSE)
    transport = os.environ.get("MCP_TRANSPORT", app_config["mcp"]["transport"]).lower()
    print("""
    ▄▖     ▌▘    ▖  ▖▄▖▄▖  ▄▖
    ▚ ▌▌▛▘▛▌▌▛▌  ▛▖▞▌▌ ▙▌  ▚ █▌▛▘▌▌█▌▛▘
    ▄▌▙▌▄▌▙▌▌▙▌  ▌▝ ▌▙▖▌   ▄▌▙▖▌ ▚▘▙▖▌
      ▄▌     ▄▌
    """)
    if transport == "stdio":
        # Run MCP server over STDIO (local)
        asyncio.run(run_stdio())
    else:
        # Run MCP server over streamable HTTP by default
        run_http()
