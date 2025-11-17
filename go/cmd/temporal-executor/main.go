package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	"go.temporal.io/sdk/client"

	executor "github.com/kagent-dev/kagent/go/internal/executor/temporal"
	"github.com/kagent-dev/kagent/go/internal/executor/temporal/activities"
	"github.com/kagent-dev/kagent/go/internal/executor/temporal/config"
)

const (
	defaultConfigPath = "config/temporal-executor.yaml"
)

func main() {
	// Parse command line arguments
	configPath := defaultConfigPath
	if len(os.Args) > 1 {
		configPath = os.Args[1]
	}

	// Load configuration
	cfg, err := loadConfiguration(configPath)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		log.Fatalf("Invalid configuration: %v", err)
	}

	log.Printf("Starting Temporal Executor Service")
	log.Printf("Temporal Server: %s", cfg.Temporal.HostPort)
	log.Printf("Task Queue: %s", cfg.Temporal.TaskQueue)
	log.Printf("HTTP Server: %s:%d", cfg.Server.Host, cfg.Server.Port)

	// Create context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Setup signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Create Temporal client
	temporalClient, err := createTemporalClient(cfg)
	if err != nil {
		log.Fatalf("Failed to create Temporal client: %v", err)
	}
	defer temporalClient.Close()

	// Create worker
	worker, err := createWorker(temporalClient, cfg)
	if err != nil {
		log.Fatalf("Failed to create worker: %v", err)
	}

	// Start worker in background
	workerCtx, workerCancel := context.WithCancel(ctx)
	defer workerCancel()

	go func() {
		if err := worker.Run(workerCtx); err != nil && err != context.Canceled {
			log.Printf("Worker error: %v", err)
		}
	}()

	// Create executor service
	eventPublisher := activities.NewA2AEventPublisher()
	executorService := executor.NewExecutorService(temporalClient, eventPublisher, cfg.Temporal.TaskQueue)

	// Create HTTP server
	httpServer := createHTTPServer(cfg, executorService, eventPublisher)

	// Start HTTP server in background
	go func() {
		addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
		log.Printf("HTTP server listening on %s", addr)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("HTTP server error: %v", err)
		}
	}()

	// Wait for shutdown signal
	<-sigChan
	log.Printf("Shutdown signal received, gracefully stopping...")

	// Shutdown HTTP server
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		log.Printf("HTTP server shutdown error: %v", err)
	}

	// Stop worker
	workerCancel()

	log.Printf("Shutdown complete")
}

func loadConfiguration(configPath string) (*config.Config, error) {
	// Check if config file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		log.Printf("Config file not found at %s, using defaults", configPath)
		cfg := config.DefaultConfig()

		// Try to save default config
		if err := config.SaveConfig(cfg, configPath); err != nil {
			log.Printf("Warning: Could not save default config: %v", err)
		} else {
			log.Printf("Default configuration saved to %s", configPath)
		}

		return cfg, nil
	}

	// Load config from file
	return config.LoadConfig(configPath)
}

func createTemporalClient(cfg *config.Config) (client.Client, error) {
	options := client.Options{
		HostPort:  cfg.Temporal.HostPort,
		Namespace: cfg.Temporal.Namespace,
	}

	return client.Dial(options)
}

func createWorker(temporalClient client.Client, cfg *config.Config) (*executor.Worker, error) {
	workerConfig := executor.WorkerConfig{
		TaskQueue:     cfg.Temporal.TaskQueue,
		MaxConcurrent: cfg.Executor.MaxConcurrentActivities,
	}

	// Convert LLM provider configs
	for _, providerCfg := range cfg.LLM.Providers {
		workerConfig.LLMProviderConfigs = append(workerConfig.LLMProviderConfigs, executor.LLMProviderConfig{
			Name:   providerCfg.Name,
			APIKey: providerCfg.APIKey,
			Config: providerCfg.Config,
		})
	}

	return executor.NewWorker(temporalClient, workerConfig)
}

func createHTTPServer(
	cfg *config.Config,
	executorService *executor.ExecutorService,
	eventPublisher activities.EventPublisher,
) *http.Server {
	router := mux.NewRouter()

	// Create A2A executor and handler
	a2aExecutor := executor.NewA2AExecutor(executorService, eventPublisher)
	a2aHandler := executor.NewA2AHTTPHandler(a2aExecutor)

	// Register routes
	router.HandleFunc("/health", healthHandler).Methods("GET")
	router.HandleFunc("/api/v1/execute", handleExecute(executorService)).Methods("POST")
	router.HandleFunc("/api/v1/status/{workflowId}", handleGetStatus(executorService)).Methods("GET")
	router.HandleFunc("/api/v1/cancel/{workflowId}", handleCancel(executorService)).Methods("POST")
	router.HandleFunc("/api/v1/approve/{workflowId}", handleApprove(executorService)).Methods("POST")

	// A2A protocol endpoint
	if cfg.A2A.Enabled {
		router.HandleFunc("/a2a/message", a2aHandler.HandleMessage).Methods("POST")
	}

	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	return &http.Server{
		Addr:         addr,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"healthy"}`))
}

func handleExecute(executorService *executor.ExecutorService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// TODO: Parse request and execute
		http.Error(w, "Not implemented", http.StatusNotImplemented)
	}
}

func handleGetStatus(executorService *executor.ExecutorService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// TODO: Get workflow status
		http.Error(w, "Not implemented", http.StatusNotImplemented)
	}
}

func handleCancel(executorService *executor.ExecutorService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// TODO: Cancel workflow
		http.Error(w, "Not implemented", http.StatusNotImplemented)
	}
}

func handleApprove(executorService *executor.ExecutorService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// TODO: Approve tool execution
		http.Error(w, "Not implemented", http.StatusNotImplemented)
	}
}
