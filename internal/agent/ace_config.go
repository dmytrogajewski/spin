package agent

import (
	"errors"
	"fmt"
	"slices"

	"github.com/dmytrogajewski/spin/internal/config"
)

// ACEConfig configures the Agentic Context Engineering system.
type ACEConfig struct {
	Enabled          bool                      `mapstructure:"enabled"           yaml:"enabled"`
	PlaybookPath     string                    `mapstructure:"playbook_path"     yaml:"playbook_path"`
	TrajectoryPath   string                    `mapstructure:"trajectory_path"   yaml:"trajectory_path"`
	Retrieval        ACERetrievalConfig        `mapstructure:"retrieval"         yaml:"retrieval"`
	ItemizedLearning ACEItemizedLearningConfig `mapstructure:"itemized_learning" yaml:"itemized_learning"`
	Generation       ACEGenerationConfig       `mapstructure:"generation"        yaml:"generation"`
	Adapter          ACEAdapterConfig          `mapstructure:"adapter"           yaml:"adapter"`
	Refine           ACERefineConfig           `mapstructure:"refine"            yaml:"refine"`
}

// ConvertACEConfig converts config.ACEConfigV2 to agent.ACEConfig.
// This is needed because the conversation package uses config.ACEConfigV2
// but the agent package uses its own ACEConfig type.
func ConvertACEConfig(v2cfg *config.ACEConfigV2) *ACEConfig {
	if v2cfg == nil {
		return nil
	}

	return &ACEConfig{
		Enabled:        v2cfg.Enabled,
		PlaybookPath:   v2cfg.PlaybookPath,
		TrajectoryPath: v2cfg.TrajectoryPath,
		Retrieval: ACERetrievalConfig{
			TopK:               v2cfg.TopK,
			MinScore:           v2cfg.MinScore,
			ProgressiveContext: DefaultProgressiveContextConfig(),
		},
		ItemizedLearning: ACEItemizedLearningConfig{
			Enabled:       true,
			ParseFeedback: true,
			UpdateAsync:   true,
		},
		Generation: ACEGenerationConfig{
			Enabled:     true,
			AutoReflect: true,
		},
		Adapter: ACEAdapterConfig{
			Enabled:          true,
			UtilityThreshold: 0.1,
			MaxMemorySize:    1000,
		},
		Refine: ACERefineConfig{
			Enabled:         true,
			Mode:            "proactive",
			MaxBullets:      1000,
			MaxTokens:       500000,
			MinUtilityScore: 0.1,
			CheckInterval:   100,
		},
	}
}

// ACERetrievalConfig configures bullet retrieval behavior.
type ACERetrievalConfig struct {
	TopK               int                      `mapstructure:"top_k"               yaml:"top_k"`
	MinScore           float64                  `mapstructure:"min_score"           yaml:"min_score"`
	ProgressiveContext ProgressiveContextConfig `mapstructure:"progressive_context" yaml:"progressive_context"`
}

// ProgressiveContextConfig configures progressive retrieval behavior.
type ProgressiveContextConfig struct {
	// Core Settings.

	// Enabled controls whether progressive context is active (default: true).
	Enabled bool `mapstructure:"enabled" yaml:"enabled"`

	// Cache Management.

	// CacheTTL is the number of turns before cache expires (default: 10).
	CacheTTL int `mapstructure:"cache_ttl" yaml:"cache_ttl"`

	// MaxBullets is the maximum number of bullets to keep in cache (default: 50).
	MaxBullets int `mapstructure:"max_bullets" yaml:"max_bullets"`

	// EvictionStrategy determines how bullets are evicted when cache is full
	// Valid values: "lru" (Least Recently Used), "lfu" (Least Frequently Used), "fifo" (First In First Out)
	// Default: "lru".
	EvictionStrategy string `mapstructure:"eviction_strategy" yaml:"eviction_strategy"`

	// Trigger Configuration.

	// ErrorLookback is the number of recent steps to check for errors (default: 5).
	ErrorLookback int `mapstructure:"error_lookback" yaml:"error_lookback"`

	// ToolChangeLookback is the number of recent steps to check for tool changes (default: 3).
	ToolChangeLookback int `mapstructure:"tool_change_lookback" yaml:"tool_change_lookback"`

	// EnabledTriggers lists which triggers are active (default: all)
	// Valid values: "initial", "error", "tool_change", "interval".
	EnabledTriggers []string `mapstructure:"enabled_triggers" yaml:"enabled_triggers"`

	// Query Composition.

	// QueryWeights controls how different context components are weighted in query building.
	QueryWeights QueryWeights `mapstructure:"query_weights" yaml:"query_weights"`

	// Performance Limits.

	// MaxRetrievalLatencyMs is the maximum time to wait for retrieval (milliseconds)
	// If exceeded, uses cached bullets only (default: 500).
	MaxRetrievalLatencyMs int `mapstructure:"max_retrieval_latency_ms" yaml:"max_retrieval_latency_ms"`

	// MaxTrajectorySteps is the maximum number of steps to track (default: 1000)
	// Prevents unbounded memory growth in very long conversations.
	MaxTrajectorySteps int `mapstructure:"max_trajectory_steps" yaml:"max_trajectory_steps"`

	// Observability.

	// LogRetrievalDecisions enables logging of why retrieval was triggered (default: true).
	LogRetrievalDecisions bool `mapstructure:"log_retrieval_decisions" yaml:"log_retrieval_decisions"`

	// LogCacheStats enables logging of cache hit/miss statistics (default: true).
	LogCacheStats bool `mapstructure:"log_cache_stats" yaml:"log_cache_stats"`

	// EmitACEEvents enables event emission for TUI integration (default: true).
	EmitACEEvents bool `mapstructure:"emit_ace_events" yaml:"emit_ace_events"`
}

// QueryWeights controls how different context components are weighted in query building.
type QueryWeights struct {
	// InitialQuery is the weight for the base user query (0.0-1.0, default: 0.5).
	InitialQuery float64 `mapstructure:"initial_query" yaml:"initial_query"`

	// ErrorContext is the weight for error-derived context (0.0-1.0, default: 0.3).
	ErrorContext float64 `mapstructure:"error_context" yaml:"error_context"`

	// ToolContext is the weight for tool-derived context (0.0-1.0, default: 0.2).
	ToolContext float64 `mapstructure:"tool_context" yaml:"tool_context"`
}

// DefaultProgressiveContextConfig returns default configuration for progressive context.
func DefaultProgressiveContextConfig() ProgressiveContextConfig {
	return ProgressiveContextConfig{
		// Core.
		Enabled: true, // Enabled by default.

		// Cache Management.
		CacheTTL:         10,    // 10 turns is reasonable for most tasks.
		MaxBullets:       50,    // Limits memory while allowing good coverage.
		EvictionStrategy: "lru", // Most recently used bullets are most relevant.

		// Trigger Configuration.
		ErrorLookback:      5,                                                       // Last 5 steps covers most error contexts.
		ToolChangeLookback: 3,                                                       // Tool changes are usually immediate.
		EnabledTriggers:    []string{"initial", "error", "tool_change", "interval"}, // All triggers enabled.

		// Query Composition.
		QueryWeights: QueryWeights{
			InitialQuery: 0.5, // Base query is most important.
			ErrorContext: 0.3, // Error context is valuable.
			ToolContext:  0.2, // Tool context provides useful hints.
		},

		// Performance Limits.
		MaxRetrievalLatencyMs: 500,  // 500ms keeps UX responsive.
		MaxTrajectorySteps:    1000, // Prevents unbounded growth in long sessions.

		// Observability.
		LogRetrievalDecisions: true, // Helpful for debugging.
		LogCacheStats:         true, // Useful for optimization.
		EmitACEEvents:         true, // Enables TUI integration.
	}
}

// ACEItemizedLearningConfig configures the ItemizedLearning workflow.
type ACEItemizedLearningConfig struct {
	Enabled       bool `mapstructure:"enabled"        yaml:"enabled"`
	ParseFeedback bool `mapstructure:"parse_feedback" yaml:"parse_feedback"`
	UpdateAsync   bool `mapstructure:"update_async"   yaml:"update_async"`
}

// ACEGenerationConfig configures bullet generation (Phase 3+).
type ACEGenerationConfig struct {
	Enabled     bool `mapstructure:"enabled"      yaml:"enabled"`
	AutoReflect bool `mapstructure:"auto_reflect" yaml:"auto_reflect"`
}

// ACEAdapterConfig configures the online learning adapter.
type ACEAdapterConfig struct {
	Enabled          bool    `mapstructure:"enabled"           yaml:"enabled"`
	UtilityThreshold float64 `mapstructure:"utility_threshold" yaml:"utility_threshold"`
	MaxMemorySize    int     `mapstructure:"max_memory_size"   yaml:"max_memory_size"`
}

// ACERefineConfig configures playbook refinement and growth management.
type ACERefineConfig struct {
	Enabled         bool    `mapstructure:"enabled"           yaml:"enabled"`
	Mode            string  `mapstructure:"mode"              yaml:"mode"` // "none", "lazy", "proactive".
	MaxBullets      int     `mapstructure:"max_bullets"       yaml:"max_bullets"`
	MaxTokens       int     `mapstructure:"max_tokens"        yaml:"max_tokens"`
	MinUtilityScore float64 `mapstructure:"min_utility_score" yaml:"min_utility_score"`
	CheckInterval   int     `mapstructure:"check_interval"    yaml:"check_interval"`
}

// Validate validates the ACE configuration.
func (c *ACEConfig) Validate() error {
	var errs []error

	// Validate retrieval config.
	err := c.Retrieval.Validate()
	if err != nil {
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

	// Validate progressive context config.
	err := c.ProgressiveContext.Validate()
	if err != nil {
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

	// Validate cache settings.
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

	// Validate lookback windows.
	if c.ErrorLookback <= 0 {
		errs = append(errs, fmt.Errorf("error_lookback must be > 0, got %d", c.ErrorLookback))
	}

	if c.ToolChangeLookback <= 0 {
		errs = append(errs, fmt.Errorf("tool_change_lookback must be > 0, got %d", c.ToolChangeLookback))
	}

	// Validate query weights.
	err := c.QueryWeights.Validate()
	if err != nil {
		errs = append(errs, fmt.Errorf("query_weights: %w", err))
	}

	// Validate performance limits.
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
func (q *QueryWeights) Validate() error {
	var errs []error

	if q.InitialQuery < 0 || q.InitialQuery > 1 {
		errs = append(errs, fmt.Errorf("initial_query must be between 0 and 1, got %f", q.InitialQuery))
	}

	if q.ErrorContext < 0 || q.ErrorContext > 1 {
		errs = append(errs, fmt.Errorf("error_context must be between 0 and 1, got %f", q.ErrorContext))
	}

	if q.ToolContext < 0 || q.ToolContext > 1 {
		errs = append(errs, fmt.Errorf("tool_context must be between 0 and 1, got %f", q.ToolContext))
	}

	// Weights should sum to ~1.0 (allow some tolerance).
	sum := q.InitialQuery + q.ErrorContext + q.ToolContext
	if sum < 0.9 || sum > 1.1 {
		errs = append(errs, fmt.Errorf("query weights should sum to approximately 1.0, got %f", sum))
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}

	return nil
}

// stringSliceContains checks if a string slice contains a given string.
func stringSliceContains(slice []string, str string) bool {
	return slices.Contains(slice, str)
}
