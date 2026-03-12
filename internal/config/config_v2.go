// Package config provides configuration management.
package config

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

var (
	ErrLlmProviderIsRequired               = errors.New("llm: provider is required")
	ErrLlmModelIsRequired                  = errors.New("llm: model is required")
	ErrLlmTemperatureRange                 = errors.New("llm: temperature must be between 0 and 2")
	ErrLlmMaxTokensPositive                = errors.New("llm: max_tokens must be positive")
	ErrLlmTimeoutPositive                  = errors.New("llm: timeout must be positive")
	ErrAgentMaxTurnsPositive               = errors.New("agent: max_turns must be positive")
	ErrAgentTimeoutPositive                = errors.New("agent: timeout must be positive")
	ErrAgentWorkDirIsRequired              = errors.New("agent: work_dir is required")
	ErrAcePlaybookPathRequired             = errors.New("ace: playbook_path is required when ACE is enabled")
	ErrAceTrajectoryPathRequired           = errors.New("ace: trajectory_path is required when ACE is enabled")
	ErrAceTopKPositive                     = errors.New("ace: top_k must be positive")
	ErrAceMinScoreRange                    = errors.New("ace: min_score must be between 0 and 1")
	ErrSecurityInvalidSandboxMode          = errors.New("security: invalid sandbox_mode")
	ErrProtocolShellTimeoutPositive        = errors.New("protocol: shell_timeout must be positive when shell is enabled")
	ErrNameIsRequired                      = errors.New("name is required")
	ErrInvalidTransport                    = errors.New("invalid transport")
	ErrSmitheryApiKeyRequired              = errors.New("smithery_api_key is required for smithery transport")
	ErrSmitheryNamespaceRequired           = errors.New("smithery_namespace is required when url is specified")
	ErrCommandNotAllowedForSmithery        = errors.New("command is not allowed for smithery transport")
	ErrCommandRequiredForStdio             = errors.New("command is required for stdio transport")
	ErrUrlNotAllowedForStdio               = errors.New("url is not allowed for stdio transport")
	ErrOauthNotAllowedForStdio             = errors.New("oauth is not allowed for stdio transport")
	ErrUrlRequiredForTransport             = errors.New("url is required for transport")
	ErrInvalidUrl                          = errors.New("invalid url")
	ErrCommandNotAllowedForRemote          = errors.New("command is not allowed for remote transport")
	ErrOauthClientIdRequired               = errors.New("oauth client_id is required")
	ErrScratchpadMaxEntriesPositive        = errors.New("memory.scratchpad: max_entries must be positive")
	ErrPersistentBasePathRequired          = errors.New("memory.persistent: base_path is required when persistent memory is enabled")
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

// V2 is the unified configuration for Spin v2.0.
// This replaces the flat Config structure with organized sections.
type V2 struct {
	Version  string           `mapstructure:"version"   yaml:"version"`
	LLM      LLMV2      `mapstructure:"llm"       yaml:"llm"`
	Agent    AgentV2    `mapstructure:"agent"     yaml:"agent"`
	ACE      ACEV2      `mapstructure:"ace"       yaml:"ace"`
	Security SecurityV2 `mapstructure:"security"  yaml:"security"`
	Protocol ProtocolV2 `mapstructure:"protocol"  yaml:"protocol"`
	AgentsMD AgentsMDV2 `mapstructure:"agents_md" yaml:"agents_md"`
	Memory   MemoryV2   `mapstructure:"memory"    yaml:"memory"`
}

// AgentsMDV2 configures AGENTS.md project instructions support.
type AgentsMDV2 struct {
	// Enabled controls whether AGENTS.md is loaded (default: true).
	Enabled bool `mapstructure:"enabled" yaml:"enabled"`

	// Path specifies a custom path to AGENTS.md.
	// If empty, auto-discovery is used.
	Path string `mapstructure:"path" yaml:"path"`

	// MaxSize is the maximum file size in bytes (default: 100KB)
	// Files larger than this are truncated with a warning.
	MaxSize int64 `mapstructure:"max_size" yaml:"max_size"`
}

// LLMV2 configures the LLM provider.
type LLMV2 struct {
	Provider       string         `mapstructure:"provider"        yaml:"provider"`
	Model          string         `mapstructure:"model"           yaml:"model"`
	Temperature    float64        `mapstructure:"temperature"     yaml:"temperature"`
	MaxTokens      int            `mapstructure:"max_tokens"      yaml:"max_tokens"`
	Timeout        time.Duration  `mapstructure:"timeout"         yaml:"timeout"`
	BaseURL        string         `mapstructure:"base_url"        yaml:"base_url"`
	APIKey         string         `mapstructure:"api_key"         yaml:"api_key"`
	ProviderConfig map[string]any `mapstructure:"provider_config" yaml:"provider_config"`

	// ContextWindow is the model's context window size in tokens.
	// If set, this overrides the provider's auto-detected context window.
	// This is useful when using custom or fine-tuned models, or when the
	// provider doesn't know the context window for a particular model.
	// If not set (0), the system will try to auto-detect from the provider.
	ContextWindow int `mapstructure:"context_window" yaml:"context_window"`
}

// AgentV2 configures the agent behavior.
type AgentV2 struct {
	MaxTurns        int           `mapstructure:"max_turns"        yaml:"max_turns"`
	Timeout         time.Duration `mapstructure:"timeout"          yaml:"timeout"`
	WorkDir         string        `mapstructure:"work_dir"         yaml:"work_dir"`
	RequireApproval bool          `mapstructure:"require_approval" yaml:"require_approval"`

	// Performance Configuration.
	StreamBuffer  int  `mapstructure:"stream_buffer"  yaml:"stream_buffer"`
	CacheCommands bool `mapstructure:"cache_commands" yaml:"cache_commands"`

	// Environment Configuration.
	MaxFiles int  `mapstructure:"max_files" yaml:"max_files"`
	MaxDepth int  `mapstructure:"max_depth" yaml:"max_depth"`
	SkipGit  bool `mapstructure:"skip_git"  yaml:"skip_git"`

	// Storage Configuration.
	SessionDir   string `mapstructure:"session_dir"   yaml:"session_dir"`
	HistoryLimit int    `mapstructure:"history_limit" yaml:"history_limit"`

	// Logging Configuration.
	LogLevel  string `mapstructure:"log_level"  yaml:"log_level"`  // debug, info, warn, error.
	LogFormat string `mapstructure:"log_format" yaml:"log_format"` // text, json.
	Debug     bool   `mapstructure:"debug"      yaml:"debug"`      // Enable debug mode.

	// Cycle Detection Configuration.
	CycleDetection CycleDetectionV2 `mapstructure:"cycle_detection" yaml:"cycle_detection"`
}

// ACEV2 configures Agentic Context Engineering.
type ACEV2 struct {
	Enabled        bool    `mapstructure:"enabled"         yaml:"enabled"`
	PlaybookPath   string  `mapstructure:"playbook_path"   yaml:"playbook_path"`
	TrajectoryPath string  `mapstructure:"trajectory_path" yaml:"trajectory_path"`
	TopK           int     `mapstructure:"top_k"           yaml:"top_k"`
	MinScore       float64 `mapstructure:"min_score"       yaml:"min_score"`
}

// SecurityV2 configures security and sandboxing.
type SecurityV2 struct {
	SandboxMode     string   `mapstructure:"sandbox_mode"     yaml:"sandbox_mode"`
	PolicyFile      string   `mapstructure:"policy_file"      yaml:"policy_file"`
	AllowedCommands []string `mapstructure:"allowed_commands" yaml:"allowed_commands"`
	// Approval persistence feature flag and TTLs.
	ApprovalPersistenceEnabled bool `mapstructure:"approval_persistence_enabled" yaml:"approval_persistence_enabled"`
	// Approval persistence TTLs.
	SessionPolicyTTL time.Duration `mapstructure:"session_policy_ttl" yaml:"session_policy_ttl"`
	GlobalPolicyTTL  time.Duration `mapstructure:"global_policy_ttl"  yaml:"global_policy_ttl"`
}

// ProtocolV2 configures protocol features (MCP, Git, Shell).
type ProtocolV2 struct {
	EnableMCP    bool                `mapstructure:"enable_mcp"    yaml:"enable_mcp"`
	MCPServers   []MCPServerConfigV2 `mapstructure:"mcp_servers"   yaml:"mcp_servers"`
	EnableGit    bool                `mapstructure:"enable_git"    yaml:"enable_git"`
	EnableShell  bool                `mapstructure:"enable_shell"  yaml:"enable_shell"`
	ShellTimeout time.Duration       `mapstructure:"shell_timeout" yaml:"shell_timeout"`
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
	ClientID     string   `mapstructure:"client_id"     yaml:"client_id"`
	ClientSecret string   `mapstructure:"client_secret" yaml:"client_secret,omitempty"`
	RedirectURL  string   `mapstructure:"redirect_url"  yaml:"redirect_url,omitempty"`
	Scopes       []string `mapstructure:"scopes"        yaml:"scopes,omitempty"`
}

// MCPServerConfigV2 configures an MCP server.
type MCPServerConfigV2 struct {
	// Common fields.
	Name      string           `mapstructure:"name"      yaml:"name"`
	Transport MCPTransportType `mapstructure:"transport" yaml:"transport,omitempty"`

	// Stdio transport fields (mutually exclusive with URL).
	Command string            `mapstructure:"command" yaml:"command,omitempty"`
	Args    []string          `mapstructure:"args"    yaml:"args,omitempty"`
	Env     map[string]string `mapstructure:"env"     yaml:"env,omitempty"`

	// Remote transport fields (mutually exclusive with Command).
	URL     string            `mapstructure:"url"     yaml:"url,omitempty"`
	Headers map[string]string `mapstructure:"headers" yaml:"headers,omitempty"`

	// OAuth configuration (optional, for protected servers).
	OAuth *MCPOAuthConfigV2 `mapstructure:"oauth" yaml:"oauth,omitempty"`

	// Smithery-specific fields.
	SmitheryAPIKey    string `mapstructure:"smithery_api_key"   yaml:"smithery_api_key,omitempty"`
	SmitheryNamespace string `mapstructure:"smithery_namespace" yaml:"smithery_namespace,omitempty"`

	// DynamicLoadout enables dynamic tool discovery via search.
	// When true, tools are discovered at runtime rather than statically configured.
	DynamicLoadout bool `mapstructure:"dynamic_loadout" yaml:"dynamic_loadout,omitempty"`
}

// CycleDetectionV2 configures automatic cycle detection and intervention.
type CycleDetectionV2 struct {
	// Enabled controls whether cycle detection is active (default: true).
	Enabled bool `mapstructure:"enabled" yaml:"enabled"`

	// WindowSize is the number of snapshots to compare for pattern detection (default: 3).
	WindowSize int `mapstructure:"window_size" yaml:"window_size"`

	// SimilarityThresh is the threshold for response similarity detection (default: 0.8).
	SimilarityThresh float64 `mapstructure:"similarity_thresh" yaml:"similarity_thresh"`

	// ToolRepeatLimit is the max identical tool calls before triggering cycle (default: 3).
	ToolRepeatLimit int `mapstructure:"tool_repeat_limit" yaml:"tool_repeat_limit"`

	// ErrorRepeatLimit is the max identical errors before triggering cycle (default: 3).
	ErrorRepeatLimit int `mapstructure:"error_repeat_limit" yaml:"error_repeat_limit"`
}

// MemoryV2 configures context offloading memory storage.
type MemoryV2 struct {
	// Scratchpad configures session-scoped ephemeral memory.
	Scratchpad ScratchpadV2 `mapstructure:"scratchpad" yaml:"scratchpad"`

	// Persistent configures cross-session persistent memory.
	Persistent PersistentMemoryV2 `mapstructure:"persistent" yaml:"persistent"`
}

// ScratchpadV2 configures the session-scoped scratchpad.
type ScratchpadV2 struct {
	// Enabled controls whether scratchpad is available (default: true).
	Enabled bool `mapstructure:"enabled" yaml:"enabled"`

	// MaxEntries is the maximum number of entries (default: 50).
	MaxEntries int `mapstructure:"max_entries" yaml:"max_entries"`

	// AutoEvict enables automatic LRU eviction (default: true).
	AutoEvict bool `mapstructure:"auto_evict" yaml:"auto_evict"`
}

// PersistentMemoryV2 configures cross-session persistent memory.
type PersistentMemoryV2 struct {
	// Enabled controls whether persistent memory is available (default: false).
	Enabled bool `mapstructure:"enabled" yaml:"enabled"`

	// BasePath is the directory for persistent memory storage (default: ~/.spin/memory).
	BasePath string `mapstructure:"base_path" yaml:"base_path"`
}

// Validate performs validation on the config.
func (c *V2) Validate() error {
	errs := &ValidationErrors{}

	// Validate each section (collect all errors, don't fail fast).
	err := c.LLM.Validate()
	if err != nil {
		errs.Add(err)
	}

	err = c.Agent.Validate()
	if err != nil {
		errs.Add(err)
	}

	err = c.ACE.Validate()
	if err != nil {
		errs.Add(err)
	}

	err = c.Security.Validate()
	if err != nil {
		errs.Add(err)
	}

	err = c.Protocol.Validate()
	if err != nil {
		errs.Add(err)
	}

	err = c.AgentsMD.Validate()
	if err != nil {
		errs.Add(err)
	}

	err = c.Memory.Validate()
	if err != nil {
		errs.Add(err)
	}

	return errs.ToError()
}

// Validate performs validation on the AgentsMD configuration.
func (a *AgentsMDV2) Validate() error {
	// MaxSize validation: if negative, treat as no limit.
	if a.MaxSize < 0 {
		a.MaxSize = 0
	}

	return nil
}

// Validate performs validation on the LLM configuration.
func (l *LLMV2) Validate() error {
	errs := &ValidationErrors{}

	// Required fields.
	if l.Provider == "" {
		errs.Add(ErrLlmProviderIsRequired)
	}

	if l.Model == "" {
		errs.Add(ErrLlmModelIsRequired)
	}

	// Numeric field ranges.
	if l.Temperature < 0 || l.Temperature > 2 {
		errs.Add(fmt.Errorf("llm: temperature must be between 0 and 2, got %.2f: %w", l.Temperature, ErrLlmTemperatureRange))
	}

	if l.MaxTokens <= 0 {
		errs.Add(fmt.Errorf("llm: max_tokens must be positive, got %d: %w", l.MaxTokens, ErrLlmMaxTokensPositive))
	}

	if l.Timeout <= 0 {
		errs.Add(fmt.Errorf("llm: timeout must be positive, got %v: %w", l.Timeout, ErrLlmTimeoutPositive))
	}

	return errs.ToError()
}

// Validate performs validation on the Agent configuration.
func (a *AgentV2) Validate() error {
	errs := &ValidationErrors{}

	// Required fields.
	if a.MaxTurns <= 0 {
		errs.Add(fmt.Errorf("agent: max_turns must be positive, got %d: %w", a.MaxTurns, ErrAgentMaxTurnsPositive))
	}

	if a.Timeout <= 0 {
		errs.Add(fmt.Errorf("agent: timeout must be positive, got %v: %w", a.Timeout, ErrAgentTimeoutPositive))
	}

	if a.WorkDir == "" {
		errs.Add(ErrAgentWorkDirIsRequired)
	}

	return errs.ToError()
}

// Validate performs validation on the ACE configuration.
func (ace *ACEV2) Validate() error {
	// Only validate if ACE is enabled.
	if !ace.Enabled {
		return nil
	}

	errs := &ValidationErrors{}

	// Required fields when enabled.
	if ace.PlaybookPath == "" {
		errs.Add(ErrAcePlaybookPathRequired)
	}

	if ace.TrajectoryPath == "" {
		errs.Add(ErrAceTrajectoryPathRequired)
	}

	// Numeric field ranges.
	if ace.TopK <= 0 {
		errs.Add(fmt.Errorf("ace: top_k must be positive, got %d: %w", ace.TopK, ErrAceTopKPositive))
	}

	if ace.MinScore < 0 || ace.MinScore > 1 {
		errs.Add(fmt.Errorf("ace: min_score must be between 0 and 1, got %.2f: %w", ace.MinScore, ErrAceMinScoreRange))
	}

	return errs.ToError()
}

// Validate performs validation on the Security configuration.
func (s *SecurityV2) Validate() error {
	// Validate sandbox mode if set.
	if s.SandboxMode != "" {
		validModes := map[string]bool{
			"none":           true,
			"workspace-only": true,
			"docker":         true,
			"firejail":       true,
		}
		if !validModes[s.SandboxMode] {
			return fmt.Errorf("security: sandbox_mode must be one of [none, workspace-only, docker, firejail], got %q: %w", s.SandboxMode, ErrSecurityInvalidSandboxMode)
		}
	}

	return nil
}

// Validate performs validation on the Protocol configuration.
func (p *ProtocolV2) Validate() error {
	errs := &ValidationErrors{}

	// Validate shell timeout if shell is enabled.
	if p.EnableShell && p.ShellTimeout <= 0 {
		errs.Add(fmt.Errorf("protocol: shell_timeout must be positive when shell is enabled, got %v: %w", p.ShellTimeout, ErrProtocolShellTimeoutPositive))
	}

	// Validate MCP servers if MCP is enabled.
	if p.EnableMCP {
		for i, server := range p.MCPServers {
			err := server.Validate()
			if err != nil {
				errs.Add(fmt.Errorf("protocol: mcp_servers[%d]: %w", i, err))
			}
		}
	}

	return errs.ToError()
}

// Validate performs validation on the MCP server configuration.
func (m *MCPServerConfigV2) Validate() error {
	errs := &ValidationErrors{}

	// Name is always required.
	if m.Name == "" {
		errs.Add(ErrNameIsRequired)
	}

	// Validate transport type.
	if !m.Transport.IsValid() {
		errs.Add(fmt.Errorf("invalid transport: %s: %w", m.Transport, ErrInvalidTransport))

		return errs.ToError()
	}

	// Determine effective transport (empty defaults to stdio).
	transport := m.Transport
	if transport == "" {
		transport = MCPTransportStdio
	}

	// Validate based on transport type.
	if transport == MCPTransportSmithery {
		m.validateSmithery(errs)
	} else if transport.IsRemote() {
		m.validateRemote(transport, errs)
	} else {
		m.validateStdio(errs)
	}

	return errs.ToError()
}

// validateSmithery validates Smithery transport configuration.
func (m *MCPServerConfigV2) validateSmithery(errs *ValidationErrors) {
	// API key is always required for Smithery.
	if m.SmitheryAPIKey == "" {
		errs.Add(ErrSmitheryApiKeyRequired)
	}

	// For dynamic loadout, URL and namespace are optional
	// For static mode (URL provided), namespace is also required.
	if m.URL != "" && m.SmitheryNamespace == "" {
		errs.Add(ErrSmitheryNamespaceRequired)
	}

	// Command is not allowed for smithery transport.
	if m.Command != "" {
		errs.Add(ErrCommandNotAllowedForSmithery)
	}
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
	// Command is required for stdio.
	if m.Command == "" {
		errs.Add(ErrCommandRequiredForStdio)
	}

	// URL is not allowed for stdio.
	if m.URL != "" {
		errs.Add(ErrUrlNotAllowedForStdio)
	}

	// OAuth is not allowed for stdio.
	if m.OAuth != nil {
		errs.Add(ErrOauthNotAllowedForStdio)
	}
}

// validateRemote validates remote transport configuration.
func (m *MCPServerConfigV2) validateRemote(transport MCPTransportType, errs *ValidationErrors) {
	// URL is required for remote transports.
	if m.URL == "" {
		errs.Add(fmt.Errorf("url is required for %s transport: %w", transport, ErrUrlRequiredForTransport))
	} else {
		// Validate URL format.
		parsedURL, err := url.Parse(m.URL)
		if err != nil || parsedURL.Scheme == "" || parsedURL.Host == "" {
			errs.Add(fmt.Errorf("invalid url: %s: %w", m.URL, ErrInvalidUrl))
		}
	}

	// Command is not allowed for remote transports.
	if m.Command != "" {
		errs.Add(ErrCommandNotAllowedForRemote)
	}

	// Validate OAuth if provided.
	if m.OAuth != nil {
		if m.OAuth.ClientID == "" {
			errs.Add(ErrOauthClientIdRequired)
		}
	}
}

// Validate performs validation on the Memory configuration.
func (m *MemoryV2) Validate() error {
	errs := &ValidationErrors{}

	// Validate scratchpad config.
	err := m.Scratchpad.Validate()
	if err != nil {
		errs.Add(err)
	}

	// Validate persistent config.
	err = m.Persistent.Validate()
	if err != nil {
		errs.Add(err)
	}

	return errs.ToError()
}

// Validate performs validation on the Scratchpad configuration.
func (s *ScratchpadV2) Validate() error {
	// Only validate if scratchpad is enabled.
	if !s.Enabled {
		return nil
	}

	errs := &ValidationErrors{}

	if s.MaxEntries <= 0 {
		errs.Add(fmt.Errorf("memory.scratchpad: max_entries must be positive, got %d: %w", s.MaxEntries, ErrScratchpadMaxEntriesPositive))
	}

	return errs.ToError()
}

// Validate performs validation on the PersistentMemory configuration.
func (p *PersistentMemoryV2) Validate() error {
	// Only validate if persistent memory is enabled.
	if !p.Enabled {
		return nil
	}

	errs := &ValidationErrors{}

	if p.BasePath == "" {
		errs.Add(ErrPersistentBasePathRequired)
	}

	return errs.ToError()
}

// DefaultV2 returns a V2 with sensible defaults.
func DefaultV2() *V2 {
	// Derive default policy file path under user config directory.
	policyFile := ""
	cfgDir, err := os.UserConfigDir()
	if err == nil && cfgDir != "" {
		policyFile = filepath.Join(cfgDir, "spin", "policies.json")
	}

	return &V2{
		Version: "2.0",
		LLM: LLMV2{
			Provider:       "ollama",
			Model:          "qwen2.5-coder:7b",
			Temperature:    0.7,
			MaxTokens:      8192,
			Timeout:        5 * time.Minute,
			BaseURL:        "", // Empty - provider will use its own default.
			APIKey:         "",
			ProviderConfig: make(map[string]any),
		},
		Agent: AgentV2{
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
			CycleDetection: CycleDetectionV2{
				Enabled:          true,
				WindowSize:       3,
				SimilarityThresh: 0.8,
				ToolRepeatLimit:  3,
				ErrorRepeatLimit: 3,
			},
		},
		ACE: ACEV2{
			Enabled:        false,
			PlaybookPath:   "~/.spin/ace/playbooks/default.json",
			TrajectoryPath: "~/.spin/ace/trajectories/",
			TopK:           5,
			MinScore:       0.3,
		},
		Security: SecurityV2{
			SandboxMode:                "workspace-only",
			PolicyFile:                 policyFile,
			AllowedCommands:            []string{},
			ApprovalPersistenceEnabled: true,
			SessionPolicyTTL:           8 * time.Hour,
			GlobalPolicyTTL:            30 * 24 * time.Hour,
		},
		Protocol: ProtocolV2{
			EnableMCP:    false,
			MCPServers:   []MCPServerConfigV2{},
			EnableGit:    true,
			EnableShell:  true,
			ShellTimeout: 5 * time.Minute,
		},
		AgentsMD: AgentsMDV2{
			Enabled: true,
			Path:    "",         // Auto-discover.
			MaxSize: 100 * 1024, // 100KB.
		},
		Memory: MemoryV2{
			Scratchpad: ScratchpadV2{
				Enabled:    true,
				MaxEntries: 50,
				AutoEvict:  true,
			},
			Persistent: PersistentMemoryV2{
				Enabled:  false,
				BasePath: "~/.spin/memory",
			},
		},
	}
}
