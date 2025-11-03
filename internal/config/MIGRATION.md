# Migration Guide: Config V1 → V2

## Overview

This guide helps you migrate from Spin's V1 flat configuration to the new V2 section-based configuration introduced in Spin 2.0.

## Why Migrate?

ConfigV2 provides:

1. **Better organization**: Logical grouping of related settings
2. **Multi-source support**: Environment variables, files, and CLI flags
3. **Comprehensive validation**: Catch all configuration errors at once
4. **Type safety**: Proper duration types instead of integer seconds
5. **Cross-section validation**: Enforce dependencies between settings

## Quick Start

### Option 1: Automatic Migration (Recommended)

If you have an existing V1 config file, Spin will automatically migrate it:

```bash
# Your old config.yaml will be automatically converted
spin agent run
```

The migration happens transparently - no manual changes needed!

### Option 2: Manual Conversion

Convert your V1 config to V2 format using this guide.

## Configuration Format Changes

### V1 (Flat Structure)

```yaml
# config.yaml (V1)
provider: "ollama"
model: "qwen2.5-coder:32b"
base_url: "http://localhost:11434"
temperature: 0.7
max_tokens: 4096
timeout: 3600              # seconds (integer)
llm_timeout: 300           # seconds (integer)

max_turns: 20
work_dir: "/tmp/spin-agent"
require_approval: false

ace_enabled: true
ace_playbook_path: "/etc/spin/playbooks/default.yaml"
ace_trajectory_path: "/var/lib/spin/trajectories"

sandbox_mode: "docker"
policy_file: "/etc/spin/policy.yaml"
allowed_commands:
  - "ls"
  - "cat"

enable_mcp: true
```

### V2 (Section-Based Structure)

```yaml
# config.yaml (V2)
version: "2.0"

llm:
  provider: "ollama"
  model: "qwen2.5-coder:32b"
  base_url: "http://localhost:11434"
  temperature: 0.7
  max_tokens: 4096
  timeout: 300s            # duration string!

agent:
  max_turns: 20
  timeout: 3600s           # duration string!
  work_dir: "/tmp/spin-agent"
  require_approval: false

ace:
  enabled: true
  playbook_path: "/etc/spin/playbooks/default.yaml"
  trajectory_path: "/var/lib/spin/trajectories"
  top_k: 5
  min_score: 0.7

security:
  sandbox_mode: "docker"
  policy_file: "/etc/spin/policy.yaml"
  allowed_commands:
    - "ls"
    - "cat"

protocol:
  enable_mcp: true
  enable_git: true
  enable_shell: true
  shell_timeout: 30s       # duration string!
```

## Field Mapping Reference

### Version Field (New)

```yaml
# V2 only - add this at the top
version: "2.0"
```

### LLM Section

| V1 Field | V2 Field | Notes |
|----------|----------|-------|
| `provider` | `llm.provider` | No change |
| `model` | `llm.model` | No change |
| `base_url` | `llm.base_url` | No change |
| `api_key` | `llm.api_key` | No change |
| `temperature` | `llm.temperature` | No change |
| `max_tokens` | `llm.max_tokens` | No change |
| `llm_timeout` | `llm.timeout` | **Changed to duration string** (see below) |

### Agent Section

| V1 Field | V2 Field | Notes |
|----------|----------|-------|
| `max_turns` | `agent.max_turns` | No change |
| `timeout` | `agent.timeout` | **Changed to duration string** (see below) |
| `work_dir` | `agent.work_dir` | No change |
| `require_approval` | `agent.require_approval` | No change |

### ACE Section

| V1 Field | V2 Field | Notes |
|----------|----------|-------|
| `ace_enabled` | `ace.enabled` | Renamed (underscore removed) |
| `ace_playbook_path` | `ace.playbook_path` | Renamed (prefix removed) |
| `ace_trajectory_path` | `ace.trajectory_path` | Renamed (prefix removed) |
| N/A | `ace.top_k` | **New field** (optional) |
| N/A | `ace.min_score` | **New field** (optional) |

### Security Section

| V1 Field | V2 Field | Notes |
|----------|----------|-------|
| `sandbox_mode` | `security.sandbox_mode` | No change |
| `policy_file` | `security.policy_file` | No change |
| `allowed_commands` | `security.allowed_commands` | No change |

### Protocol Section

| V1 Field | V2 Field | Notes |
|----------|----------|-------|
| `enable_mcp` | `protocol.enable_mcp` | No change |
| N/A | `protocol.mcp_servers` | **New field** (optional) |
| N/A | `protocol.enable_git` | **New field** (defaults to true) |
| N/A | `protocol.enable_shell` | **New field** (defaults to true) |
| N/A | `protocol.shell_timeout` | **New field** (required if shell enabled) |

## Breaking Changes

### 1. Duration Format Change

**V1 used integer seconds:**

```yaml
timeout: 3600        # 3600 seconds
llm_timeout: 300     # 300 seconds
```

**V2 uses Go duration strings:**

```yaml
agent:
  timeout: 3600s     # or "1h", "60m", etc.
llm:
  timeout: 300s      # or "5m"
protocol:
  shell_timeout: 30s
```

**Conversion table:**

| Seconds | Duration String | Alternative |
|---------|----------------|-------------|
| 30 | `30s` | - |
| 60 | `60s` | `1m` |
| 300 | `300s` | `5m` |
| 600 | `600s` | `10m` |
| 3600 | `3600s` | `1h` or `60m` |

### 2. Stricter Validation

V2 has cross-section validation rules:

```yaml
# This will FAIL in V2:
ace:
  enabled: true
  # ERROR: playbook_path required when enabled!
  # ERROR: trajectory_path required when enabled!

# This is correct:
ace:
  enabled: true
  playbook_path: "/path/to/playbook.yaml"
  trajectory_path: "/path/to/trajectories"
```

```yaml
# This will FAIL in V2:
protocol:
  enable_shell: true
  # ERROR: shell_timeout required when shell enabled!

# This is correct:
protocol:
  enable_shell: true
  shell_timeout: 30s
```

### 3. Sandbox Mode Values

Valid values remain: `none`, `docker`, `firejail`

```yaml
# V1 and V2 both accept:
security:
  sandbox_mode: "docker"  # OK
  sandbox_mode: "none"    # OK
  sandbox_mode: "firejail" # OK
  sandbox_mode: "firecracker" # ERROR in V2!
```

## Step-by-Step Migration

### Step 1: Backup Your Config

```bash
cp config.yaml config.yaml.v1.backup
```

### Step 2: Add Version Header

```yaml
version: "2.0"
```

### Step 3: Reorganize Into Sections

Group your fields under the appropriate section headers:

```yaml
llm:
  # All LLM-related fields here

agent:
  # All agent-related fields here

ace:
  # All ACE-related fields here

security:
  # All security-related fields here

protocol:
  # All protocol-related fields here
```

### Step 4: Convert Duration Fields

Change integer seconds to duration strings:

```yaml
# Before:
timeout: 3600
llm_timeout: 300

# After:
agent:
  timeout: 3600s
llm:
  timeout: 300s
```

### Step 5: Update Field Names

Rename fields that changed:

```yaml
# Before:
ace_enabled: true
ace_playbook_path: "/path"
ace_trajectory_path: "/path"

# After:
ace:
  enabled: true
  playbook_path: "/path"
  trajectory_path: "/path"
```

### Step 6: Add New Required Fields

If you use certain features, add new required fields:

```yaml
protocol:
  enable_shell: true
  shell_timeout: 30s  # Required if shell enabled!
```

### Step 7: Validate Your Config

Run Spin with your new config:

```bash
spin agent run --config config.yaml
```

If there are errors, Spin will show ALL validation errors at once:

```
Error: validation failed: 3 errors found:
  1. ace: playbook_path is required when ACE is enabled
  2. ace: trajectory_path is required when ACE is enabled  
  3. protocol: shell_timeout is required when shell is enabled
```

Fix all errors and try again.

## Migration Examples

### Example 1: Minimal Config

**V1:**

```yaml
provider: "ollama"
model: "llama3.2:3b"
max_turns: 10
timeout: 600
llm_timeout: 60
work_dir: "/tmp/spin"
require_approval: false
sandbox_mode: "docker"
enable_mcp: false
ace_enabled: false
```

**V2:**

```yaml
version: "2.0"

llm:
  provider: "ollama"
  model: "llama3.2:3b"
  timeout: 60s
  max_tokens: 4096
  temperature: 0.7

agent:
  max_turns: 10
  timeout: 600s
  work_dir: "/tmp/spin"
  require_approval: false

ace:
  enabled: false

security:
  sandbox_mode: "docker"

protocol:
  enable_mcp: false
  enable_git: true
  enable_shell: true
  shell_timeout: 30s
```

### Example 2: Full Config with ACE

**V1:**

```yaml
provider: "openai"
model: "gpt-4"
api_key: "sk-..."
base_url: "https://api.openai.com/v1"
temperature: 0.8
max_tokens: 8192
timeout: 3600
llm_timeout: 300

max_turns: 50
work_dir: "/var/lib/spin"
require_approval: true

ace_enabled: true
ace_playbook_path: "/etc/spin/playbooks/default.yaml"
ace_trajectory_path: "/var/lib/spin/trajectories"

sandbox_mode: "firejail"
policy_file: "/etc/spin/policy.yaml"
allowed_commands:
  - "ls"
  - "cat"
  - "grep"

enable_mcp: true
```

**V2:**

```yaml
version: "2.0"

llm:
  provider: "openai"
  model: "gpt-4"
  api_key: "sk-..."
  base_url: "https://api.openai.com/v1"
  temperature: 0.8
  max_tokens: 8192
  timeout: 300s

agent:
  max_turns: 50
  timeout: 3600s
  work_dir: "/var/lib/spin"
  require_approval: true

ace:
  enabled: true
  playbook_path: "/etc/spin/playbooks/default.yaml"
  trajectory_path: "/var/lib/spin/trajectories"
  top_k: 5
  min_score: 0.7

security:
  sandbox_mode: "firejail"
  policy_file: "/etc/spin/policy.yaml"
  allowed_commands:
    - "ls"
    - "cat"
    - "grep"

protocol:
  enable_mcp: true
  mcp_servers:
    - name: "filesystem"
      command: "npx"
      args:
        - "-y"
        - "@modelcontextprotocol/server-filesystem"
  enable_git: true
  enable_shell: true
  shell_timeout: 60s
```

## Environment Variables

V2 supports environment variable overrides with the `SPIN_` prefix:

```bash
# V2 environment variables
export SPIN_LLM_PROVIDER=ollama
export SPIN_LLM_MODEL=qwen2.5-coder:32b
export SPIN_AGENT_MAX_TURNS=20
export SPIN_ACE_ENABLED=true
```

These override file-based configuration.

## Programmatic Migration

If you're migrating configs programmatically:

```go
import "github.com/dmytrogajewski/spin/internal/config"

// Load V1 config
var v1Config config.Config
// ... load v1Config from file ...

// Migrate to V2
v2Config := config.MigrateV1ToV2(&v1Config)

// Validate
if err := v2Config.Validate(); err != nil {
    log.Fatal(err)
}

// Save as V2
data, _ := yaml.Marshal(v2Config)
os.WriteFile("config.yaml", data, 0644)
```

## Troubleshooting

### Error: "validation failed: N errors found"

V2 shows ALL validation errors at once. Read through the list and fix each error.

### Error: "ace: playbook_path is required when ACE is enabled"

If `ace.enabled: true`, you must provide both `playbook_path` and `trajectory_path`:

```yaml
ace:
  enabled: true
  playbook_path: "/path/to/playbook.yaml"
  trajectory_path: "/path/to/trajectories"
```

Or disable ACE:

```yaml
ace:
  enabled: false
```

### Error: "protocol: shell_timeout is required when shell is enabled"

If `protocol.enable_shell: true`, you must provide `shell_timeout`:

```yaml
protocol:
  enable_shell: true
  shell_timeout: 30s
```

### Error: "parsing time: invalid duration"

Duration must be a valid Go duration string:

```yaml
# Wrong:
timeout: 60          # Missing unit

# Right:
timeout: 60s         # With unit
```

### Error: "sandbox_mode must be one of [none, docker, firejail]"

Use one of the valid sandbox modes:

```yaml
security:
  sandbox_mode: "docker"  # OK
  # Not: "firecracker", "vm", etc.
```

## Getting Help

- **Spec**: See `internal/config/SPEC.md` for complete ConfigV2 documentation
- **Examples**: Check `internal/config/golden/` for example configs
- **Issues**: Report bugs at https://github.com/dmytrogajewski/spin/issues

## Rollback

If you need to rollback to V1:

```bash
# Restore your backup
cp config.yaml.v1.backup config.yaml

# Use an older version of Spin that supports V1
spin-v1.x agent run
```

Note: Future versions of Spin may drop V1 support entirely, so migration is recommended.
