package llm

import (
	"context"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/kagent-dev/kagent/go/pkg/adk/config"
	"github.com/kagent-dev/kagent/go/pkg/adk/converters"
	apperrors "github.com/kagent-dev/kagent/go/pkg/adk/errors"
)

// AnthropicClient implements the Client interface for Anthropic
type AnthropicClient struct {
	client *anthropic.Client
	config *config.AnthropicConfig
}

// NewAnthropicClient creates a new Anthropic client
func NewAnthropicClient(cfg *config.AnthropicConfig) (*AnthropicClient, error) {
	if cfg == nil {
		return nil, apperrors.New(apperrors.ErrCodeAgentConfig, "Anthropic config is required", nil)
	}

	opts := []option.RequestOption{}

	// Set API key if provided
	if cfg.APIKey != nil && *cfg.APIKey != "" {
		opts = append(opts, option.WithAPIKey(*cfg.APIKey))
	}

	client := anthropic.NewClient(opts...)

	return &AnthropicClient{
		client: client,
		config: cfg,
	}, nil
}

// Generate sends a message and receives a response
func (c *AnthropicClient) Generate(ctx context.Context, messages []*converters.Content, genConfig *GenerateConfig) (*Response, error) {
	// Convert messages to Anthropic format
	anthropicMessages, system := c.convertMessages(messages)

	// Build request parameters
	params := anthropic.MessageNewParams{
		Model:    anthropic.F(c.config.Model),
		Messages: anthropic.F(anthropicMessages),
	}

	// Set max tokens (required by Anthropic)
	maxTokens := 4096
	if genConfig != nil && genConfig.MaxTokens != nil {
		maxTokens = *genConfig.MaxTokens
	} else if c.config.MaxTokens != nil {
		maxTokens = *c.config.MaxTokens
	}
	params.MaxTokens = anthropic.Int(maxTokens)

	// Apply generation config
	if genConfig != nil {
		if genConfig.Temperature != nil {
			params.Temperature = anthropic.Float(*genConfig.Temperature)
		} else if c.config.Temperature != nil {
			params.Temperature = anthropic.Float(*c.config.Temperature)
		}

		if genConfig.TopP != nil {
			params.TopP = anthropic.Float(*genConfig.TopP)
		} else if c.config.TopP != nil {
			params.TopP = anthropic.Float(*c.config.TopP)
		}

		if genConfig.TopK != nil {
			params.TopK = anthropic.Int(*genConfig.TopK)
		} else if c.config.TopK != nil {
			params.TopK = anthropic.Int(*c.config.TopK)
		}

		if genConfig.StopSequences != nil {
			params.StopSequences = anthropic.F(genConfig.StopSequences)
		}

		// Add tools if provided
		if len(genConfig.Tools) > 0 {
			tools := c.convertTools(genConfig.Tools)
			params.Tools = anthropic.F(tools)
		}
	}

	// Add system message if present
	if system != "" {
		params.System = anthropic.F([]anthropic.TextBlockParam{
			anthropic.NewTextBlock(system),
		})
	}

	// Make the request
	message, err := c.client.Messages.New(ctx, params)
	if err != nil {
		return nil, apperrors.New(apperrors.ErrCodeExecutorFailed, "Anthropic API call failed", err)
	}

	// Convert response
	return c.convertResponse(message), nil
}

// GenerateStream sends a message and streams responses
func (c *AnthropicClient) GenerateStream(ctx context.Context, messages []*converters.Content, genConfig *GenerateConfig) (<-chan *StreamEvent, error) {
	// Convert messages to Anthropic format
	anthropicMessages, system := c.convertMessages(messages)

	// Build request parameters (similar to Generate)
	params := anthropic.MessageNewParams{
		Model:    anthropic.F(c.config.Model),
		Messages: anthropic.F(anthropicMessages),
	}

	maxTokens := 4096
	if genConfig != nil && genConfig.MaxTokens != nil {
		maxTokens = *genConfig.MaxTokens
	} else if c.config.MaxTokens != nil {
		maxTokens = *c.config.MaxTokens
	}
	params.MaxTokens = anthropic.Int(maxTokens)

	if genConfig != nil {
		if genConfig.Temperature != nil {
			params.Temperature = anthropic.Float(*genConfig.Temperature)
		}
		if genConfig.TopP != nil {
			params.TopP = anthropic.Float(*genConfig.TopP)
		}
		if genConfig.TopK != nil {
			params.TopK = anthropic.Int(*genConfig.TopK)
		}
		if len(genConfig.Tools) > 0 {
			tools := c.convertTools(genConfig.Tools)
			params.Tools = anthropic.F(tools)
		}
	}

	if system != "" {
		params.System = anthropic.F([]anthropic.TextBlockParam{
			anthropic.NewTextBlock(system),
		})
	}

	// Create streaming request
	stream := c.client.Messages.NewStreaming(ctx, params)

	// Create event channel
	events := make(chan *StreamEvent, 10)

	// Stream in background
	go func() {
		defer close(events)

		events <- &StreamEvent{
			Type: StreamEventTypeStart,
		}

		var accumulatedText string
		for stream.Next() {
			event := stream.Current()

			switch event.Type {
			case anthropic.MessageStreamEventTypeContentBlockDelta:
				if event.Delta.Type == anthropic.ContentBlockDeltaEventDeltaTypeTextDelta {
					text := event.Delta.Text
					accumulatedText += text
					events <- &StreamEvent{
						Type: StreamEventTypeDelta,
						Delta: &ContentDelta{
							Text: text,
						},
					}
				}

			case anthropic.MessageStreamEventTypeMessageStop:
				events <- &StreamEvent{
					Type: StreamEventTypeDelta,
					Delta: &ContentDelta{
						StopDelta: &StopDelta{
							StopReason: string(event.Message.StopReason),
						},
					},
				}
			}
		}

		if err := stream.Err(); err != nil {
			events <- &StreamEvent{
				Type:  StreamEventTypeError,
				Error: err,
			}
			return
		}

		// Send final content
		if accumulatedText != "" {
			events <- &StreamEvent{
				Type: StreamEventTypeContent,
				Content: &converters.Content{
					Role: "assistant",
					Parts: []*converters.Part{
						{
							Type: converters.PartTypeText,
							Data: &converters.TextPartData{
								Text: accumulatedText,
							},
						},
					},
				},
			}
		}

		events <- &StreamEvent{
			Type: StreamEventTypeDone,
			Done: true,
		}
	}()

	return events, nil
}

// SupportsTools returns whether this client supports tool calling
func (c *AnthropicClient) SupportsTools() bool {
	return true
}

// ModelName returns the name of the model being used
func (c *AnthropicClient) ModelName() string {
	return c.config.Model
}

func (c *AnthropicClient) convertMessages(messages []*converters.Content) ([]anthropic.MessageParam, string) {
	var result []anthropic.MessageParam
	var systemMessage string

	for _, msg := range messages {
		// Extract text from parts
		var text string
		for _, part := range msg.Parts {
			if part.Type == converters.PartTypeText {
				if textData, ok := part.Data.(*converters.TextPartData); ok {
					text += textData.Text
				}
			}
		}

		switch msg.Role {
		case "system":
			systemMessage = text
		case "user":
			result = append(result, anthropic.NewUserMessage(anthropic.NewTextBlock(text)))
		case "assistant":
			result = append(result, anthropic.NewAssistantMessage(anthropic.NewTextBlock(text)))
		}
	}

	return result, systemMessage
}

func (c *AnthropicClient) convertTools(tools []ToolDefinition) []anthropic.ToolParam {
	var result []anthropic.ToolParam

	for _, tool := range tools {
		params := anthropic.ToolParam{
			Name:        anthropic.F(tool.Name),
			Description: anthropic.F(tool.Description),
		}

		if tool.Parameters != nil {
			params.InputSchema = anthropic.F(tool.Parameters)
		}

		result = append(result, params)
	}

	return result
}

func (c *AnthropicClient) convertResponse(message *anthropic.Message) *Response {
	response := &Response{
		StopReason: string(message.StopReason),
		Usage: &Usage{
			InputTokens:  int(message.Usage.InputTokens),
			OutputTokens: int(message.Usage.OutputTokens),
			TotalTokens:  int(message.Usage.InputTokens + message.Usage.OutputTokens),
		},
	}

	var parts []*converters.Part

	for _, content := range message.Content {
		switch content.Type {
		case anthropic.ContentBlockTypeText:
			parts = append(parts, &converters.Part{
				Type: converters.PartTypeText,
				Data: &converters.TextPartData{
					Text: content.Text,
				},
			})

		case anthropic.ContentBlockTypeToolUse:
			// Convert tool use to function call
			toolCall := ToolCall{
				ID:        content.ID,
				Name:      content.Name,
				Arguments: content.Input.(map[string]interface{}),
			}

			response.ToolCalls = append(response.ToolCalls, toolCall)

			parts = append(parts, &converters.Part{
				Type: converters.PartTypeFunctionCall,
				Data: &converters.FunctionCallData{
					Name: content.Name,
					Args: content.Input.(map[string]interface{}),
					ID:   content.ID,
				},
			})
		}
	}

	response.Content = &converters.Content{
		Role:  "assistant",
		Parts: parts,
	}

	return response
}
