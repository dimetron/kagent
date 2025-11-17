package llm

import (
	"context"

	"github.com/kagent-dev/kagent/go/pkg/adk/converters"
)

// Client defines the interface for LLM clients
type Client interface {
	// Generate sends a message and receives a response
	Generate(ctx context.Context, messages []*converters.Content, config *GenerateConfig) (*Response, error)

	// GenerateStream sends a message and streams responses
	GenerateStream(ctx context.Context, messages []*converters.Content, config *GenerateConfig) (<-chan *StreamEvent, error)

	// SupportsTools returns whether this client supports tool calling
	SupportsTools() bool

	// ModelName returns the name of the model being used
	ModelName() string
}

// GenerateConfig contains configuration for generation
type GenerateConfig struct {
	Temperature   *float64               `json:"temperature,omitempty"`
	MaxTokens     *int                   `json:"max_tokens,omitempty"`
	TopP          *float64               `json:"top_p,omitempty"`
	TopK          *int                   `json:"top_k,omitempty"`
	StopSequences []string               `json:"stop_sequences,omitempty"`
	Tools         []ToolDefinition       `json:"tools,omitempty"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
}

// Response represents an LLM response
type Response struct {
	Content      *converters.Content `json:"content"`
	StopReason   string              `json:"stop_reason,omitempty"`
	Usage        *Usage              `json:"usage,omitempty"`
	ToolCalls    []ToolCall          `json:"tool_calls,omitempty"`
	FinishReason string              `json:"finish_reason,omitempty"`
}

// StreamEvent represents a streaming response event
type StreamEvent struct {
	Type    StreamEventType     `json:"type"`
	Content *converters.Content `json:"content,omitempty"`
	Delta   *ContentDelta       `json:"delta,omitempty"`
	Error   error               `json:"error,omitempty"`
	Done    bool                `json:"done"`
}

// StreamEventType represents the type of streaming event
type StreamEventType string

const (
	StreamEventTypeStart   StreamEventType = "start"
	StreamEventTypeDelta   StreamEventType = "delta"
	StreamEventTypeContent StreamEventType = "content"
	StreamEventTypeError   StreamEventType = "error"
	StreamEventTypeDone    StreamEventType = "done"
)

// ContentDelta represents incremental content updates
type ContentDelta struct {
	Text      string     `json:"text,omitempty"`
	ToolCall  *ToolCall  `json:"tool_call,omitempty"`
	StopDelta *StopDelta `json:"stop_delta,omitempty"`
}

// StopDelta represents information about why streaming stopped
type StopDelta struct {
	StopReason   string `json:"stop_reason,omitempty"`
	FinishReason string `json:"finish_reason,omitempty"`
}

// Usage represents token usage information
type Usage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
	TotalTokens  int `json:"total_tokens"`
}

// ToolDefinition defines a tool that can be called by the LLM
type ToolDefinition struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"` // JSON Schema
}

// ToolCall represents a tool call made by the LLM
type ToolCall struct {
	ID        string                 `json:"id"`
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments"`
}

// ConvertToolsFromADK converts ADK tools to LLM tool definitions
func ConvertToolsFromADK(tools []interface{}) []ToolDefinition {
	// This would convert from ADK tool format to LLM tool definitions
	// For now, returning empty slice
	return []ToolDefinition{}
}
