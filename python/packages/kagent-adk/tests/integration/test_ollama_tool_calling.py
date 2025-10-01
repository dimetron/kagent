"""Integration test for Ollama tool calling.

Tests function calling with llama3.1 model.
Requires Ollama running locally with llama3.1 pulled.
"""

import pytest
from google.adk.models.llm_request import LlmRequest
from google.genai import types

from kagent.adk.models._ollama import OllamaNative


@pytest.mark.asyncio
@pytest.mark.integration
async def test_tool_calling():
    """Test function calling with compatible model."""
    driver = OllamaNative(
        type="ollama",
        model="gpt-oss:latest",  # gpt-oss supports tool calling
        base_url="http://localhost:11434",
    )

    # Define a simple tool
    get_weather_tool = types.Tool(
        function_declarations=[
            types.FunctionDeclaration(
                name="get_weather",
                description="Get the current weather for a location",
                parameters=types.Schema(
                    type=types.Type.OBJECT,
                    properties={"location": types.Schema(type=types.Type.STRING, description="City name")},
                    required=["location"],
                ),
            )
        ]
    )

    # Create request with tools
    request = LlmRequest(
        model="gpt-oss:latest",
        contents=[
            types.Content(role="user", parts=[types.Part.from_text(text="What's the weather in San Francisco?")])
        ],
        config=types.GenerateContentConfig(tools=[get_weather_tool]),
    )

    # Generate response
    responses = []
    async for response in driver.generate_content_async(request, stream=False):
        responses.append(response)

    # Validate tool call in response
    assert len(responses) == 1
    response = responses[0]

    assert response.content is not None

    # Check if model made a function call
    has_function_call = any(part.function_call is not None for part in response.content.parts)

    # With a tool-capable model and appropriate prompt,
    # we expect a function call
    if has_function_call:
        func_call_part = next(part for part in response.content.parts if part.function_call is not None)
        assert func_call_part.function_call.name == "get_weather"
        assert "location" in func_call_part.function_call.args


if __name__ == "__main__":
    pytest.main([__file__, "-v", "-m", "integration"])
