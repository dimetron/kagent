package llm

import (
	"context"
	"fmt"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/kagent-dev/kagent/go/internal/executor/temporal/models"
)

type anthropicProvider struct {
	client *anthropic.Client
}

// NewAnthropicProvider creates a new Anthropic provider
func NewAnthropicProvider(apiKey string) Provider {
	client := anthropic.NewClient(option.WithAPIKey(apiKey))
	return &anthropicProvider{
		client: client,
	}
}

func (p *anthropicProvider) Name() string {
	return "anthropic"
}

func (p *anthropicProvider) SupportedModels() []string {
	return []string{
		"claude-3-5-sonnet-20241022",
		"claude-3-5-haiku-20241022",
		"claude-3-opus-20240229",
		"claude-3-sonnet-20240229",
		"claude-3-haiku-20240307",
	}
}

func (p *anthropicProvider) Chat(ctx context.Context, request models.LLMRequest) (*models.LLMResponse, error) {
	// Convert messages to Anthropic format
	messages := make([]anthropic.MessageParam, 0)
	var systemMessage string

	for _, msg := range request.Messages {
		switch msg.Role {
		case "system":
			systemMessage = msg.Content
		case "user":
			messages = append(messages, anthropic.NewUserMessage(anthropic.NewTextBlock(msg.Content)))
		case "assistant":
			if len(msg.ToolCalls) > 0 {
				// Handle assistant message with tool calls
				toolUseBlocks := make([]anthropic.ContentBlockParamUnion, 0)
				if msg.Content != "" {
					toolUseBlocks = append(toolUseBlocks, anthropic.NewTextBlock(msg.Content))
				}
				for _, tc := range msg.ToolCalls {
					toolUseBlocks = append(toolUseBlocks, anthropic.NewToolUseBlock(tc.ID, tc.Name, tc.Arguments))
				}
				messages = append(messages, anthropic.NewAssistantMessage(toolUseBlocks...))
			} else {
				messages = append(messages, anthropic.NewAssistantMessage(anthropic.NewTextBlock(msg.Content)))
			}
		case "tool":
			// Tool result message
			messages = append(messages, anthropic.NewUserMessage(
				anthropic.NewToolResultBlock(msg.ToolCallID, msg.Content, false),
			))
		}
	}

	// Convert tools to Anthropic format
	tools := make([]anthropic.ToolParam, 0)
	for _, tool := range request.Tools {
		tools = append(tools, anthropic.ToolParam{
			Name:        anthropic.F(tool.Name),
			Description: anthropic.F(tool.Description),
			InputSchema: anthropic.F(tool.Parameters),
		})
	}

	// Create message request
	params := anthropic.MessageNewParams{
		Model:       anthropic.F(request.ModelConfig.Model),
		Messages:    anthropic.F(messages),
		MaxTokens:   anthropic.Int(request.ModelConfig.MaxTokens),
		Temperature: anthropic.Float(request.ModelConfig.Temperature),
	}

	if systemMessage != "" {
		params.System = anthropic.F([]anthropic.TextBlockParam{
			anthropic.NewTextBlock(systemMessage),
		})
	}

	if len(tools) > 0 {
		params.Tools = anthropic.F(tools)
	}

	if request.ModelConfig.TopP > 0 {
		params.TopP = anthropic.Float(request.ModelConfig.TopP)
	}

	// Call API
	message, err := p.client.Messages.New(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("anthropic API error: %w", err)
	}

	// Convert response
	response := &models.LLMResponse{
		ModelUsed:    message.Model,
		FinishReason: string(message.StopReason),
		TokenUsage: models.TokenUsage{
			PromptTokens:     int(message.Usage.InputTokens),
			CompletionTokens: int(message.Usage.OutputTokens),
			TotalTokens:      int(message.Usage.InputTokens + message.Usage.OutputTokens),
		},
	}

	// Extract content and tool calls
	for _, block := range message.Content {
		switch b := block.AsUnion().(type) {
		case anthropic.TextBlock:
			response.Content += b.Text
		case anthropic.ToolUseBlock:
			response.ToolCalls = append(response.ToolCalls, models.ToolCall{
				ID:        b.ID,
				Name:      b.Name,
				Arguments: b.Input.(map[string]interface{}),
				Status:    "pending",
			})
		}
	}

	return response, nil
}

func (p *anthropicProvider) ChatStream(ctx context.Context, request models.LLMRequest) (<-chan StreamChunk, <-chan error) {
	chunkChan := make(chan StreamChunk, 10)
	errChan := make(chan error, 1)

	go func() {
		defer close(chunkChan)
		defer close(errChan)

		// Convert messages (same as Chat method)
		messages := make([]anthropic.MessageParam, 0)
		var systemMessage string

		for _, msg := range request.Messages {
			switch msg.Role {
			case "system":
				systemMessage = msg.Content
			case "user":
				messages = append(messages, anthropic.NewUserMessage(anthropic.NewTextBlock(msg.Content)))
			case "assistant":
				if len(msg.ToolCalls) > 0 {
					toolUseBlocks := make([]anthropic.ContentBlockParamUnion, 0)
					if msg.Content != "" {
						toolUseBlocks = append(toolUseBlocks, anthropic.NewTextBlock(msg.Content))
					}
					for _, tc := range msg.ToolCalls {
						toolUseBlocks = append(toolUseBlocks, anthropic.NewToolUseBlock(tc.ID, tc.Name, tc.Arguments))
					}
					messages = append(messages, anthropic.NewAssistantMessage(toolUseBlocks...))
				} else {
					messages = append(messages, anthropic.NewAssistantMessage(anthropic.NewTextBlock(msg.Content)))
				}
			case "tool":
				messages = append(messages, anthropic.NewUserMessage(
					anthropic.NewToolResultBlock(msg.ToolCallID, msg.Content, false),
				))
			}
		}

		// Convert tools
		tools := make([]anthropic.ToolParam, 0)
		for _, tool := range request.Tools {
			tools = append(tools, anthropic.ToolParam{
				Name:        anthropic.F(tool.Name),
				Description: anthropic.F(tool.Description),
				InputSchema: anthropic.F(tool.Parameters),
			})
		}

		// Create streaming request
		params := anthropic.MessageNewParams{
			Model:       anthropic.F(request.ModelConfig.Model),
			Messages:    anthropic.F(messages),
			MaxTokens:   anthropic.Int(request.ModelConfig.MaxTokens),
			Temperature: anthropic.Float(request.ModelConfig.Temperature),
		}

		if systemMessage != "" {
			params.System = anthropic.F([]anthropic.TextBlockParam{
				anthropic.NewTextBlock(systemMessage),
			})
		}

		if len(tools) > 0 {
			params.Tools = anthropic.F(tools)
		}

		// Stream response
		stream := p.client.Messages.NewStreaming(ctx, params)

		for stream.Next() {
			event := stream.Current()

			switch e := event.AsUnion().(type) {
			case anthropic.ContentBlockDeltaEvent:
				if delta, ok := e.Delta.AsUnion().(anthropic.TextDelta); ok {
					chunkChan <- StreamChunk{
						Content: delta.Text,
						Delta:   true,
					}
				}
			case anthropic.MessageDeltaEvent:
				if e.Delta.StopReason != "" {
					chunkChan <- StreamChunk{
						FinishReason: string(e.Delta.StopReason),
						Delta:        false,
					}
				}
			}
		}

		if err := stream.Err(); err != nil {
			errChan <- fmt.Errorf("anthropic stream error: %w", err)
		}
	}()

	return chunkChan, errChan
}
