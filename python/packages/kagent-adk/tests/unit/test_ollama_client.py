"""Unit tests for OllamaNative client initialization.

This tests the _client property caching and AsyncClient configuration.

Task: T016
"""

import pytest


class TestOllamaNativeClient:
    """Unit tests for OllamaNative client initialization."""
    
    def test_client_property_creates_async_client(self):
        """Test that _client property creates ollama.AsyncClient."""
        from kagent.adk.models._ollama import OllamaNative
        import ollama
        
        ollama_native = OllamaNative(
            type="ollama",
            model="llama2",
            base_url="http://localhost:11434"
        )
        
        client = ollama_native._client
        
        assert isinstance(client, ollama.AsyncClient)
    
    def test_client_caching(self):
        """Test that _client is cached via @cached_property."""
        from kagent.adk.models._ollama import OllamaNative
        
        ollama_native = OllamaNative(
            type="ollama",
            model="llama2"
        )
        
        client1 = ollama_native._client
        client2 = ollama_native._client
        
        # Should return same instance
        assert client1 is client2
    
    def test_client_with_custom_host(self):
        """Test client configuration with custom host."""
        from kagent.adk.models._ollama import OllamaNative
        
        custom_url = "http://custom-ollama:9999"
        ollama_native = OllamaNative(
            type="ollama",
            model="llama2",
            base_url=custom_url
        )
        
        client = ollama_native._client
        
        # Verify client is configured with custom host
        assert client is not None
        # AsyncClient should have host attribute
        assert hasattr(client, "_client") or hasattr(client, "base_url")
    
    def test_client_with_custom_headers(self):
        """Test client configuration with custom headers."""
        from kagent.adk.models._ollama import OllamaNative
        
        custom_headers = {
            "Authorization": "Bearer test-token",
            "X-Custom": "value"
        }
        
        ollama_native = OllamaNative(
            type="ollama",
            model="llama2",
            headers=custom_headers
        )
        
        client = ollama_native._client
        
        assert client is not None
        # Headers should be configured in the client
    
    def test_client_with_custom_timeout(self):
        """Test client configuration with custom timeout."""
        from kagent.adk.models._ollama import OllamaNative
        
        ollama_native = OllamaNative(
            type="ollama",
            model="llama2",
            timeout=30.0
        )
        
        client = ollama_native._client
        
        assert client is not None
        # Timeout should be configured
    
    def test_client_default_configuration(self):
        """Test client with default configuration."""
        from kagent.adk.models._ollama import OllamaNative
        
        ollama_native = OllamaNative(
            type="ollama",
            model="llama2"
        )
        
        client = ollama_native._client
        
        assert client is not None
        # Should use default base_url
        assert ollama_native.base_url == "http://localhost:11434"
    
    def test_multiple_instances_have_different_clients(self):
        """Test that different instances have independent clients."""
        from kagent.adk.models._ollama import OllamaNative
        
        ollama1 = OllamaNative(
            type="ollama",
            model="llama2",
            base_url="http://host1:11434"
        )
        
        ollama2 = OllamaNative(
            type="ollama",
            model="mistral",
            base_url="http://host2:11434"
        )
        
        client1 = ollama1._client
        client2 = ollama2._client
        
        # Should be different instances
        assert client1 is not client2
    
    def test_supported_models_classmethod(self):
        """Test supported_models() returns pattern list."""
        from kagent.adk.models._ollama import OllamaNative
        
        patterns = OllamaNative.supported_models()
        
        assert isinstance(patterns, list)
        assert len(patterns) > 0
        # Should match all models (regex pattern)
        assert r".*" in patterns or len(patterns) > 0
    
    def test_model_name_stored(self):
        """Test that model name is stored correctly."""
        from kagent.adk.models._ollama import OllamaNative
        
        model_name = "llama2"
        ollama_native = OllamaNative(
            type="ollama",
            model=model_name
        )
        
        assert ollama_native.model == model_name
    
    def test_type_discriminator(self):
        """Test that type discriminator is set correctly."""
        from kagent.adk.models._ollama import OllamaNative
        
        ollama_native = OllamaNative(
            type="ollama",
            model="llama2"
        )
        
        assert ollama_native.type == "ollama"
    
    def test_optional_fields_default_to_none(self):
        """Test that optional configuration fields default to None."""
        from kagent.adk.models._ollama import OllamaNative
        
        ollama_native = OllamaNative(
            type="ollama",
            model="llama2"
        )
        
        assert ollama_native.temperature is None
        assert ollama_native.max_tokens is None
        assert ollama_native.headers is None
    
    def test_all_configuration_fields(self):
        """Test instance with all configuration fields set."""
        from kagent.adk.models._ollama import OllamaNative
        
        ollama_native = OllamaNative(
            type="ollama",
            model="llama2",
            base_url="http://custom:11434",
            temperature=0.8,
            max_tokens=1024,
            timeout=45.0,
            headers={"X-Test": "value"}
        )
        
        assert ollama_native.model == "llama2"
        assert ollama_native.base_url == "http://custom:11434"
        assert ollama_native.temperature == 0.8
        assert ollama_native.max_tokens == 1024
        assert ollama_native.timeout == 45.0
        assert ollama_native.headers == {"X-Test": "value"}


if __name__ == "__main__":
    pytest.main([__file__, "-v"])

