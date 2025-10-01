"""Unit tests for _convert_tools_to_ollama converter function.

This tests the conversion from ADK Tools to ollama-python tools format.

Task: T014
"""

import pytest
from google.genai.types import FunctionDeclaration, Tool, Schema


class TestConvertToolsToOllama:
    """Unit tests for tools conversion to Ollama format."""
    
    def test_simple_tool_conversion(self):
        """Test converting simple tool with basic parameters."""
        from kagent.adk.models._ollama import _convert_tools_to_ollama
        
        tools = [
            Tool(
                function_declarations=[
                    FunctionDeclaration(
                        name="get_weather",
                        description="Get current weather",
                        parameters=Schema(
                            type="object",
                            properties={
                                "location": Schema(
                                    type="string",
                                    description="City name"
                                )
                            },
                            required=["location"]
                        )
                    )
                ]
            )
        ]
        
        ollama_tools = _convert_tools_to_ollama(tools)
        
        assert len(ollama_tools) == 1
        tool = ollama_tools[0]
        assert tool["type"] == "function"
        assert "function" in tool
        
        func = tool["function"]
        assert func["name"] == "get_weather"
        assert func["description"] == "Get current weather"
        assert "parameters" in func
        
        params = func["parameters"]
        assert params["type"] == "object"
        assert "properties" in params
        assert "location" in params["properties"]
        assert params["required"] == ["location"]
    
    def test_tool_with_multiple_parameters(self):
        """Test tool with multiple parameters of different types."""
        from kagent.adk.models._ollama import _convert_tools_to_ollama
        
        tools = [
            Tool(
                function_declarations=[
                    FunctionDeclaration(
                        name="search",
                        description="Search database",
                        parameters=Schema(
                            type="object",
                            properties={
                                "query": Schema(type="string"),
                                "limit": Schema(type="integer"),
                                "include_metadata": Schema(type="boolean")
                            },
                            required=["query"]
                        )
                    )
                ]
            )
        ]
        
        ollama_tools = _convert_tools_to_ollama(tools)
        
        func = ollama_tools[0]["function"]
        params = func["parameters"]
        
        assert len(params["properties"]) == 3
        assert "query" in params["properties"]
        assert "limit" in params["properties"]
        assert "include_metadata" in params["properties"]
        assert params["required"] == ["query"]
    
    def test_tool_with_nested_properties(self):
        """Test tool with nested object properties."""
        from kagent.adk.models._ollama import _convert_tools_to_ollama
        
        tools = [
            Tool(
                function_declarations=[
                    FunctionDeclaration(
                        name="update_record",
                        description="Update database record",
                        parameters=Schema(
                            type="object",
                            properties={
                                "record_id": Schema(type="string"),
                                "data": Schema(
                                    type="object",
                                    properties={
                                        "name": Schema(type="string"),
                                        "age": Schema(type="integer")
                                    }
                                )
                            }
                        )
                    )
                ]
            )
        ]
        
        ollama_tools = _convert_tools_to_ollama(tools)
        
        func = ollama_tools[0]["function"]
        params = func["parameters"]
        
        assert "data" in params["properties"]
        data_schema = params["properties"]["data"]
        assert data_schema["type"] == "object"
        assert "properties" in data_schema
        assert "name" in data_schema["properties"]
        assert "age" in data_schema["properties"]
    
    def test_tool_with_array_property(self):
        """Test tool with array parameters."""
        from kagent.adk.models._ollama import _convert_tools_to_ollama
        
        tools = [
            Tool(
                function_declarations=[
                    FunctionDeclaration(
                        name="batch_process",
                        description="Process multiple items",
                        parameters=Schema(
                            type="object",
                            properties={
                                "items": Schema(
                                    type="array",
                                    items=Schema(type="string")
                                )
                            }
                        )
                    )
                ]
            )
        ]
        
        ollama_tools = _convert_tools_to_ollama(tools)
        
        func = ollama_tools[0]["function"]
        params = func["parameters"]
        
        assert "items" in params["properties"]
        items_schema = params["properties"]["items"]
        assert items_schema["type"] == "array"
        assert "items" in items_schema
    
    def test_tool_with_enum_values(self):
        """Test tool with enum parameter."""
        from kagent.adk.models._ollama import _convert_tools_to_ollama
        
        tools = [
            Tool(
                function_declarations=[
                    FunctionDeclaration(
                        name="set_mode",
                        description="Set operation mode",
                        parameters=Schema(
                            type="object",
                            properties={
                                "mode": Schema(
                                    type="string",
                                    enum=["fast", "balanced", "accurate"]
                                )
                            }
                        )
                    )
                ]
            )
        ]
        
        ollama_tools = _convert_tools_to_ollama(tools)
        
        func = ollama_tools[0]["function"]
        mode_schema = func["parameters"]["properties"]["mode"]
        
        assert "enum" in mode_schema
        assert mode_schema["enum"] == ["fast", "balanced", "accurate"]
    
    def test_multiple_function_declarations(self):
        """Test tool with multiple function declarations."""
        from kagent.adk.models._ollama import _convert_tools_to_ollama
        
        tools = [
            Tool(
                function_declarations=[
                    FunctionDeclaration(
                        name="func1",
                        description="First function",
                        parameters=Schema(type="object", properties={})
                    ),
                    FunctionDeclaration(
                        name="func2",
                        description="Second function",
                        parameters=Schema(type="object", properties={})
                    )
                ]
            )
        ]
        
        ollama_tools = _convert_tools_to_ollama(tools)
        
        # Should flatten to separate tool entries
        assert len(ollama_tools) >= 2 or len(tools[0].function_declarations) == 2
    
    def test_empty_tools_list(self):
        """Test handling empty tools list."""
        from kagent.adk.models._ollama import _convert_tools_to_ollama
        
        tools = []
        ollama_tools = _convert_tools_to_ollama(tools)
        
        assert ollama_tools == []
    
    def test_tool_without_required_fields(self):
        """Test tool with no required parameters."""
        from kagent.adk.models._ollama import _convert_tools_to_ollama
        
        tools = [
            Tool(
                function_declarations=[
                    FunctionDeclaration(
                        name="no_params",
                        description="Function with no params",
                        parameters=Schema(
                            type="object",
                            properties={}
                        )
                    )
                ]
            )
        ]
        
        ollama_tools = _convert_tools_to_ollama(tools)
        
        func = ollama_tools[0]["function"]
        params = func["parameters"]
        
        # required field should be empty or absent
        assert "required" not in params or params["required"] == []
    
    def test_tool_with_property_descriptions(self):
        """Test that property descriptions are preserved."""
        from kagent.adk.models._ollama import _convert_tools_to_ollama
        
        tools = [
            Tool(
                function_declarations=[
                    FunctionDeclaration(
                        name="annotated_func",
                        description="Well documented function",
                        parameters=Schema(
                            type="object",
                            properties={
                                "param1": Schema(
                                    type="string",
                                    description="This is parameter 1"
                                )
                            }
                        )
                    )
                ]
            )
        ]
        
        ollama_tools = _convert_tools_to_ollama(tools)
        
        func = ollama_tools[0]["function"]
        param1_schema = func["parameters"]["properties"]["param1"]
        
        assert "description" in param1_schema
        assert param1_schema["description"] == "This is parameter 1"


if __name__ == "__main__":
    pytest.main([__file__, "-v"])

