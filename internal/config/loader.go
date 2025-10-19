// Package config provides configuration management using Viper.
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
)

// Loader handles configuration loading using Viper.
type Loader struct {
	v *viper.Viper
}

// NewLoader creates a new configuration loader with Viper.
func NewLoader() *Loader {
	v := viper.New()

	// Set config name (Viper will look for spin.yaml, spin.yml, spin.json, spin.toml)
	// Note: Using SetConfigName without extension to support multiple formats
	v.SetConfigName("spin")
	v.SetConfigType("yaml") // Default type

	// Add config search paths in order of precedence
	v.AddConfigPath(".")           // Current directory
	v.AddConfigPath("$HOME/.spin") // Home directory
	v.AddConfigPath("/etc/spin")   // System directory

	// Environment variable support
	v.SetEnvPrefix("SPIN") // Prefix for env vars (SPIN_*)
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))
	v.AutomaticEnv() // Read env vars automatically

	return &Loader{v: v}
}

// Load loads configuration from file and environment.
// If path is empty, searches default locations.
// Supports YAML, JSON, and TOML formats.
func (l *Loader) Load(path string) error {
	if path != "" {
		l.v.SetConfigFile(path)
	}

	if err := l.v.ReadInConfig(); err != nil {
		return l.handleConfigReadError(err)
	}

	return nil
}

// handleConfigReadError handles errors when reading configuration files.
func (l *Loader) handleConfigReadError(err error) error {
	if _, ok := err.(viper.ConfigFileNotFoundError); ok {
		return nil // Config file not found is acceptable - use defaults + env vars
	}

	if l.isInvalidConfigFile() {
		l.reinitializeViper()
		return nil
	}

	return fmt.Errorf("failed to read config: %w", err)
}

// isInvalidConfigFile checks if the error is due to an invalid config file.
func (l *Loader) isInvalidConfigFile() bool {
	configFile := l.v.ConfigFileUsed()
	if configFile == "" {
		return false
	}

	ext := filepath.Ext(configFile)
	return ext == "" || !l.isValidConfigExtension(ext)
}

// isValidConfigExtension checks if the file extension is valid for config files.
func (l *Loader) isValidConfigExtension(ext string) bool {
	validExts := []string{".yaml", ".yml", ".json", ".toml"}
	for _, validExt := range validExts {
		if ext == validExt {
			return true
		}
	}
	return false
}

// reinitializeViper reinitializes Viper with default settings.
func (l *Loader) reinitializeViper() {
	l.v = viper.New()
	l.v.SetEnvPrefix("SPIN")
	l.v.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))
	l.v.AutomaticEnv()
}

// LoadFromFile loads configuration from a specific file.
func (l *Loader) LoadFromFile(path string) error {
	// Check file exists
	if _, err := os.Stat(path); err != nil {
		return fmt.Errorf("config file not found: %w", err)
	}

	// Detect format from extension
	ext := filepath.Ext(path)
	switch ext {
	case ".yaml", ".yml":
		l.v.SetConfigType("yaml")
	case ".json":
		l.v.SetConfigType("json")
	case ".toml":
		l.v.SetConfigType("toml")
	default:
		return fmt.Errorf("unsupported config format: %s", ext)
	}

	l.v.SetConfigFile(path)
	return l.v.ReadInConfig()
}

// Get retrieves a value by key.
func (l *Loader) Get(key string) interface{} {
	return l.v.Get(key)
}

// GetString retrieves a string value.
func (l *Loader) GetString(key string) string {
	return l.v.GetString(key)
}

// GetInt retrieves an integer value.
func (l *Loader) GetInt(key string) int {
	return l.v.GetInt(key)
}

// GetBool retrieves a boolean value.
func (l *Loader) GetBool(key string) bool {
	return l.v.GetBool(key)
}

// Set sets a configuration value.
func (l *Loader) Set(key string, value interface{}) {
	l.v.Set(key, value)
}

// Unmarshal unmarshals configuration into a struct.
func (l *Loader) Unmarshal(rawVal interface{}) error {
	return l.v.Unmarshal(rawVal)
}

// UnmarshalKey unmarshals a specific key into a struct.
func (l *Loader) UnmarshalKey(key string, rawVal interface{}) error {
	return l.v.UnmarshalKey(key, rawVal)
}

// ConfigFileUsed returns the config file being used.
func (l *Loader) ConfigFileUsed() string {
	return l.v.ConfigFileUsed()
}

// AllSettings returns all settings as a map.
func (l *Loader) AllSettings() map[string]interface{} {
	return l.v.AllSettings()
}

// IsSet checks if a key is set.
func (l *Loader) IsSet(key string) bool {
	return l.v.IsSet(key)
}
