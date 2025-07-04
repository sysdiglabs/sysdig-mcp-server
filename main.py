"""
Main entry point for the MCP server application.
"""

import os
import signal
import sys
import logging
from dotenv import load_dotenv

# Application config loader
from utils.app_config import get_app_config

# Register all tools so they attach to the MCP server
from utils.mcp_server import run_stdio, run_http

# Set up logging
logging.basicConfig(
    format="%(asctime)s-%(process)d-%(levelname)s- %(message)s",
    level=os.environ.get("LOGLEVEL", "ERROR"),
)
log = logging.getLogger(__name__)

# Load environment variables from .env
load_dotenv()

app_config = get_app_config()


def handle_signals():
    def signal_handler(sig, frame):
        log.info(f"Received signal {sig}, shutting down...")
        os._exit(0)

    signal.signal(signal.SIGINT, signal_handler)
    signal.signal(signal.SIGTERM, signal_handler)
    signal.signal(signal.SIGHUP, signal_handler)


def main():
    # Choose transport: "stdio" or "sse" (HTTP/SSE)
    handle_signals()
    transport = os.environ.get("MCP_TRANSPORT", app_config["mcp"]["transport"]).lower()
    log.info("""
    ▄▖     ▌▘    ▖  ▖▄▖▄▖  ▄▖
    ▚ ▌▌▛▘▛▌▌▛▌  ▛▖▞▌▌ ▙▌  ▚ █▌▛▘▌▌█▌▛▘
    ▄▌▙▌▄▌▙▌▌▙▌  ▌▝ ▌▙▖▌   ▄▌▙▖▌ ▚▘▙▖▌
      ▄▌     ▄▌
    """)
    if transport == "stdio":
        # Run MCP server over STDIO (local)
        run_stdio()
    else:
        # Run MCP server over streamable HTTP by default
        run_http()


if __name__ == "__main__":
    try:
        sys.exit(main())
    except KeyboardInterrupt:
        os._exit(0)
