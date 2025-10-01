"""Pytest configuration for Ollama integration tests."""

import os

import pytest


@pytest.fixture
def ollama_model():
    """
    Returns the Ollama model name to use for testing.
    Can be overridden with OLLAMA_TEST_MODEL environment variable.
    Default: gpt-oss:latest
    """
    return os.getenv("OLLAMA_TEST_MODEL", "gpt-oss:latest")


@pytest.fixture
def ollama_tool_model():
    """
    Returns the Ollama model name to use for tool calling tests.
    Can be overridden with OLLAMA_TOOL_TEST_MODEL environment variable.
    Default: gpt-oss:latest (supports tool calling)
    """
    return os.getenv("OLLAMA_TOOL_TEST_MODEL", "gpt-oss:latest")


@pytest.fixture
def ollama_base_url():
    """
    Returns the Ollama server base URL.
    Can be overridden with OLLAMA_BASE_URL environment variable.
    Default: http://localhost:11434
    """
    return os.getenv("OLLAMA_BASE_URL", "http://localhost:11434")
