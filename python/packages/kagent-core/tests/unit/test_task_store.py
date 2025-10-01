"""Unit tests for KAgentTaskStore.

Tests the task store implementation that persists A2A tasks to KAgent via REST API.
"""

import pytest
from unittest.mock import AsyncMock, MagicMock
import httpx
from a2a.types import Task

from kagent.core.a2a import KAgentTaskStore


class TestKAgentTaskStoreInit:
    """Test KAgentTaskStore initialization."""
    
    def test_init_with_client(self, mock_http_client):
        """Test that task store initializes with httpx client."""
        task_store = KAgentTaskStore(mock_http_client)
        
        assert task_store.client is mock_http_client
    
    def test_init_requires_async_client(self):
        """Test that task store requires httpx.AsyncClient."""
        # This should work - just checking the type is accepted
        client = AsyncMock(spec=httpx.AsyncClient)
        task_store = KAgentTaskStore(client)
        
        assert task_store.client is client


class TestKAgentTaskStoreSave:
    """Test KAgentTaskStore.save() method."""
    
    @pytest.mark.asyncio
    async def test_save_task_success(self, mock_http_client, sample_task_data):
        """Test successfully saving a task."""
        # Setup
        task = Task.model_validate(sample_task_data)
        mock_response = MagicMock()
        mock_response.status_code = 200
        mock_http_client.post.return_value = mock_response
        
        task_store = KAgentTaskStore(mock_http_client)
        
        # Execute
        await task_store.save(task)
        
        # Verify
        mock_http_client.post.assert_called_once_with(
            "/api/tasks",
            json=task.model_dump()
        )
        mock_response.raise_for_status.assert_called_once()
    
    @pytest.mark.asyncio
    async def test_save_task_with_call_context(self, mock_http_client, sample_task_data):
        """Test saving a task with call_context parameter."""
        # Setup
        task = Task.model_validate(sample_task_data)
        mock_response = MagicMock()
        mock_response.status_code = 200
        mock_http_client.post.return_value = mock_response
        
        task_store = KAgentTaskStore(mock_http_client)
        call_context = {"user": "test-user"}
        
        # Execute
        await task_store.save(task, call_context)
        
        # Verify - call_context should be accepted but not used
        mock_http_client.post.assert_called_once_with(
            "/api/tasks",
            json=task.model_dump()
        )
    
    @pytest.mark.asyncio
    async def test_save_task_http_error(self, mock_http_client, sample_task_data):
        """Test handling HTTP errors when saving a task."""
        # Setup
        task = Task.model_validate(sample_task_data)
        mock_response = MagicMock()
        mock_response.status_code = 500
        mock_response.raise_for_status.side_effect = httpx.HTTPStatusError(
            "Server error",
            request=MagicMock(),
            response=mock_response
        )
        mock_http_client.post.return_value = mock_response
        
        task_store = KAgentTaskStore(mock_http_client)
        
        # Execute & Verify
        with pytest.raises(httpx.HTTPStatusError):
            await task_store.save(task)


class TestKAgentTaskStoreGet:
    """Test KAgentTaskStore.get() method."""
    
    @pytest.mark.asyncio
    async def test_get_task_success(self, mock_http_client, sample_task_id, sample_task_data):
        """Test successfully retrieving a task."""
        # Setup
        mock_response = MagicMock()
        mock_response.status_code = 200
        mock_response.json.return_value = sample_task_data
        mock_http_client.get.return_value = mock_response
        
        task_store = KAgentTaskStore(mock_http_client)
        
        # Execute
        result = await task_store.get(sample_task_id)
        
        # Verify
        assert result is not None
        assert isinstance(result, Task)
        assert result.id == sample_task_data["id"]
        mock_http_client.get.assert_called_once_with(f"/api/tasks/{sample_task_id}")
        mock_response.raise_for_status.assert_called_once()
    
    @pytest.mark.asyncio
    async def test_get_task_with_call_context(self, mock_http_client, sample_task_id, sample_task_data):
        """Test retrieving a task with call_context parameter."""
        # Setup
        mock_response = MagicMock()
        mock_response.status_code = 200
        mock_response.json.return_value = sample_task_data
        mock_http_client.get.return_value = mock_response
        
        task_store = KAgentTaskStore(mock_http_client)
        call_context = {"user": "test-user"}
        
        # Execute
        result = await task_store.get(sample_task_id, call_context)
        
        # Verify - call_context should be accepted but not used
        assert result is not None
        assert isinstance(result, Task)
        mock_http_client.get.assert_called_once_with(f"/api/tasks/{sample_task_id}")
    
    @pytest.mark.asyncio
    async def test_get_task_not_found(self, mock_http_client, sample_task_id):
        """Test retrieving a task that doesn't exist returns None."""
        # Setup
        mock_response = MagicMock()
        mock_response.status_code = 404
        mock_http_client.get.return_value = mock_response
        
        task_store = KAgentTaskStore(mock_http_client)
        
        # Execute
        result = await task_store.get(sample_task_id)
        
        # Verify
        assert result is None
        mock_http_client.get.assert_called_once_with(f"/api/tasks/{sample_task_id}")
        # raise_for_status should NOT be called for 404
        mock_response.raise_for_status.assert_not_called()
    
    @pytest.mark.asyncio
    async def test_get_task_http_error(self, mock_http_client, sample_task_id):
        """Test handling HTTP errors when retrieving a task."""
        # Setup
        mock_response = MagicMock()
        mock_response.status_code = 500
        mock_response.raise_for_status.side_effect = httpx.HTTPStatusError(
            "Server error",
            request=MagicMock(),
            response=mock_response
        )
        mock_http_client.get.return_value = mock_response
        
        task_store = KAgentTaskStore(mock_http_client)
        
        # Execute & Verify
        with pytest.raises(httpx.HTTPStatusError):
            await task_store.get(sample_task_id)


class TestKAgentTaskStoreDelete:
    """Test KAgentTaskStore.delete() method."""
    
    @pytest.mark.asyncio
    async def test_delete_task_success(self, mock_http_client, sample_task_id):
        """Test successfully deleting a task."""
        # Setup
        mock_response = MagicMock()
        mock_response.status_code = 200
        mock_http_client.delete.return_value = mock_response
        
        task_store = KAgentTaskStore(mock_http_client)
        
        # Execute
        await task_store.delete(sample_task_id)
        
        # Verify
        mock_http_client.delete.assert_called_once_with(f"/api/tasks/{sample_task_id}")
        mock_response.raise_for_status.assert_called_once()
    
    @pytest.mark.asyncio
    async def test_delete_task_with_call_context(self, mock_http_client, sample_task_id):
        """Test deleting a task with call_context parameter."""
        # Setup
        mock_response = MagicMock()
        mock_response.status_code = 200
        mock_http_client.delete.return_value = mock_response
        
        task_store = KAgentTaskStore(mock_http_client)
        call_context = {"user": "test-user"}
        
        # Execute
        await task_store.delete(sample_task_id, call_context)
        
        # Verify - call_context should be accepted but not used
        mock_http_client.delete.assert_called_once_with(f"/api/tasks/{sample_task_id}")
    
    @pytest.mark.asyncio
    async def test_delete_task_http_error(self, mock_http_client, sample_task_id):
        """Test handling HTTP errors when deleting a task."""
        # Setup
        mock_response = MagicMock()
        mock_response.status_code = 500
        mock_response.raise_for_status.side_effect = httpx.HTTPStatusError(
            "Server error",
            request=MagicMock(),
            response=mock_response
        )
        mock_http_client.delete.return_value = mock_response
        
        task_store = KAgentTaskStore(mock_http_client)
        
        # Execute & Verify
        with pytest.raises(httpx.HTTPStatusError):
            await task_store.delete(sample_task_id)


class TestKAgentTaskStoreIntegration:
    """Integration-style tests for KAgentTaskStore."""
    
    @pytest.mark.asyncio
    async def test_save_and_get_roundtrip(self, mock_http_client, sample_task_data):
        """Test saving and then retrieving a task."""
        # Setup
        task = Task.model_validate(sample_task_data)
        
        # Mock save response
        save_response = MagicMock()
        save_response.status_code = 200
        
        # Mock get response
        get_response = MagicMock()
        get_response.status_code = 200
        get_response.json.return_value = sample_task_data
        
        mock_http_client.post.return_value = save_response
        mock_http_client.get.return_value = get_response
        
        task_store = KAgentTaskStore(mock_http_client)
        
        # Execute
        await task_store.save(task)
        retrieved_task = await task_store.get(task.id)
        
        # Verify
        assert retrieved_task is not None
        assert retrieved_task.id == task.id
        assert retrieved_task.context_id == task.context_id
    
    @pytest.mark.asyncio
    async def test_save_get_delete_lifecycle(self, mock_http_client, sample_task_data):
        """Test complete task lifecycle: save, get, delete."""
        # Setup
        task = Task.model_validate(sample_task_data)
        
        save_response = MagicMock(status_code=200)
        get_response = MagicMock(status_code=200)
        get_response.json.return_value = sample_task_data
        delete_response = MagicMock(status_code=200)
        
        mock_http_client.post.return_value = save_response
        mock_http_client.get.return_value = get_response
        mock_http_client.delete.return_value = delete_response
        
        task_store = KAgentTaskStore(mock_http_client)
        
        # Execute
        await task_store.save(task)
        retrieved_task = await task_store.get(task.id)
        await task_store.delete(task.id)
        
        # Verify all operations were called
        mock_http_client.post.assert_called_once()
        mock_http_client.get.assert_called_once()
        mock_http_client.delete.assert_called_once()
        assert retrieved_task is not None

