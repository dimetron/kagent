"""Unit tests for thread-safe concurrent writes in parallel workflows.

This test suite verifies thread safety when multiple parallel agents
write to workflow state simultaneously:
- No data loss with concurrent writes
- Lock prevents race conditions
- Completion order tracking
- Execution records added atomically
"""

import asyncio
import pytest
from datetime import datetime, timezone

from kagent.adk.workflow.state import WorkflowState, WorkflowStateManager, SubAgentExecution


class TestConcurrentWrites:
    """Test thread-safe concurrent writes to workflow state."""

    @pytest.mark.asyncio
    async def test_concurrent_writes_no_data_loss(self):
        """Verify all outputs present when 50 agents write concurrently."""
        # Create state manager and workflow
        state_manager = WorkflowStateManager()
        workflow_state = state_manager.create_workflow(
            workflow_session_id="concurrent-50-123",
            user_id="user@example.com",
            agent_name="parallel-agent",
            namespace="default",
        )

        # Simulate 50 concurrent writes
        async def write_output(i: int):
            await state_manager.update_output_concurrent(
                workflow_session_id="concurrent-50-123", key=f"output_{i}", value=f"value_{i}"
            )

        # Launch all 50 writes concurrently
        tasks = [write_output(i) for i in range(50)]
        await asyncio.gather(*tasks)

        # Verify all 50 outputs present
        assert len(workflow_state.state_data) == 50

        # Verify all values correct
        for i in range(50):
            assert workflow_state.get_output(f"output_{i}") == f"value_{i}"

    @pytest.mark.asyncio
    async def test_lock_prevents_race_conditions(self):
        """Verify lock serializes writes - no interleaving."""
        state_manager = WorkflowStateManager()
        workflow_state = state_manager.create_workflow(
            workflow_session_id="race-test-123",
            user_id="user@example.com",
            agent_name="parallel-agent",
            namespace="default",
        )

        write_order = []

        # Track write order
        async def write_with_tracking(i: int):
            await state_manager.update_output_concurrent("race-test-123", f"key_{i}", f"value_{i}")
            write_order.append(i)

        # Launch concurrent writes
        tasks = [write_with_tracking(i) for i in range(10)]
        await asyncio.gather(*tasks)

        # Verify all writes completed
        assert len(write_order) == 10
        assert len(workflow_state.state_data) == 10

        # Verify no data corruption
        for i in range(10):
            assert workflow_state.get_output(f"key_{i}") == f"value_{i}"

    @pytest.mark.asyncio
    async def test_completion_order_tracking(self):
        """Verify completion_order field tracked correctly."""
        state_manager = WorkflowStateManager()
        workflow_state = state_manager.create_workflow(
            workflow_session_id="order-test-123",
            user_id="user@example.com",
            agent_name="parallel-agent",
            namespace="default",
        )

        completion_counter = 0
        completion_orders = []

        async def add_execution_with_order(idx: int):
            nonlocal completion_counter

            # Simulate agent execution
            await asyncio.sleep(0.001 * (10 - idx))  # Reverse order completion

            # Atomically increment completion order
            async with state_manager._lock:
                completion_counter += 1
                execution = SubAgentExecution(
                    index=idx,
                    agent_name=f"agent-{idx}",
                    agent_namespace="default",
                    session_id=f"order-test-123-sub-{idx}",
                    output_key=f"result_{idx}",
                    started_at=datetime.now(timezone.utc),
                    completed_at=datetime.now(timezone.utc),
                    status="success",
                    output_size_bytes=100,
                    completion_order=completion_counter,
                )
                workflow_state.add_execution(execution)
                completion_orders.append(completion_counter)

        # Launch 10 agents concurrently
        tasks = [add_execution_with_order(i) for i in range(10)]
        await asyncio.gather(*tasks)

        # Verify all have unique completion orders
        assert len(completion_orders) == 10
        assert len(set(completion_orders)) == 10  # All unique

        # Verify completion orders are 1-10
        assert set(completion_orders) == set(range(1, 11))

        # Verify execution records have correct completion orders
        for execution in workflow_state.sub_agent_executions:
            assert execution.completion_order is not None
            assert 1 <= execution.completion_order <= 10

    @pytest.mark.asyncio
    async def test_concurrent_execution_records(self):
        """Verify all execution records added with concurrent writes."""
        state_manager = WorkflowStateManager()
        workflow_state = state_manager.create_workflow(
            workflow_session_id="exec-concurrent-123",
            user_id="user@example.com",
            agent_name="parallel-agent",
            namespace="default",
        )

        async def add_execution(idx: int):
            execution = SubAgentExecution(
                index=idx,
                agent_name=f"agent-{idx}",
                agent_namespace="default",
                session_id=f"exec-concurrent-123-sub-{idx}",
                output_key=f"result_{idx}",
                started_at=datetime.now(timezone.utc),
                completed_at=datetime.now(timezone.utc),
                status="success",
                output_size_bytes=100,
                completion_order=idx + 1,
            )
            await state_manager.add_execution_concurrent("exec-concurrent-123", execution)

        # Launch 20 concurrent execution record additions
        tasks = [add_execution(i) for i in range(20)]
        await asyncio.gather(*tasks)

        # Verify all 20 execution records present
        assert len(workflow_state.sub_agent_executions) == 20

        # Verify all agents recorded
        agent_names = {exec.agent_name for exec in workflow_state.sub_agent_executions}
        assert len(agent_names) == 20
        for i in range(20):
            assert f"agent-{i}" in agent_names


class TestConcurrentErrorHandling:
    """Test error handling in concurrent write scenarios."""

    @pytest.mark.asyncio
    async def test_concurrent_write_to_nonexistent_workflow(self):
        """Verify KeyError raised for non-existent workflow."""
        state_manager = WorkflowStateManager()

        with pytest.raises(KeyError, match="not found"):
            await state_manager.update_output_concurrent("nonexistent-123", "key", "value")

    @pytest.mark.asyncio
    async def test_concurrent_write_exceeds_size_limit(self):
        """Verify ValueError raised when output exceeds 10MB."""
        state_manager = WorkflowStateManager()
        workflow_state = state_manager.create_workflow(
            workflow_session_id="size-limit-123",
            user_id="user@example.com",
            agent_name="parallel-agent",
            namespace="default",
        )

        # Try to write 11MB output
        large_output = "x" * (11 * 1024 * 1024)
        with pytest.raises(ValueError, match="exceeds maximum size"):
            await state_manager.update_output_concurrent("size-limit-123", "large_key", large_output)

        # Verify no partial data written
        assert workflow_state.get_output("large_key") is None

    @pytest.mark.asyncio
    async def test_concurrent_writes_with_failures(self):
        """Verify successful writes preserved when some fail."""
        state_manager = WorkflowStateManager()
        workflow_state = state_manager.create_workflow(
            workflow_session_id="partial-fail-123",
            user_id="user@example.com",
            agent_name="parallel-agent",
            namespace="default",
        )

        async def write_output_maybe_fail(i: int):
            if i % 3 == 0:
                # Every 3rd write is too large (will fail)
                large_output = "x" * (11 * 1024 * 1024)
                try:
                    await state_manager.update_output_concurrent("partial-fail-123", f"key_{i}", large_output)
                except ValueError:
                    pass  # Expected failure
            else:
                # Normal write (will succeed)
                await state_manager.update_output_concurrent("partial-fail-123", f"key_{i}", f"value_{i}")

        # Launch 12 writes (4 will fail, 8 will succeed)
        tasks = [write_output_maybe_fail(i) for i in range(12)]
        await asyncio.gather(*tasks)

        # Verify 8 successful writes present
        assert len(workflow_state.state_data) == 8

        # Verify failed writes not present
        for i in range(12):
            if i % 3 == 0:
                assert workflow_state.get_output(f"key_{i}") is None
            else:
                assert workflow_state.get_output(f"key_{i}") == f"value_{i}"


class TestLockPerformance:
    """Test performance characteristics of concurrent writes."""

    @pytest.mark.asyncio
    async def test_concurrent_writes_complete_quickly(self):
        """Verify concurrent writes have minimal overhead."""
        state_manager = WorkflowStateManager()
        workflow_state = state_manager.create_workflow(
            workflow_session_id="perf-test-123",
            user_id="user@example.com",
            agent_name="parallel-agent",
            namespace="default",
        )

        import time

        async def write_output(i: int):
            await state_manager.update_output_concurrent("perf-test-123", f"key_{i}", f"value_{i}")

        # Measure time for 100 concurrent writes
        start_time = time.perf_counter()
        tasks = [write_output(i) for i in range(100)]
        await asyncio.gather(*tasks)
        end_time = time.perf_counter()

        # Verify all writes succeeded
        assert len(workflow_state.state_data) == 100

        # Verify performance (100 writes should complete in < 1 second)
        elapsed = end_time - start_time
        assert elapsed < 1.0, f"100 concurrent writes took {elapsed:.3f}s (expected < 1s)"

