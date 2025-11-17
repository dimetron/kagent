# KAgent ADK (Agent Development Kit) - Go Implementation

This package provides a Go implementation of the KAgent ADK, equivalent to the `kagent-adk` package. It enables building and running AI agents with support for:

- **Session Management**: HTTP-based session service integration with KAgent
- **Tool Execution**: File operations, bash commands, skills, and artifacts
- **Multiple Model Providers**: OpenAI, Anthropic, Google Gemini
- **A2A Protocol Support**: Agent-to-Agent communication protocol
- **Local & Production Modes**: Run agents locally or with KAgent orchestration

## Architecture

The ADK package is organized into the following components:

### Core Components

- **`adk.go`**: Main application setup and HTTP server
- **`config/`**: Agent and model configuration types
- **`session/`**: Session management and path handling
- **`tools/`**: Tool implementations (file, bash, skills)
- **`auth/`**: Token service for KAgent authentication
- **`errors/`**: Error types and codes

### Package Structure

```
go/pkg/adk/
├── adk.go              # Main App and HTTP server
├── agent.go            # Legacy compatibility
├── auth/
│   └── token.go        # Token service
├── config/
│   └── types.go        # Agent and model configs
├── errors/
│   └── errors.go       # Error types
├── session/
│   ├── service.go      # Session service interface
│   ├── kagent_service.go  # KAgent HTTP client
│   ├── paths.go        # Session path manager
│   └── types.go        # Session types
└── tools/
    ├── types.go        # Tool interface
    ├── files.go        # File operations
    ├── bash.go         # Bash execution
    └── skills.go       # Skills management
```

## Usage

### CLI Commands

The ADK package is accessible through the `kagent adk` CLI commands:

```bash
# Run an agent
kagent adk run my-agent

# Run in local mode (in-memory sessions)
kagent adk run my-agent --local

# Run with static configuration
kagent adk static --config /etc/kagent

# Test an agent
kagent adk test --config config.json --task "What is 2+2?"
```

### Configuration

Create a `config.json` file with your agent configuration:

```json
{
  "model": {
    "type": "OpenAI",
    "model": "gpt-4",
    "temperature": 0.7
  },
  "description": "My AI agent",
  "instruction": "You are a helpful assistant.",
  "execute_code": true
}
```

### Environment Variables

- `KAGENT_URL`: KAgent server URL (default: `http://localhost:8083`)
- `APP_NAME`: Agent name (default: `kagent-app`)
- `KAGENT_SKILLS_FOLDER`: Path to skills directory
- `KAGENT_TOKEN_PATH`: Path to auth token (default: `/var/run/secrets/tokens/kagent-token`)
- `BASH_VENV_PATH`: Path to bash virtual environment (default: `/.kagent/sandbox-venv`)
- `HOST`: Server host (default: `0.0.0.0`)
- `PORT`: Server port (default: `8080`)
- `LOG_LEVEL`: Logging level (default: `INFO`)

### Programmatic Usage

```go
package main

import (
    "context"
    "github.com/kagent-dev/kagent/go/pkg/adk"
    "github.com/kagent-dev/kagent/go/pkg/adk/config"
)

func main() {
    // Create agent config
    agentCfg := &config.AgentConfig{
        Model: &config.OpenAIConfig{
            BaseModelConfig: config.BaseModelConfig{ModelType: "OpenAI"},
            Model:          "gpt-4",
        },
        Description: "My agent",
        Instruction: "You are a helpful assistant",
    }

    // Create app
    app, err := adk.NewApp(nil, agentCfg)
    if err != nil {
        panic(err)
    }

    // Build and start server
    ctx := context.Background()
    server, err := app.Build(ctx)
    if err != nil {
        panic(err)
    }

    server.ListenAndServe()
}
```

## Supported Model Providers

### OpenAI

```json
{
  "model": {
    "type": "OpenAI",
    "model": "gpt-4",
    "temperature": 0.7,
    "max_tokens": 2000
  }
}
```

### Anthropic

```json
{
  "model": {
    "type": "Anthropic",
    "model": "claude-3-5-sonnet-20241022",
    "temperature": 0.7,
    "max_tokens": 4096
  }
}
```

### Google Gemini

```json
{
  "model": {
    "type": "Gemini",
    "model": "gemini-2.0-flash",
    "temperature": 0.9
  }
}
```

## Tools

The ADK provides the following built-in tools:

### File Tools

- **read_file**: Read file with line numbers and optional offset/limit
- **write_file**: Write content to a file, creating directories as needed
- **edit_file**: Edit file by replacing strings (exact match or replace all)

### Bash Tool

- **bash**: Execute bash commands in a sandboxed environment
  - Automatic timeout based on command type
  - Environment isolation with virtual environment support

### Skills Tool

- **skills**: Invoke skills from the skills directory
  - Skills are defined in `SKILL.md` files with YAML frontmatter
  - Automatic discovery and caching

## Session Management

The ADK manages session-specific directories under `/tmp/kagent/{session_id}/`:

```
/tmp/kagent/{session_id}/
├── skills/      # Symlink to skills directory
├── uploads/     # Staged user files
└── outputs/     # Generated files
```

## Differences from Implementation

This Go implementation provides the same functionality as the implementation with the following differences:

1. **Concurrency**: Uses Go's goroutines and channels instead of async/await
2. **Type Safety**: Stronger compile-time type checking
3. **Performance**: Native binary compilation for better performance
4. **Deployment**: Single binary deployment without interpreter

## Feature Flag

The ADK commands are available alongside the existing agent commands. Use `kagent adk` for ADK-based agents and `kagent agent` for existing Kubernetes-based agents.

## Development Status

**Current Status**: Core implementation complete

**Completed**:
- ✅ Package structure
- ✅ Configuration types (AgentConfig, ModelConfig)
- ✅ Session management (SessionService, PathManager)
- ✅ Tools framework (files, bash, skills)
- ✅ Main application (App, HTTP server)
- ✅ CLI commands (run, static, test)
- ✅ CLI integration with feature flag

**TODO**:
- ⏳ Converters (A2A ↔ ADK protocol conversion)
- ⏳ Agent executor (A2aAgentExecutor)
- ⏳ Code executor (sandboxed execution)
- ⏳ Skills fetcher (OCI registry integration)
- ⏳ In-memory session service (for local mode)
- ⏳ Artifact tools (stage/return artifacts)
- ⏳ MCP server integration
- ⏳ Comprehensive test coverage

## Contributing

When extending the ADK package:

1. Follow Go idioms and conventions
2. Add comprehensive error handling
3. Use context for cancellation and timeouts
4. Write unit tests for new functionality
5. Update this README with new features

## License

Apache License 2.0
