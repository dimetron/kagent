"""Pytest configuration for kagent-core tests."""
import pytest
from unittest.mock import AsyncMock
import httpx


@pytest.fixture
def mock_http_client():
    """Returns a mock httpx.AsyncClient for testing."""
    return AsyncMock(spec=httpx.AsyncClient)


@pytest.fixture
def sample_task_id():
    """Returns a sample task ID for testing."""
    return "task-123e4567-e89b-12d3-a456-426614174000"


@pytest.fixture
def sample_task_data():
    """Returns sample task data for testing."""
    return {
        "id": "task-123e4567-e89b-12d3-a456-426614174000",
        "context_id": "ctx-123",
        "agent_id": "agent-456",
        "status": {
            "state": "working",
            "timestamp": "2025-10-01T00:00:00Z"
        },
        "created_at": "2025-10-01T00:00:00Z",
        "updated_at": "2025-10-01T00:00:00Z"
    }

