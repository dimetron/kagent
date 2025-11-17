package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// Config represents the Temporal executor configuration
type Config struct {
	Temporal  TemporalConfig    `yaml:"temporal"`
	Executor  ExecutorConfig    `yaml:"executor"`
	LLM       LLMConfig         `yaml:"llm"`
	Server    ServerConfig      `yaml:"server"`
	A2A       A2AConfig         `yaml:"a2a"`
}

// TemporalConfig holds Temporal client configuration
type TemporalConfig struct {
	HostPort  string `yaml:"host_port"`
	Namespace string `yaml:"namespace"`
	TaskQueue string `yaml:"task_queue"`
}

// ExecutorConfig holds executor-specific configuration
type ExecutorConfig struct {
	MaxConcurrentWorkflows int           `yaml:"max_concurrent_workflows"`
	MaxConcurrentActivities int          `yaml:"max_concurrent_activities"`
	DefaultTimeout         time.Duration `yaml:"default_timeout"`
	MaxIterations          int           `yaml:"max_iterations"`
	RequireApproval        bool          `yaml:"require_approval"`
}

// LLMConfig holds LLM provider configurations
type LLMConfig struct {
	Providers []LLMProviderConfig `yaml:"providers"`
}

// LLMProviderConfig holds individual provider configuration
type LLMProviderConfig struct {
	Name     string                 `yaml:"name"`
	APIKey   string                 `yaml:"api_key,omitempty"`
	APIKeyEnv string                `yaml:"api_key_env,omitempty"`
	Endpoint string                 `yaml:"endpoint,omitempty"`
	Config   map[string]interface{} `yaml:"config,omitempty"`
}

// ServerConfig holds HTTP server configuration
type ServerConfig struct {
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
}

// A2AConfig holds A2A protocol configuration
type A2AConfig struct {
	Enabled       bool   `yaml:"enabled"`
	WebhookURL    string `yaml:"webhook_url,omitempty"`
	AuthToken     string `yaml:"auth_token,omitempty"`
	AuthTokenEnv  string `yaml:"auth_token_env,omitempty"`
}

// LoadConfig loads configuration from a YAML file
func LoadConfig(filePath string) (*Config, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	// Resolve API keys from environment variables
	for i := range config.LLM.Providers {
		if config.LLM.Providers[i].APIKeyEnv != "" {
			config.LLM.Providers[i].APIKey = os.Getenv(config.LLM.Providers[i].APIKeyEnv)
		}
	}

	if config.A2A.AuthTokenEnv != "" {
		config.A2A.AuthToken = os.Getenv(config.A2A.AuthTokenEnv)
	}

	// Set defaults
	config.SetDefaults()

	return &config, nil
}

// SetDefaults sets default values for configuration
func (c *Config) SetDefaults() {
	if c.Temporal.HostPort == "" {
		c.Temporal.HostPort = "localhost:7233"
	}
	if c.Temporal.Namespace == "" {
		c.Temporal.Namespace = "default"
	}
	if c.Temporal.TaskQueue == "" {
		c.Temporal.TaskQueue = "agent-execution-queue"
	}

	if c.Executor.MaxConcurrentWorkflows == 0 {
		c.Executor.MaxConcurrentWorkflows = 100
	}
	if c.Executor.MaxConcurrentActivities == 0 {
		c.Executor.MaxConcurrentActivities = 100
	}
	if c.Executor.DefaultTimeout == 0 {
		c.Executor.DefaultTimeout = 5 * time.Minute
	}
	if c.Executor.MaxIterations == 0 {
		c.Executor.MaxIterations = 10
	}

	if c.Server.Host == "" {
		c.Server.Host = "0.0.0.0"
	}
	if c.Server.Port == 0 {
		c.Server.Port = 8080
	}
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if c.Temporal.HostPort == "" {
		return fmt.Errorf("temporal.host_port is required")
	}

	if len(c.LLM.Providers) == 0 {
		return fmt.Errorf("at least one LLM provider must be configured")
	}

	for _, provider := range c.LLM.Providers {
		if provider.Name == "" {
			return fmt.Errorf("LLM provider name is required")
		}
		if provider.APIKey == "" && provider.APIKeyEnv == "" {
			return fmt.Errorf("LLM provider %s requires api_key or api_key_env", provider.Name)
		}
	}

	return nil
}

// DefaultConfig returns a default configuration
func DefaultConfig() *Config {
	config := &Config{
		Temporal: TemporalConfig{
			HostPort:  "localhost:7233",
			Namespace: "default",
			TaskQueue: "agent-execution-queue",
		},
		Executor: ExecutorConfig{
			MaxConcurrentWorkflows:  100,
			MaxConcurrentActivities: 100,
			DefaultTimeout:          5 * time.Minute,
			MaxIterations:           10,
			RequireApproval:         false,
		},
		LLM: LLMConfig{
			Providers: []LLMProviderConfig{
				{
					Name:      "anthropic",
					APIKeyEnv: "ANTHROPIC_API_KEY",
				},
				{
					Name:      "openai",
					APIKeyEnv: "OPENAI_API_KEY",
				},
			},
		},
		Server: ServerConfig{
			Host: "0.0.0.0",
			Port: 8080,
		},
		A2A: A2AConfig{
			Enabled: true,
		},
	}

	return config
}

// SaveConfig saves configuration to a YAML file
func SaveConfig(config *Config, filePath string) error {
	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}
