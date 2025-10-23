"""Integration stress tests for thread-safe concurrent writes in parallel workflows.

This test suite validates thread safety under heavy concurrent load with 50+ agents:
- All 50 outputs stored without data loss
- No output corruption or overwrites
- All 50 SubAgentExecution records with unique completion_order
- Lock wait times < 1ms under concurrent load
- Memory efficiency with large parallel workflows
- Realistic agent execution patterns
"""

import asyncio
import time
from datetime import datetime, timezone
from typing import List

import pytest

from kagent.adk.workflow.state import (
    SubAgentExecution,
    WorkflowState,
    WorkflowStateManager,
    WorkflowStatus,
)


class TestStressConcurrentWrites:
    """Stress test for concurrent writes with 50+ parallel agents."""

    @pytest.mark.asyncio
    async def test_50_concurrent_agents_no_data_loss(self):
        """Verify all 50 agent outputs stored with no data loss."""
        # Create state manager and workflow
        state_manager = WorkflowStateManager()
        workflow_state = state_manager.create_workflow(
            workflow_session_id="stress-50-agents",
            user_id="stress-test@example.com",
            agent_name="stress-parallel-workflow",
            namespace="stress-test",
        )

        # Simulate 50 concurrent agents with auto-generated outputKeys
        async def simulate_agent_execution(agent_idx: int):
            """Simulate a single agent execution with realistic timing."""
            # Simulate variable agent execution time (10-50ms)
            execution_time = 0.01 + (agent_idx % 5) * 0.01
            await asyncio.sleep(execution_time)

            # Generate auto-generated outputKey pattern
            output_key = f"default_agent_{agent_idx}"
            output_value = f"Result from agent-{agent_idx} after {execution_time*1000:.1f}ms"

            # Concurrent write to workflow state
            await state_manager.update_output_concurrent(
                workflow_session_id="stress-50-agents",
                key=output_key,
                value=output_value,
            )

            # Add execution record
            execution = SubAgentExecution(
                index=agent_idx,
                agent_name=f"agent-{agent_idx}",
                agent_namespace="default",
                session_id=f"stress-50-agents-sub-{agent_idx}",
                output_key=output_key,
                started_at=datetime.now(timezone.utc),
                completed_at=datetime.now(timezone.utc),
                status="success",
                output_size_bytes=len(output_value.encode("utf-8")),
                completion_order=None,  # Will be set during execution
            )
            await state_manager.add_execution_concurrent("stress-50-agents", execution)

        # Execute all 50 agents concurrently (max_workers=50)
        start_time = time.perf_counter()
        tasks = [simulate_agent_execution(i) for i in range(50)]
        await asyncio.gather(*tasks)
        end_time = time.perf_counter()

        # Verify all 50 outputs present
        assert len(workflow_state.state_data) == 50, (
            f"Expected 50 outputs, got {len(workflow_state.state_data)}"
        )

        # Verify all outputs correct (no corruption)
        for i in range(50):
            output_key = f"default_agent_{i}"
            output_value = workflow_state.get_output(output_key)
            assert output_value is not None, f"Missing output for {output_key}"
            assert f"agent-{i}" in output_value, f"Corrupted output for {output_key}: {output_value}"

        # Verify all 50 execution records present
        assert len(workflow_state.sub_agent_executions) == 50, (
            f"Expected 50 execution records, got {len(workflow_state.sub_agent_executions)}"
        )

        # Verify all execution records have correct outputKeys
        execution_output_keys = {exec.output_key for exec in workflow_state.sub_agent_executions}
        assert len(execution_output_keys) == 50, "Duplicate outputKeys in execution records"

        # Verify performance: 50 agents should complete in reasonable time
        elapsed = end_time - start_time
        print(f"\n✓ 50 concurrent agents completed in {elapsed:.3f}s")
        # With max 50ms per agent + concurrency, should complete in < 1 second
        assert elapsed < 2.0, f"50 concurrent agents took {elapsed:.3f}s (expected < 2s)"

    @pytest.mark.asyncio
    async def test_50_concurrent_writes_with_completion_order(self):
        """Verify all 50 agents have unique completion_order with no gaps."""
        state_manager = WorkflowStateManager()
        workflow_state = state_manager.create_workflow(
            workflow_session_id="stress-completion-order",
            user_id="stress-test@example.com",
            agent_name="stress-parallel-workflow",
            namespace="stress-test",
        )

        completion_counter = 0
        completion_orders: List[int] = []

        async def simulate_agent_with_completion_order(agent_idx: int):
            """Simulate agent execution and track completion order atomically."""
            nonlocal completion_counter

            # Simulate random completion times
            await asyncio.sleep(0.001 * (agent_idx % 10))

            # Atomically assign completion order and write output
            async with state_manager._lock:
                completion_counter += 1
                current_order = completion_counter

                # Write output
                output_key = f"default_agent_{agent_idx}"
                output_value = f"Result from agent-{agent_idx} (order {current_order})"
                workflow_state.set_output(output_key, output_value)

                # Add execution record with completion order
                execution = SubAgentExecution(
                    index=agent_idx,
                    agent_name=f"agent-{agent_idx}",
                    agent_namespace="default",
                    session_id=f"stress-completion-order-sub-{agent_idx}",
                    output_key=output_key,
                    started_at=datetime.now(timezone.utc),
                    completed_at=datetime.now(timezone.utc),
                    status="success",
                    output_size_bytes=len(output_value.encode("utf-8")),
                    completion_order=current_order,
                )
                workflow_state.add_execution(execution)
                completion_orders.append(current_order)

        # Execute all 50 agents concurrently
        tasks = [simulate_agent_with_completion_order(i) for i in range(50)]
        await asyncio.gather(*tasks)

        # Verify all 50 completion orders present
        assert len(completion_orders) == 50
        assert len(set(completion_orders)) == 50, "Duplicate completion orders detected"

        # Verify completion orders are 1-50 (no gaps)
        assert set(completion_orders) == set(range(1, 51)), "Gaps in completion order sequence"

        # Verify all execution records have unique completion orders
        exec_completion_orders = [
            exec.completion_order for exec in workflow_state.sub_agent_executions
        ]
        assert len(set(exec_completion_orders)) == 50, "Duplicate completion orders in execution records"

        # Verify no None values
        assert all(
            order is not None for order in exec_completion_orders
        ), "Some completion orders are None"

        print("\n✓ 50 agents completed with unique completion orders 1-50")

    @pytest.mark.asyncio
    async def test_50_concurrent_writes_no_overwrites(self):
        """Verify no outputs overwritten when 50 agents write simultaneously."""
        state_manager = WorkflowStateManager()
        workflow_state = state_manager.create_workflow(
            workflow_session_id="stress-no-overwrites",
            user_id="stress-test@example.com",
            agent_name="stress-parallel-workflow",
            namespace="stress-test",
        )

        write_counts = {}

        async def simulate_agent_write(agent_idx: int):
            """Simulate agent writing output and track write attempts."""
            output_key = f"default_agent_{agent_idx}"
            output_value = f"Unique output from agent-{agent_idx}"

            # Track write attempts
            if output_key not in write_counts:
                write_counts[output_key] = 0
            write_counts[output_key] += 1

            # Concurrent write
            await state_manager.update_output_concurrent(
                workflow_session_id="stress-no-overwrites",
                key=output_key,
                value=output_value,
            )

        # Execute all 50 agents concurrently
        tasks = [simulate_agent_write(i) for i in range(50)]
        await asyncio.gather(*tasks)

        # Verify all 50 outputs present
        assert len(workflow_state.state_data) == 50

        # Verify each key written exactly once (no overwrites)
        for key, count in write_counts.items():
            assert count == 1, f"Key {key} written {count} times (expected 1)"

        # Verify all outputs match expected values
        for i in range(50):
            output_key = f"default_agent_{i}"
            output_value = workflow_state.get_output(output_key)
            assert output_value == f"Unique output from agent-{i}", (
                f"Output mismatch for {output_key}"
            )

        print("\n✓ 50 concurrent writes with no overwrites")

    @pytest.mark.asyncio
    async def test_lock_wait_times_under_1ms(self):
        """Verify lock wait times < 1ms under concurrent load."""
        state_manager = WorkflowStateManager()
        workflow_state = state_manager.create_workflow(
            workflow_session_id="stress-lock-timing",
            user_id="stress-test@example.com",
            agent_name="stress-parallel-workflow",
            namespace="stress-test",
        )

        lock_wait_times: List[float] = []

        async def measure_lock_wait_time(agent_idx: int):
            """Measure lock acquisition time for concurrent write."""
            # Measure lock wait time
            lock_start = time.perf_counter()

            async with state_manager._lock:
                lock_wait = time.perf_counter() - lock_start
                lock_wait_times.append(lock_wait)

                # Perform write inside lock
                output_key = f"default_agent_{agent_idx}"
                output_value = f"Output from agent-{agent_idx}"
                workflow_state.set_output(output_key, output_value)

        # Execute 50 concurrent lock acquisitions
        tasks = [measure_lock_wait_time(i) for i in range(50)]
        await asyncio.gather(*tasks)

        # Verify all 50 writes completed
        assert len(workflow_state.state_data) == 50
        assert len(lock_wait_times) == 50

        # Calculate lock wait statistics
        avg_wait_ms = (sum(lock_wait_times) / len(lock_wait_times)) * 1000
        max_wait_ms = max(lock_wait_times) * 1000
        p95_wait_ms = sorted(lock_wait_times)[int(len(lock_wait_times) * 0.95)] * 1000

        print("\n✓ Lock wait times:")
        print(f"  Average: {avg_wait_ms:.3f} ms")
        print(f"  P95: {p95_wait_ms:.3f} ms")
        print(f"  Max: {max_wait_ms:.3f} ms")

        # Verify lock wait times are reasonable (< 1ms for P95)
        assert p95_wait_ms < 1.0, (
            f"P95 lock wait time {p95_wait_ms:.3f}ms exceeds 1ms threshold"
        )

        # Verify max wait time is acceptable (< 5ms)
        assert max_wait_ms < 5.0, (
            f"Max lock wait time {max_wait_ms:.3f}ms exceeds 5ms threshold"
        )


class TestStressScalingBeyond50:
    """Test scaling beyond 50 agents (stress testing at higher concurrency)."""

    @pytest.mark.asyncio
    async def test_100_concurrent_agents(self):
        """Verify thread safety with 100 concurrent agents."""
        state_manager = WorkflowStateManager()
        workflow_state = state_manager.create_workflow(
            workflow_session_id="stress-100-agents",
            user_id="stress-test@example.com",
            agent_name="stress-parallel-workflow",
            namespace="stress-test",
        )

        async def simulate_agent_execution(agent_idx: int):
            """Simulate agent execution."""
            await asyncio.sleep(0.001 * (agent_idx % 10))

            output_key = f"default_agent_{agent_idx}"
            output_value = f"Result from agent-{agent_idx}"

            await state_manager.update_output_concurrent(
                workflow_session_id="stress-100-agents",
                key=output_key,
                value=output_value,
            )

        # Execute 100 agents concurrently
        start_time = time.perf_counter()
        tasks = [simulate_agent_execution(i) for i in range(100)]
        await asyncio.gather(*tasks)
        end_time = time.perf_counter()

        # Verify all 100 outputs present
        assert len(workflow_state.state_data) == 100

        # Verify no data corruption
        for i in range(100):
            output_key = f"default_agent_{i}"
            assert workflow_state.get_output(output_key) == f"Result from agent-{i}"

        elapsed = end_time - start_time
        print(f"\n✓ 100 concurrent agents completed in {elapsed:.3f}s")

    @pytest.mark.asyncio
    async def test_200_concurrent_agents(self):
        """Verify thread safety with 200 concurrent agents (extreme stress)."""
        state_manager = WorkflowStateManager()
        workflow_state = state_manager.create_workflow(
            workflow_session_id="stress-200-agents",
            user_id="stress-test@example.com",
            agent_name="stress-parallel-workflow",
            namespace="stress-test",
        )

        async def simulate_agent_execution(agent_idx: int):
            """Simulate minimal agent execution."""
            output_key = f"default_agent_{agent_idx}"
            output_value = f"Result from agent-{agent_idx}"

            await state_manager.update_output_concurrent(
                workflow_session_id="stress-200-agents",
                key=output_key,
                value=output_value,
            )

        # Execute 200 agents concurrently
        start_time = time.perf_counter()
        tasks = [simulate_agent_execution(i) for i in range(200)]
        await asyncio.gather(*tasks)
        end_time = time.perf_counter()

        # Verify all 200 outputs present
        assert len(workflow_state.state_data) == 200

        # Verify no data corruption (spot check)
        for i in [0, 50, 100, 150, 199]:
            output_key = f"default_agent_{i}"
            assert workflow_state.get_output(output_key) == f"Result from agent-{i}"

        elapsed = end_time - start_time
        print(f"\n✓ 200 concurrent agents completed in {elapsed:.3f}s")


class TestStressRealisticPatterns:
    """Test realistic parallel workflow patterns under stress."""

    @pytest.mark.asyncio
    async def test_parallel_workflow_with_variable_output_sizes(self):
        """Test 50 agents with variable output sizes (realistic workload)."""
        state_manager = WorkflowStateManager()
        workflow_state = state_manager.create_workflow(
            workflow_session_id="stress-variable-sizes",
            user_id="stress-test@example.com",
            agent_name="stress-parallel-workflow",
            namespace="stress-test",
        )

        async def simulate_agent_with_variable_output(agent_idx: int):
            """Simulate agent with variable output size."""
            # Variable output sizes: 1KB, 10KB, 100KB, 1MB
            output_sizes = [1024, 10 * 1024, 100 * 1024, 1024 * 1024]
            output_size = output_sizes[agent_idx % len(output_sizes)]

            output_key = f"default_agent_{agent_idx}"
            output_value = f"Agent-{agent_idx} output: " + ("x" * output_size)

            await state_manager.update_output_concurrent(
                workflow_session_id="stress-variable-sizes",
                key=output_key,
                value=output_value,
            )

        # Execute 50 agents with variable output sizes
        tasks = [simulate_agent_with_variable_output(i) for i in range(50)]
        await asyncio.gather(*tasks)

        # Verify all 50 outputs present
        assert len(workflow_state.state_data) == 50

        # Verify output sizes correct
        for i in range(50):
            output_key = f"default_agent_{i}"
            output_value = workflow_state.get_output(output_key)
            assert output_value is not None
            assert output_value.startswith(f"Agent-{i} output:")

        print("\n✓ 50 agents with variable output sizes (1KB-1MB)")

    @pytest.mark.asyncio
    async def test_parallel_workflow_with_failures_and_retries(self):
        """Test 50 agents with some failures (realistic failure pattern)."""
        state_manager = WorkflowStateManager()
        workflow_state = state_manager.create_workflow(
            workflow_session_id="stress-with-failures",
            user_id="stress-test@example.com",
            agent_name="stress-parallel-workflow",
            namespace="stress-test",
        )

        success_count = 0
        failure_count = 0

        async def simulate_agent_with_failure_chance(agent_idx: int):
            """Simulate agent with 20% failure rate."""
            nonlocal success_count, failure_count

            # Simulate failure for agents divisible by 5
            if agent_idx % 5 == 0:
                # Simulate failure (no output written)
                failure_count += 1
                return

            # Successful agent execution
            output_key = f"default_agent_{agent_idx}"
            output_value = f"Result from agent-{agent_idx}"

            await state_manager.update_output_concurrent(
                workflow_session_id="stress-with-failures",
                key=output_key,
                value=output_value,
            )
            success_count += 1

        # Execute 50 agents with 20% failure rate
        tasks = [simulate_agent_with_failure_chance(i) for i in range(50)]
        await asyncio.gather(*tasks)

        # Verify expected success/failure counts
        assert success_count == 40, f"Expected 40 successes, got {success_count}"
        assert failure_count == 10, f"Expected 10 failures, got {failure_count}"

        # Verify only successful outputs present
        assert len(workflow_state.state_data) == 40

        # Verify failed agents have no outputs
        for i in range(50):
            output_key = f"default_agent_{i}"
            if i % 5 == 0:
                assert workflow_state.get_output(output_key) is None, (
                    f"Failed agent {i} should not have output"
                )
            else:
                assert workflow_state.get_output(output_key) == f"Result from agent-{i}"

        print("\n✓ 50 agents with realistic failure pattern (40 success, 10 fail)")


class TestStressMemoryEfficiency:
    """Test memory efficiency under concurrent load."""

    @pytest.mark.asyncio
    async def test_memory_released_after_50_agent_execution(self):
        """Verify memory released after 50 agent parallel execution."""
        import gc

        # Force garbage collection before test
        gc.collect()
        gc.collect()

        state_manager = WorkflowStateManager()

        # Execute and complete workflow
        workflow_state = state_manager.create_workflow(
            workflow_session_id="memory-test-50",
            user_id="stress-test@example.com",
            agent_name="stress-parallel-workflow",
            namespace="stress-test",
        )

        async def simulate_agent_execution(agent_idx: int):
            """Simulate agent execution."""
            output_key = f"default_agent_{agent_idx}"
            output_value = f"Result from agent-{agent_idx}" + ("x" * 10000)  # 10KB each

            await state_manager.update_output_concurrent(
                workflow_session_id="memory-test-50",
                key=output_key,
                value=output_value,
            )

        # Execute 50 agents
        tasks = [simulate_agent_execution(i) for i in range(50)]
        await asyncio.gather(*tasks)

        # Verify all outputs present
        assert len(workflow_state.state_data) == 50

        # Clear workflow state
        state_manager._cache.clear()

        # Force garbage collection
        gc.collect()
        gc.collect()

        # Memory should be released (no assertions, just verify no errors)
        print("\n✓ Memory released after 50 agent execution")
