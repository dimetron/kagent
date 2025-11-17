package session

import "time"

// Session represents an agent session
type Session struct {
	ID          string                 `json:"id"`
	AppName     string                 `json:"app_name"`
	UserID      string                 `json:"user_id"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
	State       map[string]interface{} `json:"state,omitempty"`
	WorkingDir  string                 `json:"working_dir,omitempty"`
	Events      []Event                `json:"events,omitempty"`
}

// Event represents a session event
type Event struct {
	ID        string                 `json:"id"`
	Type      string                 `json:"type"`
	Timestamp time.Time              `json:"timestamp"`
	Data      map[string]interface{} `json:"data"`
}

// CreateSessionRequest represents a request to create a new session
type CreateSessionRequest struct {
	AppName    string                 `json:"app_name"`
	UserID     string                 `json:"user_id"`
	WorkingDir string                 `json:"working_dir,omitempty"`
	State      map[string]interface{} `json:"state,omitempty"`
}

// InvocationContext contains context information for tool invocations
type InvocationContext struct {
	SessionID string
	UserID    string
	TaskID    string
	ContextID string
}
