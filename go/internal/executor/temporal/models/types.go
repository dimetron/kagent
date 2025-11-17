package models

import (
	"time"

	"trpc.group/trpc-go/trpc-a2a-go/protocol"
)

// ExecutionRequest represents a request to execute an agent
type ExecutionRequest struct {
	TaskID          string                 `json:"task_id"`
	SessionID       string                 `json:"session_id"`
	UserID          string                 `json:"user_id"`
	SystemMessage   string                 `json:"system_message"`
	UserMessage     string                 `json:"user_message"`
	Tools           []Tool                 `json:"tools"`
	ModelConfig     ModelConfig            `json:"model_config"`
	MaxIterations   int                    `json:"max_iterations"`
	Timeout         time.Duration          `json:"timeout"`
	Metadata        map[string]interface{} `json:"metadata"`
	RequireApproval bool                   `json:"require_approval"` // Enable HITL
}

// ExecutionResponse represents the final result of an execution
type ExecutionResponse struct {
	TaskID       string                 `json:"task_id"`
	Status       TaskStatus             `json:"status"`
	Result       string                 `json:"result"`
	Artifacts    []Artifact             `json:"artifacts"`
	Error        string                 `json:"error,omitempty"`
	Iterations   int                    `json:"iterations"`
	TokenUsage   TokenUsage             `json:"token_usage"`
	Duration     time.Duration          `json:"duration"`
	Metadata     map[string]interface{} `json:"metadata"`
	WorkflowID   string                 `json:"workflow_id"`
	WorkflowRunID string                `json:"workflow_run_id"`
}

// ExecutionState represents the current state of agent execution
type ExecutionState struct {
	TaskID             string                 `json:"task_id"`
	SessionID          string                 `json:"session_id"`
	CurrentIteration   int                    `json:"current_iteration"`
	MaxIterations      int                    `json:"max_iterations"`
	Messages           []Message              `json:"messages"`
	ToolCalls          []ToolCall             `json:"tool_calls"`
	PendingApprovals   []ToolApproval         `json:"pending_approvals"`
	Status             TaskStatus             `json:"status"`
	TokenUsage         TokenUsage             `json:"token_usage"`
	Metadata           map[string]interface{} `json:"metadata"`
	LastLLMResponse    *LLMResponse           `json:"last_llm_response,omitempty"`
	ContinueExecution  bool                   `json:"continue_execution"`
	ExecutionStartTime time.Time              `json:"execution_start_time"`
}

// Message represents a conversation message
type Message struct {
	Role      string                 `json:"role"` // system, user, assistant, tool
	Content   string                 `json:"content"`
	ToolCalls []ToolCall             `json:"tool_calls,omitempty"`
	ToolCallID string                `json:"tool_call_id,omitempty"` // For tool responses
	Timestamp time.Time              `json:"timestamp"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// Tool represents a tool available to the agent
type Tool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"` // JSON schema
	Function    string                 `json:"function"`   // Reference to activity or MCP server
	Type        string                 `json:"type"`       // "activity", "mcp", "http"
	Config      map[string]interface{} `json:"config,omitempty"`
}

// ToolCall represents a request to call a tool
type ToolCall struct {
	ID        string                 `json:"id"`
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments"`
	Status    string                 `json:"status"` // pending, approved, denied, executing, completed, failed
	Result    string                 `json:"result,omitempty"`
	Error     string                 `json:"error,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
}

// ToolApproval represents a pending tool approval request (HITL)
type ToolApproval struct {
	ToolCallID  string                 `json:"tool_call_id"`
	ToolName    string                 `json:"tool_name"`
	Arguments   map[string]interface{} `json:"arguments"`
	RequestedAt time.Time              `json:"requested_at"`
	Approved    *bool                  `json:"approved,omitempty"`
	ApprovedAt  *time.Time             `json:"approved_at,omitempty"`
	ApprovedBy  string                 `json:"approved_by,omitempty"`
	Reason      string                 `json:"reason,omitempty"`
}

// ModelConfig represents LLM model configuration
type ModelConfig struct {
	Provider    string                 `json:"provider"`     // openai, anthropic, vertexai
	Model       string                 `json:"model"`        // gpt-4, claude-3-5-sonnet-20241022, etc.
	Temperature float64                `json:"temperature"`
	MaxTokens   int                    `json:"max_tokens"`
	TopP        float64                `json:"top_p,omitempty"`
	APIKey      string                 `json:"api_key,omitempty"`
	Endpoint    string                 `json:"endpoint,omitempty"`
	Extra       map[string]interface{} `json:"extra,omitempty"`
}

// LLMRequest represents a request to an LLM
type LLMRequest struct {
	Messages    []Message   `json:"messages"`
	Tools       []Tool      `json:"tools,omitempty"`
	ModelConfig ModelConfig `json:"model_config"`
	Stream      bool        `json:"stream"`
}

// LLMResponse represents a response from an LLM
type LLMResponse struct {
	Content      string       `json:"content"`
	ToolCalls    []ToolCall   `json:"tool_calls,omitempty"`
	FinishReason string       `json:"finish_reason"` // stop, tool_calls, length, etc.
	TokenUsage   TokenUsage   `json:"token_usage"`
	ModelUsed    string       `json:"model_used"`
	Metadata     interface{}  `json:"metadata,omitempty"`
}

// TokenUsage tracks token consumption
type TokenUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// Artifact represents an execution artifact
type Artifact struct {
	ID          string                 `json:"id"`
	Type        string                 `json:"type"` // text, file, image, etc.
	Content     string                 `json:"content"`
	ContentType string                 `json:"content_type"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt   time.Time              `json:"created_at"`
}

// TaskStatus represents the status of a task
type TaskStatus string

const (
	TaskStatusSubmitted    TaskStatus = "submitted"
	TaskStatusWorking      TaskStatus = "working"
	TaskStatusInputRequired TaskStatus = "input_required"
	TaskStatusAuthRequired TaskStatus = "auth_required"
	TaskStatusCompleted    TaskStatus = "completed"
	TaskStatusFailed       TaskStatus = "failed"
	TaskStatusCancelled    TaskStatus = "cancelled"
)

// A2AEvent represents an event to be published via A2A protocol
type A2AEvent struct {
	TaskID    string                 `json:"task_id"`
	EventType string                 `json:"event_type"` // status_update, artifact_update, message
	Timestamp time.Time              `json:"timestamp"`
	Data      map[string]interface{} `json:"data"`
}

// ConvertToA2ATaskStatus converts our TaskStatus to A2A protocol status
func (s TaskStatus) ToA2A() protocol.TaskStatus {
	switch s {
	case TaskStatusSubmitted:
		return protocol.TaskStatus_SUBMITTED
	case TaskStatusWorking:
		return protocol.TaskStatus_WORKING
	case TaskStatusInputRequired:
		return protocol.TaskStatus_INPUT_REQUIRED
	case TaskStatusAuthRequired:
		return protocol.TaskStatus_AUTH_REQUIRED
	case TaskStatusCompleted:
		return protocol.TaskStatus_COMPLETED
	case TaskStatusFailed:
		return protocol.TaskStatus_FAILED
	case TaskStatusCancelled:
		return protocol.TaskStatus_CANCELLED
	default:
		return protocol.TaskStatus_SUBMITTED
	}
}

// Add method to accumulate token usage
func (t *TokenUsage) Add(other TokenUsage) {
	t.PromptTokens += other.PromptTokens
	t.CompletionTokens += other.CompletionTokens
	t.TotalTokens += other.TotalTokens
}
