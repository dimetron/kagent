package llm

import (
	"testing"

	"github.com/kagent-dev/kagent/go/pkg/adk/converters"
)

// Benchmark message conversion for OpenAI

func BenchmarkOpenAI_ConvertMessages_Single(b *testing.B) {
	client := &OpenAIClient{}
	messages := []*converters.Content{
		{
			Role: "user",
			Parts: []*converters.Part{
				{
					Type: converters.PartTypeText,
					Data: &converters.TextPartData{Text: "Hello, how can you help me today?"},
				},
			},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = client.convertMessages(messages)
	}
}

func BenchmarkOpenAI_ConvertMessages_Conversation(b *testing.B) {
	client := &OpenAIClient{}
	messages := []*converters.Content{
		{
			Role: "user",
			Parts: []*converters.Part{
				{Type: converters.PartTypeText, Data: &converters.TextPartData{Text: "What is the weather?"}},
			},
		},
		{
			Role: "assistant",
			Parts: []*converters.Part{
				{Type: converters.PartTypeText, Data: &converters.TextPartData{Text: "I'll check the weather for you."}},
			},
		},
		{
			Role: "user",
			Parts: []*converters.Part{
				{Type: converters.PartTypeText, Data: &converters.TextPartData{Text: "Thanks!"}},
			},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = client.convertMessages(messages)
	}
}

func BenchmarkOpenAI_ConvertMessages_WithFunctionCall(b *testing.B) {
	client := &OpenAIClient{}
	messages := []*converters.Content{
		{
			Role: "user",
			Parts: []*converters.Part{
				{Type: converters.PartTypeText, Data: &converters.TextPartData{Text: "Get the weather"}},
			},
		},
		{
			Role: "assistant",
			Parts: []*converters.Part{
				{
					Type: converters.PartTypeFunctionCall,
					Data: &converters.FunctionCallData{
						Name: "get_weather",
						Arguments: map[string]interface{}{
							"location": "Boston",
							"units":    "celsius",
						},
					},
				},
			},
		},
		{
			Role: "user",
			Parts: []*converters.Part{
				{
					Type: converters.PartTypeFunctionResponse,
					Data: &converters.FunctionResponseData{
						Name:     "get_weather",
						Response: "Temperature: 22°C, Sunny",
					},
				},
			},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = client.convertMessages(messages)
	}
}

// Benchmark message conversion for Anthropic

func BenchmarkAnthropic_ConvertMessages_Single(b *testing.B) {
	client := &AnthropicClient{}
	messages := []*converters.Content{
		{
			Role: "user",
			Parts: []*converters.Part{
				{
					Type: converters.PartTypeText,
					Data: &converters.TextPartData{Text: "Hello, how can you help me today?"},
				},
			},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = client.convertMessages(messages)
	}
}

func BenchmarkAnthropic_ConvertMessages_Conversation(b *testing.B) {
	client := &AnthropicClient{}
	messages := []*converters.Content{
		{
			Role: "user",
			Parts: []*converters.Part{
				{Type: converters.PartTypeText, Data: &converters.TextPartData{Text: "What is the weather?"}},
			},
		},
		{
			Role: "assistant",
			Parts: []*converters.Part{
				{Type: converters.PartTypeText, Data: &converters.TextPartData{Text: "I'll check the weather for you."}},
			},
		},
		{
			Role: "user",
			Parts: []*converters.Part{
				{Type: converters.PartTypeText, Data: &converters.TextPartData{Text: "Thanks!"}},
			},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = client.convertMessages(messages)
	}
}

// Benchmark message conversion for Gemini

func BenchmarkGemini_ConvertMessages_Single(b *testing.B) {
	client := &GeminiClient{}
	messages := []*converters.Content{
		{
			Role: "user",
			Parts: []*converters.Part{
				{
					Type: converters.PartTypeText,
					Data: &converters.TextPartData{Text: "Hello, how can you help me today?"},
				},
			},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = client.convertMessages(messages)
	}
}

func BenchmarkGemini_ConvertMessages_WithFunctionCall(b *testing.B) {
	client := &GeminiClient{}
	messages := []*converters.Content{
		{
			Role: "user",
			Parts: []*converters.Part{
				{Type: converters.PartTypeText, Data: &converters.TextPartData{Text: "Get the weather"}},
			},
		},
		{
			Role: "assistant",
			Parts: []*converters.Part{
				{
					Type: converters.PartTypeFunctionCall,
					Data: &converters.FunctionCallData{
						Name: "get_weather",
						Arguments: map[string]interface{}{
							"location": "Boston",
						},
					},
				},
			},
		},
		{
			Role: "user",
			Parts: []*converters.Part{
				{
					Type: converters.PartTypeFunctionResponse,
					Data: &converters.FunctionResponseData{
						Name:     "get_weather",
						Response: "Temperature: 22°C",
					},
				},
			},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = client.convertMessages(messages)
	}
}

// Benchmark schema conversion for Gemini

func BenchmarkGemini_ConvertSchema_Simple(b *testing.B) {
	client := &GeminiClient{}
	schema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"location": map[string]interface{}{
				"type":        "string",
				"description": "The city name",
			},
		},
		"required": []string{"location"},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = client.convertSchema(schema)
	}
}

func BenchmarkGemini_ConvertSchema_Complex(b *testing.B) {
	client := &GeminiClient{}
	schema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"location": map[string]interface{}{
				"type":        "string",
				"description": "The city name",
			},
			"units": map[string]interface{}{
				"type": "string",
				"enum": []string{"celsius", "fahrenheit"},
			},
			"details": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"temperature": map[string]interface{}{
						"type": "boolean",
					},
					"humidity": map[string]interface{}{
						"type": "boolean",
					},
					"wind": map[string]interface{}{
						"type": "boolean",
					},
				},
			},
		},
		"required": []string{"location"},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = client.convertSchema(schema)
	}
}

// Benchmark tool definition building

func BenchmarkBuildToolDefinitions_Single(b *testing.B) {
	tools := []ToolDefinition{
		{
			Name:        "get_weather",
			Description: "Get the current weather",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"location": map[string]interface{}{
						"type": "string",
					},
				},
			},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = tools
	}
}

func BenchmarkBuildToolDefinitions_Multiple(b *testing.B) {
	tools := []ToolDefinition{
		{
			Name:        "get_weather",
			Description: "Get the current weather",
			Parameters: map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{"location": map[string]interface{}{"type": "string"}},
			},
		},
		{
			Name:        "search_web",
			Description: "Search the web",
			Parameters: map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{"query": map[string]interface{}{"type": "string"}},
			},
		},
		{
			Name:        "read_file",
			Description: "Read a file",
			Parameters: map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{"path": map[string]interface{}{"type": "string"}},
			},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = tools
	}
}

// Parallel benchmarks

func BenchmarkOpenAI_ConvertMessages_Parallel(b *testing.B) {
	client := &OpenAIClient{}
	messages := []*converters.Content{
		{
			Role: "user",
			Parts: []*converters.Part{
				{Type: converters.PartTypeText, Data: &converters.TextPartData{Text: "Hello"}},
			},
		},
	}

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = client.convertMessages(messages)
		}
	})
}

func BenchmarkAnthropic_ConvertMessages_Parallel(b *testing.B) {
	client := &AnthropicClient{}
	messages := []*converters.Content{
		{
			Role: "user",
			Parts: []*converters.Part{
				{Type: converters.PartTypeText, Data: &converters.TextPartData{Text: "Hello"}},
			},
		},
	}

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _ = client.convertMessages(messages)
		}
	})
}

func BenchmarkGemini_ConvertMessages_Parallel(b *testing.B) {
	client := &GeminiClient{}
	messages := []*converters.Content{
		{
			Role: "user",
			Parts: []*converters.Part{
				{Type: converters.PartTypeText, Data: &converters.TextPartData{Text: "Hello"}},
			},
		},
	}

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = client.convertMessages(messages)
		}
	})
}
