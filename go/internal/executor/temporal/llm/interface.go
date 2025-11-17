package llm

import (
	"context"

	"github.com/kagent-dev/kagent/go/internal/executor/temporal/models"
)

// Provider is an interface for LLM providers
type Provider interface {
	// Chat sends a chat request to the LLM and returns the response
	Chat(ctx context.Context, request models.LLMRequest) (*models.LLMResponse, error)

	// ChatStream sends a chat request and streams the response
	ChatStream(ctx context.Context, request models.LLMRequest) (<-chan StreamChunk, <-chan error)

	// Name returns the provider name
	Name() string

	// SupportedModels returns a list of supported model names
	SupportedModels() []string
}

// StreamChunk represents a chunk of streaming response
type StreamChunk struct {
	Content      string
	ToolCalls    []models.ToolCall
	FinishReason string
	Delta        bool // true if this is a delta update, false if complete
}

// ProviderRegistry manages LLM providers
type ProviderRegistry interface {
	// Register registers a provider
	Register(provider Provider) error

	// Get retrieves a provider by name
	Get(name string) (Provider, error)

	// List returns all registered provider names
	List() []string
}
