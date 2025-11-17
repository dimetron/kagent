package activities

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/kagent-dev/kagent/go/internal/executor/temporal/models"
)

// DefaultToolExecutor implements ToolExecutor interface
type DefaultToolExecutor struct {
	httpClient *http.Client
}

// NewDefaultToolExecutor creates a new default tool executor
func NewDefaultToolExecutor() ToolExecutor {
	return &DefaultToolExecutor{
		httpClient: &http.Client{
			Timeout: 5 * time.Minute,
		},
	}
}

// ExecuteTool executes a tool based on its type
func (e *DefaultToolExecutor) ExecuteTool(ctx context.Context, toolCall models.ToolCall, toolDef models.Tool) (string, error) {
	switch toolDef.Type {
	case "http":
		return e.executeHTTPTool(ctx, toolCall, toolDef)
	case "mcp":
		return e.executeMCPTool(ctx, toolCall, toolDef)
	case "builtin":
		return e.executeBuiltinTool(ctx, toolCall, toolDef)
	default:
		return "", fmt.Errorf("unsupported tool type: %s", toolDef.Type)
	}
}

// executeHTTPTool executes an HTTP-based tool
func (e *DefaultToolExecutor) executeHTTPTool(ctx context.Context, toolCall models.ToolCall, toolDef models.Tool) (string, error) {
	// Get endpoint from tool config
	endpoint, ok := toolDef.Config["endpoint"].(string)
	if !ok {
		return "", fmt.Errorf("HTTP tool missing endpoint configuration")
	}

	// Prepare request body
	requestBody := map[string]interface{}{
		"name":      toolCall.Name,
		"arguments": toolCall.Arguments,
	}

	bodyJSON, err := json.Marshal(requestBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	// Add authentication if configured
	if apiKey, ok := toolDef.Config["api_key"].(string); ok {
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}

	req.Body = io.NopCloser(json.NewDecoder(bodyJSON).Decode)

	// Execute request
	resp, err := e.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("HTTP tool returned error: %s (status %d)", string(respBody), resp.StatusCode)
	}

	return string(respBody), nil
}

// executeMCPTool executes an MCP (Model Context Protocol) tool
func (e *DefaultToolExecutor) executeMCPTool(ctx context.Context, toolCall models.ToolCall, toolDef models.Tool) (string, error) {
	// Get MCP server reference from tool config
	mcpServer, ok := toolDef.Config["mcp_server"].(string)
	if !ok {
		return "", fmt.Errorf("MCP tool missing server configuration")
	}

	// In a real implementation, this would:
	// 1. Connect to the MCP server (using existing kmcp integration)
	// 2. Call the tool via MCP protocol
	// 3. Return the result

	// For now, return a placeholder
	return fmt.Sprintf("MCP tool %s executed on server %s with args: %v",
		toolCall.Name, mcpServer, toolCall.Arguments), nil
}

// executeBuiltinTool executes a built-in tool
func (e *DefaultToolExecutor) executeBuiltinTool(ctx context.Context, toolCall models.ToolCall, toolDef models.Tool) (string, error) {
	// Built-in tools are executed directly
	switch toolCall.Name {
	case "get_current_time":
		return time.Now().Format(time.RFC3339), nil
	case "get_random_number":
		return "42", nil // TODO: Actually generate random number
	default:
		return "", fmt.Errorf("unknown builtin tool: %s", toolCall.Name)
	}
}

// MCPToolExecutor implements MCP-specific tool execution
type MCPToolExecutor struct {
	// TODO: Add MCP client integration
}

// NewMCPToolExecutor creates a new MCP tool executor
func NewMCPToolExecutor() ToolExecutor {
	return &MCPToolExecutor{}
}

// ExecuteTool executes an MCP tool
func (e *MCPToolExecutor) ExecuteTool(ctx context.Context, toolCall models.ToolCall, toolDef models.Tool) (string, error) {
	// TODO: Implement MCP tool execution using existing kmcp package
	// This would integrate with github.com/kagent-dev/kmcp
	return "", fmt.Errorf("MCP tool execution not yet implemented")
}
