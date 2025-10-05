# FRD-UI-2.2: Output Formatting for spin-exec

**Status:** ✅ COMPLETE
**Priority:** P1 (Phase 2.2)
**Depends On:** FRD-UI-2.1 (Exec Command Structure)
**Related:** Phase 2.4 will integrate with core module

---

## Overview

Implement output formatting infrastructure for `spin-exec` to support both human-readable text and structured JSON output formats. This is infrastructure-only; core module integration happens in Phase 2.4.

---

## Requirements

### Functional Requirements

1. **Text Output Format (Human-Readable)**
   - Display streaming AI responses with `[Spin]` prefix
   - Show progress indicators on stderr (not stdout)
   - Display summary at end (tokens used, files modified, commands executed)
   - Support NO_COLOR environment variable
   - Clean output suitable for piping

2. **JSON Output Format (Structured)**
   - Valid JSON structure parseable by tools like `jq`
   - Include all relevant metadata (status, messages, files, commands, tokens)
   - Support streaming JSON (JSON Lines format)
   - Machine-readable for automation

3. **Streaming Support**
   - Output chunks as they arrive (don't buffer entire response)
   - Flush output immediately for real-time display
   - Progress on stderr, content on stdout
   - Handle context cancellation gracefully

4. **Exit Code Integration**
   - Set appropriate exit codes (0-5) based on task status
   - Map completion states to exit codes
   - Ensure consistent error reporting

5. **Summary Generation**
   - Tokens used (per-turn and total)
   - Files modified (list with paths)
   - Commands executed (list with exit codes)
   - Total duration
   - Final status (success/failure)

### Non-Functional Requirements

1. **Performance**
   - Minimal overhead (<1% CPU for formatting)
   - No buffering delays (immediate flush)
   - Memory efficient (stream, don't accumulate)

2. **Compatibility**
   - Works with pipes, redirects, tty detection
   - Respects NO_COLOR, TERM environment variables
   - Valid UTF-8 output
   - Cross-platform (Linux, macOS, Windows)

3. **Testability**
   - ≥90% test coverage
   - Isolated from core module (use interfaces)
   - Mockable for testing
   - Table-driven tests

---

## Architecture

### Package Structure

```
internal/exec/
├── output.go           # Output formatting implementation
├── output_test.go      # Tests
├── types.go            # Shared types (ExecResult, etc.)
└── format/
    ├── text.go         # Text formatter
    ├── text_test.go    # Text formatter tests
    ├── json.go         # JSON formatter
    └── json_test.go    # JSON formatter tests
```

### Core Types

```go
// OutputFormat represents the output format type
type OutputFormat string

const (
    FormatText OutputFormat = "text"
    FormatJSON OutputFormat = "json"
)

// Formatter is the interface for output formatters
type Formatter interface {
    // FormatStart formats the initial message
    FormatStart(prompt string) string

    // FormatDelta formats a streaming chunk
    FormatDelta(delta string) string

    // FormatComplete formats the completion message
    FormatComplete(result *ExecResult) string

    // FormatError formats an error message
    FormatError(err error) string
}

// ExecResult represents the result of an exec run
type ExecResult struct {
    Status         string        // "complete", "failed", "timeout", "cancelled"
    Messages       []Message     // Conversation history
    FilesModified  []string      // List of modified files
    CommandsRun    []CommandLog  // Commands executed
    TokensUsed     int           // Total tokens consumed
    Duration       time.Duration // Total execution time
    Error          error         // Error if any
}

// Message represents a single message in the conversation
type Message struct {
    Role      string    // "user", "assistant", "system"
    Content   string    // Message content
    Timestamp time.Time // When message was created
}

// CommandLog represents a command that was executed
type CommandLog struct {
    Command  string // Command that was run
    ExitCode int    // Exit code
    Output   string // Command output (truncated)
}
```

### Text Formatter

**Output Example:**
```
[Spin] Starting task: Run all tests
[Spin] Reading test files...
[Spin] Found 3 failing tests
[Spin] Applying fixes...
[Spin] ✓ Tests now passing
[Spin] Task complete

Summary:
  Status: complete
  Duration: 2.3s
  Tokens: 1,234
  Files modified: 3
    - internal/core/config.go
    - internal/core/config_test.go
    - internal/llm/provider.go
  Commands executed: 2
    - go test ./... (exit 1)
    - go test ./... (exit 0)
```

### JSON Formatter

**Output Example:**
```json
{
  "status": "complete",
  "duration_ms": 2300,
  "tokens_used": 1234,
  "files_modified": [
    "internal/core/config.go",
    "internal/core/config_test.go",
    "internal/llm/provider.go"
  ],
  "commands_executed": [
    {
      "command": "go test ./...",
      "exit_code": 1,
      "output": "--- FAIL: TestConfig..."
    },
    {
      "command": "go test ./...",
      "exit_code": 0,
      "output": "PASS"
    }
  ],
  "messages": [
    {
      "role": "user",
      "content": "Run all tests",
      "timestamp": "2025-01-15T10:00:00Z"
    },
    {
      "role": "assistant",
      "content": "I'll run the tests...",
      "timestamp": "2025-01-15T10:00:01Z"
    }
  ],
  "error": null
}
```

**Streaming JSON (JSON Lines):**
```json
{"type":"start","prompt":"Run all tests"}
{"type":"delta","content":"Reading test files..."}
{"type":"delta","content":"Found 3 failing tests"}
{"type":"complete","status":"complete","tokens_used":1234}
```

---

## Implementation Plan

### Step 1: Create Types (TDD)

1. Write test for `ExecResult` struct creation
2. Write test for `Message` struct
3. Write test for `CommandLog` struct
4. Implement types in `internal/exec/types.go`
5. Verify tests pass

### Step 2: Text Formatter (TDD)

1. Write tests for `TextFormatter`:
   - `TestTextFormatter_FormatStart`
   - `TestTextFormatter_FormatDelta`
   - `TestTextFormatter_FormatComplete`
   - `TestTextFormatter_FormatError`
   - `TestTextFormatter_Summary`
   - `TestTextFormatter_NoColor`
2. Implement `internal/exec/format/text.go`
3. Run tests, verify ≥90% coverage

### Step 3: JSON Formatter (TDD)

1. Write tests for `JSONFormatter`:
   - `TestJSONFormatter_FormatStart`
   - `TestJSONFormatter_FormatDelta`
   - `TestJSONFormatter_FormatComplete`
   - `TestJSONFormatter_FormatError`
   - `TestJSONFormatter_ValidJSON`
   - `TestJSONFormatter_Streaming`
2. Implement `internal/exec/format/json.go`
3. Run tests, verify ≥90% coverage

### Step 4: Formatter Factory (TDD)

1. Write tests for `NewFormatter`:
   - `TestNewFormatter_Text`
   - `TestNewFormatter_JSON`
   - `TestNewFormatter_Invalid`
2. Implement `internal/exec/output.go`
3. Run tests

### Step 5: Integration & Quality

1. Run full test suite: `go test ./internal/exec/... -cover`
2. Run linter: `make lint`
3. Analyze with uast/herr: `uast parse internal/exec/*.go | herr analyze`
4. Fix any issues
5. Verify complexity ≤15

---

## Testing Strategy

### Unit Tests

```go
func TestTextFormatter_FormatStart(t *testing.T) {
    tests := []struct {
        name   string
        prompt string
        want   string
    }{
        {
            name:   "basic prompt",
            prompt: "Run tests",
            want:   "[Spin] Starting task: Run tests\n",
        },
        {
            name:   "empty prompt",
            prompt: "",
            want:   "[Spin] Starting task\n",
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            f := NewTextFormatter()
            got := f.FormatStart(tt.prompt)
            if got != tt.want {
                t.Errorf("FormatStart() = %q, want %q", got, tt.want)
            }
        })
    }
}

func TestJSONFormatter_ValidJSON(t *testing.T) {
    f := NewJSONFormatter()
    result := &ExecResult{
        Status:      "complete",
        TokensUsed:  1234,
        FilesModified: []string{"file.go"},
        Duration:    2 * time.Second,
    }

    output := f.FormatComplete(result)

    // Verify it's valid JSON
    var decoded map[string]interface{}
    if err := json.Unmarshal([]byte(output), &decoded); err != nil {
        t.Fatalf("FormatComplete() produced invalid JSON: %v", err)
    }

    // Verify structure
    if decoded["status"] != "complete" {
        t.Errorf("status = %v, want complete", decoded["status"])
    }
    if decoded["tokens_used"] != float64(1234) {
        t.Errorf("tokens_used = %v, want 1234", decoded["tokens_used"])
    }
}
```

### Coverage Requirements

- Overall: ≥90%
- Text formatter: ≥95%
- JSON formatter: ≥95%
- Edge cases: empty inputs, nil pointers, special characters

---

## Exit Codes

Formatters should respect these exit codes (set by caller):

| Code | Status | Meaning |
|------|--------|---------|
| 0 | `complete` | Success |
| 1 | `failed` | General error |
| 2 | - | Authentication failed |
| 3 | `failed` | Task failed |
| 4 | `timeout` | Timeout exceeded |
| 5 | `cancelled` | User cancellation (SIGINT) |

---

## Examples

### Basic Usage

```go
// Create formatter
formatter := format.NewTextFormatter()

// Start
fmt.Print(formatter.FormatStart("Run all tests"))

// Stream deltas (placeholder - core integration in Phase 2.4)
for _, delta := range []string{"Reading files...", "Running tests..."} {
    fmt.Print(formatter.FormatDelta(delta))
}

// Complete
result := &ExecResult{
    Status:      "complete",
    TokensUsed:  1234,
    Duration:    2 * time.Second,
}
fmt.Print(formatter.FormatComplete(result))
```

### JSON Output

```go
formatter := format.NewJSONFormatter()
result := &ExecResult{
    Status:      "complete",
    TokensUsed:  1234,
    FilesModified: []string{"config.go"},
    Duration:    2 * time.Second,
}

json := formatter.FormatComplete(result)
fmt.Println(json)
```

### Piping

```bash
# Text to terminal
spin exec "Run tests"

# JSON to jq
spin exec --format json "Analyze code" | jq '.files_modified'

# Redirect text
spin exec "Build" > build.log 2>&1

# Check exit code
spin exec "Deploy" && echo "Success" || echo "Failed"
```

---

## Edge Cases

1. **Empty Result**
   - No messages, no files, no commands
   - Should produce valid (empty) output

2. **Special Characters**
   - Unicode in messages
   - Control characters in command output
   - Must escape properly for JSON

3. **Large Output**
   - Truncate command output if >1KB
   - Stream efficiently without buffering all

4. **Color Codes**
   - Strip ANSI codes when piped (not a tty)
   - Respect NO_COLOR environment variable

5. **Nil/Zero Values**
   - Handle nil error gracefully
   - Zero duration, zero tokens

---

## Definition of Ready (DoR)

- [x] Phase 2.1 (Exec Command Structure) complete
- [x] Output format spec reviewed
- [x] Streaming requirements understood
- [x] Types designed (ExecResult, Message, CommandLog)
- [x] Test strategy defined

---

## Definition of Done (DoD)

- [ ] All types implemented (`types.go`)
- [ ] Text formatter complete with tests (≥95% coverage)
- [ ] JSON formatter complete with tests (≥95% coverage)
- [ ] Factory function `NewFormatter()` implemented
- [ ] All tests passing
- [ ] Overall coverage ≥90%
- [ ] Linter clean (`make lint`)
- [ ] Code analysis clean (uast/herr)
- [ ] Complexity ≤15 for all functions
- [ ] Godoc comments on all exports
- [ ] NO_COLOR support verified
- [ ] JSON output validated with `jq`
- [ ] Examples work correctly
- [ ] ROADMAP updated

---

## Notes

- **Phase 2.2 is infrastructure only** - uses placeholder data
- **Core integration happens in Phase 2.4** - will connect to real LLM streaming
- Focus on clean interfaces that make testing easy
- Keep formatters stateless where possible
- Use io.Writer interfaces for flexibility

---

## References

- [UI Modules Spec](../ui-modules/spec.md) - Section on spin-exec output
- [ROADMAP](../ui-modules/ROADMAP.md) - Phase 2 timeline
- [FRD-UI-2.1](FRD-UI-2.1.md) - Exec Command Structure
- [Go Standard Library](https://pkg.go.dev/std) - encoding/json, io, fmt
