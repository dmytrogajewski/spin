package config

import (
	"fmt"
	"time"
)

// ConfigV2 is the unified configuration for Spin v2.0.
// This replaces the flat Config structure with organized sections.
type ConfigV2 struct {
	Version string        `yaml:"version"`
	LLM     LLMConfigV2   `yaml:"llm"`
	Agent   AgentConfigV2 `yaml:"agent"`
	ACE     ACEConfigV2   `yaml:"ace"`
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

// AgentConfigV2 configures the agent behavior.
type AgentConfigV2 struct {
	MaxTurns        int           `yaml:"max_turns"`
	Timeout         time.Duration `yaml:"timeout"`
	WorkDir         string        `yaml:"work_dir"`
	RequireApproval bool          `yaml:"require_approval"`
}

// ACEConfigV2 configures Agentic Context Engineering.
type ACEConfigV2 struct {
	Enabled        bool    `yaml:"enabled"`
	PlaybookPath   string  `yaml:"playbook_path"`
	TrajectoryPath string  `yaml:"trajectory_path"`
	TopK           int     `yaml:"top_k"`
	MinScore       float64 `yaml:"min_score"`
}

// Validate performs validation on the config.
func (c *ConfigV2) Validate() error {
	// Validate each section
	if err := c.LLM.Validate(); err != nil {
		return err
	}
	if err := c.Agent.Validate(); err != nil {
		return err
	}
	if err := c.ACE.Validate(); err != nil {
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

// Validate performs validation on the Agent configuration.
func (a *AgentConfigV2) Validate() error {
	// Required fields
	if a.MaxTurns <= 0 {
		return fmt.Errorf("agent: max_turns must be positive, got %d", a.MaxTurns)
	}
	if a.Timeout <= 0 {
		return fmt.Errorf("agent: timeout must be positive, got %v", a.Timeout)
	}
	if a.WorkDir == "" {
		return fmt.Errorf("agent: work_dir is required")
	}

	return nil
}

// Validate performs validation on the ACE configuration.
func (ace *ACEConfigV2) Validate() error {
	// Only validate if ACE is enabled
	if !ace.Enabled {
		return nil
	}

	// Required fields when enabled
	if ace.PlaybookPath == "" {
		return fmt.Errorf("ace: playbook_path is required when ACE is enabled")
	}
	if ace.TrajectoryPath == "" {
		return fmt.Errorf("ace: trajectory_path is required when ACE is enabled")
	}

	// Numeric field ranges
	if ace.TopK <= 0 {
		return fmt.Errorf("ace: top_k must be positive, got %d", ace.TopK)
	}
	if ace.MinScore < 0 || ace.MinScore > 1 {
		return fmt.Errorf("ace: min_score must be between 0 and 1, got %.2f", ace.MinScore)
	}

	return nil
}
