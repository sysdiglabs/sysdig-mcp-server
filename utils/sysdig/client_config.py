"""
This module provides a function to configure the Sysdig client based on a configuration.
"""

import sysdig_client
import os
import logging
import re

# Set up logging
log = logging.getLogger(__name__)
logging.basicConfig(format="%(asctime)s-%(process)d-%(levelname)s- %(message)s", level=os.environ.get("LOGLEVEL", "ERROR"))


# Lazy-load the Sysdig client configuration
def get_configuration(token: str, sysdig_host_url: str, old_api: bool = False) -> sysdig_client.Configuration:
    """
    Returns a configured Sysdig client using environment variables.

    Args:
        token (str): The Sysdig Secure token.
        sysdig_host_url (str): The base URL of the Sysdig API.
        old_api (bool): If True, uses the old Sysdig API URL format. Defaults to False.
    Returns:
        sysdig_client.Configuration: A configured Sysdig client instance.
    """
    if not old_api:
        sysdig_host_url = _get_public_api_url(sysdig_host_url)
        log.info(f"Using public API URL: {sysdig_host_url}")

    configuration = sysdig_client.Configuration(
        access_token=token,
        host=sysdig_host_url,
    )
    return configuration


def _get_public_api_url(base_url: str) -> str:
    """
    Get the public API URL from the base URL.

    Args:
        base_url: The base URL of the Sysdig API

    Returns:
        str: The public API URL in the format https://api.<region>.sysdig.com
    """
    # Regex to capture the region pattern (like us2, us3, au1, etc.)
    # This assumes the region is a subdomain that starts with 2 lowercase letters and ends with a digit
    pattern = re.search(r"https://(?:(?P<region1>[a-z]{2}\d)\.app|app\.(?P<region2>[a-z]{2}\d))\.sysdig\.com", base_url)
    if pattern:
        region = pattern.group(1)  # Extract the region
        return f"https://api.{region}.sysdig.com"
    else:
        # Edge case for the secure API URL that is us1
        return "https://api.us1.sysdig.com"
