package converters

import (
	"testing"

	"trpc.group/trpc-go/trpc-a2a-go/protocol"
)

// Benchmark part conversion from A2A to Content

func BenchmarkPartConverter_A2AToContent_Text(b *testing.B) {
	converter := NewPartConverter()
	a2aParts := []protocol.Part{
		&protocol.TextPart{Text: "Hello, this is a test message with some reasonable length to simulate real usage patterns."},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := converter.ConvertA2AToContent(a2aParts)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkPartConverter_A2AToContent_MultipleParts(b *testing.B) {
	converter := NewPartConverter()
	a2aParts := []protocol.Part{
		&protocol.TextPart{Text: "Part 1"},
		&protocol.TextPart{Text: "Part 2"},
		&protocol.FilePart{URI: "file:///path/to/file.txt"},
		&protocol.TextPart{Text: "Part 3"},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := converter.ConvertA2AToContent(a2aParts)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkPartConverter_A2AToContent_FunctionCall(b *testing.B) {
	converter := NewPartConverter()
	a2aParts := []protocol.Part{
		&protocol.FunctionCallPart{
			Name: "test_function",
			Args: map[string]interface{}{
				"arg1": "value1",
				"arg2": 123,
				"arg3": map[string]interface{}{
					"nested": "value",
				},
			},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := converter.ConvertA2AToContent(a2aParts)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// Benchmark part conversion from Content to A2A

func BenchmarkPartConverter_ContentToA2A_Text(b *testing.B) {
	converter := NewPartConverter()
	parts := []*Part{
		{
			Type: PartTypeText,
			Data: &TextPartData{Text: "Hello, this is a test message with some reasonable length to simulate real usage patterns."},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := converter.ConvertContentToA2A(parts)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkPartConverter_ContentToA2A_MultipleParts(b *testing.B) {
	converter := NewPartConverter()
	parts := []*Part{
		{
			Type: PartTypeText,
			Data: &TextPartData{Text: "Part 1"},
		},
		{
			Type: PartTypeText,
			Data: &TextPartData{Text: "Part 2"},
		},
		{
			Type: PartTypeFile,
			Data: &FilePartData{URI: "file:///path/to/file.txt"},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := converter.ConvertContentToA2A(parts)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// Benchmark round-trip conversion

func BenchmarkPartConverter_RoundTrip(b *testing.B) {
	converter := NewPartConverter()
	a2aParts := []protocol.Part{
		&protocol.TextPart{Text: "Hello world"},
		&protocol.FunctionCallPart{
			Name: "test_func",
			Args: map[string]interface{}{"key": "value"},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		parts, err := converter.ConvertA2AToContent(a2aParts)
		if err != nil {
			b.Fatal(err)
		}
		_, err = converter.ConvertContentToA2A(parts)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// Benchmark request conversion

func BenchmarkRequestConverter_Convert(b *testing.B) {
	converter := NewRequestConverter()
	requestCtx := &RequestContext{
		SessionID: "session-123",
		UserID:    "user-456",
		TaskID:    "task-789",
		ContextID: "context-abc",
		Message: &protocol.Message{
			MessageID: "msg-123",
			Kind:      protocol.KindMessage,
			Parts: []protocol.Part{
				&protocol.TextPart{Text: "Hello, how are you doing today? I need some help with a complex task."},
			},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := converter.Convert(requestCtx)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkRequestConverter_Convert_WithHistory(b *testing.B) {
	converter := NewRequestConverter()
	history := []*protocol.Message{
		{
			MessageID: "msg-1",
			Kind:      protocol.KindMessage,
			Parts:     []protocol.Part{&protocol.TextPart{Text: "Previous message 1"}},
		},
		{
			MessageID: "msg-2",
			Kind:      protocol.KindMessage,
			Parts:     []protocol.Part{&protocol.TextPart{Text: "Previous message 2"}},
		},
	}

	requestCtx := &RequestContext{
		SessionID:      "session-123",
		UserID:         "user-456",
		TaskID:         "task-789",
		ContextID:      "context-abc",
		MessageHistory: history,
		Message: &protocol.Message{
			MessageID: "msg-3",
			Kind:      protocol.KindMessage,
			Parts:     []protocol.Part{&protocol.TextPart{Text: "Current message"}},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := converter.Convert(requestCtx)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// Benchmark event conversion

func BenchmarkEventConverter_Convert_Content(b *testing.B) {
	converter := NewEventConverter()
	event := &Event{
		Type: EventTypeContent,
		Content: &Content{
			Role: "assistant",
			Parts: []*Part{
				{
					Type: PartTypeText,
					Data: &TextPartData{Text: "This is a response from the assistant with some reasonable content length."},
				},
			},
		},
	}

	invCtx := &InvocationContext{
		SessionID: "session-123",
		UserID:    "user-456",
		TaskID:    "task-789",
		ContextID: "context-abc",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := converter.Convert(event, invCtx, "task-789", "context-abc")
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkEventConverter_Convert_Error(b *testing.B) {
	converter := NewEventConverter()
	event := &Event{
		Type: EventTypeError,
		Error: &ErrorInfo{
			Code:    "TEST_ERROR",
			Message: "Test error message",
			Details: "Detailed error information goes here",
		},
	}

	invCtx := &InvocationContext{
		SessionID: "session-123",
		UserID:    "user-456",
		TaskID:    "task-789",
		ContextID: "context-abc",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := converter.Convert(event, invCtx, "task-789", "context-abc")
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkEventConverter_Convert_StateUpdate(b *testing.B) {
	converter := NewEventConverter()
	event := &Event{
		Type: EventTypeStateUpdate,
		Metadata: map[string]interface{}{
			"state":   "WORKING",
			"tool_id": "tool-123",
		},
	}

	invCtx := &InvocationContext{
		SessionID: "session-123",
		UserID:    "user-456",
		TaskID:    "task-789",
		ContextID: "context-abc",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := converter.Convert(event, invCtx, "task-789", "context-abc")
		if err != nil {
			b.Fatal(err)
		}
	}
}

// Parallel benchmarks

func BenchmarkPartConverter_A2AToContent_Parallel(b *testing.B) {
	converter := NewPartConverter()
	a2aParts := []protocol.Part{
		&protocol.TextPart{Text: "Hello, this is a test message"},
		&protocol.TextPart{Text: "Another part"},
	}

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := converter.ConvertA2AToContent(a2aParts)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

func BenchmarkEventConverter_Convert_Parallel(b *testing.B) {
	converter := NewEventConverter()
	event := &Event{
		Type: EventTypeContent,
		Content: &Content{
			Role: "assistant",
			Parts: []*Part{
				{
					Type: PartTypeText,
					Data: &TextPartData{Text: "Response"},
				},
			},
		},
	}

	invCtx := &InvocationContext{
		SessionID: "session-123",
		UserID:    "user-456",
		TaskID:    "task-789",
		ContextID: "context-abc",
	}

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := converter.Convert(event, invCtx, "task-789", "context-abc")
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}
