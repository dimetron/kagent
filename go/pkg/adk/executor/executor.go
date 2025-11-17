package executor

import (
	"context"
	"fmt"
	"time"

	"github.com/kagent-dev/kagent/go/pkg/adk/converters"
	apperrors "github.com/kagent-dev/kagent/go/pkg/adk/errors"
	"github.com/kagent-dev/kagent/go/pkg/adk/session"
	"github.com/kagent-dev/kagent/go/pkg/adk/tools"
)

// A2AExecutor executes agent tasks and converts between A2A and ADK protocols
type A2AExecutor struct {
	sessionService    session.Service
	pathManager       *session.PathManager
	tools             []tools.Tool
	requestConverter  *converters.RequestConverter
	eventConverter    *converters.EventConverter
}

// NewA2AExecutor creates a new A2AExecutor
func NewA2AExecutor(
	sessionService session.Service,
	pathManager *session.PathManager,
	tools []tools.Tool,
) *A2AExecutor {
	return &A2AExecutor{
		sessionService:   sessionService,
		pathManager:      pathManager,
		tools:            tools,
		requestConverter: converters.NewRequestConverter(),
		eventConverter:   converters.NewEventConverter(),
	}
}

// Execute runs an agent task with A2A protocol conversion
func (e *A2AExecutor) Execute(
	ctx context.Context,
	requestCtx *converters.RequestContext,
	eventQueue chan<- *converters.Event,
) error {
	// 1. Convert A2A request to ADK RunArgs
	runArgs, err := e.requestConverter.Convert(requestCtx)
	if err != nil {
		return apperrors.New(apperrors.ErrCodeExecutorFailed, "failed to convert request", err)
	}

	// 2. Get or create session
	sess, err := e.getOrCreateSession(ctx, runArgs)
	if err != nil {
		return apperrors.New(apperrors.ErrCodeExecutorFailed, "failed to get session", err)
	}

	// 3. Initialize session path
	sessionPath, err := e.pathManager.Get(sess.ID)
	if err != nil {
		return apperrors.New(apperrors.ErrCodeExecutorFailed, "failed to initialize session path", err)
	}

	// 4. Create invocation context
	invCtx := &session.InvocationContext{
		SessionID: sess.ID,
		UserID:    runArgs.UserID,
		TaskID:    requestCtx.TaskID,
		ContextID: requestCtx.ContextID,
	}

	// 5. Send start event
	eventQueue <- &converters.Event{
		Type:      converters.EventTypeStart,
		Timestamp: time.Now(),
	}

	// 6. Execute agent (simplified - actual implementation would call LLM and tools)
	if err := e.executeAgent(ctx, runArgs, sess, sessionPath, invCtx, eventQueue); err != nil {
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

func (e *A2AExecutor) getOrCreateSession(ctx context.Context, runArgs *converters.RunArgs) (*session.Session, error) {
	// Try to get existing session
	sess, err := e.sessionService.GetSession(ctx, "default", runArgs.UserID, runArgs.SessionID)
	if err == nil {
		return sess, nil
	}

	// Create new session
	req := &session.CreateSessionRequest{
		AppName: "default",
		UserID:  runArgs.UserID,
	}

	return e.sessionService.CreateSession(ctx, req)
}

func (e *A2AExecutor) executeAgent(
	ctx context.Context,
	runArgs *converters.RunArgs,
	sess *session.Session,
	sessionPath string,
	invCtx *session.InvocationContext,
	eventQueue chan<- *converters.Event,
) error {
	// This is a simplified implementation
	// In a full implementation, this would:
	// 1. Call the LLM with the message
	// 2. Handle tool calls iteratively
	// 3. Stream events back through the event queue

	// For now, we'll demonstrate a simple echo response
	if runArgs.NewMessage == nil {
		return fmt.Errorf("no message provided")
	}

	// Extract text from message
	var responseText string
	for _, part := range runArgs.NewMessage.Parts {
		if part.Type == converters.PartTypeText {
			if textData, ok := part.Data.(*converters.TextPartData); ok {
				responseText = fmt.Sprintf("Received: %s", textData.Text)
				break
			}
		}
	}

	if responseText == "" {
		responseText = "Message received (no text content)"
	}

	// Send content event with response
	eventQueue <- &converters.Event{
		Type: converters.EventTypeContent,
		Content: &converters.Content{
			Role: "assistant",
			Parts: []*converters.Part{
				{
					Type: converters.PartTypeText,
					Data: &converters.TextPartData{
						Text: responseText,
					},
				},
			},
		},
		Timestamp: time.Now(),
	}

	return nil
}

// ExecuteTool executes a tool by name with given arguments
func (e *A2AExecutor) ExecuteTool(
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
