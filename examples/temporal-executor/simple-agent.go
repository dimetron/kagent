package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"go.temporal.io/sdk/client"

	executor "github.com/kagent-dev/kagent/go/internal/executor/temporal"
	"github.com/kagent-dev/kagent/go/internal/executor/temporal/activities"
	"github.com/kagent-dev/kagent/go/internal/executor/temporal/models"
)

func main() {
	// Connect to Temporal server
	temporalClient, err := client.Dial(client.Options{
		HostPort: "localhost:7233",
	})
	if err != nil {
		log.Fatalf("Failed to connect to Temporal: %v", err)
	}
	defer temporalClient.Close()

	// Create event publisher
	eventPublisher := activities.NewA2AEventPublisher()

	// Create executor service
	executorService := executor.NewExecutorService(
		temporalClient,
		eventPublisher,
		"agent-execution-queue",
	)

	// Create execution request
	request := models.ExecutionRequest{
		TaskID:    "example-task-1",
		SessionID: "example-session-1",
		UserID:    "user-123",
		UserMessage: "Write a Python function that calculates the factorial of a number",
		SystemMessage: "You are a helpful coding assistant. Provide clear, " +
			"well-documented code with explanations.",
		MaxIterations: 5,
		Timeout:       2 * time.Minute,
		ModelConfig: models.ModelConfig{
			Provider:    "anthropic",
			Model:       "claude-3-5-sonnet-20241022",
			Temperature: 0.7,
			MaxTokens:   4096,
		},
		Tools: []models.Tool{
			{
				Name:        "execute_python",
				Description: "Execute Python code and return the result",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"code": map[string]interface{}{
							"type":        "string",
							"description": "Python code to execute",
						},
					},
					"required": []string{"code"},
				},
				Type: "builtin",
			},
		},
		RequireApproval: false,
		Metadata:        make(map[string]interface{}),
	}

	ctx := context.Background()

	// Start execution asynchronously
	fmt.Println("Starting agent execution...")
	workflowRun, err := executorService.ExecuteAsync(ctx, request)
	if err != nil {
		log.Fatalf("Failed to start execution: %v", err)
	}

	fmt.Printf("Workflow started: %s (RunID: %s)\n",
		workflowRun.GetID(),
		workflowRun.GetRunID())

	// Subscribe to events
	eventChan := executorService.StreamEvents(ctx, request.TaskID)

	// Print events as they arrive
	go func() {
		for event := range eventChan {
			fmt.Printf("[Event] %s: %+v\n", event.EventType, event.Data)
		}
	}()

	// Wait for completion
	fmt.Println("Waiting for execution to complete...")
	var result models.ExecutionResponse
	err = workflowRun.Get(ctx, &result)
	if err != nil {
		log.Fatalf("Workflow failed: %v", err)
	}

	// Print results
	fmt.Println("\n" + "=".repeat(60))
	fmt.Println("Execution completed!")
	fmt.Println("=".repeat(60))
	fmt.Printf("Status: %s\n", result.Status)
	fmt.Printf("Iterations: %d\n", result.Iterations)
	fmt.Printf("Duration: %s\n", result.Duration)
	fmt.Printf("Tokens Used: %d\n", result.TokenUsage.TotalTokens)
	fmt.Println("\nResult:")
	fmt.Println(result.Result)

	if result.Error != "" {
		fmt.Printf("\nError: %s\n", result.Error)
	}

	fmt.Println("\nArtifacts:")
	for i, artifact := range result.Artifacts {
		fmt.Printf("%d. [%s] %s\n", i+1, artifact.Type, artifact.ID)
	}

	fmt.Println("\nWorkflow Details:")
	fmt.Printf("Workflow ID: %s\n", result.WorkflowID)
	fmt.Printf("Run ID: %s\n", result.WorkflowRunID)
	fmt.Println("\nView in Temporal UI: http://localhost:8080")
}
