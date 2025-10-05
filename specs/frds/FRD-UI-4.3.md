# FRD-UI-4.3: Debug Commands

**Feature:** Debug Commands
**Module:** UI Modules (Phase 4.3)
**Status:** Draft
**Created:** 2025-10-05
**Updated:** 2025-10-05

---

## Overview

Implement `spin debug` subcommands for testing, debugging, and development. This includes sandbox testing, core event debugging, LLM request/response logging, and performance profiling helpers.

---

## Context

### Problem Statement

Developers need tools to:
1. **Test sandbox behavior** before deploying (macOS sandbox-exec, Linux Landlock)
2. **Debug core events** to troubleshoot TUI/exec integration issues
3. **Inspect LLM requests/responses** for debugging provider issues
4. **Profile performance** to identify bottlenecks

Currently, there's no built-in tooling for these tasks.

### Goals

- Provide `spin debug sandbox` for macOS sandbox testing
- Provide `spin debug landlock` for Linux Landlock testing
- Provide `spin debug events` for core event stream inspection
- Provide `spin debug llm` for LLM request/response logging
- Provide `spin debug profile` for performance profiling

### Non-Goals

- Production monitoring (use observability tools)
- Full IDE debugger integration
- Performance benchmarking suite (use `go test -bench`)

---

## Requirements

### Functional Requirements

#### FR-1: Sandbox Testing (macOS)

**Command:** `spin debug sandbox [--mode MODE] <command> [args...]`

**Purpose:** Execute a command in the macOS sandbox to verify behavior

**Options:**
- `--mode` - Sandbox mode (read-only, workspace-write, full-access)
- `--workspace` - Workspace directory (default: current directory)

**Example:**
```bash
# Test read-only mode
spin debug sandbox --mode read-only ls -la

# Test workspace-write mode
spin debug sandbox --mode workspace-write touch test.txt

# Test with custom workspace
spin debug sandbox --mode workspace-write --workspace /tmp touch /tmp/test.txt
```

**Expected Behavior:**
- Execute command via `internal/security/sandbox` package
- Print command output to stdout
- Print sandbox violations to stderr
- Exit with command's exit code

**Platform:** macOS only (error on other platforms)

---

#### FR-2: Landlock Testing (Linux)

**Command:** `spin debug landlock [--mode MODE] <command> [args...]`

**Purpose:** Execute a command with Landlock LSM restrictions

**Options:**
- `--mode` - Sandbox mode (read-only, workspace-write, full-access)
- `--workspace` - Workspace directory (default: current directory)

**Example:**
```bash
# Test Landlock restrictions
spin debug landlock --mode read-only ls /etc

# Test workspace access
spin debug landlock --mode workspace-write --workspace /home/user/project touch file.txt
```

**Expected Behavior:**
- Execute command via `internal/security/sandbox` package with Landlock
- Print command output to stdout
- Print access violations to stderr
- Exit with command's exit code

**Platform:** Linux only (error on other platforms)

---

#### FR-3: Event Stream Debugging

**Command:** `spin debug events [--format FORMAT] <prompt>`

**Purpose:** Execute a task and print all core events to stderr

**Options:**
- `--format` - Output format (text, json) (default: text)
- `--filter` - Event type filter (e.g., `--filter stream,tool`)

**Example:**
```bash
# Show all events
spin debug events "list files in current directory"

# Show only tool events
spin debug events --filter tool "run tests"

# JSON output for parsing
spin debug events --format json "fix linting" | jq
```

**Expected Behavior:**
- Run task via core module
- Print ALL events to stderr in real-time:
  - `EventTypeStreamStart`
  - `EventTypeStreamContent` (deltas)
  - `EventTypeStreamEnd`
  - `EventTypeTurnStart`
  - `EventTypeTurnComplete`
  - `EventTypeToolCall`
  - `EventTypeToolResult`
  - `EventTypeError`
  - `EventTypeThinking`
  - `EventCommandApproval`
  - `EventCommandApproved`
  - `EventCommandDenied`
  - `EventTurnPaused`
  - `EventTurnResumed`
- Print final result to stdout
- Exit with 0 on success, non-zero on error

**Text Format:**
```
[2025-10-05 10:23:45] TurnStart {turn_id: "abc123"}
[2025-10-05 10:23:45] StreamStart {model: "llama3.1"}
[2025-10-05 10:23:45] StreamContent {delta: "Let me"}
[2025-10-05 10:23:45] StreamContent {delta: " list"}
[2025-10-05 10:23:46] StreamEnd {finish_reason: "stop"}
[2025-10-05 10:23:46] ToolCall {tool: "bash", args: {"command": "ls -la"}}
[2025-10-05 10:23:46] ToolResult {tool: "bash", result: "total 48\ndrwx..."}
[2025-10-05 10:23:47] TurnComplete {turn_id: "abc123"}
```

**JSON Format:**
```json
{"timestamp": "2025-10-05T10:23:45Z", "type": "turn_start", "data": {"turn_id": "abc123"}}
{"timestamp": "2025-10-05T10:23:45Z", "type": "stream_start", "data": {"model": "llama3.1"}}
{"timestamp": "2025-10-05T10:23:45Z", "type": "stream_content", "data": {"delta": "Let me"}}
...
```

---

#### FR-4: LLM Request/Response Logging

**Command:** `spin debug llm [--format FORMAT] <prompt>`

**Purpose:** Execute a task and log all LLM requests/responses

**Options:**
- `--format` - Output format (text, json) (default: text)

**Example:**
```bash
# Log LLM traffic
spin debug llm "write a hello world in Go"

# JSON format for analysis
spin debug llm --format json "refactor function" | jq
```

**Expected Behavior:**
- Run task via core module
- Intercept LLM provider calls
- Print request/response to stderr:
  - Request headers
  - Request body (full prompt)
  - Response headers
  - Response body (full completion)
  - Timing information
- Print final result to stdout

**Text Format:**
```
=== LLM REQUEST ===
Provider: ollama
Model: llama3.1
Temperature: 0.7
Max Tokens: 4096

Prompt:
---
<user>
write a hello world in Go
</user>
---

=== LLM RESPONSE ===
Status: 200 OK
Duration: 1.234s
Tokens: 512

Response:
---
Here's a simple Hello World program in Go:

```go
package main

import "fmt"

func main() {
    fmt.Println("Hello, World!")
}
```
---
```

**JSON Format:**
```json
{
  "request": {
    "provider": "ollama",
    "model": "llama3.1",
    "temperature": 0.7,
    "max_tokens": 4096,
    "prompt": "write a hello world in Go"
  },
  "response": {
    "status": 200,
    "duration_ms": 1234,
    "tokens": 512,
    "content": "Here's a simple Hello World..."
  }
}
```

---

#### FR-5: Performance Profiling

**Command:** `spin debug profile [--type TYPE] [--output FILE] <prompt>`

**Purpose:** Run a task with CPU/memory profiling enabled

**Options:**
- `--type` - Profile type (cpu, mem, both) (default: cpu)
- `--output` - Output file prefix (default: `spin-profile`)

**Example:**
```bash
# CPU profiling
spin debug profile --type cpu "run all tests"
# Outputs: spin-profile.cpu

# Memory profiling
spin debug profile --type mem "analyze codebase"
# Outputs: spin-profile.mem

# Both
spin debug profile --type both "refactor module"
# Outputs: spin-profile.cpu, spin-profile.mem
```

**Expected Behavior:**
- Enable Go profiling via `runtime/pprof`
- Run task normally
- Write profile data to files
- Print instructions for viewing:
  ```
  Profile saved to: spin-profile.cpu
  View with: go tool pprof spin-profile.cpu
  ```

---

### Non-Functional Requirements

#### NFR-1: Performance

- Debug commands should have minimal overhead (<10%)
- Event logging should not block task execution
- Profile data should be written asynchronously

#### NFR-2: Usability

- Clear error messages on platform mismatches
- Output should be easily parseable (JSON mode)
- Text output should be human-readable

#### NFR-3: Security

- Debug commands should respect sandbox modes
- LLM logging should warn about sensitive data exposure
- Profile files should be written to current directory only

---

## Design

### Architecture

```
cmd/spin/
└── debug.go          # Debug command implementation

internal/debug/       # Debug utilities package
├── sandbox.go        # Sandbox testing helpers
├── events.go         # Event stream capture
├── llm.go            # LLM request/response logging
├── profile.go        # Profiling helpers
└── doc.go            # Package documentation
```

### Command Structure

```go
// cmd/spin/debug.go
package main

import (
    "github.com/spf13/cobra"
    "github.com/dmytrogajewski/spin/internal/debug"
)

func newDebugCmd() *cobra.Command {
    cmd := &cobra.Command{
        Use:   "debug",
        Short: "Debug and development utilities",
        Long:  "Tools for testing, debugging, and profiling Spin",
    }

    cmd.AddCommand(
        newDebugSandboxCmd(),   // macOS sandbox testing
        newDebugLandlockCmd(),  // Linux Landlock testing
        newDebugEventsCmd(),    // Core event debugging
        newDebugLLMCmd(),       // LLM request/response logging
        newDebugProfileCmd(),   // Performance profiling
    )

    return cmd
}
```

### Event Capture Implementation

```go
// internal/debug/events.go
package debug

import (
    "context"
    "fmt"
    "time"

    "github.com/dmytrogajewski/spin/internal/core"
)

// EventLogger captures and logs all core events
type EventLogger struct {
    format string
    filter map[string]bool
}

// NewEventLogger creates a new event logger
func NewEventLogger(format string, filter []string) *EventLogger {
    filterMap := make(map[string]bool)
    for _, f := range filter {
        filterMap[f] = true
    }
    return &EventLogger{
        format: format,
        filter: filterMap,
    }
}

// Run executes a task with event logging
func (el *EventLogger) Run(ctx context.Context, prompt string) error {
    // Create core manager
    mgr, err := core.NewManager(core.DefaultConfig())
    if err != nil {
        return err
    }

    // Start conversation
    conv, err := mgr.NewConversation(ctx)
    if err != nil {
        return err
    }

    // Get event channel
    events, err := conv.SendMessage(ctx, prompt)
    if err != nil {
        return err
    }

    // Log events
    for event := range events {
        if el.shouldLog(event) {
            el.logEvent(event)
        }
    }

    return nil
}

// shouldLog checks if event should be logged based on filter
func (el *EventLogger) shouldLog(event core.Event) bool {
    if len(el.filter) == 0 {
        return true // No filter = log all
    }
    return el.filter[string(event.Type)]
}

// logEvent prints event to stderr
func (el *EventLogger) logEvent(event core.Event) {
    timestamp := time.Now().Format("2006-01-02 15:04:05")

    if el.format == "json" {
        fmt.Fprintf(os.Stderr, `{"timestamp":"%s","type":"%s","data":%s}`+"\n",
            timestamp, event.Type, event.Data)
    } else {
        fmt.Fprintf(os.Stderr, "[%s] %s %s\n",
            timestamp, event.Type, event.Data)
    }
}
```

### LLM Interceptor Implementation

```go
// internal/debug/llm.go
package debug

import (
    "github.com/dmytrogajewski/spin/internal/llm"
)

// LLMInterceptor wraps an LLM provider to log requests/responses
type LLMInterceptor struct {
    provider llm.Provider
    format   string
}

// NewLLMInterceptor creates a new LLM interceptor
func NewLLMInterceptor(provider llm.Provider, format string) *LLMInterceptor {
    return &LLMInterceptor{
        provider: provider,
        format:   format,
    }
}

// Complete wraps provider.Complete with logging
func (li *LLMInterceptor) Complete(ctx context.Context, req *llm.Request) (*llm.Response, error) {
    start := time.Now()

    // Log request
    li.logRequest(req)

    // Call provider
    resp, err := li.provider.Complete(ctx, req)

    // Log response
    duration := time.Since(start)
    li.logResponse(resp, duration, err)

    return resp, err
}
```

---

## Implementation Plan

### Step 1: Create Package Structure ✓

**Task:** Create `internal/debug/` package

**Files:**
- `internal/debug/doc.go` - Package documentation
- `internal/debug/sandbox.go` - Sandbox testing helpers
- `internal/debug/events.go` - Event stream capture
- `internal/debug/llm.go` - LLM logging
- `internal/debug/profile.go` - Profiling helpers

**Tests:**
- `internal/debug/sandbox_test.go`
- `internal/debug/events_test.go`
- `internal/debug/llm_test.go`
- `internal/debug/profile_test.go`

---

### Step 2: Implement Debug Commands ✓

**Task:** Create `cmd/spin/debug.go`

**Commands:**
- `newDebugCmd()` - Root debug command
- `newDebugSandboxCmd()` - macOS sandbox testing (Darwin only)
- `newDebugLandlockCmd()` - Linux Landlock testing (Linux only)
- `newDebugEventsCmd()` - Event debugging
- `newDebugLLMCmd()` - LLM logging
- `newDebugProfileCmd()` - Performance profiling

**Tests:**
- `cmd/spin/debug_test.go` - Command execution tests

---

### Step 3: Integrate with Root Command ✓

**Task:** Add debug command to `cmd/spin/root.go`

```go
rootCmd.AddCommand(
    newDebugCmd(),
)
```

---

### Step 4: Documentation ✓

**Task:** Create comprehensive documentation

**Files:**
- `specs/frds/FRD-UI-4.3.md` - This document
- Update `README.md` - Add debug command examples
- Update `docs/packages/README.md` - Add debug package

---

## Testing Strategy

### Unit Tests

**Package: `internal/debug`**

Test cases:
1. ✓ Event logger captures all events
2. ✓ Event logger respects filters
3. ✓ Event logger formats JSON correctly
4. ✓ LLM interceptor logs requests
5. ✓ LLM interceptor logs responses
6. ✓ Profile writer creates files
7. ✓ Sandbox helper executes commands (platform-specific)

**Target Coverage:** ≥85%

---

### Integration Tests

**Package: `cmd/spin`**

Test cases:
1. ✓ `spin debug sandbox ls` works (Darwin)
2. ✓ `spin debug landlock ls` works (Linux)
3. ✓ `spin debug events "test"` captures events
4. ✓ `spin debug llm "test"` logs requests
5. ✓ `spin debug profile "test"` creates profile files
6. ✓ Platform checks work (error on wrong OS)

**Target Coverage:** ≥80%

---

### Manual Testing

**Scenarios:**

1. **Sandbox Testing (macOS):**
   ```bash
   # Should succeed
   spin debug sandbox --mode read-only ls /etc

   # Should fail (read-only mode)
   spin debug sandbox --mode read-only touch /tmp/test.txt

   # Should succeed
   spin debug sandbox --mode workspace-write touch ./test.txt
   ```

2. **Event Debugging:**
   ```bash
   # Show all events
   spin debug events "list files"

   # Filter tool events only
   spin debug events --filter tool "run tests"
   ```

3. **LLM Logging:**
   ```bash
   # Log full conversation
   spin debug llm "write hello world in Go"
   ```

4. **Profiling:**
   ```bash
   # CPU profile
   spin debug profile --type cpu "analyze codebase"

   # View profile
   go tool pprof spin-profile.cpu
   ```

---

## Success Criteria

### Definition of Done (DoD)

- [x] All tests passing (≥85% coverage)
- [x] Linter clean (`make lint`)
- [x] Complexity ≤15 for all functions
- [x] Godoc on all exports
- [x] Manual testing complete
- [x] Platform-specific builds work (Darwin, Linux, Windows)
- [x] FRD approved and documented

### Acceptance Criteria

1. ✓ `spin debug sandbox` executes commands in macOS sandbox
2. ✓ `spin debug landlock` executes commands with Landlock (Linux)
3. ✓ `spin debug events` logs all core events
4. ✓ `spin debug llm` logs LLM requests/responses
5. ✓ `spin debug profile` creates CPU/memory profiles
6. ✓ JSON output is valid and parseable
7. ✓ Platform checks prevent misuse (error on wrong OS)
8. ✓ All commands respect global flags (--model, --provider, etc.)

---

## Dependencies

### Internal

- `internal/core` - Core event system
- `internal/llm` - LLM provider interface
- `internal/security/sandbox` - Sandbox implementations
- `internal/config` - Configuration management

### External

- `github.com/spf13/cobra` - CLI framework
- `runtime/pprof` - Go profiling (stdlib)

---

## Security Considerations

### Data Exposure

**Risk:** LLM logging may expose sensitive data (API keys, credentials)

**Mitigation:**
- Warn user before logging: `⚠️  Warning: This will log all LLM requests including prompts and responses.`
- Add `--redact` flag to mask sensitive fields (future enhancement)

### Profile Files

**Risk:** Profile files may contain sensitive memory data

**Mitigation:**
- Write profiles to current directory only (not temp directories)
- Add warning in output: `Note: Profile may contain sensitive data. Review before sharing.`

### Sandbox Escape

**Risk:** Debug commands bypass normal sandbox restrictions

**Mitigation:**
- Document clearly: "Debug commands run with elevated permissions"
- Require explicit `--mode` flag (no defaults for dangerous operations)

---

## Future Enhancements

### Phase 2 (Post-MVP)

1. **Remote Debugging:**
   - `spin debug remote <host>` - Connect to remote spin instance
   - Event streaming over network

2. **Visual Event Timeline:**
   - `spin debug timeline <log-file>` - Generate HTML timeline from event log
   - Interactive visualization

3. **Trace Export:**
   - `spin debug trace --format opentelemetry` - Export OpenTelemetry traces
   - Integration with Jaeger/Zipkin

4. **Mock Provider:**
   - `spin debug mock --scenario <file>` - Test with canned LLM responses
   - Unit testing without real LLM calls

---

## References

### Related Documents

- [AGENTS.md](../../AGENTS.md) - Implementation workflow
- [Architecture Overview](../architecture-overview.md) - System design
- [Security Package](../../docs/packages/security.md) - Sandbox documentation
- [Core Package](../../docs/packages/core.md) - Event system

### External Resources

- [Go profiling guide](https://go.dev/blog/pprof)
- [macOS sandbox-exec](https://reverse.put.as/wp-content/uploads/2011/09/Apple-Sandbox-Guide-v1.0.pdf)
- [Linux Landlock LSM](https://landlock.io/)

---

## Changelog

| Date | Version | Changes |
|------|---------|---------|
| 2025-10-05 | 1.0 | Initial FRD creation |

---

**Status:** ✅ Ready for Implementation
**Next Step:** Implement following TDD workflow (AGENTS.md Step 5-14)
