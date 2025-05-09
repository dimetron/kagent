export GIT_LFS_SKIP_SMUDGE=1

VIRTUAL_ENV ?= .venv

### Build and Test

.PHONY: update
update:
	uv sync --all-extras --all-groups
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
