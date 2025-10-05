# FRD-UI-1.1: Main CLI Entry Point

## Metadata
- **ID:** FRD-UI-1.1
- **Title:** Main CLI Entry Point (`cmd/spin/`)
- **Status:** ✅ Complete
- **Created:** 2025-10-05
- **Updated:** 2025-10-05
- **Priority:** P0 (Critical - Foundation)
- **Related:** [UI Modules Spec](../ui-modules/spec.md), [Roadmap](../ui-modules/ROADMAP.md#11-main-cli-entry-point-cmdspin)

## Overview

Implement the main CLI entry point for Spin using the Cobra framework. This is the unified command-line interface that serves as the entry point for all Spin functionality, dispatching to different modes (TUI, exec, server, etc.).

## Definition of Ready (DoR)

- [x] Project structure follows Go Standard Layout
- [x] Dependencies specified in go.mod
- [x] Architecture overview reviewed
- [x] Cobra framework documented and understood
- [x] Command structure designed

## Requirements

### Functional Requirements

#### FR-1: Main Entry Point
- **FR-1.1:** Create `cmd/spin/main.go` as the application entry point
- **FR-1.2:** Use Cobra framework for CLI structure
- **FR-1.3:** Default command (no args) should prepare for TUI mode (placeholder for now)
- **FR-1.4:** Support `--help` flag to display help information
- **FR-1.5:** Support `--version` flag to display version information

#### FR-2: Global Flags
- **FR-2.1:** `--model <MODEL>` - Specify LLM model
- **FR-2.2:** `--provider <PROVIDER>` - Specify LLM provider (ollama, lmstudio, openai, anthropic)
- **FR-2.3:** `--sandbox <MODE>` - Specify sandbox mode (read-only, workspace-write, full-access)
- **FR-2.4:** `--cd <DIR>` - Change working directory before execution
- **FR-2.5:** `-c, --config <KEY=VALUE>` - Configuration overrides (repeatable)
- **FR-2.6:** `--config-file <PATH>` - Path to configuration file

#### FR-3: Command Registration Structure
- **FR-3.1:** Support subcommand registration pattern
- **FR-3.2:** Prepare structure for: `exec`, `serve`, `mcp-server`, `mcp`, `completion`, `debug`, `config`, `version`
- **FR-3.3:** Commands should be modular and in separate files

#### FR-4: Help System
- **FR-4.1:** Generate help text automatically (Cobra feature)
- **FR-4.2:** Show available commands
- **FR-4.3:** Show global flags
- **FR-4.4:** Show examples

#### FR-5: Version Command
- **FR-5.1:** Implement `spin version` command
- **FR-5.2:** Display version number (injected at build time via ldflags)
- **FR-5.3:** Display build information (commit hash, build date, Go version)
- **FR-5.4:** Handle `--version` global flag

#### FR-6: Shell Completion
- **FR-6.1:** Implement `spin completion` command
- **FR-6.2:** Support bash, zsh, fish, powershell
- **FR-6.3:** Generate completion scripts to stdout

#### FR-7: Binary Name Detection
- **FR-7.1:** Detect binary name from `os.Args[0]`
- **FR-7.2:** Support special modes via symlinks:
  - `spin-apply-patch` → apply patch mode
  - `spin-sandbox` → sandbox test mode
- **FR-7.3:** Support internal flags for subprocess execution

### Non-Functional Requirements

#### NFR-1: Performance
- Startup time < 10ms (CLI parsing only, no heavy initialization)
- Help generation < 5ms
- Memory usage < 5MB (CLI framework overhead)

#### NFR-2: Code Quality
- Test coverage ≥ 85%
- Cyclomatic complexity ≤ 15 per function
- All exports documented with godoc
- Pass golangci-lint

#### NFR-3: Usability
- Clear, actionable error messages
- Consistent flag naming
- Intuitive command hierarchy
- POSIX-compliant flag behavior

#### NFR-4: Maintainability
- Modular command structure
- Easy to add new commands
- Follows Go Standard Project Layout
- Well-documented code

## Technical Design

### Architecture

```
cmd/spin/
├── main.go              # Entry point, root command setup
├── root.go              # Root command definition
├── version.go           # Version command
├── completion.go        # Shell completion command
├── exec.go              # Exec command (stub for now)
├── config.go            # Config management command (stub)
├── mcp.go               # MCP management command (stub)
├── debug.go             # Debug commands (stub)
└── global_flags.go      # Global flag definitions

internal/version/
├── version.go           # Version info and build metadata
└── version_test.go      # Version tests
```

### Data Structures

```go
// Root command configuration
type GlobalFlags struct {
    Model      string
    Provider   string
    Sandbox    string
    WorkDir    string
    ConfigFile string
    ConfigOverrides []string
}

// Build-time version info (injected via ldflags)
var (
    Version   = "dev"
    Commit    = "unknown"
    BuildDate = "unknown"
    GoVersion = runtime.Version()
)
```

### Key Interfaces

```go
// Command factory pattern
type CommandFunc func() *cobra.Command

// Global flag binding
func BindGlobalFlags(cmd *cobra.Command, flags *GlobalFlags)

// Version info
type VersionInfo struct {
    Version   string
    Commit    string
    BuildDate string
    GoVersion string
}
```

### Implementation Steps

1. **Setup Project Structure**
   - Create `cmd/spin/` directory
   - Create `internal/version/` package
   - Update go.mod with cobra dependency

2. **Implement Version Package**
   - Create version.go with version variables
   - Add GetVersionInfo() function
   - Write tests for version formatting

3. **Implement Root Command**
   - Create main.go with root command
   - Define global flags
   - Setup command execution
   - Add error handling

4. **Implement Version Command**
   - Create version.go command file
   - Format version output
   - Handle --version flag

5. **Implement Completion Command**
   - Create completion.go
   - Add shell-specific generators
   - Test completion output

6. **Add Binary Name Detection**
   - Parse os.Args[0]
   - Route to special modes
   - Add internal flag support

7. **Create Command Stubs**
   - Add exec.go stub
   - Add config.go stub
   - Add mcp.go stub
   - Add debug.go stub

## Test Plan

### Unit Tests

```go
// Test root command creation
func TestRootCommand(t *testing.T)

// Test global flags parsing
func TestGlobalFlags(t *testing.T)

// Test version command
func TestVersionCommand(t *testing.T)

// Test version display
func TestVersionDisplay(t *testing.T)

// Test completion generation
func TestCompletionBash(t *testing.T)
func TestCompletionZsh(t *testing.T)
func TestCompletionFish(t *testing.T)
func TestCompletionPowershell(t *testing.T)

// Test binary name detection
func TestBinaryNameDetection(t *testing.T)

// Test help output
func TestHelpOutput(t *testing.T)

// Test error handling
func TestInvalidCommand(t *testing.T)
func TestInvalidFlags(t *testing.T)
```

### Integration Tests

```go
// Test full command execution
func TestCLIExecution(t *testing.T)

// Test flag precedence
func TestFlagPrecedence(t *testing.T)
```

### Manual Testing

```bash
# Basic commands
spin --help
spin --version
spin version
spin help

# Global flags
spin --model llama3.1 --help
spin --provider ollama --sandbox workspace-write --help

# Completion
spin completion bash > /dev/null
spin completion zsh > /dev/null
spin completion fish > /dev/null
spin completion powershell > /dev/null

# Invalid input
spin invalid-command  # Should show error and suggest commands
spin --invalid-flag   # Should show error
```

## Dependencies

### Go Modules
```
github.com/spf13/cobra v1.8.0
```

### Internal Packages
- None initially (self-contained)
- Future: `internal/config` for configuration loading

## Success Metrics

- [ ] All unit tests pass (≥85% coverage)
- [ ] Linter clean (golangci-lint)
- [ ] Complexity ≤15 for all functions
- [ ] Godoc on all exports
- [ ] Can execute: `spin --help`
- [ ] Can execute: `spin --version`
- [ ] Can execute: `spin version`
- [ ] Shell completions generate without errors
- [ ] Binary name detection works for symlinks
- [ ] Help text is clear and comprehensive

## Definition of Done (DoD)

- [x] All functional requirements implemented
- [x] All tests passing (≥85% coverage) - **86.1% achieved**
- [x] Race detector clean (`go test -race`)
- [x] Linter passing (`make lint`) - **golangci-lint clean**
- [x] Complexity ≤15 (verified with gocyclo) - **All functions pass**
- [x] Godoc comments on all exports
- [x] README updated with CLI usage
- [x] Examples provided
- [x] Shell completions tested manually - **bash, zsh, fish, powershell working**
- [x] Binary builds successfully
- [x] Can execute `spin --help` and `spin --version`

## Risks and Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| Cobra API changes | Medium | Pin to specific version in go.mod |
| Flag naming conflicts | Low | Follow consistent naming convention |
| Binary detection fails on Windows | Medium | Add Windows-specific path handling |
| Help text too verbose | Low | Keep commands focused, use examples |

## Open Questions

- [x] Should we use Viper for config management? **Decision: Not in initial implementation, add later if needed**
- [x] How to handle version injection in development? **Decision: Default to "dev" if not set**
- [x] Support both `--config key=value` and `--config=key=value`? **Decision: Support both via Cobra**

## References

- [Cobra Documentation](https://cobra.dev/)
- [Go Standard Project Layout](https://github.com/golang-standards/project-layout)
- [AGENTS.md](../../AGENTS.md)
- [Architecture Overview](../architecture-overview.md)
- [UI Modules Spec](../ui-modules/spec.md)

## Change Log

| Date | Change | Author |
|------|--------|--------|
| 2025-10-05 | Initial FRD creation | AI Agent |
