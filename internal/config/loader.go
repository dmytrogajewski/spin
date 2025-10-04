// Package config provides configuration management using Viper.
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
)

// Loader handles configuration loading using Viper.
type Loader struct {
	v *viper.Viper
}

// NewLoader creates a new configuration loader with Viper.
func NewLoader() *Loader {
	v := viper.New()

	// Set config name and type
	v.SetConfigName("spin")
	v.SetConfigType("yaml")

	// Add config search paths
	v.AddConfigPath(".")                    // Current directory
	v.AddConfigPath("$HOME/.spin")          // Home directory
	v.AddConfigPath("/etc/spin")            // System directory

	// Environment variable support
	v.SetEnvPrefix("SPIN")                  // Prefix for env vars (SPIN_*)
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))
	v.AutomaticEnv()                        // Read env vars automatically

	return &Loader{v: v}
}

// Load loads configuration from file and environment.
// If path is empty, searches default locations.
// Supports YAML, JSON, and TOML formats.
func (l *Loader) Load(path string) error {
	if path != "" {
		// Explicit path provided
		l.v.SetConfigFile(path)
	}

	// Read config file
	if err := l.v.ReadInConfig(); err != nil {
		// Config file not found is acceptable - use defaults + env vars
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return fmt.Errorf("failed to read config: %w", err)
		}
	}

	return nil
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

// GetStringSlice retrieves a string slice value.
func (l *Loader) GetStringSlice(key string) []string {
	return l.v.GetStringSlice(key)
}

// Set sets a configuration value.
func (l *Loader) Set(key string, value interface{}) {
	l.v.Set(key, value)
}

// SetDefault sets a default value for a key.
func (l *Loader) SetDefault(key string, value interface{}) {
	l.v.SetDefault(key, value)
}

// Unmarshal unmarshals configuration into a struct.
func (l *Loader) Unmarshal(rawVal interface{}) error {
	return l.v.Unmarshal(rawVal)
}

// UnmarshalKey unmarshals a specific key into a struct.
func (l *Loader) UnmarshalKey(key string, rawVal interface{}) error {
	return l.v.UnmarshalKey(key, rawVal)
}

// WatchConfig enables live config reloading.
// The onChange callback is called when config changes.
func (l *Loader) WatchConfig(onChange func()) {
	l.v.WatchConfig()
	l.v.OnConfigChange(func(e fsnotify.Event) {
		if onChange != nil {
			onChange()
		}
	})
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
