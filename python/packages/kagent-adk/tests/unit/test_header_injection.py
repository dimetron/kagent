"""
Unit tests for inject_user_id_header function.

Tests header injection logic with various scenarios including:
- Happy path injection
- Precedence (existing headers)
- Graceful handling (no context)
- Generated ID skipping

These tests MUST fail initially until inject_user_id_header is implemented.
"""

import asyncio

import pytest


def test_inject_user_id_when_context_set(mock_httpx_request):
    """Test that user ID is injected when context is set."""
    from kagent.adk.context import inject_user_id_header, request_user_id_var

    # Set user ID in context
    request_user_id_var.set("test-user@example.com")

    # Inject header
    inject_user_id_header(mock_httpx_request)

    # Header should be present
    assert mock_httpx_request.headers["X-User-ID"] == "test-user@example.com"

    # Clean up
    request_user_id_var.set(None)


def test_skip_injection_when_header_exists(mock_httpx_request_with_headers):
    """Test that injection is skipped when X-User-ID already exists."""
    from kagent.adk.context import inject_user_id_header, request_user_id_var

    # Set different user ID in context
    request_user_id_var.set("context-user@example.com")

    # Original header value
    original_value = mock_httpx_request_with_headers.headers["X-User-ID"]

    # Try to inject
    inject_user_id_header(mock_httpx_request_with_headers)

    # Header should NOT be overwritten (explicit config wins)
    assert mock_httpx_request_with_headers.headers["X-User-ID"] == original_value
    assert mock_httpx_request_with_headers.headers["X-User-ID"] == "explicit-user@example.com"

    # Clean up
    request_user_id_var.set(None)


def test_skip_injection_when_header_exists_lowercase(mock_httpx_request):
    """Test case-insensitive precedence check (x-user-id vs X-User-ID)."""
    from kagent.adk.context import inject_user_id_header, request_user_id_var

    # Set lowercase header
    mock_httpx_request.headers["x-user-id"] = "lowercase-user@example.com"

    # Set context with different value
    request_user_id_var.set("context-user@example.com")

    # Try to inject
    inject_user_id_header(mock_httpx_request)

    # Should not override lowercase header
    assert mock_httpx_request.headers["x-user-id"] == "lowercase-user@example.com"

    # Clean up
    request_user_id_var.set(None)


def test_skip_injection_when_context_empty(mock_httpx_request):
    """Test graceful handling when context is not set (None)."""
    from kagent.adk.context import inject_user_id_header, request_user_id_var

    # Ensure context is empty
    request_user_id_var.set(None)

    # Try to inject
    inject_user_id_header(mock_httpx_request)

    # No header should be added
    assert "X-User-ID" not in mock_httpx_request.headers
    assert "x-user-id" not in mock_httpx_request.headers


def test_skip_injection_for_generated_ids(mock_httpx_request):
    """Test that generated IDs (A2A_USER_*) are not propagated."""
    from kagent.adk.context import inject_user_id_header, request_user_id_var

    # Set generated user ID (with A2A_USER_ prefix)
    request_user_id_var.set("A2A_USER_abc123")

    # Try to inject
    inject_user_id_header(mock_httpx_request)

    # Header should NOT be added for generated IDs
    assert "X-User-ID" not in mock_httpx_request.headers

    # Clean up
    request_user_id_var.set(None)


@pytest.mark.asyncio
async def test_context_isolation_concurrent_requests(mock_httpx_request):
    """Test that concurrent requests don't interfere with each other's context."""
    from kagent.adk.context import inject_user_id_header, request_user_id_var

    import httpx

    # Track which headers were injected
    injected_headers = []

    async def process_request(user_id: str):
        """Simulate processing a request with a specific user ID."""
        # Each task sets its own user ID
        request_user_id_var.set(user_id)

        # Create a fresh request for this task
        request = httpx.Request("POST", "http://test-agent:8080/message")

        # Inject header
        inject_user_id_header(request)

        # Record what was injected
        await asyncio.sleep(0.01)  # Simulate async work
        injected_headers.append(request.headers.get("X-User-ID"))

        # Clean up
        request_user_id_var.set(None)

    # Run multiple requests concurrently
    await asyncio.gather(
        process_request("user-1@example.com"),
        process_request("user-2@example.com"),
        process_request("user-3@example.com"),
    )

    # Each request should have gotten its own user ID
    assert set(injected_headers) == {
        "user-1@example.com",
        "user-2@example.com",
        "user-3@example.com",
    }
    assert len(injected_headers) == 3
