"""
Context variables for request-scoped data propagation.

This module provides thread-safe (task-safe in async) context variables
for passing request metadata through the agent execution chain without
explicit parameter threading.
"""

import contextvars
import logging
from typing import Optional

logger = logging.getLogger(__name__)

# Context variable to store the current request's user ID
# Set at request start, cleared at request end
# Accessible by httpx event hooks for header injection
request_user_id_var: contextvars.ContextVar[Optional[str]] = contextvars.ContextVar("request_user_id", default=None)


def inject_user_id_header(request):
    """
    httpx event hook to inject X-User-ID header from request context.

    This function is called by httpx before sending each request.
    It reads the user ID from the current context and adds it as a header,
    unless:
    1. The header already exists (explicit config takes precedence)
    2. No user ID is set in context
    3. The user ID is a generated fallback (A2A_USER_* prefix)

    Args:
        request: httpx.Request object to modify

    Returns:
        None (modifies request in-place)
    """
    # Check if X-User-ID already present (case-insensitive)
    if "X-User-ID" in request.headers or "x-user-id" in request.headers:
        logger.debug(
            "Skipping user_id injection: X-User-ID already present in headers", extra={"url": str(request.url)}
        )
        return

    user_id = request_user_id_var.get()

    # Skip if no user ID
    if not user_id:
        logger.debug("Skipping user_id injection: no user_id in context")
        return

    # Skip if it's a generated fallback ID
    if user_id.startswith("A2A_USER_"):
        logger.debug("Skipping user_id injection: generated ID detected", extra={"user_id_prefix": user_id[:15]})
        return

    # Inject the user ID header
    request.headers["X-User-ID"] = user_id
    logger.debug("Injected X-User-ID header for subagent call", extra={"user_id": user_id, "url": str(request.url)})
