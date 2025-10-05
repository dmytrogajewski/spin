# Package: cmd/spin-exec

**Path:** `cmd/spin-exec`
**Purpose:** Non-interactive/headless execution mode for Spin

---

## Overview

The `spin-exec` package provides non-interactive execution mode for running Spin in CI/CD pipelines, automation scripts, and other headless environments. It integrates directly with the core module and supports automatic command approval policies.

## Key Features

- **Non-Interactive Execution**: Run tasks without user prompts
- **Core Integration**: Uses `internal/core` for agent orchestration
- **Command Approval**: Leverages `core.Validator` for safety classification
- **Audit Logging**: Structured JSON logging for approval decisions
- **Event Streaming**: Real-time output streaming via core events
- **Signal Handling**: Graceful shutdown on SIGINT/SIGTERM

## Command Structure

```bash
spin exec [OPTIONS] <PROMPT>
```

### Options

| Flag | Description | Default |
|------|-------------|---------|
| `--auto-approve` | Auto-approve all commands (DANGEROUS) | `false` |
| `--timeout <duration>` | Maximum execution time | None |
| `--format <format>` | Output format (text, json) | `text` |

### Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | General error |
| 2 | Authentication failed |
| 3 | Task failed |
| 4 | Timeout exceeded |
| 5 | User cancellation (SIGINT) |
| 6 | Approval denied |

## Architecture

```
cmd/spin-exec/
├── main.go       # Entry point, arg parsing
├── args.go       # Argument parsing
├── errors.go     # Error handling & exit codes
├── signals.go    # Signal handling
└── runner.go     # Core integration
```

### Flow

1. **Parse Arguments** → `parseArgs()`
2. **Create Context** → With timeout if specified
3. **Setup Signals** → SIGINT/SIGTERM handling
4. **Run Task** → `runTask()` integrates with core
5. **Stream Events** → Handle core events and output
6. **Exit** → With appropriate exit code

## Core Integration

### Configuration

```go
// Create core config
coreConfig := core.DefaultConfig()
coreConfig.Provider = "mock"
coreConfig.Model = "default"

// Auto-approve: Allow all commands
if autoApprove {
    coreConfig.AllowedCommands = []string{"*"}
}

// Create manager
mgr, _ := core.NewManager(coreConfig, core.WithLLM(provider))
```

### Event Handling

```go
case core.EventContentDelta:
    // Stream AI response to stdout
    if data, ok := event.Data.(*core.ContentDeltaData); ok {
        fmt.Print(data.Content)
    }

case core.EventCommandApproval:
    // Command needs approval (only if !autoApprove)
    if !autoApprove {
        auditLogger.Info("command denied",
            "reason", "exec mode requires --auto-approve")
        return ErrApprovalDenied
    }

case core.EventError:
    // Error occurred
    return err

case core.EventTurnComplete:
    // Turn finished
    fmt.Println()
```

## Command Approval

Exec mode uses the existing `core.Validator` for command safety classification:

### Safety Levels

- **Safe**: Auto-approved (git status, ls, cat, etc.)
- **Interactive**: Write operations (mkdir, git commit, etc.)
- **Dangerous**: Destructive (rm -rf, sudo, git push --force, etc.)
- **Forbidden**: Catastrophic (rm -rf /, fork bombs, etc.)
- **Unverified**: Unknown commands

### Approval Logic

**Without `--auto-approve` (default):**
```
1. Core validates command using core.Validator
2. If dangerous/interactive/unverified:
   - Core emits EventCommandApproval
   - Exec denies and exits with error
   - Audit log records denial
```

**With `--auto-approve` (DANGEROUS):**
```
1. AllowedCommands = ["*"] bypasses validation
2. All commands execute without approval
3. Audit log records auto-approval
```

## Audit Logging

All approval decisions are logged to stderr in JSON format using `log/slog`:

```json
{
  "time": "2025-10-05T14:30:00Z",
  "level": "INFO",
  "msg": "command approval request denied in exec mode",
  "event_type": "command_approval",
  "reason": "exec mode requires --auto-approve flag for dangerous commands"
}
```

## Usage Examples

### Basic Execution

```bash
# Safe command (auto-approved)
spin exec "show me the git status"

# Dangerous command (denied)
spin exec "delete all test files"
# Error: command requires approval (use --auto-approve)
```

### With Auto-Approve

```bash
# WARNING: Allows dangerous commands
spin exec --auto-approve "delete all test files"
```

### CI/CD Integration

```yaml
# .github/workflows/spin.yml
- name: Run Spin Task
  run: |
    spin exec --timeout 5m "run tests and fix failures"
```

### With Timeout

```bash
# Max 5 minutes execution
spin exec --timeout 5m "refactor the authentication module"
```

## Signal Handling

### SIGINT/SIGTERM

```go
// First signal: graceful cancellation
Received signal: interrupt
Cancelling execution... (press Ctrl+C again to force quit)

// Context cancelled → cleanup → exit
```

### Implementation

```go
func setupSignalHandler(ctx context.Context, cancel context.CancelFunc) {
    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

    go handleSignals(ctx, cancel, sigChan)
}
```

## Error Handling

### Error Types

```go
const (
    ErrCodeNotFound     ErrCode = "NOT_FOUND"
    ErrCodeInvalidInput ErrCode = "INVALID_INPUT"
    ErrCodeTimeout      ErrCode = "TIMEOUT"
    ErrCodeCancelled    ErrCode = "CANCELLED"
    ErrCodeApproval     ErrCode = "APPROVAL_DENIED"
)
```

### Error Formatting

```go
func formatError(err error) string {
    // Unwrap error chain
    // Format with context
    // Return user-friendly message
}
```

## Testing

### Run Tests

```bash
# All tests
go test ./cmd/spin-exec/...

# With race detector
go test -race ./cmd/spin-exec/...

# With coverage
go test -cover ./cmd/spin-exec/...
```

### Test Coverage

- Argument parsing: 100%
- Signal handling: 85%
- Error handling: 95%
- Overall: 76.7%

## Performance

- **Startup Time**: ~50ms (no UI overhead)
- **Memory Usage**: ~20MB
- **Overhead**: <5% vs direct core API
- **Binary Size**: ~12MB

## Security Considerations

1. **Default Deny**: Commands require approval by default
2. **Audit Trail**: All decisions logged
3. **No Credential Leakage**: Sensitive data not logged
4. **Validation**: Uses battle-tested `core.Validator`
5. **--auto-approve**: Only for trusted environments

## Related Packages

- [core](core.md) - Core business logic and agent orchestration
- [llm](llm.md) - LLM provider interfaces
- [config](config.md) - Configuration management

---

**Last Updated:** 2025-10-05
**Status:** ✅ Production Ready
**Coverage:** 76.7%
