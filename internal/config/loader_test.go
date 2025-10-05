package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoader_LoadFromFile_YAML(t *testing.T) {
	// Create temp YAML file
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "test.yaml")

	yamlContent := `
provider: openai
model: gpt-4
max_turns: 100
enable_git: true
`
	err := os.WriteFile(configFile, []byte(yamlContent), 0644)
	require.NoError(t, err)

	// Load config
	loader := NewLoader()
	err = loader.LoadFromFile(configFile)
	require.NoError(t, err)

	// Verify values
	assert.Equal(t, "openai", loader.GetString("provider"))
	assert.Equal(t, "gpt-4", loader.GetString("model"))
	assert.Equal(t, 100, loader.GetInt("max_turns"))
	assert.Equal(t, true, loader.GetBool("enable_git"))
}

func TestLoader_LoadFromFile_JSON(t *testing.T) {
	// Create temp JSON file
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "test.json")

	jsonContent := `{
  "provider": "anthropic",
  "model": "claude-3-opus",
  "max_turns": 50
}`
	err := os.WriteFile(configFile, []byte(jsonContent), 0644)
	require.NoError(t, err)

	// Load config
	loader := NewLoader()
	err = loader.LoadFromFile(configFile)
	require.NoError(t, err)

	// Verify values
	assert.Equal(t, "anthropic", loader.GetString("provider"))
	assert.Equal(t, "claude-3-opus", loader.GetString("model"))
	assert.Equal(t, 50, loader.GetInt("max_turns"))
}

func TestLoader_EnvironmentVariables(t *testing.T) {
	// Set environment variables
	os.Setenv("SPIN_PROVIDER", "test-provider")
	os.Setenv("SPIN_MAX_TURNS", "75")
	defer func() {
		os.Unsetenv("SPIN_PROVIDER")
		os.Unsetenv("SPIN_MAX_TURNS")
	}()

	// Create loader
	loader := NewLoader()

	// Load (no file)
	_ = loader.Load("")

	// Verify env vars override
	assert.Equal(t, "test-provider", loader.GetString("provider"))
	assert.Equal(t, 75, loader.GetInt("max_turns"))
}

func TestLoader_Defaults(t *testing.T) {
	loader := NewLoader()

	// Set defaults
	loader.SetDefault("custom_key", "default_value")
	loader.SetDefault("custom_int", 42)

	// Load (no file)
	_ = loader.Load("")

	// Verify defaults
	assert.Equal(t, "default_value", loader.GetString("custom_key"))
	assert.Equal(t, 42, loader.GetInt("custom_int"))
}

func TestLoader_Unmarshal(t *testing.T) {
	type TestConfig struct {
		Provider  string `mapstructure:"provider"`
		Model     string `mapstructure:"model"`
		MaxTurns  int    `mapstructure:"max_turns"`
		EnableGit bool   `mapstructure:"enable_git"`
	}

	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "test.yaml")

	yamlContent := `
provider: openai
model: gpt-4
max_turns: 100
enable_git: true
`
	err := os.WriteFile(configFile, []byte(yamlContent), 0644)
	require.NoError(t, err)

	loader := NewLoader()
	err = loader.LoadFromFile(configFile)
	require.NoError(t, err)

	var cfg TestConfig
	err = loader.Unmarshal(&cfg)
	require.NoError(t, err)

	assert.Equal(t, "openai", cfg.Provider)
	assert.Equal(t, "gpt-4", cfg.Model)
	assert.Equal(t, 100, cfg.MaxTurns)
	assert.Equal(t, true, cfg.EnableGit)
}

func TestLoader_LoadFromFile_TOML(t *testing.T) {
	// Create temp TOML file
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "test.toml")

	tomlContent := `
provider = "ollama"
model = "llama3.2"
max_turns = 75
enable_git = false
`
	err := os.WriteFile(configFile, []byte(tomlContent), 0644)
	require.NoError(t, err)

	// Load config
	loader := NewLoader()
	err = loader.LoadFromFile(configFile)
	require.NoError(t, err)

	// Verify values
	assert.Equal(t, "ollama", loader.GetString("provider"))
	assert.Equal(t, "llama3.2", loader.GetString("model"))
	assert.Equal(t, 75, loader.GetInt("max_turns"))
	assert.Equal(t, false, loader.GetBool("enable_git"))
}

func TestLoader_UnsupportedFormat(t *testing.T) {
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "test.xml")

	err := os.WriteFile(configFile, []byte("<config></config>"), 0644)
	require.NoError(t, err)

	loader := NewLoader()
	err = loader.LoadFromFile(configFile)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported config format")
}

func TestLoader_Get(t *testing.T) {
	loader := NewLoader()
	loader.Set("test.key", "value")

	result := loader.Get("test.key")
	assert.Equal(t, "value", result)
}

func TestLoader_GetStringSlice(t *testing.T) {
	loader := NewLoader()
	loader.Set("test.slice", []string{"a", "b", "c"})

	result := loader.GetStringSlice("test.slice")
	assert.Equal(t, []string{"a", "b", "c"}, result)
}

func TestLoader_Set(t *testing.T) {
	loader := NewLoader()
	loader.Set("custom.value", 42)

	assert.Equal(t, 42, loader.GetInt("custom.value"))
}

func TestLoader_UnmarshalKey(t *testing.T) {
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "test.yaml")

	yamlContent := `
llm:
  provider: openai
  model: gpt-4
`
	err := os.WriteFile(configFile, []byte(yamlContent), 0644)
	require.NoError(t, err)

	loader := NewLoader()
	err = loader.LoadFromFile(configFile)
	require.NoError(t, err)

	type LLMConfig struct {
		Provider string `mapstructure:"provider"`
		Model    string `mapstructure:"model"`
	}

	var llm LLMConfig
	err = loader.UnmarshalKey("llm", &llm)
	require.NoError(t, err)
	assert.Equal(t, "openai", llm.Provider)
	assert.Equal(t, "gpt-4", llm.Model)
}

func TestLoader_ConfigFileUsed(t *testing.T) {
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "test.yaml")

	yamlContent := `test: value`
	err := os.WriteFile(configFile, []byte(yamlContent), 0644)
	require.NoError(t, err)

	loader := NewLoader()
	err = loader.LoadFromFile(configFile)
	require.NoError(t, err)

	usedFile := loader.ConfigFileUsed()
	assert.Equal(t, configFile, usedFile)
}

func TestLoader_AllSettings(t *testing.T) {
	loader := NewLoader()
	loader.Set("key1", "value1")
	loader.Set("key2", 42)

	settings := loader.AllSettings()
	assert.Equal(t, "value1", settings["key1"])
	assert.Equal(t, 42, settings["key2"])
}

func TestLoader_IsSet(t *testing.T) {
	loader := NewLoader()
	loader.Set("existing.key", "value")

	assert.True(t, loader.IsSet("existing.key"))
	assert.False(t, loader.IsSet("nonexistent.key"))
}

func TestLoader_WatchConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "test.yaml")

	yamlContent := `test: initial`
	err := os.WriteFile(configFile, []byte(yamlContent), 0644)
	require.NoError(t, err)

	loader := NewLoader()
	err = loader.LoadFromFile(configFile)
	require.NoError(t, err)

	// Test that WatchConfig doesn't panic
	// Note: We can't easily test the actual file watching in a unit test
	// without complex setup, but we can verify the function is callable
	loader.WatchConfig(func() {
		// Callback would be called on config change
	})

	// Just verify the function didn't panic
	assert.NotNil(t, loader)
}
