"""Unit tests for Ollama native driver edge cases.

This module tests edge cases and error scenarios for the Ollama native driver,
including empty messages, missing fields, invalid URLs, timeouts, and error handling.
"""

import pytest
from unittest.mock import AsyncMock, MagicMock, patch
from google.genai import types
from google.adk.models.llm_request import LlmRequest
from kagent.adk.models._ollama import OllamaNative, _convert_content_to_ollama_messages
import ollama


class TestEmptyAndMissingData:
    """Test handling of empty and missing data."""

    def test_empty_messages_list(self):
        """Test conversion with empty messages list."""
        result = _convert_content_to_ollama_messages([])
        assert result == []

    def test_empty_content_parts(self):
        """Test conversion with empty content parts."""
        content = types.Content(role="user", parts=[])
        result = _convert_content_to_ollama_messages([content])
        # Should skip content with no text or function calls
        assert len(result) == 0

    def test_system_instruction_only(self):
        """Test with only system instruction, no messages."""
        result = _convert_content_to_ollama_messages([], system_instruction="You are helpful")
        assert len(result) == 1
        assert result[0]["role"] == "system"
        assert result[0]["content"] == "You are helpful"

    def test_missing_function_call_fields(self):
        """Test function call with missing name or args."""
        func_call = types.FunctionCall(id="call_1", name=None, args=None)
        content = types.Content(role="assistant", parts=[types.Part(function_call=func_call)])
        result = _convert_content_to_ollama_messages([content])
        assert len(result) == 1
        assert "tool_calls" in result[0]
        assert result[0]["tool_calls"][0]["function"]["name"] == ""
        assert result[0]["tool_calls"][0]["function"]["arguments"] == {}


class TestInvalidURLs:
    """Test handling of invalid URLs and connection errors."""

    @pytest.mark.asyncio
    async def test_invalid_host_url(self):
        """Test with invalid host URL."""
        driver = OllamaNative(model="llama2", base_url="not-a-valid-url")
        request = LlmRequest(
            contents=[types.Content(role="user", parts=[types.Part(text="Hello")])]
        )
        
        # Should yield error response
        responses = []
        async for response in driver.generate_content_async(request):
            responses.append(response)
        
        assert len(responses) == 1
        assert responses[0].error_code is not None
        assert "OLLAMA" in responses[0].error_code

    @pytest.mark.asyncio
    async def test_unreachable_host(self):
        """Test with unreachable host."""
        driver = OllamaNative(model="llama2", base_url="http://localhost:99999")
        request = LlmRequest(
            contents=[types.Content(role="user", parts=[types.Part(text="Hello")])]
        )
        
        responses = []
        async for response in driver.generate_content_async(request):
            responses.append(response)
        
        assert len(responses) == 1
        assert responses[0].error_code is not None


class TestTimeoutScenarios:
    """Test timeout handling."""

    @pytest.mark.asyncio
    async def test_timeout_configuration(self):
        """Test that timeout is properly configured."""
        driver = OllamaNative(model="llama2", timeout=1.0)
        assert driver.timeout == 1.0
        
        # Check that client is configured with timeout
        # httpx.AsyncClient.timeout is a Timeout object with read/write/connect/pool properties
        client = driver._client
        assert client._client.timeout.read == 1.0


class TestResponseErrorEdgeCases:
    """Test ollama.ResponseError edge cases."""

    @pytest.mark.asyncio
    async def test_response_error_404(self):
        """Test handling of 404 model not found error."""
        driver = OllamaNative(model="nonexistent-model")
        request = LlmRequest(
            contents=[types.Content(role="user", parts=[types.Part(text="Hello")])]
        )
        
        with patch.object(driver, '_client') as mock_client:
            # Simulate 404 error
            error = ollama.ResponseError("Model not found")
            error.status_code = 404
            mock_client.chat = AsyncMock(side_effect=error)
            
            responses = []
            async for response in driver.generate_content_async(request):
                responses.append(response)
            
            assert len(responses) == 1
            assert responses[0].error_code == "OLLAMA_MODEL_NOT_FOUND"
            assert "ollama pull" in responses[0].error_message

    @pytest.mark.asyncio
    async def test_response_error_503(self):
        """Test handling of 503 service unavailable error."""
        driver = OllamaNative(model="llama2")
        request = LlmRequest(
            contents=[types.Content(role="user", parts=[types.Part(text="Hello")])]
        )
        
        with patch.object(driver, '_client') as mock_client:
            # Simulate 503 error
            error = ollama.ResponseError("Service unavailable")
            error.status_code = 503
            mock_client.chat = AsyncMock(side_effect=error)
            
            responses = []
            async for response in driver.generate_content_async(request):
                responses.append(response)
            
            assert len(responses) == 1
            assert responses[0].error_code == "OLLAMA_SERVICE_UNAVAILABLE"

    @pytest.mark.asyncio
    async def test_response_error_500(self):
        """Test handling of 500 server error."""
        driver = OllamaNative(model="llama2")
        request = LlmRequest(
            contents=[types.Content(role="user", parts=[types.Part(text="Hello")])]
        )
        
        with patch.object(driver, '_client') as mock_client:
            # Simulate 500 error
            error = ollama.ResponseError("Internal server error")
            error.status_code = 500
            error.error = "Out of memory"
            mock_client.chat = AsyncMock(side_effect=error)
            
            responses = []
            async for response in driver.generate_content_async(request):
                responses.append(response)
            
            assert len(responses) == 1
            assert responses[0].error_code == "OLLAMA_SERVER_ERROR"

    @pytest.mark.asyncio
    async def test_response_error_no_status_code(self):
        """Test handling of ResponseError without status_code attribute."""
        driver = OllamaNative(model="llama2")
        request = LlmRequest(
            contents=[types.Content(role="user", parts=[types.Part(text="Hello")])]
        )
        
        with patch.object(driver, '_client') as mock_client:
            # Simulate error without status_code
            error = ollama.ResponseError("Unknown error")
            mock_client.chat = AsyncMock(side_effect=error)
            
            responses = []
            async for response in driver.generate_content_async(request):
                responses.append(response)
            
            assert len(responses) == 1
            assert responses[0].error_code == "OLLAMA_API_ERROR"


class TestMalformedResponses:
    """Test handling of malformed or unexpected responses."""

    @pytest.mark.asyncio
    async def test_response_missing_message_field(self):
        """Test response missing 'message' field."""
        driver = OllamaNative(model="llama2")
        request = LlmRequest(
            contents=[types.Content(role="user", parts=[types.Part(text="Hello")])]
        )
        
        with patch.object(driver, '_client') as mock_client:
            # Simulate response with missing message
            mock_response = {
                "model": "llama2",
                "done": True,
            }
            mock_client.chat = AsyncMock(return_value=mock_response)
            
            responses = []
            async for response in driver.generate_content_async(request):
                responses.append(response)
            
            # Should handle gracefully
            assert len(responses) == 1
            assert response.content is not None

    @pytest.mark.asyncio
    async def test_malformed_tool_call_arguments(self):
        """Test tool call with malformed JSON arguments."""
        driver = OllamaNative(model="llama2")
        request = LlmRequest(
            contents=[types.Content(role="user", parts=[types.Part(text="Hello")])]
        )
        
        with patch.object(driver, '_client') as mock_client:
            # Simulate response with malformed tool call
            mock_response = {
                "message": {
                    "role": "assistant",
                    "content": "",
                    "tool_calls": [{
                        "id": "call_1",
                        "function": {
                            "name": "test_func",
                            "arguments": "{not valid json"
                        }
                    }]
                },
                "model": "llama2",
                "done": True,
            }
            mock_client.chat = AsyncMock(return_value=mock_response)
            
            responses = []
            async for response in driver.generate_content_async(request):
                responses.append(response)
            
            # Should handle gracefully with empty args
            assert len(responses) == 1
            assert response.content is not None
            assert len(response.content.parts) > 0


class TestConfigurationEdgeCases:
    """Test configuration edge cases."""

    def test_default_base_url(self):
        """Test default base_url is set correctly."""
        driver = OllamaNative(model="llama2")
        assert driver.base_url == "http://localhost:11434"

    def test_custom_headers(self):
        """Test custom headers configuration."""
        headers = {"Authorization": "Bearer token123", "X-Custom": "value"}
        driver = OllamaNative(model="llama2", headers=headers)
        assert driver.headers == headers

    def test_none_temperature(self):
        """Test None temperature is handled correctly."""
        driver = OllamaNative(model="llama2", temperature=None)
        assert driver.temperature is None

    def test_zero_temperature(self):
        """Test zero temperature is handled correctly."""
        driver = OllamaNative(model="llama2", temperature=0.0)
        assert driver.temperature == 0.0

    def test_none_max_tokens(self):
        """Test None max_tokens is handled correctly."""
        driver = OllamaNative(model="llama2", max_tokens=None)
        assert driver.max_tokens is None


class TestClientCaching:
    """Test client instance caching."""

    def test_client_cached_property(self):
        """Test that _client is cached."""
        driver = OllamaNative(model="llama2")
        client1 = driver._client
        client2 = driver._client
        assert client1 is client2

    def test_client_configuration(self):
        """Test client is configured with correct parameters."""
        driver = OllamaNative(
            model="llama2",
            base_url="http://example.com:11434",
            timeout=30.0
        )
        client = driver._client
        assert client._client.base_url == "http://example.com:11434"
        # httpx.AsyncClient.timeout is a Timeout object with read/write/connect/pool properties
        assert client._client.timeout.read == 30.0

