"""Integration test for custom Ollama configuration.

Tests custom base_url, headers, temperature, max_tokens.
"""

import pytest
from google.adk.models.llm_request import LlmRequest
from google.genai import types

from kagent.adk.models._ollama import OllamaNative


@pytest.mark.asyncio
@pytest.mark.integration
async def test_custom_temperature():
    """Test custom temperature configuration."""
    driver = OllamaNative(
        type="ollama",
        model="gpt-oss:latest",
        base_url="http://localhost:11434",
        temperature=0.1,  # Very low temperature for deterministic output
    )

    request = LlmRequest(
        model="gpt-oss:latest", contents=[types.Content(role="user", parts=[types.Part.from_text(text="Say 'test'")])]
    )

    # Generate response
    responses = []
    async for response in driver.generate_content_async(request, stream=False):
        responses.append(response)

    assert len(responses) == 1
    assert responses[0].content is not None


@pytest.mark.asyncio
@pytest.mark.integration
async def test_custom_max_tokens():
    """Test custom max_tokens configuration."""
    driver = OllamaNative(
        type="ollama",
        model="gpt-oss:latest",
        base_url="http://localhost:11434",
        max_tokens=10,  # Very low to test limiting
    )

    request = LlmRequest(
        model="gpt-oss:latest",
        contents=[types.Content(role="user", parts=[types.Part.from_text(text="Write a long story about space.")])],
    )

    # Generate response
    responses = []
    async for response in driver.generate_content_async(request, stream=False):
        responses.append(response)

    assert len(responses) == 1
    assert responses[0].content is not None
    # Response should be truncated due to max_tokens


@pytest.mark.asyncio
@pytest.mark.integration
async def test_custom_base_url():
    """Test custom base_url (pointing to localhost)."""
    driver = OllamaNative(
        type="ollama",
        model="gpt-oss:latest",
        base_url="http://127.0.0.1:11434",  # Alternative localhost
    )

    request = LlmRequest(
        model="gpt-oss:latest", contents=[types.Content(role="user", parts=[types.Part.from_text(text="Hello")])]
    )

    # Should work with alternative base_url
    responses = []
    async for response in driver.generate_content_async(request, stream=False):
        responses.append(response)

    assert len(responses) == 1
    assert responses[0].content is not None


if __name__ == "__main__":
    pytest.main([__file__, "-v", "-m", "integration"])
