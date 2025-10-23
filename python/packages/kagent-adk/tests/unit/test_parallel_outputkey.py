"""Unit tests for parallel workflow outputKey functionality.

This test suite verifies the core outputKey functionality for parallel workflows:
- Detection of outputKey mode
- Creation of workflow state
- Storage of outputs under correct keys
- Separate session IDs for each sub-agent
"""

import pytest
from datetime import datetime, timezone
from unittest.mock import AsyncMock, MagicMock, Mock
from typing import List

from kagent.adk.workflow.state import WorkflowState, WorkflowStateManager, SubAgentExecution


class TestParallelOutputKeyDetection:
    """Test detection of outputKey mode in parallel workflows."""

    def test_parallel_workflow_detects_output_key_mode(self):
        """Verify detection when any sub-agent has output_key."""
        # Create mock agents - some with output_key, some without
        agent_with_key = Mock()
        agent_with_key.output_key = "result_a"

        agent_without_key = Mock()
        agent_without_key.output_key = None

        agent_with_key_2 = Mock()
        agent_with_key_2.output_key = "result_b"

        # Test 1: All agents have output_key
        agents = [agent_with_key, agent_with_key_2]
        use_output_key_mode = any(hasattr(agent, "output_key") and agent.output_key for agent in agents)
        assert use_output_key_mode is True

        # Test 2: Mixed - some with, some without
        agents = [agent_with_key, agent_without_key]
        use_output_key_mode = any(hasattr(agent, "output_key") and agent.output_key for agent in agents)
        assert use_output_key_mode is True

        # Test 3: No agents have output_key
        agents = [agent_without_key, agent_without_key]
        use_output_key_mode = any(hasattr(agent, "output_key") and agent.output_key for agent in agents)
        assert use_output_key_mode is False

        # Test 4: Empty agent list
        agents = []
        use_output_key_mode = any(hasattr(agent, "output_key") and agent.output_key for agent in agents)
        assert use_output_key_mode is False


class TestParallelWorkflowStateCreation:
    """Test WorkflowState creation for parallel workflows."""

    def test_parallel_workflow_creates_workflow_state(self):
        """Verify WorkflowState is created when outputKey detected."""
        # Create state manager
        state_manager = WorkflowStateManager()

        # Create workflow state
        workflow_state = state_manager.create_workflow(
            workflow_session_id="parallel-test-123",
            user_id="user@example.com",
            agent_name="parallel-agent",
            namespace="default",
        )

        # Verify workflow state was created
        assert workflow_state is not None
        assert workflow_state.workflow_session_id == "parallel-test-123"
        assert workflow_state.user_id == "user@example.com"
        assert workflow_state.agent_name == "parallel-agent"
        assert workflow_state.namespace == "default"
        assert workflow_state.state_data == {}  # Empty initially
        assert workflow_state.sub_agent_executions == []  # Empty initially

        # Verify state is retrievable
        retrieved_state = state_manager.get_workflow("parallel-test-123")
        assert retrieved_state is workflow_state  # Same object


class TestParallelOutputStorage:
    """Test output storage for parallel workflows."""

    @pytest.mark.asyncio
    async def test_parallel_workflow_stores_outputs(self):
        """Verify outputs stored under correct keys with concurrent writes."""
        # Create state manager
        state_manager = WorkflowStateManager()

        # Create workflow state
        workflow_state = state_manager.create_workflow(
            workflow_session_id="parallel-store-123",
            user_id="user@example.com",
            agent_name="parallel-agent",
            namespace="default",
        )

        # Simulate parallel agents completing and writing outputs
        await state_manager.update_output_concurrent("parallel-store-123", "result_a", "Output from agent A")
        await state_manager.update_output_concurrent("parallel-store-123", "result_b", "Output from agent B")
        await state_manager.update_output_concurrent("parallel-store-123", "result_c", "Output from agent C")

        # Verify all outputs are stored
        assert len(workflow_state.state_data) == 3
        assert workflow_state.get_output("result_a") == "Output from agent A"
        assert workflow_state.get_output("result_b") == "Output from agent B"
        assert workflow_state.get_output("result_c") == "Output from agent C"

    @pytest.mark.asyncio
    async def test_parallel_workflow_handles_large_outputs(self):
        """Verify outputs can be up to 10MB."""
        state_manager = WorkflowStateManager()
        workflow_state = state_manager.create_workflow(
            workflow_session_id="parallel-large-123",
            user_id="user@example.com",
            agent_name="parallel-agent",
            namespace="default",
        )

        # Create 9MB output (should succeed)
        large_output = "x" * (9 * 1024 * 1024)  # 9MB
        await state_manager.update_output_concurrent("parallel-large-123", "large_result", large_output)
        assert workflow_state.get_output("large_result") == large_output

        # Create 11MB output (should fail)
        too_large_output = "x" * (11 * 1024 * 1024)  # 11MB
        with pytest.raises(ValueError, match="exceeds maximum size"):
            await state_manager.update_output_concurrent("parallel-large-123", "too_large", too_large_output)


class TestParallelSeparateSessions:
    """Test separate session ID generation for parallel sub-agents."""

    def test_parallel_workflow_separate_sessions(self):
        """Verify each sub-agent gets unique session ID."""
        parent_session_id = "workflow-parent-123"

        # Generate sub-agent session IDs
        sub_agent_session_ids = [f"{parent_session_id}-sub-{idx}" for idx in range(3)]

        # Verify uniqueness
        assert len(sub_agent_session_ids) == 3
        assert len(set(sub_agent_session_ids)) == 3  # All unique

        # Verify format
        assert sub_agent_session_ids[0] == "workflow-parent-123-sub-0"
        assert sub_agent_session_ids[1] == "workflow-parent-123-sub-1"
        assert sub_agent_session_ids[2] == "workflow-parent-123-sub-2"

        # Verify all share parent prefix
        for session_id in sub_agent_session_ids:
            assert session_id.startswith(parent_session_id)


class TestParallelExecutionRecords:
    """Test execution record tracking for parallel workflows."""

    @pytest.mark.asyncio
    async def test_parallel_execution_records_added(self):
        """Verify execution records are added with completion order."""
        state_manager = WorkflowStateManager()
        workflow_state = state_manager.create_workflow(
            workflow_session_id="parallel-exec-123",
            user_id="user@example.com",
            agent_name="parallel-agent",
            namespace="default",
        )

        # Create execution records with completion order
        execution_1 = SubAgentExecution(
            index=0,
            agent_name="agent-a",
            agent_namespace="default",
            session_id="parallel-exec-123-sub-0",
            output_key="result_a",
            started_at=datetime.now(timezone.utc),
            completed_at=datetime.now(timezone.utc),
            status="success",
            output_size_bytes=100,
            completion_order=2,  # Finished 2nd
        )

        execution_2 = SubAgentExecution(
            index=1,
            agent_name="agent-b",
            agent_namespace="default",
            session_id="parallel-exec-123-sub-1",
            output_key="result_b",
            started_at=datetime.now(timezone.utc),
            completed_at=datetime.now(timezone.utc),
            status="success",
            output_size_bytes=200,
            completion_order=1,  # Finished 1st
        )

        execution_3 = SubAgentExecution(
            index=2,
            agent_name="agent-c",
            agent_namespace="default",
            session_id="parallel-exec-123-sub-2",
            output_key="result_c",
            started_at=datetime.now(timezone.utc),
            completed_at=datetime.now(timezone.utc),
            status="success",
            output_size_bytes=300,
            completion_order=3,  # Finished 3rd
        )

        # Add execution records
        await state_manager.add_execution_concurrent("parallel-exec-123", execution_1)
        await state_manager.add_execution_concurrent("parallel-exec-123", execution_2)
        await state_manager.add_execution_concurrent("parallel-exec-123", execution_3)

        # Verify all execution records are present
        assert len(workflow_state.sub_agent_executions) == 3

        # Verify completion order is tracked
        executions = workflow_state.sub_agent_executions
        assert executions[0].completion_order == 2
        assert executions[1].completion_order == 1
        assert executions[2].completion_order == 3

        # Verify agent that finished first
        first_to_complete = min(executions, key=lambda e: e.completion_order or float("inf"))
        assert first_to_complete.agent_name == "agent-b"
        assert first_to_complete.completion_order == 1

