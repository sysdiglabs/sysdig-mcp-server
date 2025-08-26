"""
Helper functions for working with Sysdig API clients.
"""

# Sysdig permissions needed for the different set of tools
TOOL_PERMISSIONS = {
    "inventory": ["explore.read"],
    "vulnerability": ["scanning.read", "secure.vm.scanresults.read"],
    "sage": ["sage.exec", "sage.manage.exec"],
    "cli-scanner": ["secure.vm.cli-scanner.exec"],
    "threat-detection": ["custom-events.read"],
}