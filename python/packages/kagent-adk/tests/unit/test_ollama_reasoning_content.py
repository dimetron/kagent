"""Unit tests for Ollama reasoning content extraction.

This module tests the extraction and mapping of the 'thinking' field from Ollama
responses to the ADK's 'thought' field in Part structures.

Task: E001
"""

import pytest
from google.genai import types

from kagent.adk.models._ollama import _convert_ollama_response_to_llm_response


class TestReasoningContentExtraction:
    """Test reasoning content extraction from Ollama responses."""

    def test_thinking_field_mapped_to_thought(self):
        """Test that Ollama's 'thinking' field is mapped to custom_metadata.reasoning_content."""
        # Ollama response with thinking field
        ollama_response = {
            "model": "qwen-qwq:latest",
            "created_at": "2025-01-01T00:00:00Z",
            "message": {
                "role": "assistant",
                "content": "The answer is 42.",
                "thinking": "Let me think about this carefully. First, I need to understand the question..."
            },
            "done": True,
            "total_duration": 1000000000,
            "prompt_eval_count": 10,
            "eval_count": 20
        }

        llm_response = _convert_ollama_response_to_llm_response(ollama_response)

        # Verify response structure
        assert llm_response.content is not None
        assert len(llm_response.content.parts) >= 1

        # Check reasoning content in custom_metadata
        assert llm_response.custom_metadata is not None
        assert "reasoning_content" in llm_response.custom_metadata
        assert "think about this carefully" in llm_response.custom_metadata["reasoning_content"]

    def test_empty_thinking_field_ignored(self):
        """Test that empty thinking field doesn't create reasoning content."""
        ollama_response = {
            "model": "llama3.3",
            "message": {
                "role": "assistant",
                "content": "Hello!",
                "thinking": ""
            },
            "done": True
        }

        llm_response = _convert_ollama_response_to_llm_response(ollama_response)

        # Empty thinking should not create custom_metadata
        assert llm_response.custom_metadata is None

    def test_missing_thinking_field(self):
        """Test that missing thinking field doesn't cause errors."""
        ollama_response = {
            "model": "llama2",
            "message": {
                "role": "assistant",
                "content": "Hello!"
            },
            "done": True
        }

        llm_response = _convert_ollama_response_to_llm_response(ollama_response)

        # Should work without thinking field
        assert llm_response.content is not None
        assert len(llm_response.content.parts) >= 1
        
        # No reasoning content should be present
        assert llm_response.custom_metadata is None

    def test_thinking_with_empty_content(self):
        """Test that thinking works even when content is empty."""
        ollama_response = {
            "model": "qwen-qwq:latest",
            "message": {
                "role": "assistant",
                "content": "",
                "thinking": "This is complex reasoning..."
            },
            "done": True
        }

        llm_response = _convert_ollama_response_to_llm_response(ollama_response)

        # Should have reasoning content in custom_metadata
        assert llm_response.custom_metadata is not None
        assert "reasoning_content" in llm_response.custom_metadata
        assert "complex reasoning" in llm_response.custom_metadata["reasoning_content"]
        
        # Should also have a text part with the thought (since content is empty)
        assert len(llm_response.content.parts) >= 1

    def test_thinking_in_streaming_partial_response(self):
        """Test thinking field in streaming partial responses."""
        # First chunk with thinking
        chunk1 = {
            "model": "llama3.3",
            "message": {
                "role": "assistant",
                "content": "",
                "thinking": "Let me analyze"
            },
            "done": False
        }

        llm_response1 = _convert_ollama_response_to_llm_response(chunk1)
        assert llm_response1.partial is True
        
        # Check reasoning content in custom_metadata
        assert llm_response1.custom_metadata is not None
        assert "reasoning_content" in llm_response1.custom_metadata

        # Second chunk with more content
        chunk2 = {
            "model": "llama3.3",
            "message": {
                "role": "assistant",
                "content": "The answer",
                "thinking": "Let me analyze this problem"
            },
            "done": False
        }

        llm_response2 = _convert_ollama_response_to_llm_response(chunk2)
        assert llm_response2.partial is True
        assert llm_response2.custom_metadata is not None

    def test_thinking_in_final_streaming_chunk(self):
        """Test thinking field in final streaming chunk with done=True."""
        chunk = {
            "model": "qwen-qwq:latest",
            "message": {
                "role": "assistant",
                "content": "Final answer: 42",
                "thinking": "After careful consideration..."
            },
            "done": True,
            "prompt_eval_count": 10,
            "eval_count": 20
        }

        llm_response = _convert_ollama_response_to_llm_response(chunk)

        assert llm_response.partial is False
        assert llm_response.usage_metadata is not None
        
        # Check reasoning content in custom_metadata
        assert llm_response.custom_metadata is not None
        assert "reasoning_content" in llm_response.custom_metadata

    def test_thinking_with_tool_calls(self):
        """Test that thinking field works alongside tool calls."""
        ollama_response = {
            "model": "qwen-qwq:latest",
            "message": {
                "role": "assistant",
                "content": "",
                "thinking": "I should call the weather tool",
                "tool_calls": [
                    {
                        "function": {
                            "name": "get_weather",
                            "arguments": {"location": "Paris"}
                        }
                    }
                ]
            },
            "done": True
        }

        llm_response = _convert_ollama_response_to_llm_response(ollama_response)

        # Should have reasoning content and function_call parts
        assert llm_response.custom_metadata is not None
        assert "reasoning_content" in llm_response.custom_metadata
        assert "weather tool" in llm_response.custom_metadata["reasoning_content"]
        
        function_parts = [part for part in llm_response.content.parts if part.function_call]
        assert len(function_parts) == 1


if __name__ == "__main__":
    pytest.main([__file__, "-v"])

