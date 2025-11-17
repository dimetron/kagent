// +build integration

package temporal

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/client"

	"github.com/kagent-dev/kagent/go/internal/executor/temporal/activities"
	"github.com/kagent-dev/kagent/go/internal/executor/temporal/models"
)

// Integration tests require:
// - Temporal server running (docker-compose up)
// - API keys set in environment variables

func TestIntegration_EndToEnd(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Check if API keys are set
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		t.Skip("ANTHROPIC_API_KEY not set")
	}

	// Connect to Temporal
	temporalClient, err := client.Dial(client.Options{
		HostPort: "localhost:7233",
	})
	require.NoError(t, err)
	defer temporalClient.Close()

	// Start worker in background
	workerConfig := WorkerConfig{
		TaskQueue:     "test-queue",
		MaxConcurrent: 10,
		LLMProviderConfigs: []LLMProviderConfig{
			{
				Name:   "anthropic",
				APIKey: apiKey,
			},
		},
	}

	worker, err := NewWorker(temporalClient, workerConfig)
	require.NoError(t, err)

	workerCtx, workerCancel := context.WithCancel(context.Background())
	defer workerCancel()

	go func() {
		worker.Run(workerCtx)
	}()

	// Give worker time to start
	time.Sleep(2 * time.Second)

	// Create executor service
	eventPublisher := activities.NewA2AEventPublisher()
	executorService := NewExecutorService(temporalClient, eventPublisher, "test-queue")

	// Test 1: Simple chat execution
	t.Run("SimpleChatExecution", func(t *testing.T) {
		request := models.ExecutionRequest{
			TaskID:        "integration-test-1",
			SessionID:     "session-1",
			UserMessage:   "Say hello in exactly 3 words",
			SystemMessage: "You are a helpful assistant.",
			MaxIterations: 5,
			Timeout:       30 * time.Second,
			ModelConfig: models.ModelConfig{
				Provider:    "anthropic",
				Model:       "claude-3-5-sonnet-20241022",
				Temperature: 0.7,
				MaxTokens:   100,
			},
		}

		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
		defer cancel()

		response, err := executorService.Execute(ctx, request)
		require.NoError(t, err)
		require.NotNil(t, response)
		require.Equal(t, models.TaskStatusCompleted, response.Status)
		require.NotEmpty(t, response.Result)
		require.Greater(t, response.TokenUsage.TotalTokens, 0)
		require.GreaterOrEqual(t, response.Iterations, 1)

		t.Logf("Result: %s", response.Result)
		t.Logf("Tokens used: %d", response.TokenUsage.TotalTokens)
		t.Logf("Duration: %s", response.Duration)
	})

	// Test 2: Execution with builtin tools
	t.Run("ExecutionWithTools", func(t *testing.T) {
		request := models.ExecutionRequest{
			TaskID:      "integration-test-2",
			SessionID:   "session-2",
			UserMessage: "What time is it now?",
			Tools: []models.Tool{
				{
					Name:        "get_current_time",
					Description: "Get the current time",
					Parameters: map[string]interface{}{
						"type": "object",
					},
					Type: "builtin",
				},
			},
			MaxIterations: 5,
			Timeout:       30 * time.Second,
			ModelConfig: models.ModelConfig{
				Provider:    "anthropic",
				Model:       "claude-3-5-sonnet-20241022",
				Temperature: 0.7,
				MaxTokens:   200,
			},
		}

		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
		defer cancel()

		response, err := executorService.Execute(ctx, request)
		require.NoError(t, err)
		require.NotNil(t, response)
		require.Equal(t, models.TaskStatusCompleted, response.Status)
		require.NotEmpty(t, response.Result)

		t.Logf("Result: %s", response.Result)
		t.Logf("Iterations: %d", response.Iterations)
	})

	// Test 3: Async execution with status polling
	t.Run("AsyncExecution", func(t *testing.T) {
		request := models.ExecutionRequest{
			TaskID:        "integration-test-3",
			SessionID:     "session-3",
			UserMessage:   "Count from 1 to 5",
			MaxIterations: 5,
			Timeout:       30 * time.Second,
			ModelConfig: models.ModelConfig{
				Provider:    "anthropic",
				Model:       "claude-3-5-sonnet-20241022",
				Temperature: 0.7,
				MaxTokens:   100,
			},
		}

		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
		defer cancel()

		// Start async execution
		workflowRun, err := executorService.ExecuteAsync(ctx, request)
		require.NoError(t, err)
		require.NotNil(t, workflowRun)

		workflowID := workflowRun.GetID()
		t.Logf("Workflow ID: %s", workflowID)

		// Wait for completion
		var result models.ExecutionResponse
		err = workflowRun.Get(ctx, &result)
		require.NoError(t, err)
		require.Equal(t, models.TaskStatusCompleted, result.Status)

		t.Logf("Result: %s", result.Result)
	})

	// Test 4: Cancellation
	t.Run("Cancellation", func(t *testing.T) {
		request := models.ExecutionRequest{
			TaskID:        "integration-test-4",
			SessionID:     "session-4",
			UserMessage:   "Write a very long essay about AI",
			MaxIterations: 20,
			Timeout:       5 * time.Minute,
			ModelConfig: models.ModelConfig{
				Provider:    "anthropic",
				Model:       "claude-3-5-sonnet-20241022",
				Temperature: 0.7,
				MaxTokens:   2000,
			},
		}

		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
		defer cancel()

		// Start async execution
		workflowRun, err := executorService.ExecuteAsync(ctx, request)
		require.NoError(t, err)

		workflowID := workflowRun.GetID()

		// Cancel after a short delay
		time.Sleep(1 * time.Second)
		err = executorService.CancelExecution(ctx, workflowID)
		require.NoError(t, err)

		t.Logf("Cancelled workflow: %s", workflowID)
	})

	// Test 5: Error handling (invalid model)
	t.Run("ErrorHandling", func(t *testing.T) {
		request := models.ExecutionRequest{
			TaskID:      "integration-test-5",
			SessionID:   "session-5",
			UserMessage: "Test error handling",
			ModelConfig: models.ModelConfig{
				Provider:    "nonexistent",
				Model:       "invalid-model",
				Temperature: 0.7,
				MaxTokens:   100,
			},
		}

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		_, err := executorService.Execute(ctx, request)
		require.Error(t, err)
		t.Logf("Expected error: %v", err)
	})

	// Stop worker
	workerCancel()
}

func TestIntegration_MultipleProviders(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Check API keys
	anthropicKey := os.Getenv("ANTHROPIC_API_KEY")
	openaiKey := os.Getenv("OPENAI_API_KEY")

	if anthropicKey == "" && openaiKey == "" {
		t.Skip("No API keys set")
	}

	// Connect to Temporal
	temporalClient, err := client.Dial(client.Options{
		HostPort: "localhost:7233",
	})
	require.NoError(t, err)
	defer temporalClient.Close()

	// Configure worker with multiple providers
	workerConfig := WorkerConfig{
		TaskQueue:     "test-multi-provider-queue",
		MaxConcurrent: 10,
		LLMProviderConfigs: []LLMProviderConfig{},
	}

	if anthropicKey != "" {
		workerConfig.LLMProviderConfigs = append(workerConfig.LLMProviderConfigs, LLMProviderConfig{
			Name:   "anthropic",
			APIKey: anthropicKey,
		})
	}

	if openaiKey != "" {
		workerConfig.LLMProviderConfigs = append(workerConfig.LLMProviderConfigs, LLMProviderConfig{
			Name:   "openai",
			APIKey: openaiKey,
		})
	}

	worker, err := NewWorker(temporalClient, workerConfig)
	require.NoError(t, err)

	workerCtx, workerCancel := context.WithCancel(context.Background())
	defer workerCancel()

	go func() {
		worker.Run(workerCtx)
	}()

	time.Sleep(2 * time.Second)

	eventPublisher := activities.NewA2AEventPublisher()
	executorService := NewExecutorService(temporalClient, eventPublisher, "test-multi-provider-queue")

	// Test Anthropic
	if anthropicKey != "" {
		t.Run("Anthropic", func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
			defer cancel()

			request := models.ExecutionRequest{
				TaskID:      "anthropic-test",
				UserMessage: "Say hello",
				ModelConfig: models.ModelConfig{
					Provider: "anthropic",
					Model:    "claude-3-5-sonnet-20241022",
					MaxTokens: 50,
				},
			}

			response, err := executorService.Execute(ctx, request)
			require.NoError(t, err)
			require.Equal(t, models.TaskStatusCompleted, response.Status)
			t.Logf("Anthropic response: %s", response.Result)
		})
	}

	// Test OpenAI
	if openaiKey != "" {
		t.Run("OpenAI", func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
			defer cancel()

			request := models.ExecutionRequest{
				TaskID:      "openai-test",
				UserMessage: "Say hello",
				ModelConfig: models.ModelConfig{
					Provider: "openai",
					Model:    "gpt-3.5-turbo",
					MaxTokens: 50,
				},
			}

			response, err := executorService.Execute(ctx, request)
			require.NoError(t, err)
			require.Equal(t, models.TaskStatusCompleted, response.Status)
			t.Logf("OpenAI response: %s", response.Result)
		})
	}

	workerCancel()
}
