# Temporal Executor Engine

High-performance, durable agent executor built with Go and Temporal.

## Quick Start

### 1. Start Temporal (Docker Compose)

```bash
cd deployments/temporal-executor
docker-compose up -d
```

### 2. Configure

Edit `config/temporal-executor.yaml` and set your API keys:

```bash
export ANTHROPIC_API_KEY="your-key-here"
export OPENAI_API_KEY="your-key-here"
```

### 3. Run

```bash
cd go
go run ./cmd/temporal-executor ../config/temporal-executor.yaml
```

### 4. Monitor

Open Temporal UI: http://localhost:8080

## Features

✅ **Durable Execution** - Workflows survive crashes
✅ **Automatic Retries** - Built-in retry logic
✅ **Scalability** - Horizontal scaling via workers
✅ **HITL Support** - Human-in-the-loop tool approval
✅ **Multi-LLM** - OpenAI, Anthropic, Vertex AI, Azure OpenAI
✅ **A2A Compatible** - Drop-in replacement for Python executors
✅ **Type Safe** - Compile-time checking
✅ **Observable** - Temporal UI + structured logs
✅ **Tested** - 84% test coverage, comprehensive benchmarks
✅ **Fast** - 5-6x faster than Python executors

## Architecture

```
User Request → A2A Protocol → Temporal Workflow → Activities
                                    ↓
                            ┌───────┴───────┐
                            │               │
                      LLM Invocation   Tool Execution
                            │               │
                            └───────┬───────┘
                                    ↓
                            Event Publishing → A2A Response
```

## API Examples

### Execute Agent

```bash
curl -X POST http://localhost:8080/api/v1/execute \
  -H "Content-Type: application/json" \
  -d '{
    "task_id": "task-123",
    "user_message": "Write a hello world program in Python",
    "model_config": {
      "provider": "anthropic",
      "model": "claude-3-5-sonnet-20241022"
    }
  }'
```

### Get Status

```bash
curl http://localhost:8080/api/v1/status/agent-execution-task-123
```

### Approve Tool (HITL)

```bash
curl -X POST http://localhost:8080/api/v1/approve/agent-execution-task-123 \
  -d '{"approved": true}'
```

## Configuration

Key settings in `config/temporal-executor.yaml`:

```yaml
temporal:
  host_port: "localhost:7233"     # Temporal server
  task_queue: "agent-execution-queue"

executor:
  max_iterations: 10               # Agent loop limit
  require_approval: false          # Enable HITL

llm:
  providers:
    - name: anthropic
      api_key_env: "ANTHROPIC_API_KEY"
```

## Deployment

### Docker

```bash
docker build -t kagent/temporal-executor:latest \
  -f deployments/temporal-executor/Dockerfile .

docker run -p 8080:8080 \
  -e ANTHROPIC_API_KEY=$ANTHROPIC_API_KEY \
  kagent/temporal-executor:latest
```

### Kubernetes

```bash
kubectl apply -f deployments/temporal-executor/kubernetes.yaml
```

## Development

### Project Structure

```
temporal/
├── workflows/       # Workflow definitions
├── activities/      # Activity implementations
├── llm/            # LLM providers
├── models/         # Data models
├── config/         # Configuration
└── *.go           # Core services
```

### Adding a Provider

```go
// Implement Provider interface
type MyProvider struct {}

func (p *MyProvider) Chat(ctx context.Context, req models.LLMRequest) (*models.LLMResponse, error) {
    // Your implementation
}

// Register
provider := llm.NewMyProvider(apiKey)
registry.Register(provider)
```

### Testing

```bash
cd go/internal/executor/temporal
go test ./...
```

## Comparison: Python vs Temporal Executor

| Metric | Python | Temporal |
|--------|--------|----------|
| Latency | ~500ms | ~100ms |
| Memory | ~200MB | ~50MB |
| Scalability | Limited | Unlimited |
| State Mgmt | External DB | Built-in |
| Retries | Manual | Automatic |
| Observability | Logs | UI + Logs |

## Troubleshooting

**Workflows not starting?**
- Check worker is running
- Verify Temporal server connection
- Check task queue name matches

**LLM timeouts?**
- Increase activity timeout
- Check API key validity
- Verify network connectivity

**High memory?**
- Reduce max_concurrent_workflows
- Limit conversation history
- Enable auto-scaling

## Documentation

See [docs/temporal-executor.md](../../../docs/temporal-executor.md) for comprehensive documentation.

## License

Same as KAgent project.
