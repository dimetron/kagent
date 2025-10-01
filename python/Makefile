export GIT_LFS_SKIP_SMUDGE=1

### Build and Test

.PHONY: help
help:
	@echo "Available test targets:"
	@echo "  make test                  - Run all tests (with update)"
	@echo "  make test-quick            - Run tests excluding integration tests"
	@echo "  make test-unit             - Run unit tests only"
	@echo "  make test-contract         - Run contract tests only"
	@echo "  make test-integration      - Run integration tests only"
	@echo "  make test-ollama           - Run Ollama contract and unit tests"
	@echo "  make test-ollama-integration - Run Ollama integration tests (requires Ollama)"
	@echo "  make test-ollama-all       - Run all Ollama tests"
	@echo ""
	@echo "Other targets:"
	@echo "  make update                - Update dependencies"
	@echo "  make format                - Format and lint code"
	@echo "  make lint                  - Check formatting"
	@echo "  make build                 - Build (update + format)"
	@echo "  make audit                 - Run security audit"

.PHONY: update
update:
	uv python list | grep 3.12
	uv sync --all-extras --all-groups
	uv lock

.PHONY: format
format:
	uv run ruff format
	uv run ruff check

.PHONY: lint
lint: 
	uv run ruff format --diff

.PHONY: test
test: update
	uv run pytest tests

.PHONY: test-quick
test-quick:
	uv run pytest tests -m "not integration" -v

.PHONY: test-unit
test-unit:
	uv run pytest packages/kagent-adk/tests/unit -v

.PHONY: test-contract
test-contract:
	uv run pytest packages/kagent-adk/tests/contract -v

.PHONY: test-integration
test-integration:
	uv run pytest packages/kagent-adk/tests/integration -v -m integration

.PHONY: test-ollama
test-ollama:
	@echo "Running Ollama tests (requires Ollama server running with gpt-oss:latest model)"
	uv run pytest packages/kagent-adk/tests/contract/test_ollama_*.py -v
	uv run pytest packages/kagent-adk/tests/unit/test_ollama_*.py -v

.PHONY: test-ollama-integration
test-ollama-integration:
	@echo "Running Ollama integration tests (requires Ollama server running with gpt-oss:latest model)"
	uv run pytest packages/kagent-adk/tests/integration/test_ollama_*.py -v -m integration

.PHONY: test-ollama-all
test-ollama-all: test-ollama test-ollama-integration
	@echo "All Ollama tests completed"

.PHONY: build
build: update format

.PHONY: audit
audit: build
	#check pip-audit is installed
	uv run pip-audit --version || uv pip install pip-audit
	uv run pip-audit

.PHONY: basic-langchain-sample
basic-langchain-sample:
	docker build . -f samples/langgraph/currency/Dockerfile --tag localhost:5001/langgraph-currency:latest --push

.PHONY: research-crew-sample
research-crew-sample:
	docker build . -f samples/crewai/research-crew/Dockerfile --tag localhost:5001/research-crew:latest --push

.PHONY: poem-flow-sample
poem-flow-sample:
	docker build . -f samples/crewai/poem_flow/Dockerfile --tag localhost:5001/poem-flow:latest --push