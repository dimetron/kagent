"""Unit tests for state key injection functionality."""

import pytest

from kagent.adk.workflow.injection import (
    inject_state_keys,
    validate_output_key,
    validate_output_value,
    validate_unique_output_keys,
)


class TestInjectStateKeys:
    """Tests for inject_state_keys function."""

    def test_simple_injection(self):
        """Test simple single key injection."""
        template = "Review this code: {generated_code}"
        state = {"generated_code": "def hello(): print('world')"}
        result = inject_state_keys(template, state)
        assert result == "Review this code: def hello(): print('world')"

    def test_multiple_keys(self):
        """Test injection of multiple keys."""
        template = "Original: {generated_code}\nComments: {review_comments}"
        state = {
            "generated_code": "def hello(): pass",
            "review_comments": "Add type hints",
        }
        result = inject_state_keys(template, state)
        assert result == "Original: def hello(): pass\nComments: Add type hints"

    def test_missing_key_error(self):
        """Test that missing key raises ValueError."""
        template = "Review this code: {code}"
        state = {"generated_code": "def hello(): pass"}
        
        with pytest.raises(ValueError) as exc_info:
            inject_state_keys(template, state)
        
        assert "State key 'code' not found" in str(exc_info.value)
        assert "Available keys: ['generated_code']" in str(exc_info.value)

    def test_no_placeholders_passthrough(self):
        """Test that template without placeholders is unchanged."""
        template = "Write a hello world function"
        state = {}
        result = inject_state_keys(template, state)
        assert result == "Write a hello world function"

    def test_literal_braces_in_output(self):
        """Test that output values can contain literal braces."""
        template = "Format this: {data}"
        state = {"data": "Use {{variable}} for templates"}
        result = inject_state_keys(template, state)
        assert result == "Format this: Use {{variable}} for templates"

    def test_special_characters_in_value(self):
        """Test that output values can contain special characters."""
        template = "Translate: {text}"
        state = {"text": "Hello 世界! \n\t Special chars: $@#%"}
        result = inject_state_keys(template, state)
        assert result == "Translate: Hello 世界! \n\t Special chars: $@#%"

    def test_duplicate_key_usage(self):
        """Test that same key used multiple times is replaced consistently."""
        template = "Code: {code}\nAgain: {code}"
        state = {"code": "def foo(): pass"}
        result = inject_state_keys(template, state)
        assert result == "Code: def foo(): pass\nAgain: def foo(): pass"

    def test_empty_state(self):
        """Test that injection with placeholders but empty state fails."""
        template = "Process {data}"
        state = {}
        
        with pytest.raises(ValueError) as exc_info:
            inject_state_keys(template, state)
        
        assert "State key 'data' not found" in str(exc_info.value)
        assert "Available keys: []" in str(exc_info.value)

    def test_underscore_prefix_key(self):
        """Test injection with underscore-prefixed key."""
        template = "Private: {_internal}"
        state = {"_internal": "secret value"}
        result = inject_state_keys(template, state)
        assert result == "Private: secret value"

    def test_mixed_case_key(self):
        """Test injection with mixed case key."""
        template = "Result: {MyOutput_v2}"
        state = {"MyOutput_v2": "result data"}
        result = inject_state_keys(template, state)
        assert result == "Result: result data"


class TestValidateOutputKey:
    """Tests for validate_output_key function."""

    def test_valid_lowercase(self):
        """Test valid key with lowercase letters."""
        validate_output_key("generated_code")  # Should not raise

    def test_valid_uppercase(self):
        """Test valid key with uppercase letters."""
        validate_output_key("GeneratedCode")  # Should not raise

    def test_valid_with_numbers(self):
        """Test valid key with numbers."""
        validate_output_key("result_123")  # Should not raise

    def test_valid_underscore_prefix(self):
        """Test valid key starting with underscore."""
        validate_output_key("_private_result")  # Should not raise

    def test_valid_mixed_case_numbers(self):
        """Test valid key with mixed case and numbers."""
        validate_output_key("myOutput_v2")  # Should not raise

    def test_invalid_number_prefix(self):
        """Test that key starting with number is invalid."""
        with pytest.raises(ValueError) as exc_info:
            validate_output_key("1result")
        assert "must start with letter/underscore" in str(exc_info.value)

    def test_invalid_hyphen(self):
        """Test that key with hyphen is invalid."""
        with pytest.raises(ValueError) as exc_info:
            validate_output_key("result-1")
        assert "must start with letter/underscore" in str(exc_info.value)

    def test_invalid_dot(self):
        """Test that key with dot is invalid."""
        with pytest.raises(ValueError) as exc_info:
            validate_output_key("result.value")
        assert "must start with letter/underscore" in str(exc_info.value)

    def test_invalid_space(self):
        """Test that key with space is invalid."""
        with pytest.raises(ValueError) as exc_info:
            validate_output_key("result value")
        assert "must start with letter/underscore" in str(exc_info.value)

    def test_invalid_special_chars(self):
        """Test that key with special chars is invalid."""
        with pytest.raises(ValueError) as exc_info:
            validate_output_key("result@123")
        assert "must start with letter/underscore" in str(exc_info.value)

    def test_invalid_max_length(self):
        """Test that key exceeding 100 chars is invalid."""
        long_key = "a" * 101
        with pytest.raises(ValueError) as exc_info:
            validate_output_key(long_key)
        assert "exceeds 100 characters" in str(exc_info.value)

    def test_invalid_empty(self):
        """Test that empty key is invalid."""
        with pytest.raises(ValueError) as exc_info:
            validate_output_key("")
        assert "cannot be empty" in str(exc_info.value)


class TestValidateOutputValue:
    """Tests for validate_output_value function."""

    def test_valid_small_value(self):
        """Test that small value is valid."""
        validate_output_value("small value")  # Should not raise

    def test_valid_large_value_under_limit(self):
        """Test that value under 10MB is valid."""
        large_value = "x" * (9 * 1024 * 1024)  # 9MB
        validate_output_value(large_value)  # Should not raise

    def test_invalid_value_exceeds_limit(self):
        """Test that value exceeding 10MB is invalid."""
        too_large = "x" * (11 * 1024 * 1024)  # 11MB
        with pytest.raises(ValueError) as exc_info:
            validate_output_value(too_large)
        assert "exceeds maximum size" in str(exc_info.value)

    def test_custom_max_size(self):
        """Test validation with custom max size."""
        value = "x" * 1000
        with pytest.raises(ValueError):
            validate_output_value(value, max_size_bytes=500)  # Should raise
        
    def test_custom_max_size_invalid(self):
        """Test that custom max size is enforced."""
        value = "x" * 1000
        with pytest.raises(ValueError):
            validate_output_value(value, max_size_bytes=500)


class TestValidateUniqueOutputKeys:
    """Tests for validate_unique_output_keys function."""

    def test_valid_unique_keys(self):
        """Test that unique output keys are valid."""
        class Agent:
            def __init__(self, name, output_key):
                self.name = name
                self.output_key = output_key
        
        agents = [
            Agent("agent1", "generated_code"),
            Agent("agent2", "review_comments"),
            Agent("agent3", "refactored_code"),
        ]
        validate_unique_output_keys(agents)  # Should not raise

    def test_valid_with_none_keys(self):
        """Test that agents without output_key are valid."""
        class Agent:
            def __init__(self, name, output_key=None):
                self.name = name
                self.output_key = output_key
        
        agents = [
            Agent("agent1", "code"),
            Agent("agent2", None),  # No output key
            Agent("agent3", "review"),
        ]
        validate_unique_output_keys(agents)  # Should not raise

    def test_invalid_duplicate_keys(self):
        """Test that duplicate output keys raise error."""
        class Agent:
            def __init__(self, name, output_key):
                self.name = name
                self.output_key = output_key
        
        agents = [
            Agent("agent1", "code"),
            Agent("agent2", "code"),  # Duplicate!
        ]
        
        with pytest.raises(ValueError) as exc_info:
            validate_unique_output_keys(agents)
        assert "Duplicate outputKey values found" in str(exc_info.value)
        assert "code" in str(exc_info.value)

    def test_invalid_multiple_duplicates(self):
        """Test that multiple duplicates are all reported."""
        class Agent:
            def __init__(self, name, output_key):
                self.name = name
                self.output_key = output_key
        
        agents = [
            Agent("agent1", "code"),
            Agent("agent2", "code"),
            Agent("agent3", "review"),
            Agent("agent4", "review"),
        ]
        
        with pytest.raises(ValueError) as exc_info:
            validate_unique_output_keys(agents)
        assert "Duplicate outputKey values found" in str(exc_info.value)
        # Both duplicates should be mentioned
        error_msg = str(exc_info.value)
        assert "code" in error_msg or "review" in error_msg

