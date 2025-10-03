"""
Module for general utilities and fixtures used in tests.
"""

import json
import pytest
import os
from utils.app_config import AppConfig
from unittest.mock import MagicMock, create_autospec, patch
from fastmcp.server.context import Context
from sysdig_client import SecureEventsApi, ApiClient
from utils.sysdig.legacy_sysdig_api import LegacySysdigApi
from fastmcp.server import FastMCP


def util_load_json(path):
    """
    Utility function to load a JSON file from the given path.
    Args:
        path (str): The path to the JSON file.
    Returns:
        dict: The loaded JSON data.
    """
    with open(path, encoding="utf-8") as f:
        return json.loads(f.read())


@pytest.fixture
def mock_success_response():
    """
    Fixture to mock the urllib3.PoolManager.request method

    Yields:
        MagicMock: A mocked request object that simulates a successful HTTP response.
    """
    with patch("urllib3.PoolManager.request") as mock_request:
        mock_resp = patch("urllib3.response.HTTPResponse").start()
        mock_resp.status = 200
        mock_resp.data = b"{}"
        mock_request.return_value = mock_resp
        yield mock_request
        patch.stopall()


@pytest.fixture
def mock_creds():
    """
    Fixture to set up mocked credentials.
    """
    os.environ["SYSDIG_MCP_API_SECURE_TOKEN"] = os.environ.get("SYSDIG_MCP_API_SECURE_TOKEN", "mocked_token")
    os.environ["SYSDIG_MCP_API_HOST"] = os.environ.get("SYSDIG_MCP_API_HOST", "https://us2.app.sysdig.com")


def mock_app_config() -> AppConfig:
    """
    Utility function to create a mocked AppConfig instance.
    Returns:
        AppConfig: A mocked AppConfig instance.
    """
    mock_cfg = create_autospec(AppConfig, instance=True)

    mock_cfg.sysdig_endpoint.return_value = "https://us2.app.sysdig.com"
    mock_cfg.transport.return_value = "stdio"
    mock_cfg.log_level.return_value = "DEBUG"
    mock_cfg.port.return_value = 8080

    return mock_cfg


@pytest.fixture
def mock_context() -> Context:
    """
    Utility function to create a mocked FastMCP context.
    Returns:
        Context: A mocked FastMCP context.
    """

    ctx = Context(MagicMock(spec=FastMCP))

    api_instances = {
        "secure_events": SecureEventsApi(ApiClient()),
        "legacy_sysdig_api": LegacySysdigApi(ApiClient()),
    }
    ctx.set_state("api_instances", api_instances)
    return ctx
