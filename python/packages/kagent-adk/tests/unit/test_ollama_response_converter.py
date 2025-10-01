"""Unit tests for _convert_ollama_response_to_llm_response converter function.

This tests the conversion from ollama.ChatResponse to ADK LlmResponse.

Task: T015
"""

import pytest
from google.adk.models.llm_response import LlmResponse


class TestConvertOllamaResponseToLlmResponse:
    """Unit tests for response conversion from Ollama format."""
    
    def test_basic_text_response(self):
        """Test converting basic text response."""
        from kagent.adk.models._ollama import _convert_ollama_response_to_llm_response
        
        ollama_response = {
            "model": "llama2",
            "message": {
                "role": "assistant",
                "content": "Hello! How can I help you?"
            },
            "done": True,
            "prompt_eval_count": 10,
            "eval_count": 8
        }
        
        llm_response = _convert_ollama_response_to_llm_response(ollama_response)
        
        assert isinstance(llm_response, LlmResponse)
        assert llm_response.content is not None
        assert len(llm_response.content.parts) > 0
        assert llm_response.content.parts[0].text == "Hello! How can I help you?"
    
    def test_response_with_usage_metadata(self):
        """Test that usage metadata is extracted correctly."""
        from kagent.adk.models._ollama import _convert_ollama_response_to_llm_response
        
        ollama_response = {
            "model": "llama2",
            "message": {
                "role": "assistant",
                "content": "Test"
            },
            "done": True,
            "prompt_eval_count": 15,
            "eval_count": 25
        }
        
        llm_response = _convert_ollama_response_to_llm_response(ollama_response)
        
        assert llm_response.usage_metadata is not None
        assert llm_response.usage_metadata.prompt_token_count == 15
        assert llm_response.usage_metadata.candidates_token_count == 25
        assert llm_response.usage_metadata.total_token_count == 40
    
    def test_streaming_partial_response(self):
        """Test converting partial streaming response."""
        from kagent.adk.models._ollama import _convert_ollama_response_to_llm_response
        
        ollama_response = {
            "model": "llama2",
            "message": {
                "role": "assistant",
                "content": "Hello"
            },
            "done": False
        }
        
        llm_response = _convert_ollama_response_to_llm_response(ollama_response)
        
        assert llm_response.partial is True
        assert llm_response.content.parts[0].text == "Hello"
    
    def test_streaming_final_response(self):
        """Test converting final streaming response with done=True."""
        from kagent.adk.models._ollama import _convert_ollama_response_to_llm_response
        
        ollama_response = {
            "model": "llama2",
            "message": {
                "role": "assistant",
                "content": ""
            },
            "done": True,
            "prompt_eval_count": 10,
            "eval_count": 50
        }
        
        llm_response = _convert_ollama_response_to_llm_response(ollama_response)
        
        assert llm_response.partial is False
        assert llm_response.usage_metadata is not None
    
    def test_response_with_tool_calls(self):
        """Test converting response with tool calls."""
        from kagent.adk.models._ollama import _convert_ollama_response_to_llm_response
        
        ollama_response = {
            "model": "llama3.1",
            "message": {
                "role": "assistant",
                "content": "",
                "tool_calls": [
                    {
                        "id": "call_123",
                        "type": "function",
                        "function": {
                            "name": "get_weather",
                            "arguments": '{"location": "Paris"}'
                        }
                    }
                ]
            },
            "done": True
        }
        
        llm_response = _convert_ollama_response_to_llm_response(ollama_response)
        
        assert llm_response.content is not None
        # Should have function call in parts
        has_function_call = any(
            part.function_call is not None 
            for part in llm_response.content.parts
        )
        assert has_function_call
    
    def test_response_with_multiple_tool_calls(self):
        """Test converting response with multiple tool calls."""
        from kagent.adk.models._ollama import _convert_ollama_response_to_llm_response
        
        ollama_response = {
            "model": "llama3.1",
            "message": {
                "role": "assistant",
                "content": "",
                "tool_calls": [
                    {
                        "id": "call_1",
                        "type": "function",
                        "function": {
                            "name": "get_weather",
                            "arguments": '{"location": "Paris"}'
                        }
                    },
                    {
                        "id": "call_2",
                        "type": "function",
                        "function": {
                            "name": "get_time",
                            "arguments": '{"timezone": "UTC"}'
                        }
                    }
                ]
            },
            "done": True
        }
        
        llm_response = _convert_ollama_response_to_llm_response(ollama_response)
        
        # Should have multiple function calls
        function_calls = [
            part.function_call 
            for part in llm_response.content.parts 
            if part.function_call
        ]
        assert len(function_calls) == 2
    
    def test_response_without_usage_metadata(self):
        """Test handling response without token counts."""
        from kagent.adk.models._ollama import _convert_ollama_response_to_llm_response
        
        ollama_response = {
            "model": "llama2",
            "message": {
                "role": "assistant",
                "content": "Test"
            },
            "done": True
        }
        
        llm_response = _convert_ollama_response_to_llm_response(ollama_response)
        
        # Should handle missing usage metadata gracefully
        assert llm_response is not None
    
    def test_empty_content_response(self):
        """Test converting response with empty content."""
        from kagent.adk.models._ollama import _convert_ollama_response_to_llm_response
        
        ollama_response = {
            "model": "llama2",
            "message": {
                "role": "assistant",
                "content": ""
            },
            "done": True
        }
        
        llm_response = _convert_ollama_response_to_llm_response(ollama_response)
        
        assert llm_response is not None
        assert llm_response.content is not None
    
    def test_finish_reason_mapping(self):
        """Test that done flag is mapped to appropriate finish reason."""
        from kagent.adk.models._ollama import _convert_ollama_response_to_llm_response
        
        # Done response
        done_response = {
            "model": "llama2",
            "message": {
                "role": "assistant",
                "content": "Complete"
            },
            "done": True
        }
        
        llm_response = _convert_ollama_response_to_llm_response(done_response)
        
        # Should indicate completion
        # (exact mapping depends on implementation)
        assert llm_response is not None
        assert llm_response.partial is False
    
    def test_tool_call_argument_parsing(self):
        """Test that tool call arguments are parsed from JSON string."""
        from kagent.adk.models._ollama import _convert_ollama_response_to_llm_response
        
        ollama_response = {
            "model": "llama3.1",
            "message": {
                "role": "assistant",
                "content": "",
                "tool_calls": [
                    {
                        "id": "call_456",
                        "type": "function",
                        "function": {
                            "name": "complex_func",
                            "arguments": '{"param1": "value1", "param2": 123, "nested": {"key": "val"}}'
                        }
                    }
                ]
            },
            "done": True
        }
        
        llm_response = _convert_ollama_response_to_llm_response(ollama_response)
        
        function_call = None
        for part in llm_response.content.parts:
            if part.function_call:
                function_call = part.function_call
                break
        
        assert function_call is not None
        assert function_call.name == "complex_func"
        # Arguments should be parsed from JSON
        assert function_call.args is not None


if __name__ == "__main__":
    pytest.main([__file__, "-v"])

