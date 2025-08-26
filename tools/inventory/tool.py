"""
This module provides tools for interacting with the Sysdig Secure Inventory API.
"""

import logging
import time
from typing import Annotated

from fastmcp import Context
from pydantic import Field
from fastmcp.exceptions import ToolError
from sysdig_client.api import InventoryApi
from utils.app_config import AppConfig
from utils.query_helpers import create_standard_response


class InventoryTools:
    """
    A class to encapsulate the tools for interacting with the Sysdig Secure Inventory API.
    This class provides methods to list resources and retrieve a single resource by its hash.
    """
    def __init__(self, app_config: AppConfig):
        self.app_config = app_config
        # Configure logging
        self.log = logging.getLogger(__name__)

    def tool_list_resources(
        self,
        ctx: Context,
        filter_exp: Annotated[
            str,
            Field(
                description=(
                    """
                    Sysdig Secure query filter expression to filter inventory resources.

                    Use the resource://filter-query-language to get the expected filter expression format.
               
                    List of supported fields:
                    - accountName
                    - accountId
                    - cluster
                    - externalDNS
                    - distribution
                    - integrationName
                    - labels
                    - location
                    - name
                    - namespace
                    - nodeType
                    - osName
                    - osImage
                    - organization
                    - platform
                    - control.accepted
                    - policy
                    - control.severity
                    - control.failed
                    - policy.failed
                    - policy.passed
                    - projectName
                    - projectId
                    - region
                    - repository
                    - resourceOrigin
                    - type
                    - subscriptionName
                    - subscriptionId
                    - sourceType
                    - version
                    - zone
                    - category
                    - isExposed
                    - validatedExposure
                    - arn
                    - resourceId
                    - container.name
                    - architecture
                    - baseOS
                    - digest
                    - imageId
                    - os
                    - container.imageName
                    - image.registry
                    - image.tag
                    - package.inUse
                    - package.info
                    - package.path
                    - package.type
                    - vuln.cvssScore
                    - vuln.hasExploit
                    - vuln.hasFix
                    - vuln.name
                    - vuln.severity
                    - machineImage
                """
                ),
                examples=[
                    'zone in ("zone1") and machineImage = "ami-0b22b359fdfabe1b5"',
                    '(projectId = "1235495521" or projectId = "987654321") and vuln.severity in ("Critical")',
                    'vuln.name in ("CVE-2023-0049")',
                    'vuln.cvssScore >= "3"',
                    'container.name in ("sysdig-container") and not labels exists',
                    'imageId in ("sha256:3768ff6176e29a35ce1354622977a1e5c013045cbc4f30754ef3459218be8ac")',
                    'platform in ("GCP", "AWS", "Azure", "Kubernetes") and isExposed exists',
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
            ctx (Context): A context object containing configuration information.
            filter_exp (str): Sysdig query filter expression to filter inventory resources.
                Use the resource://filter-query-language to get the expected filter expression format.
                Supports operators: =, !=, in, exists, contains, startsWith.
                Combine with and/or/not.
                Examples:
                - zone in ("zone1") and machineImage = "ami-0b22b359fdfabe1b5"
                - (projectId = "1235495521" or projectId = "987654321") and vuln.severity in ("Critical")
                - vuln.name in ("CVE-2023-0049")
                - vuln.cvssScore >= "3"
                - container.name in ("sysdig-container") and not labels exists
                - imageId in ("sha256:3768ff6176e29a35ce1354622977a1e5c013045cbc4f30754ef3459218be8ac")
                - platform in ("GCP", "AWS", "Azure", "Kubernetes") and isExposed exists
            page_number (int): Page number for pagination (1-based).
            page_size (int): Number of items per page.
            with_enrich_containers (bool): Include enriched container information.

        Returns:
            dict: A dictionary containing the results of the inventory query, including pagination information.
            Or a dict containing an error message if the call fails.
        """
        try:
            api_instances: dict = ctx.get_state("api_instances")
            inventory_api: InventoryApi = api_instances.get("inventory")

            start_time = time.time()

            api_response = inventory_api.get_resources_without_preload_content(
                filter=filter_exp, page_number=page_number, page_size=page_size, with_enriched_containers=with_enrich_containers
            )

            execution_time = (time.time() - start_time) * 1000

            response = create_standard_response(results=api_response, execution_time_ms=execution_time)

            return response
        except ToolError as e:
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
            ctx (Context): A context object containing configuration information.
            resource_hash (str): The hash identifier of the resource.

        Returns:
            dict: A dictionary containing the details of the requested inventory resource.
        """
        try:
            api_instances: dict = ctx.get_state("api_instances")
            inventory_api: InventoryApi = api_instances.get("inventory")

            start_time = time.time()

            api_response = inventory_api.get_resource_without_preload_content(hash=resource_hash)
            execution_time = (time.time() - start_time) * 1000

            response = create_standard_response(results=api_response, execution_time_ms=execution_time)

            return response
        except ToolError as e:
            self.log.error(f"Exception when calling InventoryApi->get_resource: {e}")
            raise e
