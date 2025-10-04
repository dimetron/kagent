"""
Unit tests for create_remote_agent with event hook registration.

Tests that create_remote_agent() properly registers the inject_user_id_header
event hook in the httpx client.

These tests verify both existing behavior (headers, timeout) and new behavior
(event hook registration).
"""

from unittest.mock import MagicMock, patch

import pytest


def test_create_remote_agent_with_headers():
    """Test that static headers from config are passed through."""
    from kagent.adk.types import create_remote_agent

    headers = {"Authorization": "Bearer token123", "X-Custom": "value"}

    with patch("kagent.adk.types.httpx.AsyncClient") as mock_client:
        with patch("kagent.adk.types.RemoteA2aAgent") as mock_agent:
            create_remote_agent(
                name="test-agent",
                url="http://test:8080",
                headers=headers,
                timeout=30.0,
                description="Test agent",
            )

            # Verify AsyncClient was called with headers
            mock_client.assert_called_once()
            call_kwargs = mock_client.call_args.kwargs
            assert call_kwargs["headers"] == headers


def test_create_remote_agent_event_hook_registered():
    """Test that event hook is registered in httpx client."""
    from kagent.adk.types import create_remote_agent

    with patch("kagent.adk.types.httpx.AsyncClient") as mock_client:
        with patch("kagent.adk.types.RemoteA2aAgent") as mock_agent:
            create_remote_agent(
                name="test-agent",
                url="http://test:8080",
                headers={"X-Test": "value"},
                timeout=30.0,
                description="Test agent",
            )

            # Verify event_hooks parameter was passed
            call_kwargs = mock_client.call_args.kwargs
            assert "event_hooks" in call_kwargs
            assert "request" in call_kwargs["event_hooks"]

            # Verify inject_user_id_header is in the request hooks
            from kagent.adk.context import inject_user_id_header

            assert inject_user_id_header in call_kwargs["event_hooks"]["request"]


def test_create_remote_agent_without_headers():
    """Test that event hook is registered even without static headers."""
    from kagent.adk.types import create_remote_agent

    with patch("kagent.adk.types.httpx.AsyncClient") as mock_client:
        with patch("kagent.adk.types.RemoteA2aAgent") as mock_agent:
            create_remote_agent(
                name="test-agent",
                url="http://test:8080",
                headers=None,  # No static headers
                timeout=30.0,
                description="Test agent",
            )

            # Event hook should still be registered
            call_kwargs = mock_client.call_args.kwargs
            assert "event_hooks" in call_kwargs
            assert "request" in call_kwargs["event_hooks"]

            from kagent.adk.context import inject_user_id_header

            assert inject_user_id_header in call_kwargs["event_hooks"]["request"]


def test_httpx_client_created_with_timeout():
    """Test that timeout configuration is preserved."""
    from kagent.adk.types import create_remote_agent

    timeout_value = 45.0

    with patch("kagent.adk.types.httpx.AsyncClient") as mock_client:
        with patch("kagent.adk.types.httpx.Timeout") as mock_timeout:
            with patch("kagent.adk.types.RemoteA2aAgent") as mock_agent:
                create_remote_agent(
                    name="test-agent",
                    url="http://test:8080",
                    headers=None,
                    timeout=timeout_value,
                    description="Test agent",
                )

                # Verify Timeout was created with correct value
                mock_timeout.assert_called_once_with(timeout=timeout_value)

                # Verify AsyncClient received the timeout
                call_kwargs = mock_client.call_args.kwargs
                assert "timeout" in call_kwargs
