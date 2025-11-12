"""
Patch to handle MCP SDK progress notifications.

The MCP server sends progress notifications with progressToken=None, but the
MCP Python SDK's Pydantic validation expects progressToken to be str | int.

According to the MCP spec, progressToken is optional and can be null when
not tracking progress for a specific request.

This patch:
1. Captures progress notifications before they cause validation errors
2. Converts them to A2A TaskStatusUpdateEvents for UI display
3. Suppresses validation warnings as a fallback

Related issues:
- https://github.com/modelcontextprotocol/servers/issues/1810
- https://github.com/cloudwalk/hermes-mcp/issues/12
"""

import logging
import re
import warnings
from datetime import UTC, datetime
from typing import Any

logger = logging.getLogger(__name__)


def extract_progress_info(log_message: str) -> dict[str, Any] | None:
    """
    Extract progress information from MCP validation error log messages.

    Args:
        log_message: The log message containing progress notification data

    Returns:
        Dictionary with progress info if found, None otherwise
    """
    # Look for the progress notification in the log message
    # Format: "Message was: method='notifications/progress' params={'progressToken': None, 'message': '...', 'progress': X, 'total': Y}"
    pattern = r"params=\{[^}]*'message':\s*'([^']+)'[^}]*'progress':\s*([\d.]+)[^}]*'total':\s*([\d.]+)[^}]*\}"

    match = re.search(pattern, log_message)
    if match:
        message = match.group(1)
        progress = float(match.group(2))
        total = float(match.group(3))

        # Calculate percentage
        percentage = (progress / total * 100) if total > 0 else 0

        return {
            "message": message,
            "progress": progress,
            "total": total,
            "percentage": percentage,
        }

    return None


class MCPProgressNotificationHandler(logging.Filter):
    """
    Logging filter that captures MCP progress notifications and converts them
    to A2A TaskStatusUpdateEvents for UI display.
    """

    def __init__(self):
        super().__init__()
        self._last_progress = {}  # Track last progress per context to avoid spam

    def filter(self, record):
        """
        Filter log records, capturing progress notifications.

        Returns:
            False to suppress the log message, True to let it through
        """
        message = record.getMessage()

        # Check if this is a progress notification validation error
        if "Failed to validate notification" in message and "notifications/progress" in message:
            # Extract progress information
            progress_info = extract_progress_info(message)

            if progress_info:
                # Log the progress at debug level instead of warning
                logger.debug(f"MCP Progress: {progress_info['message']} ({progress_info['percentage']:.1f}%)")

                # Send progress as A2A TaskStatusUpdateEvent
                try:
                    from .mcp_progress_handler import send_mcp_progress_event_sync

                    send_mcp_progress_event_sync(
                        message=progress_info["message"],
                        progress=progress_info["progress"],
                        total=progress_info["total"],
                        metadata={"percentage": progress_info["percentage"]},
                    )
                except Exception as e:
                    logger.debug(f"Could not send MCP progress event: {e}")

            # Suppress the validation error
            return False

        # Suppress Pydantic validation errors for ProgressNotification
        if "validation error" in message.lower() and "progressnotification" in message.lower():
            return False

        return True


def suppress_mcp_validation_warnings():
    """
    Install handler to capture and process MCP progress notifications.

    This function:
    1. Installs a logging filter to capture progress notifications
    2. Suppresses validation warnings that don't affect functionality
    3. Logs progress information at debug level
    """
    # Suppress Python warnings about validation
    warnings.filterwarnings("ignore", message=".*Failed to validate notification.*")
    warnings.filterwarnings("ignore", message=".*validation error.*ProgressNotification.*")

    # Install progress notification handler on root logger
    root_logger = logging.getLogger()
    root_logger.addFilter(MCPProgressNotificationHandler())

    logger.debug("Installed MCP progress notification handler")


# Automatically apply the patch when this module is imported
try:
    suppress_mcp_validation_warnings()
    logger.info("MCP validation warning suppression enabled")
except Exception as e:
    logger.error(f"Failed to apply MCP validation warning suppression: {e}")
