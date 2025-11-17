package converters

import (
	"fmt"
	"time"

	apperrors "github.com/kagent-dev/kagent/go/pkg/adk/errors"
	"trpc.group/trpc-go/trpc-a2a-go/protocol"
)

// EventConverter handles conversion from ADK events to A2A events
type EventConverter struct {
	partConverter *PartConverter
}

// NewEventConverter creates a new EventConverter
func NewEventConverter() *EventConverter {
	return &EventConverter{
		partConverter: NewPartConverter(),
	}
}

// Convert converts an ADK event to A2A streaming events
func (c *EventConverter) Convert(
	event *Event,
	invocationCtx *InvocationContext,
	taskID, contextID string,
) ([]*protocol.StreamingMessageEvent, error) {
	var events []*protocol.StreamingMessageEvent

	// Check for errors first
	if event.Error != nil {
		errorEvent := c.createErrorEvent(event.Error, taskID, contextID)
		events = append(events, errorEvent)
		return events, nil
	}

	// Create status update based on event type
	switch event.Type {
	case EventTypeStart:
		events = append(events, c.createStatusEvent(taskID, contextID, TaskStateWorking, nil))

	case EventTypeContent:
		if event.Content != nil {
			statusEvent, err := c.createContentEvent(event.Content, taskID, contextID)
			if err != nil {
				return nil, err
			}
			events = append(events, statusEvent)
		}

	case EventTypeToolCall:
		// Tool execution in progress
		events = append(events, c.createStatusEvent(taskID, contextID, TaskStateWorking, event.Metadata))

	case EventTypeToolResponse:
		// Tool execution completed
		events = append(events, c.createStatusEvent(taskID, contextID, TaskStateWorking, event.Metadata))

	case EventTypeComplete:
		events = append(events, c.createStatusEvent(taskID, contextID, TaskStateCompleted, nil))

	case EventTypeError:
		if event.Error != nil {
			errorEvent := c.createErrorEvent(event.Error, taskID, contextID)
			events = append(events, errorEvent)
		}

	case EventTypeStateUpdate:
		// Extract state from metadata
		state := TaskStateWorking
		if event.Metadata != nil {
			if s, ok := event.Metadata["state"].(string); ok {
				state = TaskState(s)
			}
		}
		events = append(events, c.createStatusEvent(taskID, contextID, state, event.Metadata))
	}

	return events, nil
}

func (c *EventConverter) createStatusEvent(
	taskID, contextID string,
	state TaskState,
	metadata map[string]interface{},
) *protocol.StreamingMessageEvent {
	statusUpdate := &protocol.TaskStatusUpdateEvent{
		TaskID:    taskID,
		ContextID: contextID,
		State:     string(state),
		Metadata:  metadata,
	}

	return &protocol.StreamingMessageEvent{
		Event: statusUpdate,
	}
}

func (c *EventConverter) createContentEvent(
	content *Content,
	taskID, contextID string,
) (*protocol.StreamingMessageEvent, error) {
	// Convert content parts to A2A parts
	a2aParts, err := c.partConverter.ConvertContentToA2A(content.Parts)
	if err != nil {
		return nil, apperrors.New(apperrors.ErrCodeConversion, "failed to convert content parts", err)
	}

	// Create message
	message := &protocol.Message{
		MessageID: protocol.GenerateMessageID(),
		Kind:      protocol.KindMessage,
		Parts:     a2aParts,
		Metadata: map[string]interface{}{
			"role": content.Role,
		},
	}

	// Create status update with content
	statusUpdate := &protocol.TaskStatusUpdateEvent{
		TaskID:    taskID,
		ContextID: contextID,
		State:     string(TaskStateWorking),
		Content:   message,
	}

	return &protocol.StreamingMessageEvent{
		Event: statusUpdate,
	}, nil
}

func (c *EventConverter) createErrorEvent(
	errorInfo *ErrorInfo,
	taskID, contextID string,
) *protocol.StreamingMessageEvent {
	// Create error message
	errorMsg := errorInfo.Message
	if errorInfo.Details != "" {
		errorMsg = fmt.Sprintf("%s: %s", errorMsg, errorInfo.Details)
	}

	message := &protocol.Message{
		MessageID: protocol.GenerateMessageID(),
		Kind:      protocol.KindMessage,
		Parts: []protocol.Part{
			&protocol.TextPart{
				Text: fmt.Sprintf("Error [%s]: %s", errorInfo.Code, errorMsg),
			},
		},
		Metadata: map[string]interface{}{
			"error_code": errorInfo.Code,
		},
	}

	statusUpdate := &protocol.TaskStatusUpdateEvent{
		TaskID:    taskID,
		ContextID: contextID,
		State:     string(TaskStateFailed),
		Content:   message,
		Metadata: map[string]interface{}{
			"error": errorInfo,
		},
	}

	return &protocol.StreamingMessageEvent{
		Event: statusUpdate,
	}
}

// DetermineTaskState analyzes event and context to determine appropriate task state
func (c *EventConverter) DetermineTaskState(event *Event, metadata map[string]interface{}) TaskState {
	if event.Error != nil {
		return TaskStateFailed
	}

	switch event.Type {
	case EventTypeComplete:
		return TaskStateCompleted
	case EventTypeError:
		return TaskStateFailed
	default:
		// Check metadata for state hints
		if metadata != nil {
			if toolName, ok := metadata["tool_name"].(string); ok {
				// Long-running tools may require auth or input
				if toolName == "REQUEST_EUC" {
					return TaskStateAuthRequired
				}
				// Other long-running tools might need input
				if isLongRunningTool(toolName) {
					return TaskStateInputRequired
				}
			}
		}
		return TaskStateWorking
	}
}

func isLongRunningTool(toolName string) bool {
	longRunningTools := map[string]bool{
		"REQUEST_EUC":   true,
		"REQUEST_INPUT": true,
		"WAIT_FOR_USER": true,
	}
	return longRunningTools[toolName]
}

// ConvertEventsStream converts a stream of ADK events to A2A events
func (c *EventConverter) ConvertEventsStream(
	events <-chan *Event,
	invocationCtx *InvocationContext,
	taskID, contextID string,
) (<-chan *protocol.StreamingMessageEvent, <-chan error) {
	a2aEvents := make(chan *protocol.StreamingMessageEvent, 10)
	errors := make(chan error, 1)

	go func() {
		defer close(a2aEvents)
		defer close(errors)

		for event := range events {
			converted, err := c.Convert(event, invocationCtx, taskID, contextID)
			if err != nil {
				errors <- err
				return
			}

			for _, a2aEvent := range converted {
				// Add timestamp to metadata
				if statusEvent, ok := a2aEvent.Event.(*protocol.TaskStatusUpdateEvent); ok {
					if statusEvent.Metadata == nil {
						statusEvent.Metadata = make(map[string]interface{})
					}
					statusEvent.Metadata["timestamp"] = time.Now().UTC().Format(time.RFC3339)
				}

				select {
				case a2aEvents <- a2aEvent:
				default:
					errors <- apperrors.New(apperrors.ErrCodeConversion, "event channel full", nil)
					return
				}
			}
		}
	}()

	return a2aEvents, errors
}
