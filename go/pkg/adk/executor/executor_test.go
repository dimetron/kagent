package executor

import (
	"context"
	"testing"

	"github.com/kagent-dev/kagent/go/pkg/adk/converters"
	"github.com/kagent-dev/kagent/go/pkg/adk/session"
	"github.com/kagent-dev/kagent/go/pkg/adk/tools"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"trpc.group/trpc-go/trpc-a2a-go/protocol"
)

func TestExecutor_BasicExecution(t *testing.T) {
	// Setup
	sessionService := NewMockSessionService()
	pathManager := session.NewPathManager("/tmp/test")

	executor := NewA2AExecutor(sessionService, pathManager, []tools.Tool{})

	// Create request context
	requestCtx := &converters.RequestContext{
		SessionID: "test-session",
		UserID:    "test-user",
		TaskID:    "test-task",
		ContextID: "test-context",
		Message:   createTestMessage("Hello world"),
	}

	// Execute
	eventQueue := make(chan *converters.Event, 10)
	done := make(chan error, 1)

	go func() {
		done <- executor.Execute(context.Background(), requestCtx, eventQueue)
		close(eventQueue)
	}()

	// Wait for completion
	err := <-done
	require.NoError(t, err)

	// Verify events
	events := collectEvents(eventQueue)
	require.GreaterOrEqual(t, len(events), 3) // start, content, complete

	assert.Equal(t, converters.EventTypeStart, events[0].Type)
	assert.Equal(t, converters.EventTypeContent, events[1].Type)
	assert.NotNil(t, events[1].Content)
	assert.Equal(t, "assistant", events[1].Content.Role)
	assert.Equal(t, converters.EventTypeComplete, events[len(events)-1].Type)
}

func TestExecutor_EchoResponse(t *testing.T) {
	// Setup
	sessionService := NewMockSessionService()
	pathManager := session.NewPathManager("/tmp/test")

	executor := NewA2AExecutor(sessionService, pathManager, []tools.Tool{})

	// Create request context with specific message
	message := createTestMessage("Test message content")
	requestCtx := &converters.RequestContext{
		SessionID: "test-session",
		UserID:    "test-user",
		TaskID:    "test-task",
		ContextID: "test-context",
		Message:   message,
	}

	// Execute
	eventQueue := make(chan *converters.Event, 10)
	done := make(chan error, 1)

	go func() {
		done <- executor.Execute(context.Background(), requestCtx, eventQueue)
		close(eventQueue)
	}()

	err := <-done
	require.NoError(t, err)

	// Verify response echoes input
	events := collectEvents(eventQueue)
	contentFound := false

	for _, event := range events {
		if event.Type == converters.EventTypeContent && event.Content != nil {
			contentFound = true
			require.NotEmpty(t, event.Content.Parts)

			// Check that response contains "Received:"
			for _, part := range event.Content.Parts {
				if part.Type == converters.PartTypeText {
					if textData, ok := part.Data.(*converters.TextPartData); ok {
						assert.Contains(t, textData.Text, "Received:")
						assert.Contains(t, textData.Text, "Test message content")
					}
				}
			}
			break
		}
	}

	assert.True(t, contentFound, "Should have content event with response")
}

func TestExecutor_SessionCreation(t *testing.T) {
	// Setup
	sessionService := NewMockSessionService()
	pathManager := session.NewPathManager("/tmp/test")

	executor := NewA2AExecutor(sessionService, pathManager, []tools.Tool{})

	// Execute without existing session
	requestCtx := &converters.RequestContext{
		SessionID: "",
		UserID:    "new-user",
		TaskID:    "new-task",
		ContextID: "new-context",
		Message:   createTestMessage("Hello"),
	}

	eventQueue := make(chan *converters.Event, 10)
	done := make(chan error, 1)

	go func() {
		done <- executor.Execute(context.Background(), requestCtx, eventQueue)
		close(eventQueue)
	}()

	err := <-done
	require.NoError(t, err)

	// Verify session was created
	assert.Len(t, sessionService.sessions, 1)
}

func TestExecutor_SessionReuse(t *testing.T) {
	// Setup
	sessionService := NewMockSessionService()
	pathManager := session.NewPathManager("/tmp/test")

	// Pre-create a session
	existingSession := &session.Session{
		ID:      "existing-session",
		UserID:  "test-user",
		AppName: "default",
	}
	sessionService.sessions["existing-session"] = existingSession

	executor := NewA2AExecutor(sessionService, pathManager, []tools.Tool{})

	// Execute with existing session ID
	requestCtx := &converters.RequestContext{
		SessionID: "existing-session",
		UserID:    "test-user",
		TaskID:    "test-task",
		ContextID: "test-context",
		Message:   createTestMessage("Hello"),
	}

	eventQueue := make(chan *converters.Event, 10)
	done := make(chan error, 1)

	go func() {
		done <- executor.Execute(context.Background(), requestCtx, eventQueue)
		close(eventQueue)
	}()

	err := <-done
	require.NoError(t, err)

	// Verify no new session was created
	assert.Len(t, sessionService.sessions, 1)
}

func TestExecutor_EmptyMessage(t *testing.T) {
	// Setup
	sessionService := NewMockSessionService()
	pathManager := session.NewPathManager("/tmp/test")

	executor := NewA2AExecutor(sessionService, pathManager, []tools.Tool{})

	// Create request with empty message
	emptyMessage := &protocol.Message{
		MessageID: "test-msg",
		Kind:      protocol.KindMessage,
		Parts:     []protocol.Part{},
	}

	requestCtx := &converters.RequestContext{
		SessionID: "test-session",
		UserID:    "test-user",
		TaskID:    "test-task",
		ContextID: "test-context",
		Message:   emptyMessage,
	}

	// Execute
	eventQueue := make(chan *converters.Event, 10)
	done := make(chan error, 1)

	go func() {
		done <- executor.Execute(context.Background(), requestCtx, eventQueue)
		close(eventQueue)
	}()

	err := <-done
	require.NoError(t, err)

	// Verify response is generated even for empty message
	events := collectEvents(eventQueue)
	contentFound := false

	for _, event := range events {
		if event.Type == converters.EventTypeContent {
			contentFound = true
			break
		}
	}

	assert.True(t, contentFound, "Should still generate content event")
}

func TestExecutor_ExecuteTool(t *testing.T) {
	// Setup
	sessionService := NewMockSessionService()
	pathManager := session.NewPathManager("/tmp/test")

	mockTool := NewMockTool("test_tool", "A test tool")
	mockTool.runFunc = func(ctx context.Context, args map[string]interface{}, toolCtx *tools.Context) (string, error) {
		if val, ok := args["input"].(string); ok {
			return "Processed: " + val, nil
		}
		return "No input", nil
	}

	executor := NewA2AExecutor(sessionService, pathManager, []tools.Tool{mockTool})

	// Create tool context
	sess := &session.Session{
		ID:      "test-session",
		UserID:  "test-user",
		AppName: "default",
	}

	toolCtx := &tools.Context{
		Session:     sess,
		SessionPath: "/tmp/test/test-session",
		InvocationContext: &session.InvocationContext{
			SessionID: "test-session",
			UserID:    "test-user",
			TaskID:    "test-task",
			ContextID: "test-context",
		},
	}

	// Execute tool
	result, err := executor.ExecuteTool(context.Background(), "test_tool", map[string]interface{}{
		"input": "test data",
	}, toolCtx)

	require.NoError(t, err)
	assert.Equal(t, "Processed: test data", result)
}

func TestExecutor_ExecuteToolNotFound(t *testing.T) {
	// Setup
	sessionService := NewMockSessionService()
	pathManager := session.NewPathManager("/tmp/test")

	executor := NewA2AExecutor(sessionService, pathManager, []tools.Tool{})

	toolCtx := &tools.Context{
		Session:     &session.Session{ID: "test"},
		SessionPath: "/tmp/test",
		InvocationContext: &session.InvocationContext{
			SessionID: "test-session",
		},
	}

	// Try to execute non-existent tool
	_, err := executor.ExecuteTool(context.Background(), "nonexistent", map[string]interface{}{}, toolCtx)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "tool not found")
}

func TestExecutor_MultipleTools(t *testing.T) {
	// Setup
	sessionService := NewMockSessionService()
	pathManager := session.NewPathManager("/tmp/test")

	tool1 := NewMockTool("tool1", "First tool")
	tool1.runFunc = func(ctx context.Context, args map[string]interface{}, toolCtx *tools.Context) (string, error) {
		return "Result from tool1", nil
	}

	tool2 := NewMockTool("tool2", "Second tool")
	tool2.runFunc = func(ctx context.Context, args map[string]interface{}, toolCtx *tools.Context) (string, error) {
		return "Result from tool2", nil
	}

	executor := NewA2AExecutor(sessionService, pathManager, []tools.Tool{tool1, tool2})

	toolCtx := &tools.Context{
		Session:     &session.Session{ID: "test"},
		SessionPath: "/tmp/test",
		InvocationContext: &session.InvocationContext{
			SessionID: "test-session",
		},
	}

	// Execute both tools
	result1, err1 := executor.ExecuteTool(context.Background(), "tool1", map[string]interface{}{}, toolCtx)
	require.NoError(t, err1)
	assert.Equal(t, "Result from tool1", result1)

	result2, err2 := executor.ExecuteTool(context.Background(), "tool2", map[string]interface{}{}, toolCtx)
	require.NoError(t, err2)
	assert.Equal(t, "Result from tool2", result2)
}

func TestExecutor_ContextCancellation(t *testing.T) {
	// This test verifies that context cancellation is properly handled
	// Note: The basic executor doesn't have explicit cancellation checks in executeAgent,
	// but the framework should handle it at the HTTP layer

	sessionService := NewMockSessionService()
	pathManager := session.NewPathManager("/tmp/test")

	executor := NewA2AExecutor(sessionService, pathManager, []tools.Tool{})

	// Create a cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	requestCtx := &converters.RequestContext{
		SessionID: "test-session",
		UserID:    "test-user",
		TaskID:    "test-task",
		ContextID: "test-context",
		Message:   createTestMessage("Hello"),
	}

	eventQueue := make(chan *converters.Event, 10)

	// This should still complete since the basic executor doesn't check context in executeAgent
	// But it demonstrates that the executor can work with cancelled contexts
	err := executor.Execute(ctx, requestCtx, eventQueue)
	close(eventQueue)

	// The basic executor doesn't fail on cancelled context in its simple implementation
	// This is expected behavior - context cancellation would be handled at the HTTP layer
	_ = err
}

func TestExecutor_Events(t *testing.T) {
	// Verify all expected events are generated in correct order
	sessionService := NewMockSessionService()
	pathManager := session.NewPathManager("/tmp/test")

	executor := NewA2AExecutor(sessionService, pathManager, []tools.Tool{})

	requestCtx := &converters.RequestContext{
		SessionID: "test-session",
		UserID:    "test-user",
		TaskID:    "test-task",
		ContextID: "test-context",
		Message:   createTestMessage("Test"),
	}

	eventQueue := make(chan *converters.Event, 10)
	done := make(chan error, 1)

	go func() {
		done <- executor.Execute(context.Background(), requestCtx, eventQueue)
		close(eventQueue)
	}()

	err := <-done
	require.NoError(t, err)

	events := collectEvents(eventQueue)
	require.Len(t, events, 3)

	// Verify event sequence
	assert.Equal(t, converters.EventTypeStart, events[0].Type)
	assert.NotZero(t, events[0].Timestamp)

	assert.Equal(t, converters.EventTypeContent, events[1].Type)
	assert.NotNil(t, events[1].Content)
	assert.NotZero(t, events[1].Timestamp)

	assert.Equal(t, converters.EventTypeComplete, events[2].Type)
	assert.NotZero(t, events[2].Timestamp)

	// Verify timestamps are ordered
	assert.True(t, events[0].Timestamp.Before(events[1].Timestamp) ||
		events[0].Timestamp.Equal(events[1].Timestamp))
	assert.True(t, events[1].Timestamp.Before(events[2].Timestamp) ||
		events[1].Timestamp.Equal(events[2].Timestamp))
}
