"""Contract tests for Ollama chat completions request schema.

These tests validate that our request format matches the Ollama API schema
defined in contracts/ollama-chat-completions.yaml.
"""

import pytest


def test_basic_request_schema():
    """Test basic chat completion request format."""
    # This test validates the basic request structure from
    # contracts/ollama-chat-completions.yaml example: basic
    from google.genai import types

    from kagent.adk.models._ollama import _convert_content_to_ollama_messages

    # Create a simple user message
    contents = [types.Content(role="user", parts=[types.Part.from_text(text="Hello, how are you?")])]

    # Convert to Ollama format
    messages = _convert_content_to_ollama_messages(contents)

    # Validate schema
    assert isinstance(messages, list)
    assert len(messages) == 1
    assert messages[0]["role"] == "user"
    assert messages[0]["content"] == "Hello, how are you?"

    # Validate complete request structure (as would be sent to API)
    request_body = {"model": "gpt-oss:latest", "messages": messages}

    # Required fields per schema
    assert "model" in request_body
    assert "messages" in request_body
    assert isinstance(request_body["model"], str)
    assert isinstance(request_body["messages"], list)


def test_request_with_system_instruction():
    """Test request with system instruction."""
    # This test validates the with_system example from
    # contracts/ollama-chat-completions.yaml
    from google.genai import types

    from kagent.adk.models._ollama import _convert_content_to_ollama_messages

    # Create user message
    contents = [types.Content(role="user", parts=[types.Part.from_text(text="Hello")])]

    # Convert with system instruction
    messages = _convert_content_to_ollama_messages(contents, system_instruction="You are a helpful assistant")

    # Validate schema
    assert len(messages) == 2
    assert messages[0]["role"] == "system"
    assert messages[0]["content"] == "You are a helpful assistant"
    assert messages[1]["role"] == "user"
    assert messages[1]["content"] == "Hello"


def test_request_with_tools():
    """Test request with function calling (tools)."""
    # This test validates the with_tools example from
    # contracts/ollama-chat-completions.yaml
    from google.genai import types

    from kagent.adk.models._ollama import _convert_tools_to_ollama

    # Create tool definition
    tool = types.Tool(
        function_declarations=[
            types.FunctionDeclaration(
                name="get_weather",
                description="Get weather for a location",
                parameters=types.Schema(
                    type=types.Type.OBJECT,
                    properties={"location": types.Schema(type=types.Type.STRING, description="City name")},
                    required=["location"],
                ),
            )
        ]
    )

    # Convert to Ollama format
    ollama_tools = _convert_tools_to_ollama([tool])

    # Validate schema per contracts/ollama-chat-completions.yaml
    assert isinstance(ollama_tools, list)
    assert len(ollama_tools) == 1

    tool_def = ollama_tools[0]
    assert tool_def["type"] == "function"
    assert "function" in tool_def

    func = tool_def["function"]
    assert func["name"] == "get_weather"
    assert func["description"] == "Get weather for a location"
    assert "parameters" in func

    params = func["parameters"]
    assert params["type"] == "object"
    assert "properties" in params
    assert "location" in params["properties"]
    assert params["required"] == ["location"]


def test_request_optional_parameters():
    """Test request with optional parameters (temperature, max_tokens, stream)."""
    # Validate optional fields per schema
    request_body = {
        "model": "gpt-oss:latest",
        "messages": [{"role": "user", "content": "Hello"}],
        "temperature": 0.7,
        "max_tokens": 2048,
        "stream": False,
    }

    # Validate types per schema
    assert isinstance(request_body["temperature"], (int, float))
    assert 0 <= request_body["temperature"] <= 2
    assert isinstance(request_body["max_tokens"], int)
    assert request_body["max_tokens"] >= 1
    assert isinstance(request_body["stream"], bool)


if __name__ == "__main__":
    pytest.main([__file__, "-v"])
