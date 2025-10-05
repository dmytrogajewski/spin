# FRD-CORE-0.4: Provider Factory Integration in CMD Modules

**Status:** ✅ COMPLETE (2025-10-05)
**Priority:** P1 (Important - enables real LLM usage)
**Component:** internal/llm/factory, cmd/spin, internal/exec
**Related:** Phase 0.4 of UI Modules Roadmap

## Implementation Summary

**Completed:** 2025-10-05

**What Was Built:**
- ✅ Provider builder package (`internal/llm/builder`) - 98.2% test coverage
- ✅ Integration with cmd/spin/exec.go - uses real providers
- ✅ Updated internal/exec/runner.go with `RunWithProvider()`
- ✅ Configuration precedence (flags > env vars > config file > defaults)
- ✅ All authentication methods (keystore, direct key, env vars)
- ✅ Example configurations for all providers
- ✅ Comprehensive documentation

**Test Coverage:**
- internal/llm/builder: 98.2%
- internal/exec: 60.3%
- All tests passing ✅
- Linter clean ✅
- Complexity ≤15 ✅

**Files Created:**
- internal/llm/builder/builder.go
- internal/llm/builder/builder_test.go
- internal/llm/builder/doc.go
- examples/config-ollama.yaml
- examples/config-openai.yaml
- examples/config-lmstudio.yaml
- examples/config-custom.yaml
- examples/PROVIDER-CONFIG.md

**Files Modified:**
- cmd/spin/exec.go (integrated provider builder)
- internal/exec/runner.go (added RunWithProvider)

---

---

## Overview

Integrate the existing `internal/llm/factory` package with cmd modules to enable real LLM provider usage instead of mock providers. This unblocks real-world usage of `spin exec` and future `spin tui` by connecting to actual LLM providers (OpenAI, Ollama, LMStudio, etc.).

---

## Problem Statement

**Current State:**
- `cmd/spin/exec.go` uses hardcoded mock providers
- Provider factory exists in `internal/llm/factory` but not integrated
- Auth manager exists in `internal/auth` but not used by cmd modules
- Global flags `--provider` and `--model` are defined but not utilized
- No way to connect to real LLM providers from CLI

**Impact:**
- Users cannot use spin with real LLM providers
- All executions use mock responses
- CLI is not production-ready

**Goal:**
Enable real LLM provider connections via:
1. Command-line flags (`--provider`, `--model`, `--api-key`)
2. Configuration file (YAML/TOML/JSON)
3. Environment variables (fallback)
4. Secure keystore credentials (recommended)

---

## Requirements

### Functional Requirements

**FR-1: Provider Configuration Sources**
- Support provider config from multiple sources with precedence:
  1. CLI flags (highest priority)
  2. Config file (`~/.spin/spin.yaml` or `--config-file`)
  3. Environment variables (e.g., `SPIN_PROVIDER`, `OPENAI_API_KEY`)
  4. Defaults (lowest priority)

**FR-2: Provider Types**
- Support all provider types from factory:
  - `openai` - OpenAI API
  - `ollama` - Local Ollama
  - `lmstudio` - Local LMStudio
  - `openai-compatible` - Generic OpenAI-compatible APIs
  - `anthropic` - Anthropic Claude (if implemented)

**FR-3: Authentication Methods**
- KeyName (recommended): `--key-name my-openai-key` → retrieve from keystore
- API Key (deprecated): `--api-key sk-...` → direct key
- Environment variables: `OPENAI_API_KEY`, `ANTHROPIC_API_KEY`, etc.
- No auth for local providers (Ollama, LMStudio)

**FR-4: Configuration Structure**
```yaml
llm:
  provider: openai              # Provider type
  model: gpt-4o                 # Model name
  base_url: https://api.openai.com/v1  # API endpoint
  timeout: 30s                  # Request timeout

  # Auth (option 1: keystore - recommended)
  key_name: my-openai-key

  # Auth (option 2: direct - deprecated)
  # api_key: sk-...

  # Provider-specific options
  options:
    temperature: 0.7
    max_tokens: 4000
```

**FR-5: Provider Builder**
- Create `internal/llm/builder` package to centralize provider creation
- Builder combines: factory + auth + config + flags
- Single source of truth for provider instantiation

**FR-6: Error Handling**
- Clear error messages for:
  - Missing required config (provider, model)
  - Invalid provider type
  - Authentication failures (missing key, invalid key)
  - Connection failures (network, timeout)
- Suggest fixes in error messages

### Non-Functional Requirements

**NFR-1: Security**
- Never log API keys or credentials
- Prefer keystore over direct API keys
- Warn when using deprecated direct API key method
- Validate credentials before storing

**NFR-2: Backward Compatibility**
- Support legacy `--api-key` flag (with deprecation warning)
- Support environment variables for smooth migration
- Don't break existing mock provider usage in tests

**NFR-3: Performance**
- Provider creation <100ms
- Lazy initialization (only create when needed)
- Reuse providers when possible (singleton pattern for exec mode)

**NFR-4: Testability**
- Mock provider option for testing
- Dependency injection for factory/auth
- Unit tests for all configuration sources

---

## Design

### Architecture

```
cmd/spin/exec.go
    ↓ uses
internal/llm/builder.Builder
    ↓ combines
    ├── internal/config.Loader      (config file + env vars)
    ├── internal/llm/factory.Factory (provider creation)
    ├── internal/auth.Manager        (secure credentials)
    └── cobra flags                  (CLI overrides)

    ↓ creates
llm.Provider (openai/ollama/lmstudio/etc.)
    ↓ used by
internal/core.Manager
```

### Component: Provider Builder

**File:** `internal/llm/builder/builder.go`

```go
package builder

import (
    "context"
    "fmt"
    "time"

    "github.com/dmytrogajewski/spin/internal/auth"
    "github.com/dmytrogajewski/spin/internal/config"
    "github.com/dmytrogajewski/spin/internal/llm"
    "github.com/dmytrogajewski/spin/internal/llm/factory"
)

// Config holds provider configuration from all sources.
type Config struct {
    // Core settings
    Provider string
    Model    string
    BaseURL  string
    Timeout  time.Duration

    // Auth (mutually exclusive)
    KeyName string  // Recommended: from keystore
    APIKey  string  // Deprecated: direct key

    // Provider-specific options
    Options map[string]interface{}
}

// Builder builds LLM providers from multiple configuration sources.
type Builder struct {
    configLoader *config.Loader
    authMgr      *auth.Manager
    factory      *factory.Factory
}

// NewBuilder creates a new provider builder.
func NewBuilder(cfg *config.Loader, authMgr *auth.Manager) *Builder {
    return &Builder{
        configLoader: cfg,
        authMgr:      authMgr,
        factory:      factory.NewFactory(authMgr),
    }
}

// Build creates an LLM provider from merged configuration.
//
// Configuration precedence (highest to lowest):
// 1. Explicit config parameter
// 2. Environment variables
// 3. Config file
// 4. Defaults
func (b *Builder) Build(ctx context.Context, cfg Config) (llm.Provider, error) {
    // Merge with config file settings
    merged := b.mergeConfig(cfg)

    // Validate
    if err := b.validate(merged); err != nil {
        return nil, fmt.Errorf("invalid configuration: %w", err)
    }

    // Resolve auth from environment if needed
    if merged.KeyName == "" && merged.APIKey == "" {
        if key := b.resolveAPIKeyFromEnv(merged.Provider); key != "" {
            merged.APIKey = key
        }
    }

    // Create provider
    providerCfg := factory.ProviderConfig{
        Type:    merged.Provider,
        BaseURL: merged.BaseURL,
        Model:   merged.Model,
        Timeout: merged.Timeout,
        KeyName: merged.KeyName,
        APIKey:  merged.APIKey,
        Options: merged.Options,
    }

    return b.factory.NewProvider(ctx, providerCfg)
}

// mergeConfig merges explicit config with file-based config.
func (b *Builder) mergeConfig(explicit Config) Config {
    merged := explicit

    // Fill in missing values from config file
    if merged.Provider == "" {
        merged.Provider = b.configLoader.GetString("llm.provider")
    }
    if merged.Model == "" {
        merged.Model = b.configLoader.GetString("llm.model")
    }
    if merged.BaseURL == "" {
        merged.BaseURL = b.configLoader.GetString("llm.base_url")
    }
    if merged.Timeout == 0 {
        if t := b.configLoader.GetString("llm.timeout"); t != "" {
            if duration, err := time.ParseDuration(t); err == nil {
                merged.Timeout = duration
            }
        }
    }
    if merged.KeyName == "" {
        merged.KeyName = b.configLoader.GetString("llm.key_name")
    }

    // Apply defaults
    if merged.Provider == "" {
        merged.Provider = "ollama" // Default to local Ollama
    }
    if merged.BaseURL == "" {
        merged.BaseURL = b.defaultBaseURL(merged.Provider)
    }
    if merged.Timeout == 0 {
        merged.Timeout = 30 * time.Second
    }

    return merged
}

// validate validates the merged configuration.
func (b *Builder) validate(cfg Config) error {
    if cfg.Provider == "" {
        return fmt.Errorf("provider is required")
    }
    if cfg.Model == "" {
        return fmt.Errorf("model is required")
    }

    // Provider-specific validation
    switch cfg.Provider {
    case "openai", "anthropic", "openai-compatible":
        if cfg.KeyName == "" && cfg.APIKey == "" {
            return fmt.Errorf("authentication required for %s (use --key-name or set %s_API_KEY env var)",
                cfg.Provider, envKeyForProvider(cfg.Provider))
        }
    }

    return nil
}

// resolveAPIKeyFromEnv attempts to resolve API key from environment.
func (b *Builder) resolveAPIKeyFromEnv(provider string) string {
    envKey := envKeyForProvider(provider)
    return os.Getenv(envKey)
}

// envKeyForProvider returns the env var name for a provider.
func envKeyForProvider(provider string) string {
    switch provider {
    case "openai", "openai-compatible":
        return "OPENAI_API_KEY"
    case "anthropic":
        return "ANTHROPIC_API_KEY"
    default:
        return ""
    }
}

// defaultBaseURL returns the default base URL for a provider.
func (b *Builder) defaultBaseURL(provider string) string {
    switch provider {
    case "openai":
        return "https://api.openai.com/v1"
    case "anthropic":
        return "https://api.anthropic.com/v1"
    case "ollama":
        return "http://localhost:11434"
    case "lmstudio":
        return "http://localhost:1234/v1"
    default:
        return ""
    }
}
```

### Integration: cmd/spin/exec.go

**Changes to runExec:**

```go
func runExec(cmd *cobra.Command, args []string) error {
    ctx := context.Background()

    // Load config
    configLoader := config.NewLoader()
    if flagConfigFile != "" {
        if err := configLoader.LoadFromFile(flagConfigFile); err != nil {
            return fmt.Errorf("load config: %w", err)
        }
    } else {
        _ = configLoader.Load("") // Ignore error if no config found
    }

    // Create auth manager
    keystore := auth.NewKeystore() // Platform-specific keystore
    authMgr := auth.NewManager(keystore)

    // Build provider
    builder := builder.NewBuilder(configLoader, authMgr)
    provider, err := builder.Build(ctx, builder.Config{
        Provider: flagProvider,
        Model:    flagModel,
        // API key from flag (deprecated) - get from cmd flags if set
        // KeyName: flagKeyName, // To be added
    })
    if err != nil {
        return fmt.Errorf("create provider: %w", err)
    }
    defer provider.Close()

    // Rest of exec logic...
    execArgs, err := execpkg.Parse(args, os.Stdin)
    if err != nil {
        return err
    }

    // Pass provider to runner
    return execpkg.RunWithProvider(ctx, execArgs, provider)
}
```

### Integration: internal/exec/runner.go

**Update runTask to accept provider:**

```go
// RunWithProvider executes the task with a provided LLM provider.
func RunWithProvider(ctx context.Context, args *ExecArgs, provider llm.Provider) error {
    // Create core config
    coreConfig := core.DefaultConfig()

    if args.AutoApprove {
        coreConfig.AllowedCommands = []string{"*"}
        slog.Warn("auto-approve enabled - all commands will execute without validation")
    }

    // Create manager with real provider
    mgr, err := core.NewManager(coreConfig, core.WithLLM(provider))
    if err != nil {
        return fmt.Errorf("create manager: %w", err)
    }

    // Rest of execution...
}
```

---

## Implementation Plan

### Phase 1: Provider Builder (TDD)

**Step 1: Write Tests**
- `builder_test.go`: Test config merging, precedence, validation
- Test all provider types (openai, ollama, lmstudio)
- Test all auth methods (keyname, apikey, env vars)
- Test error cases (missing config, invalid provider)

**Step 2: Implement Builder**
- Create `internal/llm/builder/builder.go`
- Implement `Build()` with config merging
- Implement `mergeConfig()` with precedence rules
- Implement `validate()` for each provider type
- Implement env var resolution

**Step 3: Documentation**
- Godoc on all exports
- Usage examples
- Configuration guide

### Phase 2: CMD Integration

**Step 1: Update exec.go**
- Add builder initialization
- Wire up config loader + auth manager
- Pass provider to runner
- Update error handling

**Step 2: Update runner.go**
- Add `RunWithProvider()` function
- Replace mock provider with real provider
- Maintain backward compatibility for tests

**Step 3: Add New Flags (Optional)**
- `--key-name` for keystore credentials
- `--base-url` for custom endpoints
- `--timeout` for request timeout

### Phase 3: Testing

**Integration Tests:**
- Test with Ollama (local, no auth)
- Test with mock OpenAI server (with auth)
- Test config file + flags combination
- Test env var fallback

**Error Cases:**
- Missing provider
- Missing model
- Invalid provider type
- Auth failures
- Network failures

---

## Configuration Examples

### Example 1: Local Ollama
```bash
# Via flags
spin exec --provider ollama --model llama3.1 "fix tests"

# Via config file (~/.spin/spin.yaml)
llm:
  provider: ollama
  model: llama3.1
  base_url: http://localhost:11434

# Via env vars
export SPIN_PROVIDER=ollama
export SPIN_MODEL=llama3.1
spin exec "fix tests"
```

### Example 2: OpenAI with Keystore
```bash
# First, store API key in keystore
spin config set-key my-openai-key sk-...

# Use via flags
spin exec --provider openai --model gpt-4o --key-name my-openai-key "refactor code"

# Or via config
# ~/.spin/spin.yaml:
llm:
  provider: openai
  model: gpt-4o
  key_name: my-openai-key
```

### Example 3: OpenAI with Environment Variable
```bash
export OPENAI_API_KEY=sk-...
spin exec --provider openai --model gpt-4o "analyze code"
```

### Example 4: Custom OpenAI-Compatible API
```bash
spin exec \
  --provider openai-compatible \
  --model my-model \
  --base-url https://my-api.com/v1 \
  --api-key my-key \
  "run tests"
```

---

## Testing Strategy

### Unit Tests (≥90% coverage)

**builder_test.go:**
- `TestBuild_AllProviders` - Test each provider type
- `TestConfigPrecedence` - CLI > config > env > defaults
- `TestAuthMethods` - KeyName, APIKey, env vars
- `TestValidation` - Missing required fields, invalid values
- `TestMergeConfig` - Config merging logic
- `TestEnvVarResolution` - Environment variable fallback

### Integration Tests

**exec_integration_test.go:**
- Test with real Ollama (if available)
- Test with mock HTTP server (for OpenAI)
- Test config file loading
- Test error scenarios

### Manual Testing

- [ ] Local Ollama: `spin exec --provider ollama --model llama3.1 "test"`
- [ ] OpenAI with key: `spin exec --provider openai --model gpt-4o --api-key sk-... "test"`
- [ ] Config file: Create `~/.spin/spin.yaml` and test
- [ ] Env vars: `export OPENAI_API_KEY=... && spin exec ...`

---

## Quality Gates

**Code Quality:**
- [ ] All tests passing (≥90% coverage)
- [ ] Linter clean (`make lint`)
- [ ] Complexity ≤15 for all functions
- [ ] Godoc on all exports
- [ ] No security issues (`gosec`)

**Functionality:**
- [ ] All provider types work
- [ ] All auth methods work
- [ ] Config precedence correct
- [ ] Error messages clear and actionable

**Documentation:**
- [ ] Builder package godoc complete
- [ ] Configuration guide written
- [ ] Examples for each provider type
- [ ] Migration guide from mock to real providers

---

## Success Criteria

1. **Provider Creation Works:**
   - Can create OpenAI provider with API key
   - Can create Ollama provider (local, no auth)
   - Can create LMStudio provider
   - Can create custom OpenAI-compatible provider

2. **Configuration Sources Work:**
   - CLI flags override everything
   - Config file provides defaults
   - Env vars work as fallback
   - Precedence is correct

3. **Authentication Works:**
   - Keystore credentials (recommended)
   - Direct API key (deprecated, with warning)
   - Environment variables
   - No auth for local providers

4. **Error Handling Works:**
   - Clear messages for common errors
   - Suggests fixes (e.g., "set OPENAI_API_KEY")
   - No credential leaks in logs

5. **Tests Pass:**
   - ≥90% unit test coverage
   - Integration tests pass
   - Manual testing successful
   - No regressions in existing tests

---

## Migration Notes

**For Users:**
- No breaking changes to existing commands
- `--api-key` still works (deprecated, shows warning)
- New recommended approach: use `--key-name` with keystore

**For Developers:**
- Mock provider still available for tests
- Use dependency injection: pass provider to runner
- Update tests to use real providers where appropriate

---

## Related Files

**New:**
- `internal/llm/builder/builder.go` - Provider builder
- `internal/llm/builder/builder_test.go` - Builder tests
- `internal/llm/builder/doc.go` - Package documentation

**Modified:**
- `cmd/spin/exec.go` - Use builder instead of mock
- `internal/exec/runner.go` - Accept provider parameter
- `cmd/spin/root.go` - Add optional flags (--key-name, --base-url)

**Existing (dependencies):**
- `internal/llm/factory/factory.go` - Provider factory
- `internal/auth/auth.go` - Auth manager
- `internal/config/loader.go` - Config loader

---

## Future Enhancements

1. **Provider auto-detection**: Detect available local providers (Ollama, LMStudio)
2. **Multi-provider support**: Use different providers for different tasks
3. **Provider health checks**: Verify provider is available before use
4. **Credential wizard**: Interactive setup for API keys
5. **Provider profiles**: Named configurations (e.g., `spin exec --profile production`)

---

## Appendix: Environment Variables

| Env Var | Purpose | Example |
|---------|---------|---------|
| `SPIN_PROVIDER` | Default provider type | `ollama` |
| `SPIN_MODEL` | Default model | `llama3.1` |
| `SPIN_BASE_URL` | API endpoint | `http://localhost:11434` |
| `OPENAI_API_KEY` | OpenAI API key | `sk-...` |
| `ANTHROPIC_API_KEY` | Anthropic API key | `sk-ant-...` |
| `SPIN_CONFIG_FILE` | Config file path | `~/.spin/config.yaml` |

---

## References

- [internal/llm/factory](../../internal/llm/factory/factory.go) - Provider factory implementation
- [internal/auth](../../internal/auth/auth.go) - Auth manager
- [internal/config](../../internal/config/loader.go) - Config loader
- [Phase 0.4 Roadmap](../ui-modules/ROADMAP.md#04-provider-factory-integration-in-cmd--important)
