package config

import (
	"fmt"
	"time"
)

// ConfigV2 is the unified configuration for Spin v2.0.
// This replaces the flat Config structure with organized sections.
type ConfigV2 struct {
	Version string      `yaml:"version"`
	LLM     LLMConfigV2 `yaml:"llm"`
}

// LLMConfigV2 configures the LLM provider.
type LLMConfigV2 struct {
	Provider    string        `yaml:"provider"`
	Model       string        `yaml:"model"`
	Temperature float64       `yaml:"temperature"`
	MaxTokens   int           `yaml:"max_tokens"`
	Timeout     time.Duration `yaml:"timeout"`
	BaseURL     string        `yaml:"base_url"`
	APIKey      string        `yaml:"api_key"`
}

// Validate performs validation on the config.
func (c *ConfigV2) Validate() error {
	// Validate each section
	if err := c.LLM.Validate(); err != nil {
		return err
	}

	return nil
}

// Validate performs validation on the LLM configuration.
func (l *LLMConfigV2) Validate() error {
	// Required fields
	if l.Provider == "" {
		return fmt.Errorf("llm: provider is required")
	}
	if l.Model == "" {
		return fmt.Errorf("llm: model is required")
	}

	// Numeric field ranges
	if l.Temperature < 0 || l.Temperature > 2 {
		return fmt.Errorf("llm: temperature must be between 0 and 2, got %.2f", l.Temperature)
	}
	if l.MaxTokens <= 0 {
		return fmt.Errorf("llm: max_tokens must be positive, got %d", l.MaxTokens)
	}
	if l.Timeout <= 0 {
		return fmt.Errorf("llm: timeout must be positive, got %v", l.Timeout)
	}

	return nil
}
