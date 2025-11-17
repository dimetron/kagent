package llm

import (
	"context"
	"fmt"

	"github.com/kagent-dev/kagent/go/pkg/adk/config"
	"github.com/kagent-dev/kagent/go/pkg/adk/converters"
	apperrors "github.com/kagent-dev/kagent/go/pkg/adk/errors"
	"google.golang.org/genai"
)

// GeminiClient implements the Client interface for Google Gemini
type GeminiClient struct {
	client *genai.Client
	config *config.GeminiConfig
}

// NewGeminiClient creates a new Gemini client
func NewGeminiClient(cfg *config.GeminiConfig) (*GeminiClient, error) {
	if cfg == nil {
		return nil, apperrors.New(apperrors.ErrCodeAgentConfig, "config is required", nil)
	}

	if cfg.APIKey == nil || *cfg.APIKey == "" {
		return nil, apperrors.New(apperrors.ErrCodeAgentConfig, "API key is required", nil)
	}

	// Create client with API key
	ctx := context.Background()
	clientConfig := &genai.ClientConfig{
		APIKey: *cfg.APIKey,
	}

	// Set base URL if provided
	if cfg.BaseURL != nil && *cfg.BaseURL != "" {
		clientConfig.BaseURL = *cfg.BaseURL
	}

	client, err := genai.NewClient(ctx, clientConfig)
	if err != nil {
		return nil, apperrors.New(apperrors.ErrCodeAgentConfig, "failed to create Gemini client", err)
	}

	return &GeminiClient{
		client: client,
		config: cfg,
	}, nil
}

// Generate sends a message and receives a response
func (c *GeminiClient) Generate(ctx context.Context, messages []*converters.Content, genConfig *GenerateConfig) (*Response, error) {
	// Convert messages to Gemini format
	geminiContents := c.convertMessages(messages)

	// Build generate config
	config := c.buildConfig(genConfig)

	// Generate content
	resp, err := c.client.Models.GenerateContent(ctx, c.config.Model, geminiContents, config)
	if err != nil {
		return nil, apperrors.New(apperrors.ErrCodeExecutorFailed, "Gemini API call failed", err)
	}

	// Convert response
	return c.convertResponse(resp), nil
}

// GenerateStream sends a message and streams responses
func (c *GeminiClient) GenerateStream(ctx context.Context, messages []*converters.Content, genConfig *GenerateConfig) (<-chan *StreamEvent, error) {
	// Convert messages to Gemini format
	geminiContents := c.convertMessages(messages)

	// Build generate config
	config := c.buildConfig(genConfig)

	// Create event channel
	events := make(chan *StreamEvent, 10)

	// Start streaming in background
	go func() {
		defer close(events)

		// Send start event
		events <- &StreamEvent{
			Type: StreamEventTypeStart,
		}

		// Accumulate response
		var fullText string
		var toolCalls []ToolCall

		// Stream content
		for resp, err := range c.client.Models.GenerateContentStream(ctx, c.config.Model, geminiContents, config) {
			if err != nil {
				events <- &StreamEvent{
					Type:  StreamEventTypeError,
					Error: apperrors.New(apperrors.ErrCodeExecutorFailed, "streaming failed", err),
				}
				return
			}

			// Process candidates
			if len(resp.Candidates) > 0 {
				candidate := resp.Candidates[0]

				for _, part := range candidate.Content.Parts {
					if part.Text != "" {
						// Text delta
						fullText += part.Text
						events <- &StreamEvent{
							Type: StreamEventTypeDelta,
							Delta: &ContentDelta{
								Text: part.Text,
							},
						}
					}

					if part.FunctionCall != nil {
						// Function call
						args := make(map[string]interface{})
						if part.FunctionCall.Args != nil {
							for k, v := range part.FunctionCall.Args {
								args[k] = v
							}
						}

						toolCall := ToolCall{
							ID:        part.FunctionCall.Name, // Gemini doesn't have call IDs
							Name:      part.FunctionCall.Name,
							Arguments: args,
						}
						toolCalls = append(toolCalls, toolCall)
					}
				}
			}
		}

		// Send final content event
		content := &converters.Content{
			Role: "assistant",
			Parts: []*converters.Part{
				{
					Type: converters.PartTypeText,
					Data: &converters.TextPartData{Text: fullText},
				},
			},
		}

		events <- &StreamEvent{
			Type:    StreamEventTypeContent,
			Content: content,
		}

		// Add tool calls to response if any
		if len(toolCalls) > 0 {
			events <- &StreamEvent{
				Type:      StreamEventTypeToolCalls,
				ToolCalls: toolCalls,
			}
		}

		// Send done event
		events <- &StreamEvent{
			Type: StreamEventTypeDone,
			Done: true,
		}
	}()

	return events, nil
}

// SupportsTools returns whether this client supports tool calling
func (c *GeminiClient) SupportsTools() bool {
	return true
}

// ModelName returns the name of the model being used
func (c *GeminiClient) ModelName() string {
	return c.config.Model
}

func (c *GeminiClient) convertMessages(messages []*converters.Content) []*genai.Content {
	var geminiContents []*genai.Content

	for _, msg := range messages {
		var parts []*genai.Part

		for _, part := range msg.Parts {
			switch part.Type {
			case converters.PartTypeText:
				if textData, ok := part.Data.(*converters.TextPartData); ok {
					parts = append(parts, &genai.Part{
						Text: textData.Text,
					})
				}

			case converters.PartTypeFunctionCall:
				if funcData, ok := part.Data.(*converters.FunctionCallData); ok {
					parts = append(parts, &genai.Part{
						FunctionCall: &genai.FunctionCall{
							Name: funcData.Name,
							Args: funcData.Arguments,
						},
					})
				}

			case converters.PartTypeFunctionResponse:
				if respData, ok := part.Data.(*converters.FunctionResponseData); ok {
					// Gemini expects function responses as FunctionResponse
					responseMap := map[string]interface{}{
						"result": respData.Response,
					}
					parts = append(parts, &genai.Part{
						FunctionResponse: &genai.FunctionResponse{
							Name:     respData.Name,
							Response: responseMap,
						},
					})
				}
			}
		}

		// Convert role
		role := msg.Role
		if role == "assistant" {
			role = "model"
		}

		geminiContents = append(geminiContents, &genai.Content{
			Parts: parts,
			Role:  role,
		})
	}

	return geminiContents
}

func (c *GeminiClient) buildConfig(genConfig *GenerateConfig) *genai.GenerateContentConfig {
	config := &genai.GenerateContentConfig{}

	// Set temperature
	if genConfig != nil && genConfig.Temperature != nil {
		config.Temperature = genai.Ptr(float32(*genConfig.Temperature))
	} else if c.config.Temperature != nil {
		config.Temperature = genai.Ptr(float32(*c.config.Temperature))
	}

	// Set max tokens
	if genConfig != nil && genConfig.MaxTokens != nil {
		config.MaxOutputTokens = genai.Ptr(int32(*genConfig.MaxTokens))
	} else if c.config.MaxTokens != nil {
		config.MaxOutputTokens = genai.Ptr(int32(*c.config.MaxTokens))
	}

	// Set top P
	if genConfig != nil && genConfig.TopP != nil {
		config.TopP = genai.Ptr(float32(*genConfig.TopP))
	} else if c.config.TopP != nil {
		config.TopP = genai.Ptr(float32(*c.config.TopP))
	}

	// Set top K
	if genConfig != nil && genConfig.TopK != nil {
		config.TopK = genai.Ptr(int32(*genConfig.TopK))
	} else if c.config.TopK != nil {
		config.TopK = genai.Ptr(int32(*c.config.TopK))
	}

	// Set stop sequences
	if genConfig != nil && len(genConfig.StopSequences) > 0 {
		config.StopSequences = genConfig.StopSequences
	}

	// Convert tools to Gemini function declarations
	if genConfig != nil && len(genConfig.Tools) > 0 {
		var functionDeclarations []*genai.FunctionDeclaration

		for _, tool := range genConfig.Tools {
			functionDeclarations = append(functionDeclarations, &genai.FunctionDeclaration{
				Name:        tool.Name,
				Description: tool.Description,
				Parameters:  c.convertSchema(tool.Parameters),
			})
		}

		config.Tools = []*genai.Tool{
			{
				FunctionDeclarations: functionDeclarations,
			},
		}
	}

	return config
}

func (c *GeminiClient) convertSchema(params map[string]interface{}) *genai.Schema {
	if params == nil {
		return nil
	}

	schema := &genai.Schema{}

	// Extract type
	if schemaType, ok := params["type"].(string); ok {
		schema.Type = schemaType
	}

	// Extract description
	if desc, ok := params["description"].(string); ok {
		schema.Description = desc
	}

	// Extract properties
	if props, ok := params["properties"].(map[string]interface{}); ok {
		schema.Properties = make(map[string]*genai.Schema)
		for key, value := range props {
			if propMap, ok := value.(map[string]interface{}); ok {
				schema.Properties[key] = c.convertSchema(propMap)
			}
		}
	}

	// Extract required fields
	if required, ok := params["required"].([]interface{}); ok {
		schema.Required = make([]string, len(required))
		for i, field := range required {
			if fieldStr, ok := field.(string); ok {
				schema.Required[i] = fieldStr
			}
		}
	} else if required, ok := params["required"].([]string); ok {
		schema.Required = required
	}

	// Extract enum
	if enum, ok := params["enum"].([]interface{}); ok {
		schema.Enum = make([]string, len(enum))
		for i, val := range enum {
			if valStr, ok := val.(string); ok {
				schema.Enum[i] = valStr
			}
		}
	} else if enum, ok := params["enum"].([]string); ok {
		schema.Enum = enum
	}

	return schema
}

func (c *GeminiClient) convertResponse(resp *genai.GenerateContentResponse) *Response {
	response := &Response{
		Content: &converters.Content{
			Role:  "assistant",
			Parts: []*converters.Part{},
		},
		ToolCalls: []ToolCall{},
	}

	if len(resp.Candidates) == 0 {
		return response
	}

	candidate := resp.Candidates[0]

	// Extract text and tool calls from parts
	var textParts []string
	for _, part := range candidate.Content.Parts {
		if part.Text != "" {
			textParts = append(textParts, part.Text)
		}

		if part.FunctionCall != nil {
			args := make(map[string]interface{})
			if part.FunctionCall.Args != nil {
				for k, v := range part.FunctionCall.Args {
					args[k] = v
				}
			}

			response.ToolCalls = append(response.ToolCalls, ToolCall{
				ID:        fmt.Sprintf("call_%s", part.FunctionCall.Name),
				Name:      part.FunctionCall.Name,
				Arguments: args,
			})
		}
	}

	// Add text content
	if len(textParts) > 0 {
		var fullText string
		for _, t := range textParts {
			fullText += t
		}
		response.Content.Parts = append(response.Content.Parts, &converters.Part{
			Type: converters.PartTypeText,
			Data: &converters.TextPartData{Text: fullText},
		})
	}

	// Set finish reason
	if candidate.FinishReason != "" {
		response.FinishReason = candidate.FinishReason
	}

	// Extract usage metadata if available
	if resp.UsageMetadata != nil {
		response.Usage = &Usage{
			PromptTokens:     int(resp.UsageMetadata.PromptTokenCount),
			CompletionTokens: int(resp.UsageMetadata.CandidatesTokenCount),
			TotalTokens:      int(resp.UsageMetadata.TotalTokenCount),
		}
	}

	return response
}
