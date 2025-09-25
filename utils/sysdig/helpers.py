"""
Helper functions for working with Sysdig API clients.
"""

# Sysdig permissions needed for the different set of tools
TOOL_PERMISSIONS = {
    "cli-scanner": ["secure.vm.cli-scanner.exec"],
    "threat-detection": ["custom-events.read"],
    "sysql": ["sage.exec", "sage.manage.exec"],
}
