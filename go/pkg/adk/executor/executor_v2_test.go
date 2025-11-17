package executor

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/kagent-dev/kagent/go/pkg/adk/converters"
	"github.com/kagent-dev/kagent/go/pkg/adk/llm"
	"github.com/kagent-dev/kagent/go/pkg/adk/session"
	"github.com/kagent-dev/kagent/go/pkg/adk/tools"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"trpc.group/trpc-go/trpc-a2a-go/protocol"
)

// Mock implementations

type MockSessionService struct {
	sessions map[string]*session.Session
}

func NewMockSessionService() *MockSessionService {
	return &MockSessionService{
		sessions: make(map[string]*session.Session),
	}
}

func (m *MockSessionService) CreateSession(ctx context.Context, req *session.CreateSessionRequest) (*session.Session, error) {
	sess := &session.Session{
		ID:      "test-session-" + req.UserID,
		UserID:  req.UserID,
		AppName: req.AppName,
	}
	m.sessions[sess.ID] = sess
	return sess, nil
}

func (m *MockSessionService) GetSession(ctx context.Context, appName, userID, sessionID string) (*session.Session, error) {
	if sess, ok := m.sessions[sessionID]; ok {
		return sess, nil
	}
	return nil, fmt.Errorf("session not found")
}

func (m *MockSessionService) DeleteSession(ctx context.Context, appName, userID, sessionID string) error {
	delete(m.sessions, sessionID)
	return nil
}

type MockLLMClient struct {
	responses     []*llm.Response
	currentCall   int
	generateFunc  func(ctx context.Context, messages []*converters.Content, config *llm.GenerateConfig) (*llm.Response, error)
	supportsTools bool
	modelName     string
}

func NewMockLLMClient() *MockLLMClient {
	return &MockLLMClient{
		responses:     []*llm.Response{},
		currentCall:   0,
		supportsTools: true,
		modelName:     "mock-model",
	}
}

func (m *MockLLMClient) Generate(ctx context.Context, messages []*converters.Content, config *llm.GenerateConfig) (*llm.Response, error) {
	if m.generateFunc != nil {
		return m.generateFunc(ctx, messages, config)
	}

	if m.currentCall >= len(m.responses) {
		return nil, fmt.Errorf("no more mock responses")
	}

	resp := m.responses[m.currentCall]
	m.currentCall++
	return resp, nil
}

func (m *MockLLMClient) GenerateStream(ctx context.Context, messages []*converters.Content, config *llm.GenerateConfig) (<-chan *llm.StreamEvent, error) {
	events := make(chan *llm.StreamEvent, 1)
	go func() {
		defer close(events)
		events <- &llm.StreamEvent{Type: llm.StreamEventTypeDone, Done: true}
	}()
	return events, nil
}

func (m *MockLLMClient) SupportsTools() bool {
	return m.supportsTools
}

func (m *MockLLMClient) ModelName() string {
	return m.modelName
}

type MockTool struct {
	name        string
	description string
	runFunc     func(ctx context.Context, args map[string]interface{}, toolCtx *tools.Context) (string, error)
}

func NewMockTool(name, description string) *MockTool {
	return &MockTool{
		name:        name,
		description: description,
	}
}

func (m *MockTool) Name() string {
	return m.name
}

func (m *MockTool) Description() string {
	return m.description
}

func (m *MockTool) RunAsync(ctx context.Context, args map[string]interface{}, toolCtx *tools.Context) (string, error) {
	if m.runFunc != nil {
		return m.runFunc(ctx, args, toolCtx)
	}
	return fmt.Sprintf("Mock result from %s", m.name), nil
}

// Helper functions

func createTestMessage(text string) *protocol.Message {
	return &protocol.Message{
		MessageID: "test-msg",
		Kind:      protocol.KindMessage,
		Parts: []protocol.Part{
			&protocol.TextPart{Text: text},
		},
	}
}

func collectEvents(eventQueue <-chan *converters.Event) []*converters.Event {
	var events []*converters.Event
	for event := range eventQueue {
		events = append(events, event)
	}
	return events
}

// Tests

func TestExecutorV2_BasicExecution(t *testing.T) {
	// Setup
	sessionService := NewMockSessionService()
	pathManager := session.NewPathManager("/tmp/test")
	llmClient := NewMockLLMClient()

	// Mock LLM response without tool calls (agent is done)
	llmClient.responses = []*llm.Response{
		{
			Content: &converters.Content{
				Role: "assistant",
				Parts: []*converters.Part{
					{
						Type: converters.PartTypeText,
						Data: &converters.TextPartData{Text: "Hello! How can I help you?"},
					},
				},
			},
			ToolCalls:    []llm.ToolCall{},
			FinishReason: "stop",
		},
	}

	executor := NewA2AExecutorV2(sessionService, pathManager, []tools.Tool{}, llmClient)

	// Create request context
	requestCtx := &converters.RequestContext{
		SessionID: "test-session",
		UserID:    "test-user",
		TaskID:    "test-task",
		ContextID: "test-context",
		Message:   createTestMessage("Hello"),
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
	assert.Equal(t, converters.EventTypeComplete, events[len(events)-1].Type)
}

func TestExecutorV2_WithToolCall(t *testing.T) {
	// Setup
	sessionService := NewMockSessionService()
	pathManager := session.NewPathManager("/tmp/test")
	llmClient := NewMockLLMClient()

	mockTool := NewMockTool("test_tool", "A test tool")
	mockTool.runFunc = func(ctx context.Context, args map[string]interface{}, toolCtx *tools.Context) (string, error) {
		return "Tool executed successfully", nil
	}

	// Mock LLM responses: first with tool call, then without
	llmClient.responses = []*llm.Response{
		{
			Content: &converters.Content{
				Role: "assistant",
				Parts: []*converters.Part{
					{
						Type: converters.PartTypeText,
						Data: &converters.TextPartData{Text: "Let me use a tool"},
					},
				},
			},
			ToolCalls: []llm.ToolCall{
				{
					ID:   "call-1",
					Name: "test_tool",
					Arguments: map[string]interface{}{
						"arg1": "value1",
					},
				},
			},
			FinishReason: "tool_calls",
		},
		{
			Content: &converters.Content{
				Role: "assistant",
				Parts: []*converters.Part{
					{
						Type: converters.PartTypeText,
						Data: &converters.TextPartData{Text: "Tool executed, here's the result"},
					},
				},
			},
			ToolCalls:    []llm.ToolCall{},
			FinishReason: "stop",
		},
	}

	executor := NewA2AExecutorV2(sessionService, pathManager, []tools.Tool{mockTool}, llmClient)

	// Create request context
	requestCtx := &converters.RequestContext{
		SessionID: "test-session",
		UserID:    "test-user",
		TaskID:    "test-task",
		ContextID: "test-context",
		Message:   createTestMessage("Execute a tool"),
	}

	// Execute
	eventQueue := make(chan *converters.Event, 20)
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
	require.GreaterOrEqual(t, len(events), 6) // start, content, tool_call, tool_response, content, complete

	// Check event types
	eventTypes := make([]string, len(events))
	for i, e := range events {
		eventTypes[i] = e.Type
	}

	assert.Contains(t, eventTypes, converters.EventTypeStart)
	assert.Contains(t, eventTypes, converters.EventTypeToolCall)
	assert.Contains(t, eventTypes, converters.EventTypeToolResponse)
	assert.Contains(t, eventTypes, converters.EventTypeComplete)
}

func TestExecutorV2_MultipleToolCalls(t *testing.T) {
	// Setup
	sessionService := NewMockSessionService()
	pathManager := session.NewPathManager("/tmp/test")
	llmClient := NewMockLLMClient()

	tool1 := NewMockTool("tool1", "First tool")
	tool2 := NewMockTool("tool2", "Second tool")

	// Mock LLM responses with multiple tool calls in one response
	llmClient.responses = []*llm.Response{
		{
			Content: &converters.Content{
				Role: "assistant",
				Parts: []*converters.Part{
					{
						Type: converters.PartTypeText,
						Data: &converters.TextPartData{Text: "Using multiple tools"},
					},
				},
			},
			ToolCalls: []llm.ToolCall{
				{ID: "call-1", Name: "tool1", Arguments: map[string]interface{}{}},
				{ID: "call-2", Name: "tool2", Arguments: map[string]interface{}{}},
			},
			FinishReason: "tool_calls",
		},
		{
			Content: &converters.Content{
				Role: "assistant",
				Parts: []*converters.Part{
					{
						Type: converters.PartTypeText,
						Data: &converters.TextPartData{Text: "All tools executed"},
					},
				},
			},
			ToolCalls:    []llm.ToolCall{},
			FinishReason: "stop",
		},
	}

	executor := NewA2AExecutorV2(sessionService, pathManager, []tools.Tool{tool1, tool2}, llmClient)

	// Execute
	requestCtx := &converters.RequestContext{
		SessionID: "test-session",
		UserID:    "test-user",
		TaskID:    "test-task",
		ContextID: "test-context",
		Message:   createTestMessage("Execute multiple tools"),
	}

	eventQueue := make(chan *converters.Event, 30)
	done := make(chan error, 1)

	go func() {
		done <- executor.Execute(context.Background(), requestCtx, eventQueue)
		close(eventQueue)
	}()

	err := <-done
	require.NoError(t, err)

	// Verify tool call events
	events := collectEvents(eventQueue)
	toolCallCount := 0
	toolResponseCount := 0

	for _, event := range events {
		if event.Type == converters.EventTypeToolCall {
			toolCallCount++
		}
		if event.Type == converters.EventTypeToolResponse {
			toolResponseCount++
		}
	}

	assert.Equal(t, 2, toolCallCount, "Should have 2 tool call events")
	assert.Equal(t, 2, toolResponseCount, "Should have 2 tool response events")
}

func TestExecutorV2_MaxIterationsReached(t *testing.T) {
	// Setup
	sessionService := NewMockSessionService()
	pathManager := session.NewPathManager("/tmp/test")
	llmClient := NewMockLLMClient()

	mockTool := NewMockTool("test_tool", "A test tool")

	// Mock LLM to always return tool calls (never stops)
	llmClient.generateFunc = func(ctx context.Context, messages []*converters.Content, config *llm.GenerateConfig) (*llm.Response, error) {
		return &llm.Response{
			Content: &converters.Content{
				Role: "assistant",
				Parts: []*converters.Part{
					{
						Type: converters.PartTypeText,
						Data: &converters.TextPartData{Text: "Calling tool again"},
					},
				},
			},
			ToolCalls: []llm.ToolCall{
				{ID: "call", Name: "test_tool", Arguments: map[string]interface{}{}},
			},
			FinishReason: "tool_calls",
		}, nil
	}

	executor := NewA2AExecutorV2(sessionService, pathManager, []tools.Tool{mockTool}, llmClient)

	// Execute
	requestCtx := &converters.RequestContext{
		SessionID: "test-session",
		UserID:    "test-user",
		TaskID:    "test-task",
		ContextID: "test-context",
		Message:   createTestMessage("Start infinite loop"),
	}

	eventQueue := make(chan *converters.Event, 100)
	done := make(chan error, 1)

	go func() {
		done <- executor.Execute(context.Background(), requestCtx, eventQueue)
		close(eventQueue)
	}()

	err := <-done
	require.Error(t, err)
	assert.Contains(t, err.Error(), "max iterations")

	// Verify error event was sent
	events := collectEvents(eventQueue)
	hasErrorEvent := false
	for _, event := range events {
		if event.Type == converters.EventTypeError {
			hasErrorEvent = true
			break
		}
	}
	assert.True(t, hasErrorEvent, "Should have error event")
}

func TestExecutorV2_ToolExecutionError(t *testing.T) {
	// Setup
	sessionService := NewMockSessionService()
	pathManager := session.NewPathManager("/tmp/test")
	llmClient := NewMockLLMClient()

	mockTool := NewMockTool("failing_tool", "A tool that fails")
	mockTool.runFunc = func(ctx context.Context, args map[string]interface{}, toolCtx *tools.Context) (string, error) {
		return "", fmt.Errorf("tool execution failed")
	}

	// Mock LLM responses
	llmClient.responses = []*llm.Response{
		{
			Content: &converters.Content{
				Role: "assistant",
				Parts: []*converters.Part{
					{
						Type: converters.PartTypeText,
						Data: &converters.TextPartData{Text: "Using failing tool"},
					},
				},
			},
			ToolCalls: []llm.ToolCall{
				{ID: "call-1", Name: "failing_tool", Arguments: map[string]interface{}{}},
			},
			FinishReason: "tool_calls",
		},
		{
			Content: &converters.Content{
				Role: "assistant",
				Parts: []*converters.Part{
					{
						Type: converters.PartTypeText,
						Data: &converters.TextPartData{Text: "Handled the error"},
					},
				},
			},
			ToolCalls:    []llm.ToolCall{},
			FinishReason: "stop",
		},
	}

	executor := NewA2AExecutorV2(sessionService, pathManager, []tools.Tool{mockTool}, llmClient)

	// Execute
	requestCtx := &converters.RequestContext{
		SessionID: "test-session",
		UserID:    "test-user",
		TaskID:    "test-task",
		ContextID: "test-context",
		Message:   createTestMessage("Execute failing tool"),
	}

	eventQueue := make(chan *converters.Event, 20)
	done := make(chan error, 1)

	go func() {
		done <- executor.Execute(context.Background(), requestCtx, eventQueue)
		close(eventQueue)
	}()

	// Should complete successfully (tool errors are handled gracefully)
	err := <-done
	require.NoError(t, err)

	// Verify tool response contains error
	events := collectEvents(eventQueue)
	hasToolError := false

	for _, event := range events {
		if event.Type == converters.EventTypeToolResponse {
			if result, ok := event.Metadata["result"].(string); ok {
				if contains := (result != "" && len(result) > 0); contains {
					hasToolError = true
					break
				}
			}
		}
	}

	assert.True(t, hasToolError, "Should have tool error in response")
}

func TestExecutorV2_ContextCancellation(t *testing.T) {
	// Setup
	sessionService := NewMockSessionService()
	pathManager := session.NewPathManager("/tmp/test")
	llmClient := NewMockLLMClient()

	mockTool := NewMockTool("slow_tool", "A slow tool")
	mockTool.runFunc = func(ctx context.Context, args map[string]interface{}, toolCtx *tools.Context) (string, error) {
		time.Sleep(100 * time.Millisecond)
		return "Done", nil
	}

	// Mock LLM to keep calling tools
	llmClient.generateFunc = func(ctx context.Context, messages []*converters.Content, config *llm.GenerateConfig) (*llm.Response, error) {
		return &llm.Response{
			Content: &converters.Content{
				Role: "assistant",
				Parts: []*converters.Part{
					{
						Type: converters.PartTypeText,
						Data: &converters.TextPartData{Text: "Calling tool"},
					},
				},
			},
			ToolCalls: []llm.ToolCall{
				{ID: "call", Name: "slow_tool", Arguments: map[string]interface{}{}},
			},
			FinishReason: "tool_calls",
		}, nil
	}

	executor := NewA2AExecutorV2(sessionService, pathManager, []tools.Tool{mockTool}, llmClient)

	// Execute with cancellable context
	ctx, cancel := context.WithCancel(context.Background())
	requestCtx := &converters.RequestContext{
		SessionID: "test-session",
		UserID:    "test-user",
		TaskID:    "test-task",
		ContextID: "test-context",
		Message:   createTestMessage("Start slow operation"),
	}

	eventQueue := make(chan *converters.Event, 50)
	done := make(chan error, 1)

	go func() {
		done <- executor.Execute(ctx, requestCtx, eventQueue)
		close(eventQueue)
	}()

	// Cancel after a short delay
	time.Sleep(50 * time.Millisecond)
	cancel()

	// Should return context cancellation error
	err := <-done
	require.Error(t, err)
	assert.Contains(t, err.Error(), "context canceled")
}

func TestExecutorV2_ToolNotFound(t *testing.T) {
	// Setup
	sessionService := NewMockSessionService()
	pathManager := session.NewPathManager("/tmp/test")
	llmClient := NewMockLLMClient()

	// Mock LLM to call non-existent tool
	llmClient.responses = []*llm.Response{
		{
			Content: &converters.Content{
				Role: "assistant",
				Parts: []*converters.Part{
					{
						Type: converters.PartTypeText,
						Data: &converters.TextPartData{Text: "Calling non-existent tool"},
					},
				},
			},
			ToolCalls: []llm.ToolCall{
				{ID: "call-1", Name: "nonexistent_tool", Arguments: map[string]interface{}{}},
			},
			FinishReason: "tool_calls",
		},
		{
			Content: &converters.Content{
				Role: "assistant",
				Parts: []*converters.Part{
					{
						Type: converters.PartTypeText,
						Data: &converters.TextPartData{Text: "Handled missing tool"},
					},
				},
			},
			ToolCalls:    []llm.ToolCall{},
			FinishReason: "stop",
		},
	}

	executor := NewA2AExecutorV2(sessionService, pathManager, []tools.Tool{}, llmClient)

	// Execute
	requestCtx := &converters.RequestContext{
		SessionID: "test-session",
		UserID:    "test-user",
		TaskID:    "test-task",
		ContextID: "test-context",
		Message:   createTestMessage("Call missing tool"),
	}

	eventQueue := make(chan *converters.Event, 20)
	done := make(chan error, 1)

	go func() {
		done <- executor.Execute(context.Background(), requestCtx, eventQueue)
		close(eventQueue)
	}()

	// Should complete (tool not found is handled gracefully)
	err := <-done
	require.NoError(t, err)
}

func TestExecutorV2_BuildToolDefinitions(t *testing.T) {
	// Setup
	sessionService := NewMockSessionService()
	pathManager := session.NewPathManager("/tmp/test")
	llmClient := NewMockLLMClient()

	tool1 := NewMockTool("tool1", "First tool")
	tool2 := NewMockTool("tool2", "Second tool")

	executor := NewA2AExecutorV2(sessionService, pathManager, []tools.Tool{tool1, tool2}, llmClient)

	// Build definitions
	defs := executor.buildToolDefinitions()

	assert.Len(t, defs, 2)
	assert.Equal(t, "tool1", defs[0].Name)
	assert.Equal(t, "First tool", defs[0].Description)
	assert.Equal(t, "tool2", defs[1].Name)
	assert.Equal(t, "Second tool", defs[1].Description)
}

func TestExecutorV2_SessionCreation(t *testing.T) {
	// Setup
	sessionService := NewMockSessionService()
	pathManager := session.NewPathManager("/tmp/test")
	llmClient := NewMockLLMClient()

	llmClient.responses = []*llm.Response{
		{
			Content: &converters.Content{
				Role: "assistant",
				Parts: []*converters.Part{
					{
						Type: converters.PartTypeText,
						Data: &converters.TextPartData{Text: "Response"},
					},
				},
			},
			ToolCalls:    []llm.ToolCall{},
			FinishReason: "stop",
		},
	}

	executor := NewA2AExecutorV2(sessionService, pathManager, []tools.Tool{}, llmClient)

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
