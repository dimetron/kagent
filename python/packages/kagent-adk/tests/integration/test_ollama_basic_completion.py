"""Integration test for basic Ollama completion using ollama-python.

This test validates the end-to-end flow of creating OllamaNative,
calling AsyncClient.chat(), and converting responses to LlmResponse.

Task: T008
"""

import pytest
from google.genai import types

# Will be implemented when OllamaNative is complete
pytestmark = pytest.mark.asyncio


class TestOllamaBasicCompletion:
    """Integration tests for basic chat completion without tools."""
    
    async def test_simple_text_completion(self):
        """Test basic text completion with single user message."""
        from kagent.adk.models._ollama import OllamaNative
        
        # Create OllamaNative instance
        ollama_native = OllamaNative(
            type="ollama",
            model="llama2",
            base_url="http://localhost:11434"
        )
        
        # Create simple LlmRequest
        from google.adk.models.llm_request import LlmRequest
        request = LlmRequest(
            contents=[
                types.Content(
                    role="user",
                    parts=[types.Part(text="Say hello")]
                )
            ]
        )
        
        # Generate content (non-streaming)
        response = None
        async for resp in ollama_native.generate_content_async(request, stream=False):
            response = resp
        
        # Validate response
        assert response is not None
        assert response.content is not None
        assert len(response.content.parts) > 0
        assert response.content.parts[0].text
        assert isinstance(response.content.parts[0].text, str)
    
    async def test_completion_with_system_instruction(self):
        """Test completion with system instruction."""
        from kagent.adk.models._ollama import OllamaNative
        from google.adk.models.llm_request import LlmRequest
        
        ollama_native = OllamaNative(
            type="ollama",
            model="llama2"
        )
        
        request = LlmRequest(
            contents=[
                types.Content(
                    role="user",
                    parts=[types.Part(text="What should I say?")]
                )
            ],
            system_instruction="You are a helpful assistant. Always be polite."
        )
        
        response = None
        async for resp in ollama_native.generate_content_async(request, stream=False):
            response = resp
        
        assert response is not None
        assert response.content is not None
    
    async def test_completion_with_temperature(self):
        """Test completion with custom temperature setting."""
        from kagent.adk.models._ollama import OllamaNative
        from google.adk.models.llm_request import LlmRequest
        
        ollama_native = OllamaNative(
            type="ollama",
            model="llama2",
            temperature=0.5
        )
        
        request = LlmRequest(
            contents=[
                types.Content(
                    role="user",
                    parts=[types.Part(text="Generate a number")]
                )
            ]
        )
        
        response = None
        async for resp in ollama_native.generate_content_async(request, stream=False):
            response = resp
        
        assert response is not None
        assert response.content is not None
    
    async def test_completion_with_max_tokens(self):
        """Test completion with max_tokens limit."""
        from kagent.adk.models._ollama import OllamaNative
        from google.adk.models.llm_request import LlmRequest
        
        ollama_native = OllamaNative(
            type="ollama",
            model="llama2",
            max_tokens=50
        )
        
        request = LlmRequest(
            contents=[
                types.Content(
                    role="user",
                    parts=[types.Part(text="Write a story")]
                )
            ]
        )
        
        response = None
        async for resp in ollama_native.generate_content_async(request, stream=False):
            response = resp
        
        assert response is not None
        # Response should exist even with token limit
    
    async def test_multi_turn_conversation(self):
        """Test multi-turn conversation with history."""
        from kagent.adk.models._ollama import OllamaNative
        from google.adk.models.llm_request import LlmRequest
        
        ollama_native = OllamaNative(
            type="ollama",
            model="llama2"
        )
        
        request = LlmRequest(
            contents=[
                types.Content(
                    role="user",
                    parts=[types.Part(text="My name is Alice")]
                ),
                types.Content(
                    role="model",
                    parts=[types.Part(text="Nice to meet you, Alice!")]
                ),
                types.Content(
                    role="user",
                    parts=[types.Part(text="What is my name?")]
                )
            ]
        )
        
        response = None
        async for resp in ollama_native.generate_content_async(request, stream=False):
            response = resp
        
        assert response is not None
        assert response.content is not None
        # Model should remember the name from conversation history


if __name__ == "__main__":
    pytest.main([__file__, "-v"])

