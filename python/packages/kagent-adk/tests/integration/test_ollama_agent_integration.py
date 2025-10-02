"""Integration test for Agent creation and execution with Ollama.

This test validates the full agent workflow using AgentConfig.to_agent()
with Ollama model type, ensuring end-to-end integration.

Task: T030
"""

import pytest
from google.genai import types

pytestmark = [pytest.mark.asyncio, pytest.mark.integration]


class TestOllamaAgentIntegration:
    """Integration tests for full agent creation and execution with Ollama."""

    async def test_agent_creation_from_config(self, ollama_model, ollama_base_url):
        """Test Agent creation from AgentConfig with Ollama model."""
        from kagent.adk.types import AgentConfig, Ollama

        # Create AgentConfig with Ollama model
        config = AgentConfig(
            model=Ollama(
                type="ollama",
                model=ollama_model,
                base_url=ollama_base_url,
                temperature=0.7,
            ),
            description="Test Ollama agent",
            instruction="You are a helpful assistant. Be concise.",
        )

        # Convert to Agent
        agent = config.to_agent(name="test_ollama_agent")

        # Validate agent creation
        assert agent is not None
        assert agent.name == "test_ollama_agent"
        assert agent.description == "Test Ollama agent"
        
        # Validate model is OllamaNative
        from kagent.adk.models._ollama import OllamaNative
        assert isinstance(agent.model, OllamaNative)
        assert agent.model.model == ollama_model
        assert agent.model.base_url == ollama_base_url
        assert agent.model.temperature == 0.7

    async def test_agent_simple_query(self, ollama_model, ollama_base_url):
        """Test agent execution with simple query."""
        from kagent.adk.types import AgentConfig, Ollama

        config = AgentConfig(
            model=Ollama(
                type="ollama",
                model=ollama_model,
                base_url=ollama_base_url,
            ),
            description="Test agent",
            instruction="You are helpful. Answer in one sentence.",
        )

        agent = config.to_agent(name="test_agent")

        # Execute query through agent
        from google.adk.models.llm_request import LlmRequest

        request = LlmRequest(
            contents=[types.Content(role="user", parts=[types.Part(text="What is 2+2?")])]
        )

        response = None
        async for resp in agent.model.generate_content_async(request, stream=False):
            response = resp

        # Validate response
        assert response is not None
        assert response.content is not None
        assert len(response.content.parts) > 0
        assert response.content.parts[0].text
        assert "4" in response.content.parts[0].text

    async def test_agent_with_system_instruction(self, ollama_model):
        """Test agent with system instruction from AgentConfig."""
        from kagent.adk.types import AgentConfig, Ollama

        config = AgentConfig(
            model=Ollama(type="ollama", model=ollama_model),
            description="Math agent",
            instruction="You are a math tutor. Only respond with numbers, no explanations.",
        )

        agent = config.to_agent(name="math_agent")

        # Execute with system instruction applied via LlmRequest
        from google.adk.models.llm_request import LlmRequest

        # Create request with system instruction
        request = LlmRequest(
            contents=[types.Content(role="user", parts=[types.Part(text="What is 5+3?")])],
            system_instruction=config.instruction
        )

        response = None
        async for resp in agent.model.generate_content_async(request, stream=False):
            response = resp

        assert response is not None
        assert response.content is not None
        assert len(response.content.parts) > 0
        # Response should contain "8" (the answer)
        assert "8" in response.content.parts[0].text

    async def test_agent_with_custom_temperature(self, ollama_model):
        """Test agent with custom temperature setting."""
        from kagent.adk.types import AgentConfig, Ollama

        # Create agent with low temperature for deterministic responses
        config = AgentConfig(
            model=Ollama(
                type="ollama",
                model=ollama_model,
                temperature=0.1,
            ),
            description="Deterministic agent",
            instruction="Answer concisely.",
        )

        agent = config.to_agent(name="det_agent")

        # Verify temperature is applied
        assert agent.model.temperature == 0.1

        # Execute query
        from google.adk.models.llm_request import LlmRequest

        request = LlmRequest(
            contents=[types.Content(role="user", parts=[types.Part(text="Say 'test'")])]
        )

        response = None
        async for resp in agent.model.generate_content_async(request, stream=False):
            response = resp

        assert response is not None
        assert response.content is not None

    async def test_agent_with_max_tokens(self, ollama_model):
        """Test agent with max_tokens configuration."""
        from kagent.adk.types import AgentConfig, Ollama

        config = AgentConfig(
            model=Ollama(
                type="ollama",
                model=ollama_model,
                max_tokens=50,
            ),
            description="Limited token agent",
            instruction="Answer questions.",
        )

        agent = config.to_agent(name="limited_agent")

        # Verify max_tokens is applied
        assert agent.model.max_tokens == 50

        # Execute query
        from google.adk.models.llm_request import LlmRequest

        request = LlmRequest(
            contents=[types.Content(role="user", parts=[types.Part(text="Tell me a story")])]
        )

        response = None
        async for resp in agent.model.generate_content_async(request, stream=False):
            response = resp

        assert response is not None
        assert response.content is not None
        # Response should be limited by max_tokens
        if response.usage_metadata:
            assert response.usage_metadata.candidates_token_count <= 50

    async def test_agent_streaming_integration(self, ollama_model):
        """Test agent execution with streaming."""
        from kagent.adk.types import AgentConfig, Ollama

        config = AgentConfig(
            model=Ollama(type="ollama", model=ollama_model),
            description="Streaming agent",
            instruction="Be helpful.",
        )

        agent = config.to_agent(name="stream_agent")

        # Execute streaming query
        from google.adk.models.llm_request import LlmRequest

        request = LlmRequest(
            contents=[types.Content(role="user", parts=[types.Part(text="Count to 3")])]
        )

        responses = []
        async for resp in agent.model.generate_content_async(request, stream=True):
            responses.append(resp)

        # Validate streaming responses
        assert len(responses) > 0
        
        # Last response should have usage metadata
        final_response = responses[-1]
        assert final_response.content is not None

    async def test_agent_error_handling(self, ollama_base_url):
        """Test agent error handling with invalid model."""
        from kagent.adk.types import AgentConfig, Ollama

        config = AgentConfig(
            model=Ollama(
                type="ollama",
                model="nonexistent-model-12345",
                base_url=ollama_base_url,
            ),
            description="Error test agent",
            instruction="Test error handling.",
        )

        agent = config.to_agent(name="error_agent")

        # Execute query that should fail
        from google.adk.models.llm_request import LlmRequest

        request = LlmRequest(
            contents=[types.Content(role="user", parts=[types.Part(text="Hello")])]
        )

        response = None
        async for resp in agent.model.generate_content_async(request, stream=False):
            response = resp

        # Should receive error response
        assert response is not None
        assert response.error_code is not None
        assert "OLLAMA" in response.error_code

    async def test_agent_config_validation(self):
        """Test AgentConfig validation with Ollama model."""
        from kagent.adk.types import AgentConfig, Ollama

        # Valid configuration
        config = AgentConfig(
            model=Ollama(type="ollama", model="llama2"),
            description="Valid agent",
            instruction="Be helpful.",
        )

        assert config.model.type == "ollama"
        assert config.model.model == "llama2"

        # Convert to agent should succeed
        agent = config.to_agent(name="valid_agent")
        assert agent is not None

    async def test_agent_default_base_url(self, ollama_model):
        """Test agent uses default base_url when not specified."""
        from kagent.adk.types import AgentConfig, Ollama

        config = AgentConfig(
            model=Ollama(type="ollama", model=ollama_model),
            description="Default URL agent",
            instruction="Test defaults.",
        )

        agent = config.to_agent(name="default_agent")

        # Should use default Ollama base_url
        from kagent.adk.models._ollama import OllamaNative
        assert isinstance(agent.model, OllamaNative)
        assert agent.model.base_url == "http://localhost:11434"

    async def test_agent_with_custom_headers(self, ollama_model):
        """Test agent with custom HTTP headers."""
        from kagent.adk.types import AgentConfig, Ollama

        custom_headers = {"X-Custom-Header": "test-value"}

        config = AgentConfig(
            model=Ollama(
                type="ollama",
                model=ollama_model,
                headers=custom_headers,
            ),
            description="Custom headers agent",
            instruction="Test headers.",
        )

        agent = config.to_agent(name="headers_agent")

        # Verify headers are applied
        assert agent.model.headers == custom_headers

