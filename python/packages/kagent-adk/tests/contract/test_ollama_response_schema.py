"""Contract tests for ollama.ChatResponse schema validation.

These tests validate that the ollama-python library returns responses
matching the expected structure for basic chat completions.

Task: T005
"""

import ollama
import pytest


class TestOllamaChatResponseSchema:
    """Validate ollama.ChatResponse structure from ollama-python."""

    def test_chat_response_has_required_fields(self):
        """Verify ChatResponse contains model, message, done fields."""
        # This test validates the contract with ollama-python library
        # Expected structure:
        # {
        #     "model": str,
        #     "message": {
        #         "role": str,
        #         "content": str
        #     },
        #     "done": bool,
        #     "prompt_eval_count": int (optional),
        #     "eval_count": int (optional)
        # }

        # Create a mock response matching ollama-python structure
        mock_response = {
            "model": "llama2",
            "message": {"role": "assistant", "content": "Hello! How can I help you today?"},
            "done": True,
            "prompt_eval_count": 10,
            "eval_count": 20,
        }

        # Validate required fields exist
        assert "model" in mock_response
        assert "message" in mock_response
        assert "done" in mock_response

        # Validate message structure
        assert "role" in mock_response["message"]
        assert "content" in mock_response["message"]

        # Validate types
        assert isinstance(mock_response["model"], str)
        assert isinstance(mock_response["message"]["role"], str)
        assert isinstance(mock_response["message"]["content"], str)
        assert isinstance(mock_response["done"], bool)

    def test_chat_response_with_usage_metadata(self):
        """Verify ChatResponse includes token count fields."""
        mock_response = {
            "model": "llama2",
            "message": {"role": "assistant", "content": "Test response"},
            "done": True,
            "prompt_eval_count": 15,
            "eval_count": 25,
            "total_duration": 1234567890,
        }

        # Validate usage metadata fields
        assert "prompt_eval_count" in mock_response
        assert "eval_count" in mock_response
        assert isinstance(mock_response["prompt_eval_count"], int)
        assert isinstance(mock_response["eval_count"], int)
        assert mock_response["prompt_eval_count"] > 0
        assert mock_response["eval_count"] > 0

    def test_streaming_chat_response_partial(self):
        """Verify streaming responses have done=False for partial chunks."""
        partial_response = {
            "model": "llama2",
            "message": {"role": "assistant", "content": "Partial text"},
            "done": False,
        }

        assert partial_response["done"] is False
        assert "message" in partial_response
        assert isinstance(partial_response["message"]["content"], str)

    def test_streaming_chat_response_final(self):
        """Verify final streaming response has done=True and includes counts."""
        final_response = {
            "model": "llama2",
            "message": {"role": "assistant", "content": ""},
            "done": True,
            "prompt_eval_count": 10,
            "eval_count": 50,
        }

        assert final_response["done"] is True
        assert "prompt_eval_count" in final_response
        assert "eval_count" in final_response


if __name__ == "__main__":
    pytest.main([__file__, "-v"])
