package tools

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	apperrors "github.com/kagent-dev/kagent/go/pkg/app/errors"
)

const (
	DefaultBashTimeout      = 30 * time.Second
	PipInstallTimeout       = 120 * time.Second
	ScriptExecutionTimeout  = 60 * time.Second
)

// BashTool implements bash command execution in a sandboxed environment
type BashTool struct {
	BaseTool
	skillsDirectory string
	bashVenvPath    string
}

// NewBashTool creates a new BashTool
func NewBashTool(skillsDirectory, bashVenvPath string) *BashTool {
	if bashVenvPath == "" {
		bashVenvPath = "/.kagent/sandbox-venv"
	}
	return &BashTool{
		BaseTool:        NewBaseTool("bash", "Execute bash commands in a sandboxed environment"),
		skillsDirectory: skillsDirectory,
		bashVenvPath:    bashVenvPath,
	}
}

func (b *BashTool) RunAsync(ctx context.Context, args map[string]interface{}, toolCtx *Context) (string, error) {
	command, ok := args["command"].(string)
	if !ok {
		return "", apperrors.New(apperrors.ErrCodeInvalidInput, "command is required", nil)
	}

	// Determine timeout based on command type
	timeout := b.getTimeout(command)
	cmdCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Get working directory from session
	workingDir := toolCtx.SessionPath
	if toolCtx.Session != nil && toolCtx.Session.WorkingDir != "" {
		workingDir = toolCtx.Session.WorkingDir
	}

	// Prepare environment variables
	env := []string{
		fmt.Sprintf("PATH=%s/bin:/usr/local/bin:/usr/bin:/bin", b.bashVenvPath),
		fmt.Sprintf("VIRTUAL_ENV=%s", b.bashVenvPath),
	}
	if b.skillsDirectory != "" {
		env = append(env, fmt.Sprintf("PYTHONPATH=%s", b.skillsDirectory))
	}

	// Execute command using sandbox runtime
	// Note: In production, this should use 'srt' (Sandbox Runtime)
	// For now, we'll use a simplified approach
	cmd := exec.CommandContext(cmdCtx, "bash", "-c", command)
	cmd.Dir = workingDir
	cmd.Env = env

	output, err := cmd.CombinedOutput()
	if err != nil {
		// Check if it's a timeout
		if cmdCtx.Err() == context.DeadlineExceeded {
			return "", apperrors.New(apperrors.ErrCodeToolExecution,
				fmt.Sprintf("command timed out after %v", timeout), err)
		}
		return string(output), apperrors.New(apperrors.ErrCodeToolExecution,
			"command execution failed", err)
	}

	return string(output), nil
}

func (b *BashTool) getTimeout(command string) time.Duration {
	// Determine timeout based on command
	if strings.Contains(command, "pip install") || strings.Contains(command, "pip3 install") {
		return PipInstallTimeout
	}
	if strings.HasPrefix(command, "python") || strings.HasPrefix(command, "python3") {
		return ScriptExecutionTimeout
	}
	return DefaultBashTimeout
}
