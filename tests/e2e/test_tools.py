from __future__ import annotations
import os
import json
import pytest
from typing import Callable, cast
from fastmcp.client import Client
from fastmcp.client.transports import StdioTransport

# Define a type for JSON-like objects to avoid using Any
JsonValue = str | int | float | bool | None | dict[str, "JsonValue"] | list["JsonValue"]
JsonObject = dict[str, JsonValue]


# E2E tests for the Sysdig MCP Server tools.
#
# This script is designed to run in a CI/CD environment and requires the following prerequisites:
# - Docker installed and running.
# - The `sysdig-cli-scanner` binary installed and available in the system's PATH.
# - The following environment variables set with valid Sysdig credentials:
#   - SYSDIG_MCP_API_SECURE_TOKEN
#   - SYSDIG_MCP_API_HOST
#
# The script will start the MCP server in a separate process, run a series of tests against it,
# and then shut it down. If any of the tests fail, the script will exit with a non-zero status code.


async def run_test(tool_name: str, tool_args: JsonObject, check: str | Callable[[JsonObject], None]):
    """
    Runs a test by starting the MCP server, sending a request to it, and checking its stdout.
    """
    transport = StdioTransport(
        "uv",
        ["run", "sysdig-mcp-server"],
        env=dict(os.environ, **{"SYSDIG_MCP_LOGLEVEL": "DEBUG"}),
    )
    client = Client(transport)

    async with client:
        result = await client.call_tool(tool_name, tool_args)

        # Extract text content from the result
        output = ""
        if result.content:
            for content_block in result.content:
                output += getattr(content_block, "text", "")

        print(f"--- STDOUT ---\n{output}")

        if isinstance(check, str):
            assert check in output
        elif callable(check):
            try:
                json_output = cast(JsonObject, json.loads(output))
                check(json_output)
            except json.JSONDecodeError:
                pytest.fail(f"Output is not a valid JSON: {output}")


@pytest.mark.e2e
async def test_cli_scanner_tool_vulnerability_scan():
    """
    Tests the CliScannerTool's vulnerability scan.
    """
    def assert_vulns(output: JsonObject):
        assert output["exit_code"] == 0
        output_str = output.get("output", "")
        assert isinstance(output_str, str)
        assert "vulnerabilities found" in output_str

    await run_test(
        "run_sysdig_cli_scanner",
        {"image": "ubuntu:18.04", "mode": "vulnerability", "standalone": True, "offline_analyser": True},
        assert_vulns,
    )


@pytest.mark.e2e
async def test_cli_scanner_tool_iac_scan():
    """
    Tests the CliScannerTool's IaC scan.
    """
    def assert_iac(output: JsonObject):
        assert output["exit_code"] == 0
        output_str = output.get("output", "")
        assert isinstance(output_str, str)
        assert "OK: no resources found" in output_str

    await run_test(
        "run_sysdig_cli_scanner",
        {"path_to_scan": "tests/e2e/data/", "mode": "iac"},
        assert_iac,
    )


@pytest.mark.e2e
async def test_events_feed_tools_list_runtime_events():
    """
    Tests the EventsFeedTools' list_runtime_events.
    """
    def assert_events(output: JsonObject):
        assert output["status_code"] == 200
        results = output.get("results")
        assert isinstance(results, dict)
        assert isinstance(results.get("data"), list)
        assert isinstance(results.get("page"), dict)

    await run_test("list_runtime_events", {"scope_hours": 1}, assert_events)


@pytest.mark.e2e
async def test_events_feed_tools_get_event_info():
    """
    Tests the EventsFeedTools' get_event_info by first getting a valid event ID.
    """
    event_id = None

    def get_event_id(output: JsonObject):
        nonlocal event_id
        if output.get("results", {}).get("data"):
            event_id = output["results"]["data"][0].get("id")

    await run_test("list_runtime_events", {"scope_hours": 24, "limit": 1}, get_event_id)

    if not event_id:
        pytest.skip("No runtime events in the last 24 hours to test get_event_info.")

    def assert_event_info(output: JsonObject):
        assert output["status_code"] == 200
        assert isinstance(output.get("results"), dict)
        assert output["results"].get("id") == event_id

    await run_test("get_event_info", {"event_id": event_id}, assert_event_info)


@pytest.mark.e2e
async def test_events_feed_tools_get_event_process_tree():
    """
    Tests the EventsFeedTools' get_event_process_tree by first getting a valid event ID.
    """
    event_id = None

    def get_event_id(output: JsonObject):
        nonlocal event_id
        if output.get("results", {}).get("data"):
            event_id = output["results"]["data"][0].get("id")

    await run_test("list_runtime_events", {"scope_hours": 24, "limit": 1}, get_event_id)

    if not event_id:
        pytest.skip("No runtime events in the last 24 hours to test get_event_process_tree.")

    def assert_process_tree(output: JsonObject):
        assert isinstance(output.get("branches"), dict)
        assert isinstance(output.get("tree"), dict)
        assert isinstance(output.get("metadata"), dict)

    await run_test("get_event_process_tree", {"event_id": event_id}, assert_process_tree)


@pytest.mark.e2e
async def test_sysql_tools_generate_and_run_sysql_query():
    """
    Tests the SysQLTools' generate_and_run_sysql.
    """
    def assert_sysql(output: JsonObject):
        assert output["status_code"] == 200
        results = output.get("results")
        assert isinstance(results, dict)
        assert isinstance(results.get("entities"), dict)
        assert isinstance(results.get("items"), list)

        metadata = output.get("metadata")
        assert isinstance(metadata, dict)

        metadata_kwargs = metadata.get("metadata_kwargs")
        assert isinstance(metadata_kwargs, dict)

        sysql = metadata_kwargs.get("sysql")
        assert isinstance(sysql, str)
        assert "MATCH CloudResource AFFECTED_BY Vulnerability" in sysql

    await run_test(
        "generate_and_run_sysql",
        {"question": "Match Cloud Resource affected by Critical Vulnerability"},
        assert_sysql,
    )


@pytest.mark.e2e
async def test_sysql_tools_run_sysql_query():
    """
    Tests the SysQLTools' run_sysql.
    """
    def assert_sysql(output: JsonObject):
        assert output["status_code"] == 200
        results = output.get("results")
        assert isinstance(results, dict)
        assert isinstance(results.get("entities"), dict)
        assert isinstance(results.get("items"), list)

        metadata = output.get("metadata")
        assert isinstance(metadata, dict)

    await run_test(
        "run_sysql",
        {"sysql_query": "MATCH CloudResource AFFECTED_BY Vulnerability"},
        assert_sysql,
    )
