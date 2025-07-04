import json
import pytest
import os
from unittest.mock import patch
from fastmcp import Context
from fastmcp import FastMCP


class MockMCP(FastMCP):
    """
    Mock class for FastMCP
    """

    pass


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
def mock_ctx():
    """
    Fixture to create a mock Context object with 'fastmcp' tags.
    Returns:
        Context: A mocked Context object with 'fastmcp' tags.
    """

    fastmcp: MockMCP = MockMCP(
        tags=["sysdig", "mcp", "stdio"],
    )
    ctx = Context(fastmcp=fastmcp)
    return ctx


@pytest.fixture
def mock_creds():
    """
    Fixture to set up mocked credentials.
    """
    os.environ["SYSDIG_SECURE_TOKEN"] = "mocked_token"
    os.environ["SYSDIG_HOST"] = "https://mocked.secure"
