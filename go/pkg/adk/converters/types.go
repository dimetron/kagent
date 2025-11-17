package converters

import (
	"time"

	"github.com/kagent-dev/kagent/go/pkg/adk/session"
)

// Content represents a message content structure compatible with various LLM providers
type Content struct {
	Role  string  `json:"role"`
	Parts []*Part `json:"parts"`
}

// Part represents a message part (text, file, function call, etc.)
type Part struct {
	Type string      `json:"type"`
	Data interface{} `json:"data"`
}

// PartType constants
const (
	PartTypeText             = "text"
	PartTypeFile             = "file"
	PartTypeFunctionCall     = "function_call"
	PartTypeFunctionResponse = "function_response"
	PartTypeCodeExecution    = "code_execution"
	PartTypeExecutableCode   = "executable_code"
)

// TextPartData represents text content
type TextPartData struct {
	Text string `json:"text"`
}

// FilePartData represents file content
type FilePartData struct {
	URI      string `json:"uri,omitempty"`
	MimeType string `json:"mime_type,omitempty"`
	Data     []byte `json:"data,omitempty"`
}

// FunctionCallData represents a function call
type FunctionCallData struct {
	Name string                 `json:"name"`
	Args map[string]interface{} `json:"args"`
	ID   string                 `json:"id,omitempty"`
}

// FunctionResponseData represents a function response
type FunctionResponseData struct {
	Name     string `json:"name"`
	Response string `json:"response"`
	ID       string `json:"id,omitempty"`
}

// CodeExecutionData represents code execution result
type CodeExecutionData struct {
	Output string `json:"output"`
	Error  string `json:"error,omitempty"`
}

// ExecutableCodeData represents executable code
type ExecutableCodeData struct {
	Language string `json:"language"`
	Code     string `json:"code"`
}

// RunArgs represents arguments for running an agent
type RunArgs struct {
	UserID     string
	SessionID  string
	NewMessage *Content
	RunConfig  *RunConfig
}

// RunConfig represents runtime configuration
type RunConfig struct {
	MaxIterations int                    `json:"max_iterations,omitempty"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
}

// Event represents an agent execution event
type Event struct {
	Type      string                 `json:"type"`
	Content   *Content               `json:"content,omitempty"`
	Error     *ErrorInfo             `json:"error,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
}

// EventType constants
const (
	EventTypeStart         = "start"
	EventTypeContent       = "content"
	EventTypeToolCall      = "tool_call"
	EventTypeToolResponse  = "tool_response"
	EventTypeError         = "error"
	EventTypeComplete      = "complete"
	EventTypeStateUpdate   = "state_update"
)

// ErrorInfo represents error information
type ErrorInfo struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

// TaskState represents task execution state
type TaskState string

const (
	TaskStateWorking       TaskState = "WORKING"
	TaskStateAuthRequired  TaskState = "AUTH_REQUIRED"
	TaskStateInputRequired TaskState = "INPUT_REQUIRED"
	TaskStateFailed        TaskState = "FAILED"
	TaskStateCompleted     TaskState = "COMPLETED"
)

// RequestContext represents the context from an A2A request
type RequestContext struct {
	SessionID string
	UserID    string
	TaskID    string
	ContextID string
	Message   interface{} // A2A Message
}

// InvocationContext contains context for the current invocation
type InvocationContext struct {
	SessionID string
	UserID    string
	TaskID    string
	ContextID string
	Session   *session.Session
}
