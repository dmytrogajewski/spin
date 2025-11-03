# FRD-20251103-011: Configuration Consolidation (Phase 1)

**Date**: 2025-11-03  
**Author**: Rob Pike (AI Agent)  
**Status**: Draft  
**Priority**: HIGH  
**Related**: ROADMAP.md Phase 1

## 1. Overview

### 1.1 Purpose
Consolidate three separate configuration structures into a single, unified configuration system with multi-source loading (files, env vars, CLI flags) using industry-standard tools (Viper + Cobra).

### 1.2 Background
Current state has 3 duplicate Config structs:
- `internal/agent/config.go` (713 LOC) - Agent configuration with ACE settings
- `internal/config/config.go` - Application configuration
- `internal/protocol/config.go` - Protocol configuration

This violates DRY, causes maintenance issues, and leads to inconsistent defaults.

### 1.3 Goals
- Single `Config` struct in `internal/config/`
- Viper-based loader with automatic precedence (flags > env > file > defaults)
- 95%+ test coverage with property-based and fuzz tests
- Backward compatibility with v1 config format
- Zero lint errors, zero dead code

### 1.4 Non-Goals
- Dynamic config reload (deferred to v2.1)
- Remote config sources (deferred to v2.1)
- Encrypted config values (use env vars instead)

## 2. Requirements

### 2.1 Functional Requirements

**FR-1**: Config struct with versioning
- MUST have `Version` field for schema migration
- MUST have sections: LLM, Agent, ACE, Protocol, Security
- MUST implement `Validate()` and `ApplyDefaults()`

**FR-2**: Multi-source loading with Viper
- MUST load from: defaults < file < env < flags
- MUST search paths: `./config.yaml`, `~/.spin/config.yaml`, `/etc/spin/config.yaml`
- MUST support env vars with prefix `SPIN_` (e.g., `SPIN_LLM_PROVIDER`)
- MUST bind Cobra command flags automatically

**FR-3**: Validation with helpful errors
- MUST return ALL validation errors (not fail-fast)
- MUST validate each section independently
- MUST validate cross-section dependencies (e.g., ACE enabled → playbook required)
- MUST provide clear error messages with field paths

**FR-4**: Backward compatibility
- MUST load v1 config format with automatic migration
- MUST emit warnings for deprecated fields
- MUST support v1 until Spin v3.0

**FR-5**: Observability
- MUST track config source per field (default, file, env, flag)
- MUST log config file path when loaded
- MUST expose overrides for debugging

### 2.2 Non-Functional Requirements

**NFR-1**: Performance
- Config loading MUST complete in <10ms (99th percentile)
- Validation MUST be O(1) per field

**NFR-2**: Testability
- MUST achieve 95%+ line coverage
- MUST have property-based tests for round-trip serialization
- MUST have fuzz tests for unmarshaling
- MUST have mutation test score ≥70%

**NFR-3**: Maintainability
- MUST have zero lint errors (`golangci-lint --enable-all`)
- MUST have zero dead code
- MUST have clear separation of concerns

**NFR-4**: Documentation
- MUST have SPEC.md with invariants and constraints
- MUST have MIGRATION.md for v1→v2 upgrade
- MUST have examples for all config sources

## 3. Design

### 3.1 Architecture

```
internal/config/
├── config.go           # Main Config struct + validation
├── llm.go             # LLMConfig + defaults + validation
├── agent.go           # AgentConfig + defaults + validation
├── ace.go             # ACEConfig + defaults + validation
├── protocol.go        # ProtocolConfig + defaults + validation
├── security.go        # SecurityConfig + defaults + validation
├── loader.go          # Viper-based Loader
├── compat.go          # V1 migration layer
├── config_test.go     # Core tests
├── loader_test.go     # Loader tests
├── compat_test.go     # Migration tests
├── golden/            # Golden test configs
│   ├── valid_minimal.yaml
│   ├── valid_full.yaml
│   ├── invalid_missing_required.yaml
│   └── v1_config.yaml
```

### 3.2 Data Model

```go
// Config is the unified configuration for Spin v2.0.
type Config struct {
    Version  string         `yaml:"version"`
    LLM      LLMConfig      `yaml:"llm"`
    Agent    AgentConfig    `yaml:"agent"`
    ACE      ACEConfig      `yaml:"ace"`
    Protocol ProtocolConfig `yaml:"protocol"`
    Security SecurityConfig `yaml:"security"`
}

// LLMConfig configures the LLM provider.
type LLMConfig struct {
    Provider    string        `yaml:"provider" validate:"required,oneof=ollama openai lmstudio"`
    Model       string        `yaml:"model" validate:"required"`
    BaseURL     string        `yaml:"base_url"`
    APIKey      string        `yaml:"api_key"`
    Temperature float64       `yaml:"temperature" validate:"gte=0,lte=2"`
    MaxTokens   int           `yaml:"max_tokens" validate:"gt=0"`
    Timeout     time.Duration `yaml:"timeout" validate:"gt=0"`
}

// AgentConfig configures the agent behavior.
type AgentConfig struct {
    MaxTurns        int           `yaml:"max_turns" validate:"gt=0"`
    Timeout         time.Duration `yaml:"timeout" validate:"gt=0"`
    MaxTokens       int           `yaml:"max_tokens" validate:"gt=0"`
    RequireApproval bool          `yaml:"require_approval"`
}

// ACEConfig configures Agentic Context Engineering.
type ACEConfig struct {
    Enabled          bool                      `yaml:"enabled"`
    PlaybookPath     string                    `yaml:"playbook_path"`
    TrajectoryPath   string                    `yaml:"trajectory_path"`
    Retrieval        ACERetrievalConfig        `yaml:"retrieval"`
    ItemizedLearning ACEItemizedLearningConfig `yaml:"itemized_learning"`
    Generation       ACEGenerationConfig       `yaml:"generation"`
}
```

### 3.3 Validation Rules

**Independent Validation** (per section):
- LLM.Provider: required, one of [ollama, openai, lmstudio]
- LLM.Model: required, non-empty
- LLM.Temperature: 0 ≤ temp ≤ 2
- LLM.MaxTokens: > 0
- Agent.MaxTurns: > 0
- Agent.Timeout: > 0

**Cross-Section Validation**:
- IF ACE.Enabled THEN ACE.PlaybookPath MUST be non-empty
- IF ACE.Enabled THEN LLM.Provider MUST be set
- Agent.Timeout MUST be > LLM.Timeout

### 3.4 Multi-Source Loading

**Precedence** (highest to lowest):
1. CLI flags (`--llm-provider=ollama`)
2. Environment variables (`SPIN_LLM_PROVIDER=ollama`)
3. Config file (`llm.provider: ollama`)
4. Defaults (from `DefaultConfig()`)

**Config File Search Paths** (first found wins):
1. `./config.yaml`
2. `~/.spin/config.yaml`
3. `/etc/spin/config.yaml`
4. Explicit: `--config=/path/to/config.yaml`

### 3.5 Error Handling

**Validation Errors**:
```
validation failed: 3 errors found:
- llm: provider: required field missing
- llm: temperature: must be between 0 and 2, got 3.5
- ace: playbook_path required when ace.enabled=true
```

**Config File Errors**:
```
failed to read config: /home/user/.spin/config.yaml: yaml: line 5: invalid syntax
```

**Migration Warnings**:
```
loaded v1 config with warnings:
- field 'provider' is deprecated, use 'llm.provider' instead
- field 'temperature' moved from root to llm.temperature
```

## 4. Implementation Plan

### 4.1 Phase 1A: Core Config Struct (Days 1-2)

**Micro-Steps**:
1. Create `internal/config/config.go` with `Config` struct
2. Add `Validate()` method with independent validation
3. Add `ApplyDefaults()` method
4. Add cross-section validation
5. Test: validate accepts valid config
6. Test: validate rejects invalid config with clear errors
7. Test: validate checks cross-section rules

### 4.2 Phase 1B: Viper Loader (Days 3-4)

**Micro-Steps**:
1. Create `internal/config/loader.go` with `Loader` struct
2. Add `NewLoader()` with Viper initialization
3. Add `Load()` method with file loading
4. Add environment variable binding
5. Add flag binding (for Cobra integration)
6. Test: load from defaults
7. Test: load from file
8. Test: load from env vars
9. Test: precedence (flags > env > file > defaults)

### 4.3 Phase 1C: Backward Compatibility (Day 5)

**Micro-Steps**:
1. Create `internal/config/compat.go` with `LoadV1()`
2. Add v1→v2 field mapping
3. Add deprecation warnings
4. Test: load v1 config successfully
5. Test: emit warnings for deprecated fields

### 4.4 Phase 1D: Full Test Suite (Days 6-7)

**Micro-Steps**:
1. Property-based tests: round-trip serialization
2. Fuzz tests: unmarshal arbitrary YAML
3. Golden tests: valid/invalid config files
4. Mutation tests: ensure validation catches bugs
5. Integration tests: Viper + Cobra end-to-end

### 4.5 Phase 1E: Migration & Cleanup (Day 8)

**Micro-Steps**:
1. Update all references: `agent.Config` → `config.Config`
2. Delete old config files: `agent/config.go`, `protocol/config.go`
3. Update documentation: SPEC.md, MIGRATION.md
4. Run `make lint` and fix all issues
5. Run UAST analysis and fix dead code

## 5. Test Strategy

### 5.1 Unit Tests (Target: 95% coverage)

```go
// Core validation
func TestConfig_Validate_ValidConfig(t *testing.T)
func TestConfig_Validate_MissingRequired(t *testing.T)
func TestConfig_Validate_InvalidRange(t *testing.T)
func TestConfig_Validate_CrossSection(t *testing.T)

// Defaults
func TestConfig_ApplyDefaults(t *testing.T)
func TestConfig_DefaultsAreValid(t *testing.T)

// Loader
func TestLoader_LoadFromDefaults(t *testing.T)
func TestLoader_LoadFromFile(t *testing.T)
func TestLoader_LoadFromEnv(t *testing.T)
func TestLoader_Precedence(t *testing.T)

// Backward compat
func TestCompat_LoadV1(t *testing.T)
func TestCompat_DeprecationWarnings(t *testing.T)
```

### 5.2 Property-Based Tests

```go
func TestConfig_RoundTrip(t *testing.T) {
    rapid.Check(t, func(t *rapid.T) {
        cfg := genValidConfig(t)
        data, _ := yaml.Marshal(cfg)
        var cfg2 Config
        yaml.Unmarshal(data, &cfg2)
        assert.Equal(t, cfg, cfg2)
    })
}
```

### 5.3 Fuzz Tests

```go
func FuzzConfig_Unmarshal(f *testing.F) {
    f.Add([]byte(`version: "2.0"\nllm:\n  provider: ollama`))
    f.Fuzz(func(t *testing.T, data []byte) {
        var cfg Config
        _ = yaml.Unmarshal(data, &cfg)
        // Should never panic
    })
}
```

### 5.4 Mutation Tests

Run `go-mutesting` on validation logic to ensure:
- All validation rules are tested
- Mutation score ≥70%

## 6. Acceptance Criteria

### 6.1 Functional

- [ ] Single Config struct in `internal/config/`
- [ ] Viper loader supports files, env vars, flags
- [ ] All validation rules implemented
- [ ] V1 migration works with warnings
- [ ] Cobra integration example provided

### 6.2 Quality

- [ ] Line coverage ≥95%
- [ ] Branch coverage ≥85%
- [ ] Mutation score ≥70%
- [ ] Zero lint errors
- [ ] Zero dead code
- [ ] All tests pass
- [ ] Performance: config load <10ms

### 6.3 Documentation

- [ ] SPEC.md with invariants
- [ ] MIGRATION.md with v1→v2 guide
- [ ] Examples for all config sources
- [ ] Godoc comments on all exported types
- [ ] ROADMAP.md Phase 1 marked complete

## 7. Risks & Mitigations

### 7.1 Risk: Breaking Changes

**Impact**: HIGH  
**Probability**: HIGH  
**Mitigation**:
- V1 compatibility layer for one release
- Clear migration guide with examples
- Deprecation warnings in logs

### 7.2 Risk: Performance Regression

**Impact**: MEDIUM  
**Probability**: LOW  
**Mitigation**:
- Benchmark tests in CI
- Performance budget: <10ms load time
- Profile before/after

### 7.3 Risk: Incomplete Migration

**Impact**: MEDIUM  
**Probability**: MEDIUM  
**Mitigation**:
- Comprehensive test suite
- Manual testing of all config scenarios
- Rollback procedure documented

## 8. Dependencies

### 8.1 External Libraries

- `github.com/spf13/viper v1.18.2` - Config management
- `github.com/spf13/cobra v1.8.0` - CLI flags
- `pgregory.net/rapid v1.1.0` - Property testing
- `github.com/zimmski/go-mutesting` - Mutation testing

### 8.2 Internal Dependencies

- `internal/events` - For config change events (future)
- `internal/llm` - Uses Config.LLM
- `internal/agent` - Uses Config.Agent
- `internal/ace` - Uses Config.ACE

## 9. Rollback Plan

If Phase 1 fails:

1. Revert commits: `git revert <phase1-start>..<phase1-end>`
2. Re-enable old config files
3. Document failure in `LESSONS-LEARNED.md`
4. Re-plan with additional safeguards

Rollback triggers:
- Test coverage <95%
- Performance regression >20%
- Critical bugs in production
- Cannot meet acceptance criteria

## 10. Success Metrics

### 10.1 Quantitative

- Config files: 3 → 1 ✅
- Code duplication: -40% ✅
- Test coverage: 75% → 95% ✅
- Config load time: <10ms ✅

### 10.2 Qualitative

- Clear config precedence rules ✅
- Easy to add new config fields ✅
- Helpful validation errors ✅
- Standard Go patterns (Viper/Cobra) ✅

---

**Next Steps**:
1. Create `internal/config/config.go` with Config struct
2. Write first failing test: `TestConfig_Validate_ValidConfig`
3. Follow micro-TDD loop until all acceptance criteria met
