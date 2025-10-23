"""Contract tests for parallel workflow state thread safety guarantees.

These tests validate the thread safety contract specified in:
specs/005-add-output-key/contracts/parallel-state-api.yaml

The contract defines four key guarantees:
1. Atomicity: Each write completes fully before the next
2. Isolation: Concurrent writes don't interfere
3. Ordering: Completion order is tracked and preserved
4. Durability: Outputs persist through failures
"""

import asyncio
from datetime import datetime, timezone

import pytest

from kagent.adk.workflow.state import SubAgentExecution, WorkflowStateManager, WorkflowStatus


class TestAtomicityGuarantee:
    """Test atomicity: Each state write operation is atomic.

    Guarantee: When multiple agents write concurrently, each write completes
    fully before the next begins. No partial writes or data corruption.
    """

    @pytest.mark.asyncio
    async def test_write_atomicity_no_partial_updates(self):
        """Verify writes are atomic - no partial updates visible."""
        state_manager = WorkflowStateManager()
        workflow_session_id = "atomicity-test"

        workflow_state = state_manager.create_workflow(
            workflow_session_id=workflow_session_id,
            user_id="test-user",
            agent_name="atomicity-workflow",
            namespace="default",
        )

        # Track all observed states during concurrent writes
        observed_states = []

        async def write_and_observe(agent_id: int):
            """Write output and observe state."""
            key = f"agent_{agent_id}"
            value = f"value_{agent_id}"

            # Before write
            state_before = len(workflow_state.state_data)

            # Write
            await state_manager.update_output_concurrent(workflow_session_id=workflow_session_id, key=key, value=value)

            # After write
            state_after = len(workflow_state.state_data)

            # State should increase by exactly 1 (atomic)
            observed_states.append(
                {
                    "agent_id": agent_id,
                    "before": state_before,
                    "after": state_after,
                    "delta": state_after - state_before,
                }
            )

        # 20 concurrent writes
        tasks = [write_and_observe(i) for i in range(20)]
        await asyncio.gather(*tasks)

        # Verify all deltas are exactly 1 (atomic increments)
        for obs in observed_states:
            assert obs["delta"] == 1, (
                f"Non-atomic write detected for agent {obs['agent_id']}: delta={obs['delta']} (expected 1)"
            )

        # Final state should have exactly 20 entries
        assert len(workflow_state.state_data) == 20

    @pytest.mark.asyncio
    async def test_no_data_corruption_under_contention(self):
        """Verify no data corruption when lock is heavily contended."""
        state_manager = WorkflowStateManager()
        workflow_session_id = "contention-test"

        workflow_state = state_manager.create_workflow(
            workflow_session_id=workflow_session_id,
            user_id="test-user",
            agent_name="contention-workflow",
            namespace="default",
        )

        # Use a consistent format to detect corruption
        async def write_structured_value(agent_id: int):
            """Write a structured value that can detect corruption."""
            key = f"agent_{agent_id:03d}"  # Zero-padded
            value = f"START|agent={agent_id:03d}|data={'X' * 100}|END"

            await state_manager.update_output_concurrent(workflow_session_id=workflow_session_id, key=key, value=value)

        # High contention: 50 agents
        tasks = [write_structured_value(i) for i in range(50)]
        await asyncio.gather(*tasks)

        # Verify all values are intact (no corruption)
        assert len(workflow_state.state_data) == 50

        for agent_id in range(50):
            key = f"agent_{agent_id:03d}"
            value = workflow_state.state_data[key]

            # Verify structure
            assert value.startswith("START|"), f"Corrupted start for {key}"
            assert value.endswith("|END"), f"Corrupted end for {key}"
            assert f"agent={agent_id:03d}" in value, f"Corrupted agent_id in {key}"
            assert "X" * 100 in value, f"Corrupted data section in {key}"


class TestIsolationGuarantee:
    """Test isolation: Concurrent writes do not interfere with each other.

    Guarantee: Agent A writing output_key="data_a" and Agent B writing
    output_key="data_b" simultaneously will both succeed without interference.
    """

    @pytest.mark.asyncio
    async def test_concurrent_writes_independent(self):
        """Verify concurrent writes to different keys don't interfere."""
        state_manager = WorkflowStateManager()
        workflow_session_id = "isolation-test"

        workflow_state = state_manager.create_workflow(
            workflow_session_id=workflow_session_id,
            user_id="test-user",
            agent_name="isolation-workflow",
            namespace="default",
        )

        # Track write times to detect interference
        write_times = {}

        async def timed_write(agent_id: int):
            """Write with timing information."""
            key = f"agent_{agent_id}"
            value = f"value_{agent_id}"

            start_time = asyncio.get_event_loop().time()

            await state_manager.update_output_concurrent(workflow_session_id=workflow_session_id, key=key, value=value)

            end_time = asyncio.get_event_loop().time()
            write_times[agent_id] = {"start": start_time, "end": end_time, "duration": end_time - start_time}

        # 30 concurrent writes
        tasks = [timed_write(i) for i in range(30)]
        await asyncio.gather(*tasks)

        # Verify all writes succeeded
        assert len(workflow_state.state_data) == 30

        # Verify each key has correct value (no interference)
        for agent_id in range(30):
            key = f"agent_{agent_id}"
            expected_value = f"value_{agent_id}"
            actual_value = workflow_state.state_data[key]

            assert actual_value == expected_value, (
                f"Value interference detected for agent {agent_id}: expected '{expected_value}', got '{actual_value}'"
            )

    @pytest.mark.asyncio
    async def test_no_cross_agent_pollution(self):
        """Verify agents don't see each other's intermediate states."""
        state_manager = WorkflowStateManager()
        workflow_session_id = "pollution-test"

        workflow_state = state_manager.create_workflow(
            workflow_session_id=workflow_session_id,
            user_id="test-user",
            agent_name="pollution-workflow",
            namespace="default",
        )

        # Each agent writes multiple times
        async def multi_write_agent(agent_id: int):
            """Agent that writes multiple outputs."""
            for i in range(5):
                key = f"agent_{agent_id}_output_{i}"
                value = f"agent_{agent_id}_value_{i}"

                await state_manager.update_output_concurrent(
                    workflow_session_id=workflow_session_id, key=key, value=value
                )

                await asyncio.sleep(0.0001)  # Small delay between writes

        # 10 agents, 5 writes each = 50 total
        tasks = [multi_write_agent(i) for i in range(10)]
        await asyncio.gather(*tasks)

        # Verify all 50 outputs present and correct
        assert len(workflow_state.state_data) == 50

        for agent_id in range(10):
            for output_id in range(5):
                key = f"agent_{agent_id}_output_{output_id}"
                expected_value = f"agent_{agent_id}_value_{output_id}"

                assert key in workflow_state.state_data, f"Missing {key}"
                assert workflow_state.state_data[key] == expected_value, f"Pollution detected: {key} has wrong value"


class TestOrderingGuarantee:
    """Test ordering: Completion order is tracked and preserved.

    Guarantee: The order in which agents complete is recorded in the
    completion_order field. This order is deterministic and preserved.
    """

    @pytest.mark.asyncio
    async def test_completion_order_unique_and_sequential(self):
        """Verify completion_order values are unique and sequential."""
        state_manager = WorkflowStateManager()
        workflow_session_id = "ordering-test"

        workflow_state = state_manager.create_workflow(
            workflow_session_id=workflow_session_id,
            user_id="test-user",
            agent_name="ordering-workflow",
            namespace="default",
        )

        completion_counter = 0

        async def agent_with_order_tracking(agent_id: int):
            """Agent that tracks completion order."""
            nonlocal completion_counter

            # Simulate work
            await asyncio.sleep(0.001 * (agent_id % 3))

            # Write output and track order
            key = f"agent_{agent_id}"
            value = f"value_{agent_id}"

            async with state_manager._lock:
                completion_counter += 1
                order = completion_counter

                workflow_state.set_output(key, value)

                execution = SubAgentExecution(
                    index=agent_id,
                    agent_name=f"agent-{agent_id}",
                    session_id=f"session-{agent_id}",
                    output_key=key,
                    started_at=datetime.now(timezone.utc),
                    completed_at=datetime.now(timezone.utc),
                    status="success",
                    completion_order=order,
                )
                workflow_state.add_execution(execution)

            return order

        # 25 concurrent agents
        tasks = [agent_with_order_tracking(i) for i in range(25)]
        completion_orders = await asyncio.gather(*tasks)

        # Verify uniqueness
        assert len(set(completion_orders)) == 25, "Duplicate completion orders"

        # Verify sequential (1, 2, 3, ..., 25)
        assert sorted(completion_orders) == list(range(1, 26)), "Completion orders not sequential"

        # Verify execution records match
        execution_orders = [e.completion_order for e in workflow_state.sub_agent_executions]
        assert sorted(execution_orders) == list(range(1, 26))

    @pytest.mark.asyncio
    async def test_completion_order_preserved_across_failures(self):
        """Verify completion order preserved even when some agents fail."""
        state_manager = WorkflowStateManager()
        workflow_session_id = "order-with-failures"

        workflow_state = state_manager.create_workflow(
            workflow_session_id=workflow_session_id,
            user_id="test-user",
            agent_name="failure-workflow",
            namespace="default",
        )

        completion_counter = 0

        async def agent_with_potential_failure(agent_id: int):
            """Agent that may fail but still tracks order."""
            nonlocal completion_counter

            # Agents 5, 10, 15 fail
            will_fail = agent_id in [5, 10, 15]

            if will_fail:
                # Record failure with completion order
                async with state_manager._lock:
                    completion_counter += 1
                    order = completion_counter

                    execution = SubAgentExecution(
                        index=agent_id,
                        agent_name=f"agent-{agent_id}",
                        session_id=f"session-{agent_id}",
                        output_key=None,
                        started_at=datetime.now(timezone.utc),
                        completed_at=datetime.now(timezone.utc),
                        status="failed",
                        error=f"Agent {agent_id} failed",
                        completion_order=order,
                    )
                    workflow_state.add_execution(execution)

                raise RuntimeError(f"Agent {agent_id} failed")

            # Success path
            key = f"agent_{agent_id}"
            value = f"value_{agent_id}"

            async with state_manager._lock:
                completion_counter += 1
                order = completion_counter

                workflow_state.set_output(key, value)

                execution = SubAgentExecution(
                    index=agent_id,
                    agent_name=f"agent-{agent_id}",
                    session_id=f"session-{agent_id}",
                    output_key=key,
                    started_at=datetime.now(timezone.utc),
                    completed_at=datetime.now(timezone.utc),
                    status="success",
                    completion_order=order,
                )
                workflow_state.add_execution(execution)

            return order

        # 20 agents (3 will fail)
        tasks = [agent_with_potential_failure(i) for i in range(20)]
        results = await asyncio.gather(*tasks, return_exceptions=True)

        # Verify 17 successful, 3 failed
        successes = [r for r in results if not isinstance(r, Exception)]
        failures = [r for r in results if isinstance(r, Exception)]

        assert len(successes) == 17
        assert len(failures) == 3

        # Verify completion order still sequential (1-20)
        all_orders = [e.completion_order for e in workflow_state.sub_agent_executions]
        assert sorted(all_orders) == list(range(1, 21))


class TestDurabilityGuarantee:
    """Test durability: Once written, outputs persist through failures.

    Guarantee: After an agent's output is written to workflow state,
    it remains accessible even if subsequent agents fail.
    """

    @pytest.mark.asyncio
    async def test_successful_outputs_persist_through_failures(self):
        """Verify successful outputs remain after subsequent failures."""
        state_manager = WorkflowStateManager()
        workflow_session_id = "durability-test"

        workflow_state = state_manager.create_workflow(
            workflow_session_id=workflow_session_id,
            user_id="test-user",
            agent_name="durability-workflow",
            namespace="default",
        )

        async def agent_with_sequential_failures(agent_id: int):
            """Agents fail after agent 10."""
            # First 10 succeed
            if agent_id < 10:
                key = f"agent_{agent_id}"
                value = f"value_{agent_id}"

                await state_manager.update_output_concurrent(
                    workflow_session_id=workflow_session_id, key=key, value=value
                )
            else:
                # Remaining agents fail
                raise RuntimeError(f"Agent {agent_id} failed")

        # 20 agents (10 succeed, 10 fail)
        tasks = [agent_with_sequential_failures(i) for i in range(20)]
        _ = await asyncio.gather(*tasks, return_exceptions=True)

        # Verify 10 successful outputs persist
        assert len(workflow_state.state_data) == 10

        # Verify all successful outputs are correct
        for i in range(10):
            key = f"agent_{i}"
            expected_value = f"value_{i}"
            assert workflow_state.state_data[key] == expected_value, (
                f"Output {key} not durable (wrong value or missing)"
            )

    @pytest.mark.asyncio
    async def test_partial_workflow_state_accessible(self):
        """Verify partial state accessible after timeout/cancellation."""
        state_manager = WorkflowStateManager()
        workflow_session_id = "partial-state-test"

        workflow_state = state_manager.create_workflow(
            workflow_session_id=workflow_session_id,
            user_id="test-user",
            agent_name="partial-workflow",
            namespace="default",
        )

        async def fast_or_slow_agent(agent_id: int):
            """Some agents are fast, others slow."""
            # First 5 agents are fast
            if agent_id < 5:
                key = f"agent_{agent_id}"
                value = f"value_{agent_id}"

                await state_manager.update_output_concurrent(
                    workflow_session_id=workflow_session_id, key=key, value=value
                )
            else:
                # Remaining agents are slow (simulate timeout)
                await asyncio.sleep(10)  # Will be cancelled

                key = f"agent_{agent_id}"
                value = f"value_{agent_id}"
                await state_manager.update_output_concurrent(
                    workflow_session_id=workflow_session_id, key=key, value=value
                )

        # Launch all agents with timeout
        tasks = [fast_or_slow_agent(i) for i in range(10)]

        try:
            # Timeout after 0.5 seconds (fast agents complete, slow ones don't)
            await asyncio.wait_for(asyncio.gather(*tasks, return_exceptions=True), timeout=0.5)
        except asyncio.TimeoutError:
            pass  # Expected

        # Verify fast agents' outputs are durable (persist despite timeout)
        assert len(workflow_state.state_data) == 5, f"Expected 5 durable outputs, got {len(workflow_state.state_data)}"

        for i in range(5):
            key = f"agent_{i}"
            assert key in workflow_state.state_data, f"Fast agent {i} output not durable"

    @pytest.mark.asyncio
    async def test_workflow_state_survives_manager_stress(self):
        """Verify state remains consistent under manager stress."""
        state_manager = WorkflowStateManager()
        workflow_session_id = "stress-durability"

        workflow_state = state_manager.create_workflow(
            workflow_session_id=workflow_session_id,
            user_id="test-user",
            agent_name="stress-workflow",
            namespace="default",
        )

        # Write in waves to stress manager
        for wave in range(5):

            async def wave_agent(agent_id: int, current_wave: int = wave):
                """Agent in current wave."""
                key = f"wave_{current_wave}_agent_{agent_id}"
                value = f"wave_{current_wave}_value_{agent_id}"

                await state_manager.update_output_concurrent(
                    workflow_session_id=workflow_session_id, key=key, value=value
                )

            # 10 agents per wave
            wave_tasks = [wave_agent(i) for i in range(10)]
            await asyncio.gather(*wave_tasks)

        # Verify all 50 outputs (5 waves * 10 agents) are durable
        assert len(workflow_state.state_data) == 50

        # Verify all outputs correct
        for wave in range(5):
            for agent_id in range(10):
                key = f"wave_{wave}_agent_{agent_id}"
                expected_value = f"wave_{wave}_value_{agent_id}"

                assert workflow_state.state_data[key] == expected_value, f"Output {key} not durable across waves"
