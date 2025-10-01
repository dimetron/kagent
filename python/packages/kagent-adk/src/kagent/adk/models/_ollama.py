"""Ollama native driver for kagent ADK using ollama-python.

This module provides a native Ollama driver that communicates directly with
Ollama using the official ollama-python client library. It replaces the
previous LiteLLM-based integration with a more direct and efficient approach.

The implementation uses ollama.AsyncClient for async operations and follows
the BaseLlm interface from google-adk.
"""

from __future__ import annotations

import base64
import json
from functools import cached_property
from typing import TYPE_CHECKING, Any, AsyncGenerator, Literal, Optional

import ollama
from google.adk.models import BaseLlm
from google.adk.models.llm_response import LlmResponse
from google.genai import types
from google.genai.types import FunctionCall, FunctionResponse, Part
from pydantic import Field

if TYPE_CHECKING:
    from google.adk.models.llm_request import LlmRequest


def _convert_role_to_ollama(role: Optional[str]) -> str:
    """Convert google.genai role to Ollama role format."""
    if role in ["model", "assistant"]:
        return "assistant"
    elif role == "system":
        return "system"
    else:
        return "user"


def _convert_content_to_ollama_messages(
    contents: list[types.Content], system_instruction: Optional[str] = None
) -> list[dict[str, Any]]:
    """Convert ADK Content list to ollama-python messages format.

    Args:
        contents: List of ADK Content objects
        system_instruction: Optional system instruction

    Returns:
        List of message dicts compatible with ollama.AsyncClient.chat()
    """
    messages: list[dict[str, Any]] = []

    # Add system message if provided
    if system_instruction:
        messages.append({"role": "system", "content": system_instruction})

    # First pass: collect all function responses to match with tool calls
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
                    image_parts.append(f"data:{part.inline_data.mime_type};base64,{image_data}")

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
                        "content": json.dumps(func_response.response) if func_response.response else "",
                    }
                    tool_response_messages.append(tool_message)

            # Create assistant message with tool calls
            text_content = "\n".join(text_parts) if text_parts else ""
            assistant_message = {"role": "assistant", "content": text_content, "tool_calls": tool_calls}
            messages.append(assistant_message)

            # Add all tool response messages immediately after the assistant message
            messages.extend(tool_response_messages)

        # Handle regular text/image messages (only if no function calls)
        elif text_parts or image_parts:
            content_text = "\n".join(text_parts) if text_parts else ""

            # Ollama supports images in content
            if image_parts:
                # For multimodal, combine text and images
                message = {"role": role, "content": content_text, "images": image_parts}
            else:
                message = {"role": role, "content": content_text}
            messages.append(message)

    return messages


def _convert_tools_to_ollama(tools: list[types.Tool]) -> list[dict[str, Any]]:
    """Convert ADK Tools to ollama-python tools format.

    Args:
        tools: List of ADK Tool objects

    Returns:
        List of tool definitions for ollama.AsyncClient.chat(tools=...)
    """
    ollama_tools = []

    for tool in tools:
        for func_decl in tool.function_declarations or []:
            # Convert schema to dict
            parameters = {}
            if func_decl.parameters:
                # Convert type enum to lowercase string
                type_value = func_decl.parameters.type or "object"
                if hasattr(type_value, "value"):
                    type_value = type_value.value.lower()
                elif isinstance(type_value, str):
                    type_value = type_value.lower()

                parameters = {
                    "type": type_value,
                    "properties": _convert_schema_properties(func_decl.parameters.properties or {}),
                }
                if func_decl.parameters.required:
                    parameters["required"] = func_decl.parameters.required

            ollama_tool = {
                "type": "function",
                "function": {
                    "name": func_decl.name or "",
                    "description": func_decl.description or "",
                    "parameters": parameters,
                },
            }
            ollama_tools.append(ollama_tool)

    return ollama_tools


def _convert_schema_properties(properties: dict[str, Any]) -> dict[str, Any]:
    """Convert schema properties recursively."""
    result = {}
    for key, value in properties.items():
        if isinstance(value, types.Schema):
            # Convert Type enum to lowercase string
            type_value = value.type
            if hasattr(type_value, "value"):
                # It's an enum, get the string value and lowercase it
                type_value = type_value.value.lower()
            elif isinstance(type_value, str):
                type_value = type_value.lower()
            else:
                type_value = "string"

            prop = {"type": type_value}
            if value.description:
                prop["description"] = value.description
            if value.enum:
                prop["enum"] = value.enum
            if value.properties:
                prop["properties"] = _convert_schema_properties(value.properties)
            if value.items:
                prop["items"] = _convert_schema_properties({"item": value.items})["item"]
            result[key] = prop
        else:
            result[key] = value
    return result


def _convert_ollama_response_to_llm_response(response: dict[str, Any]) -> LlmResponse:
    """Convert ollama.ChatResponse to ADK LlmResponse.

    Args:
        response: ollama.ChatResponse object or dict from AsyncClient.chat()

    Returns:
        ADK LlmResponse object
    """
    # Extract message
    message = response.get("message", {})
    role = message.get("role", "assistant")
    content_text = message.get("content", "")
    tool_calls = message.get("tool_calls", [])

    # Create content parts
    parts: list[Part] = []

    # Add text content if present
    if content_text:
        parts.append(Part(text=content_text))

    # Add tool calls if present
    for tool_call in tool_calls:
        function_info = tool_call.get("function", {})
        function_name = function_info.get("name", "")
        arguments_str = function_info.get("arguments", "{}")

        try:
            arguments = json.loads(arguments_str) if arguments_str else {}
        except json.JSONDecodeError:
            arguments = {}

        parts.append(Part(function_call=FunctionCall(id=tool_call.get("id", ""), name=function_name, args=arguments)))

    # Create content - always create content, even if parts is empty
    if not parts and content_text == "":
        # For empty responses, add an empty text part
        parts.append(Part(text=""))

    content = types.Content(role=role, parts=parts)

    # Extract usage metadata
    usage_metadata = None
    prompt_tokens = response.get("prompt_eval_count", 0)
    completion_tokens = response.get("eval_count", 0)

    if prompt_tokens or completion_tokens:
        usage_metadata = types.GenerateContentResponseUsageMetadata(
            prompt_token_count=prompt_tokens,
            candidates_token_count=completion_tokens,
            total_token_count=prompt_tokens + completion_tokens,
        )

    # Determine if this is a partial response
    is_partial = not response.get("done", True)

    return LlmResponse(content=content, usage_metadata=usage_metadata, partial=is_partial)


class OllamaNative(BaseLlm):
    """Native Ollama driver implementing BaseLlm interface using ollama-python.

    This driver uses the official ollama-python client library to communicate
    directly with Ollama's API, providing better performance and control compared
    to using LiteLLM as a proxy layer.

    Attributes:
        type: Discriminator for model type (always "ollama")
        model: Ollama model name (e.g., "llama2", "mistral", "llama3.1")
        base_url: Ollama server URL (default: http://localhost:11434)
        temperature: Sampling temperature (0.0-2.0)
        max_tokens: Maximum tokens to generate
        timeout: HTTP request timeout in seconds
        headers: Custom HTTP headers for authentication/proxy
    """

    type: Literal["ollama"] = "ollama"
    model: str
    base_url: Optional[str] = "http://localhost:11434"
    temperature: Optional[float] = None
    max_tokens: Optional[int] = None
    timeout: Optional[float] = 60.0
    headers: Optional[dict[str, str]] = None

    @cached_property
    def _client(self) -> ollama.AsyncClient:
        """Get the cached Ollama AsyncClient instance.

        Returns:
            ollama.AsyncClient configured with host, headers, and timeout
        """
        # Create AsyncClient with configuration
        client_kwargs = {}

        if self.base_url:
            client_kwargs["host"] = self.base_url

        if self.timeout:
            client_kwargs["timeout"] = self.timeout

        # Headers are handled differently in ollama-python
        # They need to be set on individual requests, not on the client

        return ollama.AsyncClient(**client_kwargs)

    @classmethod
    def supported_models(cls) -> list[str]:
        """Returns regex patterns for supported models.

        Returns:
            List with single pattern matching all model names (Ollama supports any model)
        """
        return [r".*"]  # Ollama supports any model name

    async def generate_content_async(
        self, llm_request: "LlmRequest", stream: bool = False
    ) -> AsyncGenerator[LlmResponse, None]:
        """Generate content using ollama-python AsyncClient.chat() API.

        Args:
            llm_request: ADK LlmRequest containing messages, tools, etc.
            stream: Whether to stream responses

        Yields:
            LlmResponse objects (partial if streaming, complete if not)
        """
        try:
            # Convert ADK format to Ollama format
            messages = _convert_content_to_ollama_messages(
                llm_request.contents, system_instruction=llm_request.system_instruction
            )

            # Convert tools if present
            tools = None
            if llm_request.tools:
                tools = _convert_tools_to_ollama(llm_request.tools)

            # Prepare request options
            options = {}
            if self.temperature is not None:
                options["temperature"] = self.temperature
            if self.max_tokens is not None:
                options["num_predict"] = self.max_tokens

            # Make request
            if stream:
                # Streaming mode
                async for chunk in await self._client.chat(
                    model=self.model,
                    messages=messages,
                    tools=tools if tools else None,
                    stream=True,
                    options=options if options else None,
                ):
                    # Convert chunk to dict if it's an object
                    if hasattr(chunk, "model_dump"):
                        chunk_dict = chunk.model_dump()
                    elif hasattr(chunk, "__dict__"):
                        chunk_dict = chunk.__dict__
                    else:
                        chunk_dict = chunk

                    yield _convert_ollama_response_to_llm_response(chunk_dict)
            else:
                # Non-streaming mode
                response = await self._client.chat(
                    model=self.model,
                    messages=messages,
                    tools=tools if tools else None,
                    stream=False,
                    options=options if options else None,
                )

                # Convert response to dict if it's an object
                if hasattr(response, "model_dump"):
                    response_dict = response.model_dump()
                elif hasattr(response, "__dict__"):
                    response_dict = response.__dict__
                else:
                    response_dict = response

                yield _convert_ollama_response_to_llm_response(response_dict)

        except ollama.ResponseError as e:
            # Handle Ollama-specific errors
            error_code = "OLLAMA_API_ERROR"
            error_message = str(e.error) if hasattr(e, "error") else str(e)

            if hasattr(e, "status_code"):
                if e.status_code == 404:
                    error_code = "OLLAMA_MODEL_NOT_FOUND"
                    error_message = f"Model '{self.model}' not found. Run: ollama pull {self.model}"
                elif e.status_code == 503:
                    error_code = "OLLAMA_SERVICE_UNAVAILABLE"
                    error_message = "Ollama service is not available. Ensure Ollama is running."
                elif e.status_code >= 500:
                    error_code = "OLLAMA_SERVER_ERROR"
                    error_message = f"Ollama server error: {error_message}"

            yield LlmResponse(error_code=error_code, error_message=error_message, content=None, usage_metadata=None)

        except Exception as e:
            # Handle unexpected errors
            error_code = "OLLAMA_CONNECTION_ERROR"
            error_message = f"Failed to connect to Ollama at {self.base_url}: {str(e)}"

            yield LlmResponse(error_code=error_code, error_message=error_message, content=None, usage_metadata=None)
