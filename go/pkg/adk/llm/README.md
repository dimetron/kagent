# LLM Client Package

This package provides a unified interface for interacting with various Large Language Model (LLM) providers including OpenAI, Anthropic, and Gemini.

## Features

- **Unified Interface**: Single `Client` interface for all LLM providers
- **Tool Calling**: Support for function/tool calling across providers
- **Streaming**: Real-time streaming responses via Server-Sent Events
- **Type Safety**: Strongly-typed requests and responses
- **Error Handling**: Consistent error handling across providers

## Supported Providers

### OpenAI
- Models: GPT-4, GPT-3.5-turbo, etc.
- Features: Tool calling, streaming, vision (future)
- Configuration: API key, base URL, temperature, max tokens

### Anthropic
- Models: Claude 3.5 Sonnet, Claude 3 Opus, etc.
- Features: Tool calling, streaming, long context
- Configuration: API key, temperature, max tokens, top-p, top-k

### Gemini
- Models: Gemini 2.0 Flash, Gemini 1.5 Pro, etc.
- Features: Tool calling (function declarations), streaming, multimodal capabilities
- Configuration: API key, base URL, temperature, max tokens, top-p, top-k
- SDK: Uses official google.golang.org/genai package

## Client Interface

```go
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
```

## Usage

### Creating a Client

#### From Config
```go
import (
    "github.com/kagent-dev/kagent/go/pkg/adk/config"
    "github.com/kagent-dev/kagent/go/pkg/adk/llm"
)

// Create from agent config
agentConfig := &config.AgentConfig{
    Model: &config.OpenAIConfig{
        BaseModelConfig: config.BaseModelConfig{ModelType: "OpenAI"},
        Model:          "gpt-4",
        Temperature:    0.7,
    },
}

client, err := llm.NewClientFromConfig(agentConfig.Model)
if err != nil {
    // Handle error
}
```

#### Directly
```go
// OpenAI
apiKey := "your-api-key"
openaiConfig := &config.OpenAIConfig{
    BaseModelConfig: config.BaseModelConfig{ModelType: "OpenAI"},
    Model:          "gpt-4",
    APIKey:         &apiKey,
}
client, err := llm.NewOpenAIClient(openaiConfig)

// Anthropic
anthropicConfig := &config.AnthropicConfig{
    BaseModelConfig: config.BaseModelConfig{ModelType: "Anthropic"},
    Model:          "claude-3-5-sonnet-20241022",
    APIKey:         &apiKey,
}
client, err := llm.NewAnthropicClient(anthropicConfig)

// Gemini
geminiConfig := &config.GeminiConfig{
    BaseModelConfig: config.BaseModelConfig{ModelType: "Gemini"},
    Model:          "gemini-2.0-flash",
    APIKey:         &apiKey,
}
client, err := llm.NewGeminiClient(geminiConfig)
```

### Generating Responses

#### Simple Generation
```go
messages := []*converters.Content{
    {
        Role: "user",
        Parts: []*converters.Part{
            {
                Type: converters.PartTypeText,
                Data: &converters.TextPartData{Text: "What is 2+2?"},
            },
        },
    },
}

response, err := client.Generate(ctx, messages, nil)
if err != nil {
    // Handle error
}

// Access response
fmt.Println(response.Content.Parts[0].Data.(*converters.TextPartData).Text)
```

#### With Configuration
```go
temp := 0.7
maxTokens := 1000

config := &llm.GenerateConfig{
    Temperature: &temp,
    MaxTokens:   &maxTokens,
}

response, err := client.Generate(ctx, messages, config)
```

### Streaming Responses

```go
eventChan, err := client.GenerateStream(ctx, messages, nil)
if err != nil {
    // Handle error
}

for event := range eventChan {
    switch event.Type {
    case llm.StreamEventTypeStart:
        fmt.Println("Stream started")

    case llm.StreamEventTypeDelta:
        if event.Delta != nil && event.Delta.Text != "" {
            fmt.Print(event.Delta.Text)
        }

    case llm.StreamEventTypeContent:
        // Final content
        fmt.Println("\nComplete response:", event.Content)

    case llm.StreamEventTypeError:
        fmt.Println("Error:", event.Error)

    case llm.StreamEventTypeDone:
        fmt.Println("Stream complete")
    }
}
```

### Tool Calling

#### Define Tools
```go
tools := []llm.ToolDefinition{
    {
        Name:        "get_weather",
        Description: "Get the current weather for a location",
        Parameters: map[string]interface{}{
            "type": "object",
            "properties": map[string]interface{}{
                "location": map[string]interface{}{
                    "type":        "string",
                    "description": "City name",
                },
                "units": map[string]interface{}{
                    "type": "string",
                    "enum": []string{"celsius", "fahrenheit"},
                },
            },
            "required": []string{"location"},
        },
    },
}

config := &llm.GenerateConfig{
    Tools: tools,
}
```

#### Handle Tool Calls
```go
response, err := client.Generate(ctx, messages, config)
if err != nil {
    // Handle error
}

// Check for tool calls
if len(response.ToolCalls) > 0 {
    for _, toolCall := range response.ToolCalls {
        fmt.Printf("Tool: %s\n", toolCall.Name)
        fmt.Printf("Args: %v\n", toolCall.Arguments)

        // Execute tool and get result
        result := executeToolLocally(toolCall.Name, toolCall.Arguments)

        // Add tool result to messages
        messages = append(messages, &converters.Content{
            Role: "user",
            Parts: []*converters.Part{
                {
                    Type: converters.PartTypeFunctionResponse,
                    Data: &converters.FunctionResponseData{
                        Name:     toolCall.Name,
                        Response: result,
                        ID:       toolCall.ID,
                    },
                },
            },
        })
    }

    // Continue conversation with tool results
    response, err = client.Generate(ctx, messages, config)
}
```

## Types

### GenerateConfig
Configuration for LLM generation.

```go
type GenerateConfig struct {
    Temperature   *float64
    MaxTokens     *int
    TopP          *float64
    TopK          *int
    StopSequences []string
    Tools         []ToolDefinition
    Metadata      map[string]interface{}
}
```

### Response
LLM response with content and metadata.

```go
type Response struct {
    Content      *converters.Content
    StopReason   string
    Usage        *Usage
    ToolCalls    []ToolCall
    FinishReason string
}
```

### StreamEvent
Streaming response event.

```go
type StreamEvent struct {
    Type    StreamEventType
    Content *converters.Content
    Delta   *ContentDelta
    Error   error
    Done    bool
}
```

### ToolDefinition
Tool definition for function calling.

```go
type ToolDefinition struct {
    Name        string
    Description string
    Parameters  map[string]interface{} // JSON Schema
}
```

### ToolCall
Tool call from LLM.

```go
type ToolCall struct {
    ID        string
    Name      string
    Arguments map[string]interface{}
}
```

## Provider-Specific Features

### OpenAI
- Supports function calling with strict schemas
- Streaming with delta updates
- Vision capabilities (future)
- Compatible with Azure OpenAI

### Anthropic
- Tool use with input validation
- Long context windows (up to 200k tokens)
- System message support
- Streaming with message deltas

### Gemini
- Native Google AI integration via google.golang.org/genai
- Function declarations for tool calling
- Streaming with delta updates
- Multimodal capabilities
- Supports both Gemini API and Vertex AI

## Error Handling

All clients return errors wrapped with ADK error codes:

```go
response, err := client.Generate(ctx, messages, config)
if err != nil {
    if adkErr, ok := err.(*apperrors.AppError); ok {
        switch adkErr.Code {
        case apperrors.ErrCodeExecutorFailed:
            // Handle LLM API failure
        case apperrors.ErrCodeAgentConfig:
            // Handle configuration error
        }
    }
}
```

## Best Practices

1. **Always use context**: Pass context for cancellation and timeouts
2. **Handle tool calls**: Check for tool calls in responses and handle them appropriately
3. **Set reasonable limits**: Use MaxTokens to control cost and latency
4. **Cache clients**: Reuse client instances instead of creating new ones per request
5. **Monitor usage**: Track token usage via Response.Usage
6. **Handle errors gracefully**: LLM calls can fail, implement retry logic

## Testing

Mock clients for testing:

```go
type MockClient struct {
    GenerateFunc func(context.Context, []*converters.Content, *GenerateConfig) (*Response, error)
}

func (m *MockClient) Generate(ctx context.Context, messages []*converters.Content, config *GenerateConfig) (*Response, error) {
    if m.GenerateFunc != nil {
        return m.GenerateFunc(ctx, messages, config)
    }
    return &Response{
        Content: &converters.Content{
            Role: "assistant",
            Parts: []*converters.Part{
                {Type: converters.PartTypeText, Data: &converters.TextPartData{Text: "Mock response"}},
            },
        },
    }, nil
}

func (m *MockClient) GenerateStream(ctx context.Context, messages []*converters.Content, config *GenerateConfig) (<-chan *StreamEvent, error) {
    events := make(chan *StreamEvent)
    go func() {
        defer close(events)
        events <- &StreamEvent{Type: llm.StreamEventTypeDone, Done: true}
    }()
    return events, nil
}

func (m *MockClient) SupportsTools() bool { return true }
func (m *MockClient) ModelName() string { return "mock-model" }
```

## Future Enhancements

- [x] Gemini implementation (completed)
- [ ] Vision/multimodal support (partially available in Gemini)
- [ ] Batch API support
- [ ] Fine-tuned model support
- [ ] Cost tracking and optimization
- [ ] Response caching
- [ ] Rate limiting and retry logic
- [ ] Prompt templates and management
