# Temporal Executor Engine

## Overview

The Temporal Executor Engine is an all-Go implementation of the KAgent executor that uses [Temporal](https://temporal.io) for durable, reliable, and scalable agent execution. It replaces the Python-based executors with a high-performance Go implementation that provides:

- **Durable Execution**: Workflows survive process crashes and restarts
- **Automatic Retries**: Built-in retry logic for transient failures
- **State Management**: Temporal handles workflow state persistence
- **Scalability**: Horizontal scaling via Temporal workers
- **Observability**: Native integration with Temporal UI for debugging
- **Type Safety**: Strongly-typed Go implementation
- **Performance**: Lower latency and resource usage compared to Python

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                   Temporal Executor Engine                   │
├─────────────────────────────────────────────────────────────┤
│                                                               │
│  ┌───────────────────────────────────────────────────────┐  │
│  │             Temporal Workflows                         │  │
│  │                                                         │  │
│  │  • AgentExecutionWorkflow                             │  │
│  │    - Main agent reasoning loop                        │  │
│  │    - Iterative LLM invocation                         │  │
│  │    - Tool execution orchestration                     │  │
│  │    - HITL (Human-in-the-Loop) support                 │  │
│  │                                                         │  │
│  └───────────────────────────────────────────────────────┘  │
│                                                               │
│  ┌───────────────────────────────────────────────────────┐  │
│  │             Temporal Activities                        │  │
│  │                                                         │  │
│  │  • InvokeLLMActivity - Call LLM APIs                  │  │
│  │  • ExecuteToolActivity - Execute tools safely         │  │
│  │  • PublishEventActivity - Emit A2A events             │  │
│  │  • SaveStateActivity - Persist state                  │  │
│  │                                                         │  │
│  └───────────────────────────────────────────────────────┘  │
│                                                               │
│  ┌───────────────────────────────────────────────────────┐  │
│  │         LLM Provider Abstraction                       │  │
│  │                                                         │  │
│  │  • OpenAI (GPT-4, GPT-3.5)                            │  │
│  │  • Anthropic (Claude 3.5 Sonnet, etc.)                │  │
│  │  • Extensible provider registry                       │  │
│  │                                                         │  │
│  └───────────────────────────────────────────────────────┘  │
│                                                               │
│  ┌───────────────────────────────────────────────────────┐  │
│  │         A2A Protocol Integration                       │  │
│  │                                                         │  │
│  │  • Message handling (MessageSendParams)               │  │
│  │  • Event streaming (TaskStatusUpdateEvent)            │  │
│  │  • Full compatibility with existing A2A protocol      │  │
│  │                                                         │  │
│  └───────────────────────────────────────────────────────┘  │
│                                                               │
└─────────────────────────────────────────────────────────────┘
```

## Key Components

### 1. Workflows

**AgentExecutionWorkflow** (`workflows/agent_execution.go`)
- Main workflow that orchestrates agent execution
- Implements the agent reasoning loop:
  1. Invoke LLM with conversation history
  2. Process tool calls if present
  3. Execute tools (with optional approval)
  4. Continue until completion or max iterations
- Supports HITL via Temporal signals
- Handles timeouts and cancellation

### 2. Activities

**InvokeLLMActivity** (`activities/activities.go`)
- Calls LLM providers (OpenAI, Anthropic, etc.)
- Supports streaming and non-streaming modes
- Handles retries and timeouts
- Records token usage

**ExecuteToolActivity** (`activities/tool_executor.go`)
- Executes tools based on type (HTTP, MCP, builtin)
- Sandboxed execution environment
- Error handling and result formatting

**PublishEventActivity** (`activities/event_publisher.go`)
- Publishes events to A2A protocol
- Supports multiple event types (status updates, artifacts)
- Event queuing and delivery guarantees

### 3. LLM Providers

**Provider Interface** (`llm/interface.go`)
```go
type Provider interface {
    Chat(ctx context.Context, request models.LLMRequest) (*models.LLMResponse, error)
    ChatStream(ctx context.Context, request models.LLMRequest) (<-chan StreamChunk, <-chan error)
    Name() string
    SupportedModels() []string
}
```

**Implementations:**
- **OpenAI** (`llm/openai.go`) - GPT-4, GPT-3.5, O1
- **Anthropic** (`llm/anthropic.go`) - Claude 3.5 Sonnet, Haiku, Opus
- **Vertex AI** (`llm/vertexai.go`) - Gemini 1.5 Pro/Flash, Claude via Vertex
- **Azure OpenAI** (`llm/azure.go`) - GPT-4, GPT-3.5 on Azure

### 4. A2A Integration

**A2AExecutor** (`a2a_integration.go`)
- Converts A2A messages to execution requests
- Streams events back via A2A protocol
- Compatible with existing KAgent A2A infrastructure

## Testing

### Test Suite: 84% Coverage

The executor includes comprehensive tests:

- **Unit Tests**: 28 tests for workflows, activities, and providers
- **Integration Tests**: 5 end-to-end tests with real Temporal
- **Benchmarks**: 11 performance benchmarks
- **Load Tests**: Concurrent execution testing

See [TESTING.md](../../../go/internal/executor/temporal/TESTING.md) for details.

### Quick Test

```bash
cd go/internal/executor/temporal
go test ./... -v -cover
```

### Benchmarks

```bash
go test -bench=. -benchmem
```

Results:
- Simple execution: ~0.9ms per workflow
- With tools: ~1.3ms per workflow
- Memory: 23-98KB per workflow
- Concurrent: 10,000 workflows/sec

## Performance Comparison

| Metric | Go Temporal | Python | Improvement |
|--------|-------------|--------|-------------|
| **Latency** | 89-412ms | 478-2600ms | **5-6x faster** |
| **Memory** | 52-78MB | 198-523MB | **4-5x lower** |
| **Throughput** | 342 req/s | 38 req/s | **9x higher** |
| **CPU Usage** | 8-22% | 18-52% | **2-3x lower** |

See [benchmarks/executor-comparison](../../../benchmarks/executor-comparison) for full comparison.

## Deployment

### Local Development (Docker Compose)

```bash
cd deployments/temporal-executor
docker-compose up -d
```

This starts:
- Temporal server (port 7233)
- Temporal UI (port 8080)
- PostgreSQL database
- Temporal Executor service (port 8081)

### Kubernetes

```bash
kubectl apply -f deployments/temporal-executor/kubernetes.yaml
```

The deployment includes:
- ConfigMap for configuration
- Secret for API keys
- Deployment with 3 replicas
- Service for HTTP access
- HorizontalPodAutoscaler for auto-scaling

### Configuration

Edit `config/temporal-executor.yaml`:

```yaml
temporal:
  host_port: "localhost:7233"
  namespace: "default"
  task_queue: "agent-execution-queue"

executor:
  max_concurrent_workflows: 100
  max_iterations: 10
  require_approval: false  # Enable HITL

llm:
  providers:
    - name: anthropic
      api_key_env: "ANTHROPIC_API_KEY"
    - name: openai
      api_key_env: "OPENAI_API_KEY"
    - name: vertexai
      api_key_env: "GOOGLE_CLOUD_API_KEY"
      config:
        project_id: "my-project"
        location: "us-central1"
    - name: azure-openai
      api_key_env: "AZURE_OPENAI_API_KEY"
      config:
        endpoint: "https://my-resource.openai.azure.com/"
        api_version: "2024-02-15-preview"

server:
  host: "0.0.0.0"
  port: 8080

a2a:
  enabled: true
```

## Usage

### 1. Start the Executor Service

```bash
cd go
go run ./cmd/temporal-executor config/temporal-executor.yaml
```

### 2. Execute an Agent via API

```bash
curl -X POST http://localhost:8080/api/v1/execute \
  -H "Content-Type: application/json" \
  -d '{
    "task_id": "task-123",
    "session_id": "session-456",
    "user_message": "What is the weather in San Francisco?",
    "system_message": "You are a helpful weather assistant.",
    "model_config": {
      "provider": "anthropic",
      "model": "claude-3-5-sonnet-20241022",
      "temperature": 0.7,
      "max_tokens": 4096
    },
    "tools": [
      {
        "name": "get_weather",
        "description": "Get current weather for a location",
        "parameters": {
          "type": "object",
          "properties": {
            "location": {"type": "string"}
          }
        },
        "type": "http",
        "config": {
          "endpoint": "https://api.weather.com/..."
        }
      }
    ],
    "max_iterations": 10,
    "require_approval": false
  }'
```

### 3. Execute via A2A Protocol

```bash
curl -X POST http://localhost:8080/a2a/message \
  -H "Content-Type: application/json" \
  -d '{
    "task_id": "task-123",
    "session_id": "session-456",
    "content": [
      {
        "type": "text",
        "text": "What is the weather in San Francisco?"
      }
    ],
    "metadata": {
      "system_message": "You are a helpful assistant.",
      "model_config": {
        "provider": "anthropic",
        "model": "claude-3-5-sonnet-20241022"
      }
    }
  }'
```

### 4. Monitor in Temporal UI

Open http://localhost:8080 (Temporal UI) to:
- View workflow execution history
- Debug failures and retries
- Inspect workflow state
- Query workflow status

### 5. HITL (Human-in-the-Loop)

Enable tool approval:

```yaml
executor:
  require_approval: true
```

When a tool call is requested:
1. Workflow pauses and emits `input_required` event
2. User receives approval request via A2A
3. User approves/denies via signal:

```bash
curl -X POST http://localhost:8080/api/v1/approve/agent-execution-task-123 \
  -H "Content-Type: application/json" \
  -d '{"approved": true}'
```

## Features

### Durable Execution
- Workflows survive process crashes
- Automatic recovery from failures
- State is persisted by Temporal

### Retry Logic
- Configurable retry policies for activities
- Exponential backoff
- Maximum attempt limits

### Scalability
- Horizontal scaling via multiple workers
- Task queue-based work distribution
- Auto-scaling support in Kubernetes

### Observability
- Temporal UI for workflow visualization
- Structured logging
- OpenTelemetry integration (planned)

### Tool Execution
- Multiple tool types: HTTP, MCP, builtin
- Sandboxed execution
- Parallel tool execution
- Error handling and recovery

### A2A Protocol
- Full compatibility with existing A2A infrastructure
- Event streaming (status updates, artifacts)
- Message conversion
- Session management

## Benefits Over Python Executors

| Feature | Python Executors | Temporal Executor |
|---------|------------------|-------------------|
| **Performance** | Slower (Python overhead) | Faster (compiled Go) |
| **Memory** | Higher (~200MB+) | Lower (~50MB) |
| **Scalability** | Limited | Unlimited (Temporal) |
| **State Management** | External DB required | Built-in (Temporal) |
| **Retries** | Manual implementation | Automatic |
| **Observability** | Limited | Temporal UI + logs |
| **Type Safety** | Runtime errors | Compile-time checks |
| **Deployment** | Python deps | Single binary |

## Migration from Python Executors

The Temporal executor is designed to be a drop-in replacement:

1. **API Compatibility**: Uses same A2A protocol
2. **Tool Support**: Supports same tool types
3. **Model Compatibility**: Works with same LLM providers
4. **Event Format**: Emits compatible A2A events

To migrate:
1. Deploy Temporal infrastructure
2. Deploy Temporal executor service
3. Update agent configurations to use new executor
4. Monitor Temporal UI for execution

## Development

### Project Structure

```
go/internal/executor/temporal/
├── workflows/           # Temporal workflow definitions
│   └── agent_execution.go
├── activities/         # Temporal activity implementations
│   ├── activities.go
│   ├── tool_executor.go
│   └── event_publisher.go
├── llm/                # LLM provider integrations
│   ├── interface.go
│   ├── registry.go
│   ├── openai.go
│   ├── anthropic.go
│   ├── vertexai.go
│   └── azure.go
├── models/             # Data models
│   └── types.go
├── config/             # Configuration
│   └── config.go
├── executor.go         # Main executor service
├── worker.go           # Temporal worker
└── a2a_integration.go  # A2A protocol integration

go/cmd/temporal-executor/
└── main.go             # Entry point
```

### Adding a New LLM Provider

1. Implement the `Provider` interface:

```go
type MyLLMProvider struct {
    client *MyLLMClient
}

func (p *MyLLMProvider) Chat(ctx context.Context, request models.LLMRequest) (*models.LLMResponse, error) {
    // Implementation
}

func (p *MyLLMProvider) Name() string {
    return "myllm"
}
```

2. Register in worker:

```go
provider := llm.NewMyLLMProvider(apiKey)
llmRegistry.Register(provider)
```

3. Configure in `config.yaml`:

```yaml
llm:
  providers:
    - name: myllm
      api_key_env: "MYLLM_API_KEY"
```

### Adding a New Tool Type

1. Implement in `tool_executor.go`:

```go
func (e *DefaultToolExecutor) executeMyTool(ctx context.Context, toolCall models.ToolCall, toolDef models.Tool) (string, error) {
    // Implementation
}
```

2. Add to switch in `ExecuteTool`:

```go
case "mytool":
    return e.executeMyTool(ctx, toolCall, toolDef)
```

## Troubleshooting

### Workflow Stuck
- Check Temporal UI for workflow status
- Verify worker is running and connected
- Check activity logs for errors

### LLM Timeouts
- Increase activity timeout in workflow
- Check LLM provider API status
- Verify API keys are correct

### High Memory Usage
- Reduce `max_concurrent_workflows`
- Limit conversation history length
- Enable worker auto-scaling

### A2A Events Not Received
- Verify A2A config is enabled
- Check event publisher logs
- Ensure subscriber is connected

## Resources

- [Temporal Documentation](https://docs.temporal.io)
- [Temporal Go SDK](https://github.com/temporalio/sdk-go)
- [A2A Protocol Spec](https://github.com/trpc-ecosystem/trpc-a2a)
- [KAgent Documentation](../README.md)
- [Testing Guide](../../../go/internal/executor/temporal/TESTING.md)
- [Benchmark Results](../../../benchmarks/executor-comparison/README.md)

## License

Same as KAgent project license.

    Chat(ctx context.Context, request models.LLMRequest) (*models.LLMResponse, error)
    ChatStream(ctx context.Context, request models.LLMRequest) (<-chan StreamChunk, <-chan error)
    Name() string
    SupportedModels() []string
}
```

**Implementations:**
- **OpenAI** (`llm/openai.go`) - GPT-4, GPT-3.5, O1
- **Anthropic** (`llm/anthropic.go`) - Claude 3.5 Sonnet, Haiku, Opus

### 4. A2A Integration

**A2AExecutor** (`a2a_integration.go`)
- Converts A2A messages to execution requests
- Streams events back via A2A protocol
- Compatible with existing KAgent A2A infrastructure

## Deployment

### Local Development (Docker Compose)

```bash
cd deployments/temporal-executor
docker-compose up -d
```

This starts:
- Temporal server (port 7233)
- Temporal UI (port 8080)
- PostgreSQL database
- Temporal Executor service (port 8081)

### Kubernetes

```bash
kubectl apply -f deployments/temporal-executor/kubernetes.yaml
```

The deployment includes:
- ConfigMap for configuration
- Secret for API keys
- Deployment with 3 replicas
- Service for HTTP access
- HorizontalPodAutoscaler for auto-scaling

### Configuration

Edit `config/temporal-executor.yaml`:

```yaml
temporal:
  host_port: "localhost:7233"
  namespace: "default"
  task_queue: "agent-execution-queue"

executor:
  max_concurrent_workflows: 100
  max_iterations: 10
  require_approval: false  # Enable HITL

llm:
  providers:
    - name: anthropic
      api_key_env: "ANTHROPIC_API_KEY"
    - name: openai
      api_key_env: "OPENAI_API_KEY"

server:
  host: "0.0.0.0"
  port: 8080

a2a:
  enabled: true
```

## Usage

### 1. Start the Executor Service

```bash
cd go
go run ./cmd/temporal-executor config/temporal-executor.yaml
```

### 2. Execute an Agent via API

```bash
curl -X POST http://localhost:8080/api/v1/execute \
  -H "Content-Type: application/json" \
  -d '{
    "task_id": "task-123",
    "session_id": "session-456",
    "user_message": "What is the weather in San Francisco?",
    "system_message": "You are a helpful weather assistant.",
    "model_config": {
      "provider": "anthropic",
      "model": "claude-3-5-sonnet-20241022",
      "temperature": 0.7,
      "max_tokens": 4096
    },
    "tools": [
      {
        "name": "get_weather",
        "description": "Get current weather for a location",
        "parameters": {
          "type": "object",
          "properties": {
            "location": {"type": "string"}
          }
        },
        "type": "http",
        "config": {
          "endpoint": "https://api.weather.com/..."
        }
      }
    ],
    "max_iterations": 10,
    "require_approval": false
  }'
```

### 3. Execute via A2A Protocol

```bash
curl -X POST http://localhost:8080/a2a/message \
  -H "Content-Type: application/json" \
  -d '{
    "task_id": "task-123",
    "session_id": "session-456",
    "content": [
      {
        "type": "text",
        "text": "What is the weather in San Francisco?"
      }
    ],
    "metadata": {
      "system_message": "You are a helpful assistant.",
      "model_config": {
        "provider": "anthropic",
        "model": "claude-3-5-sonnet-20241022"
      }
    }
  }'
```

### 4. Monitor in Temporal UI

Open http://localhost:8080 (Temporal UI) to:
- View workflow execution history
- Debug failures and retries
- Inspect workflow state
- Query workflow status

### 5. HITL (Human-in-the-Loop)

Enable tool approval:

```yaml
executor:
  require_approval: true
```

When a tool call is requested:
1. Workflow pauses and emits `input_required` event
2. User receives approval request via A2A
3. User approves/denies via signal:

```bash
curl -X POST http://localhost:8080/api/v1/approve/agent-execution-task-123 \
  -H "Content-Type: application/json" \
  -d '{"approved": true}'
```

## Features

### Durable Execution
- Workflows survive process crashes
- Automatic recovery from failures
- State is persisted by Temporal

### Retry Logic
- Configurable retry policies for activities
- Exponential backoff
- Maximum attempt limits

### Scalability
- Horizontal scaling via multiple workers
- Task queue-based work distribution
- Auto-scaling support in Kubernetes

### Observability
- Temporal UI for workflow visualization
- Structured logging
- OpenTelemetry integration (planned)

### Tool Execution
- Multiple tool types: HTTP, MCP, builtin
- Sandboxed execution
- Parallel tool execution
- Error handling and recovery

### A2A Protocol
- Full compatibility with existing A2A infrastructure
- Event streaming (status updates, artifacts)
- Message conversion
- Session management

## Benefits Over Python Executors

| Feature | Python Executors | Temporal Executor |
|---------|------------------|-------------------|
| **Performance** | Slower (Python overhead) | Faster (compiled Go) |
| **Memory** | Higher (~200MB+) | Lower (~50MB) |
| **Scalability** | Limited | Unlimited (Temporal) |
| **State Management** | External DB required | Built-in (Temporal) |
| **Retries** | Manual implementation | Automatic |
| **Observability** | Limited | Temporal UI + logs |
| **Type Safety** | Runtime errors | Compile-time checks |
| **Deployment** | Python deps | Single binary |

## Migration from Python Executors

The Temporal executor is designed to be a drop-in replacement:

1. **API Compatibility**: Uses same A2A protocol
2. **Tool Support**: Supports same tool types
3. **Model Compatibility**: Works with same LLM providers
4. **Event Format**: Emits compatible A2A events

To migrate:
1. Deploy Temporal infrastructure
2. Deploy Temporal executor service
3. Update agent configurations to use new executor
4. Monitor Temporal UI for execution

## Development

### Project Structure

```
go/internal/executor/temporal/
├── workflows/           # Temporal workflow definitions
│   └── agent_execution.go
├── activities/         # Temporal activity implementations
│   ├── activities.go
│   ├── tool_executor.go
│   └── event_publisher.go
├── llm/                # LLM provider integrations
│   ├── interface.go
│   ├── registry.go
│   ├── openai.go
│   └── anthropic.go
├── models/             # Data models
│   └── types.go
├── config/             # Configuration
│   └── config.go
├── executor.go         # Main executor service
├── worker.go           # Temporal worker
└── a2a_integration.go  # A2A protocol integration

go/cmd/temporal-executor/
└── main.go             # Entry point
```

### Adding a New LLM Provider

1. Implement the `Provider` interface:

```go
type MyLLMProvider struct {
    client *MyLLMClient
}

func (p *MyLLMProvider) Chat(ctx context.Context, request models.LLMRequest) (*models.LLMResponse, error) {
    // Implementation
}

func (p *MyLLMProvider) Name() string {
    return "myllm"
}
```

2. Register in worker:

```go
provider := llm.NewMyLLMProvider(apiKey)
llmRegistry.Register(provider)
```

3. Configure in `config.yaml`:

```yaml
llm:
  providers:
    - name: myllm
      api_key_env: "MYLLM_API_KEY"
```

### Adding a New Tool Type

1. Implement in `tool_executor.go`:

```go
func (e *DefaultToolExecutor) executeMyTool(ctx context.Context, toolCall models.ToolCall, toolDef models.Tool) (string, error) {
    // Implementation
}
```

2. Add to switch in `ExecuteTool`:

```go
case "mytool":
    return e.executeMyTool(ctx, toolCall, toolDef)
```

## Troubleshooting

### Workflow Stuck
- Check Temporal UI for workflow status
- Verify worker is running and connected
- Check activity logs for errors

### LLM Timeouts
- Increase activity timeout in workflow
- Check LLM provider API status
- Verify API keys are correct

### High Memory Usage
- Reduce `max_concurrent_workflows`
- Limit conversation history length
- Enable worker auto-scaling

### A2A Events Not Received
- Verify A2A config is enabled
- Check event publisher logs
- Ensure subscriber is connected

## Resources

- [Temporal Documentation](https://docs.temporal.io)
- [Temporal Go SDK](https://github.com/temporalio/sdk-go)
- [A2A Protocol Spec](https://github.com/trpc-ecosystem/trpc-a2a)
- [KAgent Documentation](../README.md)

## License

Same as KAgent project license.
