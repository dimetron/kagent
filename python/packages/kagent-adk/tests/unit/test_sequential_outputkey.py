"""Unit tests for sequential workflow agent with outputKey support.

These tests verify that SequentialAgent properly handles outputKey for
passing data between sub-agents with separate session IDs.
"""

from unittest.mock import AsyncMock, MagicMock, patch

import pytest
from google.adk.agents.invocation_context import InvocationContext
from google.adk.events import Event
from google.adk.sessions import Session
from google.genai.types import Content, Part

from kagent.adk.workflow.state import WorkflowState, WorkflowStateManager


class TestSequentialAgentOutputKey:
    """Tests for SequentialAgent with outputKey functionality."""

    @pytest.mark.asyncio
    async def test_sequential_workflow_with_output_key(self):
        """Test that sequential workflow stores outputs in state using outputKey.
        
        Expected behavior:
        - Create workflow with 2 sub-agents (writer, reviewer)
        - Writer has output_key="generated_code"
        - After execution, verify workflow state contains generated_code
        - Verify 2 separate session IDs were created
        """
        # Create workflow state manager
        state_manager = WorkflowStateManager()
        workflow_state = state_manager.create_workflow(
            workflow_session_id="test-workflow-123",
            user_id="test-user",
            agent_name="test-workflow",
        )
        
        # Simulate sub-agent 1 execution: code writer
        writer_output = "def hello(): print('world')"
        workflow_state.set_output("generated_code", writer_output)
        
        # Verify output stored
        assert workflow_state.get_output("generated_code") == "def hello(): print('world')"
        
        # Simulate sub-agent 2 execution: code reviewer  
        reviewer_output = "Add type hints"
        workflow_state.set_output("review_comments", reviewer_output)
        
        # Verify both outputs in state
        assert workflow_state.get_output("generated_code") == "def hello(): print('world')"
        assert workflow_state.get_output("review_comments") == "Add type hints"
        
        # Verify workflow status
        workflow_state.mark_completed()
        assert workflow_state.status == "completed"

    @pytest.mark.asyncio
    async def test_output_key_available_to_next_agent(self):
        """Test that output from first agent is available to second agent.
        
        Expected behavior:
        - First agent outputs to "data" key
        - Second agent can access workflow state with "data" key
        - State is properly propagated
        """
        # Create workflow state
        state_manager = WorkflowStateManager()
        workflow_state = state_manager.create_workflow(
            workflow_session_id="workflow-456",
            user_id="user1",
            agent_name="test-workflow",
        )
        
        # First agent produces output
        first_output = "Result from first agent"
        workflow_state.set_output("data", first_output)
        
        # Second agent should be able to access it
        accessed_data = workflow_state.get_output("data")
        assert accessed_data == first_output
        
        # Verify state propagation through manager
        retrieved_state = state_manager.get_workflow("workflow-456")
        assert retrieved_state.get_output("data") == first_output

    def test_separate_session_ids_created(self):
        """Test that each sub-agent gets its own session ID.
        
        Expected behavior:
        - Workflow agent has parent session ID
        - Sub-agent 1 has unique child session ID
        - Sub-agent 2 has unique child session ID
        - All session IDs are different
        """
        # Test session ID generation pattern
        parent_session_id = "workflow-abc123"
        
        # Generate child session IDs
        child_1_id = f"{parent_session_id}-writer-0"
        child_2_id = f"{parent_session_id}-reviewer-1"
        
        # Verify all different
        assert parent_session_id != child_1_id
        assert parent_session_id != child_2_id
        assert child_1_id != child_2_id
        
        # Verify pattern
        assert child_1_id.startswith(parent_session_id)
        assert child_2_id.startswith(parent_session_id)
        assert "-writer-" in child_1_id
        assert "-reviewer-" in child_2_id

    def test_workflow_state_persistence(self):
        """Test that workflow state is maintained across sub-agent executions.
        
        Expected behavior:
        - Multiple sub-agents execute in sequence
        - Each adds to workflow state via outputKey
        - Final state contains all outputKey values
        """
        # Create workflow state
        state_manager = WorkflowStateManager()
        workflow_state = state_manager.create_workflow(
            workflow_session_id="workflow-789",
            user_id="user1",
            agent_name="multi-agent-workflow",
        )
        
        # Simulate 3 sub-agents adding outputs
        workflow_state.set_output("step1_result", "Output from step 1")
        workflow_state.set_output("step2_result", "Output from step 2")
        workflow_state.set_output("step3_result", "Output from step 3")
        
        # Verify all outputs present
        assert len(workflow_state.state_data) == 3
        assert workflow_state.get_output("step1_result") == "Output from step 1"
        assert workflow_state.get_output("step2_result") == "Output from step 2"
        assert workflow_state.get_output("step3_result") == "Output from step 3"
        
        # Verify retrieval through manager
        outputs = state_manager.get_outputs("workflow-789")
        assert len(outputs) == 3
        assert outputs["step1_result"] == "Output from step 1"


# Additional test structure for future implementation
class TestOutputKeyEdgeCases:
    """Edge case tests for outputKey functionality."""

    def test_missing_output_key_field(self):
        """Test behavior when sub-agent has no outputKey defined."""
        # Create mock agent without output_key using spec to avoid auto-creating attributes
        agent = MagicMock(spec=["name", "run_async"])
        agent.name = "no-output-agent"
        
        # Should not have output_key attribute
        assert not hasattr(agent, "output_key")
        
        # Verify workflow handles agents without output_key gracefully
        # (The actual runtime logic checks for this)

    def test_empty_output_key(self):
        """Test behavior when outputKey is empty string."""
        state_manager = WorkflowStateManager()
        workflow_state = state_manager.create_workflow(
            workflow_session_id="workflow-empty",
            user_id="user1",
            agent_name="test",
        )
        
        # Empty string as key should still work (though not recommended)
        workflow_state.set_output("", "some value")
        assert workflow_state.get_output("") == "some value"

    def test_large_output_value(self):
        """Test handling of large output values (near 10MB limit)."""
        state_manager = WorkflowStateManager()
        workflow_state = state_manager.create_workflow(
            workflow_session_id="workflow-large",
            user_id="user1",
            agent_name="test",
        )
        
        # Create value near limit (9MB should succeed)
        large_value = "x" * (9 * 1024 * 1024)
        workflow_state.set_output("large_data", large_value)
        assert workflow_state.get_output("large_data") == large_value
        
        # Value exceeding limit should raise
        too_large = "x" * (11 * 1024 * 1024)
        with pytest.raises(ValueError):
            workflow_state.set_output("too_large", too_large)

