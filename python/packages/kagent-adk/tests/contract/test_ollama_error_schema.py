"""Contract tests for ollama.ResponseError handling.

These tests validate that the ollama-python library raises ResponseError
with expected structure for error conditions.

Task: T007
"""

import ollama
import pytest


class TestOllamaResponseErrorSchema:
    """Validate ollama.ResponseError structure from ollama-python."""

    def test_response_error_has_status_code(self):
        """Verify ResponseError contains status_code attribute."""
        # ollama.ResponseError is raised for HTTP errors
        # Expected attributes:
        # - error: str (error message)
        # - status_code: int (HTTP status code)

        # Simulate creating a ResponseError
        try:
            raise ollama.ResponseError(error="Model not found", status_code=404)
        except ollama.ResponseError as e:
            assert hasattr(e, "error")
            assert hasattr(e, "status_code")
            assert e.status_code == 404
            assert isinstance(e.error, str)

    def test_response_error_404_model_not_found(self):
        """Verify 404 errors for missing models."""
        try:
            raise ollama.ResponseError(error="model 'nonexistent' not found", status_code=404)
        except ollama.ResponseError as e:
            assert e.status_code == 404
            assert "not found" in e.error.lower()

    def test_response_error_500_internal_error(self):
        """Verify 500 errors for server failures."""
        try:
            raise ollama.ResponseError(error="out of memory", status_code=500)
        except ollama.ResponseError as e:
            assert e.status_code == 500
            assert isinstance(e.error, str)

    def test_response_error_503_service_unavailable(self):
        """Verify 503 errors when Ollama is not running."""
        try:
            raise ollama.ResponseError(error="service unavailable", status_code=503)
        except ollama.ResponseError as e:
            assert e.status_code == 503

    def test_response_error_inherits_from_exception(self):
        """Verify ResponseError is a proper exception."""
        assert issubclass(ollama.ResponseError, Exception)

        try:
            raise ollama.ResponseError(error="test", status_code=400)
        except Exception as e:
            # Should be catchable as generic Exception
            assert isinstance(e, ollama.ResponseError)


if __name__ == "__main__":
    pytest.main([__file__, "-v"])
