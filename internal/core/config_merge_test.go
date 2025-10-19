package core

import (
	"testing"
)

func TestConfig_MergeStringFields(t *testing.T) {
	base := &Config{
		Provider:    "base-provider",
		Model:       "base-model",
		WorkDir:     "/base/workdir",
		SandboxMode: "base-sandbox",
		PolicyFile:  "/base/policy",
		SessionDir:  "/base/sessions",
		LogLevel:    "info",
		LogFormat:   "text",
	}

	tests := []struct {
		name  string
		other *Config
		check func(*testing.T, *Config)
	}{
		{
			name: "merge all fields",
			other: &Config{
				Provider:    "new-provider",
				Model:       "new-model",
				WorkDir:     "/new/workdir",
				SandboxMode: "new-sandbox",
				PolicyFile:  "/new/policy",
				SessionDir:  "/new/sessions",
				LogLevel:    "debug",
				LogFormat:   "json",
			},
			check: func(t *testing.T, c *Config) {
				if c.Provider != "new-provider" {
					t.Errorf("Provider = %v, want new-provider", c.Provider)
				}
				if c.Model != "new-model" {
					t.Errorf("Model = %v, want new-model", c.Model)
				}
				if c.WorkDir != "/new/workdir" {
					t.Errorf("WorkDir = %v, want /new/workdir", c.WorkDir)
				}
				if c.SandboxMode != "new-sandbox" {
					t.Errorf("SandboxMode = %v, want new-sandbox", c.SandboxMode)
				}
				if c.PolicyFile != "/new/policy" {
					t.Errorf("PolicyFile = %v, want /new/policy", c.PolicyFile)
				}
				if c.SessionDir != "/new/sessions" {
					t.Errorf("SessionDir = %v, want /new/sessions", c.SessionDir)
				}
				if c.LogLevel != "debug" {
					t.Errorf("LogLevel = %v, want debug", c.LogLevel)
				}
				if c.LogFormat != "json" {
					t.Errorf("LogFormat = %v, want json", c.LogFormat)
				}
			},
		},
		{
			name: "merge only some fields",
			other: &Config{
				Provider: "new-provider",
				Model:    "new-model",
			},
			check: func(t *testing.T, c *Config) {
				if c.Provider != "new-provider" {
					t.Errorf("Provider = %v, want new-provider", c.Provider)
				}
				if c.Model != "new-model" {
					t.Errorf("Model = %v, want new-model", c.Model)
				}
				if c.WorkDir != "/base/workdir" {
					t.Errorf("WorkDir = %v, want /base/workdir", c.WorkDir)
				}
				if c.LogLevel != "info" {
					t.Errorf("LogLevel = %v, want info", c.LogLevel)
				}
			},
		},
		{
			name:  "merge with empty other",
			other: &Config{},
			check: func(t *testing.T, c *Config) {
				if c.Provider != "base-provider" {
					t.Errorf("Provider = %v, want base-provider", c.Provider)
				}
				if c.Model != "base-model" {
					t.Errorf("Model = %v, want base-model", c.Model)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &Config{
				Provider:    base.Provider,
				Model:       base.Model,
				WorkDir:     base.WorkDir,
				SandboxMode: base.SandboxMode,
				PolicyFile:  base.PolicyFile,
				SessionDir:  base.SessionDir,
				LogLevel:    base.LogLevel,
				LogFormat:   base.LogFormat,
			}

			config.mergeStringFields(tt.other)
			tt.check(t, config)
		})
	}
}

func TestConfig_MergeIntFields(t *testing.T) {
	base := &Config{
		MaxTurns:        10,
		Timeout:         30,
		MaxTokens:       1000,
		StreamBuffer:    256,
		HistoryLimit:    50,
		ApprovalTimeout: 60,
	}

	tests := []struct {
		name  string
		other *Config
		check func(*testing.T, *Config)
	}{
		{
			name: "merge all fields",
			other: &Config{
				MaxTurns:        20,
				Timeout:         60,
				MaxTokens:       2000,
				StreamBuffer:    512,
				HistoryLimit:    100,
				ApprovalTimeout: 120,
			},
			check: func(t *testing.T, c *Config) {
				if c.MaxTurns != 20 {
					t.Errorf("MaxTurns = %v, want 20", c.MaxTurns)
				}
				if c.Timeout != 60 {
					t.Errorf("Timeout = %v, want 60", c.Timeout)
				}
				if c.MaxTokens != 2000 {
					t.Errorf("MaxTokens = %v, want 2000", c.MaxTokens)
				}
				if c.StreamBuffer != 512 {
					t.Errorf("StreamBuffer = %v, want 512", c.StreamBuffer)
				}
				if c.HistoryLimit != 100 {
					t.Errorf("HistoryLimit = %v, want 100", c.HistoryLimit)
				}
				if c.ApprovalTimeout != 120 {
					t.Errorf("ApprovalTimeout = %v, want 120", c.ApprovalTimeout)
				}
			},
		},
		{
			name: "merge with zeros",
			other: &Config{
				MaxTurns:        0,
				Timeout:         0,
				MaxTokens:       0,
				StreamBuffer:    0,
				HistoryLimit:    0,
				ApprovalTimeout: 0,
			},
			check: func(t *testing.T, c *Config) {
				if c.MaxTurns != 10 {
					t.Errorf("MaxTurns = %v, want 10", c.MaxTurns)
				}
				if c.Timeout != 30 {
					t.Errorf("Timeout = %v, want 30", c.Timeout)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &Config{
				MaxTurns:        base.MaxTurns,
				Timeout:         base.Timeout,
				MaxTokens:       base.MaxTokens,
				StreamBuffer:    base.StreamBuffer,
				HistoryLimit:    base.HistoryLimit,
				ApprovalTimeout: base.ApprovalTimeout,
			}

			config.mergeIntFields(tt.other)
			tt.check(t, config)
		})
	}
}

func TestConfig_MergeBoolFields(t *testing.T) {
	tests := []struct {
		name  string
		base  *Config
		other *Config
		check func(*testing.T, *Config)
	}{
		{
			name:  "enable MCP",
			base:  &Config{EnableMCP: false},
			other: &Config{EnableMCP: true},
			check: func(t *testing.T, c *Config) {
				if !c.EnableMCP {
					t.Error("Expected EnableMCP to be true")
				}
			},
		},
		{
			name: "enable MCP via MCPServers",
			base: &Config{EnableMCP: false},
			other: &Config{
				EnableMCP: false,
				MCPServers: []MCPServerConfig{
					{Name: "test", Command: "test"},
				},
			},
			check: func(t *testing.T, c *Config) {
				if c.EnableMCP {
					t.Error("Expected EnableMCP to remain false")
				}
			},
		},
		{
			name:  "toggle EnableGit",
			base:  &Config{EnableGit: false},
			other: &Config{EnableGit: true},
			check: func(t *testing.T, c *Config) {
				if !c.EnableGit {
					t.Error("Expected EnableGit to be true")
				}
			},
		},
		{
			name:  "toggle EnableShell",
			base:  &Config{EnableShell: false},
			other: &Config{EnableShell: true},
			check: func(t *testing.T, c *Config) {
				if !c.EnableShell {
					t.Error("Expected EnableShell to be true")
				}
			},
		},
		{
			name:  "toggle CacheCommands",
			base:  &Config{CacheCommands: false},
			other: &Config{CacheCommands: true},
			check: func(t *testing.T, c *Config) {
				if !c.CacheCommands {
					t.Error("Expected CacheCommands to be true")
				}
			},
		},
		{
			name:  "toggle Debug",
			base:  &Config{Debug: false},
			other: &Config{Debug: true},
			check: func(t *testing.T, c *Config) {
				if !c.Debug {
					t.Error("Expected Debug to be true")
				}
			},
		},
		{
			name:  "same values",
			base:  &Config{EnableGit: true, EnableShell: true},
			other: &Config{EnableGit: true, EnableShell: true},
			check: func(t *testing.T, c *Config) {
				if !c.EnableGit || !c.EnableShell {
					t.Error("Expected values to remain true")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &Config{
				EnableMCP:     tt.base.EnableMCP,
				EnableGit:     tt.base.EnableGit,
				EnableShell:   tt.base.EnableShell,
				CacheCommands: tt.base.CacheCommands,
				Debug:         tt.base.Debug,
			}

			config.mergeBoolFields(tt.base, tt.other)
			tt.check(t, config)
		})
	}
}

func TestConfig_ValidateExtended(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{
			name: "valid config",
			config: &Config{
				Provider:        "ollama",
				Model:           "llama2",
				MaxTurns:        10,
				Timeout:         30,
				MaxTokens:       1000,
				StreamBuffer:    100,
				ApprovalTimeout: 60,
				SandboxMode:     "full-access",
				CycleDetection: CycleDetectionConfig{
					Enabled:          true,
					WindowSize:       10,
					SimilarityThresh: 0.8,
					ToolRepeatLimit:  5,
					ErrorRepeatLimit: 3,
				},
			},
			wantErr: false,
		},
		{
			name: "missing provider",
			config: &Config{
				Model:           "llama2",
				MaxTurns:        10,
				Timeout:         30,
				MaxTokens:       1000,
				ApprovalTimeout: 60,
				SandboxMode:     "full-access",
				CycleDetection:  CycleDetectionConfig{Enabled: true, WindowSize: 10, SimilarityThresh: 0.8},
			},
			wantErr: true,
		},
		{
			name: "missing model",
			config: &Config{
				Provider:        "ollama",
				MaxTurns:        10,
				Timeout:         30,
				MaxTokens:       1000,
				ApprovalTimeout: 60,
				SandboxMode:     "full-access",
				CycleDetection:  CycleDetectionConfig{Enabled: true, WindowSize: 10, SimilarityThresh: 0.8},
			},
			wantErr: true,
		},
		{
			name: "zero max turns",
			config: &Config{
				Provider:        "ollama",
				Model:           "llama2",
				MaxTurns:        0,
				Timeout:         30,
				MaxTokens:       1000,
				ApprovalTimeout: 60,
				SandboxMode:     "full-access",
				CycleDetection:  CycleDetectionConfig{Enabled: true, WindowSize: 10, SimilarityThresh: 0.8},
			},
			wantErr: true,
		},
		{
			name: "zero timeout",
			config: &Config{
				Provider:        "ollama",
				Model:           "llama2",
				MaxTurns:        10,
				Timeout:         0,
				MaxTokens:       1000,
				ApprovalTimeout: 60,
				SandboxMode:     "full-access",
				CycleDetection:  CycleDetectionConfig{Enabled: true, WindowSize: 10, SimilarityThresh: 0.8},
			},
			wantErr: true,
		},
		{
			name: "zero max tokens",
			config: &Config{
				Provider:        "ollama",
				Model:           "llama2",
				MaxTurns:        10,
				Timeout:         30,
				MaxTokens:       0,
				ApprovalTimeout: 60,
				SandboxMode:     "full-access",
				CycleDetection:  CycleDetectionConfig{Enabled: true, WindowSize: 10, SimilarityThresh: 0.8},
			},
			wantErr: true,
		},
		{
			name: "invalid sandbox mode",
			config: &Config{
				Provider:        "ollama",
				Model:           "llama2",
				MaxTurns:        10,
				Timeout:         30,
				MaxTokens:       1000,
				ApprovalTimeout: 60,
				SandboxMode:     "invalid-mode",
				CycleDetection:  CycleDetectionConfig{Enabled: true, WindowSize: 10, SimilarityThresh: 0.8},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Config.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestCycleDetectionConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  *CycleDetectionConfig
		wantErr bool
	}{
		{
			name: "valid config",
			config: &CycleDetectionConfig{
				Enabled:          true,
				WindowSize:       10,
				SimilarityThresh: 0.8,
				ToolRepeatLimit:  5,
				ErrorRepeatLimit: 3,
			},
			wantErr: false,
		},
		{
			name: "invalid window size zero",
			config: &CycleDetectionConfig{
				Enabled:    true,
				WindowSize: 0,
			},
			wantErr: true,
		},
		{
			name: "invalid window size negative",
			config: &CycleDetectionConfig{
				Enabled:    true,
				WindowSize: -1,
			},
			wantErr: true,
		},
		{
			name: "invalid similarity threshold negative",
			config: &CycleDetectionConfig{
				Enabled:          true,
				WindowSize:       10,
				SimilarityThresh: -0.1,
			},
			wantErr: true,
		},
		{
			name: "invalid similarity threshold too high",
			config: &CycleDetectionConfig{
				Enabled:          true,
				WindowSize:       10,
				SimilarityThresh: 1.1,
			},
			wantErr: true,
		},
		{
			name: "invalid tool repeat limit negative",
			config: &CycleDetectionConfig{
				Enabled:         true,
				WindowSize:      10,
				ToolRepeatLimit: -1,
			},
			wantErr: true,
		},
		{
			name: "invalid error repeat limit negative",
			config: &CycleDetectionConfig{
				Enabled:          true,
				WindowSize:       10,
				ErrorRepeatLimit: -1,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("CycleDetectionConfig.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
