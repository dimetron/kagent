"""Integration test for Ollama error handling.

Tests connection errors, model not found, and API errors.
"""

import pytest
from google.adk.models.llm_request import LlmRequest
from google.genai import types

from kagent.adk.models._ollama import OllamaNative


@pytest.mark.asyncio
@pytest.mark.integration
async def test_connection_error():
    """Test handling of connection errors when Ollama is unreachable."""
    driver = OllamaNative(
        type="ollama",
        model="gpt-oss:latest",
        base_url="http://localhost:99999",  # Invalid port
        timeout=2.0,  # Short timeout
    )

    request = LlmRequest(
        model="gpt-oss:latest", contents=[types.Content(role="user", parts=[types.Part.from_text(text="Hello")])]
    )

    # Should yield error response
    responses = []
    async for response in driver.generate_content_async(request, stream=False):
        responses.append(response)

    assert len(responses) == 1
    response = responses[0]

    # Should have error information
    assert response.error_code is not None
    assert response.error_message is not None
    assert "connection" in response.error_message.lower() or "connect" in response.error_message.lower()


@pytest.mark.asyncio
@pytest.mark.integration
async def test_model_not_found_error():
    """Test handling of 404 model not found error."""
    driver = OllamaNative(type="ollama", model="nonexistent-model-xyz", base_url="http://localhost:11434")

    request = LlmRequest(
        model="nonexistent-model-xyz", contents=[types.Content(role="user", parts=[types.Part.from_text(text="Hello")])]
    )

    # Should yield error response
    responses = []
    async for response in driver.generate_content_async(request, stream=False):
        responses.append(response)

    assert len(responses) == 1
    response = responses[0]

    # Should have error information
    assert response.error_code is not None
    assert response.error_message is not None
    assert "not found" in response.error_message.lower() or "404" in response.error_message


if __name__ == "__main__":
    pytest.main([__file__, "-v", "-m", "integration"])
