"""
CLI Scanner Tool for Sysdig

This tool helps you use the Sysdig CLI Scanner to analyze your development files and directories.
"""

import logging
import os
import subprocess
from typing import Literal, Optional

from utils.app_config import get_app_config

logging.basicConfig(format="%(asctime)s-%(process)d-%(levelname)s- %(message)s", level=os.environ.get("LOGLEVEL", "ERROR"))

log = logging.getLogger(__name__)

# Load app config (expects keys: mcp.host, mcp.port, mcp.transport)
app_config = get_app_config()


class CLIScannerTool:
    """
    A class to encapsulate the tools for interacting with the Sysdig CLI Scanner.
    """

    cmd: str = "sysdig-cli-scanner"
    default_args: list = [
        "--loglevel=err",
        "--apiurl=" + app_config["sysdig"]["host"],
    ]
    iac_default_args: list = [
        "--iac",
        "--group-by=violation",
        "--recursive",
    ]

    exit_code_explained: str = """
        Exit codes:
            0: Scan evaluation "pass"
            1: Scan evaluation "fail"
            2: Invalid parameters
            3: Internal error
        """

    def check_sysdig_cli_installed(self) -> None:
        """
        Checks if the Sysdig CLI Scanner is installed by verifying the existence of the 'sysdig-cli-scanner' command.
        """
        try:
            # Attempt to run 'sysdig-cli-scanner --version' to check if it's installed
            result = subprocess.run([self.cmd, "--version"], capture_output=True, text=True, check=True)
            log.info(f"Sysdig CLI Scanner is installed: {result.stdout.strip()}")
        except subprocess.CalledProcessError as e:
            error: dict = {
                "error": "Sysdig CLI Scanner is not installed. Check the docs to install it here: https://docs.sysdig.com/en/sysdig-secure/install-vulnerability-cli-scanner/#deployment"
            }
            e.output = error
            raise e

    def check_env_credentials(self) -> None:
        """
        Checks if the necessary environment variables for Sysdig Secure are set.
        Raises:
            EnvironmentError: If the SYSDIG_SECURE_TOKEN or SYSDIG_HOST environment variables are not set.
        """
        sysdig_secure_token = os.environ.get("SYSDIG_SECURE_TOKEN")
        sysdig_host = os.environ.get("SYSDIG_HOST", app_config["sysdig"]["host"])
        if not sysdig_secure_token:
            log.error("SYSDIG_SECURE_TOKEN environment variable is not set.")
            raise EnvironmentError("SYSDIG_SECURE_TOKEN environment variable is not set.")
        else:
            os.environ["SECURE_API_TOKEN"] = sysdig_secure_token  # Ensure the token is set in the environment
        if not sysdig_host:
            log.error("SYSDIG_HOST environment variable is not set.")
            raise EnvironmentError("SYSDIG_HOST environment variable is not set.")

    def run_sysdig_cli_scanner(
        self,
        image: Optional[str] = None,
        directory_path: Optional[str] = None,
        mode: Literal["vulnerability", "iac"] = "vulnerability",
    ) -> dict:
        """
        Analyzes a Container image for vulnerabilities using the Sysdig CLI Scanner.
        Args:
            image (str): The name of the container image to analyze.
            directory_path (str): The path to the directory containing IaC files to analyze.
            mode ["vulnerability", "iac"]: The mode of analysis, either "vulnerability" or "iac".
                Defaults to "vulnerability".
        Returns:
            dict: A dictionary containing the output of the analysis of vulnerabilities.
        Raises:
            Exception: If the Sysdig CLI Scanner encounters an error.
        """
        # Check if Sysdig CLI Scanner is installed and environment credentials are set
        self.check_sysdig_cli_installed()
        self.check_env_credentials()

        # Prepare the command based on the mode
        if mode == "iac":
            log.info("Running Sysdig CLI Scanner in IaC mode.")
            cmd = [self.cmd] + self.default_args + self.iac_default_args + [directory_path]
        else:
            log.info("Running Sysdig CLI Scanner in vulnerability mode.")
            # Default to vulnerability mode
            cmd = [self.cmd] + self.default_args + [image]

        try:
            # Run the command
            with open("sysdig_cli_scanner_output.json", "w") as output_file:
                result = subprocess.run(cmd, text=True, check=True, stdout=output_file, stderr=subprocess.PIPE)
                output_result = output_file.read()
                output_file.close()
                return {
                    "exit_code": result.returncode,
                    "output": output_result,
                    "exit_codes_explained": self.exit_code_explained,
                }
        # Handle non-zero exit codes speically exit code 1
        except subprocess.CalledProcessError as e:
            log.warning(f"Sysdig CLI Scanner returned non-zero exit code: {e.returncode}")
            if e.returncode in [2, 3]:
                log.error(f"Sysdig CLI Scanner encountered an error: {e.stderr.strip()}")
                result: dict = {
                    "error": "Error running Sysdig CLI Scanner",
                    "exit_code": e.returncode,
                    "output": e.stderr.strip(),
                    "exit_codes_explained": self.exit_code_explained,
                }
                raise Exception(result)
            else:
                with open("sysdig_cli_scanner_output.json", "r") as output_file:
                    output_result = output_file.read()
                result: dict = {
                    "exit_code": e.returncode,
                    "stdout": e.stdout,
                    "output": output_result,
                    "exit_codes_explained": self.exit_code_explained,
                }
                os.remove("sysdig_cli_scanner_output.json")
                return result
        # Handle any other exceptions that may occur and exit codes 2 and 3
        except Exception as e:
            raise e
