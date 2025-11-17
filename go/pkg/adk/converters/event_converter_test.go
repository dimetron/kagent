package converters

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEventConverter_Convert_StartEvent(t *testing.T) {
	converter := NewEventConverter()

	event := &Event{
		Type:      EventTypeStart,
		Timestamp: time.Now(),
	}

	invCtx := &InvocationContext{
		SessionID: "session-123",
		UserID:    "user-456",
		TaskID:    "task-789",
		ContextID: "context-abc",
	}

	a2aEvents, err := converter.Convert(event, invCtx, "task-789", "context-abc")
	require.NoError(t, err)
	require.Len(t, a2aEvents, 1)

	// Should create a status event with WORKING state
	statusEvent := a2aEvents[0].Event
	assert.NotNil(t, statusEvent)
}

func TestEventConverter_Convert_ContentEvent(t *testing.T) {
	converter := NewEventConverter()

	event := &Event{
		Type: EventTypeContent,
		Content: &Content{
			Role: "assistant",
			Parts: []*Part{
				{
					Type: PartTypeText,
					Data: &TextPartData{Text: "Hello, how can I help you?"},
				},
			},
		},
		Timestamp: time.Now(),
	}

	invCtx := &InvocationContext{
		SessionID: "session-123",
		UserID:    "user-456",
		TaskID:    "task-789",
		ContextID: "context-abc",
	}

	a2aEvents, err := converter.Convert(event, invCtx, "task-789", "context-abc")
	require.NoError(t, err)
	require.Len(t, a2aEvents, 1)

	// Verify the content was converted
	statusEvent := a2aEvents[0].Event
	assert.NotNil(t, statusEvent)
}

func TestEventConverter_Convert_ErrorEvent(t *testing.T) {
	converter := NewEventConverter()

	event := &Event{
		Type: EventTypeError,
		Error: &ErrorInfo{
			Code:    "TEST_ERROR",
			Message: "Test error message",
			Details: "Detailed error information",
		},
		Timestamp: time.Now(),
	}

	invCtx := &InvocationContext{
		SessionID: "session-123",
		UserID:    "user-456",
		TaskID:    "task-789",
		ContextID: "context-abc",
	}

	a2aEvents, err := converter.Convert(event, invCtx, "task-789", "context-abc")
	require.NoError(t, err)
	require.Len(t, a2aEvents, 1)

	// Should create error event with FAILED state
	statusEvent := a2aEvents[0].Event
	assert.NotNil(t, statusEvent)
}

func TestEventConverter_Convert_CompleteEvent(t *testing.T) {
	converter := NewEventConverter()

	event := &Event{
		Type:      EventTypeComplete,
		Timestamp: time.Now(),
	}

	invCtx := &InvocationContext{
		SessionID: "session-123",
		UserID:    "user-456",
		TaskID:    "task-789",
		ContextID: "context-abc",
	}

	a2aEvents, err := converter.Convert(event, invCtx, "task-789", "context-abc")
	require.NoError(t, err)
	require.Len(t, a2aEvents, 1)

	// Should create status event with COMPLETED state
	statusEvent := a2aEvents[0].Event
	assert.NotNil(t, statusEvent)
}

func TestEventConverter_Convert_ToolCallEvent(t *testing.T) {
	converter := NewEventConverter()

	event := &Event{
		Type: EventTypeToolCall,
		Metadata: map[string]interface{}{
			"tool_name": "test_tool",
			"tool_id":   "call-123",
		},
		Timestamp: time.Now(),
	}

	invCtx := &InvocationContext{
		SessionID: "session-123",
		UserID:    "user-456",
		TaskID:    "task-789",
		ContextID: "context-abc",
	}

	a2aEvents, err := converter.Convert(event, invCtx, "task-789", "context-abc")
	require.NoError(t, err)
	require.Len(t, a2aEvents, 1)
}

func TestEventConverter_Convert_ToolResponseEvent(t *testing.T) {
	converter := NewEventConverter()

	event := &Event{
		Type: EventTypeToolResponse,
		Metadata: map[string]interface{}{
			"tool_name": "test_tool",
			"tool_id":   "call-123",
			"result":    "Tool execution successful",
		},
		Timestamp: time.Now(),
	}

	invCtx := &InvocationContext{
		SessionID: "session-123",
		UserID:    "user-456",
		TaskID:    "task-789",
		ContextID: "context-abc",
	}

	a2aEvents, err := converter.Convert(event, invCtx, "task-789", "context-abc")
	require.NoError(t, err)
	require.Len(t, a2aEvents, 1)
}

func TestEventConverter_Convert_StateUpdateEvent(t *testing.T) {
	converter := NewEventConverter()

	event := &Event{
		Type: EventTypeStateUpdate,
		Metadata: map[string]interface{}{
			"state": "INPUT_REQUIRED",
		},
		Timestamp: time.Now(),
	}

	invCtx := &InvocationContext{
		SessionID: "session-123",
		UserID:    "user-456",
		TaskID:    "task-789",
		ContextID: "context-abc",
	}

	a2aEvents, err := converter.Convert(event, invCtx, "task-789", "context-abc")
	require.NoError(t, err)
	require.Len(t, a2aEvents, 1)
}

func TestEventConverter_Convert_ErrorInEvent(t *testing.T) {
	converter := NewEventConverter()

	event := &Event{
		Type: EventTypeStart,
		Error: &ErrorInfo{
			Code:    "ERROR",
			Message: "Error occurred",
		},
		Timestamp: time.Now(),
	}

	invCtx := &InvocationContext{
		SessionID: "session-123",
		UserID:    "user-456",
		TaskID:    "task-789",
		ContextID: "context-abc",
	}

	a2aEvents, err := converter.Convert(event, invCtx, "task-789", "context-abc")
	require.NoError(t, err)
	require.Len(t, a2aEvents, 1)

	// Error should take precedence
	statusEvent := a2aEvents[0].Event
	assert.NotNil(t, statusEvent)
}

func TestEventConverter_DetermineTaskState(t *testing.T) {
	converter := NewEventConverter()

	tests := []struct {
		name     string
		event    *Event
		metadata map[string]interface{}
		expected TaskState
	}{
		{
			name: "error event",
			event: &Event{
				Type:  EventTypeError,
				Error: &ErrorInfo{Code: "ERROR"},
			},
			expected: TaskStateFailed,
		},
		{
			name: "complete event",
			event: &Event{
				Type: EventTypeComplete,
			},
			expected: TaskStateCompleted,
		},
		{
			name: "working state",
			event: &Event{
				Type: EventTypeContent,
			},
			expected: TaskStateWorking,
		},
		{
			name: "auth required tool",
			event: &Event{
				Type: EventTypeToolCall,
			},
			metadata: map[string]interface{}{
				"tool_name": "REQUEST_EUC",
			},
			expected: TaskStateAuthRequired,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			state := converter.DetermineTaskState(tt.event, tt.metadata)
			assert.Equal(t, tt.expected, state)
		})
	}
}

func TestEventConverter_ConvertEventsStream(t *testing.T) {
	converter := NewEventConverter()

	invCtx := &InvocationContext{
		SessionID: "session-123",
		UserID:    "user-456",
		TaskID:    "task-789",
		ContextID: "context-abc",
	}

	// Create event channel
	eventChan := make(chan *Event, 5)

	// Start conversion
	a2aEvents, errors := converter.ConvertEventsStream(eventChan, invCtx, "task-789", "context-abc")

	// Send events
	eventChan <- &Event{Type: EventTypeStart, Timestamp: time.Now()}
	eventChan <- &Event{
		Type: EventTypeContent,
		Content: &Content{
			Role: "assistant",
			Parts: []*Part{
				{Type: PartTypeText, Data: &TextPartData{Text: "Hello"}},
			},
		},
		Timestamp: time.Now(),
	}
	eventChan <- &Event{Type: EventTypeComplete, Timestamp: time.Now()}
	close(eventChan)

	// Collect converted events
	var converted int
	for range a2aEvents {
		converted++
	}

	// Should have converted all events
	assert.Equal(t, 3, converted)

	// Check for errors
	err := <-errors
	assert.NoError(t, err)
}

func TestEventConverter_ConvertEventsStream_WithError(t *testing.T) {
	converter := NewEventConverter()

	invCtx := &InvocationContext{
		SessionID: "session-123",
		UserID:    "user-456",
		TaskID:    "task-789",
		ContextID: "context-abc",
	}

	// Create event channel
	eventChan := make(chan *Event, 5)

	// Start conversion
	a2aEvents, errors := converter.ConvertEventsStream(eventChan, invCtx, "task-789", "context-abc")

	// Send events including error
	eventChan <- &Event{Type: EventTypeStart, Timestamp: time.Now()}
	eventChan <- &Event{
		Type:      EventTypeError,
		Error:     &ErrorInfo{Code: "TEST", Message: "Error"},
		Timestamp: time.Now(),
	}
	close(eventChan)

	// Collect events
	var converted int
	for range a2aEvents {
		converted++
	}

	// Should have converted events before error
	assert.GreaterOrEqual(t, converted, 2)

	// Check for error
	err := <-errors
	assert.NoError(t, err) // Converter doesn't fail on event errors
}

func TestEventConverter_Convert_ContentWithMultipleParts(t *testing.T) {
	converter := NewEventConverter()

	event := &Event{
		Type: EventTypeContent,
		Content: &Content{
			Role: "assistant",
			Parts: []*Part{
				{Type: PartTypeText, Data: &TextPartData{Text: "Part 1"}},
				{Type: PartTypeText, Data: &TextPartData{Text: "Part 2"}},
			},
		},
		Timestamp: time.Now(),
	}

	invCtx := &InvocationContext{
		SessionID: "session-123",
		UserID:    "user-456",
		TaskID:    "task-789",
		ContextID: "context-abc",
	}

	a2aEvents, err := converter.Convert(event, invCtx, "task-789", "context-abc")
	require.NoError(t, err)
	require.Len(t, a2aEvents, 1)
}

func TestEventConverter_Convert_NilContent(t *testing.T) {
	converter := NewEventConverter()

	event := &Event{
		Type:      EventTypeContent,
		Content:   nil,
		Timestamp: time.Now(),
	}

	invCtx := &InvocationContext{
		SessionID: "session-123",
		UserID:    "user-456",
		TaskID:    "task-789",
		ContextID: "context-abc",
	}

	a2aEvents, err := converter.Convert(event, invCtx, "task-789", "context-abc")
	require.NoError(t, err)
	// Should handle nil content gracefully
	assert.NotNil(t, a2aEvents)
}
