"""Unit tests for _convert_content_to_ollama_messages converter function.

This tests the conversion from ADK Content to ollama-python messages format.

Task: T013
"""

import pytest
from google.genai import types
from google.genai.types import FunctionCall, FunctionResponse


class TestConvertContentToOllamaMessages:
    """Unit tests for content conversion to Ollama format."""
    
    def test_simple_user_message(self):
        """Test converting single user text message."""
        from kagent.adk.models._ollama import _convert_content_to_ollama_messages
        
        contents = [
            types.Content(
                role="user",
                parts=[types.Part(text="Hello")]
            )
        ]
        
        messages = _convert_content_to_ollama_messages(contents)
        
        assert len(messages) == 1
        assert messages[0]["role"] == "user"
        assert messages[0]["content"] == "Hello"
    
    def test_system_instruction(self):
        """Test system instruction is converted to system message."""
        from kagent.adk.models._ollama import _convert_content_to_ollama_messages
        
        contents = [
            types.Content(
                role="user",
                parts=[types.Part(text="Hello")]
            )
        ]
        
        messages = _convert_content_to_ollama_messages(
            contents,
            system_instruction="You are a helpful assistant"
        )
        
        # System message should be first
        assert len(messages) == 2
        assert messages[0]["role"] == "system"
        assert messages[0]["content"] == "You are a helpful assistant"
        assert messages[1]["role"] == "user"
    
    def test_multi_turn_conversation(self):
        """Test multi-turn conversation with user and model messages."""
        from kagent.adk.models._ollama import _convert_content_to_ollama_messages
        
        contents = [
            types.Content(
                role="user",
                parts=[types.Part(text="Hello")]
            ),
            types.Content(
                role="model",
                parts=[types.Part(text="Hi there!")]
            ),
            types.Content(
                role="user",
                parts=[types.Part(text="How are you?")]
            )
        ]
        
        messages = _convert_content_to_ollama_messages(contents)
        
        assert len(messages) == 3
        assert messages[0]["role"] == "user"
        assert messages[1]["role"] == "assistant"  # model → assistant
        assert messages[2]["role"] == "user"
    
    def test_function_call_conversion(self):
        """Test converting function calls to tool_calls format."""
        from kagent.adk.models._ollama import _convert_content_to_ollama_messages
        
        contents = [
            types.Content(
                role="user",
                parts=[types.Part(text="What's the weather?")]
            ),
            types.Content(
                role="model",
                parts=[types.Part(
                    function_call=FunctionCall(
                        name="get_weather",
                        args={"location": "Paris"}
                    )
                )]
            )
        ]
        
        messages = _convert_content_to_ollama_messages(contents)
        
        assert len(messages) == 2
        assistant_msg = messages[1]
        assert assistant_msg["role"] == "assistant"
        assert "tool_calls" in assistant_msg
        assert len(assistant_msg["tool_calls"]) == 1
        
        tool_call = assistant_msg["tool_calls"][0]
        assert tool_call["type"] == "function"
        assert tool_call["function"]["name"] == "get_weather"
        assert "arguments" in tool_call["function"]
    
    def test_function_response_conversion(self):
        """Test converting function responses to tool message format."""
        from kagent.adk.models._ollama import _convert_content_to_ollama_messages
        
        contents = [
            types.Content(
                role="user",
                parts=[types.Part(text="What's the weather?")]
            ),
            types.Content(
                role="model",
                parts=[types.Part(
                    function_call=FunctionCall(
                        id="call_123",
                        name="get_weather",
                        args={"location": "Paris"}
                    )
                )]
            ),
            types.Content(
                role="user",
                parts=[types.Part(
                    function_response=FunctionResponse(
                        id="call_123",
                        name="get_weather",
                        response={"temperature": 20, "condition": "sunny"}
                    )
                )]
            )
        ]
        
        messages = _convert_content_to_ollama_messages(contents)
        
        # Should have user, assistant with tool_calls, and tool response
        assert len(messages) == 3
        
        tool_msg = messages[2]
        assert tool_msg["role"] == "tool"
        assert "tool_call_id" in tool_msg
        assert tool_msg["tool_call_id"] == "call_123"
        assert "content" in tool_msg
    
    def test_multimodal_content(self):
        """Test converting content with text and images."""
        from kagent.adk.models._ollama import _convert_content_to_ollama_messages
        
        contents = [
            types.Content(
                role="user",
                parts=[
                    types.Part(text="What's in this image?"),
                    types.Part(inline_data=types.Blob(
                        mime_type="image/jpeg",
                        data=b"fake_image_data"
                    ))
                ]
            )
        ]
        
        messages = _convert_content_to_ollama_messages(contents)
        
        assert len(messages) == 1
        # Multimodal handling depends on implementation
        # Should either concatenate or use structured format
        assert messages[0]["role"] == "user"
        assert "content" in messages[0]
    
    def test_multiple_function_calls(self):
        """Test content with multiple function calls."""
        from kagent.adk.models._ollama import _convert_content_to_ollama_messages
        
        contents = [
            types.Content(
                role="model",
                parts=[
                    types.Part(
                        function_call=FunctionCall(
                            id="call_1",
                            name="get_weather",
                            args={"location": "Paris"}
                        )
                    ),
                    types.Part(
                        function_call=FunctionCall(
                            id="call_2",
                            name="get_time",
                            args={"timezone": "UTC"}
                        )
                    )
                ]
            )
        ]
        
        messages = _convert_content_to_ollama_messages(contents)
        
        assert len(messages) == 1
        assert "tool_calls" in messages[0]
        assert len(messages[0]["tool_calls"]) == 2
    
    def test_empty_content_list(self):
        """Test handling empty content list."""
        from kagent.adk.models._ollama import _convert_content_to_ollama_messages
        
        contents = []
        messages = _convert_content_to_ollama_messages(contents)
        
        # Should return empty list or raise appropriate error
        assert isinstance(messages, list)
    
    def test_content_with_empty_parts(self):
        """Test handling content with no parts."""
        from kagent.adk.models._ollama import _convert_content_to_ollama_messages
        
        contents = [
            types.Content(
                role="user",
                parts=[]
            )
        ]
        
        messages = _convert_content_to_ollama_messages(contents)
        
        # Should handle gracefully
        assert isinstance(messages, list)


if __name__ == "__main__":
    pytest.main([__file__, "-v"])

