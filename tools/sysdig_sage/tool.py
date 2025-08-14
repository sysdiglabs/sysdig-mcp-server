"""
Sysdig Sage Tool
This tool provides functionality to interact with Sysdig Sage, allowing users to
generate SysQL queries based on natural language questions and execute them against the Sysdig API.
"""

import logging
import os
import time
from fastmcp.exceptions import ToolError
from fastmcp.server.context import Context
from utils.sysdig.legacy_sysdig_api import LegacySysdigApi
from starlette.requests import Request
from fastmcp.server.dependencies import get_http_request
from utils.sysdig.client_config import get_configuration
from utils.app_config import get_app_config
from utils.sysdig.api import initialize_api_client
from utils.query_helpers import create_standard_response

logging.basicConfig(format="%(asctime)s-%(process)d-%(levelname)s- %(message)s", level=os.environ.get("LOGLEVEL", "ERROR"))
log = logging.getLogger(__name__)

app_config = get_app_config()


class SageTools:
    """
    A class to encapsulate the tools for interacting with Sysdig Sage.
    This class provides methods to generate SysQL queries based on natural
    language questions and execute them against the Sysdig API.
    """

    def init_client(self, transport: str) -> LegacySysdigApi:
        """
        Initializes the LegacySysdigApi client from the request state.
        If the request does not have the API client initialized, it will create a new instance
        using the Sysdig Secure token and host from the environment variables.
        Returns:
            LegacySysdigApi: An instance of the LegacySysdigApi client.
        """
        legacy_sysdig_api: LegacySysdigApi = None
        if transport in ["streamable-http", "sse"]:
            # Try to get the HTTP request
            log.debug("Attempting to get the HTTP request to initialize the Sysdig API client.")
            request: Request = get_http_request()
            legacy_sysdig_api = request.state.api_instances["legacy_sysdig_api"]
        else:
            # If running in STDIO mode, we need to initialize the API client from environment variables
            log.debug("Trying to init the Sysdig API client from environment variables.")
            cfg = get_configuration(old_api=True)
            api_client = initialize_api_client(cfg)
            legacy_sysdig_api = LegacySysdigApi(api_client)
        return legacy_sysdig_api

    async def tool_sage_to_sysql(self, ctx: Context, question: str) -> dict:
        """
        Queries Sysdig Sage with a natural language question, retrieves a SysQL query,
        executes it against the Sysdig API, and returns the results.

        Args:
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
            legacy_sysdig_api = self.init_client(ctx.get_state("transport_method"))
            sysql_response = await legacy_sysdig_api.generate_sysql_query(question)
            if sysql_response.status > 299:
                raise ToolError(f"Sysdig Sage returned an error: {sysql_response.status} - {sysql_response.data}")
        except ToolError as e:
            log.error(f"Failed to generate SysQL query: {e}")
            raise e
        json_resp = sysql_response.json() if sysql_response.data else {}
        syslq_query: str = json_resp.get("text", "")
        if not syslq_query:
            return {"error": "Sysdig Sage did not return a query"}

        # 2) Execute generated SysQL query
        try:
            log.debug(f"Executing SysQL query: {syslq_query}")
            results = legacy_sysdig_api.execute_sysql_query(syslq_query)
            execution_time = (time.time() - start_time) * 1000
            log.debug(f"SysQL query executed in {execution_time} ms")
            response = create_standard_response(
                results=results, execution_time_ms=execution_time, metadata_kwargs={"question": question, "sysql": syslq_query}
            )

            return response
        except ToolError as e:
            log.error(f"Failed to execute SysQL query: {e}")
            raise e
