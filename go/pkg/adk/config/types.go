package config

import (
	"encoding/json"
	"fmt"

	apperrors "github.com/kagent-dev/kagent/go/pkg/app/errors"
)

// AgentConfig represents the configuration for an agent
type AgentConfig struct {
	Model        ModelConfig            `json:"model"`
	Description  string                 `json:"description,omitempty"`
	Instruction  string                 `json:"instruction,omitempty"`
	HTTPTools    []HTTPMCPServerConfig  `json:"http_tools,omitempty"`
	SSETools     []SSEMCPServerConfig   `json:"sse_tools,omitempty"`
	RemoteAgents []RemoteAgentConfig    `json:"remote_agents,omitempty"`
	ExecuteCode  *bool                  `json:"execute_code,omitempty"`
}

// ModelConfig is an interface for different model configurations
type ModelConfig interface {
	Type() string
	Validate() error
}

// BaseModelConfig contains common fields for all models
type BaseModelConfig struct {
	ModelType string `json:"type"`
}

func (b *BaseModelConfig) Type() string {
	return b.ModelType
}

// OpenAIConfig represents OpenAI model configuration
type OpenAIConfig struct {
	BaseModelConfig
	Model            string   `json:"model"`
	BaseURL          *string  `json:"base_url,omitempty"`
	FrequencyPenalty *float64 `json:"frequency_penalty,omitempty"`
	MaxTokens        *int     `json:"max_tokens,omitempty"`
	Temperature      *float64 `json:"temperature,omitempty"`
	TopP             *float64 `json:"top_p,omitempty"`
	APIKey           *string  `json:"api_key,omitempty"`
}

func (o *OpenAIConfig) Validate() error {
	if o.Model == "" {
		return apperrors.New(apperrors.ErrCodeAgentConfig, "model name is required", nil)
	}
	return nil
}

// AnthropicConfig represents Anthropic model configuration
type AnthropicConfig struct {
	BaseModelConfig
	Model       string   `json:"model"`
	MaxTokens   *int     `json:"max_tokens,omitempty"`
	Temperature *float64 `json:"temperature,omitempty"`
	TopP        *float64 `json:"top_p,omitempty"`
	TopK        *int     `json:"top_k,omitempty"`
	APIKey      *string  `json:"api_key,omitempty"`
}

func (a *AnthropicConfig) Validate() error {
	if a.Model == "" {
		return apperrors.New(apperrors.ErrCodeAgentConfig, "model name is required", nil)
	}
	return nil
}

// GeminiConfig represents Google Gemini model configuration
type GeminiConfig struct {
	BaseModelConfig
	Model       string   `json:"model"`
	MaxTokens   *int     `json:"max_tokens,omitempty"`
	Temperature *float64 `json:"temperature,omitempty"`
	TopP        *float64 `json:"top_p,omitempty"`
	TopK        *int     `json:"top_k,omitempty"`
	APIKey      *string  `json:"api_key,omitempty"`
}

func (g *GeminiConfig) Validate() error {
	if g.Model == "" {
		return apperrors.New(apperrors.ErrCodeAgentConfig, "model name is required", nil)
	}
	return nil
}

// HTTPMCPServerConfig represents HTTP MCP server configuration
type HTTPMCPServerConfig struct {
	Name    string            `json:"name"`
	URL     string            `json:"url"`
	Headers map[string]string `json:"headers,omitempty"`
}

// SSEMCPServerConfig represents SSE MCP server configuration
type SSEMCPServerConfig struct {
	Name    string            `json:"name"`
	Command string            `json:"command"`
	Args    []string          `json:"args,omitempty"`
	Env     map[string]string `json:"env,omitempty"`
}

// RemoteAgentConfig represents remote agent configuration
type RemoteAgentConfig struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

// UnmarshalJSON implements custom unmarshaling for AgentConfig to handle model discriminator
func (a *AgentConfig) UnmarshalJSON(data []byte) error {
	type Alias AgentConfig
	aux := &struct {
		Model json.RawMessage `json:"model"`
		*Alias
	}{
		Alias: (*Alias)(a),
	}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	// Parse model type first
	var modelType struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(aux.Model, &modelType); err != nil {
		return fmt.Errorf("failed to parse model type: %w", err)
	}

	// Unmarshal into appropriate model config type
	switch modelType.Type {
	case "OpenAI":
		var openai OpenAIConfig
		if err := json.Unmarshal(aux.Model, &openai); err != nil {
			return err
		}
		a.Model = &openai
	case "Anthropic":
		var anthropic AnthropicConfig
		if err := json.Unmarshal(aux.Model, &anthropic); err != nil {
			return err
		}
		a.Model = &anthropic
	case "Gemini":
		var gemini GeminiConfig
		if err := json.Unmarshal(aux.Model, &gemini); err != nil {
			return err
		}
		a.Model = &gemini
	default:
		return fmt.Errorf("unsupported model type: %s", modelType.Type)
	}

	return nil
}

// MarshalJSON implements custom marshaling for AgentConfig
func (a *AgentConfig) MarshalJSON() ([]byte, error) {
	type Alias AgentConfig
	return json.Marshal(&struct {
		*Alias
	}{
		Alias: (*Alias)(a),
	})
}
