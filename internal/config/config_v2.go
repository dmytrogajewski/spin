package config

import (
	"fmt"
	"strings"
	"time"
)

// ValidationErrors collects multiple validation errors.
type ValidationErrors struct {
	errors []error
}

// Error implements the error interface.
func (v *ValidationErrors) Error() string {
	if len(v.errors) == 0 {
		return ""
	}
	if len(v.errors) == 1 {
		return v.errors[0].Error()
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("validation failed: %d errors found:\n", len(v.errors)))
	for i, err := range v.errors {
		sb.WriteString(fmt.Sprintf("  %d. %s\n", i+1, err.Error()))
	}
	return strings.TrimSuffix(sb.String(), "\n")
}

// Add adds an error to the collection.
func (v *ValidationErrors) Add(err error) {
	if err != nil {
		v.errors = append(v.errors, err)
	}
}

// HasErrors returns true if there are any errors.
func (v *ValidationErrors) HasErrors() bool {
	return len(v.errors) > 0
}

// ToError returns nil if no errors, otherwise returns the ValidationErrors itself.
func (v *ValidationErrors) ToError() error {
	if !v.HasErrors() {
		return nil
	}
	return v
}

// ConfigV2 is the unified configuration for Spin v2.0.
// This replaces the flat Config structure with organized sections.
type ConfigV2 struct {
	Version  string           `yaml:"version" mapstructure:"version"`
	LLM      LLMConfigV2      `yaml:"llm" mapstructure:"llm"`
	Agent    AgentConfigV2    `yaml:"agent" mapstructure:"agent"`
	ACE      ACEConfigV2      `yaml:"ace" mapstructure:"ace"`
	Security SecurityConfigV2 `yaml:"security" mapstructure:"security"`
	Protocol ProtocolConfigV2 `yaml:"protocol" mapstructure:"protocol"`
}

// LLMConfigV2 configures the LLM provider.
type LLMConfigV2 struct {
	Provider    string        `yaml:"provider" mapstructure:"provider"`
	Model       string        `yaml:"model" mapstructure:"model"`
	Temperature float64       `yaml:"temperature" mapstructure:"temperature"`
	MaxTokens   int           `yaml:"max_tokens" mapstructure:"max_tokens"`
	Timeout     time.Duration `yaml:"timeout" mapstructure:"timeout"`
	BaseURL     string        `yaml:"base_url" mapstructure:"base_url"`
	APIKey      string        `yaml:"api_key" mapstructure:"api_key"`
}

// AgentConfigV2 configures the agent behavior.
type AgentConfigV2 struct {
	MaxTurns        int           `yaml:"max_turns" mapstructure:"max_turns"`
	Timeout         time.Duration `yaml:"timeout" mapstructure:"timeout"`
	WorkDir         string        `yaml:"work_dir" mapstructure:"work_dir"`
	RequireApproval bool          `yaml:"require_approval" mapstructure:"require_approval"`
}

// ACEConfigV2 configures Agentic Context Engineering.
type ACEConfigV2 struct {
	Enabled        bool    `yaml:"enabled" mapstructure:"enabled"`
	PlaybookPath   string  `yaml:"playbook_path" mapstructure:"playbook_path"`
	TrajectoryPath string  `yaml:"trajectory_path" mapstructure:"trajectory_path"`
	TopK           int     `yaml:"top_k" mapstructure:"top_k"`
	MinScore       float64 `yaml:"min_score" mapstructure:"min_score"`
}

// SecurityConfigV2 configures security and sandboxing.
type SecurityConfigV2 struct {
	SandboxMode     string   `yaml:"sandbox_mode" mapstructure:"sandbox_mode"`
	PolicyFile      string   `yaml:"policy_file" mapstructure:"policy_file"`
	AllowedCommands []string `yaml:"allowed_commands" mapstructure:"allowed_commands"`
}

// ProtocolConfigV2 configures protocol features (MCP, Git, Shell).
type ProtocolConfigV2 struct {
	EnableMCP    bool                `yaml:"enable_mcp" mapstructure:"enable_mcp"`
	MCPServers   []MCPServerConfigV2 `yaml:"mcp_servers" mapstructure:"mcp_servers"`
	EnableGit    bool                `yaml:"enable_git" mapstructure:"enable_git"`
	EnableShell  bool                `yaml:"enable_shell" mapstructure:"enable_shell"`
	ShellTimeout time.Duration       `yaml:"shell_timeout" mapstructure:"shell_timeout"`
}

// MCPServerConfigV2 configures an MCP server.
type MCPServerConfigV2 struct {
	Name    string            `yaml:"name" mapstructure:"name"`
	Command string            `yaml:"command" mapstructure:"command"`
	Args    []string          `yaml:"args" mapstructure:"args"`
	Env     map[string]string `yaml:"env" mapstructure:"env"`
}

// Validate performs validation on the config.
func (c *ConfigV2) Validate() error {
	errs := &ValidationErrors{}

	// Validate each section (collect all errors, don't fail fast)
	if err := c.LLM.Validate(); err != nil {
		errs.Add(err)
	}
	if err := c.Agent.Validate(); err != nil {
		errs.Add(err)
	}
	if err := c.ACE.Validate(); err != nil {
		errs.Add(err)
	}
	if err := c.Security.Validate(); err != nil {
		errs.Add(err)
	}
	if err := c.Protocol.Validate(); err != nil {
		errs.Add(err)
	}

	return errs.ToError()
}

// Validate performs validation on the LLM configuration.
func (l *LLMConfigV2) Validate() error {
	errs := &ValidationErrors{}

	// Required fields
	if l.Provider == "" {
		errs.Add(fmt.Errorf("llm: provider is required"))
	}
	if l.Model == "" {
		errs.Add(fmt.Errorf("llm: model is required"))
	}

	// Numeric field ranges
	if l.Temperature < 0 || l.Temperature > 2 {
		errs.Add(fmt.Errorf("llm: temperature must be between 0 and 2, got %.2f", l.Temperature))
	}
	if l.MaxTokens <= 0 {
		errs.Add(fmt.Errorf("llm: max_tokens must be positive, got %d", l.MaxTokens))
	}
	if l.Timeout <= 0 {
		errs.Add(fmt.Errorf("llm: timeout must be positive, got %v", l.Timeout))
	}

	return errs.ToError()
}

// Validate performs validation on the Agent configuration.
func (a *AgentConfigV2) Validate() error {
	errs := &ValidationErrors{}

	// Required fields
	if a.MaxTurns <= 0 {
		errs.Add(fmt.Errorf("agent: max_turns must be positive, got %d", a.MaxTurns))
	}
	if a.Timeout <= 0 {
		errs.Add(fmt.Errorf("agent: timeout must be positive, got %v", a.Timeout))
	}
	if a.WorkDir == "" {
		errs.Add(fmt.Errorf("agent: work_dir is required"))
	}

	return errs.ToError()
}

// Validate performs validation on the ACE configuration.
func (ace *ACEConfigV2) Validate() error {
	// Only validate if ACE is enabled
	if !ace.Enabled {
		return nil
	}

	errs := &ValidationErrors{}

	// Required fields when enabled
	if ace.PlaybookPath == "" {
		errs.Add(fmt.Errorf("ace: playbook_path is required when ACE is enabled"))
	}
	if ace.TrajectoryPath == "" {
		errs.Add(fmt.Errorf("ace: trajectory_path is required when ACE is enabled"))
	}

	// Numeric field ranges
	if ace.TopK <= 0 {
		errs.Add(fmt.Errorf("ace: top_k must be positive, got %d", ace.TopK))
	}
	if ace.MinScore < 0 || ace.MinScore > 1 {
		errs.Add(fmt.Errorf("ace: min_score must be between 0 and 1, got %.2f", ace.MinScore))
	}

	return errs.ToError()
}

// Validate performs validation on the Security configuration.
func (s *SecurityConfigV2) Validate() error {
	// Validate sandbox mode if set
	if s.SandboxMode != "" {
		validModes := map[string]bool{
			"none":     true,
			"docker":   true,
			"firejail": true,
		}
		if !validModes[s.SandboxMode] {
			return fmt.Errorf("security: sandbox_mode must be one of [none, docker, firejail], got %q", s.SandboxMode)
		}
	}

	return nil
}

// Validate performs validation on the Protocol configuration.
func (p *ProtocolConfigV2) Validate() error {
	errs := &ValidationErrors{}

	// Validate shell timeout if shell is enabled
	if p.EnableShell && p.ShellTimeout <= 0 {
		errs.Add(fmt.Errorf("protocol: shell_timeout must be positive when shell is enabled, got %v", p.ShellTimeout))
	}

	// Validate MCP servers if MCP is enabled
	if p.EnableMCP {
		for i, server := range p.MCPServers {
			if err := server.Validate(); err != nil {
				errs.Add(fmt.Errorf("protocol: mcp_servers[%d]: %w", i, err))
			}
		}
	}

	return errs.ToError()
}

// Validate performs validation on the MCP server configuration.
func (m *MCPServerConfigV2) Validate() error {
	errs := &ValidationErrors{}

	if m.Name == "" {
		errs.Add(fmt.Errorf("name is required"))
	}
	if m.Command == "" {
		errs.Add(fmt.Errorf("command is required"))
	}

	return errs.ToError()
}

// DefaultConfigV2 returns a ConfigV2 with sensible defaults.
func DefaultConfigV2() *ConfigV2 {
	return &ConfigV2{
		Version: "2.0",
		LLM: LLMConfigV2{
			Provider:    "ollama",
			Model:       "qwen2.5-coder:7b",
			Temperature: 0.7,
			MaxTokens:   4096,
			Timeout:     5 * time.Minute,
			BaseURL:     "",
			APIKey:      "",
		},
		Agent: AgentConfigV2{
			MaxTurns:        10,
			Timeout:         60 * time.Minute,
			WorkDir:         ".",
			RequireApproval: false,
		},
		ACE: ACEConfigV2{
			Enabled:        true,
			PlaybookPath:   "~/.spin/ace/playbooks/default.json",
			TrajectoryPath: "~/.spin/ace/trajectories/",
			TopK:           5,
			MinScore:       0.3,
		},
		Security: SecurityConfigV2{
			SandboxMode:     "none",
			PolicyFile:      "",
			AllowedCommands: []string{},
		},
		Protocol: ProtocolConfigV2{
			EnableMCP:    false,
			MCPServers:   []MCPServerConfigV2{},
			EnableGit:    true,
			EnableShell:  true,
			ShellTimeout: 30 * time.Second,
		},
	}
}
