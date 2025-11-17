package activities

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/testsuite"

	"github.com/kagent-dev/kagent/go/internal/executor/temporal/llm"
	"github.com/kagent-dev/kagent/go/internal/executor/temporal/models"
)

// MockLLMProvider is a mock implementation of LLM provider
type MockLLMProvider struct {
	mock.Mock
}

func (m *MockLLMProvider) Chat(ctx context.Context, request models.LLMRequest) (*models.LLMResponse, error) {
	args := m.Called(ctx, request)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.LLMResponse), args.Error(1)
}

func (m *MockLLMProvider) ChatStream(ctx context.Context, request models.LLMRequest) (<-chan llm.StreamChunk, <-chan error) {
	args := m.Called(ctx, request)
	return args.Get(0).(<-chan llm.StreamChunk), args.Get(1).(<-chan error)
}

func (m *MockLLMProvider) Name() string {
	return "mock"
}

func (m *MockLLMProvider) SupportedModels() []string {
	return []string{"mock-model"}
}

// MockEventPublisher is a mock implementation of EventPublisher
type MockEventPublisher struct {
	mock.Mock
}

func (m *MockEventPublisher) PublishEvent(ctx context.Context, event models.A2AEvent) error {
	args := m.Called(ctx, event)
	return args.Error(0)
}

// MockToolExecutor is a mock implementation of ToolExecutor
type MockToolExecutor struct {
	mock.Mock
}

func (m *MockToolExecutor) ExecuteTool(ctx context.Context, toolCall models.ToolCall, toolDef models.Tool) (string, error) {
	args := m.Called(ctx, toolCall, toolDef)
	return args.String(0), args.Error(1)
}

func TestInvokeLLMActivity_Success(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestActivityEnvironment()

	// Create mock provider
	mockProvider := new(MockLLMProvider)
	expectedResponse := &models.LLMResponse{
		Content:      "Hello, world!",
		FinishReason: "stop",
		TokenUsage: models.TokenUsage{
			PromptTokens:     10,
			CompletionTokens: 5,
			TotalTokens:      15,
		},
		ModelUsed: "mock-model",
	}

	mockProvider.On("Chat", mock.Anything, mock.Anything).Return(expectedResponse, nil)

	// Create registry and register mock provider
	registry := llm.NewRegistry()
	require.NoError(t, registry.Register(mockProvider))

	// Create activities with mocks
	mockEventPublisher := new(MockEventPublisher)
	mockToolExecutor := new(MockToolExecutor)
	activities := NewActivities(registry, mockEventPublisher, mockToolExecutor)

	// Register activity
	env.RegisterActivity(activities.InvokeLLMActivity)

	// Create request
	request := models.LLMRequest{
		Messages: []models.Message{
			{Role: "user", Content: "Hello"},
		},
		ModelConfig: models.ModelConfig{
			Provider: "mock",
			Model:    "mock-model",
		},
	}

	// Execute activity
	val, err := env.ExecuteActivity(activities.InvokeLLMActivity, request)
	require.NoError(t, err)

	// Get result
	var response models.LLMResponse
	require.NoError(t, val.Get(&response))

	// Assertions
	require.Equal(t, "Hello, world!", response.Content)
	require.Equal(t, "stop", response.FinishReason)
	require.Equal(t, 15, response.TokenUsage.TotalTokens)
	mockProvider.AssertExpectations(t)
}

func TestInvokeLLMActivity_ProviderNotFound(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestActivityEnvironment()

	// Create empty registry
	registry := llm.NewRegistry()
	mockEventPublisher := new(MockEventPublisher)
	mockToolExecutor := new(MockToolExecutor)
	activities := NewActivities(registry, mockEventPublisher, mockToolExecutor)

	env.RegisterActivity(activities.InvokeLLMActivity)

	// Create request with non-existent provider
	request := models.LLMRequest{
		Messages: []models.Message{
			{Role: "user", Content: "Hello"},
		},
		ModelConfig: models.ModelConfig{
			Provider: "nonexistent",
			Model:    "some-model",
		},
	}

	// Execute activity
	_, err := env.ExecuteActivity(activities.InvokeLLMActivity, request)
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to get LLM provider")
}

func TestExecuteToolActivity_Success(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestActivityEnvironment()

	// Create mock tool executor
	mockToolExecutor := new(MockToolExecutor)
	mockToolExecutor.On("ExecuteTool", mock.Anything, mock.Anything, mock.Anything).Return(
		`{"result": "success"}`, nil,
	)

	mockEventPublisher := new(MockEventPublisher)
	registry := llm.NewRegistry()
	activities := NewActivities(registry, mockEventPublisher, mockToolExecutor)

	env.RegisterActivity(activities.ExecuteToolActivity)

	// Create input
	input := ToolExecutionInput{
		ToolCall: models.ToolCall{
			ID:        "call_1",
			Name:      "test_tool",
			Arguments: map[string]interface{}{"arg": "value"},
		},
		ToolDef: models.Tool{
			Name: "test_tool",
			Type: "http",
		},
	}

	// Execute activity
	val, err := env.ExecuteActivity(activities.ExecuteToolActivity, input)
	require.NoError(t, err)

	// Get result
	var result models.ToolCall
	require.NoError(t, val.Get(&result))

	// Assertions
	require.Equal(t, "call_1", result.ID)
	require.Equal(t, "completed", result.Status)
	require.Equal(t, `{"result": "success"}`, result.Result)
	require.Empty(t, result.Error)
	mockToolExecutor.AssertExpectations(t)
}

func TestExecuteToolActivity_Failure(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestActivityEnvironment()

	// Create mock tool executor that returns error
	mockToolExecutor := new(MockToolExecutor)
	mockToolExecutor.On("ExecuteTool", mock.Anything, mock.Anything, mock.Anything).Return(
		"", mock.AnythingOfType("*errors.errorString"),
	)

	mockEventPublisher := new(MockEventPublisher)
	registry := llm.NewRegistry()
	activities := NewActivities(registry, mockEventPublisher, mockToolExecutor)

	env.RegisterActivity(activities.ExecuteToolActivity)

	// Create input
	input := ToolExecutionInput{
		ToolCall: models.ToolCall{
			ID:   "call_1",
			Name: "failing_tool",
		},
		ToolDef: models.Tool{
			Name: "failing_tool",
			Type: "http",
		},
	}

	// Execute activity
	val, err := env.ExecuteActivity(activities.ExecuteToolActivity, input)
	require.NoError(t, err) // Activity itself doesn't error, it returns failed tool call

	// Get result
	var result models.ToolCall
	require.NoError(t, val.Get(&result))

	// Assertions
	require.Equal(t, "failed", result.Status)
	require.NotEmpty(t, result.Error)
}

func TestPublishEventActivity_Success(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestActivityEnvironment()

	// Create mock event publisher
	mockEventPublisher := new(MockEventPublisher)
	mockEventPublisher.On("PublishEvent", mock.Anything, mock.Anything).Return(nil)

	registry := llm.NewRegistry()
	mockToolExecutor := new(MockToolExecutor)
	activities := NewActivities(registry, mockEventPublisher, mockToolExecutor)

	env.RegisterActivity(activities.PublishEventActivity)

	// Create event
	event := models.A2AEvent{
		TaskID:    "task-1",
		EventType: "status_update",
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"status": models.TaskStatusWorking,
		},
	}

	// Execute activity
	_, err := env.ExecuteActivity(activities.PublishEventActivity, event)
	require.NoError(t, err)
	mockEventPublisher.AssertExpectations(t)
}

func TestCancelExecutionActivity_Success(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestActivityEnvironment()

	// Create mock event publisher
	mockEventPublisher := new(MockEventPublisher)
	mockEventPublisher.On("PublishEvent", mock.Anything, mock.MatchedBy(func(event models.A2AEvent) bool {
		return event.EventType == "status_update" &&
			event.Data["status"] == models.TaskStatusCancelled
	})).Return(nil)

	registry := llm.NewRegistry()
	mockToolExecutor := new(MockToolExecutor)
	activities := NewActivities(registry, mockEventPublisher, mockToolExecutor)

	env.RegisterActivity(activities.CancelExecutionActivity)

	// Execute activity
	_, err := env.ExecuteActivity(activities.CancelExecutionActivity, "task-1")
	require.NoError(t, err)
	mockEventPublisher.AssertExpectations(t)
}

func TestSaveStateActivity(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestActivityEnvironment()

	registry := llm.NewRegistry()
	mockEventPublisher := new(MockEventPublisher)
	mockToolExecutor := new(MockToolExecutor)
	activities := NewActivities(registry, mockEventPublisher, mockToolExecutor)

	env.RegisterActivity(activities.SaveStateActivity)

	// Create state
	state := models.ExecutionState{
		TaskID:           "task-1",
		SessionID:        "session-1",
		CurrentIteration: 2,
		MaxIterations:    10,
		Status:           models.TaskStatusWorking,
	}

	// Execute activity (currently just logs, no error expected)
	_, err := env.ExecuteActivity(activities.SaveStateActivity, state)
	require.NoError(t, err)
}

func TestLoadStateActivity(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestActivityEnvironment()

	registry := llm.NewRegistry()
	mockEventPublisher := new(MockEventPublisher)
	mockToolExecutor := new(MockToolExecutor)
	activities := NewActivities(registry, mockEventPublisher, mockToolExecutor)

	env.RegisterActivity(activities.LoadStateActivity)

	// Execute activity (currently returns nil, no saved state)
	val, err := env.ExecuteActivity(activities.LoadStateActivity, "task-1")
	require.NoError(t, err)

	var state *models.ExecutionState
	require.NoError(t, val.Get(&state))
	require.Nil(t, state)
}
