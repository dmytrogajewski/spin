# FRD-UI-4.2: Config Management Commands

**Feature:** Configuration management CLI commands
**Module:** `cmd/spin/config.go`
**Roadmap:** Phase 4.2
**Priority:** P2 - Advanced Features
**Status:** Draft

---

## 1. Overview

### 1.1 Purpose

Provide command-line tools for managing Spin configuration, allowing users to view, validate, edit, and locate configuration files without manual file manipulation.

### 1.2 Goals

- ✅ View current active configuration
- ✅ Validate configuration files
- ✅ Edit configuration in preferred editor
- ✅ Show configuration file location
- ✅ Support all config formats (YAML, JSON, TOML)
- ✅ User-friendly error messages

### 1.3 Non-Goals

- ❌ Runtime configuration changes (config is loaded at startup)
- ❌ Configuration migration (deferred to future)
- ❌ Configuration templates or wizards
- ❌ Remote configuration management

---

## 2. Requirements

### 2.1 Functional Requirements

#### FR-1: Show Current Configuration
**Priority:** P0 (Critical)

Users can view their active configuration:
```bash
spin config show

# Output (human-readable):
# Configuration: /home/user/.spin/spin.yaml
#
# llm:
#   provider: openai
#   model: gpt-4o
#   base_url: https://api.openai.com/v1
# sandbox:
#   mode: workspace-write
# appearance:
#   theme: auto

# JSON output for scripting:
spin config show --format json
```

**Behavior:**
- Shows all active settings from merged config (file + env vars + defaults)
- Displays source file location
- Redacts sensitive values (api_key, credentials)
- Supports --format text|json|yaml
- Shows "no config file" if using defaults only

#### FR-2: Validate Configuration
**Priority:** P0 (Critical)

Users can validate their configuration file:
```bash
spin config validate

# Success:
# ✓ Configuration is valid: /home/user/.spin/spin.yaml

# Failure:
# ✗ Configuration is invalid: /home/user/.spin/spin.yaml
#
# Errors:
#   - llm.provider: must be one of: openai, anthropic, ollama, lmstudio
#   - llm.timeout: invalid duration format "60x" (expected: 60s)
#   - sandbox.mode: unknown mode "invalid"
```

**Behavior:**
- Validates configuration file syntax (YAML/JSON/TOML parsing)
- Validates semantic correctness (field types, required fields, enum values)
- Shows detailed error messages with line numbers if possible
- Returns exit code 0 on success, 1 on failure
- Can validate a specific file with --file flag

#### FR-3: Show Configuration Path
**Priority:** P1 (High)

Users can find their active configuration file:
```bash
spin config path

# Output:
# /home/user/.spin/spin.yaml

# If no config file:
# No configuration file found. Using defaults.
# Search paths:
#   - ./spin.yaml
#   - /home/user/.spin/spin.yaml
#   - /etc/spin/spin.yaml
```

**Behavior:**
- Shows path to active config file
- Shows search paths if no config found
- Exit code 0 if config exists, 1 if not found
- Can show all search paths with --all flag

#### FR-4: Edit Configuration
**Priority:** P1 (High)

Users can edit configuration in their preferred editor:
```bash
spin config edit

# Opens config in $EDITOR (or fallback: vi, nano)
# Creates ~/.spin/spin.yaml if it doesn't exist
```

**Behavior:**
- Opens config file in $EDITOR environment variable
- Fallback to $VISUAL, then vi, then nano
- Creates default config file if none exists
- Validates config after editing (optional with --no-validate)
- Shows helpful message if editor not found

### 2.2 Non-Functional Requirements

#### NFR-1: Performance
- All commands respond within 100ms
- Config parsing/validation < 50ms

#### NFR-2: Usability
- Clear error messages with actionable suggestions
- Consistent output formatting across commands
- Support for both human and machine-readable output

#### NFR-3: Safety
- Never expose sensitive values (API keys, tokens) in show command
- Validate config before writing changes
- Create backup before editing (future enhancement)

---

## 3. Architecture

### 3.1 Component Structure

```
cmd/spin/config.go
├── newConfigCmd()          # Parent command
├── newConfigShowCmd()      # spin config show
├── newConfigValidateCmd()  # spin config validate
├── newConfigPathCmd()      # spin config path
└── newConfigEditCmd()      # spin config edit

Uses:
- internal/config/loader.go  # Existing config loader
- internal/config/validator.go  # New validator (to be created)
```

### 3.2 Data Flow

```
User
 │
 ├─> spin config show
 │    └─> config.Loader.Load()
 │         └─> config.Loader.AllSettings()
 │              └─> Format & Redact
 │                   └─> Print
 │
 ├─> spin config validate
 │    └─> config.Loader.Load()
 │         └─> config.Validate()
 │              └─> Print errors
 │
 ├─> spin config path
 │    └─> config.Loader.ConfigFileUsed()
 │         └─> Print path
 │
 └─> spin config edit
      └─> config.Loader.ConfigFileUsed() or CreateDefault()
           └─> exec.Command($EDITOR, path)
                └─> Wait & Validate
```

---

## 4. Implementation

### 4.1 Config Show Command

```go
func runConfigShow(cmd *cobra.Command, args []string) error {
    format, _ := cmd.Flags().GetString("format")

    // Load config
    loader := config.NewLoader()
    if err := loader.Load(flagConfigFile); err != nil {
        return fmt.Errorf("failed to load config: %w", err)
    }

    // Get all settings
    settings := loader.AllSettings()

    // Redact sensitive values
    redactSensitiveValues(settings)

    // Display
    switch format {
    case "json":
        return printJSON(settings)
    case "yaml":
        return printYAML(settings)
    default:
        return printText(loader.ConfigFileUsed(), settings)
    }
}
```

### 4.2 Config Validate Command

```go
func runConfigValidate(cmd *cobra.Command, args []string) error {
    file, _ := cmd.Flags().GetString("file")

    // Load config
    loader := config.NewLoader()
    if file != "" {
        err := loader.LoadFromFile(file)
    } else {
        err := loader.Load(flagConfigFile)
    }

    if err != nil {
        fmt.Fprintf(os.Stderr, "✗ Configuration is invalid\n\n")
        fmt.Fprintf(os.Stderr, "Error: %v\n", err)
        return ExitError{Code: 1}
    }

    // Semantic validation
    if err := validateConfig(loader); err != nil {
        fmt.Fprintf(os.Stderr, "✗ Configuration is invalid\n\n")
        fmt.Fprintf(os.Stderr, "%v\n", err)
        return ExitError{Code: 1}
    }

    fmt.Printf("✓ Configuration is valid: %s\n", loader.ConfigFileUsed())
    return nil
}
```

### 4.3 Config Path Command

```go
func runConfigPath(cmd *cobra.Command, args []string) error {
    showAll, _ := cmd.Flags().GetBool("all")

    loader := config.NewLoader()
    if err := loader.Load(flagConfigFile); err != nil {
        // File not found - show search paths
        if showAll {
            fmt.Println("No configuration file found. Search paths:")
            for _, path := range getConfigSearchPaths() {
                fmt.Printf("  - %s\n", path)
            }
        } else {
            fmt.Println("No configuration file found. Using defaults.")
        }
        return ExitError{Code: 1}
    }

    fmt.Println(loader.ConfigFileUsed())
    return nil
}
```

### 4.4 Config Edit Command

```go
func runConfigEdit(cmd *cobra.Command, args []string) error {
    noValidate, _ := cmd.Flags().GetBool("no-validate")

    // Load or create config
    loader := config.NewLoader()
    _ = loader.Load(flagConfigFile)

    configPath := loader.ConfigFileUsed()
    if configPath == "" {
        // Create default config
        configPath = filepath.Join(os.Getenv("HOME"), ".spin", "spin.yaml")
        if err := createDefaultConfig(configPath); err != nil {
            return err
        }
    }

    // Find editor
    editor := getEditor()
    if editor == "" {
        return fmt.Errorf("no editor found. Set $EDITOR or $VISUAL")
    }

    // Open editor
    cmd := exec.Command(editor, configPath)
    cmd.Stdin = os.Stdin
    cmd.Stdout = os.Stdout
    cmd.Stderr = os.Stderr

    if err := cmd.Run(); err != nil {
        return fmt.Errorf("editor failed: %w", err)
    }

    // Validate after editing
    if !noValidate {
        if err := validateConfigFile(configPath); err != nil {
            fmt.Fprintf(os.Stderr, "⚠ Warning: configuration has errors:\n%v\n", err)
        }
    }

    return nil
}

func getEditor() string {
    if editor := os.Getenv("EDITOR"); editor != "" {
        return editor
    }
    if visual := os.Getenv("VISUAL"); visual != "" {
        return visual
    }
    // Try common editors
    for _, editor := range []string{"vi", "vim", "nano", "emacs"} {
        if _, err := exec.LookPath(editor); err == nil {
            return editor
        }
    }
    return ""
}
```

---

## 5. Testing

### 5.1 Test Coverage Requirements

- **Target:** ≥90% code coverage
- **Critical paths:** 100% coverage (validation, redaction)

### 5.2 Test Cases

#### Unit Tests

1. **Config Show**
   - Show with text format
   - Show with JSON format
   - Show with YAML format
   - Redaction of sensitive values (api_key, credentials)
   - No config file (defaults only)

2. **Config Validate**
   - Valid YAML config
   - Valid JSON config
   - Invalid YAML syntax
   - Invalid field values
   - Missing required fields
   - Unknown fields (should warn, not error)

3. **Config Path**
   - Config file exists
   - No config file
   - Show all search paths
   - Custom config file via --config flag

4. **Config Edit**
   - Editor found in $EDITOR
   - Editor found in $VISUAL
   - Fallback to vi/nano
   - No editor available
   - Create default config
   - Validate after edit

#### Integration Tests

1. Full workflow: show → edit → validate → show
2. Multiple config formats (YAML, JSON, TOML)
3. Environment variable overrides
4. Config file precedence

---

## 6. Success Metrics

### 6.1 Definition of Done (DoD)

- [x] Tests for config commands (≥90% coverage)
- [x] Show displays current config
- [x] Validate catches errors
- [x] Edit opens correct editor
- [x] Path returns correct location
- [x] All tests passing with `-race`
- [x] Linter clean (`golangci-lint`)
- [x] Complexity ≤15 for all functions
- [x] Godoc on all exports
- [x] Manual testing complete

### 6.2 Acceptance Criteria

- Users can view their config without opening files
- Users can quickly validate config syntax
- Users can edit config without knowing the file path
- Error messages are clear and actionable
- Works on Linux, macOS, Windows

---

## 7. Open Questions

1. **Q:** Should `config edit` create a backup before editing?
   **A:** Deferred to future enhancement. Most editors have undo.

2. **Q:** Should we support interactive config creation (wizard)?
   **A:** No, out of scope. Use `config edit` with default template.

3. **Q:** Should `config show` include environment variable overrides?
   **A:** Yes, show final merged config. Add flag to show source (file/env/default).

4. **Q:** Should we support config migration from old formats?
   **A:** Deferred to Phase 4.2 (noted in DoR but not implemented).

---

## 8. References

- [FRD-UI-4.1: MCP Management Commands](FRD-UI-4.1.md)
- [internal/config/loader.go](../../internal/config/loader.go)
- [specs/ui-modules/ROADMAP.md](../ui-modules/ROADMAP.md)
- [AGENTS.md](../../AGENTS.md)
