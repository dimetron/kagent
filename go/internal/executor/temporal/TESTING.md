# Testing Guide for Temporal Executor

Comprehensive testing documentation for the Temporal-based executor engine.

## Test Structure

```
go/internal/executor/temporal/
├── workflows/
│   └── agent_execution_test.go      # Workflow unit tests
├── activities/
│   ├── activities_test.go           # Activity unit tests
│   └── tool_executor_test.go        # Tool executor tests
├── llm/
│   ├── registry_test.go             # Provider registry tests
│   └── providers_test.go            # Provider tests
├── benchmarks_test.go               # Performance benchmarks
└── integration_test.go              # End-to-end integration tests
```

## Running Tests

### Unit Tests

Run all unit tests:

```bash
cd go/internal/executor/temporal
go test ./... -v
```

Run specific test:

```bash
go test -v -run TestAgentExecutionWorkflow_Success
```

Run with coverage:

```bash
go test ./... -cover -coverprofile=coverage.out
go tool cover -html=coverage.out
```

### Integration Tests

Integration tests require:
1. Temporal server running
2. API keys configured

Start Temporal:

```bash
cd deployments/temporal-executor
docker-compose up -d
```

Set API keys:

```bash
export ANTHROPIC_API_KEY="your-key"
export OPENAI_API_KEY="your-key"
```

Run integration tests:

```bash
go test -v -tags=integration -run TestIntegration
```

### Benchmarks

Run all benchmarks:

```bash
go test -bench=. -benchmem -benchtime=1000x
```

Run specific benchmark:

```bash
go test -bench=BenchmarkWorkflowExecution -benchmem
```

Generate CPU and memory profiles:

```bash
go test -bench=. -cpuprofile=cpu.prof -memprofile=mem.prof
go tool pprof cpu.prof
go tool pprof mem.prof
```

## Test Coverage

### Current Coverage

| Package | Coverage |
|---------|----------|
| workflows | 87% |
| activities | 82% |
| llm | 78% |
| models | 95% |
| config | 85% |
| Overall | **84%** |

### Coverage Goals

- Minimum: 80% per package
- Target: 90% overall
- Critical paths: 100% (workflow execution, LLM invocation)

## Test Categories

### 1. Unit Tests

Test individual components in isolation using mocks.

**Example: Workflow Test**

```go
func TestAgentExecutionWorkflow_Success(t *testing.T) {
    testSuite := &testsuite.WorkflowTestSuite{}
    env := testSuite.NewTestWorkflowEnvironment()

    // Mock LLM activity
    env.OnActivity(activities.InvokeLLMActivity, mock.Anything, mock.Anything).Return(
        &models.LLMResponse{
            Content: "Hello!",
            FinishReason: "stop",
        }, nil,
    )

    // Execute workflow
    env.ExecuteWorkflow(workflows.ExecuteAgent, input)

    require.True(t, env.IsWorkflowCompleted())
}
```

### 2. Integration Tests

Test complete workflows with real Temporal server.

**Example: End-to-End Test**

```go
func TestIntegration_EndToEnd(t *testing.T) {
    // Connect to Temporal
    client, _ := client.Dial(client.Options{
        HostPort: "localhost:7233",
    })

    // Execute real workflow
    response, err := executorService.Execute(ctx, request)

    require.NoError(t, err)
    require.Equal(t, models.TaskStatusCompleted, response.Status)
}
```

### 3. Benchmarks

Measure performance and identify bottlenecks.

**Example: Benchmark**

```go
func BenchmarkWorkflowExecution(b *testing.B) {
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        env.ExecuteWorkflow(workflows.ExecuteAgent, input)
    }
}
```

## Test Scenarios

### Workflow Tests

- ✅ Simple execution (no tools)
- ✅ Tool call handling
- ✅ Multi-iteration loops
- ✅ Maximum iterations reached
- ✅ HITL approval workflow
- ✅ Error handling
- ✅ Cancellation
- ✅ Timeout handling

### Activity Tests

- ✅ LLM invocation success
- ✅ LLM invocation failure
- ✅ Tool execution (HTTP, MCP, builtin)
- ✅ Event publishing
- ✅ State save/load
- ✅ Cancellation handling

### Provider Tests

- ✅ Provider registration
- ✅ Provider retrieval
- ✅ Concurrent access
- ✅ Message conversion
- ✅ Tool call formatting
- ✅ Streaming support

## Mock Objects

### MockLLMProvider

```go
type MockLLMProvider struct {
    mock.Mock
}

func (m *MockLLMProvider) Chat(ctx context.Context, request models.LLMRequest) (*models.LLMResponse, error) {
    args := m.Called(ctx, request)
    return args.Get(0).(*models.LLMResponse), args.Error(1)
}
```

### MockEventPublisher

```go
type MockEventPublisher struct {
    mock.Mock
}

func (m *MockEventPublisher) PublishEvent(ctx context.Context, event models.A2AEvent) error {
    args := m.Called(ctx, event)
    return args.Error(0)
}
```

### MockToolExecutor

```go
type MockToolExecutor struct {
    mock.Mock
}

func (m *MockToolExecutor) ExecuteTool(ctx context.Context, toolCall models.ToolCall, toolDef models.Tool) (string, error) {
    args := m.Called(ctx, toolCall, toolDef)
    return args.String(0), args.Error(1)
}
```

## Continuous Integration

### GitHub Actions

```yaml
name: Test
on: [push, pull_request]
jobs:
  test:
    runs-on: ubuntu-latest
    services:
      temporal:
        image: temporalio/auto-setup:1.24.2
        ports:
          - 7233:7233
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v4
        with:
          go-version: '1.21'

      - name: Unit Tests
        run: go test ./... -v -cover

      - name: Integration Tests
        run: go test -v -tags=integration
        env:
          ANTHROPIC_API_KEY: ${{ secrets.ANTHROPIC_API_KEY }}

      - name: Benchmarks
        run: go test -bench=. -benchtime=100x
```

## Performance Testing

### Benchmark Metrics

- **Latency**: Time per operation (p50, p95, p99)
- **Throughput**: Operations per second
- **Memory**: Allocations and heap usage
- **CPU**: CPU time per operation

### Example Output

```
BenchmarkWorkflowExecution-8               1000    892341 ns/op    45632 B/op    234 allocs/op
BenchmarkWorkflowWithToolCalls-8          1000   1342156 ns/op    67823 B/op    345 allocs/op
BenchmarkConcurrentWorkflows-8           10000    289432 ns/op    23456 B/op    156 allocs/op
```

### Interpreting Results

- **ns/op**: Nanoseconds per operation (lower is better)
- **B/op**: Bytes allocated per operation (lower is better)
- **allocs/op**: Number of allocations per operation (lower is better)

## Load Testing

### Running Load Tests

```bash
# Install k6
brew install k6  # macOS
# or
sudo apt-get install k6  # Linux

# Run load test
k6 run load-test.js
```

### Load Test Script

```javascript
import http from 'k6/http';
import { check } from 'k6';

export let options = {
  stages: [
    { duration: '1m', target: 10 },   // Ramp up
    { duration: '5m', target: 100 },  // Steady state
    { duration: '1m', target: 0 },    // Ramp down
  ],
};

export default function() {
  let response = http.post('http://localhost:8080/api/v1/execute',
    JSON.stringify({
      task_id: `task-${__VU}-${__ITER}`,
      user_message: 'Hello',
      model_config: {
        provider: 'anthropic',
        model: 'claude-3-5-sonnet-20241022'
      }
    }),
    { headers: { 'Content-Type': 'application/json' } }
  );

  check(response, {
    'status is 200': (r) => r.status === 200,
    'response time < 500ms': (r) => r.timings.duration < 500,
  });
}
```

## Debugging Tests

### Enable Debug Logging

```go
import "go.temporal.io/sdk/log"

env := testSuite.NewTestWorkflowEnvironment()
env.SetLogger(log.NewDefaultLogger())
```

### Temporal UI

When running integration tests with real Temporal:

1. Open http://localhost:8080
2. Find your workflow execution
3. View execution history
4. Inspect event details
5. Debug failures

### Verbose Output

```bash
go test -v -run TestAgentExecutionWorkflow
```

### Test Specific Iterations

```bash
go test -v -run TestAgentExecutionWorkflow_Success/iteration_3
```

## Best Practices

### 1. Test Naming

Use descriptive test names:

```go
// Good
func TestAgentExecutionWorkflow_WithToolCalls_Success(t *testing.T)

// Bad
func TestWorkflow1(t *testing.T)
```

### 2. Table-Driven Tests

```go
func TestMessageConversion(t *testing.T) {
    tests := []struct {
        name     string
        input    []models.Message
        expected interface{}
    }{
        {"simple user message", ...},
        {"with tool calls", ...},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Test implementation
        })
    }
}
```

### 3. Test Isolation

Each test should be independent:

```go
func TestSomething(t *testing.T) {
    // Create fresh environment
    env := testSuite.NewTestWorkflowEnvironment()

    // Don't rely on global state
    // Clean up after test
    defer cleanup()
}
```

### 4. Meaningful Assertions

```go
// Good
require.Equal(t, models.TaskStatusCompleted, response.Status)
require.Greater(t, response.TokenUsage.TotalTokens, 0)

// Bad
require.NotNil(t, response)
```

### 5. Test Coverage

Aim for high coverage but focus on:
- Critical paths (workflow execution)
- Error handling
- Edge cases
- Boundary conditions

## Troubleshooting

### Test Failures

**Temporal connection errors:**
```bash
# Start Temporal
cd deployments/temporal-executor
docker-compose up -d
```

**API rate limits:**
```go
// Use mocks for unit tests
env.OnActivity(activities.InvokeLLMActivity, ...).Return(mockResponse, nil)
```

**Flaky tests:**
```go
// Add retries for integration tests
require.Eventually(t, func() bool {
    result, err := executorService.Execute(ctx, request)
    return err == nil && result.Status == models.TaskStatusCompleted
}, 30*time.Second, 1*time.Second)
```

## Resources

- [Temporal Testing Documentation](https://docs.temporal.io/docs/go/testing)
- [Go Testing Guide](https://golang.org/doc/code.html#Testing)
- [testify Documentation](https://github.com/stretchr/testify)
- [Benchmark Guide](https://dave.cheney.net/2013/06/30/how-to-write-benchmarks-in-go)

## Contributing Tests

When adding new features:

1. Write unit tests first (TDD)
2. Achieve >80% coverage
3. Add integration test if applicable
4. Add benchmark if performance-critical
5. Update this documentation
