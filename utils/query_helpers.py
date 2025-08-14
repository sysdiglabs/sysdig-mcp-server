"""
Utility functions for handling API response for the MCP server responses.
"""

import datetime
from sysdig_client.rest import RESTResponseType, ApiException


def create_standard_response(results: RESTResponseType, execution_time_ms: str, **metadata_kwargs) -> dict:
    """
    Creates a standard response format for API calls.
    Args:
        results (RESTResponseType): The results from the API call.
        execution_time_ms (str): The execution time in milliseconds.
        **metadata_kwargs: Additional metadata to include in the response.

    Returns:
        dict: A dictionary containing the results and metadata.

    Raises:
        ApiException: If the API call returned an error status code.
    """
    response: dict = {}
    if results.status > 299:
        raise ApiException(
            status=results.status,
            reason=results.reason,
            data=results.data,
        )
    else:
        response = results.json() if results.data else {}

    return {
        "results": response,
        "metadata": {
            "execution_time_ms": execution_time_ms,
            "timestamp": datetime.datetime.now(datetime.UTC).isoformat().replace("+00:00", "Z"),
            **metadata_kwargs,
        },
        "status_code": results.status,
    }
