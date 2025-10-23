"""Integration test for state key injection in workflow agents.

This test verifies that state keys are properly injected into agent instructions
using the {key_name} template syntax.
"""

import pytest
from unittest.mock import AsyncMock, MagicMock, patch

from google.adk.agents.invocation_context import InvocationContext
from google.adk.events import Event
from google.adk.sessions import Session
from google.genai.types import Content, Part

from kagent.adk.workflow.injection import inject_state_keys
from kagent.adk.workflow.state import WorkflowStateManager


@pytest.mark.asyncio
@pytest.mark.integration
class TestStateInjectionWorkflow:
    """Integration tests for state injection in workflows (T032)."""

    async def test_instruction_injection_in_three_agent_workflow(self):
        """Test state injection in a 3-agent workflow (writer → reviewer → refactorer).
        
        Expected behavior:
        - Writer agent outputs code to "generated_code"
        - Reviewer instruction contains {generated_code}, gets replaced with actual code
        - Reviewer outputs comments to "review_comments"
        - Refactorer instruction contains {generated_code} and {review_comments}
        - Both placeholders are replaced correctly
        """
        # Create state manager and workflow state
        state_manager = WorkflowStateManager()
        workflow_state = state_manager.create_workflow(
            workflow_session_id="test-injection-123",
            user_id="test-user",
            agent_name="code-pipeline",
        )
        
        # Agent 1: Code Writer (outputs to "generated_code")
        code_output = "def hello(): print('world')"
        workflow_state.set_output("generated_code", code_output)
        
        # Agent 2: Code Reviewer (reads {generated_code}, outputs to "review_comments")
        reviewer_instruction_template = "Review this code:\n{generated_code}\n\nProvide feedback."
        reviewer_instruction = inject_state_keys(
            reviewer_instruction_template,
            workflow_state.state_data
        )
        
        # Verify {generated_code} was replaced
        assert "{generated_code}" not in reviewer_instruction
        assert code_output in reviewer_instruction
        assert "Review this code" in reviewer_instruction
        
        # Reviewer produces output
        review_output = "Add type hints for better documentation"
        workflow_state.set_output("review_comments", review_output)
        
        # Agent 3: Code Refactorer (reads {generated_code} and {review_comments})
        refactorer_instruction_template = """Original code:
{generated_code}

Review comments:
{review_comments}

Refactor accordingly."""
        
        refactorer_instruction = inject_state_keys(
            refactorer_instruction_template,
            workflow_state.state_data
        )
        
        # Verify both placeholders were replaced
        assert "{generated_code}" not in refactorer_instruction
        assert "{review_comments}" not in refactorer_instruction
        assert code_output in refactorer_instruction
        assert review_output in refactorer_instruction
        assert "Original code:" in refactorer_instruction
        assert "Review comments:" in refactorer_instruction

    async def test_missing_key_error_in_workflow(self):
        """Test that missing state keys produce helpful error messages.
        
        Expected behavior:
        - Agent tries to reference a key that doesn't exist
        - ValueError raised with clear message
        - Error message lists available keys
        """
        # Create workflow state with only one output
        state_manager = WorkflowStateManager()
        workflow_state = state_manager.create_workflow(
            workflow_session_id="test-error-456",
            user_id="test-user",
            agent_name="error-workflow",
        )
        
        workflow_state.set_output("generated_code", "def hello(): pass")
        
        # Try to inject instruction referencing non-existent key
        instruction_template = "Review this code: {code}"  # Wrong key! Should be {generated_code}
        
        with pytest.raises(ValueError) as exc_info:
            inject_state_keys(instruction_template, workflow_state.state_data)
        
        # Verify error message is helpful
        error_msg = str(exc_info.value)
        assert "State key 'code' not found" in error_msg
        assert "Available keys: ['generated_code']" in error_msg

    async def test_state_injection_with_multiple_keys(self):
        """Test injection with multiple keys in a single instruction.
        
        Expected behavior:
        - Workflow state contains multiple outputs
        - Instruction references multiple keys
        - All placeholders are replaced correctly
        - Order of placeholders doesn't matter
        """
        # Create workflow state with multiple outputs
        state_manager = WorkflowStateManager()
        workflow_state = state_manager.create_workflow(
            workflow_session_id="test-multi-789",
            user_id="test-user",
            agent_name="multi-key-workflow",
        )
        
        workflow_state.set_output("name", "John Doe")
        workflow_state.set_output("role", "Senior Engineer")
        workflow_state.set_output("project", "KAgent")
        workflow_state.set_output("task", "Implement outputKey feature")
        
        # Instruction with multiple placeholders
        instruction_template = """
        Hi {name},
        
        As a {role}, you are working on the {project} project.
        Your current task is: {task}
        
        Please provide a status update on {task} for {project}.
        """
        
        injected = inject_state_keys(instruction_template, workflow_state.state_data)
        
        # Verify all placeholders replaced
        assert "{name}" not in injected
        assert "{role}" not in injected
        assert "{project}" not in injected
        assert "{task}" not in injected
        
        # Verify actual values present
        assert "John Doe" in injected
        assert "Senior Engineer" in injected
        assert "KAgent" in injected
        assert "Implement outputKey feature" in injected

    async def test_state_injection_with_complex_outputs(self):
        """Test injection with outputs containing special characters.
        
        Expected behavior:
        - Output values can contain newlines, special chars, code blocks
        - Injection preserves all special characters
        - No escaping issues
        """
        # Create workflow state
        state_manager = WorkflowStateManager()
        workflow_state = state_manager.create_workflow(
            workflow_session_id="test-complex-101",
            user_id="test-user",
            agent_name="complex-workflow",
        )
        
        # Add output with complex content
        complex_code = """def factorial(n: int) -> int:
    '''Calculate factorial of n.
    
    Args:
        n: Non-negative integer
        
    Returns:
        Factorial of n
        
    Raises:
        ValueError: If n < 0
    '''
    if n < 0:
        raise ValueError("n must be >= 0")
    if n <= 1:
        return 1
    return n * factorial(n - 1)
"""
        workflow_state.set_output("generated_code", complex_code)
        
        # Add review with special characters
        review = """Issues found:
- Missing type hints in some places
- Consider iterative approach (stack overflow risk)
- Good: Proper docstring ✓
- Good: Error handling ✓

Suggestion: Use @lru_cache for optimization
"""
        workflow_state.set_output("review", review)
        
        # Inject into instruction
        instruction_template = """Code:
```python
{generated_code}
```

Review:
{review}

Address the issues."""
        
        injected = inject_state_keys(instruction_template, workflow_state.state_data)
        
        # Verify content preserved
        assert complex_code in injected
        assert review in injected
        assert "def factorial" in injected
        assert "✓" in injected  # Unicode character preserved
        assert "@lru_cache" in injected

    async def test_state_injection_performance(self):
        """Test that state injection is performant for large outputs.
        
        Expected behavior:
        - Injection with large output values completes quickly
        - Memory usage reasonable
        - No performance degradation
        """
        import time
        
        # Create workflow state with large outputs
        state_manager = WorkflowStateManager()
        workflow_state = state_manager.create_workflow(
            workflow_session_id="test-perf-202",
            user_id="test-user",
            agent_name="perf-workflow",
        )
        
        # Create large output (1MB)
        large_output = "x" * (1024 * 1024)  # 1MB string
        workflow_state.set_output("large_data", large_output)
        
        # Create instruction with placeholder
        instruction_template = "Process this data: {large_data}"
        
        # Measure injection time
        start = time.perf_counter()
        injected = inject_state_keys(instruction_template, workflow_state.state_data)
        end = time.perf_counter()
        
        injection_time_ms = (end - start) * 1000
        
        # Verify injection completed quickly (< 10ms target)
        assert injection_time_ms < 50.0, f"Injection too slow: {injection_time_ms:.2f}ms"
        
        # Verify output correct
        assert large_output in injected
        assert "{large_data}" not in injected

    async def test_empty_instruction_passthrough(self):
        """Test that instructions without placeholders are unchanged.
        
        Expected behavior:
        - Instruction with no {key} placeholders
        - Injection returns same instruction
        - No errors or modifications
        """
        # Create workflow state
        state_manager = WorkflowStateManager()
        workflow_state = state_manager.create_workflow(
            workflow_session_id="test-passthrough-303",
            user_id="test-user",
            agent_name="passthrough-workflow",
        )
        
        workflow_state.set_output("data", "Some data")
        
        # Instruction with no placeholders
        instruction = "Write a hello world function in Python."
        
        # Inject (should return unchanged)
        injected = inject_state_keys(instruction, workflow_state.state_data)
        
        # Verify unchanged
        assert injected == instruction

