'''
This module provides tools for interacting with the Sysdig Secure Inventory API.
'''
import logging, os, time
from typing import Annotated
from pydantic import Field
from fastmcp.server.dependencies import get_http_request
from fastmcp import Context
from starlette.requests import Request
from sysdig_client import ApiException
from sysdig_client.api import InventoryApi
from utils.sysdig.client_config import get_configuration
from utils.app_config import get_app_config
from utils.sysdig.api import initialize_api_client
from utils.query_helpers import create_standard_response

# Configure logging
log = logging.getLogger(__name__)
logging.basicConfig(
    format='%(asctime)s-%(process)d-%(levelname)s- %(message)s', 
    level=os.environ.get("LOGLEVEL", "ERROR")
)

# Load app config (expects keys: mcp.host, mcp.port, mcp.transport)
app_config = get_app_config()

class InventoryTools:
    """
    A class to encapsulate the tools for interacting with the Sysdig Secure Inventory API.
    This class provides methods to list resources and retrieve a single resource by its hash.
    """

    def init_client(self, config_tags: set[str]) -> InventoryApi:
        """
        Initializes the InventoryApi client from the request state.
        If the request does not have the API client initialized, it will create a new instance
        using the Sysdig Secure token and host from the environment variables.
        Args:
            config_tags (set[str]): The tags associated with the MCP server configuration, used to determine the transport mode.
        Raises:
            ValueError: If the SYSDIG_SECURE_TOKEN environment variable is not set.
            RuntimeError: If the API client cannot be initialized from the http request state. Hence running in STDIO mode.
        """
        secure_events_api: InventoryApi = None
        if "streamable-http" in config_tags:
            # Try to get the HTTP request
            log.debug("Attempting to get the HTTP request to initialize the Sysdig API client.")
            request: Request = get_http_request()
            secure_events_api = request.state.api_instances["inventory"]
        else:
            # If running in STDIO mode, we need to initialize the API client from environment variables
            log.debug("Running in STDIO mode, initializing the Sysdig API client from environment variables.")
            SYSDIG_SECURE_TOKEN = os.environ.get("SYSDIG_SECURE_TOKEN", "")
            if not SYSDIG_SECURE_TOKEN:
                raise ValueError("Can not initialize client, SYSDIG_SECURE_TOKEN environment variable is not set.")
            SYSDIG_HOST = os.environ.get("SYSDIG_HOST", app_config["sysdig"]["host"])
            cfg = get_configuration(SYSDIG_SECURE_TOKEN, SYSDIG_HOST)
            api_client = initialize_api_client(cfg)
            secure_events_api = InventoryApi(api_client)
        return secure_events_api

    def tool_list_resources(
        self,
        ctx: Context,
        filter_exp: Annotated[
            str,
            Field(
                description='Sysdig Secure filter expression for inventory resources, base filter: platform in ("GCP", "AWS", "Azure", "Kubernetes"), Examples: not isExposed exists; category in ("IAM") and isExposed exists; category in ("IAM","Audit & Monitoring")'
            )
        ] = 'platform in ("GCP", "AWS", "Azure", "Kubernetes")',
        page_number: Annotated[
            int,
            Field(
                ge=1,
                description="Page number for pagination (1-based index)"
            )
        ] = 1,
        page_size: Annotated[
            int,
            Field(
                ge=1,
                le=100,
                default=20,
                description="Number of items per page"
            )
        ] = 20,
        with_enrich_containers: Annotated[
            bool,
            Field(
                description="Whether to include enriched container details",
                example=True
            )
        ] = True,
    ) -> dict:
        """
        List inventory items based on a filter expression, with optional pagination.

        Args:
            filter_exp (str): Sysdig Secure query filter expression.
                Examples:
                - not isExposed exists
                - category in ("IAM") and isExposed exists
                - category in ("IAM","Audit & Monitoring")
            page_number (int): Page number for pagination (1-based).
            page_size (int): Number of items per page.
            with_enrich_containers (bool): Include enriched container information.

        Returns:
            InventoryResourceResponse: The API response containing inventory items.
        Raises:
            ApiException: If the API call to retrieve resources fails.
        """
        try:
            inventory_api = self.init_client(config_tags=ctx.fastmcp.tags)
            start_time = time.time()

            api_response = inventory_api.get_resources_without_preload_content(
                filter=filter_exp,
                page_number=page_number,
                page_size=page_size,
                with_enriched_containers=with_enrich_containers
            )
            
            execution_time = (time.time() - start_time) * 1000

            response = create_standard_response(
                results=api_response,
                execution_time_ms=execution_time
            )

            return response
        except ApiException as e:
            logging.error("Exception when calling InventoryApi->get_resources: %s\n" % e)
            raise e

    def tool_get_resource(
        self,
        ctx: Context,
        resource_hash: Annotated[
            str,
            Field(description="The unique hash of the inventory resource to retrieve.")
        ]
    ) ->  dict:
        """
        Fetch a specific inventory resource by hash.

        Args:
            resource_hash (str): The hash identifier of the resource.

        Returns:
            InventoryResourceExtended: The detailed resource object.
            Or a dict containing an error message if the call fails.
        """
        try:
            inventory_api = self.init_client(config_tags=ctx.fastmcp.tags)
            start_time = time.time()

            api_response = inventory_api.get_resource_without_preload_content(hash=resource_hash)
            execution_time = (time.time() - start_time) * 1000

            response = create_standard_response(
                results=api_response,
                execution_time_ms=execution_time
            )

            return response
        except ApiException as e:
            log.error(f"Exception when calling InventoryApi->get_resource: {e}")
            raise e
