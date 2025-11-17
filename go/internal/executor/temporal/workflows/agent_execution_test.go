package workflows

import (
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/testsuite"

	"github.com/kagent-dev/kagent/go/internal/executor/temporal/activities"
	"github.com/kagent-dev/kagent/go/internal/executor/temporal/models"
)

func TestAgentExecutionWorkflow_Success(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestWorkflowEnvironment()

	// Mock LLM activity
	env.OnActivity(activities.InvokeLLMActivity, mock.Anything, mock.Anything).Return(
		&models.LLMResponse{
			Content:      "Hello! I'm a helpful assistant.",
			FinishReason: "stop",
			TokenUsage: models.TokenUsage{
				PromptTokens:     10,
				CompletionTokens: 20,
				TotalTokens:      30,
			},
			ModelUsed: "claude-3-5-sonnet-20241022",
		}, nil,
	)

	// Mock event publishing activity
	env.OnActivity(activities.PublishEventActivity, mock.Anything, mock.Anything).Return(nil)

	// Create input
	input := AgentExecutionInput{
		Request: models.ExecutionRequest{
			TaskID:        "test-task-1",
			SessionID:     "test-session-1",
			UserMessage:   "Hello, how are you?",
			SystemMessage: "You are a helpful assistant.",
			MaxIterations: 5,
			ModelConfig: models.ModelConfig{
				Provider:    "anthropic",
				Model:       "claude-3-5-sonnet-20241022",
				Temperature: 0.7,
				MaxTokens:   1000,
			},
		},
	}

	// Execute workflow
	env.ExecuteWorkflow(ExecuteAgent, input)

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())

	// Get result
	var result AgentExecutionOutput
	require.NoError(t, env.GetWorkflowResult(&result))

	// Assertions
	require.Equal(t, "test-task-1", result.Response.TaskID)
	require.Equal(t, models.TaskStatusCompleted, result.Response.Status)
	require.Equal(t, "Hello! I'm a helpful assistant.", result.Response.Result)
	require.Equal(t, 1, result.Response.Iterations)
	require.Equal(t, 30, result.Response.TokenUsage.TotalTokens)
}

func TestAgentExecutionWorkflow_WithToolCalls(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestWorkflowEnvironment()

	// First LLM call returns tool calls
	env.OnActivity(activities.InvokeLLMActivity, mock.Anything, mock.MatchedBy(func(req models.LLMRequest) bool {
		return len(req.Messages) == 2 // system + user
	})).Return(
		&models.LLMResponse{
			Content:      "Let me check the weather for you.",
			FinishReason: "tool_calls",
			ToolCalls: []models.ToolCall{
				{
					ID:   "call_1",
					Name: "get_weather",
					Arguments: map[string]interface{}{
						"location": "San Francisco",
					},
					Status: "pending",
				},
			},
			TokenUsage: models.TokenUsage{
				PromptTokens:     20,
				CompletionTokens: 30,
				TotalTokens:      50,
			},
		}, nil,
	)

	// Tool execution
	env.OnActivity(activities.ExecuteToolActivity, mock.Anything, mock.Anything).Return(
		&models.ToolCall{
			ID:     "call_1",
			Name:   "get_weather",
			Status: "completed",
			Result: `{"temperature": 72, "conditions": "sunny"}`,
		}, nil,
	)

	// Second LLM call with tool results
	env.OnActivity(activities.InvokeLLMActivity, mock.Anything, mock.MatchedBy(func(req models.LLMRequest) bool {
		return len(req.Messages) > 2 // includes tool result
	})).Return(
		&models.LLMResponse{
			Content:      "The weather in San Francisco is sunny with a temperature of 72°F.",
			FinishReason: "stop",
			TokenUsage: models.TokenUsage{
				PromptTokens:     40,
				CompletionTokens: 20,
				TotalTokens:      60,
			},
		}, nil,
	)

	// Mock event publishing
	env.OnActivity(activities.PublishEventActivity, mock.Anything, mock.Anything).Return(nil)

	// Create input
	input := AgentExecutionInput{
		Request: models.ExecutionRequest{
			TaskID:      "test-task-2",
			SessionID:   "test-session-2",
			UserMessage: "What's the weather in San Francisco?",
			Tools: []models.Tool{
				{
					Name:        "get_weather",
					Description: "Get weather for a location",
					Parameters: map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"location": map[string]string{"type": "string"},
						},
					},
					Type: "http",
				},
			},
			MaxIterations: 5,
			ModelConfig: models.ModelConfig{
				Provider: "anthropic",
				Model:    "claude-3-5-sonnet-20241022",
			},
		},
	}

	// Execute workflow
	env.ExecuteWorkflow(ExecuteAgent, input)

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())

	// Get result
	var result AgentExecutionOutput
	require.NoError(t, env.GetWorkflowResult(&result))

	// Assertions
	require.Equal(t, models.TaskStatusCompleted, result.Response.Status)
	require.Equal(t, 2, result.Response.Iterations)
	require.Equal(t, 110, result.Response.TokenUsage.TotalTokens) // 50 + 60
	require.Contains(t, result.Response.Result, "72°F")
}

func TestAgentExecutionWorkflow_MaxIterations(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestWorkflowEnvironment()

	// Mock LLM to always return tool calls (infinite loop scenario)
	env.OnActivity(activities.InvokeLLMActivity, mock.Anything, mock.Anything).Return(
		&models.LLMResponse{
			Content:      "Still working...",
			FinishReason: "tool_calls",
			ToolCalls: []models.ToolCall{
				{
					ID:     "call_1",
					Name:   "some_tool",
					Status: "pending",
				},
			},
			TokenUsage: models.TokenUsage{TotalTokens: 10},
		}, nil,
	)

	env.OnActivity(activities.ExecuteToolActivity, mock.Anything, mock.Anything).Return(
		&models.ToolCall{
			ID:     "call_1",
			Status: "completed",
			Result: "done",
		}, nil,
	)

	env.OnActivity(activities.PublishEventActivity, mock.Anything, mock.Anything).Return(nil)

	// Create input with max 3 iterations
	input := AgentExecutionInput{
		Request: models.ExecutionRequest{
			TaskID:        "test-task-3",
			UserMessage:   "Test max iterations",
			MaxIterations: 3,
			ModelConfig: models.ModelConfig{
				Provider: "anthropic",
				Model:    "claude-3-5-sonnet-20241022",
			},
			Tools: []models.Tool{
				{Name: "some_tool", Type: "builtin"},
			},
		},
	}

	// Execute workflow
	env.ExecuteWorkflow(ExecuteAgent, input)

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())

	// Get result
	var result AgentExecutionOutput
	require.NoError(t, env.GetWorkflowResult(&result))

	// Should stop at max iterations
	require.Equal(t, 3, result.Response.Iterations)
	require.Equal(t, models.TaskStatusCompleted, result.Response.Status)
}

func TestAgentExecutionWorkflow_WithApproval(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestWorkflowEnvironment()

	// Mock LLM to return tool calls
	env.OnActivity(activities.InvokeLLMActivity, mock.Anything, mock.Anything).Return(
		&models.LLMResponse{
			Content:      "I'll execute that command.",
			FinishReason: "tool_calls",
			ToolCalls: []models.ToolCall{
				{
					ID:        "call_1",
					Name:      "execute_command",
					Arguments: map[string]interface{}{"cmd": "rm -rf /"},
					Status:    "pending",
				},
			},
			TokenUsage: models.TokenUsage{TotalTokens: 20},
		}, nil,
	).Once()

	env.OnActivity(activities.PublishEventActivity, mock.Anything, mock.Anything).Return(nil)

	// Register delayed callback to send approval signal
	env.RegisterDelayedCallback(func() {
		env.SignalWorkflow("tool-approval", true)
	}, time.Second)

	// Mock tool execution (only happens after approval)
	env.OnActivity(activities.ExecuteToolActivity, mock.Anything, mock.Anything).Return(
		&models.ToolCall{
			ID:     "call_1",
			Status: "completed",
			Result: "Command executed",
		}, nil,
	).Once()

	// Mock second LLM call after tool execution
	env.OnActivity(activities.InvokeLLMActivity, mock.Anything, mock.Anything).Return(
		&models.LLMResponse{
			Content:      "Command completed successfully.",
			FinishReason: "stop",
			TokenUsage:   models.TokenUsage{TotalTokens: 15},
		}, nil,
	).Once()

	// Create input with approval required
	input := AgentExecutionInput{
		Request: models.ExecutionRequest{
			TaskID:          "test-task-4",
			UserMessage:     "Delete all files",
			MaxIterations:   5,
			RequireApproval: true,
			ModelConfig: models.ModelConfig{
				Provider: "anthropic",
				Model:    "claude-3-5-sonnet-20241022",
			},
			Tools: []models.Tool{
				{Name: "execute_command", Type: "builtin"},
			},
		},
	}

	// Execute workflow
	env.ExecuteWorkflow(ExecuteAgent, input)

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())

	// Get result
	var result AgentExecutionOutput
	require.NoError(t, env.GetWorkflowResult(&result))

	// Should complete successfully after approval
	require.Equal(t, models.TaskStatusCompleted, result.Response.Status)
	require.Equal(t, 2, result.Response.Iterations)
}

func TestAgentExecutionWorkflow_LLMError(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestWorkflowEnvironment()

	// Mock LLM to return error
	env.OnActivity(activities.InvokeLLMActivity, mock.Anything, mock.Anything).Return(
		nil,
		&models.LLMResponse{}, // Will cause error in workflow
	)

	env.OnActivity(activities.PublishEventActivity, mock.Anything, mock.Anything).Return(nil)

	// Create input
	input := AgentExecutionInput{
		Request: models.ExecutionRequest{
			TaskID:      "test-task-5",
			UserMessage: "Test error handling",
			ModelConfig: models.ModelConfig{
				Provider: "anthropic",
				Model:    "claude-3-5-sonnet-20241022",
			},
		},
	}

	// Execute workflow
	env.ExecuteWorkflow(ExecuteAgent, input)

	require.True(t, env.IsWorkflowCompleted())

	// Should have workflow error
	var result AgentExecutionOutput
	err := env.GetWorkflowResult(&result)
	require.Error(t, err)
}

func TestExtractFinalResult(t *testing.T) {
	tests := []struct {
		name     string
		state    *models.ExecutionState
		expected string
	}{
		{
			name: "single assistant message",
			state: &models.ExecutionState{
				Messages: []models.Message{
					{Role: "user", Content: "Hello"},
					{Role: "assistant", Content: "Hi there!"},
				},
			},
			expected: "Hi there!",
		},
		{
			name: "multiple messages",
			state: &models.ExecutionState{
				Messages: []models.Message{
					{Role: "user", Content: "Question"},
					{Role: "assistant", Content: "First answer"},
					{Role: "tool", Content: "Tool result"},
					{Role: "assistant", Content: "Final answer"},
				},
			},
			expected: "Final answer",
		},
		{
			name: "no assistant messages",
			state: &models.ExecutionState{
				Messages: []models.Message{
					{Role: "user", Content: "Hello"},
				},
			},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractFinalResult(tt.state)
			require.Equal(t, tt.expected, result)
		})
	}
}
