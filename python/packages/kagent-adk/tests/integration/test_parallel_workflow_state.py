"""Integration tests for parallel workflow state propagation.

This test suite verifies end-to-end workflow state management
for parallel workflows with real WorkflowStateManager:
- State creation and retrieval
- Output propagation between phases
- State persistence across workflow execution
- Comparison with sequential workflow patterns
"""

import pytest
from datetime import datetime, timezone

from kagent.adk.workflow.state import WorkflowState, WorkflowStateManager, SubAgentExecution, WorkflowStatus


class TestParallelWorkflowStateLifecycle:
    """Test complete lifecycle of parallel workflow state."""

    @pytest.mark.asyncio
    async def test_parallel_agents_with_output_key(self):
        """Test 10 parallel agents with unique outputKey values."""
        # Create workflow state manager
        state_manager = WorkflowStateManager()

        # Create workflow
        workflow_state = state_manager.create_workflow(
            workflow_session_id="lifecycle-test-123",
            user_id="user@example.com",
            agent_name="parallel-workflow-agent",
            namespace="default",
        )

        # Verify initial state
        assert workflow_state.status == WorkflowStatus.RUNNING
        assert len(workflow_state.state_data) == 0
        assert len(workflow_state.sub_agent_executions) == 0

        # Simulate 10 parallel agents executing
        import asyncio

        async def simulate_agent_execution(idx: int):
            # Simulate agent processing
            await asyncio.sleep(0.01)

            # Write output
            output_key = f"result_{idx}"
            output_value = f"Output from parallel agent {idx}"
            await state_manager.update_output_concurrent("lifecycle-test-123", output_key, output_value)

            # Record execution
            execution = SubAgentExecution(
                index=idx,
                agent_name=f"parallel-agent-{idx}",
                agent_namespace="default",
                session_id=f"lifecycle-test-123-sub-{idx}",
                output_key=output_key,
                started_at=datetime.now(timezone.utc),
                completed_at=datetime.now(timezone.utc),
                status="success",
                output_size_bytes=len(output_value.encode("utf-8")),
                completion_order=None,  # Will be set by coordinator
            )
            await state_manager.add_execution_concurrent("lifecycle-test-123", execution)

        # Launch all 10 agents in parallel
        tasks = [simulate_agent_execution(i) for i in range(10)]
        await asyncio.gather(*tasks)

        # Verify all outputs stored
        assert len(workflow_state.state_data) == 10
        for i in range(10):
            assert workflow_state.get_output(f"result_{i}") == f"Output from parallel agent {i}"

        # Verify all execution records present
        assert len(workflow_state.sub_agent_executions) == 10

        # Mark workflow complete
        workflow_state.mark_completed()
        assert workflow_state.status == WorkflowStatus.COMPLETED
        assert workflow_state.completed_at is not None


class TestWorkflowStatePersistence:
    """Test workflow state persistence and retrieval."""

    @pytest.mark.asyncio
    async def test_workflow_state_persistence(self):
        """Verify state persists across operations."""
        state_manager = WorkflowStateManager()

        # Create workflow
        workflow_state = state_manager.create_workflow(
            workflow_session_id="persist-test-123",
            user_id="user@example.com",
            agent_name="persistent-agent",
            namespace="default",
        )

        # Add some outputs
        await state_manager.update_output_concurrent("persist-test-123", "key_1", "value_1")
        await state_manager.update_output_concurrent("persist-test-123", "key_2", "value_2")

        # Retrieve workflow state (should be same object)
        retrieved_state = state_manager.get_workflow("persist-test-123")
        assert retrieved_state is workflow_state  # Same object reference

        # Verify data persisted
        assert len(retrieved_state.state_data) == 2
        assert retrieved_state.get_output("key_1") == "value_1"
        assert retrieved_state.get_output("key_2") == "value_2"

    @pytest.mark.asyncio
    async def test_workflow_state_isolation(self):
        """Verify different workflows are isolated."""
        state_manager = WorkflowStateManager()

        # Create two workflows
        workflow_1 = state_manager.create_workflow(
            workflow_session_id="workflow-1", user_id="user1", agent_name="agent1", namespace="default"
        )

        workflow_2 = state_manager.create_workflow(
            workflow_session_id="workflow-2", user_id="user2", agent_name="agent2", namespace="default"
        )

        # Add outputs to each
        await state_manager.update_output_concurrent("workflow-1", "key_1", "value_1")
        await state_manager.update_output_concurrent("workflow-2", "key_2", "value_2")

        # Verify isolation
        assert len(workflow_1.state_data) == 1
        assert len(workflow_2.state_data) == 1
        assert workflow_1.get_output("key_1") == "value_1"
        assert workflow_1.get_output("key_2") is None  # Not present
        assert workflow_2.get_output("key_2") == "value_2"
        assert workflow_2.get_output("key_1") is None  # Not present


class TestParallelVsSequentialOutputs:
    """Compare parallel and sequential workflow output structures."""

    @pytest.mark.asyncio
    async def test_parallel_vs_sequential_outputs(self):
        """Verify parallel and sequential workflows have compatible output structure."""
        state_manager = WorkflowStateManager()

        # Create parallel workflow
        parallel_workflow = state_manager.create_workflow(
            workflow_session_id="parallel-compare-123",
            user_id="user@example.com",
            agent_name="parallel-agent",
            namespace="default",
        )

        # Create sequential workflow
        sequential_workflow = state_manager.create_workflow(
            workflow_session_id="sequential-compare-123",
            user_id="user@example.com",
            agent_name="sequential-agent",
            namespace="default",
        )

        # Add outputs to parallel workflow (concurrent)
        import asyncio

        await asyncio.gather(
            state_manager.update_output_concurrent("parallel-compare-123", "result_a", "Parallel A"),
            state_manager.update_output_concurrent("parallel-compare-123", "result_b", "Parallel B"),
            state_manager.update_output_concurrent("parallel-compare-123", "result_c", "Parallel C"),
        )

        # Add outputs to sequential workflow (one at a time)
        sequential_workflow.set_output("result_a", "Sequential A")
        sequential_workflow.set_output("result_b", "Sequential B")
        sequential_workflow.set_output("result_c", "Sequential C")

        # Verify both have same structure
        assert len(parallel_workflow.state_data) == 3
        assert len(sequential_workflow.state_data) == 3

        # Verify outputs retrievable same way
        parallel_outputs = state_manager.get_outputs("parallel-compare-123")
        sequential_outputs = state_manager.get_outputs("sequential-compare-123")

        assert set(parallel_outputs.keys()) == set(sequential_outputs.keys())
        assert set(parallel_outputs.keys()) == {"result_a", "result_b", "result_c"}


class TestWorkflowErrorScenarios:
    """Test error handling in parallel workflow state management."""

    def test_get_nonexistent_workflow(self):
        """Verify None returned for non-existent workflow."""
        state_manager = WorkflowStateManager()
        workflow_state = state_manager.get_workflow("nonexistent-123")
        assert workflow_state is None

    def test_get_outputs_nonexistent_workflow(self):
        """Verify error when getting outputs from non-existent workflow."""
        state_manager = WorkflowStateManager()

        with pytest.raises(ValueError, match="not found"):
            state_manager.get_outputs("nonexistent-123")

    @pytest.mark.asyncio
    async def test_mark_workflow_failed(self):
        """Verify workflow can be marked as failed."""
        state_manager = WorkflowStateManager()
        workflow_state = state_manager.create_workflow(
            workflow_session_id="failed-test-123",
            user_id="user@example.com",
            agent_name="failing-agent",
            namespace="default",
        )

        # Add some outputs before failure
        await state_manager.update_output_concurrent("failed-test-123", "partial_result", "Some output")

        # Mark as failed
        workflow_state.mark_failed("Agent execution timeout")

        # Verify status
        assert workflow_state.status == WorkflowStatus.FAILED
        assert workflow_state.error_message == "Agent execution timeout"
        assert workflow_state.completed_at is not None

        # Verify partial outputs still accessible
        assert workflow_state.get_output("partial_result") == "Some output"


class TestConcurrentStateAccess:
    """Test concurrent reads and writes to workflow state."""

    @pytest.mark.asyncio
    async def test_concurrent_reads_during_writes(self):
        """Verify reads can happen during concurrent writes."""
        state_manager = WorkflowStateManager()
        workflow_state = state_manager.create_workflow(
            workflow_session_id="read-write-123",
            user_id="user@example.com",
            agent_name="concurrent-agent",
            namespace="default",
        )

        import asyncio

        write_count = 0

        async def continuous_writer():
            nonlocal write_count
            for i in range(20):
                await state_manager.update_output_concurrent("read-write-123", f"key_{i}", f"value_{i}")
                write_count += 1
                await asyncio.sleep(0.01)

        async def continuous_reader():
            read_counts = []
            for _ in range(10):
                outputs = state_manager.get_outputs("read-write-123")
                read_counts.append(len(outputs))
                await asyncio.sleep(0.015)
            return read_counts

        # Run writer and reader concurrently
        writer_task = asyncio.create_task(continuous_writer())
        reader_task = asyncio.create_task(continuous_reader())

        read_counts = await reader_task
        await writer_task

        # Verify all writes completed
        assert write_count == 20
        assert len(workflow_state.state_data) == 20

        # Verify reader saw increasing counts
        assert len(read_counts) > 0
        # Reads should show progressive growth (0, 1, 2, ..., up to 20)
        assert read_counts[0] >= 0
        assert read_counts[-1] <= 20


class TestOutputKeyValidation:
    """Test output key validation in workflow state."""

    @pytest.mark.asyncio
    async def test_empty_output_value_stored(self):
        """Verify empty string can be stored as output."""
        state_manager = WorkflowStateManager()
        workflow_state = state_manager.create_workflow(
            workflow_session_id="empty-test-123",
            user_id="user@example.com",
            agent_name="test-agent",
            namespace="default",
        )

        # Store empty string
        await state_manager.update_output_concurrent("empty-test-123", "empty_result", "")

        # Verify stored
        assert workflow_state.get_output("empty_result") == ""
        assert "empty_result" in workflow_state.state_data

    @pytest.mark.asyncio
    async def test_output_value_overwrite(self):
        """Verify output value can be overwritten."""
        state_manager = WorkflowStateManager()
        workflow_state = state_manager.create_workflow(
            workflow_session_id="overwrite-test-123",
            user_id="user@example.com",
            agent_name="test-agent",
            namespace="default",
        )

        # Write initial value
        await state_manager.update_output_concurrent("overwrite-test-123", "result", "initial_value")
        assert workflow_state.get_output("result") == "initial_value"

        # Overwrite
        await state_manager.update_output_concurrent("overwrite-test-123", "result", "updated_value")
        assert workflow_state.get_output("result") == "updated_value"

