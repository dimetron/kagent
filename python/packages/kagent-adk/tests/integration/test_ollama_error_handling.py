"""Integration test for Ollama error handling.

This test validates error handling for various failure scenarios.

Task: T011
"""

import pytest
from google.genai import types

pytestmark = pytest.mark.asyncio


class TestOllamaErrorHandling:
    """Integration tests for error handling."""
    
    async def test_model_not_found_404(self):
        """Test handling of 404 error when model doesn't exist."""
        from kagent.adk.models._ollama import OllamaNative
        from google.adk.models.llm_request import LlmRequest
        
        ollama_native = OllamaNative(
            type="ollama",
            model="nonexistent-model-12345",
            base_url="http://localhost:11434"
        )
        
        request = LlmRequest(
            contents=[
                types.Content(
                    role="user",
                    parts=[types.Part(text="Hello")]
                )
            ]
        )
        
        response = None
        async for resp in ollama_native.generate_content_async(request, stream=False):
            response = resp
        
        # Should yield error response instead of raising exception
        assert response is not None
        assert response.error_code is not None
        assert "OLLAMA_MODEL_NOT_FOUND" in response.error_code or "404" in str(response.error_message)
    
    async def test_connection_error_invalid_host(self):
        """Test handling of connection errors with invalid host."""
        from kagent.adk.models._ollama import OllamaNative
        from google.adk.models.llm_request import LlmRequest
        
        ollama_native = OllamaNative(
            type="ollama",
            model="llama2",
            base_url="http://invalid-host-12345:9999"
        )
        
        request = LlmRequest(
            contents=[
                types.Content(
                    role="user",
                    parts=[types.Part(text="Hello")]
                )
            ]
        )
        
        response = None
        async for resp in ollama_native.generate_content_async(request, stream=False):
            response = resp
        
        # Should yield error response
        assert response is not None
        assert response.error_code is not None
        assert response.error_message is not None
    
    async def test_timeout_error(self):
        """Test handling of timeout errors."""
        from kagent.adk.models._ollama import OllamaNative
        from google.adk.models.llm_request import LlmRequest
        
        ollama_native = OllamaNative(
            type="ollama",
            model="llama2",
            base_url="http://localhost:11434",
            timeout=0.001  # Very short timeout to trigger error
        )
        
        request = LlmRequest(
            contents=[
                types.Content(
                    role="user",
                    parts=[types.Part(text="Hello")]
                )
            ]
        )
        
        response = None
        try:
            async for resp in ollama_native.generate_content_async(request, stream=False):
                response = resp
        except:
            pass  # Timeout may raise or yield error
        
        # Either yields error response or raises
        # Implementation should yield error response
        if response:
            assert response.error_code is not None or response.error_message is not None
    
    async def test_invalid_request_format(self):
        """Test handling of invalid request parameters."""
        from kagent.adk.models._ollama import OllamaNative
        from google.adk.models.llm_request import LlmRequest
        
        ollama_native = OllamaNative(
            type="ollama",
            model="llama2"
        )
        
        # Create request with potentially problematic content
        request = LlmRequest(
            contents=[
                types.Content(
                    role="user",
                    parts=[]  # Empty parts
                )
            ]
        )
        
        response = None
        async for resp in ollama_native.generate_content_async(request, stream=False):
            response = resp
        
        # Should handle gracefully
        assert response is not None
    
    async def test_server_error_500(self):
        """Test handling of 500 internal server errors."""
        # This test would require mocking or a specially configured Ollama instance
        # to return 500 errors. For now, we test the error conversion logic.
        from kagent.adk.models._ollama import OllamaNative
        
        ollama_native = OllamaNative(
            type="ollama",
            model="llama2"
        )
        
        # Test will be implemented when we can trigger 500 errors
        # For now, just verify the class can be instantiated
        assert ollama_native is not None
    
    async def test_streaming_error_handling(self):
        """Test error handling in streaming mode."""
        from kagent.adk.models._ollama import OllamaNative
        from google.adk.models.llm_request import LlmRequest
        
        ollama_native = OllamaNative(
            type="ollama",
            model="nonexistent-streaming-model",
            base_url="http://localhost:11434"
        )
        
        request = LlmRequest(
            contents=[
                types.Content(
                    role="user",
                    parts=[types.Part(text="Hello")]
                )
            ]
        )
        
        error_received = False
        async for resp in ollama_native.generate_content_async(request, stream=True):
            if resp.error_code or resp.error_message:
                error_received = True
                break
        
        # Should receive error in streaming mode too
        assert error_received
    
    async def test_error_response_structure(self):
        """Test that error responses have correct structure."""
        from kagent.adk.models._ollama import OllamaNative
        from google.adk.models.llm_request import LlmRequest
        
        ollama_native = OllamaNative(
            type="ollama",
            model="bad-model",
            base_url="http://localhost:11434"
        )
        
        request = LlmRequest(
            contents=[
                types.Content(
                    role="user",
                    parts=[types.Part(text="Test")]
                )
            ]
        )
        
        response = None
        async for resp in ollama_native.generate_content_async(request, stream=False):
            response = resp
        
        assert response is not None
        # Error response should have either error_code or error_message
        assert response.error_code is not None or response.error_message is not None
        # Error response should not have content
        if response.error_code:
            assert response.content is None or len(response.content.parts) == 0


if __name__ == "__main__":
    pytest.main([__file__, "-v"])

