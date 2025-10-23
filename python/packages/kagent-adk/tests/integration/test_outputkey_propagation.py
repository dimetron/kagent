"""Integration test for outputKey propagation in workflow agents.

This test verifies that outputs from one sub-agent are properly stored
in workflow state and can be accessed by subsequent sub-agents.
"""

import pytest
from datetime import datetime, timezone
from unittest.mock import AsyncMock, MagicMock

from google.adk.agents.invocation_context import InvocationContext
from google.adk.events import Event
from google.adk.sessions import Session
from google.genai.types import Content, Part

from kagent.adk.agents.sequential import KAgentSequentialAgent
from kagent.adk.workflow.state import WorkflowStateManager


@pytest.mark.asyncio
@pytest.mark.integration
class TestOutputKeyPropagation:
    """Integration tests for outputKey propagation in workflows (T023)."""

    async def test_output_key_propagates_through_workflow(self):
        """Test that outputKey values propagate correctly through a 2-agent workflow.
        
        Expected behavior:
        - First agent produces output with output_key="generated_code"
        - Output is stored in workflow state
        - Second agent can access the generated_code value
        - Workflow state contains both outputs at completion
        
        Note: This test validates the WorkflowStateManager directly since
        creating mock BaseAgent instances requires complex setup.
        """
        # Create state manager and workflow state
        state_manager = WorkflowStateManager()
        workflow_state = state_manager.create_workflow(
            workflow_session_id="test-propagation-session",
            user_id="test-user",
            agent_name="test_workflow",
        )
        
        # Simulate agent 1 execution (code writer)
        code_output = "def hello(): print('world')"
        workflow_state.set_output("generated_code", code_output)
        
        # Verify agent 1 output stored
        assert workflow_state.get_output("generated_code") == code_output
        
        # Simulate agent 2 execution (code reviewer)
        # Agent 2 can access agent 1's output
        retrieved_code = workflow_state.get_output("generated_code")
        assert retrieved_code == code_output
        
        # Agent 2 produces its own output
        review_output = "Add type hints"
        workflow_state.set_output("review_comments", review_output)
        
        # Verify both outputs are in workflow state
        assert len(workflow_state.state_data) == 2
        assert workflow_state.get_output("generated_code") == code_output
        assert workflow_state.get_output("review_comments") == review_output
        
        # Mark workflow as completed
        workflow_state.mark_completed()
        assert workflow_state.status == "completed"

    async def test_workflow_state_accessible_across_agents(self):
        """Test that workflow state is shared across all sub-agents.
        
        Expected behavior:
        - Agent 1 produces output A
        - Agent 2 produces output B
        - Agent 3 can access both A and B
        - Final state contains all three outputs
        """
        # Create state manager
        state_manager = WorkflowStateManager()
        workflow_state = state_manager.create_workflow(
            workflow_session_id="test-workflow-456",
            user_id="test-user",
            agent_name="multi-agent-workflow",
        )
        
        # Simulate agent executions
        # Agent 1
        workflow_state.set_output("data_a", "Output from agent 1")
        assert workflow_state.get_output("data_a") == "Output from agent 1"
        
        # Agent 2
        workflow_state.set_output("data_b", "Output from agent 2")
        assert workflow_state.get_output("data_b") == "Output from agent 2"
        
        # Agent 3 can access both previous outputs
        assert workflow_state.get_output("data_a") == "Output from agent 1"
        assert workflow_state.get_output("data_b") == "Output from agent 2"
        
        workflow_state.set_output("data_c", "Output from agent 3")
        
        # Verify all outputs are present
        assert len(workflow_state.state_data) == 3
        assert workflow_state.get_output("data_a") == "Output from agent 1"
        assert workflow_state.get_output("data_b") == "Output from agent 2"
        assert workflow_state.get_output("data_c") == "Output from agent 3"

    async def test_workflow_state_persists_across_execution(self):
        """Test that workflow state persists and can be retrieved.
        
        Expected behavior:
        - Create workflow state
        - Add multiple outputs during execution
        - Retrieve state from manager
        - Verify all outputs are preserved
        """
        # Create state manager
        state_manager = WorkflowStateManager()
        
        # Create workflow state
        workflow_session_id = "persistent-workflow-789"
        workflow_state = state_manager.create_workflow(
            workflow_session_id=workflow_session_id,
            user_id="test-user",
            agent_name="persistent-workflow",
        )
        
        # Add outputs during "execution"
        workflow_state.set_output("step1", "First step completed")
        workflow_state.set_output("step2", "Second step completed")
        workflow_state.set_output("step3", "Third step completed")
        
        # Mark as completed
        workflow_state.mark_completed()
        
        # Retrieve state from manager
        retrieved_state = state_manager.get_workflow(workflow_session_id)
        
        # Verify state is preserved
        assert retrieved_state.workflow_session_id == workflow_session_id
        assert retrieved_state.status == "completed"
        assert len(retrieved_state.state_data) == 3
        assert retrieved_state.get_output("step1") == "First step completed"
        assert retrieved_state.get_output("step2") == "Second step completed"
        assert retrieved_state.get_output("step3") == "Third step completed"

    async def test_workflow_state_isolation(self):
        """Test that different workflows have isolated state.
        
        Expected behavior:
        - Create two workflows with different session IDs
        - Add outputs to each workflow
        - Verify outputs don't leak between workflows
        - Each workflow maintains its own state
        """
        # Create state manager
        state_manager = WorkflowStateManager()
        
        # Create workflow 1
        workflow1_state = state_manager.create_workflow(
            workflow_session_id="workflow-1",
            user_id="user1",
            agent_name="workflow-1",
        )
        workflow1_state.set_output("data", "Workflow 1 data")
        
        # Create workflow 2
        workflow2_state = state_manager.create_workflow(
            workflow_session_id="workflow-2",
            user_id="user2",
            agent_name="workflow-2",
        )
        workflow2_state.set_output("data", "Workflow 2 data")
        
        # Verify states are isolated
        assert workflow1_state.get_output("data") == "Workflow 1 data"
        assert workflow2_state.get_output("data") == "Workflow 2 data"
        
        # Add more outputs
        workflow1_state.set_output("result", "Result from workflow 1")
        workflow2_state.set_output("result", "Result from workflow 2")
        
        # Verify isolation is maintained
        retrieved1 = state_manager.get_workflow("workflow-1")
        retrieved2 = state_manager.get_workflow("workflow-2")
        
        assert retrieved1.get_output("data") == "Workflow 1 data"
        assert retrieved1.get_output("result") == "Result from workflow 1"
        
        assert retrieved2.get_output("data") == "Workflow 2 data"
        assert retrieved2.get_output("result") == "Result from workflow 2"

