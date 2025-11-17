package temporal

import (
	"context"
	"fmt"
	"log"

	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"

	"github.com/kagent-dev/kagent/go/internal/executor/temporal/activities"
	"github.com/kagent-dev/kagent/go/internal/executor/temporal/llm"
	"github.com/kagent-dev/kagent/go/internal/executor/temporal/workflows"
)

const (
	DefaultTaskQueue = "agent-execution-queue"
)

// WorkerConfig holds worker configuration
type WorkerConfig struct {
	TaskQueue          string
	MaxConcurrent      int
	LLMProviderConfigs []LLMProviderConfig
}

// LLMProviderConfig holds configuration for an LLM provider
type LLMProviderConfig struct {
	Name   string
	APIKey string
	Config map[string]interface{}
}

// Worker manages Temporal workers for agent execution
type Worker struct {
	temporalClient client.Client
	worker         worker.Worker
	config         WorkerConfig
	llmRegistry    llm.ProviderRegistry
	activities     *activities.Activities
}

// NewWorker creates a new Temporal worker
func NewWorker(temporalClient client.Client, config WorkerConfig) (*Worker, error) {
	// Create LLM registry and register providers
	llmRegistry := llm.NewRegistry()

	for _, providerConfig := range config.LLMProviderConfigs {
		var provider llm.Provider

		switch providerConfig.Name {
		case "openai":
			provider = llm.NewOpenAIProvider(providerConfig.APIKey)
		case "anthropic":
			provider = llm.NewAnthropicProvider(providerConfig.APIKey)
		default:
			return nil, fmt.Errorf("unsupported LLM provider: %s", providerConfig.Name)
		}

		if err := llmRegistry.Register(provider); err != nil {
			return nil, fmt.Errorf("failed to register provider %s: %w", providerConfig.Name, err)
		}
	}

	// Create event publisher
	eventPublisher := activities.NewA2AEventPublisher()

	// Create tool executor
	toolExecutor := activities.NewDefaultToolExecutor()

	// Create activities
	acts := activities.NewActivities(llmRegistry, eventPublisher, toolExecutor)

	// Create worker
	w := worker.New(temporalClient, config.TaskQueue, worker.Options{
		MaxConcurrentActivityExecutionSize:     config.MaxConcurrent,
		MaxConcurrentWorkflowTaskExecutionSize: config.MaxConcurrent,
	})

	// Register workflows
	w.RegisterWorkflow(workflows.ExecuteAgent)

	// Register activities
	w.RegisterActivity(acts.InvokeLLMActivity)
	w.RegisterActivity(acts.ExecuteToolActivity)
	w.RegisterActivity(acts.PublishEventActivity)
	w.RegisterActivity(acts.SaveStateActivity)
	w.RegisterActivity(acts.LoadStateActivity)
	w.RegisterActivity(acts.CancelExecutionActivity)

	return &Worker{
		temporalClient: temporalClient,
		worker:         w,
		config:         config,
		llmRegistry:    llmRegistry,
		activities:     acts,
	}, nil
}

// Start starts the worker
func (w *Worker) Start() error {
	log.Printf("Starting Temporal worker on task queue: %s", w.config.TaskQueue)
	return w.worker.Start()
}

// Stop stops the worker
func (w *Worker) Stop() {
	log.Printf("Stopping Temporal worker")
	w.worker.Stop()
}

// Run runs the worker until context is cancelled
func (w *Worker) Run(ctx context.Context) error {
	// Start worker
	if err := w.Start(); err != nil {
		return fmt.Errorf("failed to start worker: %w", err)
	}

	// Wait for context cancellation
	<-ctx.Done()

	// Stop worker
	w.Stop()

	return ctx.Err()
}

// GetLLMRegistry returns the LLM registry
func (w *Worker) GetLLMRegistry() llm.ProviderRegistry {
	return w.llmRegistry
}

// GetActivities returns the activities instance
func (w *Worker) GetActivities() *activities.Activities {
	return w.activities
}
