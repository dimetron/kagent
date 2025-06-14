VERSION ?= dev
GIT_COMMIT := $(shell git rev-parse --short HEAD || echo "unknown")
BUILD_DATE := $(shell date -u '+%Y-%m-%d')

LDFLAGS := -X github.com/kagent-dev/kagent/go/cli/internal/cli.Version=$(VERSION) \
           -X github.com/kagent-dev/kagent/go/cli/internal/cli.GitCommit=$(GIT_COMMIT) \
           -X github.com/kagent-dev/kagent/go/cli/internal/cli.BuildDate=$(BUILD_DATE)

.PHONY: build
build:
	go build -ldflags "$(LDFLAGS)" -o bin/kagent ./cmd/kagent

.PHONY: install
install:
	go install -ldflags "$(LDFLAGS)" ./cmd/kagent

.PHONY: clean
clean:
	rm -rf bin/kagent*

.PHONY: test
test:
	go test ./...

.PHONY: deps
deps:
	go mod download
	go mod tidy

.DEFAULT_GOAL := build