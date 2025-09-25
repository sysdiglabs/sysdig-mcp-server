"""
Helper functions for working with Sysdig API clients.
"""

# Sysdig permissions needed for the different set of tools
TOOL_PERMISSIONS = {
    "inventory": ["explore.read"],
    "vulnerability": ["scanning.read", "secure.vm.scanresults.read"],
    "sysql": ["sage.exec", "sage.manage.exec", "explore.read"],
    "cli-scanner": ["secure.vm.cli-scanner.exec"],
    "threat-detection": ["custom-events.read"],
}
