package llm

import (
	"testing"

	"github.com/kagent-dev/kagent/go/pkg/adk/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewClientFromConfig_OpenAI(t *testing.T) {
	apiKey := "test-api-key"
	cfg := &config.OpenAIConfig{
		BaseModelConfig: config.BaseModelConfig{ModelType: "OpenAI"},
		Model:           "gpt-4",
		APIKey:          &apiKey,
	}

	client, err := NewClientFromConfig(cfg)
	require.NoError(t, err)
	assert.NotNil(t, client)
	assert.Equal(t, "gpt-4", client.ModelName())
	assert.True(t, client.SupportsTools())
}

func TestNewClientFromConfig_Anthropic(t *testing.T) {
	apiKey := "test-api-key"
	cfg := &config.AnthropicConfig{
		BaseModelConfig: config.BaseModelConfig{ModelType: "Anthropic"},
		Model:           "claude-3-5-sonnet-20241022",
		APIKey:          &apiKey,
	}

	client, err := NewClientFromConfig(cfg)
	require.NoError(t, err)
	assert.NotNil(t, client)
	assert.Equal(t, "claude-3-5-sonnet-20241022", client.ModelName())
	assert.True(t, client.SupportsTools())
}

func TestNewClientFromConfig_Gemini(t *testing.T) {
	apiKey := "test-api-key"
	cfg := &config.GeminiConfig{
		BaseModelConfig: config.BaseModelConfig{ModelType: "Gemini"},
		Model:           "gemini-2.0-flash",
		APIKey:          &apiKey,
	}

	client, err := NewClientFromConfig(cfg)
	require.NoError(t, err)
	assert.NotNil(t, client)
	assert.Equal(t, "gemini-2.0-flash", client.ModelName())
	assert.True(t, client.SupportsTools())
}

func TestNewClientFromConfig_NilConfig(t *testing.T) {
	client, err := NewClientFromConfig(nil)
	require.Error(t, err)
	assert.Nil(t, client)
	assert.Contains(t, err.Error(), "config is required")
}

func TestNewClientFromConfig_UnsupportedType(t *testing.T) {
	cfg := &config.BaseModelConfig{ModelType: "UnsupportedModel"}

	client, err := NewClientFromConfig(cfg)
	require.Error(t, err)
	assert.Nil(t, client)
	assert.Contains(t, err.Error(), "unsupported model type")
}

func TestNewClientFromConfig_InvalidOpenAIConfig(t *testing.T) {
	// Pass wrong config type for OpenAI
	cfg := &config.BaseModelConfig{ModelType: "OpenAI"}

	client, err := NewClientFromConfig(cfg)
	require.Error(t, err)
	assert.Nil(t, client)
	assert.Contains(t, err.Error(), "invalid OpenAI config")
}

func TestNewClientFromConfig_InvalidAnthropicConfig(t *testing.T) {
	// Pass wrong config type for Anthropic
	cfg := &config.BaseModelConfig{ModelType: "Anthropic"}

	client, err := NewClientFromConfig(cfg)
	require.Error(t, err)
	assert.Nil(t, client)
	assert.Contains(t, err.Error(), "invalid Anthropic config")
}

func TestNewClientFromConfig_InvalidGeminiConfig(t *testing.T) {
	// Pass wrong config type for Gemini
	cfg := &config.BaseModelConfig{ModelType: "Gemini"}

	client, err := NewClientFromConfig(cfg)
	require.Error(t, err)
	assert.Nil(t, client)
	assert.Contains(t, err.Error(), "invalid Gemini config")
}

func TestNewClientFromConfig_MissingAPIKey_OpenAI(t *testing.T) {
	cfg := &config.OpenAIConfig{
		BaseModelConfig: config.BaseModelConfig{ModelType: "OpenAI"},
		Model:           "gpt-4",
		APIKey:          nil,
	}

	client, err := NewClientFromConfig(cfg)
	require.Error(t, err)
	assert.Nil(t, client)
	assert.Contains(t, err.Error(), "API key is required")
}

func TestNewClientFromConfig_MissingAPIKey_Anthropic(t *testing.T) {
	cfg := &config.AnthropicConfig{
		BaseModelConfig: config.BaseModelConfig{ModelType: "Anthropic"},
		Model:           "claude-3-5-sonnet-20241022",
		APIKey:          nil,
	}

	client, err := NewClientFromConfig(cfg)
	require.Error(t, err)
	assert.Nil(t, client)
	assert.Contains(t, err.Error(), "API key is required")
}

func TestNewClientFromConfig_MissingAPIKey_Gemini(t *testing.T) {
	cfg := &config.GeminiConfig{
		BaseModelConfig: config.BaseModelConfig{ModelType: "Gemini"},
		Model:           "gemini-2.0-flash",
		APIKey:          nil,
	}

	client, err := NewClientFromConfig(cfg)
	require.Error(t, err)
	assert.Nil(t, client)
	assert.Contains(t, err.Error(), "API key is required")
}

func TestNewClientFromConfig_EmptyAPIKey_OpenAI(t *testing.T) {
	emptyKey := ""
	cfg := &config.OpenAIConfig{
		BaseModelConfig: config.BaseModelConfig{ModelType: "OpenAI"},
		Model:           "gpt-4",
		APIKey:          &emptyKey,
	}

	client, err := NewClientFromConfig(cfg)
	require.Error(t, err)
	assert.Nil(t, client)
	assert.Contains(t, err.Error(), "API key is required")
}

func TestNewClientFromConfig_WithConfiguration_OpenAI(t *testing.T) {
	apiKey := "test-api-key"
	temp := 0.7
	maxTokens := 1000

	cfg := &config.OpenAIConfig{
		BaseModelConfig: config.BaseModelConfig{ModelType: "OpenAI"},
		Model:           "gpt-4",
		APIKey:          &apiKey,
		Temperature:     &temp,
		MaxTokens:       &maxTokens,
	}

	client, err := NewClientFromConfig(cfg)
	require.NoError(t, err)
	assert.NotNil(t, client)

	// Verify it's an OpenAI client
	openaiClient, ok := client.(*OpenAIClient)
	assert.True(t, ok)
	assert.NotNil(t, openaiClient.config)
	assert.Equal(t, temp, *openaiClient.config.Temperature)
	assert.Equal(t, maxTokens, *openaiClient.config.MaxTokens)
}

func TestNewClientFromConfig_WithConfiguration_Anthropic(t *testing.T) {
	apiKey := "test-api-key"
	temp := 0.7
	maxTokens := 1000

	cfg := &config.AnthropicConfig{
		BaseModelConfig: config.BaseModelConfig{ModelType: "Anthropic"},
		Model:           "claude-3-5-sonnet-20241022",
		APIKey:          &apiKey,
		Temperature:     &temp,
		MaxTokens:       &maxTokens,
	}

	client, err := NewClientFromConfig(cfg)
	require.NoError(t, err)
	assert.NotNil(t, client)

	// Verify it's an Anthropic client
	anthropicClient, ok := client.(*AnthropicClient)
	assert.True(t, ok)
	assert.NotNil(t, anthropicClient.config)
	assert.Equal(t, temp, *anthropicClient.config.Temperature)
	assert.Equal(t, maxTokens, *anthropicClient.config.MaxTokens)
}

func TestNewClientFromConfig_WithConfiguration_Gemini(t *testing.T) {
	apiKey := "test-api-key"
	temp := 0.7
	maxTokens := 1000

	cfg := &config.GeminiConfig{
		BaseModelConfig: config.BaseModelConfig{ModelType: "Gemini"},
		Model:           "gemini-2.0-flash",
		APIKey:          &apiKey,
		Temperature:     &temp,
		MaxTokens:       &maxTokens,
	}

	client, err := NewClientFromConfig(cfg)
	require.NoError(t, err)
	assert.NotNil(t, client)

	// Verify it's a Gemini client
	geminiClient, ok := client.(*GeminiClient)
	assert.True(t, ok)
	assert.NotNil(t, geminiClient.config)
	assert.Equal(t, temp, *geminiClient.config.Temperature)
	assert.Equal(t, maxTokens, *geminiClient.config.MaxTokens)
}
