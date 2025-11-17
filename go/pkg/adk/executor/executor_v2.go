package executor

import (
	"context"
	"fmt"
	"time"

	"github.com/kagent-dev/kagent/go/pkg/adk/converters"
	apperrors "github.com/kagent-dev/kagent/go/pkg/adk/errors"
	"github.com/kagent-dev/kagent/go/pkg/adk/llm"
	"github.com/kagent-dev/kagent/go/pkg/adk/session"
	"github.com/kagent-dev/kagent/go/pkg/adk/tools"
)

const (
	MaxIterations     = 10 // Maximum agent iterations
	DefaultIterations = 5  // Default if not specified
)

// A2AExecutorV2 executes agent tasks with real LLM integration
type A2AExecutorV2 struct {
	sessionService session.Service
	pathManager    *session.PathManager
	tools          []tools.Tool
	llmClient      llm.Client
}

// NewA2AExecutorV2 creates a new executor with LLM client
func NewA2AExecutorV2(
	sessionService session.Service,
	pathManager *session.PathManager,
	tools []tools.Tool,
	llmClient llm.Client,
) *A2AExecutorV2 {
	return &A2AExecutorV2{
		sessionService: sessionService,
		pathManager:    pathManager,
		tools:          tools,
		llmClient:      llmClient,
	}
}

// Execute runs an agent task with LLM and tool calling
func (e *A2AExecutorV2) Execute(
	ctx context.Context,
	requestCtx *converters.RequestContext,
	eventQueue chan<- *converters.Event,
) error {
	// 1. Get or create session
	sess, err := e.getOrCreateSession(ctx, requestCtx)
	if err != nil {
		return apperrors.New(apperrors.ErrCodeExecutorFailed, "failed to get session", err)
	}

	// 2. Initialize session path
	sessionPath, err := e.pathManager.Get(sess.ID)
	if err != nil {
		return apperrors.New(apperrors.ErrCodeExecutorFailed, "failed to initialize session path", err)
	}

	// 3. Create tool context
	toolCtx := &tools.Context{
		Session:     sess,
		SessionPath: sessionPath,
		InvocationContext: &session.InvocationContext{
			SessionID: sess.ID,
			UserID:    requestCtx.UserID,
			TaskID:    requestCtx.TaskID,
			ContextID: requestCtx.ContextID,
		},
	}

	// 4. Build message history
	messages, err := e.buildMessageHistory(ctx, sess, requestCtx)
	if err != nil {
		return apperrors.New(apperrors.ErrCodeExecutorFailed, "failed to build message history", err)
	}

	// 5. Send start event
	eventQueue <- &converters.Event{
		Type:      converters.EventTypeStart,
		Timestamp: time.Now(),
	}

	// 6. Run agent loop with tool calling
	maxIter := DefaultIterations
	if err := e.runAgentLoop(ctx, messages, maxIter, toolCtx, eventQueue); err != nil {
		// Send error event
		eventQueue <- &converters.Event{
			Type: converters.EventTypeError,
			Error: &converters.ErrorInfo{
				Code:    apperrors.ErrCodeExecutorFailed,
				Message: "Agent execution failed",
				Details: err.Error(),
			},
			Timestamp: time.Now(),
		}
		return err
	}

	// 7. Send completion event
	eventQueue <- &converters.Event{
		Type:      converters.EventTypeComplete,
		Timestamp: time.Now(),
	}

	return nil
}

func (e *A2AExecutorV2) runAgentLoop(
	ctx context.Context,
	messages []*converters.Content,
	maxIterations int,
	toolCtx *tools.Context,
	eventQueue chan<- *converters.Event,
) error {
	// Build tool definitions
	toolDefs := e.buildToolDefinitions()

	for iteration := 0; iteration < maxIterations; iteration++ {
		// Generate LLM response
		genConfig := &llm.GenerateConfig{
			Tools: toolDefs,
		}

		response, err := e.llmClient.Generate(ctx, messages, genConfig)
		if err != nil {
			return fmt.Errorf("LLM generation failed: %w", err)
		}

		// Add assistant response to messages
		messages = append(messages, response.Content)

		// Send content event
		eventQueue <- &converters.Event{
			Type:      converters.EventTypeContent,
			Content:   response.Content,
			Timestamp: time.Now(),
		}

		// Check if there are tool calls
		if len(response.ToolCalls) == 0 {
			// No tool calls, agent is done
			return nil
		}

		// Execute tool calls
		toolResults, err := e.executeToolCalls(ctx, response.ToolCalls, toolCtx, eventQueue)
		if err != nil {
			return fmt.Errorf("tool execution failed: %w", err)
		}

		// Add tool results to messages
		for _, result := range toolResults {
			messages = append(messages, result)
		}

		// Check for cancellation
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
	}

	// Max iterations reached
	return fmt.Errorf("max iterations (%d) reached", maxIterations)
}

func (e *A2AExecutorV2) executeToolCalls(
	ctx context.Context,
	toolCalls []llm.ToolCall,
	toolCtx *tools.Context,
	eventQueue chan<- *converters.Event,
) ([]*converters.Content, error) {
	var results []*converters.Content

	for _, tc := range toolCalls {
		// Send tool call event
		eventQueue <- &converters.Event{
			Type: converters.EventTypeToolCall,
			Metadata: map[string]interface{}{
				"tool_name": tc.Name,
				"tool_id":   tc.ID,
			},
			Timestamp: time.Now(),
		}

		// Execute the tool
		result, err := e.executeTool(ctx, tc.Name, tc.Arguments, toolCtx)
		if err != nil {
			// Tool execution failed, but continue
			result = fmt.Sprintf("Error executing tool %s: %v", tc.Name, err)
		}

		// Send tool response event
		eventQueue <- &converters.Event{
			Type: converters.EventTypeToolResponse,
			Metadata: map[string]interface{}{
				"tool_name": tc.Name,
				"tool_id":   tc.ID,
				"result":    result,
			},
			Timestamp: time.Now(),
		}

		// Create tool result message
		toolResult := &converters.Content{
			Role: "user", // Tool results come back as user messages
			Parts: []*converters.Part{
				{
					Type: converters.PartTypeFunctionResponse,
					Data: &converters.FunctionResponseData{
						Name:     tc.Name,
						Response: result,
						ID:       tc.ID,
					},
				},
			},
		}

		results = append(results, toolResult)
	}

	return results, nil
}

func (e *A2AExecutorV2) executeTool(
	ctx context.Context,
	toolName string,
	args map[string]interface{},
	toolCtx *tools.Context,
) (string, error) {
	// Find tool by name
	for _, tool := range e.tools {
		if tool.Name() == toolName {
			return tool.RunAsync(ctx, args, toolCtx)
		}
	}

	return "", apperrors.New(apperrors.ErrCodeToolExecution,
		fmt.Sprintf("tool not found: %s", toolName), nil)
}

func (e *A2AExecutorV2) buildToolDefinitions() []llm.ToolDefinition {
	var defs []llm.ToolDefinition

	for _, tool := range e.tools {
		// Build JSON schema for tool parameters
		// For now, using a simple schema
		def := llm.ToolDefinition{
			Name:        tool.Name(),
			Description: tool.Description(),
			Parameters: map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
				// Tool-specific parameters would be defined here
			},
		}

		defs = append(defs, def)
	}

	return defs
}

func (e *A2AExecutorV2) buildMessageHistory(
	ctx context.Context,
	sess *session.Session,
	requestCtx *converters.RequestContext,
) ([]*converters.Content, error) {
	var messages []*converters.Content

	// Add system message if needed
	// messages = append(messages, &converters.Content{
	//     Role:  "system",
	//     Parts: []*converters.Part{{Type: converters.PartTypeText, Data: &converters.TextPartData{Text: "System prompt"}}},
	// })

	// TODO: Load previous messages from session history

	// Add current message from request
	reqConverter := converters.NewRequestConverter()
	runArgs, err := reqConverter.Convert(requestCtx)
	if err != nil {
		return nil, err
	}

	if runArgs.NewMessage != nil {
		messages = append(messages, runArgs.NewMessage)
	}

	return messages, nil
}

func (e *A2AExecutorV2) getOrCreateSession(ctx context.Context, requestCtx *converters.RequestContext) (*session.Session, error) {
	userID := requestCtx.UserID
	if userID == "" {
		userID = requestCtx.ContextID
	}

	sessionID := requestCtx.SessionID
	if sessionID == "" {
		sessionID = requestCtx.TaskID
	}

	// Try to get existing session
	sess, err := e.sessionService.GetSession(ctx, "default", userID, sessionID)
	if err == nil {
		return sess, nil
	}

	// Create new session
	req := &session.CreateSessionRequest{
		AppName: "default",
		UserID:  userID,
	}

	return e.sessionService.CreateSession(ctx, req)
}
