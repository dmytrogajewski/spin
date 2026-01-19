# Configuration Flows Analysis

This document provides a comprehensive analysis of configuration management in the Spin agent application.

## Overview

Spin uses a multi-layered configuration system built on [Viper](https://github.com/spf13/viper) with strict precedence rules and comprehensive validation.

## Configuration Hierarchy (Precedence Order)

From highest to lowest priority:

```
1. CLI Flags (--provider, --model, --max-turns, etc.)
2. Environment Variables (SPIN_* prefix)
3. Configuration File (YAML/JSON/TOML)
4. Built-in Defaults
```

## Key Files

| File | Purpose |
|------|---------|
| `internal/config/config_v2.go` | Configuration structs and validation |
| `internal/config/loader_v2.go` | Configuration loading and merging |
| `cmd/spin/root.go` | CLI flag definitions |
| `cmd/spin/services.go` | Service creation from config |

## Configuration File Locations

The loader searches these paths in order:

1. `./config.yaml` (current directory)
2. `$HOME/.spin/spin.yaml` (user home)
3. `/etc/spin/spin.yaml` (system-wide)

Or explicitly via `--config-file <path>`.

## Configuration Structure

### Main Config (ConfigV2)

**File:** `internal/config/config_v2.go:34-62`

```yaml
version: "2.0"

llm:                          # LLM provider settings
  provider: string            # Required: openai, anthropic, ollama, lmstudio
  model: string               # Required: model identifier
  temperature: float64        # 0.0-2.0 (default: 0.7)
  max_tokens: int             # Default: 8192
  timeout: duration           # Default: 5m
  base_url: string            # Optional custom endpoint
  api_key: string             # Optional (prefer env vars)
  provider_config: map        # Provider-specific options

agent:                        # Agent behavior
  max_turns: int              # Default: 50
  timeout: duration           # Default: 60m
  work_dir: string            # Default: "."
  require_approval: bool      # Default: false
  stream_buffer: int          # Default: 100
  cache_commands: bool        # Default: false
  max_files: int              # Default: 0 (unlimited)
  max_depth: int              # Default: 0 (unlimited)
  skip_git: bool              # Default: false
  session_dir: string         # Default: ~/.spin/sessions
  history_limit: int          # Default: 1000
  log_level: string           # debug, info, warn, error
  log_format: string          # text, json
  debug: bool                 # Default: false
  cycle_detection:
    enabled: bool             # Default: true
    window_size: int          # Default: 3
    similarity_thresh: float  # Default: 0.8
    tool_repeat_limit: int    # Default: 3
    error_repeat_limit: int   # Default: 3

ace:                          # Agentic Context Engineering
  enabled: bool               # Default: true
  playbook_path: string       # Default: ~/.spin/ace/playbooks/default.json
  trajectory_path: string     # Default: ~/.spin/ace/trajectories/
  top_k: int                  # Default: 5
  min_score: float64          # Default: 0.3

security:                     # Security & sandboxing
  sandbox_mode: string        # none, workspace-only, docker, firejail
  policy_file: string         # Optional policy file
  allowed_commands: []string  # Command whitelist
  approval_persistence_enabled: bool  # Default: true
  session_policy_ttl: duration        # Default: 8h
  global_policy_ttl: duration         # Default: 30d

protocol:                     # Protocol features
  enable_mcp: bool            # Model Context Protocol
  mcp_servers: []MCPServerConfig
  enable_git: bool            # Default: true
  enable_shell: bool          # Default: true
  shell_timeout: duration     # Default: 5m
```

## Environment Variables

**Prefix:** `SPIN_`

All configuration keys can be set via environment variables by replacing dots with underscores:

```bash
# LLM settings
SPIN_LLM_PROVIDER=openai
SPIN_LLM_MODEL=gpt-4o
SPIN_LLM_TEMPERATURE=0.5
SPIN_LLM_MAX_TOKENS=8192
SPIN_LLM_TIMEOUT=5m
SPIN_LLM_BASE_URL=https://api.openai.com/v1
SPIN_LLM_API_KEY=sk-...

# Agent settings
SPIN_AGENT_MAX_TURNS=100
SPIN_AGENT_TIMEOUT=2h
SPIN_AGENT_WORK_DIR=/path/to/project
SPIN_AGENT_REQUIRE_APPROVAL=true
SPIN_AGENT_DEBUG=true

# ACE settings
SPIN_ACE_ENABLED=true
SPIN_ACE_TOP_K=10
SPIN_ACE_MIN_SCORE=0.5

# Security settings
SPIN_SECURITY_SANDBOX_MODE=docker
SPIN_SECURITY_APPROVAL_PERSISTENCE_ENABLED=true

# Protocol settings
SPIN_PROTOCOL_ENABLE_MCP=true
SPIN_PROTOCOL_ENABLE_GIT=true
SPIN_PROTOCOL_SHELL_TIMEOUT=10m
```

**Provider-specific API keys** (checked automatically):

```bash
OPENAI_API_KEY=sk-...       # For openai provider
ANTHROPIC_API_KEY=sk-...    # For anthropic provider
```

## CLI Flags

**Global persistent flags** (`cmd/spin/root.go:36-47`):

```
--model string              Model to use
--provider string           Provider (ollama, openai, anthropic, lmstudio)
--sandbox string            Sandbox mode
--cd string                 Working directory
--config-file string        Path to configuration file
-c, --config strings        Config overrides (key=value)
-m, --mode string           Task mode
```

**Command-specific flags:**

```
# TUI
--max-turns int             Maximum conversation turns
--debug                     Enable debug mode
--auto-approve              Auto-approve all operations

# EXEC
--timeout string            Maximum execution time
--format string             Output format (text, json)
--no-stream                 Disable streaming
--exit-on-error             Exit on first error
```

## Configuration Loading

### Loader API

**File:** `internal/config/loader_v2.go:245-313`

```go
cfg, err := config.Load(config.Source{
    File: "/path/to/config.yaml",    // Optional explicit file
    Flags: config.FlagOverrides{     // CLI flag overrides
        Provider: "openai",
        Model:    "gpt-4o",
        MaxTurns: 100,
        Debug:    true,
    },
    WorkDir: "/path/to/project",
})
```

### Loading Methods

| Method | Purpose |
|--------|---------|
| `Load(Source)` | Unified API with all sources |
| `LoadFromFile(path)` | Load from specific file only |
| `LoadWithEnv()` | Load from env vars + defaults |
| `LoadFromFileWithEnv(path)` | File + env vars merged |

### Loading Flow

```
User Input
    │
    ├── CLI Flags
    ├── Environment Variables
    └── Config File (if exists)
           │
           ▼
    LoaderV2.Load(Source)
           │
    1. Load from File (if specified)
    2. Apply Flag Overrides
    3. Apply Environment Variables
    4. Apply Defaults for Missing Fields
           │
           ▼
    Validate Configuration
           │
           ▼
    ConfigV2 Instance
           │
           ▼
    Propagate to Subsystems
```

## Configuration Propagation

### Service Creation Flow

**File:** `cmd/spin/services.go:23-89`

```go
func createServices(cfg *config.ConfigV2, workDir string, logger *log.Logger) {
    // Git service
    if cfg.Protocol.EnableGit {
        gitSvc = git.NewService()
    }
    
    // Shell service
    if cfg.Protocol.EnableShell {
        shellSvc = shell.NewService(..., cfg.Protocol.ShellTimeout)
    }
    
    // MCP service
    if cfg.Protocol.EnableMCP && len(cfg.Protocol.MCPServers) > 0 {
        mcpSvc = mcp.NewService(config, logger)
    }
}
```

### LLM Provider Factory

**File:** `internal/llm/factory/factory.go`

The factory uses:
- `cfg.LLM.Provider` - determines which client to create
- `cfg.LLM.Model` - passed to provider
- `cfg.LLM.BaseURL` - API endpoint
- `cfg.LLM.APIKey` - authentication
- `cfg.LLM.Timeout` - request timeout

### Agent Builder

**File:** `internal/agent/builder.go`

Configuration methods:
- `getTimeout()` -> `cfg.Agent.Timeout`
- `getMaxTurns()` -> `cfg.Agent.MaxTurns`
- `getCycleDetectionConfig()` -> `cfg.Agent.CycleDetection.*`

## Validation

**File:** `internal/config/config_v2.go:221-352`

### Strategy

- **Non-blocking:** Collects all errors before returning
- **Per-section:** Each subsection validates independently
- **Returns:** `ValidationErrors` (multi-error type)

### Required Fields

- `llm.provider` - must not be empty
- `llm.model` - must not be empty
- `agent.max_turns` - must be positive
- `agent.timeout` - must be positive
- `agent.work_dir` - must not be empty

### Range Validation

```go
// LLM
if l.Temperature < 0 || l.Temperature > 2 { error }
if l.MaxTokens <= 0 { error }
if l.Timeout <= 0 { error }

// ACE (when enabled)
if ace.TopK <= 0 { error }
if ace.MinScore < 0 || ace.MinScore > 1 { error }

// Security
validModes := {none, workspace-only, docker, firejail}
if !validModes[s.SandboxMode] { error }
```

## Defaults

**File:** `internal/config/config_v2.go:311-373`

```go
DefaultConfigV2() *ConfigV2 {
    return &ConfigV2{
        Version: "2.0",
        LLM: LLMConfigV2{
            Provider:    "ollama",
            Model:       "qwen2.5-coder:7b",
            Temperature: 0.7,
            MaxTokens:   8192,
            Timeout:     5 * time.Minute,
        },
        Agent: AgentConfigV2{
            MaxTurns:        50,
            Timeout:         60 * time.Minute,
            WorkDir:         ".",
            RequireApproval: false,
            StreamBuffer:    100,
            SessionDir:      "~/.spin/sessions",
            HistoryLimit:    1000,
            LogLevel:        "info",
            LogFormat:       "text",
            CycleDetection: CycleDetectionConfigV2{
                Enabled:          true,
                WindowSize:       3,
                SimilarityThresh: 0.8,
                ToolRepeatLimit:  3,
                ErrorRepeatLimit: 3,
            },
        },
        ACE: ACEConfigV2{
            Enabled:        true,
            PlaybookPath:   "~/.spin/ace/playbooks/default.json",
            TrajectoryPath: "~/.spin/ace/trajectories/",
            TopK:           5,
            MinScore:       0.3,
        },
        Security: SecurityConfigV2{
            SandboxMode:                "workspace-only",
            ApprovalPersistenceEnabled: true,
            SessionPolicyTTL:           8 * time.Hour,
            GlobalPolicyTTL:            30 * 24 * time.Hour,
        },
        Protocol: ProtocolConfigV2{
            EnableMCP:    false,
            EnableGit:    true,
            EnableShell:  true,
            ShellTimeout: 5 * time.Minute,
        },
    }
}
```

## Usage Examples

### Example 1: Using Configuration File

```bash
spin --config-file ~/.spin/my-config.yaml exec "analyze code"
```

### Example 2: CLI Overrides

```bash
spin --provider openai --model gpt-4o tui
```

### Example 3: Environment Variables

```bash
export SPIN_LLM_PROVIDER=anthropic
export SPIN_LLM_MODEL=claude-3-5-sonnet
export ANTHROPIC_API_KEY=sk-...
spin tui
```

### Example 4: Combined Precedence

```bash
# Environment
export SPIN_LLM_PROVIDER=ollama
export SPIN_LLM_MODEL=llama2

# CLI flag overrides env var
spin --model qwen2.5 exec "test"

# Result:
# - provider: ollama (from env)
# - model: qwen2.5 (from CLI - higher priority)
```

## Config Management Commands

```bash
spin config show           # Display current config
spin config validate       # Validate config file
spin config edit           # Open config in editor
spin config path           # Show config file path
spin config get <key>      # Get specific value
spin config set <key> <v>  # Set specific value
```

## Example Configuration Files

### OpenAI (`examples/config-openai.yaml`)

```yaml
llm:
  provider: openai
  model: gpt-4o
  base_url: https://api.openai.com/v1
  timeout: 60s
  key_name: my-openai-key
sandbox:
  mode: workspace-write
```

### Ollama (`examples/config-ollama.yaml`)

```yaml
llm:
  provider: ollama
  model: llama3.1
  base_url: http://localhost:11434
  timeout: 30s
sandbox:
  mode: workspace-write
```

## Architecture Diagram

```
┌─────────────────────────────────────────────────────────────────┐
│                        User Input                                │
├─────────────────┬─────────────────┬─────────────────────────────┤
│   CLI Flags     │  Environment    │    Config File              │
│  (--provider,   │  (SPIN_*)       │  (yaml/json/toml)           │
│   --model...)   │                 │                             │
└────────┬────────┴────────┬────────┴────────────┬────────────────┘
         │                 │                      │
         ▼                 ▼                      ▼
┌─────────────────────────────────────────────────────────────────┐
│                     LoaderV2.Load()                              │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │  1. Merge sources by precedence                          │    │
│  │  2. Apply defaults for missing fields                    │    │
│  │  3. Validate all sections                                │    │
│  └─────────────────────────────────────────────────────────┘    │
└────────────────────────────┬────────────────────────────────────┘
                             │
                             ▼
┌─────────────────────────────────────────────────────────────────┐
│                        ConfigV2                                  │
├─────────────────┬─────────────────┬─────────────────────────────┤
│      LLM        │     Agent       │    Security/Protocol         │
└────────┬────────┴────────┬────────┴────────────┬────────────────┘
         │                 │                      │
         ▼                 ▼                      ▼
┌─────────────┐   ┌─────────────┐   ┌─────────────────────────────┐
│ LLM Factory │   │Agent Builder│   │    Service Creation          │
│  - Provider │   │  - Timeout  │   │  - Git Service               │
│  - Model    │   │  - MaxTurns │   │  - Shell Service             │
│  - Auth     │   │  - Cycles   │   │  - MCP Service               │
└─────────────┘   └─────────────┘   └─────────────────────────────┘
```
