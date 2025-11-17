package executor

import (
	"context"
	"testing"

	"github.com/kagent-dev/kagent/go/pkg/adk/converters"
	"github.com/kagent-dev/kagent/go/pkg/adk/llm"
	"github.com/kagent-dev/kagent/go/pkg/adk/session"
	"github.com/kagent-dev/kagent/go/pkg/adk/tools"
	"trpc.group/trpc-go/trpc-a2a-go/protocol"
)

// Benchmark tool execution

func BenchmarkExecutor_ExecuteTool(b *testing.B) {
	sessionService := NewMockSessionService()
	pathManager := session.NewPathManager("/tmp/test")

	mockTool := NewMockTool("test_tool", "A test tool")
	mockTool.runFunc = func(ctx context.Context, args map[string]interface{}, toolCtx *tools.Context) (string, error) {
		return "Result", nil
	}

	executor := NewA2AExecutor(sessionService, pathManager, []tools.Tool{mockTool})

	toolCtx := &tools.Context{
		Session:     &session.Session{ID: "test"},
		SessionPath: "/tmp/test",
		InvocationContext: &session.InvocationContext{
			SessionID: "test-session",
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := executor.ExecuteTool(context.Background(), "test_tool", map[string]interface{}{
			"arg": "value",
		}, toolCtx)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkExecutorV2_BuildToolDefinitions(b *testing.B) {
	sessionService := NewMockSessionService()
	pathManager := session.NewPathManager("/tmp/test")
	llmClient := NewMockLLMClient()

	tool1 := NewMockTool("tool1", "First tool")
	tool2 := NewMockTool("tool2", "Second tool")
	tool3 := NewMockTool("tool3", "Third tool")

	executor := NewA2AExecutorV2(sessionService, pathManager, []tools.Tool{tool1, tool2, tool3}, llmClient)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = executor.buildToolDefinitions()
	}
}

func BenchmarkExecutorV2_ExecuteToolCalls_Single(b *testing.B) {
	sessionService := NewMockSessionService()
	pathManager := session.NewPathManager("/tmp/test")
	llmClient := NewMockLLMClient()

	mockTool := NewMockTool("test_tool", "A test tool")
	mockTool.runFunc = func(ctx context.Context, args map[string]interface{}, toolCtx *tools.Context) (string, error) {
		return "Tool executed", nil
	}

	executor := NewA2AExecutorV2(sessionService, pathManager, []tools.Tool{mockTool}, llmClient)

	toolCalls := []llm.ToolCall{
		{
			ID:   "call-1",
			Name: "test_tool",
			Arguments: map[string]interface{}{
				"arg1": "value1",
			},
		},
	}

	toolCtx := &tools.Context{
		Session:     &session.Session{ID: "test"},
		SessionPath: "/tmp/test",
		InvocationContext: &session.InvocationContext{
			SessionID: "test-session",
		},
	}

	eventQueue := make(chan *converters.Event, 10)
	defer close(eventQueue)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := executor.executeToolCalls(context.Background(), toolCalls, toolCtx, eventQueue)
		if err != nil {
			b.Fatal(err)
		}

		// Drain event queue
		for len(eventQueue) > 0 {
			<-eventQueue
		}
	}
}

func BenchmarkExecutorV2_ExecuteToolCalls_Multiple(b *testing.B) {
	sessionService := NewMockSessionService()
	pathManager := session.NewPathManager("/tmp/test")
	llmClient := NewMockLLMClient()

	tool1 := NewMockTool("tool1", "First tool")
	tool2 := NewMockTool("tool2", "Second tool")
	tool3 := NewMockTool("tool3", "Third tool")

	executor := NewA2AExecutorV2(sessionService, pathManager, []tools.Tool{tool1, tool2, tool3}, llmClient)

	toolCalls := []llm.ToolCall{
		{ID: "call-1", Name: "tool1", Arguments: map[string]interface{}{"arg": "val1"}},
		{ID: "call-2", Name: "tool2", Arguments: map[string]interface{}{"arg": "val2"}},
		{ID: "call-3", Name: "tool3", Arguments: map[string]interface{}{"arg": "val3"}},
	}

	toolCtx := &tools.Context{
		Session:     &session.Session{ID: "test"},
		SessionPath: "/tmp/test",
		InvocationContext: &session.InvocationContext{
			SessionID: "test-session",
		},
	}

	eventQueue := make(chan *converters.Event, 50)
	defer close(eventQueue)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := executor.executeToolCalls(context.Background(), toolCalls, toolCtx, eventQueue)
		if err != nil {
			b.Fatal(err)
		}

		// Drain event queue
		for len(eventQueue) > 0 {
			<-eventQueue
		}
	}
}

// Benchmark message building

func BenchmarkExecutorV2_BuildMessageHistory(b *testing.B) {
	sessionService := NewMockSessionService()
	pathManager := session.NewPathManager("/tmp/test")
	llmClient := NewMockLLMClient()

	executor := NewA2AExecutorV2(sessionService, pathManager, []tools.Tool{}, llmClient)

	sess := &session.Session{
		ID:      "test-session",
		UserID:  "test-user",
		AppName: "test-app",
	}

	requestCtx := &converters.RequestContext{
		SessionID: "test-session",
		UserID:    "test-user",
		TaskID:    "test-task",
		ContextID: "test-context",
		Message: &protocol.Message{
			MessageID: "msg-1",
			Kind:      protocol.KindMessage,
			Parts: []protocol.Part{
				&protocol.TextPart{Text: "Hello, how can I help you?"},
			},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := executor.buildMessageHistory(context.Background(), sess, requestCtx)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// Benchmark session operations

func BenchmarkExecutorV2_GetOrCreateSession(b *testing.B) {
	sessionService := NewMockSessionService()
	pathManager := session.NewPathManager("/tmp/test")
	llmClient := NewMockLLMClient()

	executor := NewA2AExecutorV2(sessionService, pathManager, []tools.Tool{}, llmClient)

	requestCtx := &converters.RequestContext{
		SessionID: "test-session",
		UserID:    "test-user",
		TaskID:    "test-task",
		ContextID: "test-context",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := executor.getOrCreateSession(context.Background(), requestCtx)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// Benchmark full execution flow (without LLM call)

func BenchmarkExecutor_Execute_Echo(b *testing.B) {
	sessionService := NewMockSessionService()
	pathManager := session.NewPathManager("/tmp/test")

	executor := NewA2AExecutor(sessionService, pathManager, []tools.Tool{})

	requestCtx := &converters.RequestContext{
		SessionID: "test-session",
		UserID:    "test-user",
		TaskID:    "test-task",
		ContextID: "test-context",
		Message: &protocol.Message{
			MessageID: "msg-1",
			Kind:      protocol.KindMessage,
			Parts: []protocol.Part{
				&protocol.TextPart{Text: "Hello world"},
			},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		eventQueue := make(chan *converters.Event, 10)

		err := executor.Execute(context.Background(), requestCtx, eventQueue)
		close(eventQueue)

		if err != nil {
			b.Fatal(err)
		}

		// Drain event queue
		for range eventQueue {
		}
	}
}

// Parallel benchmarks

func BenchmarkExecutor_ExecuteTool_Parallel(b *testing.B) {
	sessionService := NewMockSessionService()
	pathManager := session.NewPathManager("/tmp/test")

	mockTool := NewMockTool("test_tool", "A test tool")
	mockTool.runFunc = func(ctx context.Context, args map[string]interface{}, toolCtx *tools.Context) (string, error) {
		return "Result", nil
	}

	executor := NewA2AExecutor(sessionService, pathManager, []tools.Tool{mockTool})

	toolCtx := &tools.Context{
		Session:     &session.Session{ID: "test"},
		SessionPath: "/tmp/test",
		InvocationContext: &session.InvocationContext{
			SessionID: "test-session",
		},
	}

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := executor.ExecuteTool(context.Background(), "test_tool", map[string]interface{}{
				"arg": "value",
			}, toolCtx)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

func BenchmarkExecutorV2_BuildToolDefinitions_Parallel(b *testing.B) {
	sessionService := NewMockSessionService()
	pathManager := session.NewPathManager("/tmp/test")
	llmClient := NewMockLLMClient()

	tool1 := NewMockTool("tool1", "First tool")
	tool2 := NewMockTool("tool2", "Second tool")
	tool3 := NewMockTool("tool3", "Third tool")

	executor := NewA2AExecutorV2(sessionService, pathManager, []tools.Tool{tool1, tool2, tool3}, llmClient)

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = executor.buildToolDefinitions()
		}
	})
}

// Memory allocation benchmarks

func BenchmarkExecutorV2_BuildMessageHistory_Allocs(b *testing.B) {
	sessionService := NewMockSessionService()
	pathManager := session.NewPathManager("/tmp/test")
	llmClient := NewMockLLMClient()

	executor := NewA2AExecutorV2(sessionService, pathManager, []tools.Tool{}, llmClient)

	sess := &session.Session{
		ID:      "test-session",
		UserID:  "test-user",
		AppName: "test-app",
	}

	requestCtx := &converters.RequestContext{
		SessionID: "test-session",
		UserID:    "test-user",
		TaskID:    "test-task",
		ContextID: "test-context",
		Message: &protocol.Message{
			MessageID: "msg-1",
			Kind:      protocol.KindMessage,
			Parts: []protocol.Part{
				&protocol.TextPart{Text: "Hello"},
			},
		},
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := executor.buildMessageHistory(context.Background(), sess, requestCtx)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkExecutor_Execute_Echo_Allocs(b *testing.B) {
	sessionService := NewMockSessionService()
	pathManager := session.NewPathManager("/tmp/test")

	executor := NewA2AExecutor(sessionService, pathManager, []tools.Tool{})

	requestCtx := &converters.RequestContext{
		SessionID: "test-session",
		UserID:    "test-user",
		TaskID:    "test-task",
		ContextID: "test-context",
		Message: &protocol.Message{
			MessageID: "msg-1",
			Kind:      protocol.KindMessage,
			Parts: []protocol.Part{
				&protocol.TextPart{Text: "Hello"},
			},
		},
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		eventQueue := make(chan *converters.Event, 10)

		err := executor.Execute(context.Background(), requestCtx, eventQueue)
		close(eventQueue)

		if err != nil {
			b.Fatal(err)
		}

		// Drain event queue
		for range eventQueue {
		}
	}
}
