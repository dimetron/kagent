# Temporal Executor Examples

Examples demonstrating how to use the Temporal Executor Engine.

## Prerequisites

1. **Start Temporal Server:**

```bash
cd ../../deployments/temporal-executor
docker-compose up -d
```

2. **Set API Keys:**

```bash
export ANTHROPIC_API_KEY="your-anthropic-key"
export OPENAI_API_KEY="your-openai-key"
```

3. **Start Worker:**

```bash
cd ../../go
go run ./cmd/temporal-executor ../config/temporal-executor.yaml
```

## Examples

### 1. Simple Agent Execution

Run a basic agent that writes a Python function:

```bash
cd examples/temporal-executor
go run simple-agent.go
```

This example shows:
- Creating an execution request
- Starting a workflow asynchronously
- Subscribing to events
- Waiting for completion
- Processing results

### 2. HITL (Human-in-the-Loop)

Run an agent that requires tool approval:

```bash
go run hitl-agent.go
```

This example shows:
- Enabling tool approval
- Receiving approval requests
- Approving/denying tool execution
- Resuming workflow after approval

### 3. Multi-Tool Agent

Run an agent with multiple tools:

```bash
go run multi-tool-agent.go
```

This example shows:
- Defining multiple tools
- Parallel tool execution
- Error handling
- Tool result processing

### 4. Streaming Agent

Run an agent with streaming responses:

```bash
go run streaming-agent.go
```

This example shows:
- Streaming LLM responses
- Real-time event updates
- Progress tracking

### 5. Custom LLM Provider

Example of adding a custom LLM provider:

```bash
go run custom-provider.go
```

This example shows:
- Implementing Provider interface
- Registering custom provider
- Using custom provider in workflow

## Example Output

```
Starting agent execution...
Workflow started: agent-execution-example-task-1 (RunID: abc123...)
Waiting for execution to complete...
[Event] status_update: map[status:working iteration:1]
[Event] status_update: map[status:working iteration:2]
[Event] status_update: map[status:completed iteration:2]

============================================================
Execution completed!
============================================================
Status: completed
Iterations: 2
Duration: 5.234s
Tokens Used: 1234

Result:
Here's a Python function to calculate factorial:

```python
def factorial(n):
    """Calculate the factorial of a number."""
    if n < 0:
        raise ValueError("Factorial not defined for negative numbers")
    if n == 0 or n == 1:
        return 1
    return n * factorial(n - 1)
```

Workflow Details:
Workflow ID: agent-execution-example-task-1
Run ID: abc123...

View in Temporal UI: http://localhost:8080
```

## Monitoring

Open Temporal UI to monitor workflows:
- **URL:** http://localhost:8080
- **Namespace:** default
- **Task Queue:** agent-execution-queue

## Cleanup

Stop Temporal and services:

```bash
cd ../../deployments/temporal-executor
docker-compose down
```

## Next Steps

- Read [docs/temporal-executor.md](../../docs/temporal-executor.md) for detailed documentation
- Explore workflow definitions in `go/internal/executor/temporal/workflows/`
- Check activity implementations in `go/internal/executor/temporal/activities/`
- Customize configuration in `config/temporal-executor.yaml`
