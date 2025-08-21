"""
Temporary wrapper for Legacy Sysdig API.
Will be replaced with a proper implementation in the future
"""

import urllib.parse
from sysdig_client.rest import RESTResponseType
from sysdig_client import ApiClient


class LegacySysdigApi:
    """
    [Deprecated]
    Wrapper for Legacy Sysdig API.
    """

    def __init__(self, api_client: ApiClient):
        # cfg.host should be for example "https://us2.app.sysdig.com"
        self.api_client = api_client
        self.base = f"{self.api_client.configuration._base_path}/api"
        self.headers = {
            "Authorization": f"Bearer {self.api_client.configuration.access_token}",
            "Content-Type": "application/json",
        }

    async def generate_sysql_query(self, question: str) -> RESTResponseType:
        """
        Sends a natural language question to Sysdig Sage and returns the generated SysQL query text.

        Args:
            question (str): The natural language question to ask Sysdig Sage.
        Returns:
            RESTResponseType: The response from the Sysdig Sage API containing the generated SysQL query text.
        """
        url = f"{self.base}/sage/sysql/generate?question={urllib.parse.quote(question)}"
        resp = self.api_client.call_api("GET", url, header_params=self.headers)
        return resp.response

    def execute_sysql_query(self, sysql: str) -> RESTResponseType:
        """
        Executes a SysQL query and returns the JSON response.

        Args:
            sysql (str): The SysQL query to execute.
        Returns:
            RESTResponseType: The response from the Sysdig API containing the results of the SysQL
        """
        url = f"{self.base}/sysql/v2/query?q={urllib.parse.quote(sysql)}"
        resp = self.api_client.call_api("GET", url, header_params=self.headers)
        return resp.response

    def request_process_tree_branches(self, process_id: str) -> RESTResponseType:
        """
        Requests the process tree branches for a given process ID.

        Args:
            process_id (str): The ID of the process to retrieve branches for.

        Returns:
            dict: The JSON-decoded response containing the process tree branches.
        """
        url = f"{self.base}/process-tree/v1/process-branches/{process_id}"
        resp = self.api_client.call_api("GET", url, header_params=self.headers)
        return resp.response

    def request_process_tree_trees(self, process_id: str) -> RESTResponseType:
        """
        Requests the process tree for a given process ID.

        Args:
            process_id (str): The ID of the process to retrieve the tree for.

        Returns:

        """
        url = f"{self.base}/process-tree/v1/process-trees/{process_id}"
        resp = self.api_client.call_api("GET", url, header_params=self.headers)
        return resp.response

    def get_me_permissions(self) -> RESTResponseType:
        """
        Retrieves the permissions for the current user.

        Returns:
            RESTResponseType: The response from the Sysdig API containing the user's permissions.
        """
        url = f"{self.base}/users/me/permissions"
        resp = self.api_client.call_api("GET", url, header_params=self.headers)
        return resp.response
