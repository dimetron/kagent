"""Unit tests for WorkflowState and related models."""

from datetime import datetime, timezone

import pytest

from kagent.adk.workflow.state import (
    SubAgentExecution,
    WorkflowState,
    WorkflowStateManager,
    WorkflowStatus,
)


class TestWorkflowStatus:
    """Tests for WorkflowStatus enum."""

    def test_enum_values(self):
        """Test that WorkflowStatus has expected values."""
        assert WorkflowStatus.RUNNING == "running"
        assert WorkflowStatus.COMPLETED == "completed"
        assert WorkflowStatus.FAILED == "failed"
        assert WorkflowStatus.CANCELLED == "cancelled"


class TestSubAgentExecution:
    """Tests for SubAgentExecution model."""

    def test_create_execution(self):
        """Test creating a SubAgentExecution record."""
        execution = SubAgentExecution(
            index=0,
            agent_name="code-writer-agent",
            agent_namespace="default",
            session_id="workflow-123-writer-0",
            output_key="generated_code",
            started_at=datetime.now(timezone.utc),
            status="success",
            output_size_bytes=1024,
        )
        
        assert execution.index == 0
        assert execution.agent_name == "code-writer-agent"
        assert execution.session_id == "workflow-123-writer-0"
        assert execution.output_key == "generated_code"
        assert execution.status == "success"
        assert execution.output_size_bytes == 1024

    def test_execution_with_error(self):
        """Test creating execution record with error."""
        execution = SubAgentExecution(
            index=1,
            agent_name="failing-agent",
            session_id="workflow-123-fail-1",
            started_at=datetime.now(timezone.utc),
            completed_at=datetime.now(timezone.utc),
            status="failed",
            error="Agent timed out",
        )
        
        assert execution.status == "failed"
        assert execution.error == "Agent timed out"


class TestWorkflowState:
    """Tests for WorkflowState model."""

    def test_create_workflow_state(self):
        """Test creating a new WorkflowState."""
        state = WorkflowState(
            workflow_session_id="workflow-abc123",
            user_id="user@example.com",
            agent_name="code-pipeline-agent",
            namespace="default",
        )
        
        assert state.workflow_session_id == "workflow-abc123"
        assert state.user_id == "user@example.com"
        assert state.agent_name == "code-pipeline-agent"
        assert state.namespace == "default"
        assert state.status == WorkflowStatus.RUNNING
        assert state.state_data == {}
        assert state.sub_agent_executions == []

    def test_get_output(self):
        """Test getting output value by key."""
        state = WorkflowState(
            workflow_session_id="workflow-123",
            user_id="user1",
            agent_name="test-agent",
        )
        state.state_data["code"] = "def hello(): pass"
        
        result = state.get_output("code")
        assert result == "def hello(): pass"

    def test_get_output_missing_key(self):
        """Test getting output for missing key returns None."""
        state = WorkflowState(
            workflow_session_id="workflow-123",
            user_id="user1",
            agent_name="test-agent",
        )
        
        result = state.get_output("nonexistent")
        assert result is None

    def test_set_output(self):
        """Test setting output value."""
        state = WorkflowState(
            workflow_session_id="workflow-123",
            user_id="user1",
            agent_name="test-agent",
        )
        
        state.set_output("generated_code", "def foo(): return 42")
        
        assert state.state_data["generated_code"] == "def foo(): return 42"
        # updated_at should be set
        assert state.updated_at is not None

    def test_set_output_exceeds_max_size(self):
        """Test that setting large output raises ValueError."""
        state = WorkflowState(
            workflow_session_id="workflow-123",
            user_id="user1",
            agent_name="test-agent",
        )
        
        # Create a value larger than 10MB
        large_value = "x" * (11 * 1024 * 1024)
        
        with pytest.raises(ValueError) as exc_info:
            state.set_output("large_data", large_value)
        
        assert "exceeds maximum size" in str(exc_info.value)

    def test_set_output_custom_max_size(self):
        """Test setting output with custom max size."""
        state = WorkflowState(
            workflow_session_id="workflow-123",
            user_id="user1",
            agent_name="test-agent",
        )
        
        # Should succeed with small max
        state.set_output("small", "data", max_size_bytes=100)
        
        # Should fail with tiny max
        with pytest.raises(ValueError):
            state.set_output("key", "x" * 1000, max_size_bytes=500)

    def test_add_execution(self):
        """Test adding sub-agent execution record."""
        state = WorkflowState(
            workflow_session_id="workflow-123",
            user_id="user1",
            agent_name="test-agent",
        )
        
        execution = SubAgentExecution(
            index=0,
            agent_name="sub-agent-1",
            session_id="workflow-123-sub1-0",
            output_key="result",
            started_at=datetime.now(timezone.utc),
            status="success",
        )
        
        state.add_execution(execution)
        
        assert len(state.sub_agent_executions) == 1
        assert state.sub_agent_executions[0] == execution
        # updated_at should be updated
        assert state.updated_at is not None

    def test_mark_completed(self):
        """Test marking workflow as completed."""
        state = WorkflowState(
            workflow_session_id="workflow-123",
            user_id="user1",
            agent_name="test-agent",
        )
        
        state.mark_completed()
        
        assert state.status == WorkflowStatus.COMPLETED
        assert state.completed_at is not None
        assert state.updated_at is not None

    def test_mark_failed(self):
        """Test marking workflow as failed."""
        state = WorkflowState(
            workflow_session_id="workflow-123",
            user_id="user1",
            agent_name="test-agent",
        )
        
        error_msg = "Sub-agent execution failed"
        state.mark_failed(error_msg)
        
        assert state.status == WorkflowStatus.FAILED
        assert state.error_message == error_msg
        assert state.completed_at is not None
        assert state.updated_at is not None

    def test_state_transitions(self):
        """Test workflow state transitions."""
        state = WorkflowState(
            workflow_session_id="workflow-123",
            user_id="user1",
            agent_name="test-agent",
        )
        
        # Initial state
        assert state.status == WorkflowStatus.RUNNING
        
        # Add some work
        state.set_output("result", "data")
        assert state.status == WorkflowStatus.RUNNING
        
        # Complete
        state.mark_completed()
        assert state.status == WorkflowStatus.COMPLETED

    def test_failed_state_transition(self):
        """Test transition to failed state."""
        state = WorkflowState(
            workflow_session_id="workflow-123",
            user_id="user1",
            agent_name="test-agent",
        )
        
        # Running -> Failed
        state.mark_failed("Error occurred")
        assert state.status == WorkflowStatus.FAILED
        assert state.error_message == "Error occurred"


class TestWorkflowStateManager:
    """Tests for WorkflowStateManager."""

    def test_create_workflow(self):
        """Test creating a new workflow state."""
        manager = WorkflowStateManager()
        
        state = manager.create_workflow(
            workflow_session_id="workflow-xyz",
            user_id="user@test.com",
            agent_name="my-workflow",
            namespace="production",
        )
        
        assert state.workflow_session_id == "workflow-xyz"
        assert state.user_id == "user@test.com"
        assert state.agent_name == "my-workflow"
        assert state.namespace == "production"
        
        # Should be cached
        cached_state = manager.get_workflow("workflow-xyz")
        assert cached_state == state

    def test_get_workflow_not_found(self):
        """Test getting non-existent workflow returns None."""
        manager = WorkflowStateManager()
        
        result = manager.get_workflow("nonexistent")
        assert result is None

    def test_update_output(self):
        """Test updating output in workflow state."""
        manager = WorkflowStateManager()
        manager.create_workflow(
            workflow_session_id="workflow-123",
            user_id="user1",
            agent_name="test",
        )
        
        manager.update_output("workflow-123", "code", "def test(): pass")
        
        state = manager.get_workflow("workflow-123")
        assert state.state_data["code"] == "def test(): pass"

    def test_update_output_workflow_not_found(self):
        """Test updating output for non-existent workflow raises error."""
        manager = WorkflowStateManager()
        
        with pytest.raises(ValueError) as exc_info:
            manager.update_output("nonexistent", "key", "value")
        
        assert "not found" in str(exc_info.value)

    def test_get_outputs(self):
        """Test getting all outputs from workflow."""
        manager = WorkflowStateManager()
        manager.create_workflow(
            workflow_session_id="workflow-123",
            user_id="user1",
            agent_name="test",
        )
        
        manager.update_output("workflow-123", "code", "def test(): pass")
        manager.update_output("workflow-123", "review", "Looks good")
        
        outputs = manager.get_outputs("workflow-123")
        assert outputs == {
            "code": "def test(): pass",
            "review": "Looks good",
        }

    def test_get_outputs_workflow_not_found(self):
        """Test getting outputs for non-existent workflow raises error."""
        manager = WorkflowStateManager()
        
        with pytest.raises(ValueError) as exc_info:
            manager.get_outputs("nonexistent")
        
        assert "not found" in str(exc_info.value)

    def test_cache_isolation(self):
        """Test that different workflow states are isolated."""
        manager = WorkflowStateManager()
        
        state1 = manager.create_workflow("workflow-1", "user1", "agent1")
        state2 = manager.create_workflow("workflow-2", "user2", "agent2")
        
        state1.set_output("key", "value1")
        state2.set_output("key", "value2")
        
        assert manager.get_workflow("workflow-1").state_data["key"] == "value1"
        assert manager.get_workflow("workflow-2").state_data["key"] == "value2"

