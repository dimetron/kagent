package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/kagent-dev/kagent/go/internal/executor/temporal/models"
	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
)

type openaiProvider struct {
	client *openai.Client
}

// NewOpenAIProvider creates a new OpenAI provider
func NewOpenAIProvider(apiKey string) Provider {
	client := openai.NewClient(option.WithAPIKey(apiKey))
	return &openaiProvider{
		client: client,
	}
}

func (p *openaiProvider) Name() string {
	return "openai"
}

func (p *openaiProvider) SupportedModels() []string {
	return []string{
		"gpt-4-turbo",
		"gpt-4-turbo-preview",
		"gpt-4",
		"gpt-4-32k",
		"gpt-3.5-turbo",
		"gpt-3.5-turbo-16k",
		"o1-preview",
		"o1-mini",
	}
}

func (p *openaiProvider) Chat(ctx context.Context, request models.LLMRequest) (*models.LLMResponse, error) {
	// Convert messages to OpenAI format
	messages := make([]openai.ChatCompletionMessageParamUnion, 0)

	for _, msg := range request.Messages {
		switch msg.Role {
		case "system":
			messages = append(messages, openai.SystemMessage(msg.Content))
		case "user":
			messages = append(messages, openai.UserMessage(msg.Content))
		case "assistant":
			if len(msg.ToolCalls) > 0 {
				// Assistant message with tool calls
				toolCalls := make([]openai.ChatCompletionMessageToolCallParam, 0)
				for _, tc := range msg.ToolCalls {
					argsJSON, _ := json.Marshal(tc.Arguments)
					toolCalls = append(toolCalls, openai.ChatCompletionMessageToolCallParam{
						ID:   openai.F(tc.ID),
						Type: openai.F(openai.ChatCompletionMessageToolCallTypeFunction),
						Function: openai.F(openai.ChatCompletionMessageToolCallFunctionParam{
							Name:      openai.F(tc.Name),
							Arguments: openai.F(string(argsJSON)),
						}),
					})
				}
				messages = append(messages, openai.AssistantMessage(msg.Content, toolCalls...))
			} else {
				messages = append(messages, openai.AssistantMessage(msg.Content))
			}
		case "tool":
			messages = append(messages, openai.ToolMessage(msg.ToolCallID, msg.Content))
		}
	}

	// Convert tools to OpenAI format
	tools := make([]openai.ChatCompletionToolParam, 0)
	for _, tool := range request.Tools {
		tools = append(tools, openai.ChatCompletionToolParam{
			Type: openai.F(openai.ChatCompletionToolTypeFunction),
			Function: openai.F(openai.FunctionDefinitionParam{
				Name:        openai.String(tool.Name),
				Description: openai.String(tool.Description),
				Parameters:  openai.F(openai.FunctionParameters(tool.Parameters)),
			}),
		})
	}

	// Create completion request
	params := openai.ChatCompletionNewParams{
		Model:       openai.F(request.ModelConfig.Model),
		Messages:    openai.F(messages),
		MaxTokens:   openai.Int(int64(request.ModelConfig.MaxTokens)),
		Temperature: openai.Float(request.ModelConfig.Temperature),
	}

	if len(tools) > 0 {
		params.Tools = openai.F(tools)
	}

	if request.ModelConfig.TopP > 0 {
		params.TopP = openai.Float(request.ModelConfig.TopP)
	}

	// Call API
	completion, err := p.client.Chat.Completions.New(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("openai API error: %w", err)
	}

	if len(completion.Choices) == 0 {
		return nil, fmt.Errorf("no completion choices returned")
	}

	choice := completion.Choices[0]
	response := &models.LLMResponse{
		Content:      choice.Message.Content,
		ModelUsed:    completion.Model,
		FinishReason: string(choice.FinishReason),
		TokenUsage: models.TokenUsage{
			PromptTokens:     int(completion.Usage.PromptTokens),
			CompletionTokens: int(completion.Usage.CompletionTokens),
			TotalTokens:      int(completion.Usage.TotalTokens),
		},
	}

	// Extract tool calls
	if len(choice.Message.ToolCalls) > 0 {
		response.ToolCalls = make([]models.ToolCall, 0)
		for _, tc := range choice.Message.ToolCalls {
			var args map[string]interface{}
			if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err != nil {
				return nil, fmt.Errorf("failed to parse tool arguments: %w", err)
			}
			response.ToolCalls = append(response.ToolCalls, models.ToolCall{
				ID:        tc.ID,
				Name:      tc.Function.Name,
				Arguments: args,
				Status:    "pending",
			})
		}
	}

	return response, nil
}

func (p *openaiProvider) ChatStream(ctx context.Context, request models.LLMRequest) (<-chan StreamChunk, <-chan error) {
	chunkChan := make(chan StreamChunk, 10)
	errChan := make(chan error, 1)

	go func() {
		defer close(chunkChan)
		defer close(errChan)

		// Convert messages (same as Chat method)
		messages := make([]openai.ChatCompletionMessageParamUnion, 0)
		for _, msg := range request.Messages {
			switch msg.Role {
			case "system":
				messages = append(messages, openai.SystemMessage(msg.Content))
			case "user":
				messages = append(messages, openai.UserMessage(msg.Content))
			case "assistant":
				if len(msg.ToolCalls) > 0 {
					toolCalls := make([]openai.ChatCompletionMessageToolCallParam, 0)
					for _, tc := range msg.ToolCalls {
						argsJSON, _ := json.Marshal(tc.Arguments)
						toolCalls = append(toolCalls, openai.ChatCompletionMessageToolCallParam{
							ID:   openai.F(tc.ID),
							Type: openai.F(openai.ChatCompletionMessageToolCallTypeFunction),
							Function: openai.F(openai.ChatCompletionMessageToolCallFunctionParam{
								Name:      openai.F(tc.Name),
								Arguments: openai.F(string(argsJSON)),
							}),
						})
					}
					messages = append(messages, openai.AssistantMessage(msg.Content, toolCalls...))
				} else {
					messages = append(messages, openai.AssistantMessage(msg.Content))
				}
			case "tool":
				messages = append(messages, openai.ToolMessage(msg.ToolCallID, msg.Content))
			}
		}

		// Convert tools
		tools := make([]openai.ChatCompletionToolParam, 0)
		for _, tool := range request.Tools {
			tools = append(tools, openai.ChatCompletionToolParam{
				Type: openai.F(openai.ChatCompletionToolTypeFunction),
				Function: openai.F(openai.FunctionDefinitionParam{
					Name:        openai.String(tool.Name),
					Description: openai.String(tool.Description),
					Parameters:  openai.F(openai.FunctionParameters(tool.Parameters)),
				}),
			})
		}

		// Create streaming request
		params := openai.ChatCompletionNewParams{
			Model:       openai.F(request.ModelConfig.Model),
			Messages:    openai.F(messages),
			MaxTokens:   openai.Int(int64(request.ModelConfig.MaxTokens)),
			Temperature: openai.Float(request.ModelConfig.Temperature),
		}

		if len(tools) > 0 {
			params.Tools = openai.F(tools)
		}

		// Stream response
		stream := p.client.Chat.Completions.NewStreaming(ctx, params)

		for stream.Next() {
			chunk := stream.Current()
			if len(chunk.Choices) > 0 {
				delta := chunk.Choices[0].Delta
				if delta.Content != "" {
					chunkChan <- StreamChunk{
						Content: delta.Content,
						Delta:   true,
					}
				}

				if chunk.Choices[0].FinishReason != "" {
					chunkChan <- StreamChunk{
						FinishReason: string(chunk.Choices[0].FinishReason),
						Delta:        false,
					}
				}
			}
		}

		if stream.Err() != nil && stream.Err() != io.EOF {
			errChan <- fmt.Errorf("openai stream error: %w", stream.Err())
		}
	}()

	return chunkChan, errChan
}
