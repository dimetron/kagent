"""Unit tests for OllamaNative client initialization and configuration."""

import httpx
import pytest

from kagent.adk.models._ollama import OllamaNative


def test_client_initialization():
    """Test OllamaNative client initialization."""
    driver = OllamaNative(type="ollama", model="gpt-oss:latest", base_url="http://localhost:11434")

    assert driver.model == "gpt-oss:latest"
    assert driver.base_url == "http://localhost:11434"
    assert driver.type == "ollama"


def test_client_property_caching():
    """Test that _client property is cached."""
    driver = OllamaNative(type="ollama", model="gpt-oss:latest")

    # Access client twice
    client1 = driver._client
    client2 = driver._client

    # Should be the same instance (cached)
    assert client1 is client2
    assert isinstance(client1, httpx.AsyncClient)


def test_default_base_url():
    """Test default base_url is set correctly."""
    driver = OllamaNative(type="ollama", model="gpt-oss:latest")

    assert driver.base_url == "http://localhost:11434"


def test_custom_base_url():
    """Test custom base_url is used."""
    driver = OllamaNative(type="ollama", model="gpt-oss:latest", base_url="https://ollama.example.com")

    assert driver.base_url == "https://ollama.example.com"


def test_custom_timeout():
    """Test custom timeout configuration."""
    driver = OllamaNative(type="ollama", model="gpt-oss:latest", timeout=30.0)

    assert driver.timeout == 30.0


def test_custom_headers():
    """Test custom headers configuration."""
    headers = {"Authorization": "Bearer token123"}
    driver = OllamaNative(type="ollama", model="gpt-oss:latest", headers=headers)

    assert driver.headers == headers


def test_supported_models():
    """Test supported_models returns wildcard pattern."""
    patterns = OllamaNative.supported_models()

    assert isinstance(patterns, list)
    assert len(patterns) == 1
    assert patterns[0] == r".*"  # Matches all models


def test_temperature_configuration():
    """Test temperature configuration."""
    driver = OllamaNative(type="ollama", model="gpt-oss:latest", temperature=0.7)

    assert driver.temperature == 0.7


def test_max_tokens_configuration():
    """Test max_tokens configuration."""
    driver = OllamaNative(type="ollama", model="gpt-oss:latest", max_tokens=2048)

    assert driver.max_tokens == 2048


if __name__ == "__main__":
    pytest.main([__file__, "-v"])
