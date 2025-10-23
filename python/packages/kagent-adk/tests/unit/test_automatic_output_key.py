"""Unit tests for automatic outputKey generation."""

import pytest

from kagent.adk.types import SubAgentReference, generate_output_key


class TestAutomaticOutputKeyGeneration:
    """Test suite for automatic outputKey generation functionality."""

    def test_automatic_naming_basic(self):
        """Test basic automatic outputKey generation from namespace and name."""
        ref = SubAgentReference(
            name="east-collector",
            namespace="production"
        )
        result = generate_output_key(ref)
        assert result == "production_east_collector"
        
    def test_automatic_naming_with_hyphens(self):
        """Test hyphen conversion to underscores in automatic outputKey."""
        ref = SubAgentReference(
            name="east-us-collector",
            namespace="production"
        )
        result = generate_output_key(ref)
        assert result == "production_east_us_collector"
        
    def test_automatic_naming_multiple_hyphens(self):
        """Test multiple hyphen conversion in namespace and name."""
        ref = SubAgentReference(
            name="east-us-metrics-collector",
            namespace="prod-env-west"
        )
        result = generate_output_key(ref)
        assert result == "prod_env_west_east_us_metrics_collector"
    
    def test_explicit_output_key_override(self):
        """Test that explicit output_key overrides automatic naming."""
        ref = SubAgentReference(
            name="east-collector",
            namespace="production",
            output_key="custom_key"
        )
        result = generate_output_key(ref)
        assert result == "custom_key"
    
    def test_explicit_output_key_with_underscores(self):
        """Test explicit outputKey with underscores is preserved."""
        ref = SubAgentReference(
            name="collector",
            namespace="production",
            output_key="my_custom_output_key"
        )
        result = generate_output_key(ref)
        assert result == "my_custom_output_key"
    
    def test_default_namespace(self):
        """Test automatic naming with default namespace."""
        ref = SubAgentReference(name="collector")  # namespace defaults to "default"
        result = generate_output_key(ref)
        assert result == "default_collector"
    
    def test_alphanumeric_names(self):
        """Test automatic naming with alphanumeric characters."""
        ref = SubAgentReference(
            name="collector123",
            namespace="production456"
        )
        result = generate_output_key(ref)
        assert result == "production456_collector123"
    
    def test_length_validation_at_limit(self):
        """Test automatic outputKey generation at 100 character limit."""
        # Create name that results in exactly 100 characters
        # Format: namespace_name = 100 chars total
        namespace = "a" * 49  # 49 chars
        name = "b" * 50  # 50 chars (49 + 1 underscore + 50 = 100)
        ref = SubAgentReference(name=name, namespace=namespace)
        result = generate_output_key(ref)
        assert len(result) == 100
        assert result == namespace + "_" + name
    
    def test_length_validation_exceeds_limit(self):
        """Test that auto-generated outputKey exceeding 100 chars raises ValueError."""
        # Create name that exceeds 100 characters
        namespace = "a" * 50
        name = "b" * 51  # Total: 50 + 1 (underscore) + 51 = 102 chars
        ref = SubAgentReference(name=name, namespace=namespace)
        
        with pytest.raises(ValueError) as exc_info:
            generate_output_key(ref)
        
        assert "exceeds 100 characters" in str(exc_info.value)
        assert "Please specify a shorter explicit outputKey" in str(exc_info.value)
    
    def test_length_validation_with_long_namespace(self):
        """Test length validation with very long namespace."""
        ref = SubAgentReference(
            name="collector",
            namespace="very-long-namespace-that-should-cause-validation-error-when-combined-with-agent-name-definitely"
        )
        
        with pytest.raises(ValueError) as exc_info:
            generate_output_key(ref)
        
        assert "exceeds 100 characters" in str(exc_info.value)
    
    def test_pattern_validation_valid_characters(self):
        """Test that valid characters (alphanumeric + underscore) pass validation."""
        ref = SubAgentReference(
            name="collector_123",  # Underscores in name should work
            namespace="prod_env"
        )
        result = generate_output_key(ref)
        assert result == "prod_env_collector_123"
    
    def test_pattern_validation_no_special_chars(self):
        """Test that generated keys only contain valid characters."""
        ref = SubAgentReference(
            name="east-collector",
            namespace="production"
        )
        result = generate_output_key(ref)
        # Should only contain alphanumeric and underscore
        assert all(c.isalnum() or c == "_" for c in result)
    
    def test_empty_name_generates_namespace_only_key(self):
        """Test that empty agent name generates key with just namespace."""
        ref = SubAgentReference(name="", namespace="production")
        result = generate_output_key(ref)
        assert result == "production_"
    
    def test_none_namespace_uses_default(self):
        """Test that None namespace uses default value."""
        ref = SubAgentReference(name="collector", namespace="default")
        result = generate_output_key(ref)
        assert result == "default_collector"
    
    def test_case_sensitivity(self):
        """Test that outputKey generation preserves case."""
        ref = SubAgentReference(
            name="EastCollector",
            namespace="Production"
        )
        result = generate_output_key(ref)
        assert result == "Production_EastCollector"
    
    def test_numeric_only_names(self):
        """Test automatic naming with numeric-only names."""
        ref = SubAgentReference(
            name="123",
            namespace="456"
        )
        result = generate_output_key(ref)
        assert result == "456_123"
    
    def test_single_character_names(self):
        """Test automatic naming with single character names."""
        ref = SubAgentReference(name="a", namespace="b")
        result = generate_output_key(ref)
        assert result == "b_a"
    
    def test_consecutive_hyphens(self):
        """Test handling of consecutive hyphens in names."""
        ref = SubAgentReference(
            name="east--collector",
            namespace="prod--env"
        )
        result = generate_output_key(ref)
        assert result == "prod__env_east__collector"
        # Consecutive hyphens become consecutive underscores (valid)
    
    def test_leading_hyphen_conversion(self):
        """Test that leading hyphens are converted to underscores."""
        ref = SubAgentReference(
            name="-collector",
            namespace="-production"
        )
        result = generate_output_key(ref)
        assert result == "_production__collector"
    
    def test_trailing_hyphen_conversion(self):
        """Test that trailing hyphens are converted to underscores."""
        ref = SubAgentReference(
            name="collector-",
            namespace="production-"
        )
        result = generate_output_key(ref)
        assert result == "production__collector_"
    
    def test_mixed_case_with_hyphens(self):
        """Test mixed case names with hyphens."""
        ref = SubAgentReference(
            name="East-US-Collector",
            namespace="Prod-Env"
        )
        result = generate_output_key(ref)
        assert result == "Prod_Env_East_US_Collector"


class TestOutputKeyEdgeCases:
    """Test edge cases and boundary conditions for outputKey generation."""
    
    def test_unicode_characters_in_name(self):
        """Test that unicode characters in names cause validation error."""
        ref = SubAgentReference(
            name="collector-café",
            namespace="production"
        )
        # Unicode characters should cause pattern validation to fail
        with pytest.raises(ValueError) as exc_info:
            generate_output_key(ref)
        assert "contains invalid characters" in str(exc_info.value)
    
    def test_special_kubernetes_chars(self):
        """Test that other Kubernetes-valid chars (like dots) are rejected."""
        ref = SubAgentReference(
            name="collector.service",
            namespace="production"
        )
        with pytest.raises(ValueError) as exc_info:
            generate_output_key(ref)
        assert "contains invalid characters" in str(exc_info.value)
    
    def test_explicit_key_bypasses_length_validation(self):
        """Test that explicit outputKey can exceed 100 chars (controller validates)."""
        # Explicit keys bypass ADK validation (CRD validation handles it)
        long_key = "x" * 150
        ref = SubAgentReference(
            name="collector",
            namespace="production",
            output_key=long_key
        )
        result = generate_output_key(ref)
        assert result == long_key
        assert len(result) == 150
    
    def test_explicit_key_with_invalid_chars(self):
        """Test that explicit outputKey with invalid chars is returned as-is."""
        # Explicit keys bypass ADK validation (CRD validation handles it)
        invalid_key = "my-invalid-key"
        ref = SubAgentReference(
            name="collector",
            namespace="production",
            output_key=invalid_key
        )
        result = generate_output_key(ref)
        assert result == invalid_key
    
    def test_whitespace_in_names(self):
        """Test that whitespace in names causes validation error."""
        ref = SubAgentReference(
            name="my collector",
            namespace="production"
        )
        with pytest.raises(ValueError) as exc_info:
            generate_output_key(ref)
        assert "contains invalid characters" in str(exc_info.value)
    
    def test_empty_explicit_output_key(self):
        """Test that empty string for explicit output_key triggers auto-generation."""
        ref = SubAgentReference(
            name="collector",
            namespace="production",
            output_key=""  # Empty string should be falsy, triggers auto-generation
        )
        result = generate_output_key(ref)
        # Empty string is falsy in Python, so auto-generation should kick in
        assert result == "production_collector"


class TestOutputKeyIntegration:
    """Test outputKey generation in realistic workflow scenarios."""
    
    def test_multiple_agents_unique_keys(self):
        """Test that multiple agents generate unique outputKeys."""
        refs = [
            SubAgentReference(name="east-collector", namespace="production"),
            SubAgentReference(name="west-collector", namespace="production"),
            SubAgentReference(name="central-collector", namespace="staging"),
        ]
        
        keys = [generate_output_key(ref) for ref in refs]
        
        # All keys should be unique
        assert len(keys) == len(set(keys))
        assert keys == [
            "production_east_collector",
            "production_west_collector",
            "staging_central_collector",
        ]
    
    def test_same_name_different_namespace(self):
        """Test that same agent name in different namespaces generates unique keys."""
        ref1 = SubAgentReference(name="collector", namespace="production")
        ref2 = SubAgentReference(name="collector", namespace="staging")
        
        key1 = generate_output_key(ref1)
        key2 = generate_output_key(ref2)
        
        assert key1 != key2
        assert key1 == "production_collector"
        assert key2 == "staging_collector"
    
    def test_mixed_automatic_and_explicit(self):
        """Test workflow with mix of automatic and explicit outputKeys."""
        refs = [
            SubAgentReference(name="east-collector", namespace="production"),
            SubAgentReference(name="west-collector", namespace="production", output_key="west_data"),
            SubAgentReference(name="central-collector", namespace="staging"),
        ]
        
        keys = [generate_output_key(ref) for ref in refs]
        
        assert keys == [
            "production_east_collector",  # Automatic
            "west_data",  # Explicit
            "staging_central_collector",  # Automatic
        ]
    
    def test_real_world_agent_names(self):
        """Test with realistic agent names from examples."""
        refs = [
            SubAgentReference(name="k8s-agent", namespace="default"),
            SubAgentReference(name="istio-agent", namespace="istio-system"),
            SubAgentReference(name="helm-agent", namespace="kube-system"),
        ]
        
        keys = [generate_output_key(ref) for ref in refs]
        
        assert keys == [
            "default_k8s_agent",
            "istio_system_istio_agent",
            "kube_system_helm_agent",
        ]


class TestOutputKeyDocumentation:
    """Test that documented examples work correctly."""
    
    def test_docstring_example_basic(self):
        """Test the basic example from generate_output_key docstring."""
        ref = SubAgentReference(name="east-collector", namespace="production")
        result = generate_output_key(ref)
        assert result == "production_east_collector"
    
    def test_docstring_example_explicit(self):
        """Test the explicit override example from docstring."""
        ref = SubAgentReference(
            name="west-collector",
            namespace="prod",
            output_key="west_data"
        )
        result = generate_output_key(ref)
        assert result == "west_data"
    
    def test_quickstart_example(self):
        """Test examples from quickstart.md documentation."""
        refs = [
            SubAgentReference(name="east-collector", namespace="production"),
            SubAgentReference(name="west-collector", namespace="production"),
            SubAgentReference(name="central-collector", namespace="staging"),
        ]
        
        keys = [generate_output_key(ref) for ref in refs]
        
        # These should match the comments in quickstart.md
        assert keys[0] == "production_east_collector"
        assert keys[1] == "production_west_collector"
        assert keys[2] == "staging_central_collector"

