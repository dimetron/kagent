# KAgentTaskStore Tests

This directory contains unit tests for the `KAgentTaskStore` class.

## Running Tests

To run the tests for kagent-core:

```bash
# From the kagent-core package directory
cd python/packages/kagent-core

# Install test dependencies
uv pip install -e ".[test]"

# Run all tests
pytest

# Run with coverage
pytest --cov=src/kagent/core/a2a --cov-report=term-missing

# Run specific test file
pytest tests/unit/test_task_store.py

# Run specific test class
pytest tests/unit/test_task_store.py::TestKAgentTaskStoreSave

# Run specific test
pytest tests/unit/test_task_store.py::TestKAgentTaskStoreSave::test_save_task_success

# Run with verbose output
pytest -v

# Run and stop on first failure
pytest -x
```

## Test Coverage

The test suite covers:

### 1. Initialization Tests (`TestKAgentTaskStoreInit`)
- Creating task store with httpx client
- Client type validation

### 2. Save Method Tests (`TestKAgentTaskStoreSave`)
- Successfully saving a task
- Saving with call_context parameter
- Handling HTTP errors

### 3. Get Method Tests (`TestKAgentTaskStoreGet`)
- Successfully retrieving a task
- Retrieving with call_context parameter
- Handling 404 (returns None)
- Handling HTTP errors

### 4. Delete Method Tests (`TestKAgentTaskStoreDelete`)
- Successfully deleting a task
- Deleting with call_context parameter
- Handling HTTP errors

### 5. Integration Tests (`TestKAgentTaskStoreIntegration`)
- Save and get roundtrip
- Complete lifecycle (save, get, delete)

## Test Fixtures

The `conftest.py` provides the following fixtures:

- `mock_http_client`: Mock httpx.AsyncClient for testing
- `sample_task_id`: Sample task ID
- `sample_task_data`: Sample task data dictionary

## Implementation Details

All methods in `KAgentTaskStore` accept an optional `call_context` parameter to maintain compatibility with the a2a library's TaskStore interface. This parameter is currently unused in the implementation but must be accepted to match the base class signature.

### Method Signatures

```python
async def save(self, task: Task, call_context: Any = None) -> None
async def get(self, task_id: str, call_context: Any = None) -> Task | None
async def delete(self, task_id: str, call_context: Any = None) -> None
```

## Related Files

- Implementation: `src/kagent/core/a2a/_task_store.py`
- Tests: `tests/unit/test_task_store.py`
- Fixtures: `tests/conftest.py`

