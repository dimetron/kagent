# ADK Converters

This package implements converters for translating between the A2A (Agent-to-Agent) protocol and the internal ADK representation.

## Overview

The converters handle bidirectional translation between:
- **A2A Protocol** - Used for external communication with KAgent and other agents
- **ADK Internal Format** - Used for agent execution and tool invocation

## Components

### Part Converter (`part_converter.go`)

Handles conversion between A2A message parts and ADK content parts.

**Supported Part Types:**
- Text parts
- File parts (URI and inline data)
- Function calls and responses
- Code execution results
- Executable code

**Example:**
```go
converter := converters.NewPartConverter()

// A2A → ADK
a2aParts := []protocol.Part{
    &protocol.TextPart{Text: "Hello world"},
}
adkParts, err := converter.ConvertA2AToContent(a2aParts)

// ADK → A2A
adkParts := []*converters.Part{
    {
        Type: converters.PartTypeText,
        Data: &converters.TextPartData{Text: "Response"},
    },
}
a2aParts, err := converter.ConvertContentToA2A(adkParts)
```

### Request Converter (`request_converter.go`)

Converts A2A requests to ADK RunArgs for agent execution.

**Features:**
- Extracts user ID and session ID from request context
- Converts message parts to ADK content
- Handles message history conversion

**Example:**
```go
converter := converters.NewRequestConverter()

requestCtx := &converters.RequestContext{
    SessionID: "session-123",
    UserID:    "user-456",
    TaskID:    "task-789",
    ContextID: "ctx-abc",
    Message:   &a2aMessage,
}

runArgs, err := converter.Convert(requestCtx)
// runArgs contains UserID, SessionID, NewMessage, RunConfig
```

### Event Converter (`event_converter.go`)

Converts ADK execution events to A2A streaming events.

**Event Types:**
- Start - Agent execution started
- Content - Content/response generated
- Tool Call - Tool execution initiated
- Tool Response - Tool execution completed
- Error - Execution error occurred
- Complete - Agent execution completed
- State Update - Task state changed

**Task States:**
- `WORKING` - Agent is processing (default)
- `AUTH_REQUIRED` - Requires user authentication
- `INPUT_REQUIRED` - Requires user input
- `FAILED` - Execution failed with error
- `COMPLETED` - Execution completed successfully

**Example:**
```go
converter := converters.NewEventConverter()

event := &converters.Event{
    Type: converters.EventTypeContent,
    Content: &converters.Content{
        Role: "assistant",
        Parts: []*converters.Part{
            {
                Type: converters.PartTypeText,
                Data: &converters.TextPartData{Text: "Response"},
            },
        },
    },
    Timestamp: time.Now(),
}

invCtx := &converters.InvocationContext{
    SessionID: "session-123",
    UserID:    "user-456",
    TaskID:    "task-789",
    ContextID: "ctx-abc",
}

a2aEvents, err := converter.Convert(event, invCtx, "task-789", "ctx-abc")
// Returns []protocol.StreamingMessageEvent
```

### Stream Conversion

Convert a stream of ADK events to A2A events:

```go
adkEvents := make(chan *converters.Event)
a2aEvents, errors := converter.ConvertEventsStream(
    adkEvents,
    invCtx,
    "task-789",
    "ctx-abc",
)

// Consume A2A events
for a2aEvent := range a2aEvents {
    // Handle event
}

// Check for errors
if err := <-errors; err != nil {
    // Handle error
}
```

## Types

### Content
Represents message content with role and parts.

```go
type Content struct {
    Role  string  `json:"role"`  // "user", "assistant", "system"
    Parts []*Part `json:"parts"`
}
```

### Part
Represents a message part with type and data.

```go
type Part struct {
    Type string      `json:"type"`
    Data interface{} `json:"data"`
}
```

**Part Types:**
- `text` - Text content
- `file` - File content (URI or bytes)
- `function_call` - Function/tool call
- `function_response` - Function/tool response
- `code_execution` - Code execution result
- `executable_code` - Executable code

### RunArgs
Arguments for running an agent.

```go
type RunArgs struct {
    UserID     string
    SessionID  string
    NewMessage *Content
    RunConfig  *RunConfig
}
```

### Event
Agent execution event.

```go
type Event struct {
    Type      string
    Content   *Content
    Error     *ErrorInfo
    Metadata  map[string]interface{}
    Timestamp time.Time
}
```

## Error Handling

All converters use typed errors from the `errors` package:

```go
import apperrors "github.com/kagent-dev/kagent/go/pkg/adk/errors"

if err != nil {
    if adkErr, ok := err.(*apperrors.AppError); ok {
        switch adkErr.Code {
        case apperrors.ErrCodeConversion:
            // Handle conversion error
        case apperrors.ErrCodeInvalidInput:
            // Handle invalid input
        }
    }
}
```

## Integration

The converters are used by the A2A executor to bridge the gap between the A2A protocol and ADK internals:

```go
// In executor
requestConverter := converters.NewRequestConverter()
eventConverter := converters.NewEventConverter()

// Convert incoming request
runArgs, err := requestConverter.Convert(requestCtx)

// Execute agent (generates ADK events)
adkEvents := make(chan *converters.Event)

// Convert outgoing events
a2aEvents, errors := eventConverter.ConvertEventsStream(
    adkEvents,
    invCtx,
    taskID,
    contextID,
)
```

## Testing

Example test cases:

```go
func TestPartConverter(t *testing.T) {
    converter := converters.NewPartConverter()

    // Test text conversion
    a2aParts := []protocol.Part{
        &protocol.TextPart{Text: "Hello"},
    }

    adkParts, err := converter.ConvertA2AToContent(a2aParts)
    assert.NoError(t, err)
    assert.Len(t, adkParts, 1)
    assert.Equal(t, converters.PartTypeText, adkParts[0].Type)
}
```

## Best Practices

1. **Always handle errors** - Converters can fail on invalid input
2. **Use appropriate part types** - Choose the most specific type for your data
3. **Include metadata** - Add context information to events for debugging
4. **Stream events** - Use streaming for long-running operations
5. **Set timestamps** - Include timestamps for event ordering

## Performance Considerations

- Part conversion is O(n) where n is the number of parts
- Event streaming uses buffered channels (default 10)
- JSON marshaling/unmarshaling is used for data conversion
- Consider pooling converters for high-throughput scenarios

## Future Enhancements

- [ ] Support for binary data compression
- [ ] Batch conversion for multiple messages
- [ ] Custom part type registry
- [ ] Converter middleware/plugins
- [ ] Performance profiling and optimization
