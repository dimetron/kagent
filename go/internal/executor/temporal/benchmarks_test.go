package temporal

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"go.temporal.io/sdk/testsuite"

	"github.com/kagent-dev/kagent/go/internal/executor/temporal/activities"
	"github.com/kagent-dev/kagent/go/internal/executor/temporal/models"
	"github.com/kagent-dev/kagent/go/internal/executor/temporal/workflows"
)

// BenchmarkWorkflowExecution benchmarks complete workflow execution
func BenchmarkWorkflowExecution(b *testing.B) {
	testSuite := &testsuite.WorkflowTestSuite{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		env := testSuite.NewTestWorkflowEnvironment()

		// Mock LLM activity
		env.OnActivity(activities.InvokeLLMActivity, mock.Anything, mock.Anything).Return(
			&models.LLMResponse{
				Content:      "Test response",
				FinishReason: "stop",
				TokenUsage:   models.TokenUsage{TotalTokens: 30},
			}, nil,
		)

		env.OnActivity(activities.PublishEventActivity, mock.Anything, mock.Anything).Return(nil)

		input := workflows.AgentExecutionInput{
			Request: models.ExecutionRequest{
				TaskID:      "bench-task",
				UserMessage: "Test message",
				ModelConfig: models.ModelConfig{
					Provider: "anthropic",
					Model:    "claude-3-5-sonnet-20241022",
				},
			},
		}

		env.ExecuteWorkflow(workflows.ExecuteAgent, input)
	}
}

// BenchmarkWorkflowWithToolCalls benchmarks workflow with tool execution
func BenchmarkWorkflowWithToolCalls(b *testing.B) {
	testSuite := &testsuite.WorkflowTestSuite{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		env := testSuite.NewTestWorkflowEnvironment()

		// First LLM call with tool calls
		env.OnActivity(activities.InvokeLLMActivity, mock.Anything, mock.MatchedBy(func(req models.LLMRequest) bool {
			return len(req.Messages) == 2
		})).Return(
			&models.LLMResponse{
				Content:      "Calling tool",
				FinishReason: "tool_calls",
				ToolCalls: []models.ToolCall{
					{ID: "call_1", Name: "test_tool", Status: "pending"},
				},
				TokenUsage: models.TokenUsage{TotalTokens: 20},
			}, nil,
		)

		// Tool execution
		env.OnActivity(activities.ExecuteToolActivity, mock.Anything, mock.Anything).Return(
			&models.ToolCall{ID: "call_1", Status: "completed", Result: "done"}, nil,
		)

		// Second LLM call
		env.OnActivity(activities.InvokeLLMActivity, mock.Anything, mock.MatchedBy(func(req models.LLMRequest) bool {
			return len(req.Messages) > 2
		})).Return(
			&models.LLMResponse{
				Content:      "Final response",
				FinishReason: "stop",
				TokenUsage:   models.TokenUsage{TotalTokens: 15},
			}, nil,
		)

		env.OnActivity(activities.PublishEventActivity, mock.Anything, mock.Anything).Return(nil)

		input := workflows.AgentExecutionInput{
			Request: models.ExecutionRequest{
				TaskID:      "bench-task",
				UserMessage: "Test message",
				Tools: []models.Tool{
					{Name: "test_tool", Type: "builtin"},
				},
				ModelConfig: models.ModelConfig{
					Provider: "anthropic",
					Model:    "claude-3-5-sonnet-20241022",
				},
			},
		}

		env.ExecuteWorkflow(workflows.ExecuteAgent, input)
	}
}

// BenchmarkMessageConversion benchmarks message conversion overhead
func BenchmarkMessageConversion(b *testing.B) {
	messages := []models.Message{
		{Role: "system", Content: "You are helpful"},
		{Role: "user", Content: "Hello"},
		{Role: "assistant", Content: "Hi there!"},
		{
			Role:    "assistant",
			Content: "Let me help",
			ToolCalls: []models.ToolCall{
				{ID: "call_1", Name: "tool", Arguments: map[string]interface{}{"arg": "value"}},
			},
		},
		{Role: "tool", Content: "result", ToolCallID: "call_1"},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Simulate message conversion
		converted := make([]interface{}, len(messages))
		for j, msg := range messages {
			converted[j] = map[string]interface{}{
				"role":    msg.Role,
				"content": msg.Content,
			}
		}
		_ = converted
	}
}

// BenchmarkStateManagement benchmarks state updates
func BenchmarkStateManagement(b *testing.B) {
	state := models.ExecutionState{
		TaskID:           "test-task",
		SessionID:        "test-session",
		CurrentIteration: 0,
		MaxIterations:    10,
		Messages:         make([]models.Message, 0, 100),
		ToolCalls:        make([]models.ToolCall, 0, 10),
		Status:           models.TaskStatusWorking,
		TokenUsage:       models.TokenUsage{},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Simulate state updates
		state.Messages = append(state.Messages, models.Message{
			Role:      "user",
			Content:   "Test message",
			Timestamp: time.Now(),
		})

		state.Messages = append(state.Messages, models.Message{
			Role:      "assistant",
			Content:   "Response",
			Timestamp: time.Now(),
		})

		state.TokenUsage.Add(models.TokenUsage{
			PromptTokens:     10,
			CompletionTokens: 20,
			TotalTokens:      30,
		})

		state.CurrentIteration++
	}
}

// BenchmarkEventPublishing benchmarks event publishing
func BenchmarkEventPublishing(b *testing.B) {
	publisher := activities.NewA2AEventPublisher()
	ctx := context.Background()

	event := models.A2AEvent{
		TaskID:    "test-task",
		EventType: "status_update",
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"status":    models.TaskStatusWorking,
			"iteration": 1,
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		publisher.PublishEvent(ctx, event)
	}
}

// BenchmarkToolExecution benchmarks tool executor
func BenchmarkToolExecution(b *testing.B) {
	executor := activities.NewDefaultToolExecutor()
	ctx := context.Background()

	toolCall := models.ToolCall{
		ID:        "call_1",
		Name:      "get_current_time",
		Arguments: map[string]interface{}{},
	}

	toolDef := models.Tool{
		Name: "get_current_time",
		Type: "builtin",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		executor.ExecuteTool(ctx, toolCall, toolDef)
	}
}

// BenchmarkConcurrentWorkflows benchmarks multiple concurrent workflows
func BenchmarkConcurrentWorkflows(b *testing.B) {
	testSuite := &testsuite.WorkflowTestSuite{}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			env := testSuite.NewTestWorkflowEnvironment()

			env.OnActivity(activities.InvokeLLMActivity, mock.Anything, mock.Anything).Return(
				&models.LLMResponse{
					Content:      "Response",
					FinishReason: "stop",
					TokenUsage:   models.TokenUsage{TotalTokens: 30},
				}, nil,
			)

			env.OnActivity(activities.PublishEventActivity, mock.Anything, mock.Anything).Return(nil)

			input := workflows.AgentExecutionInput{
				Request: models.ExecutionRequest{
					TaskID:      "bench-task",
					UserMessage: "Test",
					ModelConfig: models.ModelConfig{
						Provider: "anthropic",
						Model:    "claude-3-5-sonnet-20241022",
					},
				},
			}

			env.ExecuteWorkflow(workflows.ExecuteAgent, input)
		}
	})
}

// BenchmarkMemoryAllocation measures memory allocations
func BenchmarkMemoryAllocation(b *testing.B) {
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		state := models.ExecutionState{
			TaskID:           "test-task",
			SessionID:        "test-session",
			Messages:         make([]models.Message, 0, 10),
			ToolCalls:        make([]models.ToolCall, 0, 5),
			PendingApprovals: make([]models.ToolApproval, 0, 5),
			Metadata:         make(map[string]interface{}),
		}

		// Simulate typical workflow operations
		for j := 0; j < 5; j++ {
			state.Messages = append(state.Messages, models.Message{
				Role:      "user",
				Content:   "Message " + string(rune(j)),
				Timestamp: time.Now(),
			})
		}

		_ = state
	}
}

// BenchmarkTypicalAgentLoop benchmarks a typical multi-iteration agent loop
func BenchmarkTypicalAgentLoop(b *testing.B) {
	testSuite := &testsuite.WorkflowTestSuite{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		env := testSuite.NewTestWorkflowEnvironment()

		// Simulate 3 iterations with tool calls
		callCount := 0
		env.OnActivity(activities.InvokeLLMActivity, mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
			callCount++
		}).Return(func(ctx context.Context, req models.LLMRequest) *models.LLMResponse {
			if callCount%2 == 1 {
				// Odd calls: return tool calls
				return &models.LLMResponse{
					Content:      "Using tool",
					FinishReason: "tool_calls",
					ToolCalls: []models.ToolCall{
						{ID: "call_1", Name: "tool", Status: "pending"},
					},
					TokenUsage: models.TokenUsage{TotalTokens: 20},
				}
			}
			// Even calls: return final response
			return &models.LLMResponse{
				Content:      "Done",
				FinishReason: "stop",
				TokenUsage:   models.TokenUsage{TotalTokens: 15},
			}
		}, nil)

		env.OnActivity(activities.ExecuteToolActivity, mock.Anything, mock.Anything).Return(
			&models.ToolCall{ID: "call_1", Status: "completed", Result: "result"}, nil,
		)

		env.OnActivity(activities.PublishEventActivity, mock.Anything, mock.Anything).Return(nil)

		input := workflows.AgentExecutionInput{
			Request: models.ExecutionRequest{
				TaskID:        "bench-task",
				UserMessage:   "Complex task",
				MaxIterations: 10,
				Tools: []models.Tool{
					{Name: "tool", Type: "builtin"},
				},
				ModelConfig: models.ModelConfig{
					Provider: "anthropic",
					Model:    "claude-3-5-sonnet-20241022",
				},
			},
		}

		env.ExecuteWorkflow(workflows.ExecuteAgent, input)
	}
}
