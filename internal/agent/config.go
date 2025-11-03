package agent

import (
	"errors"
	"fmt"
	"time"
)

// Config contains unified configuration for the Spin agent.
// This replaces both manager.Config and agent.AgentConfig for consistency.
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
	EnableMCP   bool              `yaml:"enable_mcp" mapstructure:"enable_mcp"`
	MCPServers  []MCPServerConfig `yaml:"mcp_servers" mapstructure:"mcp_servers"`
	EnableGit   bool              `yaml:"enable_git" mapstructure:"enable_git"`
	EnableShell bool              `yaml:"enable_shell" mapstructure:"enable_shell"`

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

	// ACE Configuration
	ACE ACEConfig `yaml:"ace" mapstructure:"ace"`
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

// ACEConfig configures the Agentic Context Engineering system.
type ACEConfig struct {
	Enabled          bool                      `yaml:"enabled" mapstructure:"enabled"`
	PlaybookPath     string                    `yaml:"playbook_path" mapstructure:"playbook_path"`
	TrajectoryPath   string                    `yaml:"trajectory_path" mapstructure:"trajectory_path"`
	Retrieval        ACERetrievalConfig        `yaml:"retrieval" mapstructure:"retrieval"`
	ItemizedLearning ACEItemizedLearningConfig `yaml:"itemized_learning" mapstructure:"itemized_learning"`
	Generation       ACEGenerationConfig       `yaml:"generation" mapstructure:"generation"`
	Adapter          ACEAdapterConfig          `yaml:"adapter" mapstructure:"adapter"`
	Refine           ACERefineConfig           `yaml:"refine" mapstructure:"refine"`
}

// ACERetrievalConfig configures bullet retrieval behavior.
type ACERetrievalConfig struct {
	TopK               int                      `yaml:"top_k" mapstructure:"top_k"`
	MinScore           float64                  `yaml:"min_score" mapstructure:"min_score"`
	ProgressiveContext ProgressiveContextConfig `yaml:"progressive_context" mapstructure:"progressive_context"`
}

// ProgressiveContextConfig configures progressive retrieval behavior.
type ProgressiveContextConfig struct {
	// Core Settings

	// Enabled controls whether progressive context is active (default: true)
	Enabled bool `yaml:"enabled" mapstructure:"enabled"`

	// Cache Management

	// CacheTTL is the number of turns before cache expires (default: 10)
	CacheTTL int `yaml:"cache_ttl" mapstructure:"cache_ttl"`

	// MaxBullets is the maximum number of bullets to keep in cache (default: 50)
	MaxBullets int `yaml:"max_bullets" mapstructure:"max_bullets"`

	// EvictionStrategy determines how bullets are evicted when cache is full
	// Valid values: "lru" (Least Recently Used), "lfu" (Least Frequently Used), "fifo" (First In First Out)
	// Default: "lru"
	EvictionStrategy string `yaml:"eviction_strategy" mapstructure:"eviction_strategy"`

	// Trigger Configuration

	// ErrorLookback is the number of recent steps to check for errors (default: 5)
	ErrorLookback int `yaml:"error_lookback" mapstructure:"error_lookback"`

	// ToolChangeLookback is the number of recent steps to check for tool changes (default: 3)
	ToolChangeLookback int `yaml:"tool_change_lookback" mapstructure:"tool_change_lookback"`

	// EnabledTriggers lists which triggers are active (default: all)
	// Valid values: "initial", "error", "tool_change", "interval"
	EnabledTriggers []string `yaml:"enabled_triggers" mapstructure:"enabled_triggers"`

	// Query Composition

	// QueryWeights controls how different context components are weighted in query building
	QueryWeights QueryWeights `yaml:"query_weights" mapstructure:"query_weights"`

	// Performance Limits

	// MaxRetrievalLatencyMs is the maximum time to wait for retrieval (milliseconds)
	// If exceeded, uses cached bullets only (default: 500)
	MaxRetrievalLatencyMs int `yaml:"max_retrieval_latency_ms" mapstructure:"max_retrieval_latency_ms"`

	// MaxTrajectorySteps is the maximum number of steps to track (default: 1000)
	// Prevents unbounded memory growth in very long conversations
	MaxTrajectorySteps int `yaml:"max_trajectory_steps" mapstructure:"max_trajectory_steps"`

	// Observability

	// LogRetrievalDecisions enables logging of why retrieval was triggered (default: true)
	LogRetrievalDecisions bool `yaml:"log_retrieval_decisions" mapstructure:"log_retrieval_decisions"`

	// LogCacheStats enables logging of cache hit/miss statistics (default: true)
	LogCacheStats bool `yaml:"log_cache_stats" mapstructure:"log_cache_stats"`

	// EmitACEEvents enables event emission for TUI integration (default: true)
	EmitACEEvents bool `yaml:"emit_ace_events" mapstructure:"emit_ace_events"`
}

// QueryWeights controls how different context components are weighted in query building.
type QueryWeights struct {
	// InitialQuery is the weight for the base user query (0.0-1.0, default: 0.5)
	InitialQuery float64 `yaml:"initial_query" mapstructure:"initial_query"`

	// ErrorContext is the weight for error-derived context (0.0-1.0, default: 0.3)
	ErrorContext float64 `yaml:"error_context" mapstructure:"error_context"`

	// ToolContext is the weight for tool-derived context (0.0-1.0, default: 0.2)
	ToolContext float64 `yaml:"tool_context" mapstructure:"tool_context"`
}

// DefaultProgressiveContextConfig returns default configuration for progressive context.
func DefaultProgressiveContextConfig() ProgressiveContextConfig {
	return ProgressiveContextConfig{
		// Core
		Enabled: true, // Enabled by default

		// Cache Management
		CacheTTL:         10,    // 10 turns is reasonable for most tasks
		MaxBullets:       50,    // Limits memory while allowing good coverage
		EvictionStrategy: "lru", // Most recently used bullets are most relevant

		// Trigger Configuration
		ErrorLookback:      5,                                                       // Last 5 steps covers most error contexts
		ToolChangeLookback: 3,                                                       // Tool changes are usually immediate
		EnabledTriggers:    []string{"initial", "error", "tool_change", "interval"}, // All triggers enabled

		// Query Composition
		QueryWeights: QueryWeights{
			InitialQuery: 0.5, // Base query is most important
			ErrorContext: 0.3, // Error context is valuable
			ToolContext:  0.2, // Tool context provides useful hints
		},

		// Performance Limits
		MaxRetrievalLatencyMs: 500,  // 500ms keeps UX responsive
		MaxTrajectorySteps:    1000, // Prevents unbounded growth in long sessions

		// Observability
		LogRetrievalDecisions: true, // Helpful for debugging
		LogCacheStats:         true, // Useful for optimization
		EmitACEEvents:         true, // Enables TUI integration
	}
}

// ACEItemizedLearningConfig configures the ItemizedLearning workflow.
type ACEItemizedLearningConfig struct {
	Enabled       bool `yaml:"enabled" mapstructure:"enabled"`
	ParseFeedback bool `yaml:"parse_feedback" mapstructure:"parse_feedback"`
	UpdateAsync   bool `yaml:"update_async" mapstructure:"update_async"`
}

// ACEGenerationConfig configures bullet generation (Phase 3+).
type ACEGenerationConfig struct {
	Enabled     bool `yaml:"enabled" mapstructure:"enabled"`
	AutoReflect bool `yaml:"auto_reflect" mapstructure:"auto_reflect"`
}

// ACEAdapterConfig configures the online learning adapter.
type ACEAdapterConfig struct {
	Enabled          bool    `yaml:"enabled" mapstructure:"enabled"`
	UtilityThreshold float64 `yaml:"utility_threshold" mapstructure:"utility_threshold"`
	MaxMemorySize    int     `yaml:"max_memory_size" mapstructure:"max_memory_size"`
}

// ACERefineConfig configures playbook refinement and growth management.
type ACERefineConfig struct {
	Enabled         bool    `yaml:"enabled" mapstructure:"enabled"`
	Mode            string  `yaml:"mode" mapstructure:"mode"` // "none", "lazy", "proactive"
	MaxBullets      int     `yaml:"max_bullets" mapstructure:"max_bullets"`
	MaxTokens       int     `yaml:"max_tokens" mapstructure:"max_tokens"`
	MinUtilityScore float64 `yaml:"min_utility_score" mapstructure:"min_utility_score"`
	CheckInterval   int     `yaml:"check_interval" mapstructure:"check_interval"`
}

// Validate validates the ACE configuration.
func (c *ACEConfig) Validate() error {
	var errs []error

	// Validate retrieval config
	if err := c.Retrieval.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("retrieval: %w", err))
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}

	return nil
}

// Validate validates the ACE retrieval configuration.
func (c *ACERetrievalConfig) Validate() error {
	var errs []error

	if c.TopK <= 0 {
		errs = append(errs, fmt.Errorf("top_k must be > 0, got %d", c.TopK))
	}

	if c.MinScore < 0 || c.MinScore > 1 {
		errs = append(errs, fmt.Errorf("min_score must be between 0 and 1, got %f", c.MinScore))
	}

	// Validate progressive context config
	if err := c.ProgressiveContext.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("progressive_context: %w", err))
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}

	return nil
}

// Validate validates the progressive context configuration.
func (c *ProgressiveContextConfig) Validate() error {
	var errs []error

	// Validate cache settings
	if c.CacheTTL <= 0 {
		errs = append(errs, fmt.Errorf("cache_ttl must be > 0, got %d", c.CacheTTL))
	}

	if c.MaxBullets <= 0 {
		errs = append(errs, fmt.Errorf("max_bullets must be > 0, got %d", c.MaxBullets))
	}

	validStrategies := []string{"lru", "lfu", "fifo"}
	if !stringSliceContains(validStrategies, c.EvictionStrategy) {
		errs = append(errs, fmt.Errorf("eviction_strategy must be one of %v, got %q", validStrategies, c.EvictionStrategy))
	}

	// Validate lookback windows
	if c.ErrorLookback <= 0 {
		errs = append(errs, fmt.Errorf("error_lookback must be > 0, got %d", c.ErrorLookback))
	}

	if c.ToolChangeLookback <= 0 {
		errs = append(errs, fmt.Errorf("tool_change_lookback must be > 0, got %d", c.ToolChangeLookback))
	}

	// Validate triggers
	validTriggers := []string{"initial", "error", "tool_change", "interval"}
	for _, trigger := range c.EnabledTriggers {
		if !stringSliceContains(validTriggers, trigger) {
			errs = append(errs, fmt.Errorf("invalid trigger %q, must be one of %v", trigger, validTriggers))
		}
	}

	if len(c.EnabledTriggers) == 0 {
		errs = append(errs, fmt.Errorf("enabled_triggers cannot be empty"))
	}

	// Validate query weights
	if err := c.QueryWeights.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("query_weights: %w", err))
	}

	// Validate performance limits
	if c.MaxRetrievalLatencyMs <= 0 {
		errs = append(errs, fmt.Errorf("max_retrieval_latency_ms must be > 0, got %d", c.MaxRetrievalLatencyMs))
	}

	if c.MaxTrajectorySteps <= 0 {
		errs = append(errs, fmt.Errorf("max_trajectory_steps must be > 0, got %d", c.MaxTrajectorySteps))
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}

	return nil
}

// Validate validates query weights configuration.
func (qw *QueryWeights) Validate() error {
	var errs []error

	if qw.InitialQuery < 0 || qw.InitialQuery > 1 {
		errs = append(errs, fmt.Errorf("initial_query must be between 0 and 1, got %f", qw.InitialQuery))
	}

	if qw.ErrorContext < 0 || qw.ErrorContext > 1 {
		errs = append(errs, fmt.Errorf("error_context must be between 0 and 1, got %f", qw.ErrorContext))
	}

	if qw.ToolContext < 0 || qw.ToolContext > 1 {
		errs = append(errs, fmt.Errorf("tool_context must be between 0 and 1, got %f", qw.ToolContext))
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}

	return nil
}

// stringSliceContains checks if a string slice contains a value.
func stringSliceContains(slice []string, value string) bool {
	for _, item := range slice {
		if item == value {
			return true
		}
	}
	return false
}

// DefaultConfig returns a new Config with sensible defaults.
func DefaultConfig() *Config {
	return &Config{
		// LLM Provider defaults
		Temperature: 0.7,

		// Agent defaults
		MaxTurns:        500,              // Increased from 50 to allow complex multi-step tasks
		Timeout:         60 * time.Minute, // Increased from 5min to 1h for long-running tasks
		MaxTokens:       8192,
		RequireApproval: false,

		// Security defaults
		SandboxMode: "workspace-only",

		// Feature flags
		EnableGit:   true,
		EnableShell: true,
		EnableMCP:   false,

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

		// ACE defaults
		ACE: ACEConfig{
			Enabled:        true,
			PlaybookPath:   "~/.spin/ace/playbooks/default.json",
			TrajectoryPath: "~/.spin/ace/trajectories/",
			Retrieval: ACERetrievalConfig{
				TopK:               100, // High limit - let MinScore filter relevance
				MinScore:           0.3,
				ProgressiveContext: DefaultProgressiveContextConfig(),
			},
			ItemizedLearning: ACEItemizedLearningConfig{
				Enabled:       true,
				ParseFeedback: true,
				UpdateAsync:   true,
			},
			Generation: ACEGenerationConfig{
				Enabled:     true, // Enable bullet generation by default
				AutoReflect: true, // Use reflector+curator pipeline
			},
			Adapter: ACEAdapterConfig{
				Enabled:          true,
				UtilityThreshold: 0.1,  // Minimum utility score to keep bullets
				MaxMemorySize:    1000, // Max bullets in memory before refinement
			},
			Refine: ACERefineConfig{
				Enabled:         true,
				Mode:            "proactive", // Auto-trigger refinement
				MaxBullets:      1000,        // Trigger at 1000 bullets
				MaxTokens:       500000,      // Trigger at 500K tokens
				MinUtilityScore: 0.1,         // Prune bullets below this score
				CheckInterval:   100,         // Check every 100 bullets added
			},
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

	// Validate ACE config
	if err := c.ACE.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("ace: %w", err))
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

// Merge merges two configurations with precedence (other overrides this).
// Returns a new Config (immutable operation).
func (c *Config) Merge(other *Config) *Config {
	merged := c.copyConfig()
	merged.mergeProviderConfig(other)
	merged.mergeStringFields(other)
	merged.mergeIntFields(other)
	merged.mergeSliceFields(other)
	merged.mergeBoolFields(c, other)
	merged.mergeFloatFields(other)
	merged.mergeCycleDetection(other)
	return merged
}

// mergeFloatFields merges float configuration fields
func (c *Config) mergeFloatFields(other *Config) {
	if other.Temperature != 0 {
		c.Temperature = other.Temperature
	}
}

// mergeCycleDetection merges cycle detection configuration
func (c *Config) mergeCycleDetection(other *Config) {
	if other.CycleDetection.Enabled != c.CycleDetection.Enabled {
		c.CycleDetection.Enabled = other.CycleDetection.Enabled
	}
	if other.CycleDetection.WindowSize != 0 {
		c.CycleDetection.WindowSize = other.CycleDetection.WindowSize
	}
	if other.CycleDetection.SimilarityThresh != 0 {
		c.CycleDetection.SimilarityThresh = other.CycleDetection.SimilarityThresh
	}
	if other.CycleDetection.ToolRepeatLimit != 0 {
		c.CycleDetection.ToolRepeatLimit = other.CycleDetection.ToolRepeatLimit
	}
	if other.CycleDetection.ErrorRepeatLimit != 0 {
		c.CycleDetection.ErrorRepeatLimit = other.CycleDetection.ErrorRepeatLimit
	}
}

// copyConfig creates a copy of the configuration
func (c *Config) copyConfig() *Config {
	copied := &Config{
		// LLM Provider fields
		Provider:       c.Provider,
		Model:          c.Model,
		ProviderConfig: make(map[string]interface{}),
		Temperature:    c.Temperature,

		// Agent fields
		MaxTurns:        c.MaxTurns,
		Timeout:         c.Timeout,
		WorkDir:         c.WorkDir,
		MaxTokens:       c.MaxTokens,
		RequireApproval: c.RequireApproval,

		// Security fields
		SandboxMode:     c.SandboxMode,
		PolicyFile:      c.PolicyFile,
		AllowedCommands: append([]string{}, c.AllowedCommands...),

		// Feature flags
		EnableMCP:   c.EnableMCP,
		MCPServers:  append([]MCPServerConfig{}, c.MCPServers...),
		EnableGit:   c.EnableGit,
		EnableShell: c.EnableShell,

		// Performance fields
		StreamBuffer:  c.StreamBuffer,
		CacheCommands: c.CacheCommands,

		// Storage fields
		SessionDir:   c.SessionDir,
		HistoryLimit: c.HistoryLimit,

		// Logging fields
		LogLevel:  c.LogLevel,
		LogFormat: c.LogFormat,
		Debug:     c.Debug,

		// Cycle detection fields
		CycleDetection: c.CycleDetection,
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
	if other.RequireApproval != base.RequireApproval {
		c.RequireApproval = other.RequireApproval
	}
}
