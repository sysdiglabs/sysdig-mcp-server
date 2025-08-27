"""
Sysdig Sage Tool
This tool provides functionality to interact with Sysdig Sage, allowing users to
generate SysQL queries based on natural language questions and execute them against the Sysdig API.
"""

import logging
import time

from fastmcp.server.context import Context
from fastmcp.exceptions import ToolError
from utils.sysdig.legacy_sysdig_api import LegacySysdigApi
from utils.app_config import AppConfig
from utils.query_helpers import create_standard_response


class SageTools:
    """
    A class to encapsulate the tools for interacting with Sysdig Sage.
    This class provides methods to generate SysQL queries based on natural
    language questions and execute them against the Sysdig API.
    """

    def __init__(self, app_config: AppConfig):
        self.app_config = app_config
        self.log = logging.getLogger(__name__)

    async def tool_sage_to_sysql(self, ctx: Context, question: str) -> dict:
        """
        Queries Sysdig Sage with a natural language question, retrieves a SysQL query,
        executes it against the Sysdig API, and returns the results.

        Args:
            ctx (Context): A context object containing configuration information.
            question (str): A natural language question to send to Sage.

        Returns:
            dict: A dictionary containing the results of the SysQL query execution and the query text.

        Raises:
            ToolError: If the SysQL query generation or execution fails.

        Examples:
            # tool_sage_to_sysql(question="Match Cloud Resource affected by Critical Vulnerability")
            # tool_sage_to_sysql(question="Match Kubernetes Workload affected by Critical Vulnerability")
            # tool_sage_to_sysql(question="Match AWS EC2 Instance that violates control 'EC2 - Instances should use IMDSv2'")
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
                raise ToolError(f"Sysdig Sage returned an error: {sysql_response.status} - {sysql_response.data}")
        except ToolError as e:
            self.log.error(f"Failed to generate SysQL query: {e}")
            raise e
        json_resp = sysql_response.json() if sysql_response.data else {}
        sysql_query: str = json_resp.get("text", "")
        if not sysql_query:
            return {"error": "Sysdig Sage did not return a query"}

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
