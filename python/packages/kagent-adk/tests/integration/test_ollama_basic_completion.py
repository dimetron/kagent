"""Integration test for basic Ollama completion.

This test validates the basic completion scenario from quickstart.md.
Requires Ollama running locally with a model pulled.
"""

import pytest
from google.adk.models.llm_request import LlmRequest
from google.genai import types

from kagent.adk.models._ollama import OllamaNative


@pytest.mark.asyncio
@pytest.mark.integration
async def test_basic_completion():
    """Test basic completion without streaming."""
    # Create OllamaNative instance (from quickstart.md)
    driver = OllamaNative(
        type="ollama",
        model="gpt-oss:latest",  # Assumes gpt-oss:latest is pulled
        base_url="http://localhost:11434",
    )

    # Create simple user message
    request = LlmRequest(
        model="gpt-oss:latest",
        contents=[
            types.Content(role="user", parts=[types.Part.from_text(text="What is 2+2? Answer with just the number.")])
        ],
    )

    # Generate response
    responses = []
    async for response in driver.generate_content_async(request, stream=False):
        responses.append(response)

    # Validate response
    assert len(responses) == 1
    response = responses[0]

    assert response.error_code is None
    assert response.error_message is None
    assert response.content is not None
    assert response.content.role == "model"
    assert len(response.content.parts) > 0
    assert response.content.parts[0].text is not None
    assert len(response.content.parts[0].text) > 0


@pytest.mark.asyncio
@pytest.mark.integration
async def test_completion_with_system_instruction():
    """Test completion with system instruction."""
    driver = OllamaNative(type="ollama", model="gpt-oss:latest", base_url="http://localhost:11434")

    # Create request with system instruction
    request = LlmRequest(
        model="gpt-oss:latest",
        contents=[types.Content(role="user", parts=[types.Part.from_text(text="Hello")])],
        config=types.GenerateContentConfig(system_instruction="You are a helpful assistant. Always respond politely."),
    )

    # Generate response
    responses = []
    async for response in driver.generate_content_async(request, stream=False):
        responses.append(response)

    # Validate response
    assert len(responses) == 1
    assert responses[0].content is not None
    assert responses[0].content.parts[0].text is not None


if __name__ == "__main__":
    pytest.main([__file__, "-v", "-m", "integration"])
