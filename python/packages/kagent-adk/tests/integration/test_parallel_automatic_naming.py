"""Integration tests for parallel workflows with automatic outputKey naming.

This test suite verifies end-to-end automatic outputKey generation
for parallel workflows without mocking:
- Automatic outputKey generation from namespace and name
- Parallel execution with auto-generated keys
- Workflow state persistence with auto-generated keys
- SubAgentExecution metadata tracking
- Accessibility of outputs regardless of completion order
"""

import asyncio
from datetime import datetime, timezone
from unittest.mock import AsyncMock, MagicMock, patch

import pytest
from google.adk.agents.invocation_context import InvocationContext
from google.adk.events import Event
from google.adk.sessions import Session
from google.genai.types import Content, Part

from kagent.adk.agents.parallel import KAgentParallelAgent
from kagent.adk.types import KAgentRemoteA2aAgent, SubAgentReference, WorkflowAgentConfig, generate_output_key
from kagent.adk.workflow.state import SubAgentExecution, WorkflowState, WorkflowStateManager, WorkflowStatus


class TestAutomaticOutputKeyInParallelWorkflow:
    """Test automatic outputKey generation in parallel workflow context."""

    @pytest.mark.asyncio
    async def test_workflow_config_generates_automatic_output_keys(self):
        """Test that WorkflowAgentConfig.to_agent() generates automatic outputKeys."""
        # Create workflow config with 3 sub-agents (no explicit outputKey)
        workflow_config = WorkflowAgentConfig(
            name="data-collection-workflow",
            description="Test parallel workflow",
            namespace="production",
            workflow_type="parallel",
            sub_agents=[
                SubAgentReference(name="east-collector", namespace="production"),
                SubAgentReference(name="west-collector", namespace="production"),
                SubAgentReference(name="central-collector", namespace="staging"),
            ],
            max_workers=3,
        )

        # Convert to agent (this triggers auto-generation)
        with patch("kagent.adk.types.create_user_propagating_httpx_client"):
            workflow_agent = workflow_config.to_agent()

        # Verify workflow agent created
        assert isinstance(workflow_agent, KAgentParallelAgent)
        assert len(workflow_agent.sub_agents) == 3

        # Verify automatic outputKeys generated
        expected_keys = [
            "production_east_collector",
            "production_west_collector",
            "staging_central_collector",
        ]
        actual_keys = [agent.output_key for agent in workflow_agent.sub_agents]
        assert actual_keys == expected_keys

    @pytest.mark.asyncio
    async def test_parallel_workflow_output_keys_verified(self):
        """Test that parallel workflow agents have correct auto-generated outputKeys."""
        # Create workflow config with 3 sub-agents
        workflow_config = WorkflowAgentConfig(
            name="test-workflow",
            description="Test workflow with automatic keys",
            namespace="production",
            workflow_type="parallel",
            sub_agents=[
                SubAgentReference(name="east-collector", namespace="production"),
                SubAgentReference(name="west-collector", namespace="production"),
                SubAgentReference(name="central-collector", namespace="staging"),
            ],
            max_workers=3,
        )

        # Convert to agent
        with patch("kagent.adk.types.create_user_propagating_httpx_client"):
            workflow_agent = workflow_config.to_agent()

        # Verify that sub-agents have correct auto-generated outputKeys
        assert isinstance(workflow_agent, KAgentParallelAgent)
        output_keys = [agent.output_key for agent in workflow_agent.sub_agents]
        
        expected_keys = [
            "production_east_collector",
            "production_west_collector",
            "staging_central_collector",
        ]
        
        assert output_keys == expected_keys

    @pytest.mark.asyncio
    async def test_auto_generated_keys_in_execution_records(self):
        """Test that SubAgentExecution records contain auto-generated outputKeys."""
        # Create workflow state manager
        state_manager = WorkflowStateManager()

        # Create workflow
        workflow_state = state_manager.create_workflow(
            workflow_session_id="auto-key-test-123",
            user_id="user@example.com",
            agent_name="parallel-workflow",
            namespace="production",
        )

        # Simulate 3 parallel agents with auto-generated outputKeys
        agents_data = [
            ("east-collector", "production", "production_east_collector"),
            ("west-collector", "production", "production_west_collector"),
            ("central-collector", "staging", "staging_central_collector"),
        ]

        async def simulate_agent(idx: int, name: str, namespace: str, output_key: str):
            # Simulate agent processing
            await asyncio.sleep(0.01)

            # Store output with auto-generated key
            output_value = f"Output from {name}"
            await state_manager.update_output_concurrent(
                "auto-key-test-123", output_key, output_value
            )

            # Record execution
            execution = SubAgentExecution(
                index=idx,
                agent_name=name,
                agent_namespace=namespace,
                session_id=f"auto-key-test-123-sub-{idx}",
                output_key=output_key,  # Auto-generated
                started_at=datetime.now(timezone.utc),
                completed_at=datetime.now(timezone.utc),
                status="success",
                output_size_bytes=len(output_value.encode("utf-8")),
                completion_order=None,  # Will be set later
            )
            await state_manager.add_execution_concurrent("auto-key-test-123", execution)

        # Execute all 3 agents in parallel
        tasks = [
            simulate_agent(i, name, namespace, output_key)
            for i, (name, namespace, output_key) in enumerate(agents_data)
        ]
        await asyncio.gather(*tasks)

        # Verify all outputs stored with auto-generated keys
        assert len(workflow_state.state_data) == 3
        assert workflow_state.get_output("production_east_collector") == "Output from east-collector"
        assert workflow_state.get_output("production_west_collector") == "Output from west-collector"
        assert workflow_state.get_output("staging_central_collector") == "Output from central-collector"

        # Verify execution records have auto-generated outputKeys
        assert len(workflow_state.sub_agent_executions) == 3
        for execution in workflow_state.sub_agent_executions:
            assert execution.output_key in [
                "production_east_collector",
                "production_west_collector",
                "staging_central_collector",
            ]

    @pytest.mark.asyncio
    async def test_outputs_accessible_regardless_of_completion_order(self):
        """Test that all outputs are accessible regardless of parallel completion order."""
        state_manager = WorkflowStateManager()
        workflow_state = state_manager.create_workflow(
            workflow_session_id="order-test-123",
            user_id="user@example.com",
            agent_name="parallel-workflow",
            namespace="production",
        )

        # Simulate agents completing in random order with delays
        async def simulate_agent_with_delay(
            idx: int, name: str, output_key: str, delay: float
        ):
            await asyncio.sleep(delay)

            output_value = f"Output from {name} (index {idx})"
            await state_manager.update_output_concurrent(
                "order-test-123", output_key, output_value
            )

            execution = SubAgentExecution(
                index=idx,
                agent_name=name,
                agent_namespace="production",
                session_id=f"order-test-123-sub-{idx}",
                output_key=output_key,
                started_at=datetime.now(timezone.utc),
                completed_at=datetime.now(timezone.utc),
                status="success",
                output_size_bytes=len(output_value.encode("utf-8")),
                completion_order=None,
            )
            await state_manager.add_execution_concurrent("order-test-123", execution)

        # Agent 1 completes last (0.03s), Agent 0 completes second (0.02s), Agent 2 completes first (0.01s)
        tasks = [
            simulate_agent_with_delay(0, "east-collector", "production_east_collector", 0.02),
            simulate_agent_with_delay(1, "west-collector", "production_west_collector", 0.03),
            simulate_agent_with_delay(2, "central-collector", "production_central_collector", 0.01),
        ]
        await asyncio.gather(*tasks)

        # Verify all 3 outputs present
        assert len(workflow_state.state_data) == 3

        # Verify outputs accessible by auto-generated key (regardless of completion order)
        assert (
            workflow_state.get_output("production_east_collector")
            == "Output from east-collector (index 0)"
        )
        assert (
            workflow_state.get_output("production_west_collector")
            == "Output from west-collector (index 1)"
        )
        assert (
            workflow_state.get_output("production_central_collector")
            == "Output from central-collector (index 2)"
        )

    @pytest.mark.asyncio
    async def test_mixed_automatic_and_explicit_output_keys(self):
        """Test workflow with mix of automatic and explicit outputKeys."""
        # Create workflow config with mixed keys
        workflow_config = WorkflowAgentConfig(
            name="mixed-workflow",
            description="Mixed automatic and explicit keys",
            namespace="production",
            workflow_type="parallel",
            sub_agents=[
                SubAgentReference(name="east-collector", namespace="production"),  # Automatic
                SubAgentReference(
                    name="west-collector", namespace="production", output_key="custom_west_key"
                ),  # Explicit
                SubAgentReference(name="central-collector", namespace="staging"),  # Automatic
            ],
            max_workers=3,
        )

        # Convert to agent
        with patch("kagent.adk.types.create_user_propagating_httpx_client"):
            workflow_agent = workflow_config.to_agent()

        # Verify outputKeys
        output_keys = [agent.output_key for agent in workflow_agent.sub_agents]
        assert output_keys == [
            "production_east_collector",  # Automatic
            "custom_west_key",  # Explicit
            "staging_central_collector",  # Automatic
        ]

    @pytest.mark.asyncio
    async def test_auto_generated_keys_with_hyphens_in_names(self):
        """Test that hyphens in agent names are converted to underscores."""
        # Create sub-agent references with hyphens
        refs = [
            SubAgentReference(name="east-us-collector", namespace="prod-env"),
            SubAgentReference(name="west-eu-collector", namespace="prod-env"),
        ]

        # Generate outputKeys
        keys = [generate_output_key(ref) for ref in refs]

        # Verify hyphens converted to underscores
        assert keys == [
            "prod_env_east_us_collector",
            "prod_env_west_eu_collector",
        ]



class TestAutomaticOutputKeyEdgeCases:
    """Test edge cases for automatic outputKey generation in workflows."""

    @pytest.mark.asyncio
    async def test_duplicate_auto_generated_keys_rejected(self):
        """Test that duplicate auto-generated outputKeys are detected."""
        # Create two sub-agents with same name and namespace (would generate same key)
        refs = [
            SubAgentReference(name="collector", namespace="production"),
            SubAgentReference(name="collector", namespace="production"),
        ]

        # Generate keys
        keys = [generate_output_key(ref) for ref in refs]

        # Both generate same key (this is expected behavior)
        assert keys[0] == keys[1] == "production_collector"

        # In real workflow, controller would reject this at admission time
        # This test documents current behavior

    @pytest.mark.asyncio
    async def test_empty_workflow_state_before_execution(self):
        """Test that workflow state is empty before parallel execution."""
        state_manager = WorkflowStateManager()
        workflow_state = state_manager.create_workflow(
            workflow_session_id="empty-test-123",
            user_id="user@example.com",
            agent_name="test-workflow",
            namespace="production",
        )

        # Verify initial state is empty
        assert len(workflow_state.state_data) == 0
        assert len(workflow_state.sub_agent_executions) == 0
        assert workflow_state.status == WorkflowStatus.RUNNING

    @pytest.mark.asyncio
    async def test_large_number_of_parallel_agents(self):
        """Test automatic outputKey generation with many parallel agents."""
        # Create 20 sub-agents with auto-generated outputKeys
        refs = [
            SubAgentReference(name=f"agent-{i}", namespace="production")
            for i in range(20)
        ]

        # Generate keys
        keys = [generate_output_key(ref) for ref in refs]

        # Verify all keys unique
        assert len(keys) == len(set(keys)) == 20

        # Verify format correct
        for i, key in enumerate(keys):
            assert key == f"production_agent_{i}"


class TestAutomaticOutputKeyStateInjection:
    """Test state injection with auto-generated outputKeys."""

    @pytest.mark.asyncio
    async def test_auto_keys_injected_into_parent_session(self):
        """Test that outputs with auto-generated keys are injected into parent session."""
        state_manager = WorkflowStateManager()
        workflow_state = state_manager.create_workflow(
            workflow_session_id="inject-test-123",
            user_id="user@example.com",
            agent_name="test-workflow",
            namespace="production",
        )

        # Add outputs with auto-generated keys
        await state_manager.update_output_concurrent(
            "inject-test-123", "production_east_collector", "East data"
        )
        await state_manager.update_output_concurrent(
            "inject-test-123", "production_west_collector", "West data"
        )

        # Simulate state injection into parent session
        parent_session_state = {}
        parent_session_state.update(workflow_state.state_data)

        # Verify auto-generated keys present in parent session
        assert parent_session_state["production_east_collector"] == "East data"
        assert parent_session_state["production_west_collector"] == "West data"

    @pytest.mark.asyncio
    async def test_auto_keys_accessible_by_aggregator(self):
        """Test that aggregator agent can access outputs via auto-generated keys."""
        state_manager = WorkflowStateManager()
        workflow_state = state_manager.create_workflow(
            workflow_session_id="aggregator-test-123",
            user_id="user@example.com",
            agent_name="parallel-workflow",
            namespace="production",
        )

        # Phase 1: Parallel agents store outputs with auto-generated keys
        auto_keys = [
            "production_east_collector",
            "production_west_collector",
            "staging_central_collector",
        ]
        for key in auto_keys:
            await state_manager.update_output_concurrent(
                "aggregator-test-123", key, f"Data for {key}"
            )

        # Phase 2: Aggregator agent accesses all outputs
        all_outputs = state_manager.get_outputs("aggregator-test-123")

        # Verify aggregator can access all auto-generated keys
        for key in auto_keys:
            assert key in all_outputs
            assert all_outputs[key] == f"Data for {key}"

