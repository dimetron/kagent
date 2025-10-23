# KAgent ADK Package

**Agent Development Kit (ADK) Integration for KAgent**

This package provides the Python runtime engine for KAgent agents using Google's Agent Development Kit (ADK).

## Features

- **Declarative Agents**: Define AI agents with natural language system prompts
- **Workflow Agents**: Orchestrate multiple sub-agents in sequential or parallel patterns
- **MCP Integration**: Connect to Model Context Protocol (MCP) tool servers
- **Memory Management**: Built-in support for conversation history and vector memory
- **State Management**: Track and share outputs between workflow agents
- **OpenTelemetry**: Automatic instrumentation for observability

## Installation

```bash
# Install with uv (recommended)
uv sync --all-extras

# Or install with pip
pip install -e .
```

## Quick Start

### Running a Simple Agent

```bash
# Start the ADK agent server
uv run python -m kagent.adk.cli \
  --namespace default \
  --name my-agent \
  --model-provider openai \
  --model-name gpt-4
```

### Environment Variables

```bash
# Required: LLM provider API keys
export OPENAI_API_KEY="your-key-here"
export ANTHROPIC_API_KEY="your-key-here"  # For Claude models

# Optional: Observability
export OTEL_EXPORTER_OTLP_ENDPOINT="http://localhost:4317"
export OTEL_SERVICE_NAME="kagent-adk"
```

## Workflow Patterns

KAgent ADK supports two main workflow patterns:

### 1. Sequential Workflows

Execute agents one-at-a-time, where each agent can access outputs from previous agents:

```yaml
apiVersion: kagent.dev/v1alpha2
kind: Agent
metadata:
  name: sequential-workflow
spec:
  type: Workflow
  workflow:
    sequential:
      subAgents:
        - name: validator
          outputKey: "validation_result"
        
        - name: processor
          outputKey: "processed_data"
          # Can access: session.state["validation_result"]
        
        - name: reporter
          # Can access: session.state["validation_result"]
          #              session.state["processed_data"]
```

**Use Cases**: Pipelines, multi-step processing, dependent tasks

### 2. Parallel Workflows

Execute agents concurrently, collecting outputs for later aggregation:

```yaml
apiVersion: kagent.dev/v1alpha2
kind: Agent
metadata:
  name: parallel-workflow
spec:
  type: Workflow
  workflow:
    parallel:
      maxWorkers: 3
      subAgents:
        - name: cluster-east
          outputKey: "east_metrics"
        
        - name: cluster-west
          outputKey: "west_metrics"
        
        - name: cluster-central
          outputKey: "central_metrics"
```

**Key Features**:
- All agents run simultaneously
- Thread-safe concurrent writes
- Outputs available after all complete
- Completion order tracking for debugging

**Use Cases**: Data collection, fan-out operations, parallel processing

### 3. Hybrid Workflows (Sequential + Parallel)

Combine patterns for complex multi-stage workflows:

```yaml
apiVersion: kagent.dev/v1alpha2
kind: Agent
metadata:
  name: hybrid-workflow
spec:
  type: Workflow
  workflow:
    sequential:
      subAgents:
        # Phase 1: Sequential validation
        - name: config-validator
          outputKey: "validation_result"
        
        # Phase 2: Parallel deployment (nested workflow)
        - name: parallel-deployer
          # All deployers access session.state["validation_result"]
        
        # Phase 3: Sequential verification
        - name: deployment-verifier
          # Accesses all parallel deployment outputs
```

**Use Cases**: Validation → parallel execution → aggregation

## Parallel Workflows with OutputKey

### Overview

Parallel workflows enable concurrent execution of multiple sub-agents, each storing its output under a unique `outputKey`. This is ideal for:

- **Data Collection**: Query multiple sources concurrently
- **Fan-Out Operations**: Deploy to multiple environments in parallel
- **Independent Tasks**: Run agents that don't depend on each other

### Basic Example

```yaml
# Parallel data collection workflow
apiVersion: kagent.dev/v1alpha2
kind: Agent
metadata:
  name: cluster-monitoring-workflow
spec:
  type: Workflow
  description: "Query 3 clusters concurrently"
  workflow:
    parallel:
      maxWorkers: 3
      timeout: "2m"
      subAgents:
        - name: east-cluster-agent
          outputKey: "east_metrics"
        
        - name: west-cluster-agent
          outputKey: "west_metrics"
        
        - name: central-cluster-agent
          outputKey: "central_metrics"
```

After execution, workflow state contains:
```json
{
  "east_metrics": "...",
  "west_metrics": "...",
  "central_metrics": "..."
}
```

### Accessing Parallel Outputs

Create an aggregator agent to process all parallel outputs:

```yaml
# Aggregator agent
apiVersion: kagent.dev/v1alpha2
kind: Agent
metadata:
  name: metrics-aggregator
spec:
  type: Declarative
  declarative:
    modelConfig: default-model-config
    systemMessage: |
      You will receive metrics from 3 clusters via session.state:
      - session.state["east_metrics"]
      - session.state["west_metrics"]
      - session.state["central_metrics"]
      
      Aggregate the metrics and generate a summary report.
```

Combine them in a hybrid workflow:

```yaml
# Complete workflow: parallel collection → sequential aggregation
apiVersion: kagent.dev/v1alpha2
kind: Agent
metadata:
  name: monitoring-pipeline
spec:
  type: Workflow
  workflow:
    sequential:
      subAgents:
        - name: cluster-monitoring-workflow
          # Parallel data collection
        
        - name: metrics-aggregator
          # Sequential aggregation
```

### Thread Safety

Parallel workflows use `asyncio.Lock` to ensure thread-safe concurrent writes:

- **No data loss** when agents complete simultaneously
- **Atomic writes** - all-or-nothing guarantees
- **Negligible overhead** - < 0.001% performance impact

### Completion Order Tracking

The `completion_order` field tracks which agent finished first:

```json
{
  "sub_agent_executions": [
    {"agent_name": "west-cluster-agent", "completion_order": 1},
    {"agent_name": "east-cluster-agent", "completion_order": 2},
    {"agent_name": "central-cluster-agent", "completion_order": 3}
  ]
}
```

**Use Cases**:
- **Debugging**: Identify slow agents
- **Optimization**: Reorder agents based on typical patterns
- **Metrics**: Track performance trends

### Automatic OutputKey Naming

**New in v0.6**: `outputKey` is now optional - keys are automatically generated from agent namespace and name!

#### When to Use Automatic vs Manual

**Use Automatic Naming** (recommended):
- ✅ Standard parallel data collection
- ✅ When agent names are descriptive
- ✅ When you want to reduce configuration
- ✅ When working with many parallel agents

**Use Manual Naming**:
- ⚠️ When you need specific key formats
- ⚠️ When working with external systems expecting specific keys
- ⚠️ When agent names are not descriptive enough

#### Automatic Naming Pattern

```
{namespace}_{agent-name}
```

Hyphens in names/namespaces are converted to underscores:
- `production` + `east-collector` → `production_east_collector`
- `staging-env` + `west-us-2` → `staging_env_west_us_2`

#### Example: Automatic OutputKeys

```yaml
apiVersion: kagent.dev/v1alpha2
kind: Agent
metadata:
  name: parallel-workflow
spec:
  type: Workflow
  workflow:
    parallel:
      maxWorkers: 3
      subAgents:
        - name: east-collector
          namespace: production
          # Auto-generated: production_east_collector
        
        - name: west-collector
          namespace: production
          # Auto-generated: production_west_collector
        
        - name: central-collector
          namespace: staging
          # Auto-generated: staging_central_collector
```

**Aggregator agent can access**:
```yaml
systemMessage: |
  Aggregate metrics from:
  - session.state["production_east_collector"]
  - session.state["production_west_collector"]
  - session.state["staging_central_collector"]
```

#### Mixed Mode (Automatic + Manual)

You can mix automatic and manual naming:

```yaml
subAgents:
  - name: east-collector
    namespace: production
    # Auto: production_east_collector
  
  - name: west-collector
    namespace: production
    outputKey: "custom_west_key"  # Manual override
  
  - name: central-collector
    namespace: staging
    # Auto: staging_central_collector
```

#### Validation Rules

Automatic outputKeys are validated to ensure:
- ✅ Max 100 characters
- ✅ Only alphanumeric and underscores (`^[a-zA-Z0-9_]+$`)
- ✅ Unique within workflow
- ⚠️ If auto-generated key is invalid (e.g., too long), you must provide explicit `outputKey`

### Best Practices

1. **Use Automatic Naming (Recommended)**
   ```yaml
   # ✅ GOOD - Let system auto-generate
   subAgents:
     - name: prod-east-metrics
       namespace: production
     - name: prod-west-metrics
       namespace: production
   
   # ❌ AVOID - Manual unless necessary
   subAgents:
     - name: agent-1
       outputKey: "prod_east_metrics"
   ```

2. **Use Descriptive Agent Names**
   ```yaml
   # ✅ GOOD - Name describes purpose
   name: cluster-health-checker
   name: deployment-validator
   
   # ❌ BAD - Generic names
   name: agent-1
   name: worker-2
   ```

3. **Set Appropriate maxWorkers**
   ```yaml
   # For 3 agents - all run concurrently
   maxWorkers: 3
   
   # For 20 agents - run 10 at a time (resource-limited)
   maxWorkers: 10
   ```

4. **Handle Partial Success**
   ```yaml
   systemMessage: |
     You may receive data from UP TO N sources.
     Some may be missing if they failed or timed out.
     Generate report based on AVAILABLE data only.
   ```

5. **Monitor Performance**
   ```promql
   # Average lock wait time (should be < 1ms)
   rate(kagent_parallel_workflow_lock_wait_seconds_sum[5m]) / 
   rate(kagent_parallel_workflow_lock_wait_seconds_count[5m])
   ```

## Advanced Features

### Memory Integration

```python
from kagent.adk import create_memory_store

# Create vector memory for long-term context
memory = create_memory_store(
    provider="qdrant",
    collection_name="agent-memory",
    embedding_model="text-embedding-3-small"
)
```

### Custom Tool Integration

```python
from kagent.adk import register_tool

@register_tool
async def custom_tool(query: str) -> str:
    """Custom tool implementation."""
    return f"Processed: {query}"
```

### OpenTelemetry Tracing

Automatic instrumentation for parallel workflows:

```python
# Spans automatically created:
# - parallel.execute_with_output_key
# - parallel.sub_agent_execution
# - parallel.concurrent_write
# - parallel.state_injection
```

Attributes tracked:
- `workflow.session_id`
- `workflow.agent_count`
- `workflow.use_output_key_mode`
- `workflow.outputs_collected`
- `workflow.completion_order`

## Testing

```bash
# Run all tests
uv run pytest -v

# Run specific test suite
uv run pytest tests/unit/ -v
uv run pytest tests/integration/ -v
uv run pytest tests/e2e/ -v

# Run with coverage
uv run pytest --cov=kagent.adk --cov-report=html

# Run stress tests (slow)
uv run pytest -v -m slow
```

## Development

```bash
# Format code
uv run ruff format

# Lint code
uv run ruff check

# Type check
uv run mypy src/

# Run pre-commit hooks
pre-commit run --all-files
```

## Documentation

- **Quickstart**: [specs/005-add-output-key/quickstart.md](../../../specs/005-add-output-key/quickstart.md)
- **Architecture**: [specs/005-add-output-key/architecture.md](../../../specs/005-add-output-key/architecture.md)
- **Data Model**: [specs/005-add-output-key/data-model.md](../../../specs/005-add-output-key/data-model.md)
- **API Contracts**: [specs/005-add-output-key/contracts/](../../../specs/005-add-output-key/contracts/)

## Examples

See complete examples in:
- `samples/` - Example agent configurations
- `specs/005-add-output-key/quickstart.md` - Step-by-step tutorials
- `tests/e2e/` - End-to-end test scenarios

## Troubleshooting

### Parallel agents not executing concurrently

**Cause**: `maxWorkers` set to 1

**Solution**: Increase `maxWorkers` to match number of sub-agents:
```yaml
parallel:
  maxWorkers: 10  # Allow up to 10 concurrent agents
```

### Outputs missing after parallel execution

**Cause**: Agents failed or timed out

**Solution**: Check logs and increase timeout:
```yaml
parallel:
  timeout: "5m"  # Increase from default 2m
```

### High lock wait times

**Cause**: Too many agents completing simultaneously

**Solution**: This is expected and has negligible impact. If concerned, reduce `maxWorkers`.

## License

Apache 2.0

## Contributing

See [CONTRIBUTION.md](../../../CONTRIBUTION.md) for guidelines.

