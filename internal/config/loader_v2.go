package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/viper"
)

// Source specifies where to load configuration from.
type Source struct {
	// File path (empty = try default locations).
	File string

	// CLI flag overrides.
	Flags FlagOverrides

	// Runtime parameters.
	WorkDir string
}

// FlagOverrides contains CLI flag values.
type FlagOverrides struct {
	Provider string
	Model    string
	BaseURL  string
	APIKey   string
	MaxTurns int
	Debug    bool
	Sandbox  string
}

// LoaderV2 handles loading V2 from multiple sources with proper precedence.
// Precedence order: flags > environment > config file > defaults.
type LoaderV2 struct {
	viper *viper.Viper
}

// NewLoaderV2 creates a new configuration loader for V2.
func NewLoaderV2() *LoaderV2 {
	viperInst := viper.New()

	// Set config file properties.
	viperInst.SetConfigName("spin")
	viperInst.SetConfigType("yaml")
	viperInst.AddConfigPath(".")
	viperInst.AddConfigPath("$HOME/.spin")
	viperInst.AddConfigPath("/etc/spin")

	// Set environment variable prefix.
	viperInst.SetEnvPrefix("SPIN")
	viperInst.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viperInst.AutomaticEnv()

	// Bind all config keys to environment variables
	// This is required for Unmarshal to pick up env vars.
	bindEnvVars(viperInst)

	return &LoaderV2{viper: viperInst}
}

// bindEnvVars explicitly binds all config keys to environment variables.
// This is required because Viper's AutomaticEnv only works with Get(), not Unmarshal.
func bindEnvVars(viperInst *viper.Viper) {
	// LLM fields.
	_ = viperInst.BindEnv("llm.provider")
	_ = viperInst.BindEnv("llm.model")
	_ = viperInst.BindEnv("llm.temperature")
	_ = viperInst.BindEnv("llm.max_tokens")
	_ = viperInst.BindEnv("llm.timeout")
	_ = viperInst.BindEnv("llm.base_url")
	_ = viperInst.BindEnv("llm.api_key")

	// Agent fields.
	_ = viperInst.BindEnv("agent.max_turns")
	_ = viperInst.BindEnv("agent.timeout")
	_ = viperInst.BindEnv("agent.work_dir")
	_ = viperInst.BindEnv("agent.require_approval")

	// ACE fields.
	_ = viperInst.BindEnv("ace.enabled")
	_ = viperInst.BindEnv("ace.playbook_path")
	_ = viperInst.BindEnv("ace.trajectory_path")
	_ = viperInst.BindEnv("ace.top_k")
	_ = viperInst.BindEnv("ace.min_score")

	// Security fields.
	_ = viperInst.BindEnv("security.sandbox_mode")
	_ = viperInst.BindEnv("security.policy_file")
	_ = viperInst.BindEnv("security.allowed_commands")

	// Protocol fields.
	_ = viperInst.BindEnv("protocol.enable_mcp")
	_ = viperInst.BindEnv("protocol.enable_git")
	_ = viperInst.BindEnv("protocol.enable_shell")
	_ = viperInst.BindEnv("protocol.shell_timeout")

	// AgentsMD fields.
	_ = viperInst.BindEnv("agents_md.enabled")
	_ = viperInst.BindEnv("agents_md.path")
	_ = viperInst.BindEnv("agents_md.max_size")

	// Workflows fields.
	_ = viperInst.BindEnv("workflows.action_model")
	_ = viperInst.BindEnv("workflows.thinking_model")
	_ = viperInst.BindEnv("workflows.critique_model")
	_ = viperInst.BindEnv("workflows.compact_model")
	_ = viperInst.BindEnv("workflows.vision_model")
}

// LoadFromFile loads configuration from a specific YAML file.
func (l *LoaderV2) LoadFromFile(path string) (*V2, error) {
	l.viper.SetConfigFile(path)

	err := l.viper.ReadInConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	return l.unmarshalWithDefaults()
}

// LoadWithEnv loads configuration from environment variables and defaults.
func (l *LoaderV2) LoadWithEnv() (*V2, error) {
	// Don't try to read config file, just use env and defaults.
	return l.unmarshalWithDefaults()
}

// LoadFromFileWithEnv loads configuration from a file, with environment variable overrides.
func (l *LoaderV2) LoadFromFileWithEnv(path string) (*V2, error) {
	l.viper.SetConfigFile(path)

	err := l.viper.ReadInConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	return l.unmarshalWithDefaults()
}

// Load attempts to load configuration from default locations.
// It searches for config files in: ., ~/.spin, /etc/spin.
func (l *LoaderV2) Load() (*V2, error) {
	// Try to read config file from default locations
	// If not found, that's OK - we'll use defaults and env vars.
	_ = l.viper.ReadInConfig()

	return l.unmarshalWithDefaults()
}

// unmarshalWithDefaults unmarshals the configuration and applies defaults for missing values.
func (l *LoaderV2) unmarshalWithDefaults() (*V2, error) {
	// Unmarshal into a new config struct.
	cfg := &V2{}

	err := l.viper.Unmarshal(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Apply defaults for any unset fields.
	l.applyDefaults(cfg)

	// Validate the final configuration.
	err = cfg.Validate()
	if err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return cfg, nil
}

// applyDefaults applies default values to any unset fields in the config.
func (l *LoaderV2) applyDefaults(cfg *V2) {
	defaults := DefaultV2()

	// Apply version default.
	if !l.viper.IsSet("version") {
		cfg.Version = defaults.Version
	}

	l.applyLLMDefaults(cfg, defaults)
	l.applyAgentDefaults(cfg, defaults)
	l.applyACEDefaults(cfg, defaults)

	// Apply Security defaults.
	if !l.viper.IsSet("security.sandbox_mode") {
		cfg.Security.SandboxMode = defaults.Security.SandboxMode
	}

	l.applyProtocolDefaults(cfg, defaults)
	l.applyAgentsMDDefaults(cfg, defaults)
	l.applyMemoryDefaults(cfg, defaults)
}

// applyLLMDefaults applies default values for LLM fields.
func (l *LoaderV2) applyLLMDefaults(cfg, defaults *V2) {
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
}

// applyAgentDefaults applies default values for Agent fields.
func (l *LoaderV2) applyAgentDefaults(cfg, defaults *V2) {
	if !l.viper.IsSet("agent.max_turns") {
		cfg.Agent.MaxTurns = defaults.Agent.MaxTurns
	}

	if !l.viper.IsSet("agent.timeout") {
		cfg.Agent.Timeout = defaults.Agent.Timeout
	}

	if !l.viper.IsSet("agent.work_dir") {
		cfg.Agent.WorkDir = defaults.Agent.WorkDir
	}
}

// applyACEDefaults applies default values for ACE fields.
func (l *LoaderV2) applyACEDefaults(cfg, defaults *V2) {
	// Check if any ACE field was explicitly set.
	aceFieldsSet := l.viper.IsSet("ace.enabled") || l.viper.IsSet("ace.playbook_path") ||
		l.viper.IsSet("ace.trajectory_path") || l.viper.IsSet("ace.top_k") || l.viper.IsSet("ace.min_score")

	if !aceFieldsSet {
		cfg.ACE = defaults.ACE

		return
	}

	// Some ACE fields set, apply field-level defaults.
	if !l.viper.IsSet("ace.enabled") {
		cfg.ACE.Enabled = defaults.ACE.Enabled
	}

	if !cfg.ACE.Enabled {
		return
	}

	// Only apply path defaults if ACE is enabled.
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

// applyProtocolDefaults applies default values for Protocol fields.
func (l *LoaderV2) applyProtocolDefaults(cfg, defaults *V2) {
	if !l.viper.IsSet("protocol") {
		cfg.Protocol = defaults.Protocol

		return
	}

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

// applyAgentsMDDefaults applies default values for AgentsMD fields.
func (l *LoaderV2) applyAgentsMDDefaults(cfg, defaults *V2) {
	if !l.viper.IsSet("agents_md") {
		cfg.AgentsMD = defaults.AgentsMD

		return
	}

	if !l.viper.IsSet("agents_md.enabled") {
		cfg.AgentsMD.Enabled = defaults.AgentsMD.Enabled
	}

	if !l.viper.IsSet("agents_md.max_size") {
		cfg.AgentsMD.MaxSize = defaults.AgentsMD.MaxSize
	}
	// Path default is empty string (auto-discover), no need to apply.
}

// applyMemoryDefaults applies default values for Memory fields.
// Scratchpad is enabled by default; persistent memory requires opt-in.
func (l *LoaderV2) applyMemoryDefaults(cfg, defaults *V2) {
	if !l.viper.IsSet("memory") {
		cfg.Memory = defaults.Memory

		return
	}

	if !l.viper.IsSet("memory.scratchpad.enabled") {
		cfg.Memory.Scratchpad.Enabled = defaults.Memory.Scratchpad.Enabled
	}

	if !l.viper.IsSet("memory.scratchpad.max_entries") {
		cfg.Memory.Scratchpad.MaxEntries = defaults.Memory.Scratchpad.MaxEntries
	}

	if !l.viper.IsSet("memory.scratchpad.auto_evict") {
		cfg.Memory.Scratchpad.AutoEvict = defaults.Memory.Scratchpad.AutoEvict
	}
}

// Set sets a configuration value (useful for testing and programmatic config).
func (l *LoaderV2) Set(key string, value any) {
	l.viper.Set(key, value)
}

// Get retrieves a configuration value.
func (l *LoaderV2) Get(key string) any {
	return l.viper.Get(key)
}

// ConfigFileUsed returns the path to the config file being used.
func (l *LoaderV2) ConfigFileUsed() string {
	return l.viper.ConfigFileUsed()
}

// UnmarshalKey unmarshals a specific key into a provided struct.
func (l *LoaderV2) UnmarshalKey(key string, rawVal any) error {
	err := l.viper.UnmarshalKey(key, rawVal)
	if err != nil {
		return fmt.Errorf("unmarshal key %s: %w", key, err)
	}

	return nil
}

// AllSettings returns all settings as a map.
func (l *LoaderV2) AllSettings() map[string]any {
	return l.viper.AllSettings()
}

// Load loads and merges configuration from all sources.
// Precedence: flags > env > file > defaults.
func Load(src Source) (*V2, error) {
	loader := NewLoaderV2()

	var (
		cfg *V2
		err error
	)

	// Load from file if specified, otherwise search default paths.

	if src.File != "" {
		cfg, err = loader.LoadFromFile(src.File)
		if err != nil {
			return nil, fmt.Errorf("load config file: %w", err)
		}
	} else {
		// Search default paths: ., ~/.spin, /etc/spin.
		cfg, err = loader.Load()
		if err != nil {
			return nil, fmt.Errorf("load config: %w", err)
		}
	}

	// Apply flag overrides (before env so env knows the provider).
	if src.Flags.Provider != "" {
		cfg.LLM.Provider = src.Flags.Provider
	}

	if src.Flags.Model != "" {
		cfg.LLM.Model = src.Flags.Model
	}

	if src.Flags.BaseURL != "" {
		cfg.LLM.BaseURL = src.Flags.BaseURL
	}

	if src.Flags.MaxTurns > 0 {
		cfg.Agent.MaxTurns = src.Flags.MaxTurns
	}

	if src.Flags.Debug {
		cfg.Agent.Debug = true
		cfg.Agent.LogLevel = "debug"
	}

	if src.Flags.Sandbox != "" {
		cfg.Security.SandboxMode = src.Flags.Sandbox
	}

	// Apply environment variables (after flags so we know the provider)
	// Env vars fill in missing values but don't override explicit flags.
	applyEnvVars(cfg)

	// Merge MCP servers from mcp.servers into protocol.mcp_servers
	// The CLI stores servers at mcp.servers, so we need to merge them.
	mergeMCPServers(loader, cfg)

	// Override WorkDir if provided.
	if src.WorkDir != "" {
		cfg.Agent.WorkDir = src.WorkDir
	}

	return cfg, nil
}

// applyEnvVars applies environment variables to config.
func applyEnvVars(cfg *V2) {
	// Apply API key from environment based on provider.
	if cfg.LLM.APIKey == "" {
		apiKey := getAPIKeyFromEnv(cfg.LLM.Provider)
		if apiKey != "" {
			cfg.LLM.APIKey = apiKey
		}
	}
}

// mergeMCPServers merges MCP servers from mcp.servers into protocol.mcp_servers.
// The CLI (spin mcp add) stores servers at mcp.servers, while the runtime reads
// from protocol.mcp_servers. This function ensures both sources are unified.
func mergeMCPServers(loader *LoaderV2, cfg *V2) {
	// Try to load servers from mcp.servers path.
	var mcpServers []MCPServerConfigV2

	err := loader.UnmarshalKey("mcp.servers", &mcpServers)
	if err != nil {
		return // No servers at mcp.servers, nothing to merge.
	}

	if len(mcpServers) == 0 {
		return
	}

	// Build a set of existing server names for deduplication.
	existing := make(map[string]bool)
	for _, srv := range cfg.Protocol.MCPServers {
		existing[srv.Name] = true
	}

	// Append servers from mcp.servers that don't already exist.
	for _, srv := range mcpServers {
		if !existing[srv.Name] {
			cfg.Protocol.MCPServers = append(cfg.Protocol.MCPServers, srv)
		}
	}
}

// getAPIKeyFromEnv returns the API key from environment for the given provider.
func getAPIKeyFromEnv(provider string) string {
	envKey := getEnvKeyForProvider(provider)
	if envKey == "" {
		return ""
	}

	return os.Getenv(envKey)
}

// getEnvKeyForProvider returns the env var name for a provider's API key.
func getEnvKeyForProvider(provider string) string {
	switch provider {
	case "openai", "openai-compatible":
		return "OPENAI_API_KEY"
	case "anthropic":
		return "ANTHROPIC_API_KEY"
	default:
		return ""
	}
}
