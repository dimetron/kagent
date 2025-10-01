"""Unit tests for _convert_ollama_response_to_llm_response converter function."""

import pytest
from google.genai import types

from kagent.adk.models._ollama import _convert_ollama_response_to_llm_response


def test_simple_text_response():
    """Test converting simple text response."""
    ollama_response = {
        "id": "chatcmpl-123",
        "object": "chat.completion",
        "created": 1677652288,
        "model": "gpt-oss:latest",
        "choices": [
            {
                "index": 0,
                "message": {"role": "assistant", "content": "Hello, how can I help you?"},
                "finish_reason": "stop",
            }
        ],
        "usage": {"prompt_tokens": 10, "completion_tokens": 12, "total_tokens": 22},
    }

    llm_response = _convert_ollama_response_to_llm_response(ollama_response)

    assert llm_response.content is not None
    assert llm_response.content.role == "model"
    assert len(llm_response.content.parts) == 1
    assert llm_response.content.parts[0].text == "Hello, how can I help you?"
    assert llm_response.finish_reason == types.FinishReason.STOP


def test_response_with_usage_metadata():
    """Test converting response with usage metadata."""
    ollama_response = {
        "id": "chatcmpl-123",
        "object": "chat.completion",
        "created": 1677652288,
        "model": "gpt-oss:latest",
        "choices": [{"index": 0, "message": {"role": "assistant", "content": "Test"}, "finish_reason": "stop"}],
        "usage": {"prompt_tokens": 5, "completion_tokens": 3, "total_tokens": 8},
    }

    llm_response = _convert_ollama_response_to_llm_response(ollama_response)

    assert llm_response.usage_metadata is not None
    assert llm_response.usage_metadata.prompt_token_count == 5
    assert llm_response.usage_metadata.candidates_token_count == 3
    assert llm_response.usage_metadata.total_token_count == 8


def test_response_with_tool_calls():
    """Test converting response with tool calls."""
    ollama_response = {
        "id": "chatcmpl-456",
        "object": "chat.completion",
        "created": 1677652288,
        "model": "gpt-oss:latest",
        "choices": [
            {
                "index": 0,
                "message": {
                    "role": "assistant",
                    "content": None,
                    "tool_calls": [
                        {
                            "id": "call_1",
                            "type": "function",
                            "function": {"name": "get_weather", "arguments": '{"location": "San Francisco"}'},
                        }
                    ],
                },
                "finish_reason": "tool_calls",
            }
        ],
    }

    llm_response = _convert_ollama_response_to_llm_response(ollama_response)

    assert llm_response.content is not None
    assert len(llm_response.content.parts) == 1

    part = llm_response.content.parts[0]
    assert part.function_call is not None
    assert part.function_call.name == "get_weather"
    assert part.function_call.args == {"location": "San Francisco"}
    assert part.function_call.id == "call_1"


def test_response_finish_reasons():
    """Test mapping of finish reasons."""
    test_cases = [
        ("stop", types.FinishReason.STOP),
        ("length", types.FinishReason.MAX_TOKENS),
        ("content_filter", types.FinishReason.SAFETY),
        ("tool_calls", types.FinishReason.STOP),
    ]

    for ollama_reason, expected_adk_reason in test_cases:
        ollama_response = {
            "id": "chatcmpl-123",
            "object": "chat.completion",
            "created": 1677652288,
            "model": "gpt-oss:latest",
            "choices": [
                {"index": 0, "message": {"role": "assistant", "content": "Test"}, "finish_reason": ollama_reason}
            ],
        }

        llm_response = _convert_ollama_response_to_llm_response(ollama_response)
        assert llm_response.finish_reason == expected_adk_reason


def test_response_without_usage():
    """Test converting response without usage metadata."""
    ollama_response = {
        "id": "chatcmpl-123",
        "object": "chat.completion",
        "created": 1677652288,
        "model": "gpt-oss:latest",
        "choices": [{"index": 0, "message": {"role": "assistant", "content": "Test"}, "finish_reason": "stop"}],
    }

    llm_response = _convert_ollama_response_to_llm_response(ollama_response)

    assert llm_response.content is not None
    # usage_metadata might be None or have zero values


def test_response_with_multiple_tool_calls():
    """Test converting response with multiple tool calls."""
    ollama_response = {
        "id": "chatcmpl-789",
        "object": "chat.completion",
        "created": 1677652288,
        "model": "gpt-oss:latest",
        "choices": [
            {
                "index": 0,
                "message": {
                    "role": "assistant",
                    "content": None,
                    "tool_calls": [
                        {
                            "id": "call_1",
                            "type": "function",
                            "function": {"name": "get_weather", "arguments": '{"location": "SF"}'},
                        },
                        {
                            "id": "call_2",
                            "type": "function",
                            "function": {"name": "get_time", "arguments": '{"timezone": "PST"}'},
                        },
                    ],
                },
                "finish_reason": "tool_calls",
            }
        ],
    }

    llm_response = _convert_ollama_response_to_llm_response(ollama_response)

    assert llm_response.content is not None
    assert len(llm_response.content.parts) == 2

    # Check both function calls
    assert llm_response.content.parts[0].function_call.name == "get_weather"
    assert llm_response.content.parts[1].function_call.name == "get_time"


if __name__ == "__main__":
    pytest.main([__file__, "-v"])
