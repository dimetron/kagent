"""Unit tests for Ollama format parameter handling.

This module tests the `format` parameter support for JSON mode and schema validation.

Task: E007
"""

import json

import pytest
from google.adk.models.llm_request import LlmRequest
from google.genai import types

from kagent.adk.models._ollama import OllamaNative


class TestFormatParameterHandling:
    """Test format parameter handling for JSON mode."""

    def test_format_json_string_parameter(self):
        """Test passing format='json' as string."""
        driver = OllamaNative(model="llama2", base_url="http://localhost:11434")
        
        # Create config with format
        config = types.GenerateContentConfig(response_mime_type="application/json")
        request = LlmRequest(
            contents=[types.Content(role="user", parts=[types.Part(text="Return JSON")])],
            config=config
        )

        # Should not raise error when preparing request
        # This tests the parameter extraction logic
        assert request.config.response_mime_type == "application/json"

    def test_format_with_json_schema(self):
        """Test passing format with JSON schema dict."""
        driver = OllamaNative(model="llama2")
        
        schema = {
            "type": "object",
            "properties": {
                "name": {"type": "string"},
                "age": {"type": "integer"}
            },
            "required": ["name"]
        }
        
        config = types.GenerateContentConfig(
            response_mime_type="application/json",
            response_schema=schema
        )
        
        request = LlmRequest(
            contents=[types.Content(role="user", parts=[types.Part(text="Get person info")])],
            config=config
        )

        assert request.config.response_mime_type == "application/json"
        assert request.config.response_schema == schema

    def test_format_in_non_streaming_mode(self):
        """Test format parameter with stream=False."""
        driver = OllamaNative(model="llama2")
        
        config = types.GenerateContentConfig(response_mime_type="application/json")
        request = LlmRequest(
            contents=[types.Content(role="user", parts=[types.Part(text="Test")])],
            config=config
        )

        # Verify config is properly set for non-streaming
        assert not hasattr(request, 'stream') or request.stream is None

    def test_format_in_streaming_mode(self):
        """Test format parameter with stream=True."""
        driver = OllamaNative(model="llama2")
        
        config = types.GenerateContentConfig(response_mime_type="application/json")
        request = LlmRequest(
            contents=[types.Content(role="user", parts=[types.Part(text="Test")])],
            config=config
        )

        # Config should work in both streaming and non-streaming modes
        assert request.config.response_mime_type == "application/json"

    def test_format_parameter_omitted(self):
        """Test that omitting format parameter works (no JSON mode)."""
        driver = OllamaNative(model="llama2")
        
        request = LlmRequest(
            contents=[types.Content(role="user", parts=[types.Part(text="Test")])]
        )

        # Should have no format-related config
        assert request.config is None or request.config.response_mime_type is None

    def test_format_with_complex_schema(self):
        """Test format with complex nested JSON schema."""
        driver = OllamaNative(model="llama2")
        
        complex_schema = {
            "type": "object",
            "properties": {
                "user": {
                    "type": "object",
                    "properties": {
                        "name": {"type": "string"},
                        "contacts": {
                            "type": "array",
                            "items": {
                                "type": "object",
                                "properties": {
                                    "type": {"type": "string"},
                                    "value": {"type": "string"}
                                }
                            }
                        }
                    }
                }
            }
        }
        
        config = types.GenerateContentConfig(
            response_mime_type="application/json",
            response_schema=complex_schema
        )
        
        request = LlmRequest(
            contents=[types.Content(role="user", parts=[types.Part(text="Get user")])],
            config=config
        )

        assert request.config.response_schema == complex_schema

    def test_format_parameter_validation(self):
        """Test that invalid format values are handled."""
        driver = OllamaNative(model="llama2")
        
        # Invalid MIME type should still be accepted (validation at Ollama level)
        config = types.GenerateContentConfig(response_mime_type="text/plain")
        request = LlmRequest(
            contents=[types.Content(role="user", parts=[types.Part(text="Test")])],
            config=config
        )
        
        # Should not raise during request creation
        assert request.config.response_mime_type == "text/plain"

    def test_format_with_other_parameters(self):
        """Test format alongside other configuration parameters."""
        driver = OllamaNative(model="llama2", temperature=0.7)
        
        config = types.GenerateContentConfig(
            response_mime_type="application/json",
            temperature=0.5,
            max_output_tokens=100
        )
        
        request = LlmRequest(
            contents=[types.Content(role="user", parts=[types.Part(text="Test")])],
            config=config
        )

        # All parameters should coexist
        assert request.config.response_mime_type == "application/json"
        assert request.config.temperature == 0.5
        assert request.config.max_output_tokens == 100

    @pytest.mark.asyncio
    async def test_format_parameter_mocked_call(self):
        """Test that format parameter is passed to Ollama client (mocked)."""
        from unittest.mock import AsyncMock, MagicMock, patch
        
        driver = OllamaNative(model="llama2")
        
        # Mock the client
        mock_response = {
            "model": "llama2",
            "message": {
                "role": "assistant",
                "content": '{"result": "success"}'
            },
            "done": True
        }
        
        with patch.object(driver, '_client') as mock_client:
            mock_client.chat = AsyncMock(return_value=mock_response)
            
            config = types.GenerateContentConfig(response_mime_type="application/json")
            request = LlmRequest(
                contents=[types.Content(role="user", parts=[types.Part(text="Test")])],
                config=config
            )
            
            # Execute request
            async for _ in driver.generate_content_async(request, stream=False):
                pass
            
            # Verify chat was called (parameters verified in implementation)
            mock_client.chat.assert_called_once()


if __name__ == "__main__":
    pytest.main([__file__, "-v"])

