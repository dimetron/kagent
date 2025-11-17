package workflows

import (
	"fmt"
	"time"

	"go.temporal.io/sdk/workflow"

	"github.com/kagent-dev/kagent/go/internal/executor/temporal/activities"
	"github.com/kagent-dev/kagent/go/internal/executor/temporal/models"
)

const (
	AgentExecutionWorkflow = "AgentExecutionWorkflow"
	DefaultActivityTimeout = 5 * time.Minute
)

// AgentExecutionInput represents the input to the agent execution workflow
type AgentExecutionInput struct {
	Request models.ExecutionRequest
}

// AgentExecutionOutput represents the output from the agent execution workflow
type AgentExecutionOutput struct {
	Response models.ExecutionResponse
}

// ExecuteAgent is the main workflow for agent execution
func ExecuteAgent(ctx workflow.Context, input AgentExecutionInput) (*AgentExecutionOutput, error) {
	logger := workflow.GetLogger(ctx)
	logger.Info("Starting agent execution workflow", "taskID", input.Request.TaskID)

	// Initialize execution state
	state := models.ExecutionState{
		TaskID:             input.Request.TaskID,
		SessionID:          input.Request.SessionID,
		CurrentIteration:   0,
		MaxIterations:      input.Request.MaxIterations,
		Messages:           make([]models.Message, 0),
		ToolCalls:          make([]models.ToolCall, 0),
		PendingApprovals:   make([]models.ToolApproval, 0),
		Status:             models.TaskStatusSubmitted,
		TokenUsage:         models.TokenUsage{},
		Metadata:           input.Request.Metadata,
		ContinueExecution:  true,
		ExecutionStartTime: workflow.Now(ctx),
	}

	// Add system message if provided
	if input.Request.SystemMessage != "" {
		state.Messages = append(state.Messages, models.Message{
			Role:      "system",
			Content:   input.Request.SystemMessage,
			Timestamp: workflow.Now(ctx),
		})
	}

	// Add user message
	state.Messages = append(state.Messages, models.Message{
		Role:      "user",
		Content:   input.Request.UserMessage,
		Timestamp: workflow.Now(ctx),
	})

	// Update status to working
	state.Status = models.TaskStatusWorking
	if err := publishStatusUpdate(ctx, &state); err != nil {
		logger.Warn("Failed to publish status update", "error", err)
	}

	// Main agent loop
	for state.ContinueExecution && state.CurrentIteration < state.MaxIterations {
		state.CurrentIteration++
		logger.Info("Starting iteration", "iteration", state.CurrentIteration)

		// Invoke LLM
		llmResponse, err := invokeLLM(ctx, &state, input.Request.ModelConfig, input.Request.Tools)
		if err != nil {
			return handleError(ctx, &state, fmt.Errorf("LLM invocation failed: %w", err))
		}

		state.LastLLMResponse = llmResponse
		state.TokenUsage.Add(llmResponse.TokenUsage)

		// Add assistant message to conversation
		assistantMsg := models.Message{
			Role:      "assistant",
			Content:   llmResponse.Content,
			ToolCalls: llmResponse.ToolCalls,
			Timestamp: workflow.Now(ctx),
		}
		state.Messages = append(state.Messages, assistantMsg)

		// Check finish reason
		if llmResponse.FinishReason == "stop" && len(llmResponse.ToolCalls) == 0 {
			// Agent has completed successfully
			logger.Info("Agent completed with stop reason")
			state.ContinueExecution = false
			state.Status = models.TaskStatusCompleted
			break
		}

		// Handle tool calls if present
		if len(llmResponse.ToolCalls) > 0 {
			logger.Info("Processing tool calls", "count", len(llmResponse.ToolCalls))

			// Add tool calls to state
			state.ToolCalls = append(state.ToolCalls, llmResponse.ToolCalls...)

			// Request approval if HITL is enabled
			if input.Request.RequireApproval {
				approved, err := requestToolApproval(ctx, &state, llmResponse.ToolCalls)
				if err != nil {
					return handleError(ctx, &state, fmt.Errorf("tool approval failed: %w", err))
				}

				if !approved {
					// User denied tool execution
					state.ContinueExecution = false
					state.Status = models.TaskStatusFailed
					logger.Info("Tool execution denied by user")
					break
				}
			}

			// Execute tools
			toolResults, err := executeTools(ctx, &state, llmResponse.ToolCalls, input.Request.Tools)
			if err != nil {
				return handleError(ctx, &state, fmt.Errorf("tool execution failed: %w", err))
			}

			// Add tool results to messages
			for _, result := range toolResults {
				state.Messages = append(state.Messages, models.Message{
					Role:       "tool",
					Content:    result.Result,
					ToolCallID: result.ID,
					Timestamp:  workflow.Now(ctx),
				})
			}
		}

		// Check for maximum iterations
		if state.CurrentIteration >= state.MaxIterations {
			logger.Warn("Maximum iterations reached", "maxIterations", state.MaxIterations)
			state.Status = models.TaskStatusCompleted
			state.ContinueExecution = false
		}
	}

	// Publish final status
	if err := publishStatusUpdate(ctx, &state); err != nil {
		logger.Warn("Failed to publish final status", "error", err)
	}

	// Build response
	response := models.ExecutionResponse{
		TaskID:     state.TaskID,
		Status:     state.Status,
		Result:     extractFinalResult(&state),
		Iterations: state.CurrentIteration,
		TokenUsage: state.TokenUsage,
		Duration:   workflow.Now(ctx).Sub(state.ExecutionStartTime),
		Metadata:   state.Metadata,
		WorkflowID: workflow.GetInfo(ctx).WorkflowExecution.ID,
		WorkflowRunID: workflow.GetInfo(ctx).WorkflowExecution.RunID,
	}

	logger.Info("Agent execution workflow completed",
		"taskID", response.TaskID,
		"status", response.Status,
		"iterations", response.Iterations,
		"totalTokens", response.TokenUsage.TotalTokens)

	return &AgentExecutionOutput{Response: response}, nil
}

// invokeLLM calls the LLM via activity
func invokeLLM(
	ctx workflow.Context,
	state *models.ExecutionState,
	modelConfig models.ModelConfig,
	tools []models.Tool,
) (*models.LLMResponse, error) {
	activityCtx := workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout: DefaultActivityTimeout,
		RetryPolicy: &workflow.RetryPolicy{
			MaximumAttempts: 3,
			InitialInterval: time.Second,
			MaximumInterval: 30 * time.Second,
		},
	})

	llmRequest := models.LLMRequest{
		Messages:    state.Messages,
		Tools:       tools,
		ModelConfig: modelConfig,
		Stream:      false,
	}

	var response models.LLMResponse
	err := workflow.ExecuteActivity(activityCtx, activities.InvokeLLMActivity, llmRequest).Get(ctx, &response)
	if err != nil {
		return nil, err
	}

	// Add timestamps to tool calls
	for i := range response.ToolCalls {
		response.ToolCalls[i].Timestamp = workflow.Now(ctx)
	}

	return &response, nil
}

// executeTools executes tool calls via activities
func executeTools(
	ctx workflow.Context,
	state *models.ExecutionState,
	toolCalls []models.ToolCall,
	availableTools []models.Tool,
) ([]models.ToolCall, error) {
	results := make([]models.ToolCall, len(toolCalls))

	// Execute tools in parallel
	futures := make([]workflow.Future, len(toolCalls))
	for i, tc := range toolCalls {
		activityCtx := workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
			StartToCloseTimeout: DefaultActivityTimeout,
			RetryPolicy: &workflow.RetryPolicy{
				MaximumAttempts: 2,
				InitialInterval: time.Second,
			},
		})

		// Find tool definition
		var toolDef *models.Tool
		for j := range availableTools {
			if availableTools[j].Name == tc.Name {
				toolDef = &availableTools[j]
				break
			}
		}

		if toolDef == nil {
			results[i] = tc
			results[i].Status = "failed"
			results[i].Error = fmt.Sprintf("tool %s not found", tc.Name)
			continue
		}

		// Execute tool activity
		input := activities.ToolExecutionInput{
			ToolCall: tc,
			ToolDef:  *toolDef,
		}
		futures[i] = workflow.ExecuteActivity(activityCtx, activities.ExecuteToolActivity, input)
	}

	// Collect results
	for i, future := range futures {
		if future == nil {
			continue
		}

		var result models.ToolCall
		if err := future.Get(ctx, &result); err != nil {
			results[i] = toolCalls[i]
			results[i].Status = "failed"
			results[i].Error = err.Error()
		} else {
			results[i] = result
		}
	}

	return results, nil
}

// requestToolApproval handles HITL tool approval workflow
func requestToolApproval(
	ctx workflow.Context,
	state *models.ExecutionState,
	toolCalls []models.ToolCall,
) (bool, error) {
	logger := workflow.GetLogger(ctx)

	// Create approval requests
	approvals := make([]models.ToolApproval, len(toolCalls))
	for i, tc := range toolCalls {
		approvals[i] = models.ToolApproval{
			ToolCallID:  tc.ID,
			ToolName:    tc.Name,
			Arguments:   tc.Arguments,
			RequestedAt: workflow.Now(ctx),
		}
	}

	state.PendingApprovals = approvals
	state.Status = models.TaskStatusInputRequired

	// Publish status update to notify user
	if err := publishStatusUpdate(ctx, state); err != nil {
		logger.Warn("Failed to publish approval request", "error", err)
	}

	// Wait for approval signal
	var approved bool
	selector := workflow.NewSelector(ctx)
	approvalChannel := workflow.GetSignalChannel(ctx, "tool-approval")

	selector.AddReceive(approvalChannel, func(c workflow.ReceiveChannel, more bool) {
		c.Receive(ctx, &approved)
	})

	// Add timeout
	timer := workflow.NewTimer(ctx, 5*time.Minute)
	selector.AddFuture(timer, func(f workflow.Future) {
		logger.Warn("Tool approval timed out")
		approved = false
	})

	selector.Select(ctx)

	if approved {
		logger.Info("Tool execution approved")
		state.Status = models.TaskStatusWorking
		state.PendingApprovals = nil
	} else {
		logger.Info("Tool execution denied")
	}

	return approved, nil
}

// publishStatusUpdate publishes status updates via activity
func publishStatusUpdate(ctx workflow.Context, state *models.ExecutionState) error {
	activityCtx := workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout: 30 * time.Second,
		RetryPolicy: &workflow.RetryPolicy{
			MaximumAttempts: 3,
			InitialInterval: time.Second,
		},
	})

	event := models.A2AEvent{
		TaskID:    state.TaskID,
		EventType: "status_update",
		Timestamp: workflow.Now(ctx),
		Data: map[string]interface{}{
			"status":            state.Status,
			"iteration":         state.CurrentIteration,
			"pending_approvals": state.PendingApprovals,
			"token_usage":       state.TokenUsage,
		},
	}

	return workflow.ExecuteActivity(activityCtx, activities.PublishEventActivity, event).Get(ctx, nil)
}

// handleError handles workflow errors
func handleError(ctx workflow.Context, state *models.ExecutionState, err error) (*AgentExecutionOutput, error) {
	logger := workflow.GetLogger(ctx)
	logger.Error("Workflow error", "error", err)

	state.Status = models.TaskStatusFailed
	publishStatusUpdate(ctx, state)

	response := models.ExecutionResponse{
		TaskID:     state.TaskID,
		Status:     state.Status,
		Error:      err.Error(),
		Iterations: state.CurrentIteration,
		TokenUsage: state.TokenUsage,
		Duration:   workflow.Now(ctx).Sub(state.ExecutionStartTime),
		Metadata:   state.Metadata,
		WorkflowID: workflow.GetInfo(ctx).WorkflowExecution.ID,
		WorkflowRunID: workflow.GetInfo(ctx).WorkflowExecution.RunID,
	}

	return &AgentExecutionOutput{Response: response}, err
}

// extractFinalResult extracts the final result from the execution state
func extractFinalResult(state *models.ExecutionState) string {
	// Get the last assistant message
	for i := len(state.Messages) - 1; i >= 0; i-- {
		if state.Messages[i].Role == "assistant" {
			return state.Messages[i].Content
		}
	}
	return ""
}
