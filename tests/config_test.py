"""
Testing overall configuration of the MCP server.
"""


def test_api_url_format() -> None:
    """
    Test that the API URL is formatted correctly.
    This test checks if the public API URL is constructed properly
    when the old API format is not used.
    """
    from utils.sysdig.client_config import _get_public_api_url

    # URL regions refer to https://docs.sysdig.com/en/administration/saas-regions-and-ip-ranges/
    region_urls = {
        "us1": {"url": "https://secure.sysdig.com", "public_url": "https://api.us1.sysdig.com"},
        "us2": {"url": "https://us2.app.sysdig.com", "public_url": "https://api.us2.sysdig.com"},
        "eu1": {"url": "https://eu1.app.sysdig.com", "public_url": "https://api.eu1.sysdig.com"},
        "au1": {"url": "https://app.au1.sysdig.com", "public_url": "https://api.au1.sysdig.com"},
        "me2": {"url": "https://app.me2.sysdig.com", "public_url": "https://api.me2.sysdig.com"},
        # Edge case that does not follow the standard pattern it should return the same passed URL
        "edge": {"url": "https://edge.something.com", "public_url": "https://edge.something.com"},
    }

    # Check if the public API URL is formatted correctly
    public_api_us1 = _get_public_api_url(region_urls["us1"]["url"])
    public_api_us2 = _get_public_api_url(region_urls["us2"]["url"])
    public_api_eu1 = _get_public_api_url(region_urls["eu1"]["url"])
    public_api_au1 = _get_public_api_url(region_urls["au1"]["url"])
    public_api_me2 = _get_public_api_url(region_urls["me2"]["url"])
    public_api_edge = _get_public_api_url(region_urls["edge"]["url"])

    assert public_api_us1 == region_urls["us1"]["public_url"], (
        f"Expected {region_urls['us1']['public_url']}, got {public_api_us1}"
    )
    assert public_api_us2 == region_urls["us2"]["public_url"], (
        f"Expected {region_urls['us2']['public_url']}, got {public_api_us2}"
    )
    assert public_api_eu1 == region_urls["eu1"]["public_url"], (
        f"Expected {region_urls['eu1']['public_url']}, got {public_api_eu1}"
    )
    assert public_api_au1 == region_urls["au1"]["public_url"], (
        f"Expected {region_urls['au1']['public_url']}, got {public_api_au1}"
    )
    assert public_api_me2 == region_urls["me2"]["public_url"], (
        f"Expected {region_urls['me2']['public_url']}, got {public_api_me2}"
    )
    assert public_api_edge == region_urls["edge"]["public_url"], (
        f"Expected {region_urls['edge']['public_url']}, got {public_api_edge}"
    )
    print("All public API URLs are formatted correctly.")
