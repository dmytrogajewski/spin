# FRD-0.3: Configuration System

**Feature ID:** 0.3  
**Feature Name:** Configuration System  
**Phase:** 0 - Foundation & Setup  
**Priority:** P0 (Blocker)  
**Estimated Effort:** 8 hours  
**Status:** ✅ Complete  

---

## Overview

Implement the configuration system with file loading, environment variables, CLI flag merging, and validation. This provides the foundation for configuring the Spin agent with proper precedence and validation.

## Business Value

- Enables flexible configuration from multiple sources (files, env vars, CLI)
- Provides sane defaults for quick start
- Validates configuration to prevent runtime errors
- Supports configuration precedence (CLI > Env > File > Defaults)
- Makes agent behavior configurable per deployment

## Functional Requirements

### FR-0.3.1: Config Structure
Define comprehensive Config struct with all agent settings:

```go
type Config struct {
    // LLM Provider Configuration
    Provider       string
    Model          string
    ProviderConfig map[string]interface{}
    
    // Execution Configuration
    MaxTurns       int
    Timeout        time.Duration
    WorkDir        string
    
    // Security Configuration
    SandboxMode    string
    PolicyFile     string
    AllowedCommands []string
    
    // Feature Flags
    EnableMCP      bool
    MCPServers     []MCPServerConfig
    EnableGit      bool
    EnableShell    bool
    
    // Performance Configuration
    MaxTokens      int
    StreamBuffer   int
    CacheCommands  bool
    
    // Storage Configuration
    SessionDir     string
    HistoryLimit   int
}

type MCPServerConfig struct {
    Name    string
    Command string
    Args    []string
    Env     map[string]string
}
```

### FR-0.3.2: Load Configuration from File
Implement `Load(path string) (*Config, error)` function:
- Support YAML format (using `gopkg.in/yaml.v3`)
- Handle missing file gracefully (return defaults)
- Return error for invalid YAML
- Parse nested structures (MCPServerConfig)
- Handle environment variable expansion (e.g., `${HOME}`)

### FR-0.3.3: Configuration Validation
Implement `Validate() error` method:
- Validate required fields (Provider, Model, WorkDir)
- Validate value ranges (MaxTurns > 0, Timeout > 0)
- Validate file paths exist (PolicyFile, SessionDir)
- Validate SandboxMode is one of: "read-only", "workspace-only", "full-access"
- Validate Provider is supported
- Return detailed validation errors with field names

### FR-0.3.4: Configuration Merging
Implement `Merge(other *Config) *Config` method:
- Merge two configurations with precedence
- Non-zero values override zero values
- Slices are appended (e.g., AllowedCommands)
- Maps are merged (later overwrites earlier)
- Return new Config (immutable operation)

### FR-0.3.5: Default Configuration
Implement `DefaultConfig() *Config` function:
- Return sensible defaults for all fields
- MaxTurns: 50
- Timeout: 5 minutes
- MaxTokens: 8192
- StreamBuffer: 100
- HistoryLimit: 1000
- SessionDir: `~/.spin/sessions`
- EnableGit: true
- EnableShell: true
- SandboxMode: "workspace-only"

### FR-0.3.6: Environment Variable Support
Implement environment variable loading:
- `SPIN_PROVIDER` - LLM provider
- `SPIN_MODEL` - Model name
- `SPIN_WORKDIR` - Working directory
- `SPIN_SANDBOX_MODE` - Sandbox mode
- `SPIN_MAX_TURNS` - Maximum turns
- `SPIN_TIMEOUT` - Operation timeout
- Support standard env var parsing (strings, ints, durations, bools)

### FR-0.3.7: Configuration File Discovery
Implement configuration file discovery:
- Check explicit path if provided
- Check `./spin.yaml`
- Check `~/.spin/config.yaml`
- Check `/etc/spin/config.yaml`
- Return first found, or defaults if none found

### FR-0.3.8: Configuration Precedence
Implement full precedence chain:
1. CLI flags (highest priority) - handled by caller
2. Environment variables
3. Configuration file
4. Defaults (lowest priority)

## Non-Functional Requirements

### NFR-0.3.1: Performance
- Configuration loading should complete in <100ms
- File parsing should handle files up to 1MB
- No unnecessary allocations during merge

### NFR-0.3.2: Usability
- Clear validation error messages with field names
- Example configuration file provided
- Defaults work out of the box for common use cases

### NFR-0.3.3: Security
- No secrets in configuration files (use env vars)
- Validate file permissions on PolicyFile
- Sanitize paths (no directory traversal)

### NFR-0.3.4: Maintainability
- Easy to add new configuration fields
- Clear documentation for each field
- Configuration struct is serializable

## Technical Design

### Configuration Loading Flow
```go
func LoadConfig(path string) (*Config, error) {
    // 1. Start with defaults
    cfg := DefaultConfig()
    
    // 2. Load from file if exists
    if path != "" {
        fileCfg, err := loadFromFile(path)
        if err != nil && !os.IsNotExist(err) {
            return nil, err
        }
        if fileCfg != nil {
            cfg = cfg.Merge(fileCfg)
        }
    }
    
    // 3. Override with environment variables
    envCfg := loadFromEnv()
    cfg = cfg.Merge(envCfg)
    
    // 4. Validate final configuration
    if err := cfg.Validate(); err != nil {
        return nil, err
    }
    
    return cfg, nil
}
```

### YAML Configuration Example
```yaml
# ~/.spin/config.yaml

# LLM Provider
provider: ollama
model: codellama:13b
provider_config:
  temperature: 0.7
  top_p: 0.9

# Execution
max_turns: 50
timeout: 5m
work_dir: "."

# Security
sandbox_mode: workspace-only
policy_file: ~/.spin/policy.star
allowed_commands:
  - git
  - make
  - go

# Features
enable_mcp: true
mcp_servers:
  - name: filesystem
    command: mcp-server-filesystem
    args: ["/workspace"]
  - name: github
    command: mcp-server-github
    env:
      GITHUB_TOKEN: ${GITHUB_TOKEN}

enable_git: true
enable_shell: true

# Performance
max_tokens: 8192
stream_buffer: 100
cache_commands: true

# Storage
session_dir: ~/.spin/sessions
history_limit: 1000
```

### Validation Logic
```go
func (c *Config) Validate() error {
    var errs []error
    
    if c.Provider == "" {
        errs = append(errs, fmt.Errorf("provider is required"))
    }
    
    if c.Model == "" {
        errs = append(errs, fmt.Errorf("model is required"))
    }
    
    if c.MaxTurns <= 0 {
        errs = append(errs, fmt.Errorf("max_turns must be > 0, got %d", c.MaxTurns))
    }
    
    if c.Timeout <= 0 {
        errs = append(errs, fmt.Errorf("timeout must be > 0, got %v", c.Timeout))
    }
    
    // Check sandbox mode
    validModes := []string{"read-only", "workspace-only", "full-access"}
    if !contains(validModes, c.SandboxMode) {
        errs = append(errs, fmt.Errorf("invalid sandbox_mode: %s", c.SandboxMode))
    }
    
    if len(errs) > 0 {
        return errors.Join(errs...)
    }
    
    return nil
}
```

## Definition of Ready (DoR)

- [x] Feature 0.2 completed
- [x] Configuration schema defined
- [x] Default values determined
- [x] YAML parser available (`gopkg.in/yaml.v3`)

## Definition of Done (DoD)

- [ ] `config.go` implemented with Config struct
- [ ] `Load()` function for YAML config files
- [ ] `LoadConfig()` with file discovery
- [ ] `Validate()` method with comprehensive checks
- [ ] `Merge()` method for configuration precedence
- [ ] `DefaultConfig()` function returning defaults
- [ ] Environment variable support (SPIN_*)
- [ ] Configuration file discovery (multiple locations)
- [ ] Path expansion (~, ${VAR})
- [ ] MCPServerConfig struct and parsing
- [ ] Unit tests for all config operations (>90% coverage)
- [ ] Unit tests for YAML parsing
- [ ] Unit tests for validation (all error cases)
- [ ] Unit tests for merging
- [ ] Unit tests for environment variable loading
- [ ] Integration tests with actual config files
- [ ] Example config file (`configs/example.yaml`)
- [ ] All tests passing
- [ ] Code passes linter without errors
- [ ] Configuration documented in README
- [ ] Godoc comments for all exported symbols

## Testing Strategy

### Unit Tests

**Test File:** `internal/core/config_test.go`

Test cases:
1. **TestDefaultConfig** - Verify default values
2. **TestConfig_Load_ValidYAML** - Load valid YAML file
3. **TestConfig_Load_InvalidYAML** - Handle invalid YAML
4. **TestConfig_Load_MissingFile** - Handle missing file gracefully
5. **TestConfig_Validate_Valid** - Valid configuration passes
6. **TestConfig_Validate_MissingProvider** - Missing provider fails
7. **TestConfig_Validate_MissingModel** - Missing model fails
8. **TestConfig_Validate_InvalidMaxTurns** - Invalid MaxTurns fails
9. **TestConfig_Validate_InvalidTimeout** - Invalid Timeout fails
10. **TestConfig_Validate_InvalidSandboxMode** - Invalid SandboxMode fails
11. **TestConfig_Merge_BasicFields** - Merge basic fields
12. **TestConfig_Merge_Slices** - Merge slice fields
13. **TestConfig_Merge_Maps** - Merge map fields
14. **TestConfig_Merge_Precedence** - Later overrides earlier
15. **TestLoadFromEnv** - Load from environment variables
16. **TestLoadFromEnv_TypeParsing** - Parse int, duration, bool
17. **TestLoadConfig_FileOnly** - Load from file only
18. **TestLoadConfig_EnvOverride** - Env overrides file
19. **TestLoadConfig_FullPrecedence** - Complete precedence chain
20. **TestMCPServerConfig_Parsing** - Parse MCP server config
21. **TestPathExpansion** - Expand ~ and ${VAR}

### Integration Tests

**Test File:** `internal/core/config_integration_test.go`

1. **TestLoadConfig_RealFile** - Load actual config file
2. **TestLoadConfig_FileDiscovery** - Test file discovery order
3. **TestConfig_Serialization** - Round-trip YAML encoding/decoding

### Test Data

Create test fixtures in `internal/core/testdata/`:
- `valid_config.yaml` - Valid configuration
- `invalid_syntax.yaml` - Invalid YAML syntax
- `invalid_values.yaml` - Valid YAML, invalid values
- `minimal_config.yaml` - Minimal valid config
- `complete_config.yaml` - All fields populated

### Coverage Target
- Minimum 90% coverage for config.go
- 100% coverage for validation paths
- All error cases tested

## Implementation Tasks

1. Create `internal/core/config_test.go` with all test cases (TDD)
2. Create test fixtures in `testdata/`
3. Implement `Config` struct with all fields
4. Implement `MCPServerConfig` struct
5. Implement `DefaultConfig()` function
6. Implement `Load()` function for YAML parsing
7. Implement `Validate()` method
8. Implement `Merge()` method
9. Implement environment variable loading (`loadFromEnv()`)
10. Implement path expansion (tilde, env vars)
11. Implement `LoadConfig()` with file discovery
12. Run tests and fix failures
13. Create `configs/example.yaml`
14. Add godoc comments
15. Run linter and fix issues
16. Analyze with uast/herr
17. Update ROADMAP

## Dependencies

### Prerequisites
- Feature 0.1 (Project Structure) completed
- Feature 0.2 (Core Types & Errors) completed
- `gopkg.in/yaml.v3` dependency available

### Blocks
- Feature 1.1 (Session Management) - needs Config for session directory
- Feature 2.1 (Command Validator) - needs Config for security settings
- Feature 3.1 (Environment Context) - needs Config for WorkDir
- All other features - depend on configuration

### Blocked By
- None

## Risks and Mitigations

### Risk 1: Complex Merging Logic
**Impact:** Merge behavior might be unintuitive  
**Mitigation:** Extensive tests, clear documentation, simple precedence rules

### Risk 2: Environment Variable Type Parsing
**Impact:** Type conversion errors at runtime  
**Mitigation:** Validate after parsing, clear error messages

### Risk 3: Configuration Explosion
**Impact:** Too many config options make it hard to use  
**Mitigation:** Sensible defaults, example configs, progressive disclosure

## Success Criteria

1. All tests passing with >90% coverage
2. Can load configuration from file, env, and defaults
3. Validation catches all common errors
4. Merge works correctly for all field types
5. Linter passes without errors
6. Example config file works out of the box
7. Documentation is comprehensive

## Examples

### Basic Usage
```go
// Load configuration
cfg, err := LoadConfig("~/.spin/config.yaml")
if err != nil {
    log.Fatal(err)
}

// Use configuration
manager, err := NewManager(cfg)
```

### Programmatic Configuration
```go
// Start with defaults
cfg := DefaultConfig()

// Override specific fields
cfg.Provider = "ollama"
cfg.Model = "codellama:13b"
cfg.WorkDir = "/home/user/project"

// Validate
if err := cfg.Validate(); err != nil {
    log.Fatal(err)
}
```

### Merging Configurations
```go
// Base configuration
base := DefaultConfig()

// Override from file
file, _ := Load("config.yaml")
cfg := base.Merge(file)

// Override from environment
env := loadFromEnv()
cfg = cfg.Merge(env)
```

## Notes

- Use `gopkg.in/yaml.v3` for YAML parsing (already in dependencies)
- Support `time.Duration` parsing for timeout values
- Use `os.ExpandEnv()` for environment variable expansion
- Use `os.UserHomeDir()` for tilde expansion
- Configuration is immutable after validation
- Sensitive values (API keys) should use env vars, not config files

## References

- [Viper Configuration Library](https://github.com/spf13/viper) - inspiration
- [12 Factor App - Config](https://12factor.net/config) - best practices
- [Go YAML v3](https://github.com/go-yaml/yaml) - YAML parsing
- [Core Module Specification](../core-module/spec.md)
- [Architecture Overview](../architecture-overview.md)

---

**Created:** 2025-10-03  
**Author:** Development Team  
**Reviewers:** TBD  
**Approved:** TBD

