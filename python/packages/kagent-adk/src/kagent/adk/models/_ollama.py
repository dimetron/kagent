"""Ollama native driver for kagent ADK.

This module provides a native Ollama driver that communicates directly with
Ollama's OpenAI-compatible API (/v1/chat/completions) without requiring
LiteLLM as an intermediate layer.

The implementation follows the same pattern as the OpenAI driver, reusing
converters where possible and maintaining full backward compatibility with
existing Ollama configurations.
"""

from __future__ import annotations

import base64
import json
from functools import cached_property
from typing import TYPE_CHECKING, Any, AsyncGenerator, Literal, Optional

import httpx
from google.adk.models import BaseLlm
from google.adk.models.llm_response import LlmResponse
from google.genai import types
from google.genai.types import FunctionCall, FunctionResponse
from pydantic import Field

if TYPE_CHECKING:
    from google.adk.models.llm_request import LlmRequest


def _convert_role_to_ollama(role: Optional[str]) -> str:
    """Convert google.genai role to Ollama role (OpenAI-compatible)."""
    if role in ["model", "assistant"]:
        return "assistant"
    elif role == "system":
        return "system"
    else:
        return "user"


def _convert_content_to_ollama_messages(
    contents: list[types.Content], system_instruction: Optional[str] = None
) -> list[dict[str, Any]]:
    """Convert google.genai Content list to Ollama messages format.

    Args:
        contents: List of ADK Content objects
        system_instruction: Optional system instruction

    Returns:
        List of Ollama-compatible message dicts in OpenAI format
    """
    messages: list[dict[str, Any]] = []

    # Add system message if provided
    if system_instruction:
        messages.append({"role": "system", "content": system_instruction})

    # First pass: collect all function responses
    all_function_responses: dict[str, FunctionResponse] = {}
    for content in contents:
        for part in content.parts or []:
            if part.function_response:
                tool_call_id = part.function_response.id or "call_1"
                all_function_responses[tool_call_id] = part.function_response

    for content in contents:
        role = _convert_role_to_ollama(content.role)

        # Separate different types of parts
        text_parts: list[str] = []
        function_calls: list[FunctionCall] = []
        function_responses: list[FunctionResponse] = []
        image_parts = []

        for part in content.parts or []:
            if part.text:
                text_parts.append(part.text)
            elif part.function_call:
                function_calls.append(part.function_call)
            elif part.function_response:
                function_responses.append(part.function_response)
            elif part.inline_data and part.inline_data.mime_type and part.inline_data.mime_type.startswith("image"):
                if part.inline_data.data:
                    image_data = base64.b64encode(part.inline_data.data).decode()
                    image_part = {
                        "type": "image_url",
                        "image_url": {"url": f"data:{part.inline_data.mime_type};base64,{image_data}"},
                    }
                    image_parts.append(image_part)

        # Handle function calls (assistant messages with tool_calls)
        if function_calls:
            tool_calls = []
            tool_response_messages = []

            for func_call in function_calls:
                tool_call_id = func_call.id or "call_1"
                tool_call = {
                    "id": tool_call_id,
                    "type": "function",
                    "function": {
                        "name": func_call.name or "",
                        "arguments": json.dumps(func_call.args) if func_call.args else "{}",
                    },
                }
                tool_calls.append(tool_call)

                # Check if we have a response for this tool call
                if tool_call_id in all_function_responses:
                    func_response = all_function_responses[tool_call_id]
                    tool_message = {
                        "role": "tool",
                        "tool_call_id": tool_call_id,
                        "content": str(func_response.response.get("result", "")) if func_response.response else "",
                    }
                    tool_response_messages.append(tool_message)
                else:
                    # Placeholder response
                    tool_message = {
                        "role": "tool",
                        "tool_call_id": tool_call_id,
                        "content": "No response available for this function call.",
                    }
                    tool_response_messages.append(tool_message)

            # Create assistant message with tool calls
            text_content = "\n".join(text_parts) if text_parts else None
            assistant_message: dict[str, Any] = {
                "role": "assistant",
                "content": text_content,
                "tool_calls": tool_calls,
            }
            messages.append(assistant_message)

            # Add tool response messages
            messages.extend(tool_response_messages)

        # Handle regular text/image messages
        elif text_parts or image_parts:
            if role == "user":
                if image_parts and text_parts:
                    # Multi-modal content
                    text_part = {"type": "text", "text": "\n".join(text_parts)}
                    content_parts = [text_part] + image_parts
                    user_message = {"role": "user", "content": content_parts}
                elif image_parts:
                    # Image only
                    user_message = {"role": "user", "content": image_parts}
                else:
                    # Text only
                    user_message = {"role": "user", "content": "\n".join(text_parts)}
                messages.append(user_message)
            elif role == "assistant":
                # Assistant messages with text (no tool calls)
                assistant_message = {
                    "role": "assistant",
                    "content": "\n".join(text_parts),
                }
                messages.append(assistant_message)

    return messages


def _update_type_string(value_dict: dict[str, Any]) -> None:
    """Updates 'type' field to expected JSON schema format."""
    if "type" in value_dict:
        value_dict["type"] = value_dict["type"].lower()

    if "items" in value_dict:
        _update_type_string(value_dict["items"])
        if "properties" in value_dict["items"]:
            for _, value in value_dict["items"]["properties"].items():
                _update_type_string(value)

    if "properties" in value_dict:
        for _, value in value_dict["properties"].items():
            _update_type_string(value)


def _convert_tools_to_ollama(tools: list[types.Tool]) -> list[dict[str, Any]]:
    """Convert google.genai Tools to Ollama tools format.

    Args:
        tools: List of ADK Tool objects

    Returns:
        List of Ollama-compatible tool definitions in OpenAI format
    """
    ollama_tools: list[dict[str, Any]] = []

    for tool in tools:
        if tool.function_declarations:
            for func_decl in tool.function_declarations:
                # Build function definition
                function_def: dict[str, Any] = {
                    "name": func_decl.name or "",
                    "description": func_decl.description or "",
                }

                # Build parameters
                properties = {}
                required = []

                if func_decl.parameters:
                    if func_decl.parameters.properties:
                        for prop_name, prop_schema in func_decl.parameters.properties.items():
                            value_dict = prop_schema.model_dump(exclude_none=True)
                            _update_type_string(value_dict)
                            properties[prop_name] = value_dict

                    if func_decl.parameters.required:
                        required = func_decl.parameters.required

                function_def["parameters"] = {"type": "object", "properties": properties, "required": required}

                # Create the tool param
                ollama_tool = {"type": "function", "function": function_def}
                ollama_tools.append(ollama_tool)

    return ollama_tools


def _convert_ollama_response_to_llm_response(response: dict[str, Any]) -> LlmResponse:
    """Convert Ollama response to ADK LlmResponse.

    Args:
        response: Ollama API response dict

    Returns:
        ADK LlmResponse object
    """
    choice = response["choices"][0]
    message = choice["message"]

    parts = []

    # Handle text content
    if message.get("content"):
        parts.append(types.Part.from_text(text=message["content"]))

    # Handle function calls
    if message.get("tool_calls"):
        for tool_call in message["tool_calls"]:
            if tool_call["type"] == "function":
                try:
                    args = json.loads(tool_call["function"]["arguments"]) if tool_call["function"]["arguments"] else {}
                except json.JSONDecodeError:
                    args = {}

                part = types.Part.from_function_call(name=tool_call["function"]["name"], args=args)
                if part.function_call:
                    part.function_call.id = tool_call["id"]
                parts.append(part)

    content = types.Content(role="model", parts=parts)

    # Handle usage metadata
    usage_metadata = None
    if response.get("usage"):
        usage = response["usage"]
        usage_metadata = types.GenerateContentResponseUsageMetadata(
            prompt_token_count=usage.get("prompt_tokens", 0),
            candidates_token_count=usage.get("completion_tokens", 0),
            total_token_count=usage.get("total_tokens", 0),
        )

    # Handle finish reason
    finish_reason = types.FinishReason.STOP
    if choice.get("finish_reason") == "length":
        finish_reason = types.FinishReason.MAX_TOKENS
    elif choice.get("finish_reason") == "content_filter":
        finish_reason = types.FinishReason.SAFETY

    return LlmResponse(content=content, usage_metadata=usage_metadata, finish_reason=finish_reason)


class OllamaNative(BaseLlm):
    """Native Ollama driver implementing BaseLlm interface.

    This driver communicates directly with Ollama's OpenAI-compatible API,
    providing better performance and control compared to the LiteLLM proxy.

    Attributes:
        type: Model type discriminator (always "ollama")
        model: Ollama model name (e.g., "llama2", "mistral", "llama3.1")
        base_url: Ollama server URL (default: http://localhost:11434)
        temperature: Sampling temperature (0.0-2.0)
        max_tokens: Maximum tokens to generate
        timeout: HTTP request timeout in seconds
        headers: Custom HTTP headers for authentication/proxy
    """

    type: Literal["ollama"]
    model: str
    base_url: Optional[str] = "http://localhost:11434"
    temperature: Optional[float] = None
    max_tokens: Optional[int] = None
    timeout: Optional[float] = 60.0
    headers: Optional[dict[str, str]] = Field(default=None, exclude=True)

    @classmethod
    def supported_models(cls) -> list[str]:
        """Returns regex patterns for supported models.

        Ollama supports any model name, so we return a wildcard pattern
        that matches all models for LlmRegistry registration.
        """
        return [r".*"]

    @cached_property
    def _client(self) -> httpx.AsyncClient:
        """Get the cached HTTP client for Ollama API.

        Creates an async HTTP client with connection pooling for optimal
        performance. The client is cached per OllamaNative instance.
        """
        client_kwargs = {
            "base_url": self.base_url or "http://localhost:11434",
            "timeout": httpx.Timeout(self.timeout or 60.0),
        }

        if self.headers:
            client_kwargs["headers"] = self.headers

        return httpx.AsyncClient(**client_kwargs)

    async def generate_content_async(
        self, llm_request: LlmRequest, stream: bool = False
    ) -> AsyncGenerator[LlmResponse, None]:
        """Generate content using Ollama API.

        Args:
            llm_request: The LLM request containing messages and configuration
            stream: Whether to stream responses (SSE)

        Yields:
            LlmResponse objects containing generated content or errors
        """
        try:
            # Extract system instruction
            system_instruction = None
            if llm_request.config and llm_request.config.system_instruction:
                if isinstance(llm_request.config.system_instruction, str):
                    system_instruction = llm_request.config.system_instruction
                elif hasattr(llm_request.config.system_instruction, "parts"):
                    text_parts = []
                    parts = getattr(llm_request.config.system_instruction, "parts", [])
                    if parts:
                        for part in parts:
                            if hasattr(part, "text") and part.text:
                                text_parts.append(part.text)
                        system_instruction = "\n".join(text_parts)

            # Convert messages
            messages = _convert_content_to_ollama_messages(llm_request.contents, system_instruction)

            # Prepare request parameters
            request_data: dict[str, Any] = {
                "model": llm_request.model or self.model,
                "messages": messages,
                "stream": stream,
            }

            if self.max_tokens:
                request_data["max_tokens"] = self.max_tokens
            if self.temperature is not None:
                request_data["temperature"] = self.temperature

            # Handle tools
            if llm_request.config and llm_request.config.tools:
                # Filter to only google.genai.types.Tool objects
                genai_tools = []
                for tool in llm_request.config.tools:
                    if hasattr(tool, "function_declarations"):
                        genai_tools.append(tool)

                if genai_tools:
                    ollama_tools = _convert_tools_to_ollama(genai_tools)
                    if ollama_tools:
                        request_data["tools"] = ollama_tools
                        request_data["tool_choice"] = "auto"

            # Make API request
            if stream:
                # Streaming response
                async with self._client.stream("POST", "/v1/chat/completions", json=request_data) as response:
                    if response.status_code != 200:
                        error_text = await response.aread()
                        yield self._handle_error_response(response.status_code, error_text.decode())
                        return

                    async for line in response.aiter_lines():
                        if line.startswith("data: "):
                            chunk_data = line[6:]
                            if chunk_data == "[DONE]":
                                break

                            try:
                                chunk = json.loads(chunk_data)
                                if chunk.get("choices") and chunk["choices"][0].get("delta"):
                                    delta = chunk["choices"][0]["delta"]
                                    if delta.get("content"):
                                        content = types.Content(
                                            role="model", parts=[types.Part.from_text(text=delta["content"])]
                                        )
                                        yield LlmResponse(
                                            content=content,
                                            partial=True,
                                            turn_complete=chunk["choices"][0].get("finish_reason") is not None,
                                        )
                            except json.JSONDecodeError:
                                continue
            else:
                # Non-streaming response
                response = await self._client.post("/v1/chat/completions", json=request_data)

                if response.status_code != 200:
                    yield self._handle_error_response(response.status_code, response.text)
                    return

                response_data = response.json()
                yield _convert_ollama_response_to_llm_response(response_data)

        except httpx.ConnectError as e:
            yield LlmResponse(
                error_code="OLLAMA_CONNECTION_ERROR",
                error_message=f"Failed to connect to Ollama at {self.base_url}: {str(e)}",
            )
        except httpx.TimeoutException as e:
            yield LlmResponse(
                error_code="OLLAMA_TIMEOUT", error_message=f"Ollama request timed out after {self.timeout}s: {str(e)}"
            )
        except Exception as e:
            yield LlmResponse(error_code="OLLAMA_ERROR", error_message=f"Ollama error: {str(e)}")

    def _handle_error_response(self, status_code: int, response_text: str) -> LlmResponse:
        """Handle error responses from Ollama API.

        Args:
            status_code: HTTP status code
            response_text: Response text

        Returns:
            LlmResponse with error information
        """
        if status_code == 404:
            return LlmResponse(
                error_code="OLLAMA_MODEL_NOT_FOUND",
                error_message=f"Model '{self.model}' not found. Run: ollama pull {self.model}",
            )
        elif status_code >= 500:
            return LlmResponse(
                error_code="OLLAMA_SERVER_ERROR",
                error_message=f"Ollama server error (HTTP {status_code}): {response_text}",
            )
        else:
            return LlmResponse(
                error_code="OLLAMA_API_ERROR", error_message=f"Ollama API error (HTTP {status_code}): {response_text}"
            )
