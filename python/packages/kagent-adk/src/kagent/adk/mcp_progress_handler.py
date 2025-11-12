"""
Handler for MCP progress notifications to A2A event conversion.

This module provides a context manager and event propagation mechanism
to convert MCP server progress notifications into A2A TaskStatusUpdateEvents
that can be displayed in the UI.
"""

import asyncio
import logging
import uuid
from contextlib import asynccontextmanager
from contextvars import ContextVar
from datetime import UTC, datetime
from typing import Any

from a2a.server.events.event_queue import EventQueue
from a2a.types import Message, Part, Role, TaskState, TaskStatus, TaskStatusUpdateEvent, TextPart

logger = logging.getLogger(__name__)

# Context variable to store the current event queue and context
_event_queue_context: ContextVar[tuple[EventQueue, str, str] | None] = ContextVar(
    "mcp_event_queue_context", default=None
)


@asynccontextmanager
async def mcp_progress_context(event_queue: EventQueue, task_id: str, context_id: str):
    """
    Context manager to enable MCP progress notification forwarding.

    Usage:
        async with mcp_progress_context(event_queue, task_id, context_id):
            # Run agent with MCP tools
            # Progress notifications will be forwarded as A2A events
            pass

    Args:
        event_queue: The A2A event queue to send progress updates to
        task_id: The task ID for the current execution
        context_id: The context ID for the current execution
    """
    token = _event_queue_context.set((event_queue, task_id, context_id))
    try:
        logger.debug(f"Enabled MCP progress forwarding for task {task_id}")
        yield
    finally:
        _event_queue_context.reset(token)
        logger.debug(f"Disabled MCP progress forwarding for task {task_id}")


async def send_mcp_progress_event(
    message: str,
    progress: float | None = None,
    total: float | None = None,
    metadata: dict[str, Any] | None = None,
):
    """
    Send an MCP progress notification as an A2A TaskStatusUpdateEvent.

    This function should be called when MCP progress notifications are received.
    It will only work if called within an mcp_progress_context.

    Args:
        message: Progress message from the MCP server
        progress: Current progress value (optional)
        total: Total progress value (optional)
        metadata: Additional metadata to include (optional)
    """
    context = _event_queue_context.get()
    if context is None:
        # No event queue registered, skip
        logger.debug(f"No event queue context, skipping progress event: {message}")
        return

    event_queue, task_id, context_id = context

    # Build progress message text
    progress_text = message
    if progress is not None and total is not None:
        percentage = (progress / total * 100) if total > 0 else 0
        progress_text = f"{message} ({percentage:.1f}%)"

    # Create A2A message
    a2a_message = Message(
        message_id=str(uuid.uuid4()),
        role=Role.agent,
        parts=[Part(TextPart(text=f"🔄 {progress_text}"))],
    )

    # Build event metadata
    event_metadata = {
        "mcp_progress": True,
        "progress_source": "mcp_tool",
    }
    if metadata:
        event_metadata.update(metadata)
    if progress is not None:
        event_metadata["progress_current"] = progress
    if total is not None:
        event_metadata["progress_total"] = total

    # Create and send status update event
    event = TaskStatusUpdateEvent(
        task_id=task_id,
        status=TaskStatus(
            state=TaskState.working,
            timestamp=datetime.now(UTC).isoformat(),
            message=a2a_message,
        ),
        context_id=context_id,
        final=False,
        metadata=event_metadata,
    )

    try:
        await event_queue.enqueue_event(event)
        logger.debug(f"Sent MCP progress event: {progress_text}")
    except Exception as e:
        logger.warning(f"Failed to send MCP progress event: {e}")


def send_mcp_progress_event_sync(
    message: str,
    progress: float | None = None,
    total: float | None = None,
    metadata: dict[str, Any] | None = None,
):
    """
    Synchronous wrapper for send_mcp_progress_event.

    Attempts to send the progress event from a synchronous context.
    If no event loop is running, the event is skipped.

    Args:
        message: Progress message from the MCP server
        progress: Current progress value (optional)
        total: Total progress value (optional)
        metadata: Additional metadata to include (optional)
    """
    try:
        loop = asyncio.get_event_loop()
        if loop.is_running():
            # Schedule the coroutine to run in the event loop
            asyncio.create_task(send_mcp_progress_event(message, progress, total, metadata))
        else:
            # No running loop, run synchronously
            loop.run_until_complete(send_mcp_progress_event(message, progress, total, metadata))
    except RuntimeError:
        # No event loop available
        logger.debug(f"No event loop available for progress event: {message}")
    except Exception as e:
        logger.warning(f"Failed to send MCP progress event synchronously: {e}")


def get_event_queue_context() -> tuple[EventQueue, str, str] | None:
    """
    Get the current event queue context if available.

    Returns:
        Tuple of (event_queue, task_id, context_id) or None
    """
    return _event_queue_context.get()
