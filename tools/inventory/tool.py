"""
This module provides tools for interacting with the Sysdig Secure Inventory API.
"""

import logging
import os
import time
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
logging.basicConfig(format="%(asctime)s-%(process)d-%(levelname)s- %(message)s", level=os.environ.get("LOGLEVEL", "ERROR"))
log = logging.getLogger(__name__)

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
        Returns:
            InventoryApi: An instance of the InventoryApi client.
        Raises:
            ValueError: If the SYSDIG_SECURE_TOKEN environment variable is not set.
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
                description=(
                    """
                    Use the filter-query-language to filter the results.
                    
                    List of supported fields:
                    - field: accountName
                      Description:  The account name that will be included in the results.
                    - field: accountId
                      Description:  The account id that will be included in the results.
                    - field: cluster
                      Description:  The kubernetes cluster that will be included in the results.
                    - field: externalDNS
                      Description:  The external DNS that will be included in the results.
                    - field: distribution
                      Description:  The kubernetes distribution that will be included in the results.
                    - field: integrationName
                      Description:  The name of the integration an IaC resource belongs to.
                    - field: labels
                      Description:  The resource labels that will be included in the results.
                    - field: location
                      Description:  The web address of an IaC Manifest.
                    - field: name
                      Description:  The names that will be included in the results.
                    - field: namespace
                      Description:  The namespace that will be included in the results.
                    - field: nodeType
                      Description:  The nodeType that will be included in the results.
                    - field: osName
                      Description:  The operating system that will be included in the results.
                    - field: osImage
                      Description:  The operating system image that will be included in the results.
                    - field: organization
                      Description:  The organization that will be included in the results.
                    - field: platform
                      Description:  The platform that will be included in the results.
                    - field: control.accepted
                      Description:  Include (or Exclude) only resources with accepted results.
                                    Supported operators: exists and not exists.
                    - field: policy
                      Description:  Include resources that applied the selected policies.
                                    Supported operators: in, not in, exists, not exists.
                    - field: control.severity
                      Description:  Include resources that have violated risks in the selected severities.
                                    Supported operators: in, not in.
                    - field: control.failed
                      Description:  Include resources that have violated the selected risks.
                                    Supported operators: in, not in, exists, not exists.
                    - field: policy.failed
                      Description:  Include resources that failed the selected policies.
                                    Supported operators: in, not in, exists, not exists.
                    - field: policy.passed
                      Description:  Include resources that passed the selected policies.
                                    Supported operators: in, not in, exists, not exists.
                    - field: projectName
                      Description:  The project name that will be included in the results.
                    - field: projectId
                      Description:  The project id that will be included in the results.
                    - field: region
                      Description:  The regions that will be included in the results.
                    - field: repository
                      Description:  The Repository an IaC resource belongs to.
                    - field: resourceOrigin
                      Description:  Origin of the resource. Supported values: Code, Deployed.
                    - field: type
                      Description:  The resource types that will be included in the results.
                    - field: subscriptionName
                      Description:  The Azure subscription name that will be included in the results.
                    - field: subscriptionId
                      Description:  The Azure subscription id that will be included in the results.
                    - field: sourceType
                      Description:  The source type of an IaC resource.
                                    Supported values: YAML, Kustomize, Terraform, Helm.
                    - field: version
                      Description:  OCP Cluster versions that will be included in the results.
                    - field: zone
                      Description:  The zones that will be included in the results.
                    - field: category
                      Description:  The category that will be included in the results.
                                    Supported operators: in, not in.
                    - field: isExposed
                      Description:  Specifies whether the resource to return is exposed to the internet.
                                    Supported operators: exists and not exists.
                    - field: validatedExposure
                      Description:  Specifies whether the resource to return is exposed to the internet and could be reach
                                    by our network exposure validator. Supported operators: exists and not exists.
                    - field: arn
                      Description:  The AWS ARN of the resource.
                    - field: resourceId
                      Description:  The Azure or GCP Resource Identifier of the resource.
                    - field: container.name
                      Description:  Filters the resource by a container.
                    - field: architecture
                      Description:  Image architecture.
                    - field: baseOS
                      Description:  Image Base OS.
                    - field: digest
                      Description:  Image Digest.
                    - field: imageId
                      Description:  Image Id.
                    - field: os
                      Description:  Image OS.
                    - field: container.imageName
                      Description:  Image Pullstring.
                    - field: image.registry
                      Description:  Image Registry.
                    - field: image.tag
                      Description:  Image tag.
                    - field: package.inUse
                      Description:  Package in use filter. Supported operators: exists and not exists.
                    - field: package.info
                      Description:  Filters by a package using the format [packge name] - field: [version].
                    - field: package.path
                      Description:  Filters by package path.
                    - field: package.type
                      Description:  Package type.
                    - field: vuln.cvssScore
                      Description:  Filter by vulnerability CVSS. Supported operators: = and >=.
                    - field: vuln.hasExploit
                      Description:  Filters resources by the existence of vulnerabilities with exploits.
                      Supported operators: exists and not exists.
                    - field: vuln.hasFix
                      Description:  Filters resources by the existence of vulnerabilities with fixes.
                                    Supported operators: exists and not exists.
                    - field: vuln.name
                      Description:  Filter by vulnerability name.
                    - field: vuln.severity
                      Description:  Filter by vulnerability severity. Supported operators: in, not in, exists and not exists.
                    - field: machineImage
                      Description:  Filter by host machine image.
                """
                ),
                examples=[
                    'zone in ("zone1") and machineImage = "ami-0b22b359fdfabe1b5"',
                    '(projectId = "1235495521" or projectId = "987654321") and vuln.severity in ("Critical")',
                    'vuln.name in ("CVE-2023-0049")',
                    'vuln.cvssScore >= "3"',
                    'container.name in ("sysdig-container") and not labels exists',
                    'imageId in ("sha256:3768ff6176e29a35ce1354622977a1e5c013045cbc4f30754ef3459218be8ac")',
                ],
            ),
        ] = 'platform in ("GCP", "AWS", "Azure", "Kubernetes")',
        page_number: Annotated[int, Field(ge=1, description="Page number for pagination (1-based index)")] = 1,
        page_size: Annotated[int, Field(ge=1, le=100, default=20, description="Number of items per page")] = 20,
        with_enrich_containers: Annotated[
            bool, Field(description="Whether to include enriched container details", example=True)
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
        """
        try:
            inventory_api = self.init_client(config_tags=ctx.fastmcp.tags)
            start_time = time.time()

            api_response = inventory_api.get_resources_without_preload_content(
                filter=filter_exp, page_number=page_number, page_size=page_size, with_enriched_containers=with_enrich_containers
            )

            execution_time = (time.time() - start_time) * 1000

            response = create_standard_response(results=api_response, execution_time_ms=execution_time)

            return response
        except ApiException as e:
            logging.error("Exception when calling InventoryApi->get_resources: %s\n" % e)
            raise e

    def tool_get_resource(
        self,
        ctx: Context,
        resource_hash: Annotated[str, Field(description="The unique hash of the inventory resource to retrieve.")],
    ) -> dict:
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

            response = create_standard_response(results=api_response, execution_time_ms=execution_time)

            return response
        except ApiException as e:
            log.error(f"Exception when calling InventoryApi->get_resource: {e}")
            raise e
