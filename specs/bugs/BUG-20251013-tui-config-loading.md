# BUG: TUI Command Doesn't Load Config File

**Date:** 2025-10-13  
**Severity:** High  
**Component:** `cmd/spin/tui.go`  
**Status:** OPEN

---

## Summary

The `tui` subcommand fails to load provider and model from config files. It only reads from CLI flags.

## Reproduction

```bash
# Create config file
cat > test.yaml <<EOF
provider: ollama
model: qwen3:1.7b
max_turns: 10
timeout: 5m
EOF

# Try to use it
./bin/spin --config-file test.yaml tui

# Result:
# Error: create manager: create manager: invalid config: provider is required
# model is required
# max_turns must be > 0, got 0
```

## Root Cause

In `cmd/spin/tui.go`:

```go:246:265:cmd/spin/tui.go
func createManagerForTUI(provider llm.Provider, maxTurns int, requireApproval bool) (*core.Manager, error) {
	// ...
	cfg := core.DefaultConfig()
	cfg.MaxTurns = maxTurns
	cfg.WorkDir = workDir
	cfg.Provider = flagProvider  // ← Only reads from flags
	cfg.Model = flagModel        // ← Only reads from flags
	// ...
}
```

The function creates a fresh `core.DefaultConfig()` instead of loading from the `configLoader` that was already loaded in `runTUI:59`.

## Expected Behavior

The TUI command should load configuration from:
1. Config file (if provided)
2. Environment variables (override)
3. CLI flags (final override)

## Impact

- Users cannot use config files for TUI mode
- PTY E2E testing cannot use config files
- Inconsistent with `exec` mode which DOES load config files

## Suggested Fix

```go
func createManagerForTUI(provider llm.Provider, maxTurns int, requireApproval bool, configLoader *config.Loader) (*core.Manager, error) {
	// Load base config from loader
	cfg := core.DefaultConfig()
	
	// Merge from config file
	var coreCfg core.Config
	if err := configLoader.Unmarshal(&coreCfg); err == nil {
		if coreCfg.Provider != "" {
			cfg.Provider = coreCfg.Provider
		}
		if coreCfg.Model != "" {
			cfg.Model = coreCfg.Model
		}
		if coreCfg.MaxTurns > 0 {
			cfg.MaxTurns = coreCfg.MaxTurns
		}
		// ... merge other fields
	}
	
	// Override with flags (if provided)
	if flagProvider != "" {
		cfg.Provider = flagProvider
	}
	if flagModel != "" {
		cfg.Model = flagModel
	}
	if maxTurns > 0 {
		cfg.MaxTurns = maxTurns
	}
	
	// ... rest of function
}
```

## Workaround

Use CLI flags instead of config file:

```bash
./bin/spin --provider ollama --model qwen3:1.7b tui --max-turns 10
```

## Related

- `exec` mode DOES load config files correctly (see `cmd/spin/exec.go`)
- Config loading works for LLM provider, but not for core.Manager config

---

**Next Steps:**
1. Refactor `createManagerForTUI` to accept `configLoader`
2. Merge config file → env vars → flags in priority order
3. Add E2E test for config file loading in TUI mode
4. Document config file behavior in TUI docs

