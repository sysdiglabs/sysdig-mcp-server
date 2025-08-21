"""
Utility functions for handling API response for the MCP server responses.
"""

from datetime import datetime
from sysdig_client.rest import RESTResponseType, ApiException
import json
import logging

log = logging.getLogger(__name__)


def _parse_response_to_obj(results: RESTResponseType | dict | list | str | bytes) -> dict | list:
    """Best-effort conversion of various response types into a Python object.
    Returns {} on empty/non-JSON bodies.
    """
    # Already a Python structure
    if results is None:
        return {}
    if isinstance(results, (dict, list)):
        return results

    # `requests.Response`-like: has .json() / .text
    if hasattr(results, "json") and hasattr(results, "text"):
        try:
            return results.json()
        except Exception:
            txt = getattr(results, "text", "") or ""
            txt = txt.strip()
            if not txt:
                return {}
            try:
                return json.loads(txt)
            except Exception:
                log.debug("create_standard_response: non-JSON text: %r", txt[:200])
                return {}

    # urllib3.HTTPResponse-like: has .data (bytes)
    if hasattr(results, "data"):
        data = getattr(results, "data", b"") or b""
        if not data:
            return {}
        try:
            return json.loads(data.decode("utf-8"))
        except Exception:
            log.debug("create_standard_response: non-JSON bytes: %r", data[:200])
            return {}

    # Pydantic v2 BaseModel
    if hasattr(results, "model_dump"):
        try:
            return results.model_dump()
        except Exception:
            return {}

    # Raw JSON string/bytes
    if isinstance(results, (bytes, str)):
        s = results.decode("utf-8") if isinstance(results, bytes) else results
        s = s.strip()
        if not s:
            return {}
        try:
            return json.loads(s)
        except Exception:
            log.debug("create_standard_response: raw string not JSON: %r", s[:200])
            return {}

    # Fallback
    return {}


def create_standard_response(results: RESTResponseType, execution_time_ms: float, **metadata_kwargs) -> dict:
    """
    Creates a standard response format for API calls. Tolerates empty/non-JSON bodies.
    Raises ApiException if the HTTP status is >= 300 (when available).
    """
    status = getattr(results, "status", 200)
    reason = getattr(results, "reason", "")

    # Propagate API errors if we have status
    if hasattr(results, "status") and status > 299:
        raise ApiException(
            status=status,
            reason=reason,
            data=getattr(results, "data", None),
        )

    parsed = _parse_response_to_obj(results)

    return {
        "results": parsed,
        "metadata": {
            "execution_time_ms": execution_time_ms,
            "timestamp": datetime.utcnow().isoformat() + "Z",
            **metadata_kwargs,
        },
        "status_code": status,
    }
