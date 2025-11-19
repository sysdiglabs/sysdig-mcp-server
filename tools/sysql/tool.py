"""
Sysdig SysQL Tool
This tool provides functionality to interact with Sysdig, allowing users to
generate SysQL queries based on natural language questions and execute them against the Sysdig API.
"""

import logging
import time

from fastmcp.server.context import Context
from fastmcp.exceptions import ToolError
from utils.sysdig.legacy_sysdig_api import LegacySysdigApi
from utils.app_config import AppConfig
from utils.query_helpers import create_standard_response


class SysQLTools:
    """
    A class to encapsulate the tools for interacting with Sysdig SysQL.
    This class provides methods to generate SysQL queries based on natural
    language questions and execute them against the Sysdig API.
    """

    def __init__(self, app_config: AppConfig):
        self.app_config = app_config
        self.log = logging.getLogger(__name__)

    async def tool_generate_and_run_sysql(self, ctx: Context, question: str) -> dict:
        """
        Queries Sysdig with a natural language question, retrieves a SysQL query,
        executes it against the Sysdig API, and returns the results.

        Args:
            ctx (Context): A context object containing configuration information.
            question (str): A natural language question to send to Sysdig.

        Returns:
            dict: A dictionary containing the results of the SysQL query execution and the query text.

        Raises:
            ToolError: If the SysQL query generation or execution fails.

        Examples:
            # tool_generate_and_run_sysql(question="Match Cloud Resource affected by Critical Vulnerability")
            # tool_generate_and_run_sysql(question="Match Kubernetes Workload affected by Critical Vulnerability")
            # tool_generate_and_run_sysql(
            #     question="Match AWS EC2 Instance that violates control 'EC2 - Instances should use IMDSv2'"
            # )
        """
        # 1) Generate SysQL query
        try:
            start_time = time.time()
            api_instances: dict = ctx.get_state("api_instances")
            legacy_api_client: LegacySysdigApi = api_instances.get("legacy_sysdig_api")
            if not legacy_api_client:
                self.log.error("LegacySysdigApi instance not found")
                raise ToolError("LegacySysdigApi instance not found")

            sysql_response = await legacy_api_client.generate_sysql_query(question)
            if sysql_response.status > 299:
                raise ToolError(f"Sysdig returned an error: {sysql_response.status} - {sysql_response.data}")
        except ToolError as e:
            self.log.error(f"Failed to generate SysQL query: {e}")
            raise e
        json_resp = sysql_response.json() if sysql_response.data else {}
        sysql_query: str = json_resp.get("text", "")
        if not sysql_query:
            return {"error": "Sysdig did not return a query"}

        # 2) Execute generated SysQL query
        try:
            self.log.debug(f"Executing SysQL query: {sysql_query}")
            results = legacy_api_client.execute_sysql_query(sysql_query)
            execution_time = (time.time() - start_time) * 1000
            self.log.debug(f"SysQL query executed in {execution_time} ms")
            response = create_standard_response(
                results=results, execution_time_ms=execution_time, metadata_kwargs={"question": question, "sysql": sysql_query}
            )

            return response
        except ToolError as e:
            self.log.error(f"Failed to execute SysQL query: {e}")
            raise e

    async def tool_run_sysql(self, ctx: Context, sysql_query: str) -> dict:
        """
        Executes a pre-written SysQL query directly against the Sysdig API and returns the results.

        Use this tool ONLY when the user provides an explicit SysQL query. Do not improvise or
        generate queries. For natural language questions, use generate_and_run_sysql instead.

        Args:
            ctx (Context): A context object containing configuration information.
            sysql_query (str): A valid SysQL query string to execute directly.

        Returns:
            dict: A dictionary containing the results of the SysQL query execution with metadata.

        Raises:
            ToolError: If the SysQL query execution fails or if the query is invalid.

        Examples:
            # tool_run_sysql(sysql_query="MATCH Vulnerability WHERE severity = 'Critical' LIMIT 10")
            # tool_run_sysql(sysql_query="MATCH KubeWorkload AS k AFFECTED_BY Vulnerability WHERE k.namespace = 'production'")
            # tool_run_sysql(sysql_query="MATCH CloudResource WHERE type = 'aws_s3_bucket' RETURN *")
            # tool_run_sysql(sysql_query="MATCH Vulnerability AS v WHERE v.name = 'CVE-2024-1234' RETURN v")
        """
        # Start timer
        start_time = time.time()
        # Get API instance
        api_instances: dict = ctx.get_state("api_instances")
        legacy_api_client: LegacySysdigApi = api_instances.get("legacy_sysdig_api")
        if not legacy_api_client:
            self.log.error("LegacySysdigApi instance not found")
            raise ToolError("LegacySysdigApi instance not found")

        if not sysql_query:
            raise ToolError("No SysQL query provided. Please provide a valid SysQL query string.")

        # Ensure the query ends with a semicolon
        if not sysql_query.strip().endswith(";"):
            sysql_query += ";"

        try:
            self.log.debug(f"Executing SysQL query: {sysql_query}")
            results = legacy_api_client.execute_sysql_query(sysql_query)
            execution_time = (time.time() - start_time) * 1000
            self.log.debug(f"SysQL query executed in {execution_time} ms")
            response = create_standard_response(
                results=results, execution_time_ms=execution_time, metadata_kwargs={"sysql_query": sysql_query}
            )

            return response
        except ToolError as e:
            self.log.error(f"Failed to execute SysQL query: {e}")
            raise e
