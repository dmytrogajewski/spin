package config

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestV2_Validate_MinimalValid tests that a minimal valid v2 config passes validation.
// This is the first step in the new v2.0 config structure.
func TestV2_Validate_MinimalValid(t *testing.T) {
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

// TestV2_Validate_LLMFieldRanges tests validation of numeric field ranges.
// Kills mutants: removing range checks would make these tests fail.
func TestV2_Validate_LLMFieldRanges(t *testing.T) {
	tests := []struct {
		name    string
		cfg     V2
		wantErr string
	}{
		{
			name: "temperature too low",
			cfg: V2{
				Version: "2.0",
				LLM: LLMV2{
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
			cfg: V2{
				Version: "2.0",
				LLM: LLMV2{
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
			cfg: V2{
				Version: "2.0",
				LLM: LLMV2{
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
			cfg: V2{
				Version: "2.0",
				LLM: LLMV2{
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
			cfg: V2{
				Version: "2.0",
				LLM: LLMV2{
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
			cfg: V2{
				Version: "2.0",
				LLM: LLMV2{
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

// TestV2_Validate_LLMValidRanges tests that valid values pass validation.
func TestV2_Validate_LLMValidRanges(t *testing.T) {
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
	validModes := []string{"", "none", "docker", "firejail"}
	for _, mode := range validModes {
		t.Run(mode, func(t *testing.T) {
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
	cfg := DefaultV2()

	// Should be valid by default.
	err := cfg.Validate()
	require.NoError(t, err, "default config should pass validation")

	// Check some expected defaults.
	assert.Equal(t, "2.0", cfg.Version, "version should be 2.0")
	assert.Equal(t, "ollama", cfg.LLM.Provider, "default provider should be ollama")
	assert.Equal(t, 0.7, cfg.LLM.Temperature, "default temperature should be 0.7")
	assert.Equal(t, 50, cfg.Agent.MaxTurns, "default max_turns should be 50")
	assert.False(t, cfg.ACE.Enabled, "ACE should be disabled by default")
}

// TestV2_CrossSectionValidation_ACEPlaybookRequired tests that ACE playbook is required when enabled.
// Kills mutant: removing cross-section validation would make this test fail.
func TestV2_CrossSectionValidation_ACEPlaybookRequired(t *testing.T) {
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
			assert.Equal(t, tt.want, tt.transport.IsValid())
		})
	}
}

// TestMCPTransportType_IsRemote tests transport type remote detection.
func TestMCPTransportType_IsRemote(t *testing.T) {
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
			assert.Equal(t, tt.want, tt.transport.IsRemote())
		})
	}
}
