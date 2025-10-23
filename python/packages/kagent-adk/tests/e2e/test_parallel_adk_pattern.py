"""End-to-end tests for parallel data collection → aggregation pattern.

This module tests User Story 2 with real ADK agent patterns:
- Parallel data collection agents execute concurrently
- Outputs stored in workflow state
- Aggregator receives all outputs in session state
- Final result includes aggregated data

These tests use mock agents to avoid external dependencies while still
testing the full integration between parallel agents and aggregators.
"""

import asyncio
import logging
from typing import AsyncGenerator

import pytest
from google.adk.agents.base_agent import BaseAgent
from google.adk.agents.invocation_context import InvocationContext, Session
from google.adk.events import Event
from google.genai.types import Content, Part

from kagent.adk.agents.parallel import KAgentParallelAgent
from kagent.adk.agents.sequential import KAgentSequentialAgent
from tests.fixtures.memory_utils import create_test_invocation_context

logger = logging.getLogger(__name__)


class MockClusterMetricsAgent(BaseAgent):
    """Mock agent simulating Kubernetes cluster metrics collection."""

    # Configure model to allow extra fields
    model_config = {"extra": "allow"}

    def __init__(self, cluster_name: str, output_key: str):
        super().__init__(name=f"{cluster_name}_metrics_agent")
        self.cluster_name = cluster_name
        self.output_key = output_key

    async def run_async(self, context: InvocationContext) -> AsyncGenerator[Event, None]:
        """Simulate collecting cluster metrics."""
        # Simulate API call delay
        await asyncio.sleep(0.1)

        # Generate mock metrics
        metrics = {
            "cluster": self.cluster_name,
            "nodes": {"total": 10, "ready": 10},
            "pods": {"running": 50, "pending": 0},
            "cpu_usage": "45%",
            "memory_usage": "60%",
        }

        output = f"Cluster: {self.cluster_name}\nNodes: {metrics['nodes']['total']} ready\nPods: {metrics['pods']['running']} running\nCPU: {metrics['cpu_usage']}\nMemory: {metrics['memory_usage']}"

        yield Event(
            author=self.name,
            content=Content(parts=[Part(text=output)], role="model"),
        )


class MockMetricsAggregatorAgent(BaseAgent):
    """Mock agent that aggregates metrics from multiple clusters."""

    # Configure model to allow extra fields
    model_config = {"extra": "allow"}

    def __init__(self, expected_cluster_keys: list[str]):
        super().__init__(name="metrics_aggregator")
        self.expected_cluster_keys = expected_cluster_keys

    async def run_async(self, context: InvocationContext) -> AsyncGenerator[Event, None]:
        """Aggregate metrics from all clusters in session.state."""
        logger.info(f"Aggregator session.state keys: {list(context.session.state.keys())}")

        # Collect metrics from session state
        collected_metrics = {}
        for key in self.expected_cluster_keys:
            if key in context.session.state:
                collected_metrics[key] = context.session.state[key]
                logger.info(f"Aggregator found metrics for {key}")
            else:
                logger.warning(f"Aggregator missing metrics for {key}")

        # Generate aggregation report
        total_clusters = len(self.expected_cluster_keys)
        collected_clusters = len(collected_metrics)

        report = f"""CLUSTER MONITORING REPORT
========================
Clusters Monitored: {collected_clusters}/{total_clusters}

Details:
"""

        for key, metrics in collected_metrics.items():
            report += f"\n{key}:\n{metrics}\n"

        if collected_clusters < total_clusters:
            missing = set(self.expected_cluster_keys) - set(collected_metrics.keys())
            report += f"\n⚠️ Missing data from: {', '.join(missing)}"

        yield Event(
            author=self.name,
            content=Content(parts=[Part(text=report)], role="model"),
        )


@pytest.mark.asyncio
async def test_parallel_data_collection_aggregation_pattern():
    """Test the full parallel data collection → aggregation pattern.

    Scenario:
    1. Create 3 data collection agents (parallel)
    2. Create 1 aggregator agent (sequential)
    3. Combine in workflow: parallel → sequential
    4. Verify all outputs flow correctly
    """
    # Step 1: Create 3 mock cluster metrics agents
    cluster_agents = [
        MockClusterMetricsAgent(cluster_name="east_cluster", output_key="east_metrics"),
        MockClusterMetricsAgent(cluster_name="west_cluster", output_key="west_metrics"),
        MockClusterMetricsAgent(cluster_name="central_cluster", output_key="central_metrics"),
    ]

    # Step 2: Create parallel data collection workflow
    parallel_collector = KAgentParallelAgent(
        name="cluster_metrics_collector",
        description="Collect metrics from 3 clusters in parallel",
        sub_agents=cluster_agents,
        max_workers=3,
    )

    # Step 3: Create aggregator agent
    aggregator = MockMetricsAggregatorAgent(
        expected_cluster_keys=["east_metrics", "west_metrics", "central_metrics"],
    )

    # Step 4: Combine in sequential workflow (parallel → aggregator)
    combined_workflow = KAgentSequentialAgent(
        name="monitoring_workflow",
        description="Collect metrics in parallel, then aggregate",
        sub_agents=[parallel_collector, aggregator],
    )

    # Step 5: Execute combined workflow
    parent_context = create_test_invocation_context(
        session_id="e2e_test_session",
        user_id="test_user",
        app_name="test_app",
    )

    events = []
    async for event in combined_workflow.run_async(parent_context):
        events.append(event)
        logger.info(
            f"Event from {event.author}: {event.content.parts[0].text[:100] if event.content and event.content.parts else 'no content'}..."
        )

    # Step 6: Verify results
    assert len(events) > 0, "Should generate events"

    # Verify we got events from all agents
    event_authors = [event.author for event in events]
    assert "east_cluster_metrics_agent" in event_authors
    assert "west_cluster_metrics_agent" in event_authors
    assert "central_cluster_metrics_agent" in event_authors
    assert "metrics_aggregator" in event_authors

    # Verify aggregator generated final report
    aggregator_events = [e for e in events if e.author == "metrics_aggregator"]
    assert len(aggregator_events) > 0, "Aggregator should generate events"

    # Verify aggregator report contains data from all clusters
    aggregator_output = "".join([e.content.parts[0].text for e in aggregator_events if e.content and e.content.parts])
    assert "east_cluster" in aggregator_output
    assert "west_cluster" in aggregator_output
    assert "central_cluster" in aggregator_output
    assert "Clusters Monitored: 3/3" in aggregator_output


@pytest.mark.asyncio
async def test_hybrid_workflow_with_multiple_phases():
    """Test complex workflow: sequential validation → parallel deployment → sequential reporting.

    Scenario:
    1. Validation agent (sequential)
    2. Parallel deployment agents
    3. Reporting agent (sequential)
    4. Verify state flows through all phases
    """

    class MockValidationAgent(BaseAgent):
        """Mock validation agent that produces validation_result."""

        def __init__(self):
            super().__init__(name="validation_agent")
            self.output_key = "validation_result"

        async def run_async(self, context: InvocationContext) -> AsyncGenerator[Event, None]:
            yield Event(
                author=self.name,
                content=Content(parts=[Part(text="Validation: PASSED")], role="model"),
            )

    class MockDeploymentAgent(BaseAgent):
        """Mock deployment agent that reads validation_result and deploys."""

        def __init__(self, service_name: str, output_key: str):
            super().__init__(name=f"deploy_{service_name}")
            self.service_name = service_name
            self.output_key = output_key

        async def run_async(self, context: InvocationContext) -> AsyncGenerator[Event, None]:
            # Check if validation passed
            validation = context.session.state.get("validation_result", "UNKNOWN")
            logger.info(f"Deployment agent {self.service_name} sees validation: {validation}")

            await asyncio.sleep(0.05)  # Simulate deployment

            result = f"Deployed {self.service_name} (validation: {validation})"
            yield Event(
                author=self.name,
                content=Content(parts=[Part(text=result)], role="model"),
            )

    class MockReportingAgent(BaseAgent):
        """Mock reporting agent that aggregates all deployment results."""

        def __init__(self):
            super().__init__(name="reporting_agent")

        async def run_async(self, context: InvocationContext) -> AsyncGenerator[Event, None]:
            # Access validation result
            validation = context.session.state.get("validation_result", "UNKNOWN")

            # Access deployment results
            service_a = context.session.state.get("deploy_service_a", "MISSING")
            service_b = context.session.state.get("deploy_service_b", "MISSING")
            service_c = context.session.state.get("deploy_service_c", "MISSING")

            report = f"""DEPLOYMENT REPORT
Validation: {validation}
Service A: {service_a}
Service B: {service_b}
Service C: {service_c}
"""
            yield Event(
                author=self.name,
                content=Content(parts=[Part(text=report)], role="model"),
            )

    # Step 1: Create validation agent
    validation_agent = MockValidationAgent()

    # Step 2: Create parallel deployment agents
    deployment_agents = [
        MockDeploymentAgent("service_a", "deploy_service_a"),
        MockDeploymentAgent("service_b", "deploy_service_b"),
        MockDeploymentAgent("service_c", "deploy_service_c"),
    ]

    parallel_deployer = KAgentParallelAgent(
        name="parallel_deployer",
        sub_agents=deployment_agents,
        max_workers=3,
    )

    # Step 3: Create reporting agent
    reporting_agent = MockReportingAgent()

    # Step 4: Combine in sequential workflow
    hybrid_workflow = KAgentSequentialAgent(
        name="hybrid_deployment_workflow",
        sub_agents=[validation_agent, parallel_deployer, reporting_agent],
    )

    # Step 5: Execute
    parent_context = create_test_invocation_context(
        session_id="hybrid_test_session",
        user_id="test_user",
        app_name="test_app",
    )

    events = []
    async for event in hybrid_workflow.run_async(parent_context):
        events.append(event)

    # Step 6: Verify results
    # Verify all agents executed
    event_authors = [event.author for event in events]
    assert "validation_agent" in event_authors
    assert "deploy_service_a" in event_authors
    assert "deploy_service_b" in event_authors
    assert "deploy_service_c" in event_authors
    assert "reporting_agent" in event_authors

    # Verify reporting agent saw all deployment results
    reporting_events = [e for e in events if e.author == "reporting_agent"]
    assert len(reporting_events) > 0
    report = "".join([e.content.parts[0].text for e in reporting_events if e.content and e.content.parts])

    assert "Validation: Validation: PASSED" in report
    assert "Service A: Deployed service_a" in report
    assert "Service B: Deployed service_b" in report
    assert "Service C: Deployed service_c" in report


@pytest.mark.asyncio
async def test_parallel_aggregation_with_large_outputs():
    """Test parallel → aggregation with larger output data (stress test).

    Scenario:
    1. Create 10 parallel agents with 10KB outputs each
    2. Aggregator processes all 100KB of data
    3. Verify no data loss
    """

    class MockLargeOutputAgent(BaseAgent):
        """Mock agent generating large outputs."""

        def __init__(self, agent_id: int):
            super().__init__(name=f"large_output_agent_{agent_id}")
            self.output_key = f"large_output_{agent_id}"
            self.agent_id = agent_id

        async def run_async(self, context: InvocationContext) -> AsyncGenerator[Event, None]:
            # Generate 10KB of output (simulate large logs/metrics)
            large_output = f"Agent {self.agent_id} data: " + "x" * 10000

            yield Event(
                author=self.name,
                content=Content(parts=[Part(text=large_output)], role="model"),
            )

    class MockLargeAggregatorAgent(BaseAgent):
        """Mock aggregator that verifies large outputs."""

        def __init__(self, expected_agent_count: int):
            super().__init__(name="large_aggregator")
            self.expected_agent_count = expected_agent_count

        async def run_async(self, context: InvocationContext) -> AsyncGenerator[Event, None]:
            # Count how many outputs we received
            received_count = sum(1 for key in context.session.state if key.startswith("large_output_"))

            # Verify total size
            total_size = sum(
                len(str(value)) for key, value in context.session.state.items() if key.startswith("large_output_")
            )

            result = (
                f"Aggregated {received_count}/{self.expected_agent_count} large outputs, total size: {total_size} bytes"
            )

            yield Event(
                author=self.name,
                content=Content(parts=[Part(text=result)], role="model"),
            )

    # Create 10 agents with large outputs
    large_agents = [MockLargeOutputAgent(agent_id=i) for i in range(10)]

    parallel_collector = KAgentParallelAgent(
        name="large_output_collector",
        sub_agents=large_agents,
        max_workers=10,
    )

    aggregator = MockLargeAggregatorAgent(expected_agent_count=10)

    combined_workflow = KAgentSequentialAgent(
        name="large_output_workflow",
        sub_agents=[parallel_collector, aggregator],
    )

    # Execute
    parent_context = create_test_invocation_context(
        session_id="large_output_session",
        user_id="test_user",
        app_name="test_app",
    )

    events = []
    async for event in combined_workflow.run_async(parent_context):
        events.append(event)

    # Verify aggregator processed all outputs
    aggregator_events = [e for e in events if e.author == "large_aggregator"]
    assert len(aggregator_events) > 0
    aggregator_output = "".join([e.content.parts[0].text for e in aggregator_events if e.content and e.content.parts])

    assert "Aggregated 10/10 large outputs" in aggregator_output
    # Total size should be ~100KB (10 agents * 10KB each)
    assert "total size:" in aggregator_output
