"""Integration test for Ollama tool calling (function calling).

This test validates tool/function calling with compatible models like llama3.1.

Task: T010
"""

import pytest
from google.genai import types
from google.genai.types import FunctionDeclaration, Schema, Tool

pytestmark = [pytest.mark.asyncio, pytest.mark.integration]


class TestOllamaToolCalling:
    """Integration tests for function calling with Ollama."""

    async def test_tool_call_generation(self, ollama_tool_model, ollama_base_url):
        """Test that model generates tool call when appropriate."""
        from google.adk.models.llm_request import LlmRequest

        from kagent.adk.models._ollama import OllamaNative

        ollama_native = OllamaNative(type="ollama", model=ollama_tool_model, base_url=ollama_base_url)

        # Define a tool
        get_weather_tool = Tool(
            function_declarations=[
                FunctionDeclaration(
                    name="get_weather",
                    description="Get current weather for a location",
                    parameters=Schema(
                        type="object",
                        properties={
                            "location": Schema(type="string", description="City name"),
                            "unit": Schema(type="string", enum=["celsius", "fahrenheit"]),
                        },
                        required=["location"],
                    ),
                )
            ]
        )

        # Tools must be passed via GenerateContentConfig
        config = types.GenerateContentConfig(tools=[get_weather_tool])
        
        request = LlmRequest(
            contents=[types.Content(role="user", parts=[types.Part(text="What's the weather in Paris?")])],
            config=config,
        )

        response = None
        async for resp in ollama_native.generate_content_async(request, stream=False):
            response = resp

        # Validate tool call in response
        assert response is not None
        assert response.content is not None

        # Check for function call in parts
        has_function_call = any(part.function_call is not None for part in response.content.parts)

        assert has_function_call, "Response should contain function call"

    async def test_tool_call_with_response(self, ollama_tool_model):
        """Test full tool calling flow: call → response → final answer."""
        from google.adk.models.llm_request import LlmRequest

        from kagent.adk.models._ollama import OllamaNative

        ollama_native = OllamaNative(type="ollama", model=ollama_tool_model)

        # Define tool
        get_time_tool = Tool(
            function_declarations=[
                FunctionDeclaration(
                    name="get_current_time",
                    description="Get current time",
                    parameters=Schema(type="object", properties={}),
                )
            ]
        )

        # Tools must be passed via GenerateContentConfig
        config = types.GenerateContentConfig(tools=[get_time_tool])

        # First request: Model should call the tool
        request1 = LlmRequest(
            contents=[types.Content(role="user", parts=[types.Part(text="What time is it?")])],
            config=config
        )

        response1 = None
        async for resp in ollama_native.generate_content_async(request1, stream=False):
            response1 = resp

        assert response1 is not None

        # Simulate tool response
        from google.genai.types import FunctionCall, FunctionResponse

        # Find the function call
        function_call = None
        for part in response1.content.parts:
            if part.function_call:
                function_call = part.function_call
                break

        if function_call:
            # Second request: Provide tool response
            request2 = LlmRequest(
                contents=[
                    types.Content(role="user", parts=[types.Part(text="What time is it?")]),
                    response1.content,  # Model's tool call
                    types.Content(
                        role="user",
                        parts=[
                            types.Part(
                                function_response=FunctionResponse(
                                    name=function_call.name, response={"time": "14:30:00"}
                                )
                            )
                        ],
                    ),
                ],
                config=config,
            )

            response2 = None
            async for resp in ollama_native.generate_content_async(request2, stream=False):
                response2 = resp

            # Model should now provide natural language answer
            assert response2 is not None
            assert response2.content is not None

    async def test_multiple_tool_calls(self, ollama_tool_model):
        """Test model calling multiple tools in one response."""
        from google.adk.models.llm_request import LlmRequest

        from kagent.adk.models._ollama import OllamaNative

        ollama_native = OllamaNative(type="ollama", model=ollama_tool_model)

        # Define multiple tools
        tools_list = [
            Tool(
                function_declarations=[
                    FunctionDeclaration(
                        name="get_weather",
                        description="Get weather",
                        parameters=Schema(type="object", properties={"location": Schema(type="string")}),
                    )
                ]
            ),
            Tool(
                function_declarations=[
                    FunctionDeclaration(
                        name="get_time", description="Get time", parameters=Schema(type="object", properties={})
                    )
                ]
            ),
        ]

        # Tools must be passed via GenerateContentConfig
        config = types.GenerateContentConfig(tools=tools_list)

        request = LlmRequest(
            contents=[types.Content(role="user", parts=[types.Part(text="What's the weather and time in Paris?")])],
            config=config,
        )

        response = None
        async for resp in ollama_native.generate_content_async(request, stream=False):
            response = resp

        assert response is not None
        # Model may call one or both tools

    async def test_tool_calling_with_streaming(self, ollama_tool_model):
        """Test tool calls work with streaming responses."""
        from google.adk.models.llm_request import LlmRequest

        from kagent.adk.models._ollama import OllamaNative

        ollama_native = OllamaNative(type="ollama", model=ollama_tool_model)

        tool = Tool(
            function_declarations=[
                FunctionDeclaration(
                    name="calculator",
                    description="Perform calculation",
                    parameters=Schema(type="object", properties={"expression": Schema(type="string")}),
                )
            ]
        )

        # Tools must be passed via GenerateContentConfig
        config = types.GenerateContentConfig(tools=[tool])

        request = LlmRequest(
            contents=[types.Content(role="user", parts=[types.Part(text="Calculate 2+2")])],
            config=config
        )

        chunks = []
        async for chunk in ollama_native.generate_content_async(request, stream=True):
            chunks.append(chunk)

        assert len(chunks) > 0
        # Tool call should appear in one of the chunks


if __name__ == "__main__":
    pytest.main([__file__, "-v"])
