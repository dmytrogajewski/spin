package manager

import (
	"errors"
	"fmt"
	"time"
)

// Config contains unified configuration for the Spin agent.
// This replaces both core.Config and core.AgentConfig for consistency.
type Config struct {
	// LLM Provider Configuration
	Provider       string                 `yaml:"provider" mapstructure:"provider"`
	Model          string                 `yaml:"model" mapstructure:"model"`
	ProviderConfig map[string]interface{} `yaml:"provider_config" mapstructure:"provider_config"`
	Temperature    float64                `yaml:"temperature" mapstructure:"temperature"`

	// Agent Configuration
	MaxTurns        int           `yaml:"max_turns" mapstructure:"max_turns"`
	Timeout         time.Duration `yaml:"timeout" mapstructure:"timeout"`
	WorkDir         string        `yaml:"work_dir" mapstructure:"work_dir"`
	MaxTokens       int           `yaml:"max_tokens" mapstructure:"max_tokens"`
	RequireApproval bool          `yaml:"require_approval" mapstructure:"require_approval"`

	// Security Configuration
	SandboxMode     string   `yaml:"sandbox_mode" mapstructure:"sandbox_mode"`
	PolicyFile      string   `yaml:"policy_file" mapstructure:"policy_file"`
	AllowedCommands []string `yaml:"allowed_commands" mapstructure:"allowed_commands"`

	// Feature Flags
	EnableMCP    bool              `yaml:"enable_mcp" mapstructure:"enable_mcp"`
	MCPServers   []MCPServerConfig `yaml:"mcp_servers" mapstructure:"mcp_servers"`
	EnableGit    bool              `yaml:"enable_git" mapstructure:"enable_git"`
	EnableShell  bool              `yaml:"enable_shell" mapstructure:"enable_shell"`
	ShellTimeout time.Duration     `yaml:"shell_timeout" mapstructure:"shell_timeout"`

	// Performance Configuration
	StreamBuffer  int  `yaml:"stream_buffer" mapstructure:"stream_buffer"`
	CacheCommands bool `yaml:"cache_commands" mapstructure:"cache_commands"`

	// Environment Configuration
	MaxFiles int  `yaml:"max_files" mapstructure:"max_files"`
	MaxDepth int  `yaml:"max_depth" mapstructure:"max_depth"`
	SkipGit  bool `yaml:"skip_git" mapstructure:"skip_git"`

	// Storage Configuration
	SessionDir   string `yaml:"session_dir" mapstructure:"session_dir"`
	HistoryLimit int    `yaml:"history_limit" mapstructure:"history_limit"`

	// Logging Configuration
	LogLevel  string `yaml:"log_level" mapstructure:"log_level"`   // debug, info, warn, error
	LogFormat string `yaml:"log_format" mapstructure:"log_format"` // text, json
	Debug     bool   `yaml:"debug" mapstructure:"debug"`           // Enable debug mode

	// Cycle Detection Configuration
	CycleDetection CycleDetectionConfig `yaml:"cycle_detection" mapstructure:"cycle_detection"`
}

// CycleDetectionConfig configures automatic cycle detection and intervention.
type CycleDetectionConfig struct {
	// Enabled controls whether cycle detection is active (default: true)
	Enabled bool `yaml:"enabled" mapstructure:"enabled"`

	// WindowSize is the number of snapshots to compare for pattern detection (default: 3)
	WindowSize int `yaml:"window_size" mapstructure:"window_size"`

	// SimilarityThresh is the threshold for response similarity detection (default: 0.8)
	SimilarityThresh float64 `yaml:"similarity_thresh" mapstructure:"similarity_thresh"`

	// ToolRepeatLimit is the max identical tool calls before triggering cycle (default: 3)
	ToolRepeatLimit int `yaml:"tool_repeat_limit" mapstructure:"tool_repeat_limit"`

	// ErrorRepeatLimit is the max identical errors before triggering cycle (default: 3)
	ErrorRepeatLimit int `yaml:"error_repeat_limit" mapstructure:"error_repeat_limit"`
}

// MCPServerConfig represents configuration for an MCP server.
type MCPServerConfig struct {
	Name    string            `yaml:"name" mapstructure:"name"`
	Command string            `yaml:"command" mapstructure:"command"`
	Args    []string          `yaml:"args" mapstructure:"args"`
	Env     map[string]string `yaml:"env" mapstructure:"env"`
}

// DefaultConfig returns a new Config with sensible defaults.
func DefaultConfig() *Config {
	return &Config{
		// LLM Provider defaults
		Temperature: 0.7,

		// Agent defaults
		MaxTurns:        50,
		Timeout:         5 * time.Minute,
		MaxTokens:       8192,
		RequireApproval: false,
		// Security defaults
		SandboxMode: "workspace-only",

		// Feature flags
		EnableGit:    true,
		EnableShell:  true,
		EnableMCP:    false,
		ShellTimeout: 30 * time.Second,

		// Performance defaults
		StreamBuffer:  100,
		CacheCommands: false,

		// Storage defaults
		SessionDir:   "~/.spin/sessions",
		HistoryLimit: 1000,

		// Logging defaults
		LogLevel:  "info",
		LogFormat: "text",
		Debug:     false,

		// Cycle detection defaults
		CycleDetection: CycleDetectionConfig{
			Enabled:          true,
			WindowSize:       3,
			SimilarityThresh: 0.8,
			ToolRepeatLimit:  3,
			ErrorRepeatLimit: 3,
		},
	}
}

// Validate validates the configuration.
func (c *Config) Validate() error {
	var errs []error

	if c.Provider == "" {
		errs = append(errs, fmt.Errorf("provider is required"))
	}

	if c.Model == "" {
		errs = append(errs, fmt.Errorf("model is required"))
	}

	if c.MaxTurns <= 0 {
		errs = append(errs, fmt.Errorf("max_turns must be > 0, got %d", c.MaxTurns))
	}

	if c.Timeout <= 0 {
		errs = append(errs, fmt.Errorf("timeout must be > 0, got %v", c.Timeout))
	}

	if c.MaxTokens <= 0 {
		errs = append(errs, fmt.Errorf("max_tokens must be > 0, got %d", c.MaxTokens))
	}

	if c.Temperature < 0 || c.Temperature > 2 {
		errs = append(errs, fmt.Errorf("temperature must be between 0 and 2, got %f", c.Temperature))
	}

	if c.ShellTimeout <= 0 {
		errs = append(errs, fmt.Errorf("shell_timeout must be > 0, got %v", c.ShellTimeout))
	}

	// Validate sandbox mode
	validModes := []string{"read-only", "workspace-only", "full-access"}
	validMode := false
	for _, mode := range validModes {
		if c.SandboxMode == mode {
			validMode = true
			break
		}
	}
	if !validMode {
		errs = append(errs, fmt.Errorf("invalid sandbox_mode: %s (must be one of: read-only, workspace-only, full-access)", c.SandboxMode))
	}

	// Validate cycle detection config
	if err := c.CycleDetection.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("cycle_detection: %w", err))
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}

	return nil
}

// Validate validates the cycle detection configuration.
func (c *CycleDetectionConfig) Validate() error {
	var errs []error

	if c.WindowSize <= 0 {
		errs = append(errs, fmt.Errorf("window_size must be > 0, got %d", c.WindowSize))
	}

	if c.SimilarityThresh < 0 || c.SimilarityThresh > 1 {
		errs = append(errs, fmt.Errorf("similarity_thresh must be between 0 and 1, got %f", c.SimilarityThresh))
	}

	if c.ToolRepeatLimit <= 0 {
		errs = append(errs, fmt.Errorf("tool_repeat_limit must be > 0, got %d", c.ToolRepeatLimit))
	}

	if c.ErrorRepeatLimit <= 0 {
		errs = append(errs, fmt.Errorf("error_repeat_limit must be > 0, got %d", c.ErrorRepeatLimit))
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}

	return nil
}
