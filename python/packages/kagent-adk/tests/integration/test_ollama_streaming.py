"""Integration test for Ollama streaming responses.

Tests streaming with partial=True responses.
Requires Ollama running locally.
"""

import pytest
from google.adk.models.llm_request import LlmRequest
from google.genai import types

from kagent.adk.models._ollama import OllamaNative


@pytest.mark.asyncio
@pytest.mark.integration
async def test_streaming_response():
    """Test streaming responses from Ollama."""
    driver = OllamaNative(type="ollama", model="gpt-oss:latest", base_url="http://localhost:11434")

    request = LlmRequest(
        model="gpt-oss:latest",
        contents=[types.Content(role="user", parts=[types.Part.from_text(text="Count from 1 to 5.")])],
    )

    # Generate streaming response
    responses = []
    async for response in driver.generate_content_async(request, stream=True):
        responses.append(response)

    # Validate streaming
    assert len(responses) > 1, "Should receive multiple chunks when streaming"

    # Most responses should be partial
    partial_responses = [r for r in responses if r.partial]
    assert len(partial_responses) > 0, "Should have partial responses"

    # Last response might be complete
    # All should have content
    for response in responses:
        assert response.content is not None
        assert len(response.content.parts) > 0


@pytest.mark.asyncio
@pytest.mark.integration
async def test_streaming_accumulation():
    """Test that streaming chunks can be accumulated into full response."""
    driver = OllamaNative(type="ollama", model="gpt-oss:latest", base_url="http://localhost:11434")

    request = LlmRequest(
        model="gpt-oss:latest",
        contents=[types.Content(role="user", parts=[types.Part.from_text(text="Say 'Hello World'")])],
    )

    # Accumulate streaming response
    accumulated_text = ""
    async for response in driver.generate_content_async(request, stream=True):
        if response.content and response.content.parts:
            for part in response.content.parts:
                if part.text:
                    accumulated_text += part.text

    # Validate accumulated response
    assert len(accumulated_text) > 0
    assert isinstance(accumulated_text, str)


if __name__ == "__main__":
    pytest.main([__file__, "-v", "-m", "integration"])
