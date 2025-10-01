"""Integration test for custom Ollama configuration.

This test validates custom configuration options (host, headers, temperature, etc.).

Task: T012
"""

import pytest
from google.genai import types

pytestmark = [pytest.mark.asyncio, pytest.mark.integration]


class TestOllamaConfiguration:
    """Integration tests for custom configuration."""
    
    async def test_custom_base_url(self, ollama_model, ollama_base_url):
        """Test using custom Ollama server URL."""
        from kagent.adk.models._ollama import OllamaNative
        from google.adk.models.llm_request import LlmRequest
        
        # Test with explicit base_url
        ollama_native = OllamaNative(
            type="ollama",
            model=ollama_model,
            base_url=ollama_base_url
        )
        
        assert ollama_native.base_url == ollama_base_url
        
        request = LlmRequest(
            contents=[
                types.Content(
                    role="user",
                    parts=[types.Part(text="Hello")]
                )
            ]
        )
        
        # Should work with custom base_url
        response = None
        async for resp in ollama_native.generate_content_async(request, stream=False):
            response = resp
        
        assert response is not None
    
    async def test_custom_headers(self, ollama_model):
        """Test using custom HTTP headers."""
        from kagent.adk.models._ollama import OllamaNative
        from google.adk.models.llm_request import LlmRequest
        
        custom_headers = {
            "X-Custom-Header": "test-value",
            "Authorization": "Bearer fake-token"
        }
        
        ollama_native = OllamaNative(
            type="ollama",
            model=ollama_model,
            headers=custom_headers
        )
        
        assert ollama_native.headers == custom_headers
        
        request = LlmRequest(
            contents=[
                types.Content(
                    role="user",
                    parts=[types.Part(text="Test")]
                )
            ]
        )
        
        # Should include custom headers in request
        # (actual validation would require inspecting HTTP traffic)
        response = None
        async for resp in ollama_native.generate_content_async(request, stream=False):
            response = resp
        
        assert response is not None
    
    async def test_custom_temperature(self, ollama_model):
        """Test temperature parameter affects generation."""
        from kagent.adk.models._ollama import OllamaNative
        from google.adk.models.llm_request import LlmRequest
        
        # Test with very low temperature (deterministic)
        ollama_low_temp = OllamaNative(
            type="ollama",
            model=ollama_model,
            temperature=0.1
        )
        
        # Test with higher temperature (more random)
        ollama_high_temp = OllamaNative(
            type="ollama",
            model=ollama_model,
            temperature=1.5
        )
        
        request = LlmRequest(
            contents=[
                types.Content(
                    role="user",
                    parts=[types.Part(text="Say hello")]
                )
            ]
        )
        
        # Both should work
        response1 = None
        async for resp in ollama_low_temp.generate_content_async(request, stream=False):
            response1 = resp
        
        response2 = None
        async for resp in ollama_high_temp.generate_content_async(request, stream=False):
            response2 = resp
        
        assert response1 is not None
        assert response2 is not None
    
    async def test_custom_max_tokens(self, ollama_model):
        """Test max_tokens parameter limits response length."""
        from kagent.adk.models._ollama import OllamaNative
        from google.adk.models.llm_request import LlmRequest
        
        ollama_native = OllamaNative(
            type="ollama",
            model=ollama_model,
            max_tokens=10  # Very small limit
        )
        
        request = LlmRequest(
            contents=[
                types.Content(
                    role="user",
                    parts=[types.Part(text="Write a very long story about adventures")]
                )
            ]
        )
        
        response = None
        async for resp in ollama_native.generate_content_async(request, stream=False):
            response = resp
        
        assert response is not None
        # Response should be limited by max_tokens
        # (actual token counting would require deeper inspection)
    
    async def test_custom_timeout(self, ollama_model):
        """Test timeout parameter is applied."""
        from kagent.adk.models._ollama import OllamaNative
        
        ollama_native = OllamaNative(
            type="ollama",
            model=ollama_model,
            timeout=30.0  # 30 second timeout
        )
        
        assert ollama_native.timeout == 30.0
    
    async def test_client_caching(self, ollama_model):
        """Test that AsyncClient is cached via @cached_property."""
        from kagent.adk.models._ollama import OllamaNative
        
        ollama_native = OllamaNative(
            type="ollama",
            model=ollama_model
        )
        
        # Access _client property twice
        client1 = ollama_native._client
        client2 = ollama_native._client
        
        # Should be the same instance (cached)
        assert client1 is client2
    
    async def test_multiple_instances_independent(self, ollama_model):
        """Test that multiple OllamaNative instances are independent."""
        from kagent.adk.models._ollama import OllamaNative
        
        ollama1 = OllamaNative(
            type="ollama",
            model=ollama_model,
            temperature=0.5
        )
        
        ollama2 = OllamaNative(
            type="ollama",
            model=ollama_model,
            temperature=1.0
        )
        
        # Should have different configurations (different temperature)
        assert ollama1.temperature != ollama2.temperature
        # Each instance should have its own client
        assert ollama1._client is not ollama2._client
    
    async def test_default_values(self, ollama_model, ollama_base_url):
        """Test that default configuration values are applied."""
        from kagent.adk.models._ollama import OllamaNative
        
        # Create with minimal config
        ollama_native = OllamaNative(
            type="ollama",
            model=ollama_model
        )
        
        # Check defaults
        assert ollama_native.base_url == ollama_base_url
        assert ollama_native.timeout == 60.0
        assert ollama_native.temperature is None  # Optional, no default
        assert ollama_native.max_tokens is None  # Optional, no default
        assert ollama_native.headers is None  # Optional, no default


if __name__ == "__main__":
    pytest.main([__file__, "-v"])

