"""Unit tests for OllamaNative.generate_content_async with mocked AsyncClient.

This tests the main generation logic with mocked ollama.AsyncClient to avoid
requiring a running Ollama server.

Task: Additional coverage for T020-T024
"""

from unittest.mock import AsyncMock, MagicMock, patch

import pytest
from google.adk.models.llm_request import LlmRequest
from google.genai import types
from google.genai.types import GenerateContentConfig


class TestOllamaNativeGenerateContent:
    """Unit tests for generate_content_async with mocked client."""

    @pytest.mark.asyncio
    async def test_generate_content_non_streaming_success(self):
        """Test non-streaming generation with mocked response."""
        from kagent.adk.models._ollama import OllamaNative

        ollama_native = OllamaNative(type="ollama", model="llama2")

        # Mock the AsyncClient.chat method
        mock_response = {
            "model": "llama2",
            "message": {"role": "assistant", "content": "Hello!"},
            "done": True,
            "prompt_eval_count": 5,
            "eval_count": 10,
        }

        with patch.object(ollama_native, "_client") as mock_client:
            mock_client.chat = AsyncMock(return_value=mock_response)

            # Create request
            request = LlmRequest(
                contents=[types.Content(role="user", parts=[types.Part(text="Hi")])],
                config=GenerateContentConfig(),
            )

            # Generate content
            result = None
            async for response in ollama_native.generate_content_async(request, stream=False):
                result = response

            # Verify response
            assert result is not None
            assert result.content is not None
            assert len(result.content.parts) > 0
            assert result.content.parts[0].text == "Hello!"
            assert result.usage_metadata is not None
            assert result.usage_metadata.prompt_token_count == 5
            assert result.usage_metadata.candidates_token_count == 10

    @pytest.mark.asyncio
    async def test_generate_content_streaming_success(self):
        """Test streaming generation with mocked chunks."""
        from kagent.adk.models._ollama import OllamaNative

        ollama_native = OllamaNative(type="ollama", model="llama2")

        # Mock streaming chunks
        async def mock_stream():
            chunks = [
                {"model": "llama2", "message": {"role": "assistant", "content": "Hello"}, "done": False},
                {"model": "llama2", "message": {"role": "assistant", "content": " world"}, "done": False},
                {
                    "model": "llama2",
                    "message": {"role": "assistant", "content": "!"},
                    "done": True,
                    "prompt_eval_count": 5,
                    "eval_count": 3,
                },
            ]
            for chunk in chunks:
                yield chunk

        with patch.object(ollama_native, "_client") as mock_client:
            mock_client.chat = AsyncMock(return_value=mock_stream())

            request = LlmRequest(
                contents=[types.Content(role="user", parts=[types.Part(text="Hi")])],
                config=GenerateContentConfig(),
            )

            # Collect all chunks
            chunks = []
            async for response in ollama_native.generate_content_async(request, stream=True):
                chunks.append(response)

            # Verify we got multiple chunks
            assert len(chunks) == 3
            assert chunks[0].partial is True
            assert chunks[-1].partial is False
            assert chunks[-1].usage_metadata is not None

    @pytest.mark.asyncio
    async def test_generate_content_with_system_instruction(self):
        """Test generation with system instruction."""
        from kagent.adk.models._ollama import OllamaNative

        ollama_native = OllamaNative(type="ollama", model="llama2")

        mock_response = {
            "model": "llama2",
            "message": {"role": "assistant", "content": "I will help you."},
            "done": True,
        }

        with patch.object(ollama_native, "_client") as mock_client:
            mock_client.chat = AsyncMock(return_value=mock_response)

            config = GenerateContentConfig(system_instruction="You are helpful")
            request = LlmRequest(
                contents=[types.Content(role="user", parts=[types.Part(text="Help me")])], config=config
            )

            result = None
            async for response in ollama_native.generate_content_async(request, stream=False):
                result = response

            assert result is not None
            # Verify chat was called with system instruction in messages
            call_kwargs = mock_client.chat.call_args[1]
            messages = call_kwargs["messages"]
            assert any(msg.get("role") == "system" for msg in messages)

    @pytest.mark.asyncio
    async def test_generate_content_with_tools(self):
        """Test generation with tools/function calling."""
        from google.genai.types import FunctionDeclaration, Schema, Tool

        from kagent.adk.models._ollama import OllamaNative

        ollama_native = OllamaNative(type="ollama", model="llama3.1")

        mock_response = {
            "model": "llama3.1",
            "message": {
                "role": "assistant",
                "content": "",
                "tool_calls": [
                    {
                        "id": "call_123",
                        "type": "function",
                        "function": {"name": "get_weather", "arguments": '{"location": "Paris"}'},
                    }
                ],
            },
            "done": True,
        }

        with patch.object(ollama_native, "_client") as mock_client:
            mock_client.chat = AsyncMock(return_value=mock_response)

            tool = Tool(
                function_declarations=[
                    FunctionDeclaration(
                        name="get_weather", description="Get weather", parameters=Schema(type="OBJECT", properties={})
                    )
                ]
            )

            config = GenerateContentConfig(tools=[tool])
            request = LlmRequest(
                contents=[types.Content(role="user", parts=[types.Part(text="Weather?")])], config=config
            )

            result = None
            async for response in ollama_native.generate_content_async(request, stream=False):
                result = response

            assert result is not None
            # Verify chat was called with tools
            call_kwargs = mock_client.chat.call_args[1]
            assert "tools" in call_kwargs
            assert call_kwargs["tools"] is not None

    @pytest.mark.asyncio
    async def test_generate_content_with_temperature(self):
        """Test generation with custom temperature."""
        from kagent.adk.models._ollama import OllamaNative

        ollama_native = OllamaNative(type="ollama", model="llama2", temperature=0.7)

        mock_response = {"model": "llama2", "message": {"role": "assistant", "content": "Response"}, "done": True}

        with patch.object(ollama_native, "_client") as mock_client:
            mock_client.chat = AsyncMock(return_value=mock_response)

            request = LlmRequest(
                contents=[types.Content(role="user", parts=[types.Part(text="Test")])], config=GenerateContentConfig()
            )

            async for _ in ollama_native.generate_content_async(request, stream=False):
                pass

            # Verify options included temperature
            call_kwargs = mock_client.chat.call_args[1]
            assert "options" in call_kwargs
            assert call_kwargs["options"]["temperature"] == 0.7

    @pytest.mark.asyncio
    async def test_generate_content_error_404(self):
        """Test error handling for model not found (404)."""
        import ollama

        from kagent.adk.models._ollama import OllamaNative

        ollama_native = OllamaNative(type="ollama", model="nonexistent")

        with patch.object(ollama_native, "_client") as mock_client:
            error = ollama.ResponseError(error="model not found", status_code=404)
            mock_client.chat = AsyncMock(side_effect=error)

            request = LlmRequest(
                contents=[types.Content(role="user", parts=[types.Part(text="Test")])], config=GenerateContentConfig()
            )

            result = None
            async for response in ollama_native.generate_content_async(request, stream=False):
                result = response

            assert result is not None
            assert result.error_code == "OLLAMA_MODEL_NOT_FOUND"
            assert "nonexistent" in result.error_message

    @pytest.mark.asyncio
    async def test_generate_content_error_500(self):
        """Test error handling for server error (500)."""
        import ollama

        from kagent.adk.models._ollama import OllamaNative

        ollama_native = OllamaNative(type="ollama", model="llama2")

        with patch.object(ollama_native, "_client") as mock_client:
            error = ollama.ResponseError(error="internal error", status_code=500)
            mock_client.chat = AsyncMock(side_effect=error)

            request = LlmRequest(
                contents=[types.Content(role="user", parts=[types.Part(text="Test")])], config=GenerateContentConfig()
            )

            result = None
            async for response in ollama_native.generate_content_async(request, stream=False):
                result = response

            assert result is not None
            assert result.error_code == "OLLAMA_SERVER_ERROR"

    @pytest.mark.asyncio
    async def test_generate_content_connection_error(self):
        """Test error handling for connection errors."""
        from kagent.adk.models._ollama import OllamaNative

        ollama_native = OllamaNative(type="ollama", model="llama2")

        with patch.object(ollama_native, "_client") as mock_client:
            mock_client.chat = AsyncMock(side_effect=ConnectionError("Connection refused"))

            request = LlmRequest(
                contents=[types.Content(role="user", parts=[types.Part(text="Test")])], config=GenerateContentConfig()
            )

            result = None
            async for response in ollama_native.generate_content_async(request, stream=False):
                result = response

            assert result is not None
            assert result.error_code == "OLLAMA_CONNECTION_ERROR"
            assert "Connection refused" in result.error_message

    @pytest.mark.asyncio
    async def test_generate_content_with_model_override(self):
        """Test that request model overrides instance model."""
        from kagent.adk.models._ollama import OllamaNative

        ollama_native = OllamaNative(type="ollama", model="llama2")

        mock_response = {"model": "mistral", "message": {"role": "assistant", "content": "Response"}, "done": True}

        with patch.object(ollama_native, "_client") as mock_client:
            mock_client.chat = AsyncMock(return_value=mock_response)

            request = LlmRequest(
                contents=[types.Content(role="user", parts=[types.Part(text="Test")])],
                config=GenerateContentConfig(),
                model="mistral",  # Override model
            )

            async for _ in ollama_native.generate_content_async(request, stream=False):
                pass

            # Verify chat was called with overridden model
            call_kwargs = mock_client.chat.call_args[1]
            assert call_kwargs["model"] == "mistral"


if __name__ == "__main__":
    pytest.main([__file__, "-v"])
