"""
Events Feed Test Module
"""

from http import HTTPStatus
from tools.events_feed.tool import EventsFeedTools
from utils.app_config import AppConfig
from .conftest import util_load_json
from unittest.mock import MagicMock, AsyncMock, create_autospec
import os

# Get the absolute path of the current module file
module_path = os.path.abspath(__file__)

# Get the directory containing the current module
module_directory = os.path.dirname(module_path)

EVENT_INFO_RESPONSE = util_load_json(f"{module_directory}/test_data/events_feed/event_info_response.json")


def mock_app_config() -> AppConfig:
    mock_cfg = create_autospec(AppConfig, instance=True)

    mock_cfg.sysdig_endpoint.return_value = "https://us2.app.sysdig.com"
    mock_cfg.transport.return_value = "stdio"
    mock_cfg.log_level.return_value = "DEBUG"
    mock_cfg.port.return_value = 8080

    return mock_cfg

def test_get_event_info(mock_success_response: MagicMock | AsyncMock, mock_creds) -> None:
    """Test the get_event_info tool method.
    Args:
        mock_success_response (MagicMock | AsyncMock): Mocked response object.
        mock_creds: Mocked credentials.
    """
    # Successful response
    mock_success_response.return_value.json.return_value = EVENT_INFO_RESPONSE
    mock_success_response.return_value.status_code = HTTPStatus.OK

    tools_client = EventsFeedTools(app_config=mock_app_config())

    # Pass the mocked Context object
    result: dict = tools_client.tool_get_event_info("12345")
    results: dict = result["results"]

    assert result.get("status_code") == HTTPStatus.OK
    assert results.get("results").get("name") == "Sysdig Runtime Threat Intelligence"
    assert results.get("results").get("content", {}).get("ruleName") == "Fileless execution via memfd_create"
    assert results.get("results").get("id") == "123456789012"
    assert results.get("results").get("content", {}).get("type") == "workloadRuntimeDetection"
