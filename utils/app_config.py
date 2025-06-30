"""
Utility functions to load and manage the application configuration from a YAML file.
It will load a singleton configuration object that can be accessed throughout the application.
"""

import yaml
import logging
import os
from typing import Optional

# Set up logging
log = logging.getLogger(__name__)
logging.basicConfig(format="%(asctime)s-%(process)d-%(levelname)s- %(message)s", level=os.environ.get("LOGLEVEL", "ERROR"))

# app_config singleton
_app_config: Optional[dict] = None
APP_CONFIG_FILE: str = os.getenv("APP_CONFIG_FILE", "./app_config.yaml")


def env_constructor(loader, node):
    return os.environ[node.value[0:]]


def check_config_file_exists() -> bool:
    """
    Check if the config file exists

    Returns:
        bool: True if the config file exists, False otherwise
    """
    if os.path.exists(APP_CONFIG_FILE):
        log.debug("Config file exists")
        return True
    else:
        log.error("Config file does not exist")
        return False


def load_app_config() -> dict:
    """
    Load the app config from the YAML file

    Returns:
        dict: The app config loaded from the YAML file
    """
    if not check_config_file_exists():
        log.error("Config file does not exist")
        return {}
    # Load the config file
    app_config: dict = {}
    log.debug(f"Loading app config from YAML file: {APP_CONFIG_FILE}")
    with open(APP_CONFIG_FILE, "r", encoding="utf8") as file:
        try:
            yaml.add_constructor("!env", env_constructor, Loader=yaml.SafeLoader)
            app_config: dict = yaml.safe_load(file)
        except Exception as exc:
            logging.error(exc)
    return app_config


def get_app_config() -> dict:
    """
    Get the the overall app config

    Returns:
        dict: The app config loaded from the YAML file, or an empty dict if the file
    """
    global _app_config
    if _app_config is None:
        _app_config = load_app_config()
    return _app_config
