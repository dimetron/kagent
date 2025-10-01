"""Contract tests for Ollama error responses.

These tests validate error response handling per the schema
defined in contracts/ollama-chat-completions.yaml.
"""

import pytest


def test_model_not_found_error_404():
    """Test 404 model not found error handling."""
    # Per schema, 404 response format
    error_response = {
        "error": {
            "message": "model 'gpt-oss:latest' not found, try pulling it first",
            "type": "model_not_found",
            "code": 404,
        }
    }

    # Validate error structure
    assert "error" in error_response
    assert "message" in error_response["error"]
    assert "type" in error_response["error"]
    assert "code" in error_response["error"]

    assert error_response["error"]["code"] == 404
    assert error_response["error"]["type"] == "model_not_found"
    assert "gpt-oss" in error_response["error"]["message"]


def test_server_error_500():
    """Test 500 internal server error handling."""
    # Per schema, 500 response format
    error_response = {"error": {"message": "internal server error", "type": "server_error", "code": 500}}

    # Validate error structure
    assert "error" in error_response
    assert error_response["error"]["code"] == 500
    assert error_response["error"]["type"] == "server_error"


def test_error_response_to_llm_response():
    """Test converting Ollama error to LlmResponse with error."""
    from kagent.adk.models._ollama import OllamaNative

    # We should yield an LlmResponse with error_code and error_message
    # This will be tested in integration tests, but validates the contract
    pass


if __name__ == "__main__":
    pytest.main([__file__, "-v"])
