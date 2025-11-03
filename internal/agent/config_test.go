package agent

import "testing"

// Test ACE config is present in Config struct
func TestConfig_ACE_Field(t *testing.T) {
	cfg := DefaultConfig()

	if !cfg.ACE.Enabled {
		t.Error("ACE should be enabled by default")
	}
}

// Test ACE config has default playbook path
func TestConfig_ACE_PlaybookPath(t *testing.T) {
	cfg := DefaultConfig()

	expected := "~/.spin/ace/playbooks/default.json"
	if cfg.ACE.PlaybookPath != expected {
		t.Errorf("ACE.PlaybookPath = %q, want %q", cfg.ACE.PlaybookPath, expected)
	}
}

// Test ACE config has all required fields with correct defaults
func TestConfig_ACE_Defaults(t *testing.T) {
	cfg := DefaultConfig()

	tests := []struct {
		name string
		got  interface{}
		want interface{}
	}{
		{"Enabled", cfg.ACE.Enabled, true},
		{"PlaybookPath", cfg.ACE.PlaybookPath, "~/.spin/ace/playbooks/default.json"},
		{"TrajectoryPath", cfg.ACE.TrajectoryPath, "~/.spin/ace/trajectories/"},
		{"Retrieval.TopK", cfg.ACE.Retrieval.TopK, 100},
		{"Retrieval.MinScore", cfg.ACE.Retrieval.MinScore, 0.3},
		{"ItemizedLearning.Enabled", cfg.ACE.ItemizedLearning.Enabled, true},
		{"ItemizedLearning.ParseFeedback", cfg.ACE.ItemizedLearning.ParseFeedback, true},
		{"ItemizedLearning.UpdateAsync", cfg.ACE.ItemizedLearning.UpdateAsync, true},
		{"Generation.Enabled", cfg.ACE.Generation.Enabled, true},
		{"Generation.AutoReflect", cfg.ACE.Generation.AutoReflect, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.want {
				t.Errorf("%s = %v, want %v", tt.name, tt.got, tt.want)
			}
		})
	}
}

// Test ACE config validation
func TestConfig_ACE_Validation(t *testing.T) {
	tests := []struct {
		name    string
		modify  func(*Config)
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid config",
			modify:  func(c *Config) {},
			wantErr: false,
		},
		{
			name: "invalid TopK (zero)",
			modify: func(c *Config) {
				c.ACE.Retrieval.TopK = 0
			},
			wantErr: true,
			errMsg:  "top_k must be > 0",
		},
		{
			name: "invalid TopK (negative)",
			modify: func(c *Config) {
				c.ACE.Retrieval.TopK = -1
			},
			wantErr: true,
			errMsg:  "top_k must be > 0",
		},
		{
			name: "invalid MinScore (negative)",
			modify: func(c *Config) {
				c.ACE.Retrieval.MinScore = -0.1
			},
			wantErr: true,
			errMsg:  "min_score must be between 0 and 1",
		},
		{
			name: "invalid MinScore (too high)",
			modify: func(c *Config) {
				c.ACE.Retrieval.MinScore = 1.5
			},
			wantErr: true,
			errMsg:  "min_score must be between 0 and 1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := DefaultConfig()
			cfg.Provider = "openai"
			cfg.Model = "gpt-4"
			tt.modify(cfg)

			err := cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && err != nil {
				if tt.errMsg != "" && !contains(err.Error(), tt.errMsg) {
					t.Errorf("Validate() error = %v, want error containing %q", err, tt.errMsg)
				}
			}
		})
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || containsMiddle(s, substr)))
}

func containsMiddle(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// Test QueryWeights validation - valid weights
func TestQueryWeights_Validate_Valid(t *testing.T) {
	tests := []struct {
		name    string
		weights QueryWeights
	}{
		{
			name: "default weights",
			weights: QueryWeights{
				InitialQuery: 0.5,
				ErrorContext: 0.3,
				ToolContext:  0.2,
			},
		},
		{
			name: "all zeros",
			weights: QueryWeights{
				InitialQuery: 0.0,
				ErrorContext: 0.0,
				ToolContext:  0.0,
			},
		},
		{
			name: "all ones (boundary)",
			weights: QueryWeights{
				InitialQuery: 1.0,
				ErrorContext: 1.0,
				ToolContext:  1.0,
			},
		},
		{
			name: "sum not 1.0 (allowed)",
			weights: QueryWeights{
				InitialQuery: 0.4,
				ErrorContext: 0.4,
				ToolContext:  0.4, // sum = 1.2, allowed
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.weights.Validate()
			if err != nil {
				t.Errorf("Validate() unexpected error = %v", err)
			}
		})
	}
}

// Test QueryWeights validation - invalid weights
func TestQueryWeights_Validate_Invalid(t *testing.T) {
	tests := []struct {
		name    string
		weights QueryWeights
		errMsg  string
	}{
		{
			name: "InitialQuery negative",
			weights: QueryWeights{
				InitialQuery: -0.1,
				ErrorContext: 0.5,
				ToolContext:  0.5,
			},
			errMsg: "initial_query must be between 0 and 1",
		},
		{
			name: "InitialQuery > 1",
			weights: QueryWeights{
				InitialQuery: 1.5,
				ErrorContext: 0.0,
				ToolContext:  0.0,
			},
			errMsg: "initial_query must be between 0 and 1",
		},
		{
			name: "ErrorContext negative",
			weights: QueryWeights{
				InitialQuery: 0.5,
				ErrorContext: -0.1,
				ToolContext:  0.5,
			},
			errMsg: "error_context must be between 0 and 1",
		},
		{
			name: "ErrorContext > 1",
			weights: QueryWeights{
				InitialQuery: 0.5,
				ErrorContext: 1.1,
				ToolContext:  0.0,
			},
			errMsg: "error_context must be between 0 and 1",
		},
		{
			name: "ToolContext negative",
			weights: QueryWeights{
				InitialQuery: 0.5,
				ErrorContext: 0.5,
				ToolContext:  -0.1,
			},
			errMsg: "tool_context must be between 0 and 1",
		},
		{
			name: "ToolContext > 1",
			weights: QueryWeights{
				InitialQuery: 0.0,
				ErrorContext: 0.0,
				ToolContext:  1.5,
			},
			errMsg: "tool_context must be between 0 and 1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.weights.Validate()
			if err == nil {
				t.Error("Validate() expected error, got nil")
				return
			}
			if !contains(err.Error(), tt.errMsg) {
				t.Errorf("Validate() error = %v, want error containing %q", err, tt.errMsg)
			}
		})
	}
}

// Test DefaultProgressiveContextConfig returns valid configuration
func TestDefaultProgressiveContextConfig(t *testing.T) {
	cfg := DefaultProgressiveContextConfig()

	// Verify all fields have expected defaults
	if !cfg.Enabled {
		t.Error("Enabled should be true by default")
	}
	if cfg.CacheTTL != 10 {
		t.Errorf("CacheTTL = %d, want 10", cfg.CacheTTL)
	}
	if cfg.MaxBullets != 50 {
		t.Errorf("MaxBullets = %d, want 50", cfg.MaxBullets)
	}
	if cfg.EvictionStrategy != "lru" {
		t.Errorf("EvictionStrategy = %q, want \"lru\"", cfg.EvictionStrategy)
	}
	if cfg.ErrorLookback != 5 {
		t.Errorf("ErrorLookback = %d, want 5", cfg.ErrorLookback)
	}
	if cfg.ToolChangeLookback != 3 {
		t.Errorf("ToolChangeLookback = %d, want 3", cfg.ToolChangeLookback)
	}
	if len(cfg.EnabledTriggers) != 4 {
		t.Errorf("EnabledTriggers length = %d, want 4", len(cfg.EnabledTriggers))
	}
	if cfg.QueryWeights.InitialQuery != 0.5 {
		t.Errorf("QueryWeights.InitialQuery = %f, want 0.5", cfg.QueryWeights.InitialQuery)
	}
	if cfg.QueryWeights.ErrorContext != 0.3 {
		t.Errorf("QueryWeights.ErrorContext = %f, want 0.3", cfg.QueryWeights.ErrorContext)
	}
	if cfg.QueryWeights.ToolContext != 0.2 {
		t.Errorf("QueryWeights.ToolContext = %f, want 0.2", cfg.QueryWeights.ToolContext)
	}
	if cfg.MaxRetrievalLatencyMs != 500 {
		t.Errorf("MaxRetrievalLatencyMs = %d, want 500", cfg.MaxRetrievalLatencyMs)
	}
	if cfg.MaxTrajectorySteps != 1000 {
		t.Errorf("MaxTrajectorySteps = %d, want 1000", cfg.MaxTrajectorySteps)
	}
	if !cfg.LogRetrievalDecisions {
		t.Error("LogRetrievalDecisions should be true by default")
	}
	if !cfg.LogCacheStats {
		t.Error("LogCacheStats should be true by default")
	}
	if !cfg.EmitACEEvents {
		t.Error("EmitACEEvents should be true by default")
	}

	// Verify default config passes validation
	err := cfg.Validate()
	if err != nil {
		t.Errorf("DefaultProgressiveContextConfig() validation failed: %v", err)
	}
}

// Test ProgressiveContextConfig validation - valid configurations
func TestProgressiveContextConfig_Validate_Valid(t *testing.T) {
	tests := []struct {
		name   string
		config ProgressiveContextConfig
	}{
		{
			name:   "default config",
			config: DefaultProgressiveContextConfig(),
		},
		{
			name: "minimal valid config",
			config: ProgressiveContextConfig{
				Enabled:            true,
				CacheTTL:           1,
				MaxBullets:         1,
				EvictionStrategy:   "lru",
				ErrorLookback:      1,
				ToolChangeLookback: 1,
				EnabledTriggers:    []string{"initial"},
				QueryWeights: QueryWeights{
					InitialQuery: 1.0,
					ErrorContext: 0.0,
					ToolContext:  0.0,
				},
				MaxRetrievalLatencyMs: 1,
				MaxTrajectorySteps:    1,
			},
		},
		{
			name: "all eviction strategies",
			config: ProgressiveContextConfig{
				Enabled:            true,
				CacheTTL:           10,
				MaxBullets:         50,
				EvictionStrategy:   "lfu",
				ErrorLookback:      5,
				ToolChangeLookback: 3,
				EnabledTriggers:    []string{"initial", "error"},
				QueryWeights: QueryWeights{
					InitialQuery: 0.5,
					ErrorContext: 0.5,
					ToolContext:  0.0,
				},
				MaxRetrievalLatencyMs: 500,
				MaxTrajectorySteps:    1000,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if err != nil {
				t.Errorf("Validate() unexpected error = %v", err)
			}
		})
	}
}

// Test ProgressiveContextConfig validation - invalid configurations
func TestProgressiveContextConfig_Validate_Invalid(t *testing.T) {
	tests := []struct {
		name   string
		modify func(*ProgressiveContextConfig)
		errMsg string
	}{
		{
			name: "CacheTTL zero",
			modify: func(c *ProgressiveContextConfig) {
				c.CacheTTL = 0
			},
			errMsg: "cache_ttl must be > 0",
		},
		{
			name: "CacheTTL negative",
			modify: func(c *ProgressiveContextConfig) {
				c.CacheTTL = -1
			},
			errMsg: "cache_ttl must be > 0",
		},
		{
			name: "MaxBullets zero",
			modify: func(c *ProgressiveContextConfig) {
				c.MaxBullets = 0
			},
			errMsg: "max_bullets must be > 0",
		},
		{
			name: "MaxBullets negative",
			modify: func(c *ProgressiveContextConfig) {
				c.MaxBullets = -5
			},
			errMsg: "max_bullets must be > 0",
		},
		{
			name: "invalid EvictionStrategy",
			modify: func(c *ProgressiveContextConfig) {
				c.EvictionStrategy = "invalid"
			},
			errMsg: "eviction_strategy must be one of",
		},
		{
			name: "ErrorLookback zero",
			modify: func(c *ProgressiveContextConfig) {
				c.ErrorLookback = 0
			},
			errMsg: "error_lookback must be > 0",
		},
		{
			name: "ToolChangeLookback zero",
			modify: func(c *ProgressiveContextConfig) {
				c.ToolChangeLookback = 0
			},
			errMsg: "tool_change_lookback must be > 0",
		},
		{
			name: "empty EnabledTriggers",
			modify: func(c *ProgressiveContextConfig) {
				c.EnabledTriggers = []string{}
			},
			errMsg: "enabled_triggers cannot be empty",
		},
		{
			name: "invalid trigger",
			modify: func(c *ProgressiveContextConfig) {
				c.EnabledTriggers = []string{"invalid"}
			},
			errMsg: "invalid trigger",
		},
		{
			name: "invalid QueryWeights",
			modify: func(c *ProgressiveContextConfig) {
				c.QueryWeights.InitialQuery = -0.1
			},
			errMsg: "initial_query must be between 0 and 1",
		},
		{
			name: "MaxRetrievalLatencyMs zero",
			modify: func(c *ProgressiveContextConfig) {
				c.MaxRetrievalLatencyMs = 0
			},
			errMsg: "max_retrieval_latency_ms must be > 0",
		},
		{
			name: "MaxTrajectorySteps zero",
			modify: func(c *ProgressiveContextConfig) {
				c.MaxTrajectorySteps = 0
			},
			errMsg: "max_trajectory_steps must be > 0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := DefaultProgressiveContextConfig()
			tt.modify(&cfg)

			err := cfg.Validate()
			if err == nil {
				t.Error("Validate() expected error, got nil")
				return
			}
			if !contains(err.Error(), tt.errMsg) {
				t.Errorf("Validate() error = %v, want error containing %q", err, tt.errMsg)
			}
		})
	}
}
