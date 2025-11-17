package config

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestConfigV2_Validate_MinimalValid tests that a minimal valid v2 config passes validation.
// This is the first step in the new v2.0 config structure.
func TestConfigV2_Validate_MinimalValid(t *testing.T) {
	cfg := &ConfigV2{
		Version: "2.0",
		LLM: LLMConfigV2{
			Provider:    "ollama",
			Model:       "qwen",
			Temperature: 0.7,
			MaxTokens:   4096,
			Timeout:     30 * time.Second,
		},
		Agent: AgentConfigV2{
			MaxTurns: 10,
			Timeout:  60 * time.Second,
			WorkDir:  "/tmp",
		},
	}

	err := cfg.Validate()
	require.NoError(t, err, "minimal valid config should pass validation")
}

// TestConfigV2_Validate_LLMProviderRequired tests that validation fails when LLM.Provider is empty.
// Kills mutant: removing the provider check would make this test fail.
func TestConfigV2_Validate_LLMProviderRequired(t *testing.T) {
	cfg := &ConfigV2{
		Version: "2.0",
		LLM: LLMConfigV2{
			Provider: "", // Empty provider should fail
			Model:    "qwen",
		},
	}

	err := cfg.Validate()
	require.Error(t, err, "empty LLM provider should fail validation")
	assert.Contains(t, err.Error(), "provider", "error should mention provider field")
	assert.Contains(t, err.Error(), "required", "error should indicate field is required")
}

// TestConfigV2_Validate_LLMModelRequired tests that validation fails when LLM.Model is empty.
// Kills mutant: removing the model check would make this test fail.
func TestConfigV2_Validate_LLMModelRequired(t *testing.T) {
	cfg := &ConfigV2{
		Version: "2.0",
		LLM: LLMConfigV2{
			Provider: "ollama",
			Model:    "", // Empty model should fail
		},
	}

	err := cfg.Validate()
	require.Error(t, err, "empty LLM model should fail validation")
	assert.Contains(t, err.Error(), "model", "error should mention model field")
	assert.Contains(t, err.Error(), "required", "error should indicate field is required")
}

// TestConfigV2_Validate_LLMFieldRanges tests validation of numeric field ranges.
// Kills mutants: removing range checks would make these tests fail.
func TestConfigV2_Validate_LLMFieldRanges(t *testing.T) {
	tests := []struct {
		name    string
		cfg     ConfigV2
		wantErr string
	}{
		{
			name: "temperature too low",
			cfg: ConfigV2{
				Version: "2.0",
				LLM: LLMConfigV2{
					Provider:    "ollama",
					Model:       "qwen",
					Temperature: -0.1,
					MaxTokens:   4096,
					Timeout:     30 * time.Second,
				},
			},
			wantErr: "temperature",
		},
		{
			name: "temperature too high",
			cfg: ConfigV2{
				Version: "2.0",
				LLM: LLMConfigV2{
					Provider:    "ollama",
					Model:       "qwen",
					Temperature: 2.1,
					MaxTokens:   4096,
					Timeout:     30 * time.Second,
				},
			},
			wantErr: "temperature",
		},
		{
			name: "max_tokens zero",
			cfg: ConfigV2{
				Version: "2.0",
				LLM: LLMConfigV2{
					Provider:    "ollama",
					Model:       "qwen",
					Temperature: 0.7,
					MaxTokens:   0,
					Timeout:     30 * time.Second,
				},
			},
			wantErr: "max_tokens",
		},
		{
			name: "max_tokens negative",
			cfg: ConfigV2{
				Version: "2.0",
				LLM: LLMConfigV2{
					Provider:    "ollama",
					Model:       "qwen",
					Temperature: 0.7,
					MaxTokens:   -100,
					Timeout:     30 * time.Second,
				},
			},
			wantErr: "max_tokens",
		},
		{
			name: "timeout zero",
			cfg: ConfigV2{
				Version: "2.0",
				LLM: LLMConfigV2{
					Provider:    "ollama",
					Model:       "qwen",
					Temperature: 0.7,
					MaxTokens:   4096,
					Timeout:     0,
				},
			},
			wantErr: "timeout",
		},
		{
			name: "timeout negative",
			cfg: ConfigV2{
				Version: "2.0",
				LLM: LLMConfigV2{
					Provider:    "ollama",
					Model:       "qwen",
					Temperature: 0.7,
					MaxTokens:   4096,
					Timeout:     -5 * time.Second,
				},
			},
			wantErr: "timeout",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			require.Error(t, err, "validation should fail for %s", tt.name)
			assert.Contains(t, err.Error(), tt.wantErr, "error should mention %s", tt.wantErr)
		})
	}
}

// TestConfigV2_Validate_LLMValidRanges tests that valid values pass validation.
func TestConfigV2_Validate_LLMValidRanges(t *testing.T) {
	cfg := &ConfigV2{
		Version: "2.0",
		LLM: LLMConfigV2{
			Provider:    "ollama",
			Model:       "qwen",
			Temperature: 0.7,
			MaxTokens:   4096,
			Timeout:     5 * time.Minute,
		},
		Agent: AgentConfigV2{
			MaxTurns: 10,
			Timeout:  60 * time.Second,
			WorkDir:  "/tmp",
		},
	}

	err := cfg.Validate()
	require.NoError(t, err, "valid config should pass validation")
}

// TestConfigV2_Validate_AgentMaxTurnsRequired tests that MaxTurns must be positive.
// Kills mutant: removing the MaxTurns check would make this test fail.
func TestConfigV2_Validate_AgentMaxTurnsRequired(t *testing.T) {
	agent := validAgentConfig()
	agent.MaxTurns = 0 // Invalid

	cfg := &ConfigV2{
		Version: "2.0",
		LLM:     validLLMConfig(),
		Agent:   agent,
	}

	err := cfg.Validate()
	require.Error(t, err, "zero MaxTurns should fail validation")
	assert.Contains(t, err.Error(), "max_turns", "error should mention max_turns field")
}

// TestConfigV2_Validate_AgentTimeoutRequired tests that Timeout must be positive.
// Kills mutant: removing the Timeout check would make this test fail.
func TestConfigV2_Validate_AgentTimeoutRequired(t *testing.T) {
	agent := validAgentConfig()
	agent.Timeout = 0 // Invalid

	cfg := &ConfigV2{
		Version: "2.0",
		LLM:     validLLMConfig(),
		Agent:   agent,
	}

	err := cfg.Validate()
	require.Error(t, err, "zero Timeout should fail validation")
	assert.Contains(t, err.Error(), "timeout", "error should mention timeout field")
}

// TestConfigV2_Validate_AgentWorkDirRequired tests that WorkDir is required.
// Kills mutant: removing the WorkDir check would make this test fail.
func TestConfigV2_Validate_AgentWorkDirRequired(t *testing.T) {
	agent := validAgentConfig()
	agent.WorkDir = "" // Invalid

	cfg := &ConfigV2{
		Version: "2.0",
		LLM:     validLLMConfig(),
		Agent:   agent,
	}

	err := cfg.Validate()
	require.Error(t, err, "empty WorkDir should fail validation")
	assert.Contains(t, err.Error(), "work_dir", "error should mention work_dir field")
}

// validLLMConfig returns a valid LLM configuration for testing other sections.
func validLLMConfig() LLMConfigV2 {
	return LLMConfigV2{
		Provider:    "ollama",
		Model:       "qwen",
		Temperature: 0.7,
		MaxTokens:   4096,
		Timeout:     5 * time.Minute,
	}
}

// validAgentConfig returns a valid Agent configuration for testing other sections.
func validAgentConfig() AgentConfigV2 {
	return AgentConfigV2{
		MaxTurns: 10,
		Timeout:  60 * time.Second,
		WorkDir:  "/tmp",
	}
}

// validACEConfig returns a valid ACE configuration for testing other sections.
func validACEConfig() ACEConfigV2 {
	return ACEConfigV2{
		Enabled:        true,
		PlaybookPath:   "~/.spin/ace/playbooks/default.json",
		TrajectoryPath: "~/.spin/ace/trajectories/",
		TopK:           5,
		MinScore:       0.3,
	}
}

// validSecurityConfig returns a valid Security configuration for testing other sections.
func validSecurityConfig() SecurityConfigV2 {
	return SecurityConfigV2{
		SandboxMode:                "none",
		PolicyFile:                 "",
		AllowedCommands:            []string{},
		ApprovalPersistenceEnabled: true,
	}
}

// validProtocolConfig returns a valid Protocol configuration for testing other sections.
func validProtocolConfig() ProtocolConfigV2 {
	return ProtocolConfigV2{
		EnableMCP:    false,
		MCPServers:   []MCPServerConfigV2{},
		EnableGit:    true,
		EnableShell:  false,
		ShellTimeout: 0,
	}
}

// TestConfigV2_Validate_ACEPlaybookPathRequired tests that PlaybookPath is required when ACE is enabled.
// Kills mutant: removing the PlaybookPath check would make this test fail.
func TestConfigV2_Validate_ACEPlaybookPathRequired(t *testing.T) {
	ace := validACEConfig()
	ace.PlaybookPath = "" // Invalid

	cfg := &ConfigV2{
		Version: "2.0",
		LLM:     validLLMConfig(),
		Agent:   validAgentConfig(),
		ACE:     ace,
	}

	err := cfg.Validate()
	require.Error(t, err, "empty PlaybookPath should fail validation when ACE enabled")
	assert.Contains(t, err.Error(), "playbook_path", "error should mention playbook_path field")
}

// TestConfigV2_Validate_ACETrajectoryPathRequired tests that TrajectoryPath is required when ACE is enabled.
// Kills mutant: removing the TrajectoryPath check would make this test fail.
func TestConfigV2_Validate_ACETrajectoryPathRequired(t *testing.T) {
	ace := validACEConfig()
	ace.TrajectoryPath = "" // Invalid

	cfg := &ConfigV2{
		Version: "2.0",
		LLM:     validLLMConfig(),
		Agent:   validAgentConfig(),
		ACE:     ace,
	}

	err := cfg.Validate()
	require.Error(t, err, "empty TrajectoryPath should fail validation when ACE enabled")
	assert.Contains(t, err.Error(), "trajectory_path", "error should mention trajectory_path field")
}

// TestConfigV2_Validate_ACETopKPositive tests that TopK must be positive when ACE is enabled.
// Kills mutant: removing the TopK check would make this test fail.
func TestConfigV2_Validate_ACETopKPositive(t *testing.T) {
	ace := validACEConfig()
	ace.TopK = 0 // Invalid

	cfg := &ConfigV2{
		Version: "2.0",
		LLM:     validLLMConfig(),
		Agent:   validAgentConfig(),
		ACE:     ace,
	}

	err := cfg.Validate()
	require.Error(t, err, "zero TopK should fail validation when ACE enabled")
	assert.Contains(t, err.Error(), "top_k", "error should mention top_k field")
}

// TestConfigV2_Validate_ACEMinScoreRange tests that MinScore must be between 0 and 1.
// Kills mutant: removing the MinScore check would make this test fail.
func TestConfigV2_Validate_ACEMinScoreRange(t *testing.T) {
	tests := []struct {
		name     string
		minScore float64
	}{
		{"negative", -0.1},
		{"too high", 1.1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ace := validACEConfig()
			ace.MinScore = tt.minScore

			cfg := &ConfigV2{
				Version: "2.0",
				LLM:     validLLMConfig(),
				Agent:   validAgentConfig(),
				ACE:     ace,
			}

			err := cfg.Validate()
			require.Error(t, err, "MinScore %s should fail validation", tt.name)
			assert.Contains(t, err.Error(), "min_score", "error should mention min_score field")
		})
	}
}

// TestConfigV2_Validate_ACEDisabled tests that validation passes when ACE is disabled.
// Kills mutant: removing the ACE.Enabled check would make this test fail.
func TestConfigV2_Validate_ACEDisabled(t *testing.T) {
	cfg := &ConfigV2{
		Version: "2.0",
		LLM:     validLLMConfig(),
		Agent:   validAgentConfig(),
		ACE: ACEConfigV2{
			Enabled: false,
			// Empty paths should be OK when disabled
			PlaybookPath:   "",
			TrajectoryPath: "",
			TopK:           0,
			MinScore:       0,
		},
	}

	err := cfg.Validate()
	require.NoError(t, err, "ACE disabled config should pass validation")
}

// TestConfigV2_Validate_SecuritySandboxModeValid tests that only valid sandbox modes are accepted.
// Kills mutant: removing the SandboxMode validation would make this test fail.
func TestConfigV2_Validate_SecuritySandboxModeValid(t *testing.T) {
	validModes := []string{"", "none", "docker", "firejail"}
	for _, mode := range validModes {
		t.Run(mode, func(t *testing.T) {
			cfg := &ConfigV2{
				Version:  "2.0",
				LLM:      validLLMConfig(),
				Agent:    validAgentConfig(),
				ACE:      validACEConfig(),
				Security: SecurityConfigV2{SandboxMode: mode},
				Protocol: validProtocolConfig(),
			}

			err := cfg.Validate()
			require.NoError(t, err, "sandbox mode %q should be valid", mode)
		})
	}
}

// TestConfigV2_Validate_SecuritySandboxModeInvalid tests that invalid sandbox modes are rejected.
// Kills mutant: removing the SandboxMode validation would make this test fail.
func TestConfigV2_Validate_SecuritySandboxModeInvalid(t *testing.T) {
	cfg := &ConfigV2{
		Version:  "2.0",
		LLM:      validLLMConfig(),
		Agent:    validAgentConfig(),
		ACE:      validACEConfig(),
		Security: SecurityConfigV2{SandboxMode: "invalid"},
		Protocol: validProtocolConfig(),
	}

	err := cfg.Validate()
	require.Error(t, err, "invalid sandbox mode should fail validation")
	assert.Contains(t, err.Error(), "sandbox_mode", "error should mention sandbox_mode field")
}

// TestConfigV2_Validate_ProtocolShellTimeoutPositive tests that ShellTimeout must be positive when shell is enabled.
// Kills mutant: removing the ShellTimeout check would make this test fail.
func TestConfigV2_Validate_ProtocolShellTimeoutPositive(t *testing.T) {
	protocol := validProtocolConfig()
	protocol.EnableShell = true
	protocol.ShellTimeout = 0 // Invalid

	cfg := &ConfigV2{
		Version:  "2.0",
		LLM:      validLLMConfig(),
		Agent:    validAgentConfig(),
		ACE:      validACEConfig(),
		Security: validSecurityConfig(),
		Protocol: protocol,
	}

	err := cfg.Validate()
	require.Error(t, err, "zero ShellTimeout should fail validation when shell enabled")
	assert.Contains(t, err.Error(), "shell_timeout", "error should mention shell_timeout field")
}

// TestConfigV2_Validate_ProtocolMCPServersValid tests that MCP server configs are validated when MCP is enabled.
// Kills mutant: removing the MCP server validation would make this test fail.
func TestConfigV2_Validate_ProtocolMCPServersValid(t *testing.T) {
	protocol := validProtocolConfig()
	protocol.EnableMCP = true
	protocol.MCPServers = []MCPServerConfigV2{
		{Name: "", Command: "/usr/bin/mcp"}, // Invalid: empty name
	}

	cfg := &ConfigV2{
		Version:  "2.0",
		LLM:      validLLMConfig(),
		Agent:    validAgentConfig(),
		ACE:      validACEConfig(),
		Security: validSecurityConfig(),
		Protocol: protocol,
	}

	err := cfg.Validate()
	require.Error(t, err, "MCP server with empty name should fail validation")
	assert.Contains(t, err.Error(), "mcp", "error should mention mcp")
}

// TestDefaultConfigV2 tests that the default configuration is valid.
// Kills mutant: changing defaults to invalid values would make this test fail.
func TestDefaultConfigV2(t *testing.T) {
	cfg := DefaultConfigV2()

	// Should be valid by default
	err := cfg.Validate()
	require.NoError(t, err, "default config should pass validation")

	// Check some expected defaults
	assert.Equal(t, "2.0", cfg.Version, "version should be 2.0")
	assert.Equal(t, "ollama", cfg.LLM.Provider, "default provider should be ollama")
	assert.Equal(t, 0.7, cfg.LLM.Temperature, "default temperature should be 0.7")
	assert.Equal(t, 50, cfg.Agent.MaxTurns, "default max_turns should be 50")
	assert.True(t, cfg.ACE.Enabled, "ACE should be enabled by default")
}

// TestConfigV2_CrossSectionValidation_ACEPlaybookRequired tests that ACE playbook is required when enabled.
// Kills mutant: removing cross-section validation would make this test fail.
func TestConfigV2_CrossSectionValidation_ACEPlaybookRequired(t *testing.T) {
	cfg := &ConfigV2{
		Version: "2.0",
		LLM:     validLLMConfig(),
		Agent:   validAgentConfig(),
		ACE: ACEConfigV2{
			Enabled:      true,
			PlaybookPath: "", // Invalid - required when enabled
		},
		Security: validSecurityConfig(),
		Protocol: validProtocolConfig(),
	}

	err := cfg.Validate()
	require.Error(t, err, "validation should fail when ACE enabled but playbook_path empty")
	assert.Contains(t, err.Error(), "playbook_path", "error should mention playbook_path")
}

// TestConfigV2_CrossSectionValidation_ACETrajectoryRequired tests that ACE trajectory is required when enabled.
// Kills mutant: removing cross-section validation would make this test fail.
func TestConfigV2_CrossSectionValidation_ACETrajectoryRequired(t *testing.T) {
	cfg := &ConfigV2{
		Version: "2.0",
		LLM:     validLLMConfig(),
		Agent:   validAgentConfig(),
		ACE: ACEConfigV2{
			Enabled:        true,
			PlaybookPath:   "/path/to/playbook.json",
			TrajectoryPath: "", // Invalid - required when enabled
		},
		Security: validSecurityConfig(),
		Protocol: validProtocolConfig(),
	}

	err := cfg.Validate()
	require.Error(t, err, "validation should fail when ACE enabled but trajectory_path empty")
	assert.Contains(t, err.Error(), "trajectory_path", "error should mention trajectory_path")
}

// TestConfigV2_CrossSectionValidation_ShellTimeoutRequired tests that shell timeout is required when shell enabled.
// Kills mutant: removing cross-section validation would make this test fail.
func TestConfigV2_CrossSectionValidation_ShellTimeoutRequired(t *testing.T) {
	cfg := &ConfigV2{
		Version:  "2.0",
		LLM:      validLLMConfig(),
		Agent:    validAgentConfig(),
		ACE:      validACEConfig(),
		Security: validSecurityConfig(),
		Protocol: ProtocolConfigV2{
			EnableShell:  true,
			ShellTimeout: 0, // Invalid - required when enabled
		},
	}

	err := cfg.Validate()
	require.Error(t, err, "validation should fail when shell enabled but timeout is zero")
	assert.Contains(t, err.Error(), "shell_timeout", "error should mention shell_timeout")
}

// TestConfigV2_Validation_AllErrors tests that validation returns ALL errors, not just the first.
// Kills mutant: fail-fast validation would make this test fail.
func TestConfigV2_Validation_AllErrors(t *testing.T) {
	cfg := &ConfigV2{
		Version: "2.0",
		LLM: LLMConfigV2{
			Provider:    "",  // Error 1
			Model:       "",  // Error 2
			Temperature: 3.0, // Error 3
			MaxTokens:   -10, // Error 4
			Timeout:     0,   // Error 5
		},
		Agent:    validAgentConfig(),
		ACE:      validACEConfig(),
		Security: validSecurityConfig(),
		Protocol: validProtocolConfig(),
	}

	err := cfg.Validate()
	require.Error(t, err, "validation should fail with multiple errors")

	// Check that error message contains multiple issues
	errMsg := err.Error()
	assert.Contains(t, errMsg, "provider", "should mention provider error")
	assert.Contains(t, errMsg, "model", "should mention model error")
	// Note: Current implementation is fail-fast, this test documents desired behavior
}
