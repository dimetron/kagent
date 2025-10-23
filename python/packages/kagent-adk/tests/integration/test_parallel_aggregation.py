"""Integration tests for aggregator access to parallel workflow outputs.

This module tests User Story 2: Enable subsequent agents (aggregators) to access
all outputs from parallel sub-agents via session state injection.

Test Coverage:
- Aggregators can read all parallel outputs via session.state
- Parallel state available regardless of completion order
- Aggregators work with partial failures (some agents fail)
"""

import asyncio
import logging
from datetime import datetime, timezone
from typing import AsyncGenerator

import pytest
from google.adk.agents.base_agent import BaseAgent
from google.adk.agents.invocation_context import InvocationContext, Session
from google.adk.events import Event
from google.genai.types import Content, Part

from kagent.adk.agents.parallel import KAgentParallelAgent
from kagent.adk.workflow.state import WorkflowStateManager, SubAgentExecution
from tests.fixtures.memory_utils import create_test_invocation_context

logger = logging.getLogger(__name__)


class MockDataCollectorAgent(BaseAgent):
    """Mock agent that simulates data collection with configurable behavior."""

    # Configure model to allow extra fields
    model_config = {"extra": "allow"}

    def __init__(
        self,
        name: str,
        output_key: str,
        output_value: str,
        execution_time_ms: int = 100,
        should_fail: bool = False,
    ):
        """Initialize mock data collector agent.

        Args:
            name: Agent name
            output_key: Key to store output under
            output_value: Value to output
            execution_time_ms: Simulated execution time in milliseconds
            should_fail: Whether this agent should fail
        """
        super().__init__(name=name)
        self.output_key = output_key
        self.output_value = output_value
        self.execution_time_ms = execution_time_ms
        self.should_fail = should_fail

    async def run_async(self, context: InvocationContext) -> AsyncGenerator[Event, None]:
        """Execute mock agent with simulated delay."""
        # Simulate execution time
        await asyncio.sleep(self.execution_time_ms / 1000.0)

        if self.should_fail:
            raise RuntimeError(f"Simulated failure in {self.name}")

        # Yield output event
        yield Event(
            author=self.name,
            content=Content(
                parts=[Part(text=self.output_value)],
                role="model",
            ),
        )


class MockAggregatorAgent(BaseAgent):
    """Mock aggregator agent that accesses parallel outputs from session.state."""

    # Configure model to allow extra fields
    model_config = {"extra": "allow"}

    def __init__(self, name: str, expected_keys: list[str]):
        """Initialize mock aggregator agent.

        Args:
            name: Agent name
            expected_keys: List of outputKeys expected in session.state
        """
        super().__init__(name=name)
        self.expected_keys = expected_keys
        self.accessed_keys = []
        self.accessed_values = {}

    async def run_async(self, context: InvocationContext) -> AsyncGenerator[Event, None]:
        """Execute aggregator - access parallel outputs from session.state."""
        logger.info(f"Aggregator {self.name} starting with session.state keys: {list(context.session.state.keys())}")

        # Try to access each expected key
        for key in self.expected_keys:
            if key in context.session.state:
                self.accessed_keys.append(key)
                self.accessed_values[key] = context.session.state[key]
                logger.info(f"Aggregator accessed key '{key}': {context.session.state[key][:50]}...")
            else:
                logger.warning(f"Aggregator expected key '{key}' not found in session.state")

        # Generate aggregation result
        summary = (
            f"Aggregated {len(self.accessed_keys)}/{len(self.expected_keys)} outputs: {', '.join(self.accessed_keys)}"
        )
        yield Event(
            author=self.name,
            content=Content(
                parts=[Part(text=summary)],
                role="model",
            ),
        )


@pytest.mark.asyncio
async def test_aggregator_reads_all_parallel_outputs():
    """Test that aggregator can access all parallel outputs via session.state.

    Scenario:
    1. Create parallel workflow with 3 data collectors
    2. Execute parallel phase
    3. Verify workflow state contains all 3 outputs
    4. Create aggregator agent
    5. Inject workflow state into aggregator's session
    6. Verify aggregator can read all 3 outputs
    """
    # Step 1: Create 3 mock data collectors
    collectors = [
        MockDataCollectorAgent(
            name="collector_a",
            output_key="data_a",
            output_value="Data from region A",
            execution_time_ms=50,
        ),
        MockDataCollectorAgent(
            name="collector_b",
            output_key="data_b",
            output_value="Data from region B",
            execution_time_ms=30,  # Finishes first
        ),
        MockDataCollectorAgent(
            name="collector_c",
            output_key="data_c",
            output_value="Data from region C",
            execution_time_ms=70,  # Finishes last
        ),
    ]

    # Step 2: Create parallel workflow
    parallel_workflow = KAgentParallelAgent(
        name="data_collection_workflow",
        description="Collect data from 3 regions in parallel",
        sub_agents=collectors,
        max_workers=3,
    )

    # Step 3: Execute parallel phase
    parent_context = create_test_invocation_context(
        session_id="test_workflow_session",
        user_id="test_user",
        app_name="test_app",
    )

    # Collect events from parallel execution
    events = []
    async for event in parallel_workflow.run_async(parent_context):
        events.append(event)

    # Verify parallel execution completed
    assert len(events) >= 3, "Should generate at least 3 events (one per agent)"

    # Step 4: Verify workflow state contains all outputs
    # NOTE: In the actual implementation, we need to inject state into session
    # For now, we'll manually access workflow state to verify it was created
    # This test will FAIL until T020 is implemented

    # EXPECTED BEHAVIOR (after T020 implementation):
    # The parallel workflow should have injected state into parent_context.session.state
    # So aggregator can access it directly

    # For now, let's verify the workflow manager has the state
    # (This is a temporary check - after T020, state should be in session)

    # Step 5: Create aggregator agent
    aggregator = MockAggregatorAgent(
        name="data_aggregator",
        expected_keys=["data_a", "data_b", "data_c"],
    )

    # Step 6: Execute aggregator with updated session
    # NOTE: This will FAIL until T020 injects state into session
    aggregator_events = []
    async for event in aggregator.run_async(parent_context):
        aggregator_events.append(event)

    # Verify aggregator accessed all outputs
    assert len(aggregator.accessed_keys) == 3, f"Aggregator should access all 3 keys, got {aggregator.accessed_keys}"
    assert "data_a" in aggregator.accessed_keys
    assert "data_b" in aggregator.accessed_keys
    assert "data_c" in aggregator.accessed_keys

    # Verify values
    assert aggregator.accessed_values["data_a"] == "Data from region A"
    assert aggregator.accessed_values["data_b"] == "Data from region B"
    assert aggregator.accessed_values["data_c"] == "Data from region C"


@pytest.mark.asyncio
async def test_parallel_state_available_regardless_of_order():
    """Test that aggregator gets all outputs regardless of completion order.

    Scenario:
    1. Create parallel workflow with 5 agents with varying execution times
    2. Execute parallel phase (agents complete in random order)
    3. Verify aggregator receives all outputs in session.state
    """
    # Create 5 agents with different execution times
    collectors = [
        MockDataCollectorAgent(
            name=f"collector_{i}", output_key=f"result_{i}", output_value=f"Result {i}", execution_time_ms=50 + i * 20
        )
        for i in range(5)
    ]

    # Create parallel workflow
    parallel_workflow = KAgentParallelAgent(
        name="multi_collector_workflow",
        sub_agents=collectors,
        max_workers=5,
    )

    # Execute parallel phase
    parent_context = create_test_invocation_context(
        session_id="test_session_order",
        user_id="test_user",
        app_name="test_app",
    )

    events = []
    async for event in parallel_workflow.run_async(parent_context):
        events.append(event)

    # Create aggregator expecting all 5 outputs
    aggregator = MockAggregatorAgent(
        name="aggregator",
        expected_keys=[f"result_{i}" for i in range(5)],
    )

    # Execute aggregator
    async for event in aggregator.run_async(parent_context):
        pass

    # Verify all 5 outputs accessible
    assert len(aggregator.accessed_keys) == 5, f"Expected 5 keys, got {len(aggregator.accessed_keys)}"
    for i in range(5):
        assert f"result_{i}" in aggregator.accessed_keys


@pytest.mark.asyncio
async def test_aggregator_with_partial_failures():
    """Test that aggregator can access successful outputs even when some agents fail.

    Scenario:
    1. Create parallel workflow with 5 agents where 2 will fail
    2. Execute parallel phase (3 succeed, 2 fail)
    3. Verify aggregator receives outputs from 3 successful agents
    """
    # Create 5 agents: 3 succeed, 2 fail
    collectors = [
        MockDataCollectorAgent(name="collector_0", output_key="result_0", output_value="Success 0"),
        MockDataCollectorAgent(name="collector_1", output_key="result_1", output_value="Success 1"),
        MockDataCollectorAgent(
            name="collector_2",
            output_key="result_2",
            output_value="Will fail",
            should_fail=True,  # This one fails
        ),
        MockDataCollectorAgent(name="collector_3", output_key="result_3", output_value="Success 3"),
        MockDataCollectorAgent(
            name="collector_4",
            output_key="result_4",
            output_value="Will fail",
            should_fail=True,  # This one fails
        ),
    ]

    # Create parallel workflow
    parallel_workflow = KAgentParallelAgent(
        name="partial_failure_workflow",
        sub_agents=collectors,
        max_workers=5,
    )

    # Execute parallel phase
    parent_context = create_test_invocation_context(
        session_id="test_session_partial",
        user_id="test_user",
        app_name="test_app",
    )

    events = []
    async for event in parallel_workflow.run_async(parent_context):
        events.append(event)

    # Create aggregator expecting all 5 keys (but only 3 will be present)
    aggregator = MockAggregatorAgent(
        name="resilient_aggregator",
        expected_keys=["result_0", "result_1", "result_2", "result_3", "result_4"],
    )

    # Execute aggregator
    async for event in aggregator.run_async(parent_context):
        pass

    # Verify aggregator accessed only successful outputs
    assert len(aggregator.accessed_keys) == 3, f"Expected 3 successful outputs, got {len(aggregator.accessed_keys)}"
    assert "result_0" in aggregator.accessed_keys
    assert "result_1" in aggregator.accessed_keys
    assert "result_3" in aggregator.accessed_keys
    # result_2 and result_4 should NOT be present (agents failed)
    assert "result_2" not in aggregator.accessed_keys
    assert "result_4" not in aggregator.accessed_keys

    # Verify aggregator received correct values
    assert aggregator.accessed_values["result_0"] == "Success 0"
    assert aggregator.accessed_values["result_1"] == "Success 1"
    assert aggregator.accessed_values["result_3"] == "Success 3"
