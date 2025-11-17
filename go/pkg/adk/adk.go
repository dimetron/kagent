// Package adk provides a Go implementation of the KAgent ADK (Agent Development Kit).
// This package mirrors the functionality of the kagent-adk package, providing
// agent runtime capabilities, session management, tool execution, and A2A protocol support.
package adk

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/mux"
	"github.com/kagent-dev/kagent/go/pkg/adk/auth"
	"github.com/kagent-dev/kagent/go/pkg/adk/config"
	apperrors "github.com/kagent-dev/kagent/go/pkg/adk/errors"
	"github.com/kagent-dev/kagent/go/pkg/adk/session"
	"github.com/kagent-dev/kagent/go/pkg/adk/tools"
)

// App represents the main KAgent ADK application
type App struct {
	Config          *Config
	AgentConfig     *config.AgentConfig
	SessionService  session.Service
	PathManager     *session.PathManager
	TokenService    *auth.TokenService
	Tools           []tools.Tool
	router          *mux.Router
}

// Config represents the ADK application configuration
type Config struct {
	KAgentURL    string
	AppName      string
	SkillsDir    string
	TokenPath    string
	BashVenvPath string
	Host         string
	Port         int
	LogLevel     string
}

// NewApp creates a new KAgent ADK application
func NewApp(cfg *Config, agentCfg *config.AgentConfig) (*App, error) {
	if cfg == nil {
		cfg = DefaultConfig()
	}

	app := &App{
		Config:      cfg,
		AgentConfig: agentCfg,
		PathManager: session.NewPathManager("/tmp/kagent"),
	}

	// Initialize token service
	app.TokenService = auth.NewTokenService(cfg.AppName, cfg.TokenPath)

	// Initialize session service
	app.SessionService = session.NewKAgentService(cfg.KAgentURL, app.TokenService.GetToken)

	// Initialize tools
	app.initializeTools()

	return app, nil
}

func (a *App) initializeTools() {
	a.Tools = []tools.Tool{
		tools.NewReadFileTool(),
		tools.NewWriteFileTool(),
		tools.NewEditFileTool(),
		tools.NewBashTool(a.Config.SkillsDir, a.Config.BashVenvPath),
		tools.NewSkillsTool(a.Config.SkillsDir),
	}
}

// Build creates an HTTP server for production mode with KAgent session service
func (a *App) Build(ctx context.Context) (*http.Server, error) {
	// Start token service
	if err := a.TokenService.Start(ctx); err != nil {
		return nil, apperrors.New(apperrors.ErrCodeAuthFailed, "failed to start token service", err)
	}

	// Create router
	a.router = mux.NewRouter()
	a.setupRoutes()

	server := &http.Server{
		Addr:         fmt.Sprintf("%s:%d", a.Config.Host, a.Config.Port),
		Handler:      a.router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
	}

	return server, nil
}

// BuildLocal creates an HTTP server for local mode with in-memory session service
func (a *App) BuildLocal(ctx context.Context) (*http.Server, error) {
	// Use in-memory session service for local development
	// TODO: Implement InMemorySessionService

	// Create router
	a.router = mux.NewRouter()
	a.setupRoutes()

	server := &http.Server{
		Addr:         fmt.Sprintf("%s:%d", a.Config.Host, a.Config.Port),
		Handler:      a.router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
	}

	return server, nil
}

// Test runs the agent in test mode with a single task
func (a *App) Test(ctx context.Context, task string) error {
	// TODO: Implement test mode
	// This would create a temporary session, run the task, and output results
	return fmt.Errorf("test mode not yet implemented")
}

func (a *App) setupRoutes() {
	// Health check endpoint
	a.router.HandleFunc("/health", a.handleHealth).Methods("GET")

	// Thread dump endpoint (for debugging)
	a.router.HandleFunc("/threaddump", a.handleThreadDump).Methods("GET")

	// Agent info endpoint
	a.router.HandleFunc("/info", a.handleInfo).Methods("GET")

	// TODO: Add A2A protocol endpoints
	// a.router.HandleFunc("/a2a/execute", a.handleA2AExecute).Methods("POST")
}

func (a *App) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "healthy",
		"app":    a.Config.AppName,
	})
}

func (a *App) handleThreadDump(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement goroutine dump
	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte("Thread dump not yet implemented\n"))
}

func (a *App) handleInfo(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	info := map[string]interface{}{
		"app_name":    a.Config.AppName,
		"description": a.AgentConfig.Description,
		"tools":       len(a.Tools),
		"model_type":  a.AgentConfig.Model.Type(),
	}
	json.NewEncoder(w).Encode(info)
}

// DefaultConfig returns the default configuration from environment variables
func DefaultConfig() *Config {
	port := 8080
	if p := os.Getenv("PORT"); p != "" {
		fmt.Sscanf(p, "%d", &port)
	}

	return &Config{
		KAgentURL:    getEnvOrDefault("KAGENT_URL", "http://localhost:8083"),
		AppName:      getEnvOrDefault("APP_NAME", "kagent-app"),
		SkillsDir:    os.Getenv("KAGENT_SKILLS_FOLDER"),
		TokenPath:    getEnvOrDefault("KAGENT_TOKEN_PATH", auth.DefaultTokenPath),
		BashVenvPath: getEnvOrDefault("BASH_VENV_PATH", "/.kagent/sandbox-venv"),
		Host:         getEnvOrDefault("HOST", "0.0.0.0"),
		Port:         port,
		LogLevel:     getEnvOrDefault("LOG_LEVEL", "INFO"),
	}
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
