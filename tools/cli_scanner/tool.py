"""
CLI Scanner Tool for Sysdig

This tool helps you use the Sysdig CLI Scanner to analyze your development files and directories.
"""

import logging
import os
import subprocess
from typing import Literal, Optional
from tempfile import NamedTemporaryFile

from utils.app_config import AppConfig


class CLIScannerTool:
    """
    A class to encapsulate the tools for interacting with the Sysdig CLI Scanner.
    """

    def __init__(self, app_config: AppConfig):
        self.app_config = app_config
        self.log = logging.getLogger(__name__)
        self.cmd: str = "sysdig-cli-scanner"
        self.default_args: list = [
            "--loglevel=err",
            "--apiurl=" + app_config.sysdig_endpoint(),
        ]
        self.iac_default_args: list = [
            "--iac",
            "--group-by=violation",
            "--recursive",
        ]

        self.exit_code_explained: str = """
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
            self.log.info(f"Sysdig CLI Scanner is installed: {result.stdout.strip()}")
        except subprocess.CalledProcessError as e:
            error: dict = {
                "error": "Sysdig CLI Scanner is not installed or not in the $PATH. Check the docs to install it here: https://docs.sysdig.com/en/sysdig-secure/install-vulnerability-cli-scanner/#deployment"
            }
            e.output = error
            raise e

    def check_env_credentials(self) -> None:
        """
        Checks if the necessary environment variables for Sysdig Secure are set.
        Raises:
            EnvironmentError: If the SYSDIG_SECURE_TOKEN or SYSDIG_HOST environment variables are not set.
        """
        sysdig_secure_token = self.app_config.sysdig_secure_token()
        sysdig_host = self.app_config.sysdig_endpoint()
        if not sysdig_secure_token:
            self.log.error("SYSDIG_SECURE_TOKEN environment variable is not set.")
            raise EnvironmentError("SYSDIG_SECURE_TOKEN environment variable is not set.")
        else:
            os.environ["SECURE_API_TOKEN"] = sysdig_secure_token  # Ensure the token is set in the environment
        if not sysdig_host:
            self.log.error("SYSDIG_HOST environment variable is not set.")
            raise EnvironmentError("SYSDIG_HOST environment variable is not set.")

    def run_sysdig_cli_scanner(
        self,
        image: Optional[str] = None,
        mode: Literal["vulnerability", "iac"] = "vulnerability",
        standalone: Optional[bool] = False,
        offline_analyser: Optional[bool] = False,
        full_vulnerability_table: Optional[bool] = False,
        separate_by_layer: Optional[bool] = False,
        separate_by_image: Optional[bool] = False,
        detailed_policies_evaluation: Optional[bool] = False,
        path_to_scan: Optional[str] = None,
        iac_group_by: Optional[Literal["policy", "resource", "violation"]] = "policy",
        iac_recursive: Optional[bool] = True,
        iac_severity_threshold: Optional[Literal["never", "high", "medium", "low"]] = "high",
        iac_list_unsupported_resources: Optional[bool] = False,
    ) -> dict:
        """
        Analyzes a Container image for vulnerabilities using the Sysdig CLI Scanner.
        Args:
            image (str): The name of the container image to analyze.
            mode ["vulnerability", "iac"]: The mode of analysis, either "vulnerability" or "iac".
                Defaults to "vulnerability".
            standalone (bool): In vulnerability mode, run the scan in standalone mode.
                Not dependent on Sysdig backend.
            offline_analyser (bool): In vulnerability mode, does not perform calls to the Sysdig backend.
            full_vulnerability_table (bool): In vulnerability mode, generates a table with all the vulnerabilities,
                not just the most important ones.
            separate_by_layer (bool): In vulnerability mode, separates vulnerabilities by layer.
            separate_by_image (bool): In vulnerability mode, separates vulnerabilities by image.
            detailed_policies_evaluation (bool): In vulnerability mode, evaluates policies in detail.
            path_to_scan (str): The path to the directory/file to scan in IaC mode.
            iac_group_by (str): In IaC mode, groups the results by the specified field.
                Options are "policy", "resource", or "violation". Defaults to "policy".
            iac_recursive (bool): In IaC mode, scans the directory recursively. Defaults to True.
            iac_severity_threshold (str): In IaC mode, sets the severity threshold for vulnerabilities.
                Options are "never", "high", "medium", or "low". Defaults to "high".
            iac_list_unsupported_resources (bool): In IaC mode, lists unsupported resources.
                Defaults to False.

        Returns:
            dict: A dictionary containing the output of the analysis of vulnerabilities.
        Raises:
            Exception: If the Sysdig CLI Scanner encounters an error.
        """
        # Check if Sysdig CLI Scanner is installed and environment credentials are set
        self.check_sysdig_cli_installed()
        self.check_env_credentials()

        tmp_result_file = NamedTemporaryFile(suffix=".json", prefix="sysdig_cli_scanner_", delete_on_close=False)
        # Prepare the command based on the mode
        if mode == "iac":
            self.log.info("Running Sysdig CLI Scanner in IaC mode.")
            extra_iac_args = [
                f"--group-by={iac_group_by}",
                f"--severity-threshold={iac_severity_threshold}",
                "--recursive" if iac_recursive else "",
                "--list-unsupported-resources" if iac_list_unsupported_resources else "",
            ]
            # Remove empty strings from the list
            extra_iac_args = [arg for arg in extra_iac_args if arg]
            cmd = [self.cmd] + self.default_args + self.iac_default_args + extra_iac_args + [path_to_scan]
        else:
            self.log.info("Running Sysdig CLI Scanner in vulnerability mode.")
            # Default to vulnerability mode
            extra_args = [
                "--standalone" if standalone else "",
                "--offline-analyzer" if offline_analyser and standalone else "",
                "--full-vulns-table" if full_vulnerability_table else "",
                "--separate-by-layer" if separate_by_layer else "",
                "--separate-by-image" if separate_by_image else "",
                "--detailed-policies-eval" if detailed_policies_evaluation else "",
            ]
            extra_args = [arg for arg in extra_args if arg]  # Remove empty strings from the list
            cmd = [self.cmd] + self.default_args + extra_args + [image]

        try:
            # Run the command
            with open(tmp_result_file.name, "w") as output_file:
                result = subprocess.run(cmd, text=True, check=True, stdout=output_file, stderr=subprocess.PIPE)
            with open(tmp_result_file.name, "rt") as output_file:
                output_result = output_file.read()
                return {
                    "exit_code": result.returncode,
                    "output": output_result + result.stderr.strip(),
                    "exit_codes_explained": self.exit_code_explained,
                }
        # Handle non-zero exit codes specially exit code 1
        except subprocess.CalledProcessError as e:
            self.log.warning(f"Sysdig CLI Scanner returned non-zero exit code: {e.returncode}")
            if e.returncode in [2, 3]:
                self.log.error(f"Sysdig CLI Scanner encountered an error: {e.stderr.strip()}")
                result: dict = {
                    "error": "Error running Sysdig CLI Scanner",
                    "exit_code": e.returncode,
                    "output": e.stderr.strip(),
                    "exit_codes_explained": self.exit_code_explained,
                }
                raise Exception(result)
            else:
                with open(tmp_result_file.name, "r") as output_file:
                    output_result = output_file.read()
                result: dict = {
                    "exit_code": e.returncode,
                    "stdout": e.stdout,
                    "output": output_result,
                    "exit_codes_explained": self.exit_code_explained,
                }
                return result
        # Handle any other exceptions that may occur and exit codes 2 and 3
        except Exception as e:
            raise e
        finally:
            if os.path.exists(tmp_result_file.name):
                os.remove(tmp_result_file.name)
