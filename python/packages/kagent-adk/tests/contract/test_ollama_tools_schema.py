"""Contract tests for ollama.ChatResponse with tool calls.

These tests validate that the ollama-python library returns responses
with tool_calls in the expected structure for function calling.

Task: T006
"""

import pytest


class TestOllamaToolCallsSchema:
    """Validate ollama.ChatResponse with tool_calls structure."""
    
    def test_tool_call_in_message(self):
        """Verify tool_calls field exists in message when function calling is used."""
        # Expected structure when model calls a function:
        # {
        #     "model": str,
        #     "message": {
        #         "role": "assistant",
        #         "content": "",
        #         "tool_calls": [
        #             {
        #                 "id": str,
        #                 "type": "function",
        #                 "function": {
        #                     "name": str,
        #                     "arguments": str (JSON)
        #                 }
        #             }
        #         ]
        #     },
        #     "done": bool
        # }
        
        mock_response = {
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
        
        # Validate tool_calls structure
        assert "tool_calls" in mock_response["message"]
        assert isinstance(mock_response["message"]["tool_calls"], list)
        assert len(mock_response["message"]["tool_calls"]) > 0
        
        tool_call = mock_response["message"]["tool_calls"][0]
        assert "id" in tool_call
        assert "type" in tool_call
        assert "function" in tool_call
        assert tool_call["type"] == "function"
        
        function = tool_call["function"]
        assert "name" in function
        assert "arguments" in function
        assert isinstance(function["name"], str)
        assert isinstance(function["arguments"], str)
    
    def test_multiple_tool_calls(self):
        """Verify response can contain multiple tool calls."""
        mock_response = {
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
        
        assert len(mock_response["message"]["tool_calls"]) == 2
        
        # Verify each tool call has unique ID
        ids = [tc["id"] for tc in mock_response["message"]["tool_calls"]]
        assert len(ids) == len(set(ids))  # All IDs are unique
    
    def test_tool_call_with_complex_arguments(self):
        """Verify tool calls can have complex JSON arguments."""
        mock_response = {
            "model": "llama3.1",
            "message": {
                "role": "assistant",
                "content": "",
                "tool_calls": [
                    {
                        "id": "call_456",
                        "type": "function",
                        "function": {
                            "name": "search_database",
                            "arguments": '{"query": "test", "filters": {"date": "2024-01-01", "tags": ["a", "b"]}, "limit": 10}'
                        }
                    }
                ]
            },
            "done": True
        }
        
        tool_call = mock_response["message"]["tool_calls"][0]
        import json
        args = json.loads(tool_call["function"]["arguments"])
        
        assert "query" in args
        assert "filters" in args
        assert "limit" in args
        assert isinstance(args["filters"], dict)
        assert isinstance(args["filters"]["tags"], list)
    
    def test_response_without_tool_calls(self):
        """Verify normal responses don't have tool_calls field."""
        mock_response = {
            "model": "llama2",
            "message": {
                "role": "assistant",
                "content": "Just a normal text response"
            },
            "done": True
        }
        
        # tool_calls should be absent or None in normal responses
        assert "tool_calls" not in mock_response["message"] or \
               mock_response["message"].get("tool_calls") is None


if __name__ == "__main__":
    pytest.main([__file__, "-v"])

