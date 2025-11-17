package adk

import (
	"github.com/spf13/cobra"
)

// NewADKCmd creates the root adk command
func NewADKCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "adk",
		Short: "ADK (Agent Development Kit) commands",
		Long: `ADK commands for running and managing ADK-based agents.

The ADK package provides a Go implementation equivalent to the kagent-adk package,
offering agent runtime capabilities, session management, tool execution, and A2A protocol support.

Available subcommands:
  run         Run an ADK agent
  static      Run with static configuration files
  test        Test an agent with a task

Examples:
  kagent adk run my-agent
  kagent adk run my-agent --local
  kagent adk static --config /path/to/config
  kagent adk test --config config.json --task "Hello world"`,
	}

	// Add subcommands
	cmd.AddCommand(NewRunCmd())
	cmd.AddCommand(NewStaticCmd())
	cmd.AddCommand(NewTestCmd())

	return cmd
}
