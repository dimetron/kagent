"""Unit tests for _convert_content_to_ollama_messages converter function."""

import pytest
from google.genai import types

from kagent.adk.models._ollama import _convert_content_to_ollama_messages


def test_simple_text_message():
    """Test converting simple text message."""
    contents = [types.Content(role="user", parts=[types.Part.from_text(text="Hello, world!")])]

    messages = _convert_content_to_ollama_messages(contents)

    assert len(messages) == 1
    assert messages[0]["role"] == "user"
    assert messages[0]["content"] == "Hello, world!"


def test_system_instruction():
    """Test adding system instruction."""
    contents = [types.Content(role="user", parts=[types.Part.from_text(text="Hello")])]

    messages = _convert_content_to_ollama_messages(contents, system_instruction="You are a helpful assistant")

    assert len(messages) == 2
    assert messages[0]["role"] == "system"
    assert messages[0]["content"] == "You are a helpful assistant"
    assert messages[1]["role"] == "user"


def test_multiple_messages():
    """Test converting conversation with multiple messages."""
    contents = [
        types.Content(role="user", parts=[types.Part.from_text(text="Hello")]),
        types.Content(role="model", parts=[types.Part.from_text(text="Hi there!")]),
        types.Content(role="user", parts=[types.Part.from_text(text="How are you?")]),
    ]

    messages = _convert_content_to_ollama_messages(contents)

    assert len(messages) == 3
    assert messages[0]["role"] == "user"
    assert messages[1]["role"] == "assistant"
    assert messages[2]["role"] == "user"


def test_function_call_conversion():
    """Test converting function calls to tool_calls format."""
    contents = [
        types.Content(
            role="model", parts=[types.Part.from_function_call(name="get_weather", args={"location": "San Francisco"})]
        )
    ]

    # Set the ID manually since from_function_call might not set it
    if contents[0].parts[0].function_call:
        contents[0].parts[0].function_call.id = "call_1"

    messages = _convert_content_to_ollama_messages(contents)

    # Should have assistant message with tool_calls
    assert len(messages) >= 1
    assert messages[0]["role"] == "assistant"
    assert "tool_calls" in messages[0]


def test_function_response_conversion():
    """Test converting function responses to tool messages."""
    contents = [
        types.Content(role="model", parts=[types.Part.from_function_call(name="get_weather", args={"location": "SF"})]),
        types.Content(
            role="user",
            parts=[types.Part.from_function_response(name="get_weather", response={"result": "Sunny, 72°F"})],
        ),
    ]

    # Set IDs
    if contents[0].parts[0].function_call:
        contents[0].parts[0].function_call.id = "call_1"
    if contents[1].parts[0].function_response:
        contents[1].parts[0].function_response.id = "call_1"

    messages = _convert_content_to_ollama_messages(contents)

    # Should have assistant message with tool_calls and tool response
    assert len(messages) >= 2


def test_multimodal_content():
    """Test converting multimodal content (text + images)."""
    # This is a placeholder - actual implementation depends on how
    # Ollama handles images
    contents = [
        types.Content(
            role="user",
            parts=[
                types.Part.from_text(text="What's in this image?"),
                # Image part would be added here in real scenario
            ],
        )
    ]

    messages = _convert_content_to_ollama_messages(contents)

    assert len(messages) == 1
    assert messages[0]["role"] == "user"


if __name__ == "__main__":
    pytest.main([__file__, "-v"])
