
# Makefile for Python project using Hatch and Ruff

export http_proxy=http://genproxy:8080
export UV_PROXY=http://genproxy:8080
export UV_PROJECT= .

export GIT_LFS_SKIP_SMUDGE=1

### Build and Test

.PHONY: update
update:
	uv venv .venv
	uv python install 3.12
	uv sync --all-extras --all-groups
	uv pip install hatchling hatch setuptools
	uv lock

.PHONY: format
format:
	uv run ruff format
	uv run ruff check

.PHONY: test
test: update
	uv run pytest tests

.PHONY: build
build: update format
