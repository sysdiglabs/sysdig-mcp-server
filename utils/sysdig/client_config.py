"""
This module provides a function to configure the Sysdig client based on a configuration.
"""

import sysdig_client
import logging
import re
from typing import Optional

# Application config loader
from utils.app_config import AppConfig
from sysdig_client.configuration import Configuration
from sysdig_client import ApiClient, SecureEventsApi, VulnerabilityManagementApi, InventoryApi

# Set up logging
log = logging.getLogger(__name__)


def initialize_api_client(config: Configuration = None) -> ApiClient:
    """
    Initializes the Sysdig API client with the provided configuration.
    Args:
        config (Configuration): The Sysdig client configuration containing the access token and host URL.
    Returns:
        ApiClient: An instance of ApiClient configured with the provided settings.
    """
    api_client = ApiClient(config)
    return api_client


def get_sysdig_api_instances(api_client: ApiClient) -> dict:
    """
    Returns a dictionary of Sysdig API instances using the provided ApiClient.
    Args:
        api_client (ApiClient): The ApiClient instance to use for creating API instances.
    Returns:
        dict: A dictionary containing instances of multiple Sysdig API classes.
    """
    return {
        "secure_events": SecureEventsApi(api_client),
        "vulnerability_management": VulnerabilityManagementApi(api_client),
        "inventory": InventoryApi(api_client),
    }

# Lazy-load the Sysdig client configuration
def get_configuration(
    app_config: AppConfig,
    token: Optional[str] = None,
    sysdig_host_url: Optional[str] = None,
    old_api: bool = False
) -> sysdig_client.Configuration:
    """
    Returns a configured Sysdig client using environment variables.

    Args:
        app_config (SysdigClient): MCP Server configuration.
        token (str): The Sysdig Secure token.
        sysdig_host_url (str): The base URL of the Sysdig API,
            refer to the docs https://docs.sysdig.com/en/administration/saas-regions-and-ip-ranges/#sysdig-platform-regions.
        old_api (bool): If True, uses the old Sysdig API URL format.
            Defaults to False using the public API URL format https://api.{region}.sysdig.com.
    Returns:
        sysdig_client.Configuration: A configured Sysdig client instance.
    Raises:
        ValueError: If the Sysdig host URL is not provided or is invalid.
    """
    # Check if the token and sysdig_host_url are provided, otherwise fetch from environment variables
    if not token:
        token = app_config.sysdig_secure_token()
    if not sysdig_host_url:
        sysdig_host_url = app_config.sysdig_endpoint()
    if not old_api:
        """
        Client expecting the public API URL in the format https://api.{region}.sysdig.com. We will check the following:
        - A valid Sysdig host URL is provided by matching the expected patterns with a regex.
        - If not, we will try to fetch the public API URL from the app config yaml 'sysdig.public_api_url'.
        - If neither is available, we will raise an error.
        """
        sysdig_host_url = _get_public_api_url(sysdig_host_url)
        if not sysdig_host_url:
            raise ValueError(
                "No valid Sysdig public API URL found. Please check your Sysdig host URL or"
                "explicitly set the public API URL in the app config 'sysdig.public_api_url'."
                "The expected format is https://api.{region}.sysdig.com."
            )

    configuration = sysdig_client.Configuration(
        access_token=token,
        host=sysdig_host_url,
    )
    return configuration


def _get_public_api_url(base_url: str) -> str:
    """
    Maps a Sysdig base URL to its corresponding public API URL.
    This function extracts the region from the base URL and constructs the public API URL in the format
    https://api.{region}.sysdig.com.

    If the base URL does not match any known patterns, it returns an empty string.

    Args:
        base_url: The base URL of the Sysdig API

    Returns:
        str: The public API URL in the format https://api.{region}.sysdig.com
    """

    patterns = [
        (r"^https://secure\.sysdig\.com$", lambda m: "us1"),
        (r"^https://([a-z]{2}\d)\.app\.sysdig\.com$", lambda m: m.group(1)),
        (r"^https://app\.([a-z]{2}\d)\.sysdig\.com$", lambda m: m.group(1)),
    ]

    for pattern, region_fn in patterns:
        match = re.match(pattern, base_url)
        if match:
            region = region_fn(match)
            return f"https://api.{region}.sysdig.com"

    log.warning("A not recognized Sysdig URL was provided, returning an empty string. This may lead to unexpected behavior.")
    return ""
