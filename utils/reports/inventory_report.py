"""
This script lists all resources in the Sysdig inventory and saves them to a CSV file.
"""

import logging
import os
import dask.dataframe as dd
import pandas as pd
from tools.inventory.tool import InventoryTools
from fastmcp import Context, FastMCP

# Configure logging
logging.basicConfig(format="%(asctime)s-%(process)d-%(levelname)s- %(message)s", level=os.environ.get("LOGLEVEL", "ERROR"))
log = logging.getLogger(__name__)


inventory = InventoryTools()


class MockMCP(FastMCP):
    """
    Mock class for FastMCP
    """

    pass


# Mocking MCP context for the inventory tool
fastmcp: MockMCP = MockMCP(
    tags=["sysdig", "mcp", "stdio"],
)
ctx = Context(fastmcp=fastmcp)


def list_all_resources(filter_exp: str = 'platform in ("GCP")') -> dd.DataFrame:
    """
    List all resources in the Sysdig inventory.

    Args:
        filter_exp (str): Filter expression to apply to the inventory items.
    Returns:
        dd.DataFrame: DataFrame containing all resources.
    """
    # Get the list of all resources
    df: dd.DataFrame = None
    logging.debug(f"Listing all resources with filter: {filter_exp}")
    try:
        resources = inventory.tool_list_resources(ctx=ctx, filter_exp=filter_exp, page_number=1, page_size=1000)
        df = pd.DataFrame.from_records([r for r in resources.get("results", {}).get("data", [])])
        while resources.get("results", {}).get("page", {}).get("next"):
            # Get the next page of resources
            next_page = resources.get("results", {}).get("page", {}).get("next")
            logging.debug(f"Fetching next page: {next_page}")
            resources = inventory.tool_list_resources(ctx=ctx, filter_exp=filter_exp, page_number=next_page, page_size=1000)
            df = dd.concat(
                [df, pd.DataFrame.from_records([r for r in resources.get("results", {}).get("data", [])])], ignore_index=True
            )
            dd.from_pandas
        return df
    except Exception as e:
        logging.error(f"Error listing resources: {e}")
        return None


def save_resources_to_csv(df: dd.DataFrame, filename: str) -> None:
    """
    Save the DataFrame to a CSV file.

    Args:
            df (dd.DataFrame): DataFrame to save.
            filename (str): Name of the CSV file.
    """
    df.to_csv(filename, index=False, header=True, single_file=True)


def main(filter_exp: str = 'platform in ("GCP")', filename: str = "resources.csv") -> None:
    """
    Main function to list resources and save them to a CSV file.

    Args:
        filter_exp (str): Filter expression to apply to the inventory items.
        filename (str): Name of the CSV file to save the resources.
    """
    # List resources with the specified filter expression
    df = list_all_resources(filter_exp)

    # Save the resources to a CSV file
    save_resources_to_csv(df, filename)

    print("Resources saved to resources.csv")


if __name__ == "__main__":
    main()
