package config

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// TestV2_Validate_MinimalValid tests that a minimal valid v2 config passes validation.
// This is the first step in the new v2.0 config structure.
func TestV2_Validate_MinimalValid(t *testing.T) {
	t.Parallel()

	cfg := &V2{
		Version: "2.0",
		LLM: LLMV2{
			Provider:    "ollama",
			Model:       "qwen",
			Temperature: 0.7,
			MaxTokens:   4096,
			Timeout:     30 * time.Second,
		},
		Agent: AgentV2{
			MaxTurns: 10,
			Timeout:  60 * time.Second,
			WorkDir:  "/tmp",
		},
	}

	err := cfg.Validate()
	require.NoError(t, err, "minimal valid config should pass validation")
}

// TestV2_Validate_LLMProviderRequired tests that validation fails when LLM.Provider is empty.
// Kills mutant: removing the provider check would make this test fail.
func TestV2_Validate_LLMProviderRequired(t *testing.T) {
	t.Parallel()

	cfg := &V2{
		Version: "2.0",
		LLM: LLMV2{
			Provider: "", // Empty provider should fail.
			Model:    "qwen",
		},
	}

	err := cfg.Validate()
	require.Error(t, err, "empty LLM provider should fail validation")
	assert.Contains(t, err.Error(), "provider", "error should mention provider field")
	assert.Contains(t, err.Error(), "required", "error should indicate field is required")
}

// TestV2_Validate_LLMModelRequired tests that validation fails when LLM.Model is empty.
// Kills mutant: removing the model check would make this test fail.
func TestV2_Validate_LLMModelRequired(t *testing.T) {
	t.Parallel()

	cfg := &V2{
		Version: "2.0",
		LLM: LLMV2{
			Provider: "ollama",
			Model:    "", // Empty model should fail.
		},
	}

	err := cfg.Validate()
	require.Error(t, err, "empty LLM model should fail validation")
	assert.Contains(t, err.Error(), "model", "error should mention model field")
	assert.Contains(t, err.Error(), "required", "error should indicate field is required")
}

// newTestLLMConfig creates a base valid V2 config with the given LLM overrides.
func newTestLLMConfig(temperature float64, maxTokens int, timeout time.Duration) V2 {
	return V2{
		Version: "2.0",
		LLM: LLMV2{
			Provider:    "ollama",
			Model:       "qwen",
			Temperature: temperature,
			MaxTokens:   maxTokens,
			Timeout:     timeout,
		},
	}
}

// TestV2_Validate_LLMFieldRanges tests validation of numeric field ranges.
// Kills mutants: removing range checks would make these tests fail.
func TestV2_Validate_LLMFieldRanges(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		cfg     V2
		wantErr string
	}{
		{"temperature too low", newTestLLMConfig(-0.1, 4096, 30*time.Second), "temperature"},
		{"temperature too high", newTestLLMConfig(2.1, 4096, 30*time.Second), "temperature"},
		{"max_tokens zero", newTestLLMConfig(0.7, 0, 30*time.Second), "max_tokens"},
		{"max_tokens negative", newTestLLMConfig(0.7, -100, 30*time.Second), "max_tokens"},
		{"timeout zero", newTestLLMConfig(0.7, 4096, 0), "timeout"},
		{"timeout negative", newTestLLMConfig(0.7, 4096, -5*time.Second), "timeout"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := tt.cfg.Validate()
			require.Error(t, err, "validation should fail for %s", tt.name)
			assert.Contains(t, err.Error(), tt.wantErr, "error should mention %s", tt.wantErr)
		})
	}
}

// TestV2_Validate_LLMValidRanges tests that valid values pass validation.
func TestV2_Validate_LLMValidRanges(t *testing.T) {
	t.Parallel()

	cfg := &V2{
		Version: "2.0",
		LLM: LLMV2{
			Provider:    "ollama",
			Model:       "qwen",
			Temperature: 0.7,
			MaxTokens:   4096,
			Timeout:     5 * time.Minute,
		},
		Agent: AgentV2{
			MaxTurns: 10,
			Timeout:  60 * time.Second,
			WorkDir:  "/tmp",
		},
	}

	err := cfg.Validate()
	require.NoError(t, err, "valid config should pass validation")
}

// TestV2_Validate_AgentMaxTurnsRequired tests that MaxTurns must be positive.
// Kills mutant: removing the MaxTurns check would make this test fail.
func TestV2_Validate_AgentMaxTurnsRequired(t *testing.T) {
	t.Parallel()

	agent := validAgentConfig()
	agent.MaxTurns = 0 // Invalid.

	cfg := &V2{
		Version: "2.0",
		LLM:     validLLMConfig(),
		Agent:   agent,
	}

	err := cfg.Validate()
	require.Error(t, err, "zero MaxTurns should fail validation")
	assert.Contains(t, err.Error(), "max_turns", "error should mention max_turns field")
}

// TestV2_Validate_AgentTimeoutRequired tests that Timeout must be positive.
// Kills mutant: removing the Timeout check would make this test fail.
func TestV2_Validate_AgentTimeoutRequired(t *testing.T) {
	t.Parallel()

	agent := validAgentConfig()
	agent.Timeout = 0 // Invalid.

	cfg := &V2{
		Version: "2.0",
		LLM:     validLLMConfig(),
		Agent:   agent,
	}

	err := cfg.Validate()
	require.Error(t, err, "zero Timeout should fail validation")
	assert.Contains(t, err.Error(), "timeout", "error should mention timeout field")
}

// TestV2_Validate_AgentWorkDirRequired tests that WorkDir is required.
// Kills mutant: removing the WorkDir check would make this test fail.
func TestV2_Validate_AgentWorkDirRequired(t *testing.T) {
	t.Parallel()

	agent := validAgentConfig()
	agent.WorkDir = "" // Invalid.

	cfg := &V2{
		Version: "2.0",
		LLM:     validLLMConfig(),
		Agent:   agent,
	}

	err := cfg.Validate()
	require.Error(t, err, "empty WorkDir should fail validation")
	assert.Contains(t, err.Error(), "work_dir", "error should mention work_dir field")
}

// validLLMConfig returns a valid LLM configuration for testing other sections.
func validLLMConfig() LLMV2 {
	return LLMV2{
		Provider:    "ollama",
		Model:       "qwen",
		Temperature: 0.7,
		MaxTokens:   4096,
		Timeout:     5 * time.Minute,
	}
}

// validAgentConfig returns a valid Agent configuration for testing other sections.
func validAgentConfig() AgentV2 {
	return AgentV2{
		MaxTurns: 10,
		Timeout:  60 * time.Second,
		WorkDir:  "/tmp",
	}
}

// validACEConfig returns a valid ACE configuration for testing other sections.
func validACEConfig() ACEV2 {
	return ACEV2{
		Enabled:        true,
		PlaybookPath:   "~/.spin/ace/playbooks/default.json",
		TrajectoryPath: "~/.spin/ace/trajectories/",
		TopK:           5,
		MinScore:       0.3,
	}
}

// validSecurityConfig returns a valid Security configuration for testing other sections.
func validSecurityConfig() SecurityV2 {
	return SecurityV2{
		SandboxMode:                "none",
		PolicyFile:                 "",
		AllowedCommands:            []string{},
		ApprovalPersistenceEnabled: true,
	}
}

// validProtocolConfig returns a valid Protocol configuration for testing other sections.
func validProtocolConfig() ProtocolV2 {
	return ProtocolV2{
		EnableMCP:    false,
		MCPServers:   []MCPServerConfigV2{},
		EnableGit:    true,
		EnableShell:  false,
		ShellTimeout: 0,
	}
}

// TestV2_Validate_ACEPlaybookPathRequired tests that PlaybookPath is required when ACE is enabled.
// Kills mutant: removing the PlaybookPath check would make this test fail.
func TestV2_Validate_ACEPlaybookPathRequired(t *testing.T) {
	t.Parallel()

	ace := validACEConfig()
	ace.PlaybookPath = "" // Invalid.

	cfg := &V2{
		Version: "2.0",
		LLM:     validLLMConfig(),
		Agent:   validAgentConfig(),
		ACE:     ace,
	}

	err := cfg.Validate()
	require.Error(t, err, "empty PlaybookPath should fail validation when ACE enabled")
	assert.Contains(t, err.Error(), "playbook_path", "error should mention playbook_path field")
}

// TestV2_Validate_ACETrajectoryPathRequired tests that TrajectoryPath is required when ACE is enabled.
// Kills mutant: removing the TrajectoryPath check would make this test fail.
func TestV2_Validate_ACETrajectoryPathRequired(t *testing.T) {
	t.Parallel()

	ace := validACEConfig()
	ace.TrajectoryPath = "" // Invalid.

	cfg := &V2{
		Version: "2.0",
		LLM:     validLLMConfig(),
		Agent:   validAgentConfig(),
		ACE:     ace,
	}

	err := cfg.Validate()
	require.Error(t, err, "empty TrajectoryPath should fail validation when ACE enabled")
	assert.Contains(t, err.Error(), "trajectory_path", "error should mention trajectory_path field")
}

// TestV2_Validate_ACETopKPositive tests that TopK must be positive when ACE is enabled.
// Kills mutant: removing the TopK check would make this test fail.
func TestV2_Validate_ACETopKPositive(t *testing.T) {
	t.Parallel()

	ace := validACEConfig()
	ace.TopK = 0 // Invalid.

	cfg := &V2{
		Version: "2.0",
		LLM:     validLLMConfig(),
		Agent:   validAgentConfig(),
		ACE:     ace,
	}

	err := cfg.Validate()
	require.Error(t, err, "zero TopK should fail validation when ACE enabled")
	assert.Contains(t, err.Error(), "top_k", "error should mention top_k field")
}

// TestV2_Validate_ACEMinScoreRange tests that MinScore must be between 0 and 1.
// Kills mutant: removing the MinScore check would make this test fail.
func TestV2_Validate_ACEMinScoreRange(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		minScore float64
	}{
		{"negative", -0.1},
		{"too high", 1.1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ace := validACEConfig()
			ace.MinScore = tt.minScore

			cfg := &V2{
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

// TestV2_Validate_ACEDisabled tests that validation passes when ACE is disabled.
// Kills mutant: removing the ACE.Enabled check would make this test fail.
func TestV2_Validate_ACEDisabled(t *testing.T) {
	t.Parallel()

	cfg := &V2{
		Version: "2.0",
		LLM:     validLLMConfig(),
		Agent:   validAgentConfig(),
		ACE: ACEV2{
			Enabled: false,
			// Empty paths should be OK when disabled.
			PlaybookPath:   "",
			TrajectoryPath: "",
			TopK:           0,
			MinScore:       0,
		},
	}

	err := cfg.Validate()
	require.NoError(t, err, "ACE disabled config should pass validation")
}

// TestV2_Validate_SecuritySandboxModeValid tests that only valid sandbox modes are accepted.
// Kills mutant: removing the SandboxMode validation would make this test fail.
func TestV2_Validate_SecuritySandboxModeValid(t *testing.T) {
	t.Parallel()

	validModes := []string{"", "none", "docker", "firejail"}
	for _, mode := range validModes {
		t.Run(mode, func(t *testing.T) {
			t.Parallel()

			cfg := &V2{
				Version:  "2.0",
				LLM:      validLLMConfig(),
				Agent:    validAgentConfig(),
				ACE:      validACEConfig(),
				Security: SecurityV2{SandboxMode: mode},
				Protocol: validProtocolConfig(),
			}

			err := cfg.Validate()
			require.NoError(t, err, "sandbox mode %q should be valid", mode)
		})
	}
}

// TestV2_Validate_SecuritySandboxModeInvalid tests that invalid sandbox modes are rejected.
// Kills mutant: removing the SandboxMode validation would make this test fail.
func TestV2_Validate_SecuritySandboxModeInvalid(t *testing.T) {
	t.Parallel()

	cfg := &V2{
		Version:  "2.0",
		LLM:      validLLMConfig(),
		Agent:    validAgentConfig(),
		ACE:      validACEConfig(),
		Security: SecurityV2{SandboxMode: "invalid"},
		Protocol: validProtocolConfig(),
	}

	err := cfg.Validate()
	require.Error(t, err, "invalid sandbox mode should fail validation")
	assert.Contains(t, err.Error(), "sandbox_mode", "error should mention sandbox_mode field")
}

// TestV2_Validate_ProtocolShellTimeoutPositive tests that ShellTimeout must be positive when shell is enabled.
// Kills mutant: removing the ShellTimeout check would make this test fail.
func TestV2_Validate_ProtocolShellTimeoutPositive(t *testing.T) {
	t.Parallel()

	protocol := validProtocolConfig()
	protocol.EnableShell = true
	protocol.ShellTimeout = 0 // Invalid.

	cfg := &V2{
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

// TestV2_Validate_ProtocolMCPServersValid tests that MCP server configs are validated when MCP is enabled.
// Kills mutant: removing the MCP server validation would make this test fail.
func TestV2_Validate_ProtocolMCPServersValid(t *testing.T) {
	t.Parallel()

	protocol := validProtocolConfig()
	protocol.EnableMCP = true
	protocol.MCPServers = []MCPServerConfigV2{
		{Name: "", Command: "/usr/bin/mcp"}, // Invalid: empty name.
	}

	cfg := &V2{
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

// TestDefaultV2 tests that the default configuration is valid.
// Kills mutant: changing defaults to invalid values would make this test fail.
func TestDefaultV2(t *testing.T) {
	t.Parallel()

	cfg := DefaultV2()

	// Should be valid by default.
	err := cfg.Validate()
	require.NoError(t, err, "default config should pass validation")

	// Check some expected defaults.
	assert.Equal(t, "2.0", cfg.Version, "version should be 2.0")
	assert.Equal(t, "ollama", cfg.LLM.Provider, "default provider should be ollama")
	assert.InDelta(t, 0.7, cfg.LLM.Temperature, 1e-9, "default temperature should be 0.7")
	assert.Equal(t, 50, cfg.Agent.MaxTurns, "default max_turns should be 50")
	assert.False(t, cfg.ACE.Enabled, "ACE should be disabled by default")
}

// Journey: specs/journeys/JOURNEY-011-apply-compact-to-shell-exec.md.
func TestDefaultV2_CompactEnabledOn(t *testing.T) {
	t.Parallel()

	cfg := DefaultV2()
	assert.True(t, cfg.Compact.Enabled, "compact.enabled should default on")
	assert.Empty(t, cfg.Compact.Backend, "compact.backend should default empty (Go pipeline)")
	assert.Equal(t, DefaultCompactReadLevel, cfg.Compact.ReadLevel)
}

func TestCompactV2_ActiveEnvOff(t *testing.T) {
	t.Setenv("SPIN_COMPACT", "0")

	cfg := CompactV2{Enabled: true}
	assert.False(t, cfg.Active(), "SPIN_COMPACT=0 must disable compact")
}

func TestCompactV2_ActiveConfigOff(t *testing.T) {
	t.Setenv("SPIN_COMPACT", "")

	cfg := CompactV2{Enabled: false}
	assert.False(t, cfg.Active(), "compact.enabled false must disable compact")
}

// TestV2_CrossSectionValidation_ACEPlaybookRequired tests that ACE playbook is required when enabled.
// Kills mutant: removing cross-section validation would make this test fail.
func TestV2_CrossSectionValidation_ACEPlaybookRequired(t *testing.T) {
	t.Parallel()

	cfg := &V2{
		Version: "2.0",
		LLM:     validLLMConfig(),
		Agent:   validAgentConfig(),
		ACE: ACEV2{
			Enabled:      true,
			PlaybookPath: "", // Invalid - required when enabled.
		},
		Security: validSecurityConfig(),
		Protocol: validProtocolConfig(),
	}

	err := cfg.Validate()
	require.Error(t, err, "validation should fail when ACE enabled but playbook_path empty")
	assert.Contains(t, err.Error(), "playbook_path", "error should mention playbook_path")
}

// TestV2_CrossSectionValidation_ACETrajectoryRequired tests that ACE trajectory is required when enabled.
// Kills mutant: removing cross-section validation would make this test fail.
func TestV2_CrossSectionValidation_ACETrajectoryRequired(t *testing.T) {
	t.Parallel()

	cfg := &V2{
		Version: "2.0",
		LLM:     validLLMConfig(),
		Agent:   validAgentConfig(),
		ACE: ACEV2{
			Enabled:        true,
			PlaybookPath:   "/path/to/playbook.json",
			TrajectoryPath: "", // Invalid - required when enabled.
		},
		Security: validSecurityConfig(),
		Protocol: validProtocolConfig(),
	}

	err := cfg.Validate()
	require.Error(t, err, "validation should fail when ACE enabled but trajectory_path empty")
	assert.Contains(t, err.Error(), "trajectory_path", "error should mention trajectory_path")
}

// TestV2_CrossSectionValidation_ShellTimeoutRequired tests that shell timeout is required when shell enabled.
// Kills mutant: removing cross-section validation would make this test fail.
func TestV2_CrossSectionValidation_ShellTimeoutRequired(t *testing.T) {
	t.Parallel()

	cfg := &V2{
		Version:  "2.0",
		LLM:      validLLMConfig(),
		Agent:    validAgentConfig(),
		ACE:      validACEConfig(),
		Security: validSecurityConfig(),
		Protocol: ProtocolV2{
			EnableShell:  true,
			ShellTimeout: 0, // Invalid - required when enabled.
		},
	}

	err := cfg.Validate()
	require.Error(t, err, "validation should fail when shell enabled but timeout is zero")
	assert.Contains(t, err.Error(), "shell_timeout", "error should mention shell_timeout")
}

// TestV2_Validation_AllErrors tests that validation returns ALL errors, not just the first.
// Kills mutant: fail-fast validation would make this test fail.
func TestV2_Validation_AllErrors(t *testing.T) {
	t.Parallel()

	cfg := &V2{
		Version: "2.0",
		LLM: LLMV2{
			Provider:    "",  // Error 1.
			Model:       "",  // Error 2.
			Temperature: 3.0, // Error 3.
			MaxTokens:   -10, // Error 4.
			Timeout:     0,   // Error 5.
		},
		Agent:    validAgentConfig(),
		ACE:      validACEConfig(),
		Security: validSecurityConfig(),
		Protocol: validProtocolConfig(),
	}

	err := cfg.Validate()
	require.Error(t, err, "validation should fail with multiple errors")

	// Check that error message contains multiple issues.
	errMsg := err.Error()
	assert.Contains(t, errMsg, "provider", "should mention provider error")
	assert.Contains(t, errMsg, "model", "should mention model error")
	// Note: Current implementation is fail-fast, this test documents desired behavior.
}

// TestMCPServerConfigV2_Validate_SSE tests SSE transport validation.
func TestMCPServerConfigV2_Validate_SSE(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		config  MCPServerConfigV2
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid sse config",
			config: MCPServerConfigV2{
				Name:      "smithery-server",
				Transport: MCPTransportSSE,
				URL:       "https://server.smithery.ai/sse",
			},
			wantErr: false,
		},
		{
			name: "valid sse config with headers",
			config: MCPServerConfigV2{
				Name:      "smithery-server",
				Transport: MCPTransportSSE,
				URL:       "https://server.smithery.ai/sse",
				Headers: map[string]string{
					"Authorization": "Bearer token",
				},
			},
			wantErr: false,
		},
		{
			name: "invalid sse missing url",
			config: MCPServerConfigV2{
				Name:      "smithery-server",
				Transport: MCPTransportSSE,
			},
			wantErr: true,
			errMsg:  "url is required",
		},
		{
			name: "invalid sse with command",
			config: MCPServerConfigV2{
				Name:      "smithery-server",
				Transport: MCPTransportSSE,
				URL:       "https://server.smithery.ai/sse",
				Command:   "echo",
			},
			wantErr: true,
			errMsg:  "command is not allowed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := tt.config.Validate()
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// TestMCPServerConfigV2_Validate_StreamableHTTP tests streamable-http transport validation.
func TestMCPServerConfigV2_Validate_StreamableHTTP(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		config  MCPServerConfigV2
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid streamable-http config",
			config: MCPServerConfigV2{
				Name:      "remote-server",
				Transport: MCPTransportStreamableHTTP,
				URL:       "https://mcp.example.com/v1",
			},
			wantErr: false,
		},
		{
			name: "invalid streamable-http missing url",
			config: MCPServerConfigV2{
				Name:      "remote-server",
				Transport: MCPTransportStreamableHTTP,
			},
			wantErr: true,
			errMsg:  "url is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := tt.config.Validate()
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// TestMCPServerConfigV2_Validate_OAuth tests OAuth configuration validation.
func TestMCPServerConfigV2_Validate_OAuth(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		config  MCPServerConfigV2
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid sse with oauth",
			config: MCPServerConfigV2{
				Name:      "protected-server",
				Transport: MCPTransportSSE,
				URL:       "https://protected.example.com/mcp",
				OAuth: &MCPOAuthConfigV2{
					ClientID: "my-client-id",
				},
			},
			wantErr: false,
		},
		{
			name: "invalid oauth with stdio",
			config: MCPServerConfigV2{
				Name:      "local-server",
				Transport: MCPTransportStdio,
				Command:   "echo",
				OAuth: &MCPOAuthConfigV2{
					ClientID: "my-client-id",
				},
			},
			wantErr: true,
			errMsg:  "oauth is not allowed for stdio",
		},
		{
			name: "invalid oauth missing client_id",
			config: MCPServerConfigV2{
				Name:      "protected-server",
				Transport: MCPTransportSSE,
				URL:       "https://protected.example.com/mcp",
				OAuth:     &MCPOAuthConfigV2{},
			},
			wantErr: true,
			errMsg:  "oauth client_id is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := tt.config.Validate()
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// TestMCPTransportType_IsValid tests transport type validation.
func TestMCPTransportType_IsValid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		transport MCPTransportType
		want      bool
	}{
		{"", true},
		{MCPTransportStdio, true},
		{MCPTransportSSE, true},
		{MCPTransportStreamableHTTP, true},
		{"websocket", false},
		{"invalid", false},
	}

	for _, tt := range tests {
		t.Run(string(tt.transport), func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, tt.transport.IsValid())
		})
	}
}

// Journey: specs/journeys/JOURNEY-1.1.md.

// TestV2_Validate_WorkflowsPresent tests that workflows section is parsed correctly.
// Kills mutant: removing workflows struct fields would make this test fail.
func TestV2_Validate_WorkflowsPresent(t *testing.T) {
	t.Parallel()

	cfg := &V2{
		Version: "2.0",
		LLM:     validLLMConfig(),
		Agent:   validAgentConfig(),
		Workflows: WorkflowsV2{
			ActionModel:   "claude-opus-4-6",
			ThinkingModel: "deepseek-r1",
			CritiqueModel: "claude-sonnet-4-6",
			CompactModel:  "claude-haiku-4-5",
			VisionModel:   "claude-opus-4-6",
		},
	}

	err := cfg.Validate()
	require.NoError(t, err, "config with workflows should pass validation")

	assert.Equal(t, "claude-opus-4-6", cfg.Workflows.ActionModel)
	assert.Equal(t, "deepseek-r1", cfg.Workflows.ThinkingModel)
	assert.Equal(t, "claude-sonnet-4-6", cfg.Workflows.CritiqueModel)
	assert.Equal(t, "claude-haiku-4-5", cfg.Workflows.CompactModel)
	assert.Equal(t, "claude-opus-4-6", cfg.Workflows.VisionModel)
}

// TestV2_Validate_WorkflowsEmpty tests that empty workflows section passes validation.
// Kills mutant: adding required-field validation for optional workflows would make this test fail.
func TestV2_Validate_WorkflowsEmpty(t *testing.T) {
	t.Parallel()

	cfg := &V2{
		Version:   "2.0",
		LLM:       validLLMConfig(),
		Agent:     validAgentConfig(),
		Workflows: WorkflowsV2{},
	}

	err := cfg.Validate()
	require.NoError(t, err, "config with empty workflows should pass validation")
}

// TestWorkflowsV2_ResolveThinkingModel tests the thinking model fallback chain.
// Kills mutant: removing fallback logic would make this test fail.
func TestWorkflowsV2_ResolveThinkingModel(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		wf       WorkflowsV2
		expected string
	}{
		{
			name:     "explicit thinking model",
			wf:       WorkflowsV2{ActionModel: "action", ThinkingModel: "thinking"},
			expected: "thinking",
		},
		{
			name:     "fallback to action model",
			wf:       WorkflowsV2{ActionModel: "action", ThinkingModel: ""},
			expected: "action",
		},
		{
			name:     "both empty returns empty",
			wf:       WorkflowsV2{},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, tt.wf.ResolveThinkingModel())
		})
	}
}

// TestWorkflowsV2_ResolveCritiqueModel tests the critique model fallback chain.
// Kills mutant: removing two-level fallback would make this test fail.
func TestWorkflowsV2_ResolveCritiqueModel(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		wf       WorkflowsV2
		expected string
	}{
		{
			name:     "explicit critique model",
			wf:       WorkflowsV2{ActionModel: "action", ThinkingModel: "thinking", CritiqueModel: "critique"},
			expected: "critique",
		},
		{
			name:     "fallback to thinking model",
			wf:       WorkflowsV2{ActionModel: "action", ThinkingModel: "thinking", CritiqueModel: ""},
			expected: "thinking",
		},
		{
			name:     "fallback through thinking to action",
			wf:       WorkflowsV2{ActionModel: "action"},
			expected: "action",
		},
		{
			name:     "all empty returns empty",
			wf:       WorkflowsV2{},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, tt.wf.ResolveCritiqueModel())
		})
	}
}

// TestWorkflowsV2_ResolveCompactAndVisionModel tests compact and vision model fallback chains.
// Kills mutant: removing fallback logic would make this test fail.
func TestWorkflowsV2_ResolveCompactAndVisionModel(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		wf       WorkflowsV2
		role     string
		expected string
	}{
		{
			name:     "explicit compact model",
			wf:       WorkflowsV2{ActionModel: "action", CompactModel: "compact"},
			role:     "compact",
			expected: "compact",
		},
		{
			name:     "compact fallback to action model",
			wf:       WorkflowsV2{ActionModel: "action"},
			role:     "compact",
			expected: "action",
		},
		{
			name:     "explicit vision model",
			wf:       WorkflowsV2{ActionModel: "action", VisionModel: "vision"},
			role:     "vision",
			expected: "vision",
		},
		{
			name:     "vision fallback to action model",
			wf:       WorkflowsV2{ActionModel: "action"},
			role:     "vision",
			expected: "action",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var got string

			switch tt.role {
			case "compact":
				got = tt.wf.ResolveCompactModel()
			case "vision":
				got = tt.wf.ResolveVisionModel()
			}

			assert.Equal(t, tt.expected, got)
		})
	}
}

// TestV2_Validate_SubagentsPresent tests that subagents section is parsed correctly.
// Kills mutant: removing subagents field from V2 would make this test fail.
func TestV2_Validate_SubagentsPresent(t *testing.T) {
	t.Parallel()

	cfg := &V2{
		Version: "2.0",
		LLM:     validLLMConfig(),
		Agent:   validAgentConfig(),
		Subagents: map[string]SubagentConfigV2{
			"planner": {
				Model:         "claude-sonnet-4-6",
				MaxIterations: 50,
			},
			"explorer": {
				Model:         "claude-haiku-4-5",
				MaxIterations: 30,
			},
		},
	}

	err := cfg.Validate()
	require.NoError(t, err, "config with valid subagents should pass validation")

	require.Len(t, cfg.Subagents, 2)
	assert.Equal(t, "claude-sonnet-4-6", cfg.Subagents["planner"].Model)
	assert.Equal(t, 50, cfg.Subagents["planner"].MaxIterations)
	assert.Equal(t, "claude-haiku-4-5", cfg.Subagents["explorer"].Model)
	assert.Equal(t, 30, cfg.Subagents["explorer"].MaxIterations)
}

// TestV2_Validate_SubagentsNil tests that nil subagents map passes validation.
// Kills mutant: requiring non-nil subagents would make this test fail.
func TestV2_Validate_SubagentsNil(t *testing.T) {
	t.Parallel()

	cfg := &V2{
		Version:   "2.0",
		LLM:       validLLMConfig(),
		Agent:     validAgentConfig(),
		Subagents: nil,
	}

	err := cfg.Validate()
	require.NoError(t, err, "config with nil subagents should pass validation")
}

// TestV2_Validate_SubagentsMaxIterationsNegative tests that negative MaxIterations fails.
// Kills mutant: removing MaxIterations validation would make this test fail.
func TestV2_Validate_SubagentsMaxIterationsNegative(t *testing.T) {
	t.Parallel()

	cfg := &V2{
		Version: "2.0",
		LLM:     validLLMConfig(),
		Agent:   validAgentConfig(),
		Subagents: map[string]SubagentConfigV2{
			"planner": {
				Model:         "claude-sonnet-4-6",
				MaxIterations: -1,
			},
		},
	}

	err := cfg.Validate()
	require.Error(t, err, "negative MaxIterations should fail validation")
	assert.Contains(t, err.Error(), "max_iterations", "error should mention max_iterations")
}

// TestV2_Validate_SubagentsMaxIterationsZeroAllowed tests that zero MaxIterations passes
// validation (consumers apply default of 30).
// Kills mutant: rejecting zero MaxIterations would make this test fail.
func TestV2_Validate_SubagentsMaxIterationsZeroAllowed(t *testing.T) {
	t.Parallel()

	cfg := &V2{
		Version: "2.0",
		LLM:     validLLMConfig(),
		Agent:   validAgentConfig(),
		Subagents: map[string]SubagentConfigV2{
			"planner": {
				Model:         "claude-sonnet-4-6",
				MaxIterations: 0,
			},
		},
	}

	err := cfg.Validate()
	require.NoError(t, err, "zero MaxIterations should pass validation (consumers apply default)")
}

// TestSubagentConfigV2_EffectiveMaxIterations tests the default application.
// Kills mutant: removing default logic would make this test fail.
func TestSubagentConfigV2_EffectiveMaxIterations(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		cfg      SubagentConfigV2
		expected int
	}{
		{
			name:     "explicit value",
			cfg:      SubagentConfigV2{MaxIterations: 50},
			expected: 50,
		},
		{
			name:     "zero uses default",
			cfg:      SubagentConfigV2{MaxIterations: 0},
			expected: 30,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, tt.cfg.EffectiveMaxIterations())
		})
	}
}

// TestDefaultV2_WorkflowsAndSubagents tests that defaults include empty workflows and nil subagents.
// Kills mutant: changing defaults would make this test fail.
func TestDefaultV2_WorkflowsAndSubagents(t *testing.T) {
	t.Parallel()

	cfg := DefaultV2()

	assert.Equal(t, WorkflowsV2{}, cfg.Workflows, "workflows should be zero value by default")
	assert.Nil(t, cfg.Subagents, "subagents should be nil by default")
	assert.Empty(t, cfg.Plugins.Paths, "plugins.paths should be empty by default")
}

// Journey: specs/journeys/JOURNEY-024-remote-a2a-https-client-and-card-allowlist.md.
func TestDefaultV2_A2AAllowlistEmpty(t *testing.T) {
	t.Parallel()

	cfg := DefaultV2()
	assert.Empty(t, cfg.A2A.Allowlist, "a2a.allowlist must default empty (no remote cards)")
}

func TestPluginsV2_PathsUnmarshal(t *testing.T) {
	t.Parallel()

	const payload = `
llm:
  provider: ollama
  model: test
  temperature: 0.7
  max_tokens: 100
  timeout: 1m
agent:
  max_turns: 1
  timeout: 1m
  work_dir: /tmp
plugins:
  paths:
    - /opt/spin/plugins/acme
`

	var cfg V2
	require.NoError(t, yaml.Unmarshal([]byte(payload), &cfg))
	require.Equal(t, []string{"/opt/spin/plugins/acme"}, cfg.Plugins.Paths)
}

// Journey: specs/journeys/JOURNEY-024-remote-a2a-https-client-and-card-allowlist.md.
func TestV2_A2AAllowlistUnmarshal(t *testing.T) {
	t.Parallel()

	const payload = `
a2a:
  allowlist:
    - https://peer.example.com/.well-known/agent-card.json
`

	var cfg V2
	require.NoError(t, yaml.Unmarshal([]byte(payload), &cfg))
	require.Equal(t, []string{"https://peer.example.com/.well-known/agent-card.json"}, cfg.A2A.Allowlist)
}

// Journey: specs/journeys/JOURNEY-024-remote-a2a-https-client-and-card-allowlist.md.
func TestV2_A2AAllowlistRejectsHTTP(t *testing.T) {
	t.Parallel()

	cfg := DefaultV2()
	cfg.A2A.Allowlist = []string{"http://peer.example.com/card.json"}
	err := cfg.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "a2a.allowlist")
}

// Journey: specs/journeys/JOURNEY-024-remote-a2a-https-client-and-card-allowlist.md.
func TestA2AV2_AllowsRequiresExactEntry(t *testing.T) {
	t.Parallel()

	empty := A2AV2{}
	assert.False(t, empty.Allows("https://peer.example.com/card"))

	listed := A2AV2{Allowlist: []string{"https://peer.example.com/card"}}
	assert.True(t, listed.Allows("https://peer.example.com/card"))
	assert.False(t, listed.Allows("https://other.example.com/card"))
}

// TestMCPTransportType_IsRemote tests transport type remote detection.
func TestMCPTransportType_IsRemote(t *testing.T) {
	t.Parallel()

	tests := []struct {
		transport MCPTransportType
		want      bool
	}{
		{"", false},
		{MCPTransportStdio, false},
		{MCPTransportSSE, true},
		{MCPTransportStreamableHTTP, true},
	}

	for _, tt := range tests {
		t.Run(string(tt.transport), func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, tt.transport.IsRemote())
		})
	}
}
