import importlib.metadata

from ._a2a import KAgentApp
from .types import AgentConfig

# Apply MCP progress notification patch to suppress validation warnings
# This must be done early before any MCP connections are established
try:
    from . import mcp_patch  # noqa: F401
except ImportError:
    pass  # MCP not installed or patch not needed

__version__ = importlib.metadata.version("kagent_adk")

__all__ = ["KAgentApp", "AgentConfig"]
