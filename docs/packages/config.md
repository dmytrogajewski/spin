# Package: internal/config

**Path:** `internal/config`
**Purpose:** Configuration file management with multi-format support

---

## Overview

The `config` package manages Spin's configuration files, supporting multiple formats (YAML, TOML, JSON) with environment variable overrides and validation. It provides a centralized configuration system with sensible defaults and precedence rules.

## Key Features

- **Multiple Formats**: YAML (default), TOML, and JSON support
- **Auto-Detection**: Format detection based on file extension
- **Environment Variables**: Override via `SPIN_*` variables
- **Validation**: Schema validation and type checking
- **Defaults**: Built-in sensible defaults
- **Precedence**: CLI > Env > File > Defaults
- **Hot Reload**: Watch for configuration changes (optional)

## Package Structure

```
internal/config/
├── config.go       # Main configuration types
├── loader.go       # Configuration loading logic
├── validator.go    # Validation rules
└── defaults.go     # Default values
```

---

## Configuration Formats

Spin supports three configuration formats with automatic format detection based on file extension:

### YAML Format (Default)

```yaml
# $HOME/.spin/spin.yaml or ./spin.yaml

llm:
  provider: openai
  model: gpt-4
  base_url: https://api.openai.com/v1
  temperature: 0.7
  max_tokens: 4096

agent:
  max_turns: 50
  timeout: 5m
  require_approval: true

sandbox:
  mode: workspace-write  # read-only, workspace-write, full-access

appearance:
  theme: auto
  no_color: false

mcp:
  servers:
    - name: filesystem
      command: npx
      args:
        - "-y"
        - "@modelcontextprotocol/server-filesystem"
        - "/workspace"
    - name: github
      command: mcp-server-github
      args:
        - "--token-file"
        - "~/.github-token"
```

### TOML Format (Alternative)

```toml
# $HOME/.spin/spin.toml or ./spin.toml

[llm]
provider = "openai"
model = "gpt-4"
base_url = "https://api.openai.com/v1"
temperature = 0.7
max_tokens = 4096

[agent]
max_turns = 50
timeout = "5m"
require_approval = true

[sandbox]
mode = "workspace-write"  # read-only, workspace-write, full-access

[appearance]
theme = "auto"
no_color = false

[[mcp.servers]]
name = "filesystem"
command = "npx"
args = ["-y", "@modelcontextprotocol/server-filesystem", "/workspace"]

[[mcp.servers]]
name = "github"
command = "mcp-server-github"
args = ["--token-file", "~/.github-token"]
```

### JSON Format (Alternative)

```json
{
  "llm": {
    "provider": "openai",
    "model": "gpt-4",
    "base_url": "https://api.openai.com/v1",
    "temperature": 0.7,
    "max_tokens": 4096
  },
  "agent": {
    "max_turns": 50,
    "timeout": "5m",
    "require_approval": true
  },
  "sandbox": {
    "mode": "workspace-write"
  },
  "appearance": {
    "theme": "auto",
    "no_color": false
  },
  "mcp": {
    "servers": [
      {
        "name": "filesystem",
        "command": "npx",
        "args": ["-y", "@modelcontextprotocol/server-filesystem", "/workspace"]
      },
      {
        "name": "github",
        "command": "mcp-server-github",
        "args": ["--token-file", "~/.github-token"]
      }
    ]
  }
}
```

---

## Usage

### Loading Configuration

```go
import "github.com/dmytrogajewski/spin/internal/config"

// Load from default locations (searches for spin.yaml, spin.toml, or spin.json)
loader := config.NewLoader()
err := loader.Load("")
if err != nil {
    log.Fatal(err)
}

// Load from specific file (format auto-detected from extension)
loader := config.NewLoader()
err := loader.LoadFromFile("custom-config.toml")
if err != nil {
    log.Fatal(err)
}

// Access configuration values
provider := loader.GetString("llm.provider")
maxTurns := loader.GetInt("agent.max_turns")
```

### Environment Variable Overrides

```bash
export SPIN_LLM_MODEL="gpt-4-turbo"
export SPIN_AGENT_MAX_TURNS="100"
export SPIN_SANDBOX_MODE="read-only"
```

```go
// Environment variables automatically override config file
loader := config.NewLoader()
loader.Load("")
fmt.Println(loader.GetString("llm.model")) // "gpt-4-turbo"
```

### Programmatic Configuration

```go
loader := config.NewLoader()

// Set values programmatically
loader.Set("llm.model", "claude-3-5-sonnet-20241022")
loader.Set("agent.max_turns", 100)

// Or unmarshal into a struct
type Config struct {
    LLM struct {
        Provider string
        Model    string
    }
    Agent struct {
        MaxTurns int `mapstructure:"max_turns"`
    }
}

var cfg Config
if err := loader.Unmarshal(&cfg); err != nil {
    log.Fatal(err)
}
```

---

## Default Values

The loader provides sensible defaults that can be overridden:

```go
loader := config.NewLoader()

// Set defaults before loading
loader.SetDefault("llm.provider", "openai")
loader.SetDefault("llm.model", "gpt-4")
loader.SetDefault("llm.temperature", 0.7)
loader.SetDefault("llm.max_tokens", 4096)
loader.SetDefault("agent.max_turns", 50)
loader.SetDefault("agent.require_approval", true)
loader.SetDefault("sandbox.mode", "workspace-write")

// Load config (defaults used if not in file/env)
loader.Load("")
```

---

## Precedence Rules

Configuration values are resolved in this order (highest to lowest):

1. **Command-line flags** (e.g., `--model gpt-4`)
2. **Environment variables** (e.g., `SPIN_LLM_MODEL=gpt-4`)
3. **Configuration file** (`~/.config/spin/config.toml`)
4. **Built-in defaults**

---

## File Locations

### Default Search Paths

The loader searches for configuration files in the following order:

1. **Current directory:** `./spin.{yaml,yml,toml,json}`
2. **Home directory:** `$HOME/.spin/spin.{yaml,yml,toml,json}`
3. **System directory:** `/etc/spin/spin.{yaml,yml,toml,json}`

**Supported extensions:**
- `.yaml`, `.yml` - YAML format
- `.toml` - TOML format
- `.json` - JSON format

### Custom Path

```bash
# Specify exact file (format auto-detected from extension)
spin --config-file /path/to/custom-config.toml
spin --config-file /path/to/custom-config.yaml
spin --config-file /path/to/custom-config.json
```

---

## Testing

```go
func TestConfig(t *testing.T) {
    // Create temp config file
    tmpDir := t.TempDir()
    configFile := filepath.Join(tmpDir, "test.yaml")

    yamlContent := `
llm:
  provider: openai
  model: test-model
`
    os.WriteFile(configFile, []byte(yamlContent), 0644)

    // Load and test
    loader := config.NewLoader()
    err := loader.LoadFromFile(configFile)
    assert.NoError(t, err)
    assert.Equal(t, "test-model", loader.GetString("llm.model"))
}
```

### Format Testing

```go
// Test YAML format
func TestYAML(t *testing.T) {
    loader := config.NewLoader()
    err := loader.LoadFromFile("test.yaml")
    require.NoError(t, err)
}

// Test TOML format
func TestTOML(t *testing.T) {
    loader := config.NewLoader()
    err := loader.LoadFromFile("test.toml")
    require.NoError(t, err)
}

// Test JSON format
func TestJSON(t *testing.T) {
    loader := config.NewLoader()
    err := loader.LoadFromFile("test.json")
    require.NoError(t, err)
}
```

---

**Last Updated:** 2025-10-05
**Test Coverage:** 88.1%
**Supported Formats:** YAML (default), TOML, JSON
**Status:** ✅ Production Ready
