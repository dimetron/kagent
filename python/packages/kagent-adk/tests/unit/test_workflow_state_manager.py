"""Unit tests for WorkflowStateManager lock behavior.

This test suite specifically tests the asyncio.Lock implementation
in WorkflowStateManager for thread-safe concurrent operations:
- Lock acquisition and release
- Atomic updates to workflow state
- Lock serialization of concurrent writes
- Lock acquisition metrics
"""

import asyncio
import time
from datetime import datetime, timezone

import pytest

from kagent.adk.workflow.state import (
    SubAgentExecution,
    WorkflowState,
    WorkflowStateManager,
)


class TestWorkflowStateManagerLock:
    """Test WorkflowStateManager lock behavior."""

    @pytest.mark.asyncio
    async def test_lock_exists_and_is_asyncio_lock(self):
        """Verify WorkflowStateManager has an asyncio.Lock."""
        manager = WorkflowStateManager()

        # Verify lock exists
        assert hasattr(manager, "_lock")
        assert isinstance(manager._lock, asyncio.Lock)

    @pytest.mark.asyncio
    async def test_lock_serializes_concurrent_writes(self):
        """Verify lock ensures sequential execution of concurrent writes."""
        manager = WorkflowStateManager()
        manager.create_workflow(
            workflow_session_id="lock-test-123",
            user_id="user@example.com",
            agent_name="test-agent",
            namespace="default",
        )

        execution_order = []
        lock_held_count = 0

        async def write_with_lock_tracking(idx: int):
            """Track lock acquisition and execution order."""
            nonlocal lock_held_count

            async with manager._lock:
                # Track that we're inside the lock
                lock_held_count += 1
                current_held = lock_held_count

                # Simulate some work
                await asyncio.sleep(0.001)

                # Record execution order
                execution_order.append(idx)

                # Verify only one task holds lock at a time
                assert current_held == 1, f"Lock held by {current_held} tasks simultaneously"

                # Release lock (decrement counter)
                lock_held_count -= 1

        # Launch 10 concurrent tasks
        tasks = [write_with_lock_tracking(i) for i in range(10)]
        await asyncio.gather(*tasks)

        # Verify all 10 tasks executed
        assert len(execution_order) == 10

        # Verify lock was never held by multiple tasks
        assert lock_held_count == 0, "Lock still held after all tasks completed"

    @pytest.mark.asyncio
    async def test_update_output_concurrent_uses_lock(self):
        """Verify update_output_concurrent uses the lock."""
        manager = WorkflowStateManager()
        manager.create_workflow(
            workflow_session_id="lock-usage-test",
            user_id="user@example.com",
            agent_name="test-agent",
            namespace="default",
        )

        # Track lock acquisitions
        lock_acquisitions = []

        original_lock_acquire = manager._lock.acquire

        async def tracked_acquire():
            """Track when lock is acquired."""
            lock_acquisitions.append(time.perf_counter())
            return await original_lock_acquire()

        manager._lock.acquire = tracked_acquire

        # Perform concurrent writes
        async def write_output(idx: int):
            await manager.update_output_concurrent(
                workflow_session_id="lock-usage-test",
                key=f"key_{idx}",
                value=f"value_{idx}",
            )

        tasks = [write_output(i) for i in range(5)]
        await asyncio.gather(*tasks)

        # Verify lock was acquired 5 times (once per write)
        assert len(lock_acquisitions) == 5

    @pytest.mark.asyncio
    async def test_add_execution_concurrent_uses_lock(self):
        """Verify add_execution_concurrent uses the lock."""
        manager = WorkflowStateManager()
        manager.create_workflow(
            workflow_session_id="exec-lock-test",
            user_id="user@example.com",
            agent_name="test-agent",
            namespace="default",
        )

        lock_acquisitions = []

        original_lock_acquire = manager._lock.acquire

        async def tracked_acquire():
            """Track lock acquisitions."""
            lock_acquisitions.append(time.perf_counter())
            return await original_lock_acquire()

        manager._lock.acquire = tracked_acquire

        # Add execution records concurrently
        async def add_execution(idx: int):
            execution = SubAgentExecution(
                index=idx,
                agent_name=f"agent-{idx}",
                agent_namespace="default",
                session_id=f"exec-lock-test-sub-{idx}",
                output_key=f"result_{idx}",
                started_at=datetime.now(timezone.utc),
                completed_at=datetime.now(timezone.utc),
                status="success",
                output_size_bytes=100,
                completion_order=idx + 1,
            )
            await manager.add_execution_concurrent("exec-lock-test", execution)

        tasks = [add_execution(i) for i in range(5)]
        await asyncio.gather(*tasks)

        # Verify lock acquired 5 times
        assert len(lock_acquisitions) == 5

    @pytest.mark.asyncio
    async def test_lock_prevents_race_condition_in_state_data(self):
        """Verify lock prevents race conditions when updating state_data."""
        manager = WorkflowStateManager()
        workflow_state = manager.create_workflow(
            workflow_session_id="race-prevention-test",
            user_id="user@example.com",
            agent_name="test-agent",
            namespace="default",
        )

        # Counter for verifying atomic increments
        counter = {"value": 0}

        async def increment_counter_with_state_write(idx: int):
            """Simulate a race-prone operation."""
            async with manager._lock:
                # Read counter
                current_value = counter["value"]

                # Simulate processing delay
                await asyncio.sleep(0.001)

                # Increment and write
                counter["value"] = current_value + 1

                # Also write to state
                workflow_state.set_output(f"counter_{idx}", str(counter["value"]))

        # Launch 20 concurrent increments
        tasks = [increment_counter_with_state_write(i) for i in range(20)]
        await asyncio.gather(*tasks)

        # Verify counter incremented exactly 20 times (no race condition)
        assert counter["value"] == 20

        # Verify all state writes present
        assert len(workflow_state.state_data) == 20

    @pytest.mark.asyncio
    async def test_lock_acquisition_is_fast(self):
        """Verify lock acquisition is fast under no contention."""
        manager = WorkflowStateManager()
        manager.create_workflow(
            workflow_session_id="lock-speed-test",
            user_id="user@example.com",
            agent_name="test-agent",
            namespace="default",
        )

        # Measure lock acquisition time with no contention
        lock_times = []

        for _ in range(10):
            start_time = time.perf_counter()
            async with manager._lock:
                pass  # Minimal work
            end_time = time.perf_counter()
            lock_times.append(end_time - start_time)

        # Verify average lock acquisition time is fast (< 10μs)
        avg_lock_time_us = (sum(lock_times) / len(lock_times)) * 1_000_000
        assert avg_lock_time_us < 10.0, (
            f"Average lock acquisition time {avg_lock_time_us:.2f}μs exceeds 10μs"
        )

    @pytest.mark.asyncio
    async def test_lock_timeout_under_contention(self):
        """Verify lock wait times under concurrent contention."""
        manager = WorkflowStateManager()
        manager.create_workflow(
            workflow_session_id="lock-contention-test",
            user_id="user@example.com",
            agent_name="test-agent",
            namespace="default",
        )

        lock_wait_times = []

        async def measure_lock_wait(idx: int):
            """Measure time waiting for lock."""
            wait_start = time.perf_counter()

            async with manager._lock:
                wait_time = time.perf_counter() - wait_start
                lock_wait_times.append(wait_time)

                # Hold lock briefly
                await asyncio.sleep(0.001)

        # Launch 50 concurrent tasks (create contention)
        tasks = [measure_lock_wait(i) for i in range(50)]
        await asyncio.gather(*tasks)

        # Verify all 50 acquired lock
        assert len(lock_wait_times) == 50

        # Calculate statistics
        avg_wait_ms = (sum(lock_wait_times) / len(lock_wait_times)) * 1_000
        max_wait_ms = max(lock_wait_times) * 1_000

        # Verify reasonable wait times
        # With 50 tasks holding lock for 1ms each, average wait should be around 25ms
        # Max wait should be around 50ms (last task waits for all 49 others)
        assert avg_wait_ms < 50.0, f"Average wait {avg_wait_ms:.2f}ms exceeds 50ms"
        assert max_wait_ms < 100.0, f"Max wait {max_wait_ms:.2f}ms exceeds 100ms"

        print("\nLock contention stats (50 tasks):")
        print(f"  Average wait: {avg_wait_ms:.2f} ms")
        print(f"  Max wait: {max_wait_ms:.2f} ms")


class TestLockErrorHandling:
    """Test lock behavior in error scenarios."""

    @pytest.mark.asyncio
    async def test_lock_released_on_exception(self):
        """Verify lock is released even when exception occurs."""
        manager = WorkflowStateManager()
        manager.create_workflow(
            workflow_session_id="lock-exception-test",
            user_id="user@example.com",
            agent_name="test-agent",
            namespace="default",
        )

        async def operation_that_raises():
            """Operation that acquires lock then raises."""
            async with manager._lock:
                raise ValueError("Test exception")

        # Verify exception raised
        with pytest.raises(ValueError, match="Test exception"):
            await operation_that_raises()

        # Verify lock was released (can acquire again)
        lock_acquired = False

        async def try_acquire_lock():
            nonlocal lock_acquired
            async with manager._lock:
                lock_acquired = True

        await try_acquire_lock()
        assert lock_acquired, "Lock not released after exception"

    @pytest.mark.asyncio
    async def test_concurrent_operations_with_partial_failures(self):
        """Verify lock works correctly when some operations fail."""
        manager = WorkflowStateManager()
        workflow_state = manager.create_workflow(
            workflow_session_id="partial-fail-lock-test",
            user_id="user@example.com",
            agent_name="test-agent",
            namespace="default",
        )

        success_count = 0
        failure_count = 0

        async def operation_maybe_fail(idx: int):
            """Operation that may fail."""
            nonlocal success_count, failure_count

            try:
                async with manager._lock:
                    if idx % 3 == 0:
                        raise ValueError(f"Task {idx} failed")

                    workflow_state.set_output(f"key_{idx}", f"value_{idx}")
                    success_count += 1
            except ValueError:
                failure_count += 1

        # Run 12 operations (4 will fail, 8 will succeed)
        tasks = [operation_maybe_fail(i) for i in range(12)]
        await asyncio.gather(*tasks)

        # Verify expected counts
        assert success_count == 8
        assert failure_count == 4

        # Verify successful writes present
        assert len(workflow_state.state_data) == 8


class TestLockIsolation:
    """Test lock isolation between different workflow state managers."""

    @pytest.mark.asyncio
    async def test_different_managers_have_different_locks(self):
        """Verify each WorkflowStateManager has its own lock."""
        manager1 = WorkflowStateManager()
        manager2 = WorkflowStateManager()

        # Verify different lock instances
        assert manager1._lock is not manager2._lock

        # Verify locks are independent
        lock1_acquired = False
        lock2_acquired = False

        async def acquire_lock1():
            nonlocal lock1_acquired
            async with manager1._lock:
                lock1_acquired = True
                await asyncio.sleep(0.01)  # Hold lock

        async def acquire_lock2():
            nonlocal lock2_acquired
            async with manager2._lock:
                lock2_acquired = True

        # Both locks should be acquirable concurrently
        await asyncio.gather(acquire_lock1(), acquire_lock2())

        assert lock1_acquired and lock2_acquired

    @pytest.mark.asyncio
    async def test_lock_does_not_affect_different_workflows(self):
        """Verify lock serializes writes within same manager, but not across managers."""
        manager1 = WorkflowStateManager()
        manager2 = WorkflowStateManager()

        manager1.create_workflow("workflow-1", "user1", "agent1")
        manager2.create_workflow("workflow-2", "user2", "agent2")

        write_times = []

        async def write_to_manager(manager, workflow_id, idx):
            """Write to a specific manager."""
            start_time = time.perf_counter()
            await manager.update_output_concurrent(workflow_id, f"key_{idx}", f"value_{idx}")
            end_time = time.perf_counter()
            write_times.append((workflow_id, end_time - start_time))

        # Write to both managers concurrently
        tasks = []
        for i in range(10):
            tasks.append(write_to_manager(manager1, "workflow-1", i))
            tasks.append(write_to_manager(manager2, "workflow-2", i))

        await asyncio.gather(*tasks)

        # Verify all writes completed
        assert len(write_times) == 20

        # Verify both workflows have 10 outputs
        assert len(manager1.get_workflow("workflow-1").state_data) == 10
        assert len(manager2.get_workflow("workflow-2").state_data) == 10


class TestLockWithCompletionOrder:
    """Test lock behavior when tracking completion order."""

    @pytest.mark.asyncio
    async def test_atomic_completion_order_assignment(self):
        """Verify completion order assigned atomically under lock."""
        manager = WorkflowStateManager()
        workflow_state = manager.create_workflow(
            workflow_session_id="completion-order-lock-test",
            user_id="user@example.com",
            agent_name="test-agent",
            namespace="default",
        )

        completion_counter = 0
        assigned_orders = []

        async def assign_completion_order(idx: int):
            """Atomically assign completion order."""
            nonlocal completion_counter

            # Simulate random completion time
            await asyncio.sleep(0.001 * (idx % 5))

            async with manager._lock:
                completion_counter += 1
                current_order = completion_counter
                assigned_orders.append(current_order)

                # Write execution record
                execution = SubAgentExecution(
                    index=idx,
                    agent_name=f"agent-{idx}",
                    agent_namespace="default",
                    session_id=f"completion-order-lock-test-sub-{idx}",
                    output_key=f"result_{idx}",
                    started_at=datetime.now(timezone.utc),
                    completed_at=datetime.now(timezone.utc),
                    status="success",
                    output_size_bytes=100,
                    completion_order=current_order,
                )
                workflow_state.add_execution(execution)

        # Launch 30 concurrent agents
        tasks = [assign_completion_order(i) for i in range(30)]
        await asyncio.gather(*tasks)

        # Verify all 30 orders assigned
        assert len(assigned_orders) == 30

        # Verify all orders unique
        assert len(set(assigned_orders)) == 30

        # Verify orders are 1-30
        assert set(assigned_orders) == set(range(1, 31))

        # Verify execution records match
        exec_orders = [exec.completion_order for exec in workflow_state.sub_agent_executions]
        assert set(exec_orders) == set(range(1, 31))

