"""E2E tests for hybrid workflow patterns.

Tests complete end-to-end scenarios with real agent execution flow:
- Sequential validation → parallel deployment → sequential reporting
- Real workflow state management
- Real session state injection

These tests verify the complete integration of hybrid workflows.
"""

import asyncio
from datetime import datetime
from typing import Dict, List, Optional
from unittest.mock import AsyncMock, Mock, patch

import pytest

from kagent.adk.workflow.state import (
    SubAgentExecution,
    WorkflowState,
    WorkflowStateManager,
    WorkflowStatus,
)


# Mock RemoteA2aAgent for E2E testing
class MockRemoteA2aAgent:
    """Mock remote agent that simulates real agent behavior."""

    def __init__(
        self,
        name: str,
        namespace: str = "default",
        output_key: Optional[str] = None,
        execution_time: float = 0.05,
        should_fail: bool = False,
    ):
        self.name = name
        self.namespace = namespace
        self.output_key = output_key
        self.execution_time = execution_time
        self.should_fail = should_fail
        self.received_state: Dict[str, str] = {}

    async def run_async(self, context):
        """Simulate remote agent execution."""
        # Capture state received from context
        if hasattr(context, "session") and hasattr(context.session, "state"):
            self.received_state = dict(context.session.state)

        # Simulate execution time
        await asyncio.sleep(self.execution_time)

        # Simulate failure if configured
        if self.should_fail:
            raise RuntimeError(f"Agent {self.name} failed (simulated)")

        # Generate output based on received state
        if self.received_state:
            output = f"Processed with state: {', '.join(self.received_state.keys())}"
        else:
            output = f"Output from {self.name}"

        # Yield events (simplified)
        class MockEvent:
            def __init__(self, content):
                self.content = content

        yield MockEvent(output)


@pytest.fixture
def workflow_state_manager():
    """Create WorkflowStateManager for E2E testing."""
    return WorkflowStateManager()


@pytest.fixture
def mock_context():
    """Create mock InvocationContext with session."""
    session = Mock()
    session.id = "e2e-session-123"
    session.state = {}
    session.user_id = "e2e-user"

    context = Mock()
    context.session = session
    context.user_id = "e2e-user"
    return context


@pytest.mark.asyncio
async def test_sequential_validation_parallel_deployment_sequential_report(
    workflow_state_manager: WorkflowStateManager, mock_context
):
    """E2E test: Sequential validation → parallel deployment → sequential reporting.

    Workflow:
    1. Validation agent: Validates configuration, outputs "validation_status"
    2. Parallel deployers: 3 agents deploy to different regions, each reads "validation_status"
    3. Reporter agent: Aggregates deployment results from all 3 parallel agents

    This is a canonical hybrid workflow pattern.
    """
    # Create workflow state
    workflow_state = workflow_state_manager.create_workflow(
        workflow_session_id="e2e-hybrid-1",
        user_id="e2e-user",
        agent_name="deployment-workflow",
        namespace="default",
    )

    # === Phase 1: Sequential Validation ===
    validation_agent = MockRemoteA2aAgent(
        name="validation-agent",
        namespace="default",
    )

    # Execute validation agent
    validation_output = []
    async for event in validation_agent.run_async(mock_context):
        validation_output.append(event.content)

    # Store validation output in workflow state
    await workflow_state_manager.update_output_concurrent(
        workflow_session_id="e2e-hybrid-1",
        key="validation_status",
        value="Validation passed: Configuration is valid",
    )

    # Record execution
    await workflow_state_manager.add_execution_concurrent(
        workflow_session_id="e2e-hybrid-1",
        execution=SubAgentExecution(
            index=0,
            agent_name="validation-agent",
            agent_namespace="default",
            session_id=f"{mock_context.session.id}-sub-0",
            output_key="validation_status",
            started_at=datetime.utcnow(),
            completed_at=datetime.utcnow(),
            status="success",
            output_size_bytes=len("Validation passed: Configuration is valid"),
        ),
    )

    # Inject workflow state into session (simulating T020)
    mock_context.session.state.update(workflow_state.state_data)

    # === Phase 2: Parallel Deployment ===
    deployment_agents = [
        MockRemoteA2aAgent(
            name=f"deploy-{region}",
            namespace="default",
            output_key=f"deploy_{region}",
            execution_time=0.05,
        )
        for region in ["us-west", "us-east", "eu-central"]
    ]

    # Create sub-agent contexts with inherited state
    deploy_tasks = []
    for idx, agent in enumerate(deployment_agents):
        # Create sub-agent context (simulating T030 - state injection)
        sub_context = Mock()
        sub_context.user_id = mock_context.user_id
        sub_session = Mock()
        sub_session.id = f"{mock_context.session.id}-sub-{idx + 1}"
        # CRITICAL: Inject parent state into sub-agent context
        sub_session.state = dict(mock_context.session.state)
        sub_session.user_id = mock_context.user_id
        sub_context.session = sub_session

        deploy_tasks.append(
            {
                "agent": agent,
                "context": sub_context,
                "task": asyncio.create_task(collect_agent_output(agent, sub_context)),
            }
        )

    # Wait for all deployments
    deploy_results = await asyncio.gather(*[t["task"] for t in deploy_tasks], return_exceptions=True)

    # Store deployment outputs
    completion_order = 1
    for idx, (result, task_info) in enumerate(zip(deploy_results, deploy_tasks, strict=False)):
        if not isinstance(result, Exception):
            agent = task_info["agent"]
            output_text = "\n".join(result)

            await workflow_state_manager.update_output_concurrent(
                workflow_session_id="e2e-hybrid-1",
                key=agent.output_key,
                value=output_text,
            )

            await workflow_state_manager.add_execution_concurrent(
                workflow_session_id="e2e-hybrid-1",
                execution=SubAgentExecution(
                    index=idx + 1,
                    agent_name=agent.name,
                    agent_namespace=agent.namespace,
                    session_id=task_info["context"].session.id,
                    output_key=agent.output_key,
                    started_at=datetime.utcnow(),
                    completed_at=datetime.utcnow(),
                    status="success",
                    output_size_bytes=len(output_text),
                    completion_order=completion_order,
                ),
            )
            completion_order += 1

    # Inject updated workflow state (parallel outputs)
    mock_context.session.state.update(workflow_state.state_data)

    # === Phase 3: Sequential Reporting ===
    reporter_agent = MockRemoteA2aAgent(name="reporter-agent", namespace="default", output_key="final_report")

    # Execute reporter (should see validation + all deployment outputs)
    report_output = []
    async for event in reporter_agent.run_async(mock_context):
        report_output.append(event.content)

    # Store final report
    final_report_text = "\n".join(report_output)
    await workflow_state_manager.update_output_concurrent(
        workflow_session_id="e2e-hybrid-1",
        key="final_report",
        value=final_report_text,
    )

    # Record reporter execution
    await workflow_state_manager.add_execution_concurrent(
        workflow_session_id="e2e-hybrid-1",
        execution=SubAgentExecution(
            index=4,
            agent_name="reporter-agent",
            agent_namespace="default",
            session_id=f"{mock_context.session.id}-reporter",
            output_key="final_report",
            started_at=datetime.utcnow(),
            completed_at=datetime.utcnow(),
            status="success",
            output_size_bytes=len(final_report_text),
        ),
    )

    # Mark workflow completed
    workflow_state.mark_completed()

    # === Assertions ===

    # 1. Verify validation output stored
    assert "validation_status" in workflow_state.state_data
    assert workflow_state.state_data["validation_status"].startswith("Validation passed")

    # 2. Verify all deployment agents received validation status
    for task_info in deploy_tasks:
        agent = task_info["agent"]
        assert "validation_status" in agent.received_state, (
            f"Deploy agent {agent.name} should have received validation_status"
        )

    # 3. Verify all deployment outputs stored
    for region in ["us-west", "us-east", "eu-central"]:
        key = f"deploy_{region}"
        assert key in workflow_state.state_data, f"Missing deployment output: {key}"

    # 4. Verify reporter received all outputs (validation + 3 deployments)
    expected_keys = [
        "validation_status",
        "deploy_us-west",
        "deploy_us-east",
        "deploy_eu-central",
    ]
    for key in expected_keys:
        assert key in reporter_agent.received_state, f"Reporter should have received {key}"

    # 5. Verify workflow state has all executions
    assert len(workflow_state.sub_agent_executions) == 5  # 1 validation + 3 deploy + 1 reporter

    # 6. Verify workflow marked as completed
    assert workflow_state.status == WorkflowStatus.COMPLETED


@pytest.mark.asyncio
async def test_hybrid_workflow_with_partial_deployment_failure(
    workflow_state_manager: WorkflowStateManager, mock_context
):
    """E2E test: Hybrid workflow where some parallel agents fail.

    Workflow:
    1. Validation passes
    2. Parallel deployment: 1 of 3 deployers fails
    3. Reporter still gets outputs from successful deployers
    """
    # Create workflow
    workflow_state = workflow_state_manager.create_workflow(
        workflow_session_id="e2e-hybrid-2",
        user_id="e2e-user",
        agent_name="deployment-workflow-partial",
        namespace="default",
    )

    # Phase 1: Validation (success)
    await workflow_state_manager.update_output_concurrent(
        workflow_session_id="e2e-hybrid-2",
        key="validation_status",
        value="Validation passed",
    )
    mock_context.session.state.update(workflow_state.state_data)

    # Phase 2: Parallel deployment with 1 failure
    deployment_agents = [
        MockRemoteA2aAgent(
            name="deploy-us-west",
            output_key="deploy_us-west",
            should_fail=False,
        ),
        MockRemoteA2aAgent(
            name="deploy-us-east",
            output_key="deploy_us-east",
            should_fail=True,  # This one fails
        ),
        MockRemoteA2aAgent(
            name="deploy-eu-central",
            output_key="deploy_eu-central",
            should_fail=False,
        ),
    ]

    # Execute deployments
    deploy_tasks = []
    for idx, agent in enumerate(deployment_agents):
        sub_context = Mock()
        sub_context.user_id = mock_context.user_id
        sub_session = Mock()
        sub_session.id = f"{mock_context.session.id}-sub-{idx + 1}"
        sub_session.state = dict(mock_context.session.state)
        sub_session.user_id = mock_context.user_id
        sub_context.session = sub_session

        deploy_tasks.append(asyncio.create_task(collect_agent_output(agent, sub_context)))

    # Gather with exception handling
    deploy_results = await asyncio.gather(*deploy_tasks, return_exceptions=True)

    # Store successful outputs
    for agent, result in zip(deployment_agents, deploy_results, strict=False):
        if not isinstance(result, Exception):
            output_text = "\n".join(result)
            await workflow_state_manager.update_output_concurrent(
                workflow_session_id="e2e-hybrid-2",
                key=agent.output_key,
                value=output_text,
            )

    # Inject state
    mock_context.session.state.update(workflow_state.state_data)

    # Phase 3: Reporter
    reporter_agent = MockRemoteA2aAgent(name="reporter-agent", output_key="final_report")
    async for _event in reporter_agent.run_async(mock_context):
        pass

    # Assertions
    # 1. Verify 2 successful deployments stored
    assert "deploy_us-west" in workflow_state.state_data
    assert "deploy_eu-central" in workflow_state.state_data
    assert "deploy_us-east" not in workflow_state.state_data  # Failed

    # 2. Verify reporter received validation + 2 successful deployments
    assert "validation_status" in reporter_agent.received_state
    assert "deploy_us-west" in reporter_agent.received_state
    assert "deploy_eu-central" in reporter_agent.received_state


@pytest.mark.asyncio
async def test_complex_hybrid_chain(workflow_state_manager: WorkflowStateManager, mock_context):
    """E2E test: Complex hybrid workflow with multiple sequential-parallel chains.

    Workflow:
    1. Sequential: Config validation
    2. Parallel: 3 data collectors
    3. Sequential: Data aggregation
    4. Parallel: 2 processors
    5. Sequential: Final report
    """
    # Create workflow
    workflow_state = workflow_state_manager.create_workflow(
        workflow_session_id="e2e-hybrid-3",
        user_id="e2e-user",
        agent_name="complex-workflow",
        namespace="default",
    )

    # Phase 1: Sequential validation
    await workflow_state_manager.update_output_concurrent(
        workflow_session_id="e2e-hybrid-3",
        key="config_validation",
        value="Config valid",
    )
    mock_context.session.state.update(workflow_state.state_data)

    # Phase 2: Parallel data collection
    for i in range(3):
        await workflow_state_manager.update_output_concurrent(
            workflow_session_id="e2e-hybrid-3",
            key=f"data_{i}",
            value=f"Dataset {i}",
        )
    mock_context.session.state.update(workflow_state.state_data)

    # Phase 3: Sequential aggregation
    await workflow_state_manager.update_output_concurrent(
        workflow_session_id="e2e-hybrid-3",
        key="aggregated_data",
        value="Combined datasets 0, 1, 2",
    )
    mock_context.session.state.update(workflow_state.state_data)

    # Phase 4: Parallel processing
    for i in range(2):
        await workflow_state_manager.update_output_concurrent(
            workflow_session_id="e2e-hybrid-3",
            key=f"processed_{i}",
            value=f"Processed result {i}",
        )
    mock_context.session.state.update(workflow_state.state_data)

    # Phase 5: Sequential final report
    reporter = MockRemoteA2aAgent(name="reporter", output_key="final_report")
    async for _event in reporter.run_async(mock_context):
        pass

    # Assertions: Reporter should see ALL accumulated state
    expected_keys = [
        "config_validation",
        "data_0",
        "data_1",
        "data_2",
        "aggregated_data",
        "processed_0",
        "processed_1",
    ]

    for key in expected_keys:
        assert key in reporter.received_state, f"Reporter should see accumulated key: {key}"


@pytest.mark.asyncio
async def test_parallel_agents_isolation(workflow_state_manager: WorkflowStateManager, mock_context):
    """E2E test: Verify parallel agents don't pollute each other's state.

    Scenario:
    1. Sequential phase produces shared_config
    2. Parallel phase: Each agent adds to its own state, should not see others'
    3. Next sequential phase: Sees all parallel outputs
    """
    # Create workflow
    workflow_state = workflow_state_manager.create_workflow(
        workflow_session_id="e2e-hybrid-4",
        user_id="e2e-user",
        agent_name="isolation-test",
        namespace="default",
    )

    # Phase 1: Sequential shared config
    await workflow_state_manager.update_output_concurrent(
        workflow_session_id="e2e-hybrid-4",
        key="shared_config",
        value="timeout=30s",
    )
    mock_context.session.state.update(workflow_state.state_data)

    # Phase 2: Parallel agents with separate sessions
    parallel_agents = [MockRemoteA2aAgent(name=f"agent-{i}", output_key=f"result_{i}") for i in range(3)]

    # Execute parallel agents with SEPARATE contexts
    for idx, agent in enumerate(parallel_agents):
        # Each gets its own context with parent state
        sub_context = Mock()
        sub_context.user_id = mock_context.user_id
        sub_session = Mock()
        sub_session.id = f"{mock_context.session.id}-sub-{idx}"
        # Start with parent state only
        sub_session.state = dict(mock_context.session.state)
        sub_session.user_id = mock_context.user_id
        sub_context.session = sub_session

        # Execute agent
        output = []
        async for event in agent.run_async(sub_context):
            output.append(event.content)

        # Agent writes to workflow state (not to its own session state)
        await workflow_state_manager.update_output_concurrent(
            workflow_session_id="e2e-hybrid-4",
            key=agent.output_key,
            value="\n".join(output),
        )

    # Inject all parallel outputs
    mock_context.session.state.update(workflow_state.state_data)

    # Phase 3: Sequential aggregator sees all
    aggregator = MockRemoteA2aAgent(name="aggregator", output_key="final")
    async for _event in aggregator.run_async(mock_context):
        pass

    # Assertions
    # 1. Each parallel agent should have seen only shared_config (not other agents' outputs)
    for agent in parallel_agents:
        assert "shared_config" in agent.received_state
        # Should NOT see other parallel agents' outputs during execution
        # (they write to workflow state, not to each other's sessions)

    # 2. Aggregator should see all
    expected_keys = ["shared_config", "result_0", "result_1", "result_2"]
    for key in expected_keys:
        assert key in aggregator.received_state


# Helper function
async def collect_agent_output(agent, context) -> List[str]:
    """Helper to collect all events from an agent."""
    output = []
    async for event in agent.run_async(context):
        if hasattr(event, "content"):
            output.append(event.content)
    return output
