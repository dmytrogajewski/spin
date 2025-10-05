package core

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/dmytrogajewski/spin/internal/config"
	"gopkg.in/yaml.v3"
)

// Config contains core configuration for the Spin agent.
type Config struct {
	// LLM Provider Configuration
	Provider       string                 `yaml:"provider" mapstructure:"provider"`
	Model          string                 `yaml:"model" mapstructure:"model"`
	ProviderConfig map[string]interface{} `yaml:"provider_config" mapstructure:"provider_config"`

	// Execution Configuration
	MaxTurns int           `yaml:"max_turns" mapstructure:"max_turns"`
	Timeout  time.Duration `yaml:"timeout" mapstructure:"timeout"`
	WorkDir  string        `yaml:"work_dir" mapstructure:"work_dir"`

	// Security Configuration
	SandboxMode     string   `yaml:"sandbox_mode" mapstructure:"sandbox_mode"`
	PolicyFile      string   `yaml:"policy_file" mapstructure:"policy_file"`
	AllowedCommands []string `yaml:"allowed_commands" mapstructure:"allowed_commands"`

	// Feature Flags
	EnableMCP   bool              `yaml:"enable_mcp" mapstructure:"enable_mcp"`
	MCPServers  []MCPServerConfig `yaml:"mcp_servers" mapstructure:"mcp_servers"`
	EnableGit   bool              `yaml:"enable_git" mapstructure:"enable_git"`
	EnableShell bool              `yaml:"enable_shell" mapstructure:"enable_shell"`

	// Performance Configuration
	MaxTokens     int  `yaml:"max_tokens" mapstructure:"max_tokens"`
	StreamBuffer  int  `yaml:"stream_buffer" mapstructure:"stream_buffer"`
	CacheCommands bool `yaml:"cache_commands" mapstructure:"cache_commands"`

	// Storage Configuration
	SessionDir   string `yaml:"session_dir" mapstructure:"session_dir"`
	HistoryLimit int    `yaml:"history_limit" mapstructure:"history_limit"`

	// Logging Configuration
	LogLevel  string `yaml:"log_level" mapstructure:"log_level"`   // debug, info, warn, error
	LogFormat string `yaml:"log_format" mapstructure:"log_format"` // text, json
	Debug     bool   `yaml:"debug" mapstructure:"debug"`           // Enable debug mode

	// Tracing Configuration
	EnableTrace bool `yaml:"enable_trace" mapstructure:"enable_trace"` // Enable OpenTelemetry tracing
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
		MaxTurns:      50,
		Timeout:       5 * time.Minute,
		MaxTokens:     8192,
		StreamBuffer:  100,
		HistoryLimit:  1000,
		SessionDir:    "~/.spin/sessions",
		EnableGit:     true,
		EnableShell:   true,
		SandboxMode:   "workspace-only",
		CacheCommands: false,
		EnableMCP:     false,
		LogLevel:      "info",
		LogFormat:     "text",
		Debug:         false,
		EnableTrace:   false,
	}
}

// Load loads configuration from a YAML file.
func Load(path string) (*Config, error) {
	const op = "Config.Load"

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, &Error{
			Op:   op,
			Err:  err,
			Code: ErrCodeNotFound,
			Context: map[string]interface{}{
				"path": path,
			},
		}
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, &Error{
			Op:   op,
			Err:  err,
			Code: ErrCodeInvalidInput,
			Context: map[string]interface{}{
				"path": path,
			},
		}
	}

	return &cfg, nil
}

// LoadConfig loads configuration with full precedence chain:
// Defaults -> File -> Environment Variables
func LoadConfig(path string) (*Config, error) {
	const op = "LoadConfig"

	// Start with defaults
	cfg := DefaultConfig()

	// Load from file if exists
	if path != "" {
		fileCfg, err := Load(path)
		if err != nil {
			// Only return error if it's not a "file not found" error
			var e *Error
			if errors.As(err, &e) && e.Code != ErrCodeNotFound {
				return nil, &Error{Op: op, Err: err}
			}
		}
		if fileCfg != nil {
			cfg = cfg.Merge(fileCfg)
		}
	}

	// Override with environment variables
	envCfg := loadFromEnv()
	cfg = cfg.Merge(envCfg)

	// Validate final configuration
	if err := cfg.Validate(); err != nil {
		return nil, &Error{
			Op:   op,
			Err:  err,
			Code: ErrCodeInvalidInput,
		}
	}

	return cfg, nil
}

// LoadWithViper loads configuration using Viper for enhanced features.
// Supports YAML, JSON, TOML formats and environment variables.
// Precedence: Defaults -> Config File -> Environment Variables
func LoadWithViper(path string) (*Config, error) {
	const op = "LoadWithViper"

	// Create viper loader
	loader := config.NewLoader()

	// Set defaults
	setViperDefaults(loader)

	// Load config file
	var err error
	if path != "" {
		err = loader.LoadFromFile(path)
	} else {
		err = loader.Load("")
	}

	if err != nil {
		return nil, WrapError(op, err)
	}

	// Unmarshal into Config struct
	cfg := &Config{}
	if err := loader.Unmarshal(cfg); err != nil {
		return nil, NewInternalError(op, fmt.Errorf("failed to unmarshal config: %w", err))
	}

	// Validate
	if err := cfg.Validate(); err != nil {
		return nil, NewValidationError(op, err.Error())
	}

	return cfg, nil
}

// setViperDefaults sets default values in the Viper loader.
func setViperDefaults(loader *config.Loader) {
	defaults := DefaultConfig()

	loader.SetDefault("max_turns", defaults.MaxTurns)
	loader.SetDefault("timeout", defaults.Timeout)
	loader.SetDefault("max_tokens", defaults.MaxTokens)
	loader.SetDefault("stream_buffer", defaults.StreamBuffer)
	loader.SetDefault("history_limit", defaults.HistoryLimit)
	loader.SetDefault("session_dir", defaults.SessionDir)
	loader.SetDefault("enable_git", defaults.EnableGit)
	loader.SetDefault("enable_shell", defaults.EnableShell)
	loader.SetDefault("sandbox_mode", defaults.SandboxMode)
	loader.SetDefault("cache_commands", defaults.CacheCommands)
	loader.SetDefault("enable_mcp", defaults.EnableMCP)
	loader.SetDefault("log_level", defaults.LogLevel)
	loader.SetDefault("log_format", defaults.LogFormat)
	loader.SetDefault("debug", defaults.Debug)
	loader.SetDefault("enable_trace", defaults.EnableTrace)
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

	if c.MaxTokens <= 0 {
		errs = append(errs, fmt.Errorf("max_tokens must be > 0, got %d", c.MaxTokens))
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}

	return nil
}

// Merge merges two configurations with precedence (other overrides this).
// Returns a new Config (immutable operation).
func (c *Config) Merge(other *Config) *Config {
	merged := c.copyConfig()
	merged.mergeProviderConfig(other)
	merged.mergeStringFields(other)
	merged.mergeIntFields(other)
	merged.mergeSliceFields(other)
	merged.mergeBoolFields(c, other)
	return merged
}

// copyConfig creates a copy of the configuration
func (c *Config) copyConfig() *Config {
	copied := &Config{
		Provider:        c.Provider,
		Model:           c.Model,
		ProviderConfig:  make(map[string]interface{}),
		MaxTurns:        c.MaxTurns,
		Timeout:         c.Timeout,
		WorkDir:         c.WorkDir,
		SandboxMode:     c.SandboxMode,
		PolicyFile:      c.PolicyFile,
		AllowedCommands: append([]string{}, c.AllowedCommands...),
		EnableMCP:       c.EnableMCP,
		MCPServers:      append([]MCPServerConfig{}, c.MCPServers...),
		EnableGit:       c.EnableGit,
		EnableShell:     c.EnableShell,
		MaxTokens:       c.MaxTokens,
		StreamBuffer:    c.StreamBuffer,
		CacheCommands:   c.CacheCommands,
		SessionDir:      c.SessionDir,
		HistoryLimit:    c.HistoryLimit,
		LogLevel:        c.LogLevel,
		LogFormat:       c.LogFormat,
		Debug:           c.Debug,
		EnableTrace:     c.EnableTrace,
	}

	// Copy ProviderConfig map from base
	for k, v := range c.ProviderConfig {
		copied.ProviderConfig[k] = v
	}

	return copied
}

// mergeProviderConfig merges provider configuration maps from other into this
func (c *Config) mergeProviderConfig(other *Config) {
	for k, v := range other.ProviderConfig {
		c.ProviderConfig[k] = v
	}
}

// mergeStringFields merges string configuration fields
func (c *Config) mergeStringFields(other *Config) {
	if other.Provider != "" {
		c.Provider = other.Provider
	}
	if other.Model != "" {
		c.Model = other.Model
	}
	if other.WorkDir != "" {
		c.WorkDir = other.WorkDir
	}
	if other.SandboxMode != "" {
		c.SandboxMode = other.SandboxMode
	}
	if other.PolicyFile != "" {
		c.PolicyFile = other.PolicyFile
	}
	if other.SessionDir != "" {
		c.SessionDir = other.SessionDir
	}
	if other.LogLevel != "" {
		c.LogLevel = other.LogLevel
	}
	if other.LogFormat != "" {
		c.LogFormat = other.LogFormat
	}
}

// mergeIntFields merges integer and duration fields
func (c *Config) mergeIntFields(other *Config) {
	if other.MaxTurns != 0 {
		c.MaxTurns = other.MaxTurns
	}
	if other.Timeout != 0 {
		c.Timeout = other.Timeout
	}
	if other.MaxTokens != 0 {
		c.MaxTokens = other.MaxTokens
	}
	if other.StreamBuffer != 0 {
		c.StreamBuffer = other.StreamBuffer
	}
	if other.HistoryLimit != 0 {
		c.HistoryLimit = other.HistoryLimit
	}
}

// mergeSliceFields merges slice configuration fields
func (c *Config) mergeSliceFields(other *Config) {
	if len(other.AllowedCommands) > 0 {
		c.AllowedCommands = append(c.AllowedCommands, other.AllowedCommands...)
	}
	if len(other.MCPServers) > 0 {
		c.MCPServers = append(c.MCPServers, other.MCPServers...)
	}
}

// mergeBoolFields merges boolean configuration fields
func (c *Config) mergeBoolFields(base, other *Config) {
	if other.EnableMCP || (!other.EnableMCP && len(other.MCPServers) > 0) {
		c.EnableMCP = other.EnableMCP
	}
	if other.EnableGit != base.EnableGit {
		c.EnableGit = other.EnableGit
	}
	if other.EnableShell != base.EnableShell {
		c.EnableShell = other.EnableShell
	}
	if other.CacheCommands != base.CacheCommands {
		c.CacheCommands = other.CacheCommands
	}
	if other.Debug != base.Debug {
		c.Debug = other.Debug
	}
	if other.EnableTrace != base.EnableTrace {
		c.EnableTrace = other.EnableTrace
	}
}

// loadFromEnv loads configuration from environment variables.
func loadFromEnv() *Config {
	cfg := &Config{
		ProviderConfig: make(map[string]interface{}),
	}

	loadStringEnvVars(cfg)
	loadIntEnvVars(cfg)
	loadBoolEnvVars(cfg)

	// Parse duration
	if v := os.Getenv("SPIN_TIMEOUT"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			cfg.Timeout = d
		}
	}

	return cfg
}

// loadStringEnvVars loads string environment variables into config
func loadStringEnvVars(cfg *Config) {
	envMap := map[string]*string{
		"SPIN_PROVIDER":     &cfg.Provider,
		"SPIN_MODEL":        &cfg.Model,
		"SPIN_WORKDIR":      &cfg.WorkDir,
		"SPIN_WORK_DIR":     &cfg.WorkDir,
		"SPIN_SANDBOX_MODE": &cfg.SandboxMode,
		"SPIN_POLICY_FILE":  &cfg.PolicyFile,
		"SPIN_SESSION_DIR":  &cfg.SessionDir,
		"SPIN_LOG_LEVEL":    &cfg.LogLevel,
		"SPIN_LOG_FORMAT":   &cfg.LogFormat,
	}

	for key, dest := range envMap {
		if v := os.Getenv(key); v != "" {
			*dest = v
		}
	}
}

// loadIntEnvVars loads integer environment variables into config
func loadIntEnvVars(cfg *Config) {
	intEnvMap := map[string]*int{
		"SPIN_MAX_TURNS":     &cfg.MaxTurns,
		"SPIN_MAX_TOKENS":    &cfg.MaxTokens,
		"SPIN_STREAM_BUFFER": &cfg.StreamBuffer,
		"SPIN_HISTORY_LIMIT": &cfg.HistoryLimit,
	}

	for key, dest := range intEnvMap {
		if v := os.Getenv(key); v != "" {
			if i, err := strconv.Atoi(v); err == nil {
				*dest = i
			}
		}
	}
}

// loadBoolEnvVars loads boolean environment variables into config
func loadBoolEnvVars(cfg *Config) {
	boolEnvMap := map[string]*bool{
		"SPIN_ENABLE_MCP":     &cfg.EnableMCP,
		"SPIN_ENABLE_GIT":     &cfg.EnableGit,
		"SPIN_ENABLE_SHELL":   &cfg.EnableShell,
		"SPIN_CACHE_COMMANDS": &cfg.CacheCommands,
		"SPIN_ENABLE_TRACE":   &cfg.EnableTrace,
	}

	for key, dest := range boolEnvMap {
		if v := os.Getenv(key); v != "" {
			if b, err := strconv.ParseBool(v); err == nil {
				*dest = b
			}
		}
	}

	// Special handling for SPIN_DEBUG (can be "1" or "true")
	if v := os.Getenv("SPIN_DEBUG"); v != "" {
		if v == "1" || v == "true" {
			cfg.Debug = true
			cfg.LogLevel = "debug"
		}
	}
}
