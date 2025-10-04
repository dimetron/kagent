"""
Test fixtures for RequestContext and httpx Request mocking.

These fixtures provide reusable mock objects for testing user ID propagation
through the agent execution chain.
"""

import pytest
from unittest.mock import MagicMock

import httpx
from a2a.server.agent_execution.context import RequestContext
from kagent.core.a2a._requests import KAgentUser


@pytest.fixture
def mock_request_context():
    """RequestContext with test user ID.

    Returns:
        RequestContext: Mock context with user_id="test-user@example.com"
    """
    context = MagicMock(spec=RequestContext)
    context.task_id = "test-task-123"
    context.context_id = "test-context-456"
    context.call_context = MagicMock()
    context.call_context.user = KAgentUser(user_id="test-user@example.com")
    return context


@pytest.fixture
def mock_request_context_no_user():
    """RequestContext without user (fallback scenario).

    Returns:
        RequestContext: Mock context with no user (call_context=None)
    """
    context = MagicMock(spec=RequestContext)
    context.task_id = "test-task-789"
    context.context_id = "test-context-012"
    context.call_context = None
    return context


@pytest.fixture
def mock_request_context_generated_user():
    """RequestContext with generated user ID (A2A_USER_ prefix).

    Returns:
        RequestContext: Mock context with generated user ID
    """
    context = MagicMock(spec=RequestContext)
    context.task_id = "test-task-345"
    context.context_id = "test-context-678"
    context.call_context = MagicMock()
    context.call_context.user = KAgentUser(user_id="A2A_USER_test-context-678")
    return context


@pytest.fixture
def mock_httpx_request():
    """Mock httpx.Request for header injection testing.

    Returns:
        httpx.Request: Mock request to http://test-agent:8080/message
    """
    request = httpx.Request("POST", "http://test-agent:8080/message")
    return request


@pytest.fixture
def mock_httpx_request_with_headers():
    """Mock httpx.Request with existing headers.

    Returns:
        httpx.Request: Request with X-User-ID already set
    """
    request = httpx.Request(
        "POST", "http://test-agent:8080/message", headers={"X-User-ID": "explicit-user@example.com"}
    )
    return request
