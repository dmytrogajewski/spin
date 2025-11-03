package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMigrateV1ToV2_BasicFields tests migration of basic LLM and Agent fields.
// Kills mutant: removing field mapping would make this test fail.
func TestMigrateV1ToV2_BasicFields(t *testing.T) {
	v1 := &Config{
		Provider:    "openai",
		Model:       "gpt-4",
		Temperature: 0.8,
		MaxTokens:   2048,
		LLMTimeout:  5 * time.Minute,
		MaxTurns:    15,
		Timeout:     30 * time.Minute,
		WorkDir:     "/tmp/test",
	}

	v2 := MigrateV1ToV2(v1)

	// Verify LLM fields
	assert.Equal(t, "openai", v2.LLM.Provider)
	assert.Equal(t, "gpt-4", v2.LLM.Model)
	assert.Equal(t, 0.8, v2.LLM.Temperature)
	assert.Equal(t, 2048, v2.LLM.MaxTokens)
	assert.Equal(t, 5*time.Minute, v2.LLM.Timeout)

	// Verify Agent fields
	assert.Equal(t, 15, v2.Agent.MaxTurns)
	assert.Equal(t, 30*time.Minute, v2.Agent.Timeout)
	assert.Equal(t, "/tmp/test", v2.Agent.WorkDir)
}

// TestMigrateV1ToV2_ACEFields tests migration of ACE configuration.
// Kills mutant: removing ACE mapping would make this test fail.
func TestMigrateV1ToV2_ACEFields(t *testing.T) {
	v1 := &Config{
		Provider:          "ollama",
		Model:             "qwen",
		Temperature:       0.7,
		MaxTokens:         4096,
		LLMTimeout:        5 * time.Minute,
		MaxTurns:          10,
		Timeout:           60 * time.Minute,
		WorkDir:           ".",
		ACEEnabled:        true,
		ACEPlaybookPath:   "/custom/playbook.json",
		ACETrajectoryPath: "/custom/trajectories/",
		ACETopK:           10,
		ACEMinScore:       0.5,
	}

	v2 := MigrateV1ToV2(v1)

	assert.True(t, v2.ACE.Enabled)
	assert.Equal(t, "/custom/playbook.json", v2.ACE.PlaybookPath)
	assert.Equal(t, "/custom/trajectories/", v2.ACE.TrajectoryPath)
	assert.Equal(t, 10, v2.ACE.TopK)
	assert.Equal(t, 0.5, v2.ACE.MinScore)
}

// TestMigrateV1ToV2_SecurityFields tests migration of security configuration.
// Kills mutant: removing security mapping would make this test fail.
func TestMigrateV1ToV2_SecurityFields(t *testing.T) {
	v1 := &Config{
		Provider:        "ollama",
		Model:           "qwen",
		Temperature:     0.7,
		MaxTokens:       4096,
		LLMTimeout:      5 * time.Minute,
		MaxTurns:        10,
		Timeout:         60 * time.Minute,
		WorkDir:         ".",
		SandboxMode:     "docker",
		PolicyFile:      "/etc/spin/policy.json",
		AllowedCommands: []string{"ls", "cat", "grep"},
	}

	v2 := MigrateV1ToV2(v1)

	assert.Equal(t, "docker", v2.Security.SandboxMode)
	assert.Equal(t, "/etc/spin/policy.json", v2.Security.PolicyFile)
	assert.Equal(t, []string{"ls", "cat", "grep"}, v2.Security.AllowedCommands)
}

// TestMigrateV1ToV2_ProtocolFields tests migration of protocol configuration.
// Kills mutant: removing protocol mapping would make this test fail.
func TestMigrateV1ToV2_ProtocolFields(t *testing.T) {
	v1 := &Config{
		Provider:    "ollama",
		Model:       "qwen",
		Temperature: 0.7,
		MaxTokens:   4096,
		LLMTimeout:  5 * time.Minute,
		MaxTurns:    10,
		Timeout:     60 * time.Minute,
		WorkDir:     ".",
		EnableMCP:   true,
		MCPServers: []MCPServerConfig{
			{Name: "server1", Command: "/usr/bin/mcp", Args: []string{"--port", "8080"}},
		},
		EnableGit:    true,
		EnableShell:  true,
		ShellTimeout: 30 * time.Second,
	}

	v2 := MigrateV1ToV2(v1)

	assert.True(t, v2.Protocol.EnableMCP)
	assert.Len(t, v2.Protocol.MCPServers, 1)
	assert.Equal(t, "server1", v2.Protocol.MCPServers[0].Name)
	assert.Equal(t, "/usr/bin/mcp", v2.Protocol.MCPServers[0].Command)
	assert.Equal(t, []string{"--port", "8080"}, v2.Protocol.MCPServers[0].Args)
	assert.True(t, v2.Protocol.EnableGit)
	assert.True(t, v2.Protocol.EnableShell)
	assert.Equal(t, 30*time.Second, v2.Protocol.ShellTimeout)
}

// TestMigrateV1ToV2_MCPServers tests MCP server configuration migration.
// Kills mutant: removing MCP server mapping would make this test fail.
func TestMigrateV1ToV2_MCPServers(t *testing.T) {
	v1 := &Config{
		Provider:    "ollama",
		Model:       "qwen",
		Temperature: 0.7,
		MaxTokens:   4096,
		LLMTimeout:  5 * time.Minute,
		MaxTurns:    10,
		Timeout:     60 * time.Minute,
		WorkDir:     ".",
		EnableMCP:   true,
		MCPServers: []MCPServerConfig{
			{
				Name:    "server1",
				Command: "/usr/bin/mcp1",
				Args:    []string{"--config", "/etc/mcp1.conf"},
				Env:     map[string]string{"MCP_PORT": "8080"},
			},
			{
				Name:    "server2",
				Command: "/usr/bin/mcp2",
				Args:    []string{"--verbose"},
				Env:     map[string]string{"MCP_PORT": "9090"},
			},
		},
	}

	v2 := MigrateV1ToV2(v1)

	require.Len(t, v2.Protocol.MCPServers, 2)

	// Check first server
	assert.Equal(t, "server1", v2.Protocol.MCPServers[0].Name)
	assert.Equal(t, "/usr/bin/mcp1", v2.Protocol.MCPServers[0].Command)
	assert.Equal(t, []string{"--config", "/etc/mcp1.conf"}, v2.Protocol.MCPServers[0].Args)
	assert.Equal(t, map[string]string{"MCP_PORT": "8080"}, v2.Protocol.MCPServers[0].Env)

	// Check second server
	assert.Equal(t, "server2", v2.Protocol.MCPServers[1].Name)
	assert.Equal(t, "/usr/bin/mcp2", v2.Protocol.MCPServers[1].Command)
	assert.Equal(t, []string{"--verbose"}, v2.Protocol.MCPServers[1].Args)
	assert.Equal(t, map[string]string{"MCP_PORT": "9090"}, v2.Protocol.MCPServers[1].Env)
}

// TestMigrateV1ToV2_Validation tests that migrated config passes validation.
// Kills mutant: removing validation would make this test fail.
func TestMigrateV1ToV2_Validation(t *testing.T) {
	v1 := DefaultConfig()
	// Set required fields that DefaultConfig doesn't provide or need updating for v2
	v1.Provider = "ollama"
	v1.Model = "qwen2.5-coder:7b"
	v1.WorkDir = "."
	v1.SandboxMode = "none" // v2 requires specific values

	v2 := MigrateV1ToV2(v1)

	err := v2.Validate()
	require.NoError(t, err, "migrated config should pass validation")
}

// TestLoadV1ConfigFile tests loading a v1 config file and auto-migrating to v2.
// Kills mutant: removing auto-migration would make this test fail.
func TestLoadV1ConfigFile(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	// Write a v1 config file (no version field, flat structure)
	v1YAML := `provider: "openai"
model: "gpt-4"
temperature: 0.8
max_tokens: 2048
llm_timeout: 5m
max_turns: 15
timeout: 30m
work_dir: "/tmp/test"
ace_enabled: true
ace_playbook_path: "/custom/playbook.json"
sandbox_mode: "docker"
enable_git: true
enable_shell: true
shell_timeout: 30s
`

	err := os.WriteFile(configPath, []byte(v1YAML), 0644)
	require.NoError(t, err)

	// Load and auto-migrate
	loader := NewLoaderV2()
	v2, err := loader.LoadV1Compatible(configPath)
	require.NoError(t, err)

	// Verify migration worked
	assert.Equal(t, "2.0", v2.Version)
	assert.Equal(t, "openai", v2.LLM.Provider)
	assert.Equal(t, "gpt-4", v2.LLM.Model)
	assert.Equal(t, 15, v2.Agent.MaxTurns)
	assert.True(t, v2.ACE.Enabled)
	assert.Equal(t, "docker", v2.Security.SandboxMode)
}
