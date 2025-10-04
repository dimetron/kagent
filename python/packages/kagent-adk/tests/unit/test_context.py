"""
Unit tests for kagent.adk.context module.

Tests the ContextVar behavior for request-scoped user ID storage.
These tests MUST fail initially (TDD approach) until context.py is implemented.
"""

import asyncio

import pytest


def test_context_var_default_is_none():
    """Test that request_user_id_var has default value of None."""
    from kagent.adk.context import request_user_id_var

    # Default value should be None
    assert request_user_id_var.get() is None


def test_context_var_set_and_get():
    """Test setting and retrieving user ID from context variable."""
    from kagent.adk.context import request_user_id_var

    # Set a user ID
    test_user_id = "test-user@example.com"
    request_user_id_var.set(test_user_id)

    # Should be able to retrieve it
    assert request_user_id_var.get() == test_user_id

    # Clean up
    request_user_id_var.set(None)


@pytest.mark.asyncio
async def test_context_var_isolation_between_tasks():
    """Test that ContextVar provides isolation between async tasks."""
    from kagent.adk.context import request_user_id_var

    # Track results from each task
    results = []

    async def task_with_user_id(user_id: str):
        """Async task that sets its own user ID."""
        request_user_id_var.set(user_id)
        await asyncio.sleep(0.01)  # Simulate async work
        # Each task should see its own value
        results.append(request_user_id_var.get())

    # Run two tasks concurrently with different user IDs
    await asyncio.gather(
        task_with_user_id("user-a@example.com"),
        task_with_user_id("user-b@example.com"),
    )

    # Each task should have seen its own user ID (order may vary)
    assert set(results) == {"user-a@example.com", "user-b@example.com"}
    assert len(results) == 2


def test_context_var_cleared_after_use():
    """Test that context variable can be cleared."""
    from kagent.adk.context import request_user_id_var

    # Set a value
    request_user_id_var.set("temp-user@example.com")
    assert request_user_id_var.get() == "temp-user@example.com"

    # Clear it
    request_user_id_var.set(None)
    assert request_user_id_var.get() is None
