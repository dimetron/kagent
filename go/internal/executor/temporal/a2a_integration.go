package temporal

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/kagent-dev/kagent/go/internal/executor/temporal/activities"
	"github.com/kagent-dev/kagent/go/internal/executor/temporal/models"
	v1alpha2 "github.com/kagent-dev/kagent/go/api/v1alpha2"
	"trpc.group/trpc-go/trpc-a2a-go/protocol"
)

// A2AExecutor implements agent execution using Temporal and A2A protocol
type A2AExecutor struct {
	executorService *ExecutorService
	eventPublisher  activities.EventPublisher
}

// NewA2AExecutor creates a new A2A-compatible executor
func NewA2AExecutor(executorService *ExecutorService, eventPublisher activities.EventPublisher) *A2AExecutor {
	return &A2AExecutor{
		executorService: executorService,
		eventPublisher:  eventPublisher,
	}
}

// HandleA2AMessage handles an A2A message and starts execution
func (e *A2AExecutor) HandleA2AMessage(ctx context.Context, message *protocol.MessageSendParams) (*models.ExecutionResponse, error) {
	// Convert A2A message to execution request
	request, err := e.convertA2AToExecutionRequest(message)
	if err != nil {
		return nil, fmt.Errorf("failed to convert A2A message: %w", err)
	}

	// Execute asynchronously
	workflowRun, err := e.executorService.ExecuteAsync(ctx, *request)
	if err != nil {
		return nil, fmt.Errorf("failed to start execution: %w", err)
	}

	// Return immediately with submitted status
	response := &models.ExecutionResponse{
		TaskID:        request.TaskID,
		Status:        models.TaskStatusSubmitted,
		WorkflowID:    workflowRun.GetID(),
		WorkflowRunID: workflowRun.GetRunID(),
	}

	// Start event streaming in background
	go e.streamEventsToA2A(ctx, request.TaskID, workflowRun)

	return response, nil
}

// convertA2AToExecutionRequest converts A2A message to execution request
func (e *A2AExecutor) convertA2AToExecutionRequest(message *protocol.MessageSendParams) (*models.ExecutionRequest, error) {
	// Extract user message
	userMessage := ""
	if len(message.Content) > 0 {
		if textPart, ok := message.Content[0].(*protocol.TextPart); ok {
			userMessage = textPart.Text
		}
	}

	// Create execution request
	request := &models.ExecutionRequest{
		TaskID:        message.TaskId,
		SessionID:     message.SessionId,
		UserID:        extractUserID(message),
		UserMessage:   userMessage,
		MaxIterations: 10, // Default
		Timeout:       5 * time.Minute,
		Metadata:      make(map[string]interface{}),
	}

	// Parse metadata if present
	if message.Metadata != nil {
		var metadata map[string]interface{}
		if err := json.Unmarshal([]byte(*message.Metadata), &metadata); err == nil {
			request.Metadata = metadata

			// Extract agent configuration from metadata
			if systemMsg, ok := metadata["system_message"].(string); ok {
				request.SystemMessage = systemMsg
			}

			if maxIter, ok := metadata["max_iterations"].(float64); ok {
				request.MaxIterations = int(maxIter)
			}

			if requireApproval, ok := metadata["require_approval"].(bool); ok {
				request.RequireApproval = requireApproval
			}

			// Extract model config
			if modelConfig, ok := metadata["model_config"].(map[string]interface{}); ok {
				request.ModelConfig = parseModelConfig(modelConfig)
			}

			// Extract tools
			if tools, ok := metadata["tools"].([]interface{}); ok {
				request.Tools = parseTools(tools)
			}
		}
	}

	// Set defaults if not provided
	if request.ModelConfig.Provider == "" {
		request.ModelConfig = models.ModelConfig{
			Provider:    "anthropic",
			Model:       "claude-3-5-sonnet-20241022",
			Temperature: 0.7,
			MaxTokens:   4096,
		}
	}

	return request, nil
}

// streamEventsToA2A streams execution events to A2A protocol
func (e *A2AExecutor) streamEventsToA2A(ctx context.Context, taskID string, workflowRun interface{}) {
	// Subscribe to events
	eventChan := e.executorService.StreamEvents(ctx, taskID)

	for {
		select {
		case event, ok := <-eventChan:
			if !ok {
				return
			}

			// Convert and publish event
			if err := e.publishA2AEvent(ctx, event); err != nil {
				// Log error but continue
				fmt.Printf("Failed to publish A2A event: %v\n", err)
			}

		case <-ctx.Done():
			return
		}
	}
}

// publishA2AEvent publishes an event to A2A protocol
func (e *A2AExecutor) publishA2AEvent(ctx context.Context, event models.A2AEvent) error {
	switch event.EventType {
	case "status_update":
		statusUpdate, err := activities.ConvertToA2ATaskStatusUpdate(event)
		if err != nil {
			return err
		}
		// TODO: Send to A2A handler
		_ = statusUpdate

	case "artifact_update":
		artifactUpdate, err := activities.ConvertToA2AArtifactUpdate(event)
		if err != nil {
			return err
		}
		// TODO: Send to A2A handler
		_ = artifactUpdate
	}

	return nil
}

// extractUserID extracts user ID from A2A message
func extractUserID(message *protocol.MessageSendParams) string {
	// TODO: Extract from authentication context
	return "default-user"
}

// parseModelConfig parses model configuration from metadata
func parseModelConfig(config map[string]interface{}) models.ModelConfig {
	modelConfig := models.ModelConfig{
		Temperature: 0.7,
		MaxTokens:   4096,
	}

	if provider, ok := config["provider"].(string); ok {
		modelConfig.Provider = provider
	}
	if model, ok := config["model"].(string); ok {
		modelConfig.Model = model
	}
	if temp, ok := config["temperature"].(float64); ok {
		modelConfig.Temperature = temp
	}
	if maxTokens, ok := config["max_tokens"].(float64); ok {
		modelConfig.MaxTokens = int(maxTokens)
	}
	if apiKey, ok := config["api_key"].(string); ok {
		modelConfig.APIKey = apiKey
	}

	return modelConfig
}

// parseTools parses tool definitions from metadata
func parseTools(toolsData []interface{}) []models.Tool {
	tools := make([]models.Tool, 0)

	for _, toolData := range toolsData {
		if toolMap, ok := toolData.(map[string]interface{}); ok {
			tool := models.Tool{}

			if name, ok := toolMap["name"].(string); ok {
				tool.Name = name
			}
			if desc, ok := toolMap["description"].(string); ok {
				tool.Description = desc
			}
			if params, ok := toolMap["parameters"].(map[string]interface{}); ok {
				tool.Parameters = params
			}
			if toolType, ok := toolMap["type"].(string); ok {
				tool.Type = toolType
			}
			if config, ok := toolMap["config"].(map[string]interface{}); ok {
				tool.Config = config
			}

			tools = append(tools, tool)
		}
	}

	return tools
}

// ConvertAgentCRDToExecutionRequest converts Agent CRD to execution request
func ConvertAgentCRDToExecutionRequest(agent *v1alpha2.Agent, userMessage string) (*models.ExecutionRequest, error) {
	request := &models.ExecutionRequest{
		TaskID:         fmt.Sprintf("task-%d", time.Now().Unix()),
		SessionID:      fmt.Sprintf("session-%d", time.Now().Unix()),
		UserMessage:    userMessage,
		SystemMessage:  agent.Spec.SystemMessage,
		MaxIterations:  10,
		Timeout:        5 * time.Minute,
		Metadata:       make(map[string]interface{}),
		Tools:          make([]models.Tool, 0),
	}

	// Parse model config from Agent CRD
	if agent.Spec.ModelConfigRef != nil {
		// TODO: Load ModelConfig from K8s and convert
		request.ModelConfig = models.ModelConfig{
			Provider:    "anthropic",
			Model:       "claude-3-5-sonnet-20241022",
			Temperature: 0.7,
			MaxTokens:   4096,
		}
	}

	// Parse tools from Agent CRD
	for _, toolRef := range agent.Spec.Tools {
		// TODO: Load tool definitions and convert
		tool := models.Tool{
			Name: toolRef.Name,
			Type: "mcp", // Default to MCP
		}
		request.Tools = append(request.Tools, tool)
	}

	return request, nil
}

// A2AHTTPHandler provides HTTP handlers for A2A protocol
type A2AHTTPHandler struct {
	executor *A2AExecutor
}

// NewA2AHTTPHandler creates a new HTTP handler
func NewA2AHTTPHandler(executor *A2AExecutor) *A2AHTTPHandler {
	return &A2AHTTPHandler{
		executor: executor,
	}
}

// HandleMessage handles incoming A2A messages via HTTP
func (h *A2AHTTPHandler) HandleMessage(w http.ResponseWriter, r *http.Request) {
	// Parse request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// Parse A2A message
	var message protocol.MessageSendParams
	if err := json.Unmarshal(body, &message); err != nil {
		http.Error(w, "Invalid A2A message format", http.StatusBadRequest)
		return
	}

	// Handle message
	response, err := h.executor.HandleA2AMessage(r.Context(), &message)
	if err != nil {
		http.Error(w, fmt.Sprintf("Execution failed: %v", err), http.StatusInternalServerError)
		return
	}

	// Return response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
