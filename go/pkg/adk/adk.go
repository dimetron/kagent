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
	"github.com/kagent-dev/kagent/go/pkg/adk/converters"
	apperrors "github.com/kagent-dev/kagent/go/pkg/adk/errors"
	"github.com/kagent-dev/kagent/go/pkg/adk/executor"
	"github.com/kagent-dev/kagent/go/pkg/adk/llm"
	"github.com/kagent-dev/kagent/go/pkg/adk/session"
	"github.com/kagent-dev/kagent/go/pkg/adk/tools"
	"trpc.group/trpc-go/trpc-a2a-go/protocol"
)

// App represents the main KAgent ADK application
type App struct {
	Config          *Config
	AgentConfig     *config.AgentConfig
	SessionService  session.Service
	PathManager     *session.PathManager
	TokenService    *auth.TokenService
	Tools           []tools.Tool
	Executor        *executor.A2AExecutor
	ExecutorV2      *executor.A2AExecutorV2
	LLMClient       llm.Client
	EventConverter  *converters.EventConverter
	router          *mux.Router
	useLLM          bool
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

	// Try to initialize LLM client
	llmClient, err := llm.NewClientFromConfig(agentCfg.Model)
	if err == nil {
		app.LLMClient = llmClient
		app.ExecutorV2 = executor.NewA2AExecutorV2(app.SessionService, app.PathManager, app.Tools, llmClient)
		app.useLLM = true
		fmt.Printf("✓ LLM client initialized: %s\n", llmClient.ModelName())
	} else {
		// Fall back to echo executor
		fmt.Fprintf(os.Stderr, "⚠ LLM client initialization failed: %v\n", err)
		fmt.Fprintf(os.Stderr, "→ Using echo executor instead\n")
		app.useLLM = false
	}

	// Initialize fallback executor
	app.Executor = executor.NewA2AExecutor(app.SessionService, app.PathManager, app.Tools)

	// Initialize event converter
	app.EventConverter = converters.NewEventConverter()

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

	// A2A protocol endpoints
	a.router.HandleFunc("/a2a/message", a.handleA2AMessage).Methods("POST")
	a.router.HandleFunc("/a2a/stream", a.handleA2AStream).Methods("POST")
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
		"llm_enabled": a.useLLM,
	}
	if a.LLMClient != nil {
		info["model_name"] = a.LLMClient.ModelName()
	}
	json.NewEncoder(w).Encode(info)
}

func (a *App) handleA2AMessage(w http.ResponseWriter, r *http.Request) {
	// Decode A2A message request
	var params protocol.SendMessageParams
	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		http.Error(w, fmt.Sprintf("invalid request: %v", err), http.StatusBadRequest)
		return
	}

	// Create request context
	requestCtx := &converters.RequestContext{
		SessionID: params.SessionID,
		UserID:    params.UserID,
		TaskID:    params.TaskID,
		ContextID: params.ContextID,
		Message:   &params.Message,
	}

	// Create event queue
	eventQueue := make(chan *converters.Event, 100)
	done := make(chan error, 1)

	// Execute in background using appropriate executor
	go func() {
		var err error
		if a.useLLM && a.ExecutorV2 != nil {
			err = a.ExecutorV2.Execute(r.Context(), requestCtx, eventQueue)
		} else {
			err = a.Executor.Execute(r.Context(), requestCtx, eventQueue)
		}
		done <- err
		close(eventQueue)
	}()

	// Collect all events
	var allEvents []*converters.Event
	for event := range eventQueue {
		allEvents = append(allEvents, event)
	}

	// Wait for completion
	if err := <-done; err != nil {
		http.Error(w, fmt.Sprintf("execution failed: %v", err), http.StatusInternalServerError)
		return
	}

	// Convert last content event to response
	var responseMessage *protocol.Message
	for i := len(allEvents) - 1; i >= 0; i-- {
		if allEvents[i].Type == converters.EventTypeContent && allEvents[i].Content != nil {
			partConverter := converters.NewPartConverter()
			parts, err := partConverter.ConvertContentToA2A(allEvents[i].Content.Parts)
			if err == nil {
				responseMessage = &protocol.Message{
					MessageID: protocol.GenerateMessageID(),
					Kind:      protocol.KindMessage,
					Parts:     parts,
				}
				break
			}
		}
	}

	if responseMessage == nil {
		responseMessage = &protocol.Message{
			MessageID: protocol.GenerateMessageID(),
			Kind:      protocol.KindMessage,
			Parts: []protocol.Part{
				&protocol.TextPart{Text: "No response generated"},
			},
		}
	}

	result := &protocol.MessageResult{
		Message: *responseMessage,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func (a *App) handleA2AStream(w http.ResponseWriter, r *http.Request) {
	// Decode A2A message request
	var params protocol.SendMessageParams
	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		http.Error(w, fmt.Sprintf("invalid request: %v", err), http.StatusBadRequest)
		return
	}

	// Set up SSE streaming
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	// Create request context
	requestCtx := &converters.RequestContext{
		SessionID: params.SessionID,
		UserID:    params.UserID,
		TaskID:    params.TaskID,
		ContextID: params.ContextID,
		Message:   &params.Message,
	}

	// Create event queue
	eventQueue := make(chan *converters.Event, 100)
	done := make(chan error, 1)

	// Execute in background using appropriate executor
	go func() {
		var err error
		if a.useLLM && a.ExecutorV2 != nil {
			err = a.ExecutorV2.Execute(r.Context(), requestCtx, eventQueue)
		} else {
			err = a.Executor.Execute(r.Context(), requestCtx, eventQueue)
		}
		done <- err
		close(eventQueue)
	}()

	// Create invocation context for event conversion
	invCtx := &converters.InvocationContext{
		SessionID: params.SessionID,
		UserID:    params.UserID,
		TaskID:    params.TaskID,
		ContextID: params.ContextID,
	}

	// Stream events
	for event := range eventQueue {
		// Convert to A2A events
		a2aEvents, err := a.EventConverter.Convert(event, invCtx, params.TaskID, params.ContextID)
		if err != nil {
			fmt.Fprintf(w, "event: error\ndata: %s\n\n", err.Error())
			flusher.Flush()
			continue
		}

		// Send each A2A event
		for _, a2aEvent := range a2aEvents {
			data, err := json.Marshal(a2aEvent)
			if err != nil {
				continue
			}
			fmt.Fprintf(w, "event: message\ndata: %s\n\n", string(data))
			flusher.Flush()
		}
	}

	// Wait for completion
	if err := <-done; err != nil {
		fmt.Fprintf(w, "event: error\ndata: %s\n\n", err.Error())
		flusher.Flush()
	}
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
