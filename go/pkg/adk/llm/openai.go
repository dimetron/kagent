package llm

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/kagent-dev/kagent/go/pkg/adk/config"
	"github.com/kagent-dev/kagent/go/pkg/adk/converters"
	apperrors "github.com/kagent-dev/kagent/go/pkg/adk/errors"
	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
)

// OpenAIClient implements the Client interface for OpenAI
type OpenAIClient struct {
	client *openai.Client
	config *config.OpenAIConfig
}

// NewOpenAIClient creates a new OpenAI client
func NewOpenAIClient(cfg *config.OpenAIConfig) (*OpenAIClient, error) {
	if cfg == nil {
		return nil, apperrors.New(apperrors.ErrCodeAgentConfig, "OpenAI config is required", nil)
	}

	opts := []option.RequestOption{}

	// Set API key if provided
	if cfg.APIKey != nil && *cfg.APIKey != "" {
		opts = append(opts, option.WithAPIKey(*cfg.APIKey))
	}

	// Set base URL if provided
	if cfg.BaseURL != nil && *cfg.BaseURL != "" {
		opts = append(opts, option.WithBaseURL(*cfg.BaseURL))
	}

	client := openai.NewClient(opts...)

	return &OpenAIClient{
		client: client,
		config: cfg,
	}, nil
}

// Generate sends a message and receives a response
func (c *OpenAIClient) Generate(ctx context.Context, messages []*converters.Content, genConfig *GenerateConfig) (*Response, error) {
	// Convert messages to OpenAI format
	openaiMessages := c.convertMessages(messages)

	// Build request parameters
	params := openai.ChatCompletionNewParams{
		Model:    openai.F(c.config.Model),
		Messages: openai.F(openaiMessages),
	}

	// Apply generation config
	if genConfig != nil {
		if genConfig.Temperature != nil {
			params.Temperature = openai.Float(*genConfig.Temperature)
		} else if c.config.Temperature != nil {
			params.Temperature = openai.Float(*c.config.Temperature)
		}

		if genConfig.MaxTokens != nil {
			params.MaxTokens = openai.Int(*genConfig.MaxTokens)
		} else if c.config.MaxTokens != nil {
			params.MaxTokens = openai.Int(*c.config.MaxTokens)
		}

		if genConfig.TopP != nil {
			params.TopP = openai.Float(*genConfig.TopP)
		} else if c.config.TopP != nil {
			params.TopP = openai.Float(*c.config.TopP)
		}

		if c.config.FrequencyPenalty != nil {
			params.FrequencyPenalty = openai.Float(*c.config.FrequencyPenalty)
		}

		if genConfig.StopSequences != nil {
			params.Stop = openai.F[openai.ChatCompletionNewParamsStopUnion](
				openai.ChatCompletionNewParamsStopArray(genConfig.StopSequences),
			)
		}

		// Add tools if provided
		if len(genConfig.Tools) > 0 {
			tools := c.convertTools(genConfig.Tools)
			params.Tools = openai.F(tools)
		}
	}

	// Make the request
	completion, err := c.client.Chat.Completions.New(ctx, params)
	if err != nil {
		return nil, apperrors.New(apperrors.ErrCodeExecutorFailed, "OpenAI API call failed", err)
	}

	// Convert response
	return c.convertResponse(completion), nil
}

// GenerateStream sends a message and streams responses
func (c *OpenAIClient) GenerateStream(ctx context.Context, messages []*converters.Content, genConfig *GenerateConfig) (<-chan *StreamEvent, error) {
	// Convert messages to OpenAI format
	openaiMessages := c.convertMessages(messages)

	// Build request parameters
	params := openai.ChatCompletionNewParams{
		Model:    openai.F(c.config.Model),
		Messages: openai.F(openaiMessages),
	}

	// Apply generation config
	if genConfig != nil {
		if genConfig.Temperature != nil {
			params.Temperature = openai.Float(*genConfig.Temperature)
		}
		if genConfig.MaxTokens != nil {
			params.MaxTokens = openai.Int(*genConfig.MaxTokens)
		}
		if genConfig.TopP != nil {
			params.TopP = openai.Float(*genConfig.TopP)
		}
		if len(genConfig.Tools) > 0 {
			tools := c.convertTools(genConfig.Tools)
			params.Tools = openai.F(tools)
		}
	}

	// Create streaming request
	stream := c.client.Chat.Completions.NewStreaming(ctx, params)

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
			chunk := stream.Current()

			if len(chunk.Choices) > 0 {
				delta := chunk.Choices[0].Delta

				// Handle text delta
				if delta.Content != "" {
					accumulatedText += delta.Content
					events <- &StreamEvent{
						Type: StreamEventTypeDelta,
						Delta: &ContentDelta{
							Text: delta.Content,
						},
					}
				}

				// Handle tool calls
				if len(delta.ToolCalls) > 0 {
					for _, tc := range delta.ToolCalls {
						if tc.Function.Name != "" {
							var args map[string]interface{}
							json.Unmarshal([]byte(tc.Function.Arguments), &args)

							events <- &StreamEvent{
								Type: StreamEventTypeDelta,
								Delta: &ContentDelta{
									ToolCall: &ToolCall{
										ID:        tc.ID,
										Name:      tc.Function.Name,
										Arguments: args,
									},
								},
							}
						}
					}
				}

				// Check for finish
				if chunk.Choices[0].FinishReason != "" {
					events <- &StreamEvent{
						Type: StreamEventTypeDelta,
						Delta: &ContentDelta{
							StopDelta: &StopDelta{
								FinishReason: string(chunk.Choices[0].FinishReason),
							},
						},
					}
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
func (c *OpenAIClient) SupportsTools() bool {
	return true
}

// ModelName returns the name of the model being used
func (c *OpenAIClient) ModelName() string {
	return c.config.Model
}

func (c *OpenAIClient) convertMessages(messages []*converters.Content) []openai.ChatCompletionMessageParamUnion {
	var result []openai.ChatCompletionMessageParamUnion

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
		case "user":
			result = append(result, openai.UserMessage(text))
		case "assistant":
			result = append(result, openai.AssistantMessage(text))
		case "system":
			result = append(result, openai.SystemMessage(text))
		}
	}

	return result
}

func (c *OpenAIClient) convertTools(tools []ToolDefinition) []openai.ChatCompletionToolParam {
	var result []openai.ChatCompletionToolParam

	for _, tool := range tools {
		params := openai.FunctionDefinitionParam{
			Name:        openai.String(tool.Name),
			Description: openai.String(tool.Description),
		}

		if tool.Parameters != nil {
			params.Parameters = openai.F(openai.FunctionParameters(tool.Parameters))
		}

		result = append(result, openai.ChatCompletionToolParam{
			Type:     openai.F(openai.ChatCompletionToolTypeFunction),
			Function: openai.F(params),
		})
	}

	return result
}

func (c *OpenAIClient) convertResponse(completion *openai.ChatCompletion) *Response {
	response := &Response{
		FinishReason: string(completion.Choices[0].FinishReason),
		Usage: &Usage{
			InputTokens:  int(completion.Usage.PromptTokens),
			OutputTokens: int(completion.Usage.CompletionTokens),
			TotalTokens:  int(completion.Usage.TotalTokens),
		},
	}

	choice := completion.Choices[0]

	// Convert message content
	var parts []*converters.Part

	if choice.Message.Content != "" {
		parts = append(parts, &converters.Part{
			Type: converters.PartTypeText,
			Data: &converters.TextPartData{
				Text: choice.Message.Content,
			},
		})
	}

	// Convert tool calls
	if len(choice.Message.ToolCalls) > 0 {
		for _, tc := range choice.Message.ToolCalls {
			var args map[string]interface{}
			json.Unmarshal([]byte(tc.Function.Arguments), &args)

			response.ToolCalls = append(response.ToolCalls, ToolCall{
				ID:        tc.ID,
				Name:      tc.Function.Name,
				Arguments: args,
			})

			parts = append(parts, &converters.Part{
				Type: converters.PartTypeFunctionCall,
				Data: &converters.FunctionCallData{
					Name: tc.Function.Name,
					Args: args,
					ID:   tc.ID,
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
