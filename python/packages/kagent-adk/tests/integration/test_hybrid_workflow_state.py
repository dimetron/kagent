"""Integration tests for hybrid workflow state management.

Tests the scenario where sequential workflow phases produce outputs that
are then accessible to parallel workflow phases (hybrid sequential-parallel workflows).

Test Approach:
1. Simulate sequential phase: Mock agent produces outputs in workflow state
2. Simulate parallel phase: Multiple agents read from workflow state
3. Verify: All parallel agents can access sequential outputs
"""

import asyncio
from datetime import datetime
from typing import Any, Dict, List
from unittest.mock import AsyncMock, Mock

import pytest

from kagent.adk.workflow.state import (
    SubAgentExecution,
    WorkflowState,
    WorkflowStateManager,
    WorkflowStatus,
)


# Mock agent class for testing
class MockSequentialAgent:
    """Mock agent that simulates sequential workflow behavior."""

    def __init__(self, name: str, output_key: str, output_value: str):
        self.name = name
        self.output_key = output_key
        self.output_value = output_value

    async def run_async(self, context) -> str:
        """Simulate agent execution."""
        await asyncio.sleep(0.01)  # Simulate work
        return self.output_value


class MockParallelAgent:
    """Mock agent that simulates parallel workflow behavior."""

    def __init__(self, name: str, output_key: str, reads_from_state: List[str]):
        self.name = name
        self.output_key = output_key
        self.reads_from_state = reads_from_state
        self.accessed_state: Dict[str, str] = {}

    async def run_async(self, context) -> str:
        """Simulate agent execution that reads from state."""
        await asyncio.sleep(0.01)  # Simulate work

        # Try to read expected keys from context session state
        for key in self.reads_from_state:
            if hasattr(context, "session") and hasattr(context.session, "state"):
                value = context.session.state.get(key)
                if value:
                    self.accessed_state[key] = value

        return f"Processed with inputs: {list(self.accessed_state.keys())}"


@pytest.fixture
def workflow_state_manager():
    """Create a WorkflowStateManager instance for testing."""
    return WorkflowStateManager()


@pytest.fixture
def mock_context():
    """Create a mock InvocationContext."""
    session = Mock()
    session.id = "test-session-123"
    session.state = {}
    session.user_id = "test-user"

    context = Mock()
    context.session = session
    context.user_id = "test-user"
    return context


@pytest.mark.asyncio
async def test_sequential_output_available_to_parallel(workflow_state_manager: WorkflowStateManager, mock_context):
    """Test that outputs from sequential phase are accessible to parallel agents.

    Scenario:
    1. Sequential phase: Agent produces output with key "validation_result"
    2. Parallel phase: 3 agents each try to read "validation_result"
    3. Verify: All 3 parallel agents can access the value
    """
    # Phase 1: Simulate sequential agent producing output
    workflow_state = workflow_state_manager.create_workflow(
        workflow_session_id="workflow-hybrid-1",
        user_id="test-user",
        agent_name="hybrid-workflow",
        namespace="default",
    )

    # Sequential agent produces output
    sequential_output_key = "validation_result"
    sequential_output_value = "Validation passed: All systems nominal"

    await workflow_state_manager.update_output_concurrent(
        workflow_session_id="workflow-hybrid-1",
        key=sequential_output_key,
        value=sequential_output_value,
    )

    # Inject workflow state into session (this is what parallel.py does)
    mock_context.session.state.update(workflow_state.state_data)

    # Phase 2: Create parallel agents that read from state
    parallel_agents = [
        MockParallelAgent(
            name=f"parallel-agent-{i}",
            output_key=f"result_{i}",
            reads_from_state=[sequential_output_key],
        )
        for i in range(3)
    ]

    # Execute parallel agents
    tasks = []
    for agent in parallel_agents:
        task = agent.run_async(mock_context)
        tasks.append(task)

    await asyncio.gather(*tasks)

    # Verify: All parallel agents accessed the sequential output
    for agent in parallel_agents:
        assert sequential_output_key in agent.accessed_state, (
            f"Agent {agent.name} did not access {sequential_output_key}"
        )
        assert agent.accessed_state[sequential_output_key] == sequential_output_value, (
            f"Agent {agent.name} got wrong value"
        )


@pytest.mark.asyncio
async def test_parallel_agents_see_same_state(workflow_state_manager: WorkflowStateManager, mock_context):
    """Test that all parallel agents see identical state from sequential phase.

    Scenario:
    1. Sequential phase: Produces 5 different outputs
    2. Parallel phase: 10 agents each try to read all 5 keys
    3. Verify: All 10 agents see the same 5 values
    """
    # Phase 1: Simulate sequential phase producing multiple outputs
    workflow_state = workflow_state_manager.create_workflow(
        workflow_session_id="workflow-hybrid-2",
        user_id="test-user",
        agent_name="hybrid-workflow",
        namespace="default",
    )

    # Sequential phase produces 5 outputs
    sequential_outputs = {
        "config": "enabled=true",
        "validation": "passed",
        "authentication": "oauth2",
        "region": "us-west-2",
        "version": "v1.2.3",
    }

    for key, value in sequential_outputs.items():
        await workflow_state_manager.update_output_concurrent(
            workflow_session_id="workflow-hybrid-2",
            key=key,
            value=value,
        )

    # Inject workflow state into session
    mock_context.session.state.update(workflow_state.state_data)

    # Phase 2: Create 10 parallel agents that all read the same state
    parallel_agents = [
        MockParallelAgent(
            name=f"parallel-agent-{i}",
            output_key=f"result_{i}",
            reads_from_state=list(sequential_outputs.keys()),
        )
        for i in range(10)
    ]

    # Execute parallel agents
    tasks = [agent.run_async(mock_context) for agent in parallel_agents]
    await asyncio.gather(*tasks)

    # Verify: All parallel agents see identical state
    for i, agent in enumerate(parallel_agents):
        assert len(agent.accessed_state) == len(sequential_outputs), (
            f"Agent {i} accessed {len(agent.accessed_state)} keys, expected {len(sequential_outputs)}"
        )

        for key, expected_value in sequential_outputs.items():
            assert key in agent.accessed_state, f"Agent {i} missing key {key}"
            assert agent.accessed_state[key] == expected_value, f"Agent {i} got wrong value for {key}"

    # Verify all agents got identical results
    first_agent_state = parallel_agents[0].accessed_state
    for agent in parallel_agents[1:]:
        assert agent.accessed_state == first_agent_state, f"Agent {agent.name} has different state than first agent"


@pytest.mark.asyncio
async def test_state_injection_with_multiple_keys(workflow_state_manager: WorkflowStateManager, mock_context):
    """Test state injection with complex state containing many keys.

    Scenario:
    1. Sequential phase: Produces 20 different outputs (complex state)
    2. Parallel phase: 5 agents each read subset of keys
    3. Verify: All agents can access their required keys correctly
    """
    # Phase 1: Simulate complex sequential phase
    workflow_state = workflow_state_manager.create_workflow(
        workflow_session_id="workflow-hybrid-3",
        user_id="test-user",
        agent_name="hybrid-workflow",
        namespace="default",
    )

    # Sequential phase produces 20 outputs
    sequential_outputs = {f"output_{i}": f"value_{i}" for i in range(20)}

    for key, value in sequential_outputs.items():
        await workflow_state_manager.update_output_concurrent(
            workflow_session_id="workflow-hybrid-3",
            key=key,
            value=value,
        )

    # Inject workflow state into session
    mock_context.session.state.update(workflow_state.state_data)

    # Phase 2: Create parallel agents that read different subsets
    parallel_agents = [
        MockParallelAgent(
            name=f"parallel-agent-{i}",
            output_key=f"result_{i}",
            reads_from_state=[f"output_{j}" for j in range(i * 4, (i + 1) * 4)],
        )
        for i in range(5)
    ]

    # Execute parallel agents
    tasks = [agent.run_async(mock_context) for agent in parallel_agents]
    await asyncio.gather(*tasks)

    # Verify: Each agent accessed its required subset
    for i, agent in enumerate(parallel_agents):
        expected_keys = [f"output_{j}" for j in range(i * 4, (i + 1) * 4)]
        assert len(agent.accessed_state) == len(expected_keys), (
            f"Agent {i} accessed {len(agent.accessed_state)} keys, expected {len(expected_keys)}"
        )

        for key in expected_keys:
            assert key in agent.accessed_state, f"Agent {i} missing key {key}"
            assert agent.accessed_state[key] == sequential_outputs[key], f"Agent {i} got wrong value for {key}"


@pytest.mark.asyncio
async def test_hybrid_workflow_with_partial_failure(workflow_state_manager: WorkflowStateManager, mock_context):
    """Test hybrid workflow where some parallel agents fail but state is still accessible.

    Scenario:
    1. Sequential phase: Produces outputs
    2. Parallel phase: Some agents fail, others succeed
    3. Verify: Successful agents still accessed sequential outputs
    """
    # Phase 1: Sequential phase
    workflow_state = workflow_state_manager.create_workflow(
        workflow_session_id="workflow-hybrid-4",
        user_id="test-user",
        agent_name="hybrid-workflow",
        namespace="default",
    )

    sequential_outputs = {
        "shared_config": "timeout=30s",
        "api_key": "secret-key-123",
    }

    for key, value in sequential_outputs.items():
        await workflow_state_manager.update_output_concurrent(
            workflow_session_id="workflow-hybrid-4",
            key=key,
            value=value,
        )

    # Inject workflow state into session
    mock_context.session.state.update(workflow_state.state_data)

    # Phase 2: Parallel agents (some will fail)
    class FailingMockAgent(MockParallelAgent):
        """Mock agent that fails during execution."""

        async def run_async(self, context):
            await asyncio.sleep(0.01)
            raise RuntimeError("Simulated agent failure")

    parallel_agents = [
        MockParallelAgent("agent-0", "result_0", list(sequential_outputs.keys())),
        FailingMockAgent("agent-1", "result_1", list(sequential_outputs.keys())),
        MockParallelAgent("agent-2", "result_2", list(sequential_outputs.keys())),
        FailingMockAgent("agent-3", "result_3", list(sequential_outputs.keys())),
        MockParallelAgent("agent-4", "result_4", list(sequential_outputs.keys())),
    ]

    # Execute parallel agents (with exception handling)
    tasks = [agent.run_async(mock_context) for agent in parallel_agents]
    results = await asyncio.gather(*tasks, return_exceptions=True)

    # Verify: Successful agents accessed sequential outputs
    successful_agents = [
        agent for agent, result in zip(parallel_agents, results, strict=False) if not isinstance(result, Exception)
    ]

    assert len(successful_agents) == 3, "Expected 3 successful agents"

    for agent in successful_agents:
        for key in sequential_outputs.keys():
            assert key in agent.accessed_state, f"Successful agent {agent.name} should have accessed {key}"


@pytest.mark.asyncio
async def test_empty_sequential_state(workflow_state_manager: WorkflowStateManager, mock_context):
    """Test parallel phase when sequential phase produces no outputs.

    Scenario:
    1. Sequential phase: Produces no outputs (empty state)
    2. Parallel phase: Agents try to read non-existent keys
    3. Verify: Agents handle missing keys gracefully
    """
    # Phase 1: Create workflow with no outputs
    workflow_state = workflow_state_manager.create_workflow(
        workflow_session_id="workflow-hybrid-5",
        user_id="test-user",
        agent_name="hybrid-workflow",
        namespace="default",
    )

    # No outputs added to workflow state

    # Inject empty workflow state into session
    mock_context.session.state.update(workflow_state.state_data)

    # Phase 2: Parallel agents try to read non-existent keys
    parallel_agents = [
        MockParallelAgent(
            name=f"parallel-agent-{i}",
            output_key=f"result_{i}",
            reads_from_state=["non_existent_key"],
        )
        for i in range(3)
    ]

    # Execute parallel agents
    tasks = [agent.run_async(mock_context) for agent in parallel_agents]
    await asyncio.gather(*tasks)

    # Verify: Agents handle missing keys gracefully (accessed_state is empty)
    for agent in parallel_agents:
        assert len(agent.accessed_state) == 0, f"Agent {agent.name} should not have accessed any state"


@pytest.mark.asyncio
async def test_state_accumulation_across_phases(workflow_state_manager: WorkflowStateManager, mock_context):
    """Test that state accumulates across multiple phases.

    Scenario:
    1. Sequential phase 1: Produces outputs A, B
    2. Parallel phase: Produces outputs C, D, E
    3. Sequential phase 2: Can access A, B, C, D, E
    """
    # Phase 1: First sequential phase
    workflow_state = workflow_state_manager.create_workflow(
        workflow_session_id="workflow-hybrid-6",
        user_id="test-user",
        agent_name="hybrid-workflow",
        namespace="default",
    )

    # Sequential phase 1 outputs
    await workflow_state_manager.update_output_concurrent(
        workflow_session_id="workflow-hybrid-6",
        key="output_a",
        value="value_a",
    )
    await workflow_state_manager.update_output_concurrent(
        workflow_session_id="workflow-hybrid-6",
        key="output_b",
        value="value_b",
    )

    # Inject into session
    mock_context.session.state.update(workflow_state.state_data)

    # Phase 2: Parallel phase adds more outputs
    await workflow_state_manager.update_output_concurrent(
        workflow_session_id="workflow-hybrid-6",
        key="output_c",
        value="value_c",
    )
    await workflow_state_manager.update_output_concurrent(
        workflow_session_id="workflow-hybrid-6",
        key="output_d",
        value="value_d",
    )
    await workflow_state_manager.update_output_concurrent(
        workflow_session_id="workflow-hybrid-6",
        key="output_e",
        value="value_e",
    )

    # Inject updated workflow state into session (simulating T020)
    mock_context.session.state.update(workflow_state.state_data)

    # Phase 3: Next sequential agent should see all outputs
    final_agent = MockParallelAgent(
        name="final-agent",
        output_key="final_result",
        reads_from_state=["output_a", "output_b", "output_c", "output_d", "output_e"],
    )

    await final_agent.run_async(mock_context)

    # Verify: Final agent sees all accumulated state
    expected_keys = ["output_a", "output_b", "output_c", "output_d", "output_e"]
    assert len(final_agent.accessed_state) == len(expected_keys), (
        f"Final agent should see all {len(expected_keys)} outputs"
    )

    for key in expected_keys:
        assert key in final_agent.accessed_state, f"Final agent missing key {key}"
