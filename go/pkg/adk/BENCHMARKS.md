# ADK Performance Benchmarks

This document describes the performance benchmarks for the Go ADK package and how to run them.

## Overview

The ADK includes comprehensive benchmarks for:
- **Converters**: A2A ↔ ADK protocol conversion
- **Executors**: Agent execution and tool calling
- **LLM Clients**: Message conversion and configuration

## Running Benchmarks

### Run All Benchmarks

```bash
cd go/pkg/adk
go test -bench=. -benchmem ./...
```

### Run Specific Package Benchmarks

```bash
# Converter benchmarks
go test -bench=. -benchmem ./converters

# Executor benchmarks
go test -bench=. -benchmem ./executor

# LLM client benchmarks
go test -bench=. -benchmem ./llm
```

### Run Specific Benchmarks

```bash
# Part conversion benchmarks
go test -bench=BenchmarkPartConverter -benchmem ./converters

# Tool execution benchmarks
go test -bench=BenchmarkExecutor_ExecuteTool -benchmem ./executor

# Message conversion benchmarks
go test -bench=BenchmarkOpenAI_ConvertMessages -benchmem ./llm
```

### Benchmark Options

```bash
# Run benchmarks for longer duration (default is 1 second)
go test -bench=. -benchtime=5s ./converters

# Run benchmarks with CPU profiling
go test -bench=. -cpuprofile=cpu.prof ./converters

# Run benchmarks with memory profiling
go test -bench=. -memprofile=mem.prof ./converters

# Specify benchmark iterations
go test -bench=. -benchtime=10000x ./converters

# Run parallel benchmarks only
go test -bench=Parallel -benchmem ./converters
```

## Benchmark Categories

### Converter Benchmarks

Located in `converters/benchmarks_test.go`:

#### Part Conversion (A2A → ADK)
- `BenchmarkPartConverter_A2AToContent_Text` - Single text part
- `BenchmarkPartConverter_A2AToContent_MultipleParts` - Multiple parts
- `BenchmarkPartConverter_A2AToContent_FunctionCall` - Function call conversion

#### Part Conversion (ADK → A2A)
- `BenchmarkPartConverter_ContentToA2A_Text` - Single text part
- `BenchmarkPartConverter_ContentToA2A_MultipleParts` - Multiple parts

#### Round-Trip Conversion
- `BenchmarkPartConverter_RoundTrip` - Complete A2A → ADK → A2A conversion

#### Request Conversion
- `BenchmarkRequestConverter_Convert` - Basic request conversion
- `BenchmarkRequestConverter_Convert_WithHistory` - With message history

#### Event Conversion
- `BenchmarkEventConverter_Convert_Content` - Content event conversion
- `BenchmarkEventConverter_Convert_Error` - Error event conversion
- `BenchmarkEventConverter_Convert_StateUpdate` - State update conversion

#### Parallel Benchmarks
- `BenchmarkPartConverter_A2AToContent_Parallel` - Concurrent part conversion
- `BenchmarkEventConverter_Convert_Parallel` - Concurrent event conversion

### Executor Benchmarks

Located in `executor/benchmarks_test.go`:

#### Tool Execution
- `BenchmarkExecutor_ExecuteTool` - Single tool execution
- `BenchmarkExecutorV2_ExecuteToolCalls_Single` - V2 single tool call
- `BenchmarkExecutorV2_ExecuteToolCalls_Multiple` - V2 multiple tool calls

#### Tool Definition Building
- `BenchmarkExecutorV2_BuildToolDefinitions` - Build tool definitions

#### Message Building
- `BenchmarkExecutorV2_BuildMessageHistory` - Build message history

#### Session Operations
- `BenchmarkExecutorV2_GetOrCreateSession` - Session creation/retrieval

#### Full Execution Flow
- `BenchmarkExecutor_Execute_Echo` - Complete echo execution

#### Parallel Benchmarks
- `BenchmarkExecutor_ExecuteTool_Parallel` - Concurrent tool execution
- `BenchmarkExecutorV2_BuildToolDefinitions_Parallel` - Concurrent definition building

#### Memory Allocation Benchmarks
- `BenchmarkExecutorV2_BuildMessageHistory_Allocs` - Message building allocations
- `BenchmarkExecutor_Execute_Echo_Allocs` - Echo execution allocations

### LLM Client Benchmarks

Located in `llm/benchmarks_test.go`:

#### OpenAI Message Conversion
- `BenchmarkOpenAI_ConvertMessages_Single` - Single message
- `BenchmarkOpenAI_ConvertMessages_Conversation` - Multi-turn conversation
- `BenchmarkOpenAI_ConvertMessages_WithFunctionCall` - With function calling

#### Anthropic Message Conversion
- `BenchmarkAnthropic_ConvertMessages_Single` - Single message
- `BenchmarkAnthropic_ConvertMessages_Conversation` - Multi-turn conversation

#### Gemini Message Conversion
- `BenchmarkGemini_ConvertMessages_Single` - Single message
- `BenchmarkGemini_ConvertMessages_WithFunctionCall` - With function calling

#### Gemini Schema Conversion
- `BenchmarkGemini_ConvertSchema_Simple` - Simple schema
- `BenchmarkGemini_ConvertSchema_Complex` - Complex nested schema

#### Tool Definition Building
- `BenchmarkBuildToolDefinitions_Single` - Single tool
- `BenchmarkBuildToolDefinitions_Multiple` - Multiple tools

#### Parallel Benchmarks
- `BenchmarkOpenAI_ConvertMessages_Parallel` - Concurrent OpenAI conversion
- `BenchmarkAnthropic_ConvertMessages_Parallel` - Concurrent Anthropic conversion
- `BenchmarkGemini_ConvertMessages_Parallel` - Concurrent Gemini conversion

## Understanding Benchmark Output

Example output:
```
BenchmarkPartConverter_A2AToContent_Text-8    1000000    1234 ns/op    512 B/op    8 allocs/op
```

- `BenchmarkPartConverter_A2AToContent_Text` - Benchmark name
- `-8` - Number of CPU cores used (GOMAXPROCS)
- `1000000` - Number of iterations run
- `1234 ns/op` - Nanoseconds per operation
- `512 B/op` - Bytes allocated per operation
- `8 allocs/op` - Number of allocations per operation

## Performance Targets

### Converters
- Part conversion: < 1000 ns/op
- Request conversion: < 5000 ns/op
- Event conversion: < 5000 ns/op

### Executors
- Tool execution: < 10 µs/op (without actual tool logic)
- Message building: < 5000 ns/op
- Session operations: < 1000 ns/op

### LLM Clients
- Message conversion: < 5000 ns/op per message
- Schema conversion: < 2000 ns/op

## Profiling

### CPU Profiling

```bash
go test -bench=BenchmarkPartConverter -cpuprofile=cpu.prof ./converters
go tool pprof cpu.prof
```

In pprof:
```
(pprof) top10
(pprof) list FunctionName
(pprof) web
```

### Memory Profiling

```bash
go test -bench=BenchmarkPartConverter -memprofile=mem.prof ./converters
go tool pprof mem.prof
```

In pprof:
```
(pprof) top10
(pprof) list FunctionName
(pprof) alloc_space  # Total allocations
(pprof) inuse_space  # Current memory usage
```

### Trace Analysis

```bash
go test -bench=BenchmarkPartConverter -trace=trace.out ./converters
go tool trace trace.out
```

## Comparing Benchmarks

### Using benchstat

Install benchstat:
```bash
go install golang.org/x/perf/cmd/benchstat@latest
```

Run benchmarks and save results:
```bash
go test -bench=. -benchmem ./converters > old.txt
# Make changes
go test -bench=. -benchmem ./converters > new.txt

benchstat old.txt new.txt
```

### Continuous Benchmarking

For CI/CD integration:

```bash
# Run benchmarks and output JSON
go test -bench=. -benchmem -json ./... > benchmark-results.json

# Compare with baseline
benchstat baseline.txt benchmark-results.json
```

## Optimization Guidelines

### Reduce Allocations
- Use sync.Pool for frequently allocated objects
- Pre-allocate slices with known capacity
- Avoid unnecessary string conversions
- Reuse buffers

### Improve CPU Performance
- Minimize interface conversions
- Avoid reflection in hot paths
- Use efficient data structures
- Consider caching expensive computations

### Parallel Processing
- Ensure goroutine-safe operations
- Minimize lock contention
- Use channels appropriately
- Consider work-stealing patterns

## Best Practices

1. **Run benchmarks multiple times** to ensure stable results
2. **Disable CPU frequency scaling** for consistent results
3. **Close unnecessary applications** to reduce noise
4. **Use -benchmem** to track memory allocations
5. **Profile before optimizing** to find actual bottlenecks
6. **Compare with baseline** when making optimizations
7. **Document performance changes** in commit messages

## Common Issues

### Benchmark Variation

If benchmarks show high variation:
- Close other applications
- Disable CPU frequency scaling
- Increase benchmark time: `-benchtime=10s`
- Run on dedicated hardware

### Memory Leaks

If memory usage increases:
- Check for goroutine leaks
- Ensure proper cleanup
- Use `-memprofile` to identify leaks
- Run with race detector: `-race`

## CI/CD Integration

Example GitHub Actions workflow:

```yaml
name: Benchmarks

on:
  pull_request:
    branches: [ main ]

jobs:
  benchmark:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v2

      - name: Set up Go
        uses: actions/setup-go@v2
        with:
          go-version: 1.22

      - name: Run benchmarks
        run: |
          cd go/pkg/adk
          go test -bench=. -benchmem ./... > benchmark.txt

      - name: Upload results
        uses: actions/upload-artifact@v2
        with:
          name: benchmark-results
          path: benchmark.txt
```

## References

- [Go Benchmarking](https://go.dev/doc/diagnostics#profiling)
- [Go pprof](https://go.dev/blog/pprof)
- [benchstat](https://pkg.go.dev/golang.org/x/perf/cmd/benchstat)
- [Go Performance Tips](https://github.com/dgryski/go-perfbook)
