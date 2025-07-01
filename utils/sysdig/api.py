"""
This module provides functions to initialize and manage Sysdig API clients.
"""

from sysdig_client import ApiClient, SecureEventsApi, VulnerabilityManagementApi, InventoryApi
from utils.sysdig.old_sysdig_api import OldSysdigApi
from sysdig_client.configuration import Configuration


def get_api_client(config: Configuration) -> ApiClient:
    """
    Creates a unique instance of ApiClient with the provided configuration.

    Args:
        config (Configuration): The Sysdig client configuration containing the access token and host URL.
    Returns:
        ApiClient: An instance of ApiClient configured with the provided settings.
    """
    api_client_instance = ApiClient(config)
    return api_client_instance


def initialize_api_client(config: Configuration) -> ApiClient:
    """
    Initializes the Sysdig API client with the provided token and host.
    This function creates a new ApiClient instance and returns a dictionary of API instances

    Args:
        config (Configuration): The Sysdig client configuration containing the access token and host URL.
    Returns:
        dict: A dictionary containing instances of multiple Sysdig API classes.
    """
    api_client = get_api_client(config)
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
        "old_sysdig_api": OldSysdigApi(api_client),
    }
