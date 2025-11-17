package adk

import (
	"context"
	"fmt"

	"github.com/kagent-dev/kagent/go/pkg/adk"
	"github.com/spf13/cobra"
)

// TestConfig holds configuration for the test command
type TestConfig struct {
	ConfigFile string
	Task       string
}

// NewTestCmd creates the test command
func NewTestCmd() *cobra.Command {
	cfg := &TestConfig{}

	cmd := &cobra.Command{
		Use:   "test",
		Short: "Test an agent with a task",
		Long: `Test an agent by running it with a specific task.

This command creates a temporary session, runs the agent with the provided task,
and outputs the results. Useful for testing agent configuration and behavior.

Examples:
  kagent adk test --config config.json --task "What is 2+2?"
  kagent adk test --config ./agent/config.json --task "List files in current directory"`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runTest(cmd.Context(), cfg)
		},
	}

	cmd.Flags().StringVar(&cfg.ConfigFile, "config", "config.json", "Path to agent configuration file")
	cmd.Flags().StringVar(&cfg.Task, "task", "", "Task to test the agent with")
	cmd.MarkFlagRequired("task")

	return cmd
}

func runTest(ctx context.Context, cfg *TestConfig) error {
	// Load agent configuration
	agentCfg, err := loadAgentConfig(cfg.ConfigFile)
	if err != nil {
		return fmt.Errorf("failed to load agent configuration: %w", err)
	}

	// Create ADK app
	appCfg := adk.DefaultConfig()
	appCfg.AppName = "test-agent"

	app, err := adk.NewApp(appCfg, agentCfg)
	if err != nil {
		return fmt.Errorf("failed to create app: %w", err)
	}

	fmt.Printf("Testing agent with task: %s\n", cfg.Task)
	fmt.Printf("Model: %s\n", agentCfg.Model.Type())
	fmt.Println("---")

	// Run test
	if err := app.Test(ctx, cfg.Task); err != nil {
		return fmt.Errorf("test failed: %w", err)
	}

	return nil
}
