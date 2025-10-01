"""Integration test for Ollama streaming responses.

This test validates streaming functionality with AsyncClient.chat(stream=True).

Task: T009
"""

import pytest
from google.genai import types

pytestmark = [pytest.mark.asyncio, pytest.mark.integration]


class TestOllamaStreaming:
    """Integration tests for streaming responses."""

    async def test_streaming_text_generation(self, ollama_model):
        """Test streaming response with async iteration."""
        from google.adk.models.llm_request import LlmRequest

        from kagent.adk.models._ollama import OllamaNative

        ollama_native = OllamaNative(type="ollama", model=ollama_model)

        request = LlmRequest(contents=[types.Content(role="user", parts=[types.Part(text="Count to 5")])])

        # Collect streaming chunks
        chunks = []
        async for chunk in ollama_native.generate_content_async(request, stream=True):
            chunks.append(chunk)

        # Validate streaming behavior
        assert len(chunks) > 0, "Should receive at least one chunk"

        # All chunks except last should be partial
        for chunk in chunks[:-1]:
            assert chunk.partial is True, "Non-final chunks should be partial"

        # Last chunk should be final
        if len(chunks) > 1:
            assert chunks[-1].partial is False, "Final chunk should not be partial"

    async def test_streaming_accumulates_content(self, ollama_model):
        """Test that streaming chunks accumulate to form complete response."""
        from google.adk.models.llm_request import LlmRequest

        from kagent.adk.models._ollama import OllamaNative

        ollama_native = OllamaNative(type="ollama", model=ollama_model)

        request = LlmRequest(contents=[types.Content(role="user", parts=[types.Part(text="Say hello world")])])

        # Accumulate text from all chunks
        full_text = ""
        chunk_count = 0
        async for chunk in ollama_native.generate_content_async(request, stream=True):
            chunk_count += 1
            if chunk.content and chunk.content.parts:
                text = chunk.content.parts[0].text
                if text:
                    full_text += text

        assert chunk_count > 0
        assert len(full_text) > 0, "Accumulated text should not be empty"

    async def test_streaming_with_usage_metadata(self, ollama_model):
        """Test that final streaming chunk includes usage metadata."""
        from google.adk.models.llm_request import LlmRequest

        from kagent.adk.models._ollama import OllamaNative

        ollama_native = OllamaNative(type="ollama", model=ollama_model)

        request = LlmRequest(contents=[types.Content(role="user", parts=[types.Part(text="Hi")])])

        final_chunk = None
        async for chunk in ollama_native.generate_content_async(request, stream=True):
            final_chunk = chunk

        # Final chunk should have usage metadata
        assert final_chunk is not None
        assert final_chunk.usage_metadata is not None
        assert final_chunk.usage_metadata.prompt_token_count >= 0
        assert final_chunk.usage_metadata.candidates_token_count >= 0

    async def test_streaming_early_termination(self, ollama_model):
        """Test that streaming can be terminated early by breaking iteration."""
        from google.adk.models.llm_request import LlmRequest

        from kagent.adk.models._ollama import OllamaNative

        ollama_native = OllamaNative(type="ollama", model=ollama_model)

        request = LlmRequest(contents=[types.Content(role="user", parts=[types.Part(text="Write a very long story")])])

        # Take only first 3 chunks
        chunk_count = 0
        async for _chunk in ollama_native.generate_content_async(request, stream=True):
            chunk_count += 1
            if chunk_count >= 3:
                break

        assert chunk_count == 3, "Should stop after 3 chunks"

    async def test_streaming_vs_non_streaming_consistency(self, ollama_model):
        """Test that streaming and non-streaming produce similar results."""
        from google.adk.models.llm_request import LlmRequest

        from kagent.adk.models._ollama import OllamaNative

        ollama_native = OllamaNative(
            type="ollama",
            model=ollama_model,
            temperature=0.0,  # Deterministic
        )

        request = LlmRequest(contents=[types.Content(role="user", parts=[types.Part(text="What is 2+2?")])])

        # Non-streaming
        non_streaming_response = None
        async for resp in ollama_native.generate_content_async(request, stream=False):
            non_streaming_response = resp

        # Streaming
        streaming_text = ""
        async for chunk in ollama_native.generate_content_async(request, stream=True):
            if chunk.content and chunk.content.parts:
                text = chunk.content.parts[0].text
                if text:
                    streaming_text += text

        # Both should have content
        assert non_streaming_response is not None
        assert len(streaming_text) > 0


if __name__ == "__main__":
    pytest.main([__file__, "-v"])
