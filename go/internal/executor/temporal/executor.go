package temporal

import (
	"context"
	"fmt"

	"go.temporal.io/sdk/client"

	"github.com/kagent-dev/kagent/go/internal/executor/temporal/activities"
	"github.com/kagent-dev/kagent/go/internal/executor/temporal/models"
	"github.com/kagent-dev/kagent/go/internal/executor/temporal/workflows"
)

// ExecutorService manages agent execution via Temporal
type ExecutorService struct {
	temporalClient client.Client
	eventPublisher activities.EventPublisher
	taskQueue      string
}

// NewExecutorService creates a new executor service
func NewExecutorService(
	temporalClient client.Client,
	eventPublisher activities.EventPublisher,
	taskQueue string,
) *ExecutorService {
	return &ExecutorService{
		temporalClient: temporalClient,
		eventPublisher: eventPublisher,
		taskQueue:      taskQueue,
	}
}

// Execute starts an agent execution workflow
func (s *ExecutorService) Execute(ctx context.Context, request models.ExecutionRequest) (*models.ExecutionResponse, error) {
	// Create workflow options
	workflowOptions := client.StartWorkflowOptions{
		ID:        fmt.Sprintf("agent-execution-%s", request.TaskID),
		TaskQueue: s.taskQueue,
	}

	// Start workflow
	workflowRun, err := s.temporalClient.ExecuteWorkflow(
		ctx,
		workflowOptions,
		workflows.ExecuteAgent,
		workflows.AgentExecutionInput{Request: request},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to start workflow: %w", err)
	}

	// Wait for workflow to complete
	var result workflows.AgentExecutionOutput
	err = workflowRun.Get(ctx, &result)
	if err != nil {
		return nil, fmt.Errorf("workflow execution failed: %w", err)
	}

	return &result.Response, nil
}

// ExecuteAsync starts an agent execution workflow asynchronously
func (s *ExecutorService) ExecuteAsync(ctx context.Context, request models.ExecutionRequest) (client.WorkflowRun, error) {
	// Create workflow options
	workflowOptions := client.StartWorkflowOptions{
		ID:        fmt.Sprintf("agent-execution-%s", request.TaskID),
		TaskQueue: s.taskQueue,
	}

	// Start workflow
	workflowRun, err := s.temporalClient.ExecuteWorkflow(
		ctx,
		workflowOptions,
		workflows.ExecuteAgent,
		workflows.AgentExecutionInput{Request: request},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to start workflow: %w", err)
	}

	return workflowRun, nil
}

// GetExecutionStatus retrieves the status of an execution
func (s *ExecutorService) GetExecutionStatus(ctx context.Context, workflowID string) (*models.ExecutionResponse, error) {
	// Get workflow execution
	workflowRun := s.temporalClient.GetWorkflow(ctx, workflowID, "")

	// Get workflow result
	var result workflows.AgentExecutionOutput
	err := workflowRun.Get(ctx, &result)
	if err != nil {
		return nil, fmt.Errorf("failed to get workflow result: %w", err)
	}

	return &result.Response, nil
}

// CancelExecution cancels an ongoing execution
func (s *ExecutorService) CancelExecution(ctx context.Context, workflowID string) error {
	err := s.temporalClient.CancelWorkflow(ctx, workflowID, "")
	if err != nil {
		return fmt.Errorf("failed to cancel workflow: %w", err)
	}
	return nil
}

// ApproveToolExecution sends approval signal to workflow
func (s *ExecutorService) ApproveToolExecution(ctx context.Context, workflowID string, approved bool) error {
	err := s.temporalClient.SignalWorkflow(ctx, workflowID, "", "tool-approval", approved)
	if err != nil {
		return fmt.Errorf("failed to send approval signal: %w", err)
	}
	return nil
}

// StreamEvents streams execution events
func (s *ExecutorService) StreamEvents(ctx context.Context, taskID string) <-chan models.A2AEvent {
	// Subscribe to events for this task
	if publisher, ok := s.eventPublisher.(*activities.A2AEventPublisher); ok {
		return publisher.Subscribe(taskID)
	}

	// Return empty channel if publisher doesn't support subscriptions
	ch := make(chan models.A2AEvent)
	close(ch)
	return ch
}

// QueryWorkflow queries workflow state
func (s *ExecutorService) QueryWorkflow(ctx context.Context, workflowID string, queryType string) (interface{}, error) {
	resp, err := s.temporalClient.QueryWorkflow(ctx, workflowID, "", queryType)
	if err != nil {
		return nil, fmt.Errorf("failed to query workflow: %w", err)
	}

	var result interface{}
	if err := resp.Get(&result); err != nil {
		return nil, fmt.Errorf("failed to decode query result: %w", err)
	}

	return result, nil
}
