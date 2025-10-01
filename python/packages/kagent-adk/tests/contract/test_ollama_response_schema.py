"""Contract tests for Ollama chat completions response schema.

These tests validate that we correctly handle Ollama API responses
per the schema defined in contracts/ollama-chat-completions.yaml.
"""

import pytest


def test_basic_response_schema():
    """Test basic chat completion response format."""
    # This test validates the BasicResponse example from
    # contracts/ollama-chat-completions.yaml
    from kagent.adk.models._ollama import _convert_ollama_response_to_llm_response

    # Sample response per schema
    ollama_response = {
        "id": "chatcmpl-123",
        "object": "chat.completion",
        "created": 1677652288,
        "model": "gpt-oss:latest",
        "choices": [
            {
                "index": 0,
                "message": {"role": "assistant", "content": "I'm doing well, thank you for asking!"},
                "finish_reason": "stop",
            }
        ],
        "usage": {"prompt_tokens": 10, "completion_tokens": 12, "total_tokens": 22},
    }

    # Convert to ADK format
    llm_response = _convert_ollama_response_to_llm_response(ollama_response)

    # Validate conversion
    assert llm_response.content is not None
    assert llm_response.content.role == "model"
    assert len(llm_response.content.parts) == 1
    assert llm_response.content.parts[0].text == "I'm doing well, thank you for asking!"

    # Validate usage metadata
    assert llm_response.usage_metadata is not None
    assert llm_response.usage_metadata.prompt_token_count == 10
    assert llm_response.usage_metadata.candidates_token_count == 12
    assert llm_response.usage_metadata.total_token_count == 22


def test_tool_call_response_schema():
    """Test response with tool calls."""
    # This test validates the ToolCallResponse example from
    # contracts/ollama-chat-completions.yaml
    from kagent.adk.models._ollama import _convert_ollama_response_to_llm_response

    # Sample tool call response per schema
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
        "usage": {"prompt_tokens": 50, "completion_tokens": 20, "total_tokens": 70},
    }

    # Convert to ADK format
    llm_response = _convert_ollama_response_to_llm_response(ollama_response)

    # Validate conversion
    assert llm_response.content is not None
    assert llm_response.content.role == "model"

    # Should have a function call part
    assert len(llm_response.content.parts) == 1
    part = llm_response.content.parts[0]
    assert part.function_call is not None
    assert part.function_call.name == "get_weather"
    assert part.function_call.args == {"location": "San Francisco"}
    assert part.function_call.id == "call_1"


def test_response_required_fields():
    """Test that all required fields are present in response."""
    # Per schema, required fields are: id, object, created, model, choices
    ollama_response = {
        "id": "chatcmpl-123",
        "object": "chat.completion",
        "created": 1677652288,
        "model": "gpt-oss:latest",
        "choices": [{"index": 0, "message": {"role": "assistant", "content": "Hello"}, "finish_reason": "stop"}],
        # Note: usage is optional per schema
    }

    from kagent.adk.models._ollama import _convert_ollama_response_to_llm_response

    # Should not raise an error
    llm_response = _convert_ollama_response_to_llm_response(ollama_response)
    assert llm_response.content is not None


def test_response_finish_reasons():
    """Test all valid finish reasons per schema."""
    from google.genai import types

    from kagent.adk.models._ollama import _convert_ollama_response_to_llm_response

    # Valid finish reasons per schema: stop, length, tool_calls, content_filter
    finish_reason_map = {
        "stop": types.FinishReason.STOP,
        "length": types.FinishReason.MAX_TOKENS,
        "tool_calls": types.FinishReason.STOP,  # Tool calls is a type of stop
        "content_filter": types.FinishReason.SAFETY,
    }

    for ollama_reason, expected_adk_reason in finish_reason_map.items():
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


if __name__ == "__main__":
    pytest.main([__file__, "-v"])
