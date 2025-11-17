package activities

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/kagent-dev/kagent/go/internal/executor/temporal/llm"
	"github.com/kagent-dev/kagent/go/internal/executor/temporal/models"
	"go.temporal.io/sdk/activity"
)

// Activities struct holds all activity dependencies
type Activities struct {
	llmRegistry    llm.ProviderRegistry
	eventPublisher EventPublisher
	toolExecutor   ToolExecutor
}

// NewActivities creates a new Activities instance
func NewActivities(
	llmRegistry llm.ProviderRegistry,
	eventPublisher EventPublisher,
	toolExecutor ToolExecutor,
) *Activities {
	return &Activities{
		llmRegistry:    llmRegistry,
		eventPublisher: eventPublisher,
		toolExecutor:   toolExecutor,
	}
}

// EventPublisher interface for publishing events
type EventPublisher interface {
	PublishEvent(ctx context.Context, event models.A2AEvent) error
}

// ToolExecutor interface for executing tools
type ToolExecutor interface {
	ExecuteTool(ctx context.Context, toolCall models.ToolCall, toolDef models.Tool) (string, error)
}

// InvokeLLMActivity invokes an LLM provider
func (a *Activities) InvokeLLMActivity(ctx context.Context, request models.LLMRequest) (*models.LLMResponse, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Invoking LLM", "provider", request.ModelConfig.Provider, "model", request.ModelConfig.Model)

	// Get provider from registry
	provider, err := a.llmRegistry.Get(request.ModelConfig.Provider)
	if err != nil {
		return nil, fmt.Errorf("failed to get LLM provider: %w", err)
	}

	// Record heartbeat
	activity.RecordHeartbeat(ctx, "invoking LLM")

	// Invoke LLM
	startTime := time.Now()
	response, err := provider.Chat(ctx, request)
	if err != nil {
		return nil, fmt.Errorf("LLM invocation failed: %w", err)
	}

	duration := time.Since(startTime)
	logger.Info("LLM invocation completed",
		"duration", duration,
		"totalTokens", response.TokenUsage.TotalTokens,
		"finishReason", response.FinishReason)

	return response, nil
}

// ToolExecutionInput represents input to tool execution activity
type ToolExecutionInput struct {
	ToolCall models.ToolCall
	ToolDef  models.Tool
}

// ExecuteToolActivity executes a tool
func (a *Activities) ExecuteToolActivity(ctx context.Context, input ToolExecutionInput) (*models.ToolCall, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Executing tool", "tool", input.ToolCall.Name, "id", input.ToolCall.ID)

	result := input.ToolCall
	result.Status = "executing"

	// Record heartbeat
	activity.RecordHeartbeat(ctx, fmt.Sprintf("executing tool %s", input.ToolCall.Name))

	// Execute tool
	output, err := a.toolExecutor.ExecuteTool(ctx, input.ToolCall, input.ToolDef)
	if err != nil {
		logger.Error("Tool execution failed", "tool", input.ToolCall.Name, "error", err)
		result.Status = "failed"
		result.Error = err.Error()
		return &result, nil // Don't return error, return failed tool call
	}

	result.Status = "completed"
	result.Result = output

	logger.Info("Tool execution completed", "tool", input.ToolCall.Name, "resultLength", len(output))
	return &result, nil
}

// PublishEventActivity publishes an A2A event
func (a *Activities) PublishEventActivity(ctx context.Context, event models.A2AEvent) error {
	logger := activity.GetLogger(ctx)
	logger.Info("Publishing event", "taskID", event.TaskID, "eventType", event.EventType)

	// Record heartbeat
	activity.RecordHeartbeat(ctx, "publishing event")

	// Publish event
	err := a.eventPublisher.PublishEvent(ctx, event)
	if err != nil {
		logger.Error("Failed to publish event", "error", err)
		return fmt.Errorf("event publishing failed: %w", err)
	}

	logger.Info("Event published successfully")
	return nil
}

// SaveStateActivity saves execution state
func (a *Activities) SaveStateActivity(ctx context.Context, state models.ExecutionState) error {
	logger := activity.GetLogger(ctx)
	logger.Info("Saving execution state", "taskID", state.TaskID, "iteration", state.CurrentIteration)

	// Record heartbeat
	activity.RecordHeartbeat(ctx, "saving state")

	// In a real implementation, this would save to a database or storage
	// For now, we'll just log it as Temporal maintains workflow state
	stateJSON, _ := json.MarshalIndent(state, "", "  ")
	logger.Debug("Execution state", "state", string(stateJSON))

	return nil
}

// LoadStateActivity loads execution state
func (a *Activities) LoadStateActivity(ctx context.Context, taskID string) (*models.ExecutionState, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Loading execution state", "taskID", taskID)

	// Record heartbeat
	activity.RecordHeartbeat(ctx, "loading state")

	// In a real implementation, this would load from database
	// For now, return nil to indicate no saved state (Temporal handles this)
	return nil, nil
}

// CancelExecutionActivity cancels an ongoing execution
func (a *Activities) CancelExecutionActivity(ctx context.Context, taskID string) error {
	logger := activity.GetLogger(ctx)
	logger.Info("Cancelling execution", "taskID", taskID)

	// Record heartbeat
	activity.RecordHeartbeat(ctx, "cancelling execution")

	// Publish cancellation event
	event := models.A2AEvent{
		TaskID:    taskID,
		EventType: "status_update",
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"status": models.TaskStatusCancelled,
		},
	}

	return a.eventPublisher.PublishEvent(ctx, event)
}
