package tools

import (
	"context"

	"github.com/kagent-dev/kagent/go/pkg/app/session"
)

// Tool defines the interface for agent tools
type Tool interface {
	Name() string
	Description() string
	RunAsync(ctx context.Context, args map[string]interface{}, toolCtx *Context) (string, error)
}

// Context contains context information for tool execution
type Context struct {
	Session           *session.Session
	InvocationContext *session.InvocationContext
	SessionPath       string
}

// BaseTool provides common functionality for tools
type BaseTool struct {
	name        string
	description string
}

// NewBaseTool creates a new BaseTool
func NewBaseTool(name, description string) BaseTool {
	return BaseTool{
		name:        name,
		description: description,
	}
}

// Name returns the tool name
func (b *BaseTool) Name() string {
	return b.name
}

// Description returns the tool description
func (b *BaseTool) Description() string {
	return b.description
}
