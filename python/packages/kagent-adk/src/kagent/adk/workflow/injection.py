"""State key injection for workflow agents.

This module provides functionality to inject workflow state into agent instructions
using template syntax like {key_name}. This enables sub-agents to reference outputs
from previous agents in their system prompts.
"""

import re
from typing import Dict, List


def inject_state_keys(template: str, state: Dict[str, str]) -> str:
    """Replace {key_name} placeholders with values from workflow state.

    This function performs template variable replacement using regex to find all
    placeholders matching the pattern {key_name} and replaces them with the
    corresponding values from the state dictionary.

    Args:
        template: String containing {key} placeholders
        state: Dictionary of key-value pairs for replacement

    Returns:
        String with placeholders replaced by state values

    Raises:
        ValueError: If a required key is missing from state

    Example:
        >>> template = "Review this code: {generated_code}"
        >>> state = {"generated_code": "def hello(): pass"}
        >>> inject_state_keys(template, state)
        'Review this code: def hello(): pass'
    """
    # Pattern matches {key_name} where key_name is a valid Python identifier
    pattern = r"\{([a-zA-Z_][a-zA-Z0-9_]*)\}"

    def replace_key(match):
        key = match.group(1)
        if key not in state:
            available_keys = list(state.keys())
            raise ValueError(f"State key '{key}' not found. Available keys: {available_keys}")
        return state[key]

    # Single-pass replacement to avoid recursive substitution
    return re.sub(pattern, replace_key, template)


def validate_output_key(key: str) -> None:
    """Validate outputKey naming rules.

    OutputKey names must:
    - Start with a letter (a-z, A-Z) or underscore (_)
    - Contain only letters, digits, and underscores
    - Be at most 127 characters long (namespace 63 + underscore 1 + name 63)
    - Not be empty

    Args:
        key: OutputKey name to validate

    Raises:
        ValueError: If key violates naming rules

    Example:
        >>> validate_output_key("generated_code")  # Valid
        >>> validate_output_key("1result")  # Raises ValueError
    """
    if not key:
        raise ValueError("OutputKey cannot be empty")

    if len(key) > 127:
        raise ValueError(f"OutputKey '{key}' exceeds 127 characters")

    # Pattern: must start with letter or underscore, then letters/digits/underscores
    if not re.match(r"^[a-zA-Z_][a-zA-Z0-9_]*$", key):
        raise ValueError(
            f"OutputKey '{key}' must start with letter/underscore and contain only letters, digits, underscores"
        )


def validate_output_value(value: str, max_size_bytes: int = 10 * 1024 * 1024) -> None:
    """Validate output value size.

    Args:
        value: Output value to validate
        max_size_bytes: Maximum size in bytes (default 10MB)

    Raises:
        ValueError: If value exceeds max_size_bytes

    Example:
        >>> validate_output_value("small value")  # Valid
        >>> validate_output_value("x" * (11 * 1024 * 1024))  # Raises ValueError
    """
    value_bytes = len(value.encode("utf-8"))
    if value_bytes > max_size_bytes:
        raise ValueError(f"Output value exceeds maximum size ({value_bytes} > {max_size_bytes} bytes)")


def validate_unique_output_keys(sub_agents: List) -> None:
    """Ensure outputKey values are unique within workflow.

    Args:
        sub_agents: List of sub-agent references with output_key attributes

    Raises:
        ValueError: If duplicate outputKey found

    Example:
        >>> class Agent:
        ...     def __init__(self, name, output_key):
        ...         self.name = name
        ...         self.output_key = output_key
        >>> agents = [Agent("a", "code"), Agent("b", "review")]
        >>> validate_unique_output_keys(agents)  # Valid
        >>> agents = [Agent("a", "code"), Agent("b", "code")]
        >>> validate_unique_output_keys(agents)  # Raises ValueError
    """
    # Extract output_keys, filtering out None/empty values
    output_keys = []
    for agent in sub_agents:
        if hasattr(agent, "output_key") and agent.output_key:
            output_keys.append(agent.output_key)

    # Check for duplicates
    if len(output_keys) != len(set(output_keys)):
        # Find which keys are duplicated
        seen = set()
        duplicates = set()
        for key in output_keys:
            if key in seen:
                duplicates.add(key)
            seen.add(key)
        raise ValueError(f"Duplicate outputKey values found: {duplicates}")
