package config

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
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
	AgentsMD AgentsMDConfigV2 `yaml:"agents_md" mapstructure:"agents_md"`
	Memory   MemoryConfigV2   `yaml:"memory" mapstructure:"memory"`
}

// AgentsMDConfigV2 configures AGENTS.md project instructions support.
type AgentsMDConfigV2 struct {
	// Enabled controls whether AGENTS.md is loaded (default: true)
	Enabled bool `yaml:"enabled" mapstructure:"enabled"`

	// Path specifies a custom path to AGENTS.md.
	// If empty, auto-discovery is used.
	Path string `yaml:"path" mapstructure:"path"`

	// MaxSize is the maximum file size in bytes (default: 100KB)
	// Files larger than this are truncated with a warning.
	MaxSize int64 `yaml:"max_size" mapstructure:"max_size"`
}

// LLMConfigV2 configures the LLM provider.
type LLMConfigV2 struct {
	Provider       string                 `yaml:"provider" mapstructure:"provider"`
	Model          string                 `yaml:"model" mapstructure:"model"`
	Temperature    float64                `yaml:"temperature" mapstructure:"temperature"`
	MaxTokens      int                    `yaml:"max_tokens" mapstructure:"max_tokens"`
	Timeout        time.Duration          `yaml:"timeout" mapstructure:"timeout"`
	BaseURL        string                 `yaml:"base_url" mapstructure:"base_url"`
	APIKey         string                 `yaml:"api_key" mapstructure:"api_key"`
	ProviderConfig map[string]interface{} `yaml:"provider_config" mapstructure:"provider_config"`

	// ContextWindow is the model's context window size in tokens.
	// If set, this overrides the provider's auto-detected context window.
	// This is useful when using custom or fine-tuned models, or when the
	// provider doesn't know the context window for a particular model.
	// If not set (0), the system will try to auto-detect from the provider.
	ContextWindow int `yaml:"context_window" mapstructure:"context_window"`
}

// AgentConfigV2 configures the agent behavior.
type AgentConfigV2 struct {
	MaxTurns        int           `yaml:"max_turns" mapstructure:"max_turns"`
	Timeout         time.Duration `yaml:"timeout" mapstructure:"timeout"`
	WorkDir         string        `yaml:"work_dir" mapstructure:"work_dir"`
	RequireApproval bool          `yaml:"require_approval" mapstructure:"require_approval"`

	// Performance Configuration
	StreamBuffer  int  `yaml:"stream_buffer" mapstructure:"stream_buffer"`
	CacheCommands bool `yaml:"cache_commands" mapstructure:"cache_commands"`

	// Environment Configuration
	MaxFiles int  `yaml:"max_files" mapstructure:"max_files"`
	MaxDepth int  `yaml:"max_depth" mapstructure:"max_depth"`
	SkipGit  bool `yaml:"skip_git" mapstructure:"skip_git"`

	// Storage Configuration
	SessionDir   string `yaml:"session_dir" mapstructure:"session_dir"`
	HistoryLimit int    `yaml:"history_limit" mapstructure:"history_limit"`

	// Logging Configuration
	LogLevel  string `yaml:"log_level" mapstructure:"log_level"`   // debug, info, warn, error
	LogFormat string `yaml:"log_format" mapstructure:"log_format"` // text, json
	Debug     bool   `yaml:"debug" mapstructure:"debug"`           // Enable debug mode

	// Cycle Detection Configuration
	CycleDetection CycleDetectionConfigV2 `yaml:"cycle_detection" mapstructure:"cycle_detection"`
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
	// Approval persistence feature flag and TTLs
	ApprovalPersistenceEnabled bool `yaml:"approval_persistence_enabled" mapstructure:"approval_persistence_enabled"`
	// Approval persistence TTLs
	SessionPolicyTTL time.Duration `yaml:"session_policy_ttl" mapstructure:"session_policy_ttl"`
	GlobalPolicyTTL  time.Duration `yaml:"global_policy_ttl" mapstructure:"global_policy_ttl"`
}

// ProtocolConfigV2 configures protocol features (MCP, Git, Shell).
type ProtocolConfigV2 struct {
	EnableMCP    bool                `yaml:"enable_mcp" mapstructure:"enable_mcp"`
	MCPServers   []MCPServerConfigV2 `yaml:"mcp_servers" mapstructure:"mcp_servers"`
	EnableGit    bool                `yaml:"enable_git" mapstructure:"enable_git"`
	EnableShell  bool                `yaml:"enable_shell" mapstructure:"enable_shell"`
	ShellTimeout time.Duration       `yaml:"shell_timeout" mapstructure:"shell_timeout"`
}

// MCPTransportType defines the MCP server connection transport.
type MCPTransportType string

// MCP transport type constants.
const (
	// MCPTransportStdio uses stdio for local process communication.
	MCPTransportStdio MCPTransportType = "stdio"

	// MCPTransportSSE uses Server-Sent Events for remote communication.
	MCPTransportSSE MCPTransportType = "sse"

	// MCPTransportStreamableHTTP uses HTTP streaming for remote communication.
	MCPTransportStreamableHTTP MCPTransportType = "streamable-http"

	// MCPTransportSmithery uses Smithery's connection-based API.
	MCPTransportSmithery MCPTransportType = "smithery"
)

// MCPOAuthConfigV2 holds OAuth configuration for protected MCP servers.
type MCPOAuthConfigV2 struct {
	ClientID     string   `yaml:"client_id" mapstructure:"client_id"`
	ClientSecret string   `yaml:"client_secret,omitempty" mapstructure:"client_secret"`
	RedirectURL  string   `yaml:"redirect_url,omitempty" mapstructure:"redirect_url"`
	Scopes       []string `yaml:"scopes,omitempty" mapstructure:"scopes"`
}

// MCPServerConfigV2 configures an MCP server.
type MCPServerConfigV2 struct {
	// Common fields
	Name      string           `yaml:"name" mapstructure:"name"`
	Transport MCPTransportType `yaml:"transport,omitempty" mapstructure:"transport"`

	// Stdio transport fields (mutually exclusive with URL)
	Command string            `yaml:"command,omitempty" mapstructure:"command"`
	Args    []string          `yaml:"args,omitempty" mapstructure:"args"`
	Env     map[string]string `yaml:"env,omitempty" mapstructure:"env"`

	// Remote transport fields (mutually exclusive with Command)
	URL     string            `yaml:"url,omitempty" mapstructure:"url"`
	Headers map[string]string `yaml:"headers,omitempty" mapstructure:"headers"`

	// OAuth configuration (optional, for protected servers)
	OAuth *MCPOAuthConfigV2 `yaml:"oauth,omitempty" mapstructure:"oauth"`

	// Smithery-specific fields
	SmitheryAPIKey    string `yaml:"smithery_api_key,omitempty" mapstructure:"smithery_api_key"`
	SmitheryNamespace string `yaml:"smithery_namespace,omitempty" mapstructure:"smithery_namespace"`
}

// CycleDetectionConfigV2 configures automatic cycle detection and intervention.
type CycleDetectionConfigV2 struct {
	// Enabled controls whether cycle detection is active (default: true)
	Enabled bool `yaml:"enabled" mapstructure:"enabled"`

	// WindowSize is the number of snapshots to compare for pattern detection (default: 3)
	WindowSize int `yaml:"window_size" mapstructure:"window_size"`

	// SimilarityThresh is the threshold for response similarity detection (default: 0.8)
	SimilarityThresh float64 `yaml:"similarity_thresh" mapstructure:"similarity_thresh"`

	// ToolRepeatLimit is the max identical tool calls before triggering cycle (default: 3)
	ToolRepeatLimit int `yaml:"tool_repeat_limit" mapstructure:"tool_repeat_limit"`

	// ErrorRepeatLimit is the max identical errors before triggering cycle (default: 3)
	ErrorRepeatLimit int `yaml:"error_repeat_limit" mapstructure:"error_repeat_limit"`
}

// MemoryConfigV2 configures context offloading memory storage.
type MemoryConfigV2 struct {
	// Scratchpad configures session-scoped ephemeral memory
	Scratchpad ScratchpadConfigV2 `yaml:"scratchpad" mapstructure:"scratchpad"`

	// Persistent configures cross-session persistent memory
	Persistent PersistentMemoryConfigV2 `yaml:"persistent" mapstructure:"persistent"`
}

// ScratchpadConfigV2 configures the session-scoped scratchpad.
type ScratchpadConfigV2 struct {
	// Enabled controls whether scratchpad is available (default: true)
	Enabled bool `yaml:"enabled" mapstructure:"enabled"`

	// MaxEntries is the maximum number of entries (default: 50)
	MaxEntries int `yaml:"max_entries" mapstructure:"max_entries"`

	// AutoEvict enables automatic LRU eviction (default: true)
	AutoEvict bool `yaml:"auto_evict" mapstructure:"auto_evict"`
}

// PersistentMemoryConfigV2 configures cross-session persistent memory.
type PersistentMemoryConfigV2 struct {
	// Enabled controls whether persistent memory is available (default: false)
	Enabled bool `yaml:"enabled" mapstructure:"enabled"`

	// BasePath is the directory for persistent memory storage (default: ~/.spin/memory)
	BasePath string `yaml:"base_path" mapstructure:"base_path"`
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
	if err := c.AgentsMD.Validate(); err != nil {
		errs.Add(err)
	}
	if err := c.Memory.Validate(); err != nil {
		errs.Add(err)
	}

	return errs.ToError()
}

// Validate performs validation on the AgentsMD configuration.
func (a *AgentsMDConfigV2) Validate() error {
	// MaxSize validation: if negative, treat as no limit
	if a.MaxSize < 0 {
		a.MaxSize = 0
	}
	return nil
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
			"none":           true,
			"workspace-only": true,
			"docker":         true,
			"firejail":       true,
		}
		if !validModes[s.SandboxMode] {
			return fmt.Errorf("security: sandbox_mode must be one of [none, workspace-only, docker, firejail], got %q", s.SandboxMode)
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

	// Name is always required
	if m.Name == "" {
		errs.Add(fmt.Errorf("name is required"))
	}

	// Validate transport type
	if !m.Transport.IsValid() {
		errs.Add(fmt.Errorf("invalid transport: %s", m.Transport))
		return errs.ToError()
	}

	// Determine effective transport (empty defaults to stdio)
	transport := m.Transport
	if transport == "" {
		transport = MCPTransportStdio
	}

	// Validate based on transport type
	if transport.IsRemote() {
		m.validateRemote(transport, errs)
	} else {
		m.validateStdio(errs)
	}

	return errs.ToError()
}

// IsValid returns true if the transport type is valid.
func (t MCPTransportType) IsValid() bool {
	switch t {
	case "", MCPTransportStdio, MCPTransportSSE, MCPTransportStreamableHTTP, MCPTransportSmithery:
		return true
	default:
		return false
	}
}

// IsRemote returns true if the transport requires a remote URL.
func (t MCPTransportType) IsRemote() bool {
	switch t {
	case MCPTransportSSE, MCPTransportStreamableHTTP, MCPTransportSmithery:
		return true
	default:
		return false
	}
}

// validateStdio validates stdio transport configuration.
func (m *MCPServerConfigV2) validateStdio(errs *ValidationErrors) {
	// Command is required for stdio
	if m.Command == "" {
		errs.Add(fmt.Errorf("command is required for stdio transport"))
	}

	// URL is not allowed for stdio
	if m.URL != "" {
		errs.Add(fmt.Errorf("url is not allowed for stdio transport"))
	}

	// OAuth is not allowed for stdio
	if m.OAuth != nil {
		errs.Add(fmt.Errorf("oauth is not allowed for stdio transport"))
	}
}

// validateRemote validates remote transport configuration.
func (m *MCPServerConfigV2) validateRemote(transport MCPTransportType, errs *ValidationErrors) {
	// URL is required for remote transports
	if m.URL == "" {
		errs.Add(fmt.Errorf("url is required for %s transport", transport))
	} else {
		// Validate URL format
		parsedURL, err := url.Parse(m.URL)
		if err != nil || parsedURL.Scheme == "" || parsedURL.Host == "" {
			errs.Add(fmt.Errorf("invalid url: %s", m.URL))
		}
	}

	// Command is not allowed for remote transports
	if m.Command != "" {
		errs.Add(fmt.Errorf("command is not allowed for remote transport"))
	}

	// Validate OAuth if provided
	if m.OAuth != nil {
		if m.OAuth.ClientID == "" {
			errs.Add(fmt.Errorf("oauth client_id is required"))
		}
	}
}

// Validate performs validation on the Memory configuration.
func (m *MemoryConfigV2) Validate() error {
	errs := &ValidationErrors{}

	// Validate scratchpad config
	if err := m.Scratchpad.Validate(); err != nil {
		errs.Add(err)
	}

	// Validate persistent config
	if err := m.Persistent.Validate(); err != nil {
		errs.Add(err)
	}

	return errs.ToError()
}

// Validate performs validation on the Scratchpad configuration.
func (s *ScratchpadConfigV2) Validate() error {
	// Only validate if scratchpad is enabled
	if !s.Enabled {
		return nil
	}

	errs := &ValidationErrors{}

	if s.MaxEntries <= 0 {
		errs.Add(fmt.Errorf("memory.scratchpad: max_entries must be positive, got %d", s.MaxEntries))
	}

	return errs.ToError()
}

// Validate performs validation on the PersistentMemory configuration.
func (p *PersistentMemoryConfigV2) Validate() error {
	// Only validate if persistent memory is enabled
	if !p.Enabled {
		return nil
	}

	errs := &ValidationErrors{}

	if p.BasePath == "" {
		errs.Add(fmt.Errorf("memory.persistent: base_path is required when persistent memory is enabled"))
	}

	return errs.ToError()
}

// DefaultConfigV2 returns a ConfigV2 with sensible defaults.
func DefaultConfigV2() *ConfigV2 {
	// Derive default policy file path under user config directory
	policyFile := ""
	if cfgDir, err := os.UserConfigDir(); err == nil && cfgDir != "" {
		policyFile = filepath.Join(cfgDir, "spin", "policies.json")
	}
	return &ConfigV2{
		Version: "2.0",
		LLM: LLMConfigV2{
			Provider:       "ollama",
			Model:          "qwen2.5-coder:7b",
			Temperature:    0.7,
			MaxTokens:      8192,
			Timeout:        5 * time.Minute,
			BaseURL:        "", // Empty - provider will use its own default
			APIKey:         "",
			ProviderConfig: make(map[string]interface{}),
		},
		Agent: AgentConfigV2{
			MaxTurns:        50,
			Timeout:         60 * time.Minute,
			WorkDir:         ".",
			RequireApproval: false,
			StreamBuffer:    100,
			CacheCommands:   false,
			MaxFiles:        0,
			MaxDepth:        0,
			SkipGit:         false,
			SessionDir:      "~/.spin/sessions",
			HistoryLimit:    1000,
			LogLevel:        "info",
			LogFormat:       "text",
			Debug:           false,
			CycleDetection: CycleDetectionConfigV2{
				Enabled:          true,
				WindowSize:       3,
				SimilarityThresh: 0.8,
				ToolRepeatLimit:  3,
				ErrorRepeatLimit: 3,
			},
		},
		ACE: ACEConfigV2{
			Enabled:        false,
			PlaybookPath:   "~/.spin/ace/playbooks/default.json",
			TrajectoryPath: "~/.spin/ace/trajectories/",
			TopK:           5,
			MinScore:       0.3,
		},
		Security: SecurityConfigV2{
			SandboxMode:                "workspace-only",
			PolicyFile:                 policyFile,
			AllowedCommands:            []string{},
			ApprovalPersistenceEnabled: true,
			SessionPolicyTTL:           8 * time.Hour,
			GlobalPolicyTTL:            30 * 24 * time.Hour,
		},
		Protocol: ProtocolConfigV2{
			EnableMCP:    false,
			MCPServers:   []MCPServerConfigV2{},
			EnableGit:    true,
			EnableShell:  true,
			ShellTimeout: 5 * time.Minute,
		},
		AgentsMD: AgentsMDConfigV2{
			Enabled: true,
			Path:    "",         // Auto-discover
			MaxSize: 100 * 1024, // 100KB
		},
		Memory: MemoryConfigV2{
			Scratchpad: ScratchpadConfigV2{
				Enabled:    true,
				MaxEntries: 50,
				AutoEvict:  true,
			},
			Persistent: PersistentMemoryConfigV2{
				Enabled:  false,
				BasePath: "~/.spin/memory",
			},
		},
	}
}
