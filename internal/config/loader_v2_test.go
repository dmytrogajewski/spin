package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestLoaderV2_LoadFromFile tests loading configuration from a YAML file.
// Kills mutant: removing file loading would make this test fail.
func TestLoaderV2_LoadFromFile(t *testing.T) {
	// Create a temporary config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	configYAML := `version: "2.0"
llm:
  provider: "openai"
  model: "gpt-4"
  temperature: 0.8
  max_tokens: 2048
  timeout: 3m
agent:
  max_turns: 20
  timeout: 30m
  work_dir: "/tmp/spin"
ace:
  enabled: false
security:
  sandbox_mode: "docker"
protocol:
  enable_git: false
  enable_shell: true
  shell_timeout: 60s
`

	err := os.WriteFile(configPath, []byte(configYAML), 0644)
	require.NoError(t, err, "failed to write test config file")

	// Load the configuration
	loader := NewLoaderV2()
	cfg, err := loader.LoadFromFile(configPath)
	require.NoError(t, err, "failed to load config from file")

	// Verify loaded values
	assert.Equal(t, "2.0", cfg.Version)
	assert.Equal(t, "openai", cfg.LLM.Provider)
	assert.Equal(t, "gpt-4", cfg.LLM.Model)
	assert.Equal(t, 0.8, cfg.LLM.Temperature)
	assert.Equal(t, 2048, cfg.LLM.MaxTokens)
	assert.Equal(t, 3*time.Minute, cfg.LLM.Timeout)
	assert.Equal(t, 20, cfg.Agent.MaxTurns)
	assert.Equal(t, 30*time.Minute, cfg.Agent.Timeout)
	assert.Equal(t, "/tmp/spin", cfg.Agent.WorkDir)
	assert.False(t, cfg.ACE.Enabled)
	assert.Equal(t, "docker", cfg.Security.SandboxMode)
	assert.False(t, cfg.Protocol.EnableGit)
	assert.True(t, cfg.Protocol.EnableShell)
	assert.Equal(t, 60*time.Second, cfg.Protocol.ShellTimeout)
}

// TestLoaderV2_LoadFromFileNotFound tests that loading from a non-existent file returns an error.
// Kills mutant: removing error handling would make this test fail.
func TestLoaderV2_LoadFromFileNotFound(t *testing.T) {
	loader := NewLoaderV2()
	_, err := loader.LoadFromFile("/nonexistent/config.yaml")
	require.Error(t, err, "loading from non-existent file should return error")
}

// TestLoaderV2_LoadFromFileInvalidYAML tests that invalid YAML returns an error.
// Kills mutant: removing YAML validation would make this test fail.
func TestLoaderV2_LoadFromFileInvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "invalid.yaml")

	invalidYAML := `version: "2.0"
llm:
  provider: "openai"
  invalid yaml here: [unclosed
`

	err := os.WriteFile(configPath, []byte(invalidYAML), 0644)
	require.NoError(t, err, "failed to write test config file")

	loader := NewLoaderV2()
	_, err = loader.LoadFromFile(configPath)
	require.Error(t, err, "loading invalid YAML should return error")
}

// TestLoaderV2_LoadWithDefaults tests that missing fields use default values.
// Kills mutant: removing default value merging would make this test fail.
func TestLoaderV2_LoadWithDefaults(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "minimal.yaml")

	// Minimal config with only required fields
	minimalYAML := `version: "2.0"
llm:
  provider: "ollama"
  model: "qwen"
  temperature: 0.7
  max_tokens: 4096
  timeout: 5m
agent:
  max_turns: 10
  timeout: 60m
  work_dir: "."
`

	err := os.WriteFile(configPath, []byte(minimalYAML), 0644)
	require.NoError(t, err, "failed to write test config file")

	loader := NewLoaderV2()
	cfg, err := loader.LoadFromFile(configPath)
	require.NoError(t, err, "failed to load minimal config")

	// Verify explicitly set values
	assert.Equal(t, "ollama", cfg.LLM.Provider)
	assert.Equal(t, "qwen", cfg.LLM.Model)

	// Verify defaults for unset fields
	// ACE should have defaults since not specified
	assert.True(t, cfg.ACE.Enabled, "ACE should be enabled by default")
	assert.Equal(t, "~/.spin/ace/playbooks/default.json", cfg.ACE.PlaybookPath)
	assert.Equal(t, 5, cfg.ACE.TopK)

	// Security defaults
	assert.Equal(t, "workspace-only", cfg.Security.SandboxMode)

	// Protocol defaults
	assert.True(t, cfg.Protocol.EnableGit)
	assert.True(t, cfg.Protocol.EnableShell)
}

// TestLoaderV2_LoadFromEnv tests loading configuration from environment variables.
// Kills mutant: removing environment variable support would make this test fail.
func TestLoaderV2_LoadFromEnv(t *testing.T) {
	// Set environment variables
	t.Setenv("SPIN_LLM_PROVIDER", "anthropic")
	t.Setenv("SPIN_LLM_MODEL", "claude-3")
	t.Setenv("SPIN_LLM_TEMPERATURE", "0.5")
	t.Setenv("SPIN_AGENT_MAX_TURNS", "15")
	t.Setenv("SPIN_ACE_ENABLED", "false")
	t.Setenv("SPIN_SECURITY_SANDBOX_MODE", "firejail")

	loader := NewLoaderV2()
	cfg, err := loader.LoadWithEnv()
	require.NoError(t, err, "failed to load config with env vars")

	// Verify env var values override defaults
	assert.Equal(t, "anthropic", cfg.LLM.Provider)
	assert.Equal(t, "claude-3", cfg.LLM.Model)
	assert.Equal(t, 0.5, cfg.LLM.Temperature)
	assert.Equal(t, 15, cfg.Agent.MaxTurns)
	assert.False(t, cfg.ACE.Enabled)
	assert.Equal(t, "firejail", cfg.Security.SandboxMode)
}

// TestLoaderV2_Precedence tests that configuration sources have correct precedence.
// Kills mutant: changing precedence order would make this test fail.
func TestLoaderV2_Precedence(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	// File has provider="openai"
	configYAML := `version: "2.0"
llm:
  provider: "openai"
  model: "gpt-4"
  temperature: 0.7
  max_tokens: 4096
  timeout: 5m
agent:
  max_turns: 10
  timeout: 60m
  work_dir: "."
`

	err := os.WriteFile(configPath, []byte(configYAML), 0644)
	require.NoError(t, err, "failed to write test config file")

	// Env var has provider="anthropic" - should override file
	t.Setenv("SPIN_LLM_PROVIDER", "anthropic")
	t.Setenv("SPIN_LLM_MODEL", "claude-3")

	loader := NewLoaderV2()
	cfg, err := loader.LoadFromFileWithEnv(configPath)
	require.NoError(t, err, "failed to load config")

	// Env var should win over file
	assert.Equal(t, "anthropic", cfg.LLM.Provider, "env var should override file")
	assert.Equal(t, "claude-3", cfg.LLM.Model, "env var should override file")

	// File value should be used when no env var
	assert.Equal(t, 0.7, cfg.LLM.Temperature, "file value should be used")
}

// TestLoad_EmptySource tests that Load() with empty Source uses defaults.
// Kills mutant: removing default application would make this test fail.
func TestLoad_EmptySource(t *testing.T) {
	cfg, err := Load(Source{})
	require.NoError(t, err, "Load with empty source should succeed")

	// Should have default values
	assert.Equal(t, "ollama", cfg.LLM.Provider, "should use default provider")
	assert.Equal(t, "qwen2.5-coder:7b", cfg.LLM.Model, "should use default model")
	assert.Equal(t, 50, cfg.Agent.MaxTurns, "should use default max_turns")
}

// TestLoad_FlagOverrides tests that flags override defaults.
// Kills mutant: removing flag application would make this test fail.
func TestLoad_FlagOverrides(t *testing.T) {
	cfg, err := Load(Source{
		Flags: FlagOverrides{
			Provider: "openai",
			Model:    "gpt-4",
			MaxTurns: 20,
		},
	})
	require.NoError(t, err, "Load with flags should succeed")

	// Flags should override defaults
	assert.Equal(t, "openai", cfg.LLM.Provider, "flag should override default provider")
	assert.Equal(t, "gpt-4", cfg.LLM.Model, "flag should override default model")
	assert.Equal(t, 20, cfg.Agent.MaxTurns, "flag should override default max_turns")

	// Non-overridden values should keep defaults
	assert.Equal(t, 0.7, cfg.LLM.Temperature, "should keep default temperature")
}

// TestLoad_FileConfig tests that Load() loads config from file.
// Kills mutant: removing file loading would make this test fail.
func TestLoad_FileConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	configYAML := `version: "2.0"
llm:
  provider: "anthropic"
  model: "claude-3"
  temperature: 0.5
agent:
  max_turns: 30
`

	err := os.WriteFile(configPath, []byte(configYAML), 0644)
	require.NoError(t, err, "failed to write test config file")

	cfg, err := Load(Source{File: configPath})
	require.NoError(t, err, "Load with file should succeed")

	// File values should override defaults
	assert.Equal(t, "anthropic", cfg.LLM.Provider, "file should override default provider")
	assert.Equal(t, "claude-3", cfg.LLM.Model, "file should override default model")
	assert.Equal(t, 0.5, cfg.LLM.Temperature, "file should override default temperature")
	assert.Equal(t, 30, cfg.Agent.MaxTurns, "file should override default max_turns")
}

// TestLoad_PrecedenceFlagsOverFile tests that flags override file config.
// Kills mutant: changing precedence order would make this test fail.
func TestLoad_PrecedenceFlagsOverFile(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	configYAML := `version: "2.0"
llm:
  provider: "anthropic"
  model: "claude-3"
agent:
  max_turns: 30
`

	err := os.WriteFile(configPath, []byte(configYAML), 0644)
	require.NoError(t, err, "failed to write test config file")

	cfg, err := Load(Source{
		File: configPath,
		Flags: FlagOverrides{
			Provider: "openai",
			Model:    "gpt-4",
		},
	})
	require.NoError(t, err, "Load with file and flags should succeed")

	// Flags should win over file
	assert.Equal(t, "openai", cfg.LLM.Provider, "flag should override file provider")
	assert.Equal(t, "gpt-4", cfg.LLM.Model, "flag should override file model")

	// File value should be used when no flag override
	assert.Equal(t, 30, cfg.Agent.MaxTurns, "file should be used when no flag override")
}

// TestLoad_AllFlagFields tests that all flag fields are applied.
// Kills mutant: removing any flag field application would make this test fail.
func TestLoad_AllFlagFields(t *testing.T) {
	cfg, err := Load(Source{
		Flags: FlagOverrides{
			Provider: "openai",
			Model:    "gpt-4",
			BaseURL:  "https://api.custom.com",
			MaxTurns: 15,
			Debug:    true,
			Sandbox:  "docker",
		},
		WorkDir: "/custom/dir",
	})
	require.NoError(t, err, "Load with all flags should succeed")

	assert.Equal(t, "openai", cfg.LLM.Provider)
	assert.Equal(t, "gpt-4", cfg.LLM.Model)
	assert.Equal(t, "https://api.custom.com", cfg.LLM.BaseURL)
	assert.Equal(t, 15, cfg.Agent.MaxTurns)
	assert.True(t, cfg.Agent.Debug)
	assert.Equal(t, "debug", cfg.Agent.LogLevel)
	assert.Equal(t, "docker", cfg.Security.SandboxMode)
	assert.Equal(t, "/custom/dir", cfg.Agent.WorkDir)
}

// TestLoad_APIKeyFromEnv tests that API key is loaded from environment.
// Kills mutant: removing env API key resolution would make this test fail.
func TestLoad_APIKeyFromEnv(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "test-key-123")

	cfg, err := Load(Source{
		Flags: FlagOverrides{
			Provider: "openai",
			Model:    "gpt-4",
		},
	})
	require.NoError(t, err, "Load should succeed")

	assert.Equal(t, "test-key-123", cfg.LLM.APIKey, "should load API key from env")
}
