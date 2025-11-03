# ConfigV2 Specification

## Overview

ConfigV2 is the unified configuration system for Spin v2.0. It replaces the flat V1 configuration with a structured, section-based design that supports multiple configuration sources and provides comprehensive validation.

## Design Principles

1. **Section-based organization**: Configuration is divided into logical sections (LLM, Agent, ACE, Security, Protocol)
2. **Multi-source loading**: Supports configuration from files, environment variables, and CLI flags with proper precedence
3. **Comprehensive validation**: Returns all validation errors at once, not fail-fast
4. **Type safety**: Strong typing with explicit field types and constraints
5. **Backward compatibility**: V1 configurations can be automatically migrated to V2

## Configuration Structure

```yaml
version: "2.0"

llm:
  provider: string        # Required: LLM provider (ollama, openai, anthropic, etc.)
  model: string           # Required: Model identifier
  api_key: string         # Optional: API key for the provider
  base_url: string        # Optional: Custom base URL
  timeout: duration       # Required: HTTP request timeout (> 0)
  max_tokens: int         # Required: Maximum tokens (> 0)
  temperature: float64    # Required: Temperature [0.0, 2.0]

agent:
  max_turns: int          # Required: Maximum conversation turns (> 0)
  timeout: duration       # Required: Agent execution timeout (> 0)
  work_dir: string        # Required: Working directory path
  require_approval: bool  # Required: Whether to require user approval

ace:
  enabled: bool           # Required: Enable ACE features
  playbook_path: string   # Required if enabled: Path to playbook file
  trajectory_path: string # Required if enabled: Path to trajectory storage
  top_k: int              # Optional: Top K results for retrieval (≥ 0)
  min_score: float64      # Optional: Minimum similarity score [0.0, 1.0]

security:
  sandbox_mode: string    # Required: Sandbox mode (none, docker, firejail)
  policy_file: string     # Optional: Path to security policy file
  allowed_commands: []string # Optional: List of allowed shell commands

protocol:
  enable_mcp: bool        # Required: Enable MCP protocol
  mcp_servers: []MCPServer # Optional: List of MCP servers
  enable_git: bool        # Required: Enable Git protocol
  enable_shell: bool      # Required: Enable shell execution
  shell_timeout: duration # Required if enable_shell: Shell command timeout (> 0)
```

## Invariants

### Type Invariants

1. **Duration fields**: All duration fields must be positive (> 0)
   - `llm.timeout`
   - `agent.timeout`
   - `protocol.shell_timeout`

2. **Integer fields**: All integer fields must be positive (> 0) or non-negative (≥ 0)
   - Positive: `llm.max_tokens`, `agent.max_turns`
   - Non-negative: `ace.top_k`

3. **Float fields**: Must be within specified ranges
   - `llm.temperature`: [0.0, 2.0]
   - `ace.min_score`: [0.0, 1.0]

4. **String fields**: Required string fields must not be empty
   - `llm.provider`
   - `llm.model`
   - `agent.work_dir`

5. **Enum fields**: Must be one of allowed values
   - `security.sandbox_mode`: "none", "docker", "firejail"

### Cross-Section Invariants

1. **ACE enabled requires paths**:
   - If `ace.enabled == true`, then:
     - `ace.playbook_path` must not be empty
     - `ace.trajectory_path` must not be empty

2. **Shell enabled requires timeout**:
   - If `protocol.enable_shell == true`, then:
     - `protocol.shell_timeout` must be > 0

3. **MCP enabled requires servers**:
   - If `protocol.enable_mcp == true`, then:
     - `protocol.mcp_servers` should be configured (not enforced, but recommended)

### Version Invariant

- `version` field must be "2.0" for V2 configurations

## Validation Behavior

### Comprehensive Error Collection

ConfigV2 validation collects ALL errors before returning, rather than failing on the first error. This provides users with a complete picture of what needs to be fixed.

Example error output:
```
validation failed: 4 errors found:
  1. llm: provider is required
  2. llm: temperature must be between 0 and 2, got 3.50
  3. agent: max_turns must be positive, got 0
  4. ace: playbook_path is required when ACE is enabled
```

### Validation Order

1. Section-level validation (LLM, Agent, ACE, Security, Protocol)
2. Cross-section validation (dependencies between sections)
3. All errors are collected and returned together

## Configuration Sources

### Source Precedence (highest to lowest)

1. **CLI flags** (not yet implemented in Phase 1)
2. **Environment variables** (SPIN_LLM_PROVIDER, etc.)
3. **Configuration file** (YAML)
4. **Default values**

### Environment Variable Naming

Environment variables follow the pattern: `SPIN_<SECTION>_<FIELD>`

Examples:
- `SPIN_LLM_PROVIDER=ollama`
- `SPIN_LLM_MODEL=qwen2.5-coder:32b`
- `SPIN_AGENT_MAX_TURNS=20`
- `SPIN_ACE_ENABLED=true`

### Default Values

LoaderV2 applies smart defaults for optional fields. See `loader_v2.go:applyDefaults()` for the complete list.

## File Format

### Supported Formats

- **Primary**: YAML (.yaml, .yml)
- **Future**: JSON, TOML (via Viper)

### File Locations (searched in order)

1. Explicit path provided to `LoadFromFile(path)`
2. `./config.yaml`
3. `~/.config/spin/config.yaml`
4. `/etc/spin/config.yaml`

## Migration from V1

### Automatic Migration

V1 configurations can be automatically migrated using `MigrateV1ToV2()` or loaded with `LoadV1Compatible()`.

### Field Mappings

| V1 Field | V2 Field |
|----------|----------|
| `provider` | `llm.provider` |
| `model` | `llm.model` |
| `temperature` | `llm.temperature` |
| `max_tokens` | `llm.max_tokens` |
| `timeout` → duration | `agent.timeout` |
| `llm_timeout` → duration | `llm.timeout` |
| `max_turns` | `agent.max_turns` |
| `work_dir` | `agent.work_dir` |
| `require_approval` | `agent.require_approval` |
| `ace_enabled` | `ace.enabled` |
| `ace_playbook_path` | `ace.playbook_path` |
| `ace_trajectory_path` | `ace.trajectory_path` |
| `sandbox_mode` | `security.sandbox_mode` |
| `policy_file` | `security.policy_file` |
| `allowed_commands` | `security.allowed_commands` |
| `enable_mcp` | `protocol.enable_mcp` |

### Breaking Changes

1. **Durations**: V1 used integer seconds, V2 uses Go duration strings (e.g., "60s", "5m")
2. **Structure**: Flat structure replaced with nested sections
3. **Validation**: V2 has stricter validation with cross-section rules

## Testing

### Test Coverage

ConfigV2 has comprehensive test coverage including:

1. **Unit tests**: Individual validation rules (`config_v2_test.go`)
2. **Property-based tests**: Random config generation and round-trip testing (`property_test.go`)
3. **Fuzz tests**: Native Go fuzzing for YAML unmarshaling (`fuzz_test.go`)
4. **Golden tests**: Known-good and known-bad configurations (`golden_test.go`)
5. **Mutation testing**: 94.19% test efficacy, 69.35% mutator coverage

### Test Results

```
✓ Unit tests: 38+ test cases
✓ Property tests: 100 iterations, 0 failures
✓ Fuzz tests: 157k+ executions, 0 crashes
✓ Golden tests: 6 files, 11 test functions
✓ Mutation testing: 81/86 mutations killed (94.19%)
```

## Implementation Notes

### Tag Requirements

All struct fields must have both `yaml` and `mapstructure` tags:

```go
type LLMConfigV2 struct {
    Provider string `yaml:"provider" mapstructure:"provider"`
    Model    string `yaml:"model" mapstructure:"model"`
}
```

- `yaml`: For direct YAML marshaling/unmarshaling
- `mapstructure`: Required by Viper for multi-source configuration

### Error Handling

Use `ValidationErrors` type to collect multiple errors:

```go
errs := &ValidationErrors{}
if condition {
    errs.Add(fmt.Errorf("field: description"))
}
return errs.ToError()
```

### Loader Usage

```go
// Load from file with environment overrides
loader := NewLoaderV2()
cfg, err := loader.LoadFromFileWithEnv("config.yaml")

// Load from environment only
cfg, err := loader.LoadWithEnv()

// Load with auto-search and defaults
cfg, err := loader.Load()
```

## Future Enhancements (Post-Phase 1)

1. **CLI flag support**: `--llm-provider=ollama` overrides
2. **Config validation command**: `spin config validate`
3. **Config generation**: `spin config init --interactive`
4. **Hot reload**: Watch config file for changes
5. **Config templates**: Pre-configured profiles for common use cases
6. **Secrets management**: Integration with vault/keyring for API keys

## References

- FRD: `specs/frds/FRD-20251103-011-config-consolidation-phase1.md`
- Implementation: `internal/config/config_v2.go`
- Tests: `internal/config/*_test.go`
- Migration: `internal/config/MIGRATION.md`
