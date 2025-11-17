package adk

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"
)

// StaticConfig holds configuration for the static command
type StaticConfig struct {
	ConfigPath string
	Host       string
	Port       int
}

// NewStaticCmd creates the static command
func NewStaticCmd() *cobra.Command {
	cfg := &StaticConfig{}

	cmd := &cobra.Command{
		Use:   "static",
		Short: "Run agent with static configuration files",
		Long: `Run an agent using static configuration files (config.json and agent-card.json).

The static mode expects the following files in the config directory:
  - config.json: Agent configuration including model and tools
  - agent-card.json: Agent card with metadata

Examples:
  kagent adk static --config /etc/kagent
  kagent adk static --config ./config --host 0.0.0.0 --port 8080`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runStatic(cmd.Context(), cfg)
		},
	}

	cmd.Flags().StringVar(&cfg.ConfigPath, "config", "/config", "Path to configuration directory")
	cmd.Flags().StringVar(&cfg.Host, "host", "0.0.0.0", "Host to bind to")
	cmd.Flags().IntVar(&cfg.Port, "port", 8080, "Port to bind to")

	return cmd
}

func runStatic(ctx context.Context, cfg *StaticConfig) error {
	configFile := filepath.Join(cfg.ConfigPath, "config.json")

	// Load agent configuration
	agentCfg, err := loadAgentConfig(configFile)
	if err != nil {
		return fmt.Errorf("failed to load agent configuration: %w", err)
	}

	// Run agent with loaded config
	runCfg := &RunConfig{
		AgentName:  "static-agent",
		Local:      false,
		WorkingDir: cfg.ConfigPath,
		ConfigFile: configFile,
	}

	fmt.Printf("Starting agent in static mode from %s\n", cfg.ConfigPath)
	fmt.Printf("Model: %s\n", agentCfg.Model.Type())
	if agentCfg.Description != "" {
		fmt.Printf("Description: %s\n", agentCfg.Description)
	}

	return runAgent(ctx, runCfg)
}
