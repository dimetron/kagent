"""Unit tests for _convert_tools_to_ollama converter function."""

import pytest
from google.genai import types

from kagent.adk.models._ollama import _convert_tools_to_ollama


def test_simple_function_no_parameters():
    """Test converting function with no parameters."""
    tool = types.Tool(
        function_declarations=[types.FunctionDeclaration(name="get_current_time", description="Gets the current time")]
    )

    ollama_tools = _convert_tools_to_ollama([tool])

    assert len(ollama_tools) == 1
    assert ollama_tools[0]["type"] == "function"
    assert ollama_tools[0]["function"]["name"] == "get_current_time"
    assert ollama_tools[0]["function"]["description"] == "Gets the current time"
    assert ollama_tools[0]["function"]["parameters"]["type"] == "object"
    assert ollama_tools[0]["function"]["parameters"]["properties"] == {}
    assert ollama_tools[0]["function"]["parameters"]["required"] == []


def test_function_with_parameters():
    """Test converting function with parameters."""
    tool = types.Tool(
        function_declarations=[
            types.FunctionDeclaration(
                name="get_weather",
                description="Get weather for a location",
                parameters=types.Schema(
                    type=types.Type.OBJECT,
                    properties={
                        "location": types.Schema(type=types.Type.STRING, description="City name"),
                        "units": types.Schema(type=types.Type.STRING, description="Temperature units"),
                    },
                    required=["location"],
                ),
            )
        ]
    )

    ollama_tools = _convert_tools_to_ollama([tool])

    assert len(ollama_tools) == 1
    func = ollama_tools[0]["function"]

    assert func["name"] == "get_weather"
    assert "parameters" in func
    assert func["parameters"]["type"] == "object"
    assert "location" in func["parameters"]["properties"]
    assert "units" in func["parameters"]["properties"]
    assert func["parameters"]["required"] == ["location"]


def test_function_with_nested_properties():
    """Test converting function with nested object properties."""
    tool = types.Tool(
        function_declarations=[
            types.FunctionDeclaration(
                name="create_user",
                description="Create a user profile",
                parameters=types.Schema(
                    type=types.Type.OBJECT,
                    properties={
                        "name": types.Schema(type=types.Type.STRING),
                        "address": types.Schema(
                            type=types.Type.OBJECT,
                            properties={
                                "street": types.Schema(type=types.Type.STRING),
                                "city": types.Schema(type=types.Type.STRING),
                                "zip": types.Schema(type=types.Type.STRING),
                            },
                        ),
                    },
                    required=["name"],
                ),
            )
        ]
    )

    ollama_tools = _convert_tools_to_ollama([tool])

    assert len(ollama_tools) == 1
    params = ollama_tools[0]["function"]["parameters"]

    assert "name" in params["properties"]
    assert "address" in params["properties"]
    assert params["properties"]["address"]["type"] == "object"
    assert "properties" in params["properties"]["address"]


def test_function_with_array_parameter():
    """Test converting function with array parameter."""
    tool = types.Tool(
        function_declarations=[
            types.FunctionDeclaration(
                name="send_emails",
                description="Send emails to multiple recipients",
                parameters=types.Schema(
                    type=types.Type.OBJECT,
                    properties={
                        "recipients": types.Schema(
                            type=types.Type.ARRAY,
                            description="List of email addresses",
                            items=types.Schema(type=types.Type.STRING),
                        ),
                        "subject": types.Schema(type=types.Type.STRING),
                    },
                    required=["recipients", "subject"],
                ),
            )
        ]
    )

    ollama_tools = _convert_tools_to_ollama([tool])

    assert len(ollama_tools) == 1
    params = ollama_tools[0]["function"]["parameters"]

    assert "recipients" in params["properties"]
    assert params["properties"]["recipients"]["type"] == "array"
    assert "items" in params["properties"]["recipients"]


def test_multiple_functions():
    """Test converting multiple function declarations."""
    tool = types.Tool(
        function_declarations=[
            types.FunctionDeclaration(name="function_one", description="First function"),
            types.FunctionDeclaration(name="function_two", description="Second function"),
        ]
    )

    ollama_tools = _convert_tools_to_ollama([tool])

    assert len(ollama_tools) == 2
    assert ollama_tools[0]["function"]["name"] == "function_one"
    assert ollama_tools[1]["function"]["name"] == "function_two"


if __name__ == "__main__":
    pytest.main([__file__, "-v"])

