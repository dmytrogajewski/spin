## NAME

Configuration Manual - complete reference for Spin configuration

## DESCRIPTION

This manual documents all configuration options available in Spin. Configuration can be set via command-line flags, environment variables, or a YAML configuration file. This document provides a complete reference for all options, their defaults, and usage examples.

## CONFIGURATION PRECEDENCE

Configuration is merged in this order (highest to lowest priority):

1. **Command-line flags** (highest priority)
2. **Environment variables**
3. **Config file** (`~/.spin/spin.yaml` or `~/.spin/config.yaml`)
4. **Built-in defaults** (lowest priority)

## CONFIGURATION FILE LOCATION

Spin looks for configuration files in this order:

1. Path specified by `--config-file` flag
2. `~/.spin/spin.yaml`
3. `~/.spin/config.yaml`
4. `~/.config/spin/spin.yaml`
5. `~/.config/spin/config.yaml`

## CONFIGURATION FILE FORMAT

Spin uses YAML format (version 2.0) for configuration files:

```yaml
version: "2.0"

# Configuration sections...
```

## CONFIGURATION SECTIONS

### LLM Configuration (`llm`)

Configures the Large Language Model provider and model settings.

#### Fields

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `provider` | string | Yes | `ollama` | LLM provider: `ollama`, `openai`, `anthropic`, `lmstudio`, `openai-compatible` |
| `model` | string | Yes | `qwen2.5-coder:7b` | Model name (provider-specific) |
| `base_url` | string | No | Provider default | Base URL for API endpoint |
| `temperature` | float | No | `0.7` | Sampling temperature (0.0-2.0) |
| `max_tokens` | int | No | `8192` | Maximum tokens per request |
| `timeout` | duration | No | `5m` | Request timeout (e.g., `30s`, `5m`, `1h`) |
| `api_key` | string | No | - | API key (deprecated, use keystore) |
| `provider_config` | map | No | `{}` | Provider-specific configuration |

#### Examples

**Ollama (Local):**
```yaml
llm:
  provider: ollama
  model: qwen3:0.6b
  base_url: http://localhost:11434
  timeout: 30s
```

**OpenAI:**
```yaml
llm:
  provider: openai
  model: gpt-4o
  base_url: https://api.openai.com/v1
  temperature: 0.7
  max_tokens: 4096
  timeout: 60s
```

**Anthropic:**
```yaml
llm:
  provider: anthropic
  model: claude-3-5-sonnet-20241022
  base_url: https://api.anthropic.com/v1
  temperature: 0.7
  max_tokens: 8192
```

**LM Studio (Local):**
```yaml
llm:
  provider: lmstudio
  model: codellama-13b
  base_url: http://localhost:1234/v1
  timeout: 30s
```

**OpenAI-Compatible (Together AI, Anyscale, etc.):**
```yaml
llm:
  provider: openai-compatible
  model: mixtral-8x7b-instruct
  base_url: https://api.together.xyz/v1
  timeout: 60s
```

#### Provider-Specific Configuration

Some providers support additional configuration via `provider_config`:

```yaml
llm:
  provider: ollama
  model: qwen3:1.7b
  provider_config:
    top_p: 0.9
    top_k: 40
    repeat_penalty: 1.1
```

### Agent Configuration (`agent`)

Configures agent behavior, performance, and execution limits.

#### Fields

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `max_turns` | int | Yes | `50` | Maximum agent turns per task |
| `timeout` | duration | Yes | `60m` | Total task timeout |
| `work_dir` | string | Yes | `.` | Working directory |
| `require_approval` | bool | No | `false` | Require approval for all operations |
| `stream_buffer` | int | No | `100` | Stream buffer size for real-time updates |
| `cache_commands` | bool | No | `false` | Enable command result caching |
| `max_files` | int | No | `0` | Maximum files to process (0 = unlimited) |
| `max_depth` | int | No | `0` | Maximum directory depth (0 = unlimited) |
| `skip_git` | bool | No | `false` | Skip Git integration |
| `session_dir` | string | No | `~/.spin/sessions` | Directory for session storage |
| `history_limit` | int | No | `1000` | Maximum history messages to keep |
| `log_level` | string | No | `info` | Log level: `debug`, `info`, `warn`, `error` |
| `log_format` | string | No | `text` | Log format: `text`, `json` |
| `debug` | bool | No | `false` | Enable debug mode |

#### Cycle Detection Configuration (`agent.cycle_detection`)

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `enabled` | bool | No | `true` | Enable cycle detection |
| `window_size` | int | No | `3` | Number of snapshots for pattern detection |
| `similarity_thresh` | float | No | `0.8` | Similarity threshold (0.0-1.0) |
| `tool_repeat_limit` | int | No | `3` | Max identical tool calls before cycle |
| `error_repeat_limit` | int | No | `3` | Max identical errors before cycle |

#### Examples

**Basic Agent Configuration:**
```yaml
agent:
  max_turns: 50
  timeout: 60m
  work_dir: /path/to/project
  require_approval: false
```

**Performance Tuning:**
```yaml
agent:
  max_turns: 100
  timeout: 2h
  stream_buffer: 200
  cache_commands: true
  max_files: 100
  max_depth: 5
```

**Cycle Detection:**
```yaml
agent:
  cycle_detection:
    enabled: true
    window_size: 5
    similarity_thresh: 0.85
    tool_repeat_limit: 5
    error_repeat_limit: 3
```

**Debug Configuration:**
```yaml
agent:
  debug: true
  log_level: debug
  log_format: json
```

### ACE Configuration (`ace`)

Configures Agentic Context Engineering (ACE) for learning from past trajectories.

#### Fields

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `enabled` | bool | No | `true` | Enable ACE |
| `playbook_path` | string | Conditional | `~/.spin/ace/playbooks/default.json` | Path to playbook file (required if enabled) |
| `trajectory_path` | string | Conditional | `~/.spin/ace/trajectories/` | Path to trajectory directory (required if enabled) |
| `top_k` | int | No | `5` | Number of top trajectories to retrieve |
| `min_score` | float | No | `0.3` | Minimum similarity score (0.0-1.0) |

#### Examples

**Enable ACE:**
```yaml
ace:
  enabled: true
  playbook_path: ~/.spin/ace/playbooks/default.json
  trajectory_path: ~/.spin/ace/trajectories/
  top_k: 5
  min_score: 0.3
```

**Disable ACE:**
```yaml
ace:
  enabled: false
```

### Security Configuration (`security`)

Configures security, sandboxing, and approval policies.

#### Fields

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `sandbox_mode` | string | No | `workspace-only` | Sandbox mode: `none`, `workspace-only`, `docker`, `firejail` |
| `policy_file` | string | No | `~/.spin/policies.json` | Path to approval policy file |
| `allowed_commands` | []string | No | `[]` | Commands always allowed (in addition to policy) |
| `approval_persistence_enabled` | bool | No | `true` | Enable approval decision persistence |
| `session_policy_ttl` | duration | No | `8h` | TTL for session-scoped policies |
| `global_policy_ttl` | duration | No | `720h` (30 days) | TTL for global policies |

#### Sandbox Modes

- **`none`**: No sandboxing (use with caution)
- **`workspace-only`**: Restrict access to workspace directory only
- **`docker`**: Use Docker container for isolation (if available)
- **`firejail`**: Use Firejail for isolation (if available)

#### Examples

**Restrictive Security:**
```yaml
security:
  sandbox_mode: workspace-only
  policy_file: ~/.spin/policies.json
  allowed_commands: []
  approval_persistence_enabled: true
  session_policy_ttl: 4h
  global_policy_ttl: 168h  # 7 days
```

**Permissive Security (Development):**
```yaml
security:
  sandbox_mode: none
  allowed_commands:
    - git
    - make
    - go
    - npm
    - python
```

**Custom Policy File:**
```yaml
security:
  policy_file: /path/to/custom/policies.json
  approval_persistence_enabled: true
```

### Protocol Configuration (`protocol`)

Configures protocol features: MCP, Git, and Shell integration.

#### Fields

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `enable_mcp` | bool | No | `false` | Enable Model Context Protocol |
| `mcp_servers` | []MCPServer | Conditional | `[]` | MCP server configurations (required if `enable_mcp` is true) |
| `enable_git` | bool | No | `true` | Enable Git integration |
| `enable_shell` | bool | No | `true` | Enable shell command execution |
| `shell_timeout` | duration | No | `5m` | Timeout for shell commands |

#### MCP Server Configuration (`protocol.mcp_servers[]`)

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | string | Yes | Server name (unique identifier) |
| `command` | string | Yes | Command to start the server |
| `args` | []string | No | Command arguments |
| `env` | map[string]string | No | Environment variables |

#### Examples

**Enable MCP with Servers:**
```yaml
protocol:
  enable_mcp: true
  mcp_servers:
    - name: filesystem
      command: mcp-server-filesystem
      args:
        - /workspace
    - name: github
      command: mcp-server-github
      env:
        GITHUB_TOKEN: ${GITHUB_TOKEN}
  enable_git: true
  enable_shell: true
  shell_timeout: 5m
```

**Disable Shell (Read-Only Mode):**
```yaml
protocol:
  enable_mcp: false
  enable_git: true
  enable_shell: false
```

**Custom Shell Timeout:**
```yaml
protocol:
  enable_shell: true
  shell_timeout: 10m
```

## AUTHENTICATION

### Keystore (Recommended)

Store API keys securely in your OS keyring:

```bash
# Store a key
spin config set-key my-openai-key sk-...

# Use in config file
llm:
  provider: openai
  model: gpt-4o
  # Note: key_name is not a config field, use --key-name flag or env var
```

**Usage with flags:**
```bash
spin --provider openai --model gpt-4o --key-name my-openai-key
```

### Environment Variables

Provider-specific environment variables:

- `OPENAI_API_KEY` - OpenAI API key
- `ANTHROPIC_API_KEY` - Anthropic API key
- `GOOGLE_API_KEY` - Google AI API key

General configuration environment variables:

- `SPIN_PROVIDER` - Default provider
- `SPIN_MODEL` - Default model
- `SPIN_WORKDIR` or `SPIN_WORK_DIR` - Working directory
- `SPIN_SANDBOX_MODE` - Sandbox mode
- `SPIN_MAX_TURNS` - Maximum turns
- `SPIN_TIMEOUT` - Operation timeout
- `SPIN_MAX_TOKENS` - Maximum tokens
- `SPIN_ENABLE_GIT` - Enable Git (`true`/`false`)
- `SPIN_ENABLE_SHELL` - Enable shell (`true`/`false`)
- `SPIN_ENABLE_MCP` - Enable MCP (`true`/`false`)

### Direct API Key (Deprecated)

```yaml
llm:
  provider: openai
  model: gpt-4o
  api_key: sk-...  # Deprecated, shows warning
```

**Warning:** Direct API keys in config files are deprecated. Use keystore or environment variables instead.

## COMMAND-LINE FLAGS

Flags override configuration file and environment variables:

### Provider and Model

```bash
--provider ollama              # LLM provider
--model qwen3:0.6b            # Model name
--base-url http://localhost:11434  # Base URL
```

### Authentication

```bash
--key-name my-key             # Key name from keystore
--api-key sk-...              # Direct API key (deprecated)
```

### Execution

```bash
--mode regular                # Task mode: regular, review, compact, planning
--sandbox workspace-write     # Sandbox mode
--timeout 5m                  # Task timeout
--max-turns 50                # Maximum turns
--cd /path/to/workspace       # Working directory
```

### Configuration

```bash
--config-file /path/to/config.yaml  # Config file path
--config llm.model=gpt-4o           # Override specific key
--debug                             # Enable debug mode
```

## COMPLETE CONFIGURATION EXAMPLES

### Example 1: Local Development with Ollama

```yaml
version: "2.0"

llm:
  provider: ollama
  model: qwen3:1.7b
  base_url: http://localhost:11434
  temperature: 0.7
  max_tokens: 8192
  timeout: 30s

agent:
  max_turns: 50
  timeout: 60m
  work_dir: .
  debug: false
  log_level: info
  cycle_detection:
    enabled: true
    window_size: 3
    similarity_thresh: 0.8

protocol:
  enable_git: true
  enable_shell: true
  shell_timeout: 5m

security:
  sandbox_mode: workspace-only
  policy_file: ~/.spin/policies.json
  approval_persistence_enabled: true

ace:
  enabled: true
  playbook_path: ~/.spin/ace/playbooks/default.json
  trajectory_path: ~/.spin/ace/trajectories/
```

### Example 2: Production with OpenAI

```yaml
version: "2.0"

llm:
  provider: openai
  model: gpt-4o
  base_url: https://api.openai.com/v1
  temperature: 0.7
  max_tokens: 4096
  timeout: 60s
  # Use --key-name flag or OPENAI_API_KEY env var

agent:
  max_turns: 20
  timeout: 30m
  work_dir: /workspace
  require_approval: true
  log_level: warn
  log_format: json
  cycle_detection:
    enabled: true

protocol:
  enable_git: true
  enable_shell: false  # Disable shell in production
  enable_mcp: false

security:
  sandbox_mode: workspace-only
  policy_file: ~/.spin/policies.json
  approval_persistence_enabled: true
  session_policy_ttl: 4h
  global_policy_ttl: 168h

ace:
  enabled: false  # Disable ACE in production
```

### Example 3: Code Review Mode

```yaml
version: "2.0"

llm:
  provider: anthropic
  model: claude-3-5-sonnet-20241022
  base_url: https://api.anthropic.com/v1
  temperature: 0.3  # Lower temperature for review
  max_tokens: 8192
  # Use ANTHROPIC_API_KEY env var

agent:
  max_turns: 10
  timeout: 15m
  work_dir: .
  debug: false

protocol:
  enable_git: true
  enable_shell: false  # Read-only for reviews
  enable_mcp: false

security:
  sandbox_mode: workspace-only
  policy_file: ~/.spin/policies.json

ace:
  enabled: false
```

### Example 4: MCP Integration

```yaml
version: "2.0"

llm:
  provider: ollama
  model: qwen3:0.6b

agent:
  max_turns: 50
  timeout: 60m

protocol:
  enable_mcp: true
  mcp_servers:
    - name: filesystem
      command: mcp-server-filesystem
      args:
        - /workspace
    - name: github
      command: mcp-server-github
      env:
        GITHUB_TOKEN: ${GITHUB_TOKEN}
    - name: sqlite
      command: mcp-server-sqlite
      args:
        - /path/to/database.db
  enable_git: true
  enable_shell: true

security:
  sandbox_mode: workspace-only
```

### Example 5: Performance Tuning

```yaml
version: "2.0"

llm:
  provider: ollama
  model: qwen3:1.7b
  timeout: 60s

agent:
  max_turns: 100
  timeout: 2h
  stream_buffer: 200
  cache_commands: true
  max_files: 100
  max_depth: 5
  history_limit: 2000
  cycle_detection:
    enabled: true
    window_size: 5
    similarity_thresh: 0.85
    tool_repeat_limit: 5

protocol:
  enable_git: true
  enable_shell: true
  shell_timeout: 10m

security:
  sandbox_mode: workspace-only
```

## CONFIGURATION COMMANDS

Manage configuration via CLI:

```bash
# Show current configuration
spin config show
spin config show --format json
spin config show --format yaml

# Get a specific value
spin config get llm.model
spin config get agent.max_turns

# Set a value (in-memory, use 'edit' to persist)
spin config set llm.model gpt-4o
spin config set agent.max_turns 100

# Edit configuration file
spin config edit

# Validate configuration
spin config validate
spin config validate --file /path/to/config.yaml

# Show config file path
spin config path
```

## VALIDATION

Spin validates configuration on load. Common validation errors:

- **Missing required fields**: `llm.provider` and `llm.model` are required
- **Invalid values**: Temperature must be 0.0-2.0, timeouts must be positive
- **Invalid sandbox mode**: Must be one of `none`, `workspace-only`, `docker`, `firejail`
- **MCP server errors**: Name and command required when MCP is enabled

Validate your configuration:

```bash
spin config validate
```

## ENVIRONMENT VARIABLE SUBSTITUTION

Configuration files support environment variable substitution:

```yaml
llm:
  provider: openai
  model: ${SPIN_MODEL:-gpt-4o}  # Default to gpt-4o if not set

protocol:
  mcp_servers:
    - name: github
      command: mcp-server-github
      env:
        GITHUB_TOKEN: ${GITHUB_TOKEN}  # Required, no default
```

## PATH EXPANSION

Tilde (`~`) expands to home directory:

```yaml
agent:
  session_dir: ~/.spin/sessions

security:
  policy_file: ~/.spin/policies.json

ace:
  playbook_path: ~/.spin/ace/playbooks/default.json
  trajectory_path: ~/.spin/ace/trajectories/
```

## RELATED DOCUMENTS

- `docs/job-local-agent.md` – using Spin in your terminal
- `docs/job-ci-automation.md` – CI/CD configuration
- `docs/job-acp-ide.md` – IDE integration setup
- `docs/troubleshooting.md` – configuration troubleshooting
- `examples/PROVIDER-CONFIG.md` – additional provider examples
- `examples/config-*.yaml` – example configuration files
