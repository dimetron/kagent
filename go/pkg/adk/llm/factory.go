package llm

import (
	"fmt"

	"github.com/kagent-dev/kagent/go/pkg/adk/config"
	apperrors "github.com/kagent-dev/kagent/go/pkg/adk/errors"
)

// NewClientFromConfig creates an LLM client from agent configuration
func NewClientFromConfig(cfg config.ModelConfig) (Client, error) {
	if cfg == nil {
		return nil, apperrors.New(apperrors.ErrCodeAgentConfig, "model config is required", nil)
	}

	switch cfg.Type() {
	case "OpenAI":
		openaiCfg, ok := cfg.(*config.OpenAIConfig)
		if !ok {
			return nil, apperrors.New(apperrors.ErrCodeAgentConfig, "invalid OpenAI config", nil)
		}
		return NewOpenAIClient(openaiCfg)

	case "Anthropic":
		anthropicCfg, ok := cfg.(*config.AnthropicConfig)
		if !ok {
			return nil, apperrors.New(apperrors.ErrCodeAgentConfig, "invalid Anthropic config", nil)
		}
		return NewAnthropicClient(anthropicCfg)

	case "Gemini":
		geminiCfg, ok := cfg.(*config.GeminiConfig)
		if !ok {
			return nil, apperrors.New(apperrors.ErrCodeAgentConfig, "invalid Gemini config", nil)
		}
		return NewGeminiClient(geminiCfg)

	default:
		return nil, apperrors.New(apperrors.ErrCodeAgentConfig,
			fmt.Sprintf("unsupported model type: %s", cfg.Type()), nil)
	}
}

// ConvertToolsToDefinitions converts ADK tools to LLM tool definitions
func ConvertToolsToDefinitions(tools interface{}) []ToolDefinition {
	// This would convert from ADK tool interface to LLM tool definitions
	// For now, return empty slice
	// In a full implementation, this would introspect the tool interface
	// and generate JSON Schema for parameters
	return []ToolDefinition{}
}
