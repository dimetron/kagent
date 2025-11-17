package activities

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/kagent-dev/kagent/go/internal/executor/temporal/models"
)

func TestDefaultToolExecutor_ExecuteBuiltinTool(t *testing.T) {
	executor := NewDefaultToolExecutor()
	ctx := context.Background()

	tests := []struct {
		name        string
		toolCall    models.ToolCall
		toolDef     models.Tool
		expectError bool
	}{
		{
			name: "get_current_time",
			toolCall: models.ToolCall{
				ID:        "call_1",
				Name:      "get_current_time",
				Arguments: map[string]interface{}{},
			},
			toolDef: models.Tool{
				Name: "get_current_time",
				Type: "builtin",
			},
			expectError: false,
		},
		{
			name: "get_random_number",
			toolCall: models.ToolCall{
				ID:        "call_2",
				Name:      "get_random_number",
				Arguments: map[string]interface{}{},
			},
			toolDef: models.Tool{
				Name: "get_random_number",
				Type: "builtin",
			},
			expectError: false,
		},
		{
			name: "unknown_builtin_tool",
			toolCall: models.ToolCall{
				ID:        "call_3",
				Name:      "unknown_tool",
				Arguments: map[string]interface{}{},
			},
			toolDef: models.Tool{
				Name: "unknown_tool",
				Type: "builtin",
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := executor.ExecuteTool(ctx, tt.toolCall, tt.toolDef)

			if tt.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.NotEmpty(t, result)
			}
		})
	}
}

func TestDefaultToolExecutor_UnsupportedType(t *testing.T) {
	executor := NewDefaultToolExecutor()
	ctx := context.Background()

	toolCall := models.ToolCall{
		ID:   "call_1",
		Name: "some_tool",
	}
	toolDef := models.Tool{
		Name: "some_tool",
		Type: "unsupported_type",
	}

	_, err := executor.ExecuteTool(ctx, toolCall, toolDef)
	require.Error(t, err)
	require.Contains(t, err.Error(), "unsupported tool type")
}

func TestDefaultToolExecutor_HTTPToolMissingEndpoint(t *testing.T) {
	executor := NewDefaultToolExecutor()
	ctx := context.Background()

	toolCall := models.ToolCall{
		ID:   "call_1",
		Name: "http_tool",
	}
	toolDef := models.Tool{
		Name:   "http_tool",
		Type:   "http",
		Config: map[string]interface{}{},
	}

	_, err := executor.ExecuteTool(ctx, toolCall, toolDef)
	require.Error(t, err)
	require.Contains(t, err.Error(), "missing endpoint")
}

func TestDefaultToolExecutor_MCPToolMissingServer(t *testing.T) {
	executor := NewDefaultToolExecutor()
	ctx := context.Background()

	toolCall := models.ToolCall{
		ID:   "call_1",
		Name: "mcp_tool",
	}
	toolDef := models.Tool{
		Name:   "mcp_tool",
		Type:   "mcp",
		Config: map[string]interface{}{},
	}

	_, err := executor.ExecuteTool(ctx, toolCall, toolDef)
	require.Error(t, err)
	require.Contains(t, err.Error(), "missing server")
}
