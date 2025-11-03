package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

// LoaderV2 handles loading ConfigV2 from multiple sources with proper precedence.
// Precedence order: flags > environment > config file > defaults
type LoaderV2 struct {
	viper *viper.Viper
}

// NewLoaderV2 creates a new configuration loader for ConfigV2.
func NewLoaderV2() *LoaderV2 {
	v := viper.New()

	// Set config file properties
	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath(".")
	v.AddConfigPath("$HOME/.spin")
	v.AddConfigPath("/etc/spin")

	// Set environment variable prefix
	v.SetEnvPrefix("SPIN")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	// Bind all config keys to environment variables
	// This is required for Unmarshal to pick up env vars
	bindEnvVars(v)

	return &LoaderV2{viper: v}
}

// bindEnvVars explicitly binds all config keys to environment variables.
// This is required because Viper's AutomaticEnv only works with Get(), not Unmarshal.
func bindEnvVars(v *viper.Viper) {
	// LLM fields
	v.BindEnv("llm.provider")
	v.BindEnv("llm.model")
	v.BindEnv("llm.temperature")
	v.BindEnv("llm.max_tokens")
	v.BindEnv("llm.timeout")
	v.BindEnv("llm.base_url")
	v.BindEnv("llm.api_key")

	// Agent fields
	v.BindEnv("agent.max_turns")
	v.BindEnv("agent.timeout")
	v.BindEnv("agent.work_dir")
	v.BindEnv("agent.require_approval")

	// ACE fields
	v.BindEnv("ace.enabled")
	v.BindEnv("ace.playbook_path")
	v.BindEnv("ace.trajectory_path")
	v.BindEnv("ace.top_k")
	v.BindEnv("ace.min_score")

	// Security fields
	v.BindEnv("security.sandbox_mode")
	v.BindEnv("security.policy_file")
	v.BindEnv("security.allowed_commands")

	// Protocol fields
	v.BindEnv("protocol.enable_mcp")
	v.BindEnv("protocol.enable_git")
	v.BindEnv("protocol.enable_shell")
	v.BindEnv("protocol.shell_timeout")
}

// LoadFromFile loads configuration from a specific YAML file.
func (l *LoaderV2) LoadFromFile(path string) (*ConfigV2, error) {
	l.viper.SetConfigFile(path)

	if err := l.viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	return l.unmarshalWithDefaults()
}

// LoadWithEnv loads configuration from environment variables and defaults.
func (l *LoaderV2) LoadWithEnv() (*ConfigV2, error) {
	// Don't try to read config file, just use env and defaults
	return l.unmarshalWithDefaults()
}

// LoadFromFileWithEnv loads configuration from a file, with environment variable overrides.
func (l *LoaderV2) LoadFromFileWithEnv(path string) (*ConfigV2, error) {
	l.viper.SetConfigFile(path)

	if err := l.viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	return l.unmarshalWithDefaults()
}

// Load attempts to load configuration from default locations.
// It searches for config files in: ., ~/.spin, /etc/spin
func (l *LoaderV2) Load() (*ConfigV2, error) {
	// Try to read config file from default locations
	// If not found, that's OK - we'll use defaults and env vars
	_ = l.viper.ReadInConfig()

	return l.unmarshalWithDefaults()
}

// unmarshalWithDefaults unmarshals the configuration and applies defaults for missing values.
func (l *LoaderV2) unmarshalWithDefaults() (*ConfigV2, error) {
	// Unmarshal into a new config struct
	cfg := &ConfigV2{}
	if err := l.viper.Unmarshal(cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Apply defaults for any unset fields
	l.applyDefaults(cfg)

	// Validate the final configuration
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return cfg, nil
}

// applyDefaults applies default values to any unset fields in the config.
func (l *LoaderV2) applyDefaults(cfg *ConfigV2) {
	defaults := DefaultConfigV2()

	// Apply version default
	if !l.viper.IsSet("version") {
		cfg.Version = defaults.Version
	}

	// Apply LLM defaults
	if !l.viper.IsSet("llm.provider") {
		cfg.LLM.Provider = defaults.LLM.Provider
	}
	if !l.viper.IsSet("llm.model") {
		cfg.LLM.Model = defaults.LLM.Model
	}
	if !l.viper.IsSet("llm.temperature") {
		cfg.LLM.Temperature = defaults.LLM.Temperature
	}
	if !l.viper.IsSet("llm.max_tokens") {
		cfg.LLM.MaxTokens = defaults.LLM.MaxTokens
	}
	if !l.viper.IsSet("llm.timeout") {
		cfg.LLM.Timeout = defaults.LLM.Timeout
	}

	// Apply Agent defaults
	if !l.viper.IsSet("agent.max_turns") {
		cfg.Agent.MaxTurns = defaults.Agent.MaxTurns
	}
	if !l.viper.IsSet("agent.timeout") {
		cfg.Agent.Timeout = defaults.Agent.Timeout
	}
	if !l.viper.IsSet("agent.work_dir") {
		cfg.Agent.WorkDir = defaults.Agent.WorkDir
	}

	// Apply ACE defaults
	// Check if any ACE field was explicitly set
	aceFieldsSet := l.viper.IsSet("ace.enabled") || l.viper.IsSet("ace.playbook_path") ||
		l.viper.IsSet("ace.trajectory_path") || l.viper.IsSet("ace.top_k") || l.viper.IsSet("ace.min_score")

	if !aceFieldsSet {
		// No ACE fields set - apply full defaults
		cfg.ACE = defaults.ACE
	} else {
		// Some ACE fields set, apply field-level defaults
		if !l.viper.IsSet("ace.enabled") {
			cfg.ACE.Enabled = defaults.ACE.Enabled
		}
		if cfg.ACE.Enabled {
			// Only apply path defaults if ACE is enabled
			if !l.viper.IsSet("ace.playbook_path") {
				cfg.ACE.PlaybookPath = defaults.ACE.PlaybookPath
			}
			if !l.viper.IsSet("ace.trajectory_path") {
				cfg.ACE.TrajectoryPath = defaults.ACE.TrajectoryPath
			}
			if !l.viper.IsSet("ace.top_k") {
				cfg.ACE.TopK = defaults.ACE.TopK
			}
			if !l.viper.IsSet("ace.min_score") {
				cfg.ACE.MinScore = defaults.ACE.MinScore
			}
		}
	}

	// Apply Security defaults
	if !l.viper.IsSet("security.sandbox_mode") {
		cfg.Security.SandboxMode = defaults.Security.SandboxMode
	}

	// Apply Protocol defaults
	if !l.viper.IsSet("protocol") {
		cfg.Protocol = defaults.Protocol
	} else {
		if !l.viper.IsSet("protocol.enable_mcp") {
			cfg.Protocol.EnableMCP = defaults.Protocol.EnableMCP
		}
		if !l.viper.IsSet("protocol.enable_git") {
			cfg.Protocol.EnableGit = defaults.Protocol.EnableGit
		}
		if !l.viper.IsSet("protocol.enable_shell") {
			cfg.Protocol.EnableShell = defaults.Protocol.EnableShell
		}
		if !l.viper.IsSet("protocol.shell_timeout") {
			cfg.Protocol.ShellTimeout = defaults.Protocol.ShellTimeout
		}
	}
}

// Set sets a configuration value (useful for testing and programmatic config).
func (l *LoaderV2) Set(key string, value interface{}) {
	l.viper.Set(key, value)
}

// Get retrieves a configuration value.
func (l *LoaderV2) Get(key string) interface{} {
	return l.viper.Get(key)
}
