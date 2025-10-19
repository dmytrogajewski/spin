package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewLoader(t *testing.T) {
	loader := NewLoader()
	if loader == nil {
		t.Errorf("NewLoader() returned nil")
	}
}

func TestLoader_LoadFromFile(t *testing.T) {
	// Create a temporary config file
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "test.yaml")

	configContent := `
api_key: "test-key"
base_url: "https://api.example.com"
timeout: 30
debug: true
`

	err := os.WriteFile(configFile, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test config file: %v", err)
	}

	loader := NewLoader()
	err = loader.LoadFromFile(configFile)
	if err != nil {
		t.Errorf("LoadFromFile() error = %v", err)
	}

	// Test that values were loaded
	apiKey := loader.Get("api_key")
	if apiKey != "test-key" {
		t.Errorf("Get(\"api_key\") = %v, want %v", apiKey, "test-key")
	}

	baseURL := loader.Get("base_url")
	if baseURL != "https://api.example.com" {
		t.Errorf("Get(\"base_url\") = %v, want %v", baseURL, "https://api.example.com")
	}

	timeout := loader.Get("timeout")
	if timeout != 30 {
		t.Errorf("Get(\"timeout\") = %v, want %v", timeout, 30)
	}

	debug := loader.Get("debug")
	if debug != true {
		t.Errorf("Get(\"debug\") = %v, want %v", debug, true)
	}
}

func TestLoader_LoadFromFile_Nonexistent(t *testing.T) {
	loader := NewLoader()
	err := loader.LoadFromFile("/nonexistent/file.yaml")
	if err == nil {
		t.Errorf("LoadFromFile() expected error for nonexistent file")
	}
}

func TestLoader_LoadFromFile_InvalidYAML(t *testing.T) {
	// Create a temporary config file with invalid YAML
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "invalid.yaml")

	invalidContent := `
api_key: "test-key"
base_url: "https://api.example.com"
timeout: 30
debug: true
invalid: [unclosed bracket
`

	err := os.WriteFile(configFile, []byte(invalidContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test config file: %v", err)
	}

	loader := NewLoader()
	err = loader.LoadFromFile(configFile)
	if err == nil {
		t.Errorf("LoadFromFile() expected error for invalid YAML")
	}
}

func TestLoader_LoadFromFile_JSON(t *testing.T) {
	// Create a temporary JSON config file
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "test.json")

	configContent := `{
	"api_key": "test-key",
	"base_url": "https://api.example.com",
	"timeout": 30,
	"debug": true
}`

	err := os.WriteFile(configFile, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test config file: %v", err)
	}

	loader := NewLoader()
	err = loader.LoadFromFile(configFile)
	if err != nil {
		t.Errorf("LoadFromFile() error = %v", err)
	}

	// Test that values were loaded
	apiKey := loader.Get("api_key")
	if apiKey != "test-key" {
		t.Errorf("Get(\"api_key\") = %v, want %v", apiKey, "test-key")
	}
}

func TestLoader_LoadFromFile_TOML(t *testing.T) {
	// Create a temporary TOML config file
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "test.toml")

	configContent := `
api_key = "test-key"
base_url = "https://api.example.com"
timeout = 30
debug = true
`

	err := os.WriteFile(configFile, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test config file: %v", err)
	}

	loader := NewLoader()
	err = loader.LoadFromFile(configFile)
	if err != nil {
		t.Errorf("LoadFromFile() error = %v", err)
	}

	// Test that values were loaded
	apiKey := loader.Get("api_key")
	if apiKey != "test-key" {
		t.Errorf("Get(\"api_key\") = %v, want %v", apiKey, "test-key")
	}
}

func TestLoader_Get(t *testing.T) {
	loader := NewLoader()

	// Test getting a non-existent key
	value := loader.Get("nonexistent")
	if value != nil {
		t.Errorf("Get(\"nonexistent\") = %v, want %v", value, nil)
	}
}

func TestLoader_Set(t *testing.T) {
	loader := NewLoader()

	// Set a value
	loader.Set("test_key", "test_value")

	// Get the value back
	value := loader.Get("test_key")
	if value != "test_value" {
		t.Errorf("Get(\"test_key\") = %v, want %v", value, "test_value")
	}
}

func TestLoader_SetDefault(t *testing.T) {
	loader := NewLoader()

	// Set a default value
	loader.SetDefault("default_key", "default_value")

	// Get the default value
	value := loader.Get("default_key")
	if value != "default_value" {
		t.Errorf("Get(\"default_key\") = %v, want %v", value, "default_value")
	}

	// Set a new value - should override default
	loader.Set("default_key", "new_value")
	value = loader.Get("default_key")
	if value != "new_value" {
		t.Errorf("Get(\"default_key\") = %v, want %v", value, "new_value")
	}
}

func TestLoader_Unmarshal(t *testing.T) {
	loader := NewLoader()

	// Set some values
	loader.Set("api_key", "test-key")
	loader.Set("base_url", "https://api.example.com")
	loader.Set("timeout", 30)
	loader.Set("debug", true)

	// Define a struct to unmarshal into
	type Config struct {
		APIKey  string `mapstructure:"api_key"`
		BaseURL string `mapstructure:"base_url"`
		Timeout int    `mapstructure:"timeout"`
		Debug   bool   `mapstructure:"debug"`
	}

	var config Config
	err := loader.Unmarshal(&config)
	if err != nil {
		t.Errorf("Unmarshal() error = %v", err)
	}

	if config.APIKey != "test-key" {
		t.Errorf("config.APIKey = %v, want %v", config.APIKey, "test-key")
	}

	if config.BaseURL != "https://api.example.com" {
		t.Errorf("config.BaseURL = %v, want %v", config.BaseURL, "https://api.example.com")
	}

	if config.Timeout != 30 {
		t.Errorf("config.Timeout = %v, want %v", config.Timeout, 30)
	}

	if config.Debug != true {
		t.Errorf("config.Debug = %v, want %v", config.Debug, true)
	}
}

func TestLoader_UnmarshalKey(t *testing.T) {
	loader := NewLoader()

	// Set some values
	loader.Set("database.host", "localhost")
	loader.Set("database.port", 5432)
	loader.Set("database.name", "testdb")

	// Define a struct to unmarshal into
	type DatabaseConfig struct {
		Host string `mapstructure:"host"`
		Port int    `mapstructure:"port"`
		Name string `mapstructure:"name"`
	}

	var dbConfig DatabaseConfig
	err := loader.UnmarshalKey("database", &dbConfig)
	if err != nil {
		t.Errorf("UnmarshalKey() error = %v", err)
	}

	if dbConfig.Host != "localhost" {
		t.Errorf("dbConfig.Host = %v, want %v", dbConfig.Host, "localhost")
	}

	if dbConfig.Port != 5432 {
		t.Errorf("dbConfig.Port = %v, want %v", dbConfig.Port, 5432)
	}

	if dbConfig.Name != "testdb" {
		t.Errorf("dbConfig.Name = %v, want %v", dbConfig.Name, "testdb")
	}
}

func TestLoader_AllSettings(t *testing.T) {
	loader := NewLoader()

	// Set some values
	loader.Set("key1", "value1")
	loader.Set("key2", 42)
	loader.Set("key3", true)

	// Get all settings
	settings := loader.AllSettings()

	if len(settings) != 3 {
		t.Errorf("AllSettings() length = %v, want %v", len(settings), 3)
	}

	if settings["key1"] != "value1" {
		t.Errorf("settings[\"key1\"] = %v, want %v", settings["key1"], "value1")
	}

	if settings["key2"] != 42 {
		t.Errorf("settings[\"key2\"] = %v, want %v", settings["key2"], 42)
	}

	if settings["key3"] != true {
		t.Errorf("settings[\"key3\"] = %v, want %v", settings["key3"], true)
	}
}

func TestLoader_LoadFromFile_MultipleFiles(t *testing.T) {
	// Create multiple config files
	tempDir := t.TempDir()

	// Base config
	baseConfig := filepath.Join(tempDir, "base.yaml")
	baseContent := `
api_key: "base-key"
base_url: "https://base.example.com"
timeout: 30
`
	err := os.WriteFile(baseConfig, []byte(baseContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create base config file: %v", err)
	}

	// Override config
	overrideConfig := filepath.Join(tempDir, "override.yaml")
	overrideContent := `
api_key: "override-key"
debug: true
`
	err = os.WriteFile(overrideConfig, []byte(overrideContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create override config file: %v", err)
	}

	loader := NewLoader()

	// Load base config
	err = loader.LoadFromFile(baseConfig)
	if err != nil {
		t.Errorf("LoadFromFile(base) error = %v", err)
	}

	// Load override config (this replaces the base config)
	err = loader.LoadFromFile(overrideConfig)
	if err != nil {
		t.Errorf("LoadFromFile(override) error = %v", err)
	}

	// Check that override values are present
	apiKey := loader.Get("api_key")
	if apiKey != "override-key" {
		t.Errorf("Get(\"api_key\") = %v, want %v", apiKey, "override-key")
	}

	// Check that base values are NOT preserved (LoadFromFile replaces config)
	baseURL := loader.Get("base_url")
	if baseURL != nil {
		t.Errorf("Get(\"base_url\") = %v, want %v (should be nil because config was replaced)", baseURL, nil)
	}

	// Check that new values are added
	debug := loader.Get("debug")
	if debug != true {
		t.Errorf("Get(\"debug\") = %v, want %v", debug, true)
	}
}

func TestLoader_Concurrency(t *testing.T) {
	loader := NewLoader()

	// Test concurrent operations
	done := make(chan bool, 10)

	for i := 0; i < 10; i++ {
		go func(i int) {
			key := "key" + string(rune('0'+i))
			value := "value" + string(rune('0'+i))

			loader.Set(key, value)
			result := loader.Get(key)

			if result != value {
				t.Errorf("Concurrent Get(%s) = %v, want %v", key, result, value)
			}

			done <- true
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}
}
