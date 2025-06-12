
# Makefile for Python project using Hatch and Ruff

export http_proxy=http://genproxy:8080
export UV_PROXY=http://genproxy:8080
export GIT_LFS_SKIP_SMUDGE=1

### Build and Test

.PHONY: clean
clean:
	rm -rf .venv

.PHONY: update
update:
	uv venv .venv
	uv sync --all-extras --all-groups
	uv pip install hatchling hatch setuptools
	uv lock
	uv pip tree
	uv pip list

.PHONY: format
format:
	uv run ruff format
	uv run ruff check

.PHONY: test
test: update
	uv run pytest tests

.PHONY: build
build: update format
