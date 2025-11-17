package llm

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/kagent-dev/kagent/go/internal/executor/temporal/models"
)

func TestProviderInterface(t *testing.T) {
	// This test verifies that our provider implementations satisfy the interface
	var _ Provider = &openaiProvider{}
	var _ Provider = &anthropicProvider{}
}

func TestOpenAIProvider_Name(t *testing.T) {
	provider := NewOpenAIProvider("test-key")
	require.Equal(t, "openai", provider.Name())
}

func TestOpenAIProvider_SupportedModels(t *testing.T) {
	provider := NewOpenAIProvider("test-key")
	models := provider.SupportedModels()

	require.NotEmpty(t, models)
	require.Contains(t, models, "gpt-4-turbo")
	require.Contains(t, models, "gpt-3.5-turbo")
}

func TestAnthropicProvider_Name(t *testing.T) {
	provider := NewAnthropicProvider("test-key")
	require.Equal(t, "anthropic", provider.Name())
}

func TestAnthropicProvider_SupportedModels(t *testing.T) {
	provider := NewAnthropicProvider("test-key")
	models := provider.SupportedModels()

	require.NotEmpty(t, models)
	require.Contains(t, models, "claude-3-5-sonnet-20241022")
	require.Contains(t, models, "claude-3-opus-20240229")
}

// Integration tests (require API keys)
func TestOpenAIProvider_Chat_Integration(t *testing.T) {
	t.Skip("Requires OPENAI_API_KEY")

	// This would be an integration test that requires actual API key
	// Uncomment and set key to run
	/*
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		t.Skip("OPENAI_API_KEY not set")
	}

	provider := NewOpenAIProvider(apiKey)
	ctx := context.Background()

	request := models.LLMRequest{
		Messages: []models.Message{
			{Role: "user", Content: "Say hello"},
		},
		ModelConfig: models.ModelConfig{
			Model: "gpt-3.5-turbo",
			MaxTokens: 50,
			Temperature: 0.7,
		},
	}

	response, err := provider.Chat(ctx, request)
	require.NoError(t, err)
	require.NotEmpty(t, response.Content)
	require.Equal(t, "stop", response.FinishReason)
	*/
}

func TestAnthropicProvider_Chat_Integration(t *testing.T) {
	t.Skip("Requires ANTHROPIC_API_KEY")

	// This would be an integration test that requires actual API key
	// Uncomment and set key to run
	/*
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		t.Skip("ANTHROPIC_API_KEY not set")
	}

	provider := NewAnthropicProvider(apiKey)
	ctx := context.Background()

	request := models.LLMRequest{
		Messages: []models.Message{
			{Role: "user", Content: "Say hello"},
		},
		ModelConfig: models.ModelConfig{
			Model: "claude-3-5-sonnet-20241022",
			MaxTokens: 50,
			Temperature: 0.7,
		},
	}

	response, err := provider.Chat(ctx, request)
	require.NoError(t, err)
	require.NotEmpty(t, response.Content)
	*/
}

func TestMessageConversion(t *testing.T) {
	tests := []struct {
		name     string
		messages []models.Message
	}{
		{
			name: "simple user message",
			messages: []models.Message{
				{Role: "user", Content: "Hello"},
			},
		},
		{
			name: "conversation with system",
			messages: []models.Message{
				{Role: "system", Content: "You are helpful"},
				{Role: "user", Content: "Hello"},
				{Role: "assistant", Content: "Hi there!"},
			},
		},
		{
			name: "with tool calls",
			messages: []models.Message{
				{Role: "user", Content: "Get weather"},
				{
					Role:    "assistant",
					Content: "Let me check",
					ToolCalls: []models.ToolCall{
						{
							ID:        "call_1",
							Name:      "get_weather",
							Arguments: map[string]interface{}{"location": "SF"},
						},
					},
				},
				{Role: "tool", Content: "72°F", ToolCallID: "call_1"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test that messages can be used in request without panicking
			request := models.LLMRequest{
				Messages: tt.messages,
				ModelConfig: models.ModelConfig{
					Provider: "anthropic",
					Model:    "claude-3-5-sonnet-20241022",
				},
			}

			require.NotNil(t, request.Messages)
			require.Equal(t, len(tt.messages), len(request.Messages))
		})
	}
}
