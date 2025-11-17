package adk

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/kagent-dev/kagent/go/pkg/adk"
	"github.com/kagent-dev/kagent/go/pkg/adk/config"
	"github.com/spf13/cobra"
)

// RunConfig holds configuration for the run command
type RunConfig struct {
	AgentName   string
	Local       bool
	WorkingDir  string
	ConfigFile  string
}

// NewRunCmd creates the run command
func NewRunCmd() *cobra.Command {
	cfg := &RunConfig{}

	cmd := &cobra.Command{
		Use:   "run [agent-name]",
		Short: "Run an ADK agent",
		Long: `Run an ADK agent from the current working directory.

The agent configuration should be in a file named config.json in the working directory,
or specified with the --config flag.

Examples:
  kagent adk run my-agent
  kagent adk run my-agent --local
  kagent adk run my-agent --config /path/to/config.json
  kagent adk run my-agent --working-dir /path/to/agent`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg.AgentName = args[0]
			return runAgent(cmd.Context(), cfg)
		},
	}

	cmd.Flags().BoolVar(&cfg.Local, "local", false, "Run in local mode with in-memory session service")
	cmd.Flags().StringVar(&cfg.WorkingDir, "working-dir", ".", "Working directory for the agent")
	cmd.Flags().StringVar(&cfg.ConfigFile, "config", "", "Path to agent configuration file (default: <working-dir>/config.json)")

	return cmd
}

func runAgent(ctx context.Context, cfg *RunConfig) error {
	// Change to working directory if specified
	if cfg.WorkingDir != "." {
		if err := os.Chdir(cfg.WorkingDir); err != nil {
			return fmt.Errorf("failed to change directory: %w", err)
		}
	}

	// Determine config file path
	configFile := cfg.ConfigFile
	if configFile == "" {
		configFile = "config.json"
	}

	// Load agent configuration
	agentCfg, err := loadAgentConfig(configFile)
	if err != nil {
		return fmt.Errorf("failed to load agent configuration: %w", err)
	}

	// Create ADK app
	appCfg := adk.DefaultConfig()
	appCfg.AppName = cfg.AgentName

	app, err := adk.NewApp(appCfg, agentCfg)
	if err != nil {
		return fmt.Errorf("failed to create app: %w", err)
	}

	// Build server
	var server interface {
		ListenAndServe() error
		Shutdown(context.Context) error
	}

	if cfg.Local {
		fmt.Println("Starting agent in local mode...")
		srv, err := app.BuildLocal(ctx)
		if err != nil {
			return fmt.Errorf("failed to build local server: %w", err)
		}
		server = srv
	} else {
		fmt.Println("Starting agent in production mode...")
		srv, err := app.Build(ctx)
		if err != nil {
			return fmt.Errorf("failed to build server: %w", err)
		}
		server = srv
	}

	// Start server in background
	errChan := make(chan error, 1)
	go func() {
		fmt.Printf("Agent '%s' listening on %s:%d\n", cfg.AgentName, appCfg.Host, appCfg.Port)
		if err := server.ListenAndServe(); err != nil {
			errChan <- err
		}
	}()

	// Wait for interrupt signal or error
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	select {
	case err := <-errChan:
		return fmt.Errorf("server error: %w", err)
	case <-sigChan:
		fmt.Println("\nShutting down agent...")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 30)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("failed to shutdown gracefully: %w", err)
		}
		fmt.Println("Agent stopped")
	case <-ctx.Done():
		fmt.Println("\nContext cancelled, shutting down...")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 30)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("failed to shutdown gracefully: %w", err)
		}
	}

	return nil
}

func loadAgentConfig(path string) (*config.AgentConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg config.AgentConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	// Validate configuration
	if cfg.Model == nil {
		return nil, fmt.Errorf("model configuration is required")
	}

	if err := cfg.Model.Validate(); err != nil {
		return nil, fmt.Errorf("invalid model config: %w", err)
	}

	return &cfg, nil
}
