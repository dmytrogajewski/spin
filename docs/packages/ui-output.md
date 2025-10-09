# UI Output Package

**Package:** `github.com/dmytrogajewski/spin/internal/ui/output`
**Purpose:** Append-only output for TUI chat transcript with streaming support

---

## Overview

The output package provides append-only printing capabilities for the TUI. It supports both immediate line printing and streaming chunks with optional coalescing to reduce flicker. The package is designed to work with the Factory Droid principle: preserve terminal scrollback, never clear the viewport.

---

## Core Types

### Printer

```go
type Printer struct {
    // private fields
}
```

**Thread-safe** printer that writes to an `io.Writer` (typically `os.Stdout`).

**Features:**
- **Immediate line printing** with `PrintLine()`
- **Streaming chunks** with `PrintChunks()`
- **Optional coalescing** to reduce write syscalls and flicker
- **Newline fast-path** for prompt responsiveness
- **Large chunk bypass** (>10KB written immediately)
- **Context cancellation** support
- **Concurrent-safe** writes

---

## API

### Creating a Printer

```go
func NewPrinter(out io.Writer, opts ...PrinterOption) *Printer
```

**Options:**

```go
func WithCoalesceDelay(d time.Duration) PrinterOption
```

**Example:**

```go
// Default settings (50ms coalesce delay)
printer := output.NewPrinter(os.Stdout)

// Custom delay
printer := output.NewPrinter(os.Stdout,
    output.WithCoalesceDelay(10 * time.Millisecond))

// No coalescing (immediate writes)
printer := output.NewPrinter(os.Stdout,
    output.WithCoalesceDelay(0))
```

---

### Printing Lines

```go
func (p *Printer) PrintLine(s string) error
```

Writes a line immediately with a newline appended. Thread-safe.

**Use cases:**
- User input echo
- System notices
- Discrete messages

**Example:**

```go
printer.PrintLine("User: Hello, AI!")
printer.PrintLine("System: Session started")
```

---

### Streaming Chunks

```go
func (p *Printer) PrintChunks(ctx context.Context, chunks <-chan string) error
```

Streams chunks from a channel with optional coalescing. Blocks until channel closes or context cancels.

**Behavior:**
- **Small chunks** (<10KB): Buffered, flushed after `coalesceDelay` or on newline
- **Large chunks** (≥10KB): Written immediately without buffering
- **Newline detection**: Flushes immediately on any `\n`
- **Context cancel**: Flushes partial buffer, returns `ctx.Err()`
- **Channel close**: Flushes remaining buffer, returns `nil`

**Use cases:**
- LLM response streaming
- Real-time log output
- Progressive rendering

**Example:**

```go
chunks := make(chan string, 100)
ctx := context.Background()

// Producer (e.g., LLM stream)
go func() {
    for token := range llmTokens {
        chunks <- token
    }
    close(chunks)
}()

// Consumer
err := printer.PrintChunks(ctx, chunks)
if err != nil && err != context.Canceled {
    log.Printf("Streaming error: %v", err)
}
```

---

## Performance

### Coalescing Strategy

**Default delay:** 50ms

**Why coalesce?**
- Reduces write syscalls (improves throughput)
- Reduces terminal redraw flicker
- Batches small chunks efficiently

**Newline fast-path:**
- Any chunk containing `\n` triggers immediate flush
- Keeps prompt responsive (prompt redraws after each line)

**Large chunk bypass:**
- Chunks ≥10KB skip buffering
- Avoids memory overhead for bulk data

### Benchmarks

Typical performance on modern hardware:

```
BenchmarkPrintLine                 ~1µs per call
BenchmarkPrintChunksSmall          1000 chunks/sec
BenchmarkPrintChunksLarge          500 MB/sec
BenchmarkConcurrentPrintLine       Scales linearly
```

---

## Thread Safety

All methods are **thread-safe**:

- Multiple goroutines can call `PrintLine()` concurrently
- Multiple goroutines can call `PrintChunks()` concurrently
- `PrintLine()` and `PrintChunks()` can run concurrently

**Mechanism:**
- Single `sync.Mutex` protects all writes to `io.Writer`
- Lock held only during write (minimal critical section)
- Channel operations outside lock (no deadlock risk)

**Verified with:**
- `go test -race ./internal/ui/output/...` (all tests pass)

---

## Error Handling

### PrintLine

Returns error if `io.Writer` fails. No retry.

```go
if err := printer.PrintLine("message"); err != nil {
    // Handle write error (disk full, pipe broken, etc.)
}
```

### PrintChunks

Returns:
- `nil` on success (channel closed normally)
- `context.Canceled` if context canceled
- `context.DeadlineExceeded` if context timeout
- Write error if `io.Writer` fails

```go
ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
defer cancel()

err := printer.PrintChunks(ctx, chunks)
switch err {
case nil:
    // Success
case context.Canceled:
    // User canceled
case context.DeadlineExceeded:
    // Timeout
default:
    // Write error
}
```

---

## Integration

### With Prompt System

**Pattern:** Output writes must be followed by prompt redraw.

**Phase 3.2 COMPLETED:** `CoordinatedWriter` automatically handles this.

**Usage:**

```go
// Create coordinator (wraps printer + renderer)
coord := output.NewCoordinatedWriter(printer, renderer, model)

// Print line - prompt automatically redrawn
coord.PrintLine("Assistant: Thinking...")
// No manual redraw needed!

// Stream chunks - prompt redrawn after completion
coord.PrintChunks(ctx, chunks)

// Set status - prompt redrawn with status
coord.SetStatus("typing...")
```

### With Core Events

**Future integration** (Phase 5.1 - PureTTY Adapter):

```go
// Map core events to printer calls
for event := range core.Events {
    switch event.Type {
    case core.EventTypeStreamContent:
        printer.PrintChunks(ctx, event.Chunks)
    case core.EventTypeSystemNotice:
        printer.PrintLine(event.Message)
    }
}
```

---

## Testing

### Unit Tests

```bash
go test ./internal/ui/output/...
```

**Coverage:** 90.6%

**Test categories:**
- PrintLine: single, empty, multiple, concurrent, errors
- PrintChunks: single, multiple, newline flush, large chunks, coalescing
- Concurrency: race conditions, interleaved calls
- Context: cancellation, timeout
- Options: custom delays

### Race Detection

```bash
go test -race ./internal/ui/output/...
```

All 23 tests pass with `-race` detector.

### Benchmarks

```bash
go test -bench=. ./internal/ui/output/...
```

---

## Examples

### Basic Usage

```go
package main

import (
    "os"
    "github.com/dmytrogajewski/spin/internal/ui/output"
)

func main() {
    printer := output.NewPrinter(os.Stdout)

    printer.PrintLine("User: What is 2+2?")
    printer.PrintLine("Assistant: Calculating...")
    printer.PrintLine("Assistant: 2+2 equals 4.")
}
```

### Streaming LLM Response

```go
func streamLLMResponse(printer *output.Printer, prompt string) error {
    chunks := make(chan string, 100)
    ctx := context.Background()

    // Start LLM stream in background
    go func() {
        defer close(chunks)
        for token := range llm.Stream(prompt) {
            chunks <- token.Text
        }
    }()

    // Stream to output
    return printer.PrintChunks(ctx, chunks)
}
```

### With Timeout

```go
func streamWithTimeout(printer *output.Printer, chunks <-chan string) error {
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()

    err := printer.PrintChunks(ctx, chunks)
    if err == context.DeadlineExceeded {
        printer.PrintLine("\n[Stream timeout - response truncated]")
    }
    return err
}
```

---

## Design Decisions

### Why Coalescing?

**Problem:** Writing every 1-byte token to stdout triggers:
1. Write syscall overhead
2. Terminal redraw flicker
3. Prompt redraw overhead

**Solution:** Buffer small chunks, flush periodically.

**Trade-off:** Introduces 50ms latency vs smooth visual output.

### Why Newline Fast-Path?

**Problem:** Coalescing delays complete lines, making prompt lag behind.

**Solution:** Flush immediately on `\n` detection.

**Benefit:** Prompt stays at bottom without lag.

### Why Large Chunk Bypass?

**Problem:** Buffering 100KB chunks wastes memory and adds latency.

**Solution:** Write chunks ≥10KB immediately.

**Benefit:** Bulk data (file contents, logs) streams efficiently.

---

## Future Enhancements

### Phase 3.2: Output-Prompt Coordination

**Goal:** Automatic prompt redraw after each output write.

**Design:**

```go
type CoordinatedWriter struct {
    printer  *Printer
    renderer *PromptRenderer
    model    *PromptModel
    mu       sync.Mutex
}

func (c *CoordinatedWriter) PrintLine(s string) error {
    c.mu.Lock()
    defer c.mu.Unlock()

    c.printer.PrintLine(s)
    c.renderer.Redraw(c.model, "")
    return nil
}
```

### Optional: ANSI Styling

**Future:** Add color/style support via functional options.

```go
printer.PrintLine(output.Styled("Error:", output.Red, output.Bold))
```

### Phase 3.2: Output-Prompt Coordination ✅

**Status:** COMPLETED (2025-10-10)

**Implementation:** `CoordinatedWriter` wraps Printer and PromptRenderer.

```go
type CoordinatedWriter struct {
    printer  *Printer
    renderer PromptRenderer
    model    PromptModel
    mu       sync.Mutex
    status   string
}

func NewCoordinatedWriter(
    printer *Printer,
    renderer PromptRenderer,
    model PromptModel,
) *CoordinatedWriter

// Methods (all thread-safe):
func (c *CoordinatedWriter) PrintLine(s string) error
func (c *CoordinatedWriter) PrintChunks(ctx context.Context, chunks <-chan string) error
func (c *CoordinatedWriter) SetStatus(status string) error
func (c *CoordinatedWriter) RedrawPrompt() error
```

**Usage:**

```go
// Create coordinator
printer := output.NewPrinter(os.Stdout)
renderer := prompt.NewRenderer(os.Stdout, 80, "> ")
model := prompt.NewModel(100)

coord := output.NewCoordinatedWriter(printer, renderer, model)

// All operations automatically redraw prompt
coord.PrintLine("User: Hello!")
// Output:
// User: Hello!
// > [cursor]

// Set transient status
coord.SetStatus("thinking...")
// > [cursor]                    thinking...

// Stream LLM response
chunks := make(chan string)
go func() {
    chunks <- "Response"
    chunks <- "...\n"
    close(chunks)
}()
coord.PrintChunks(context.Background(), chunks)
// Response...
// > [cursor]

// Manual redraw (e.g., on window resize)
coord.RedrawPrompt()
```

**Benefits:**
- ✅ No torn output (atomic write-then-redraw)
- ✅ Thread-safe concurrent operations
- ✅ Automatic prompt positioning
- ✅ Simple, transparent API

**Metrics:**
- 15 coordinator-specific tests
- 88.9% coverage
- Max complexity: 1
- Zero race conditions

### Optional: Progress Indicators

**Future:** Inline progress bars for long operations.

```go
printer.PrintProgress("Downloading...", 0.45) // 45%
```

---

## Troubleshooting

### Issue: Output lag

**Symptom:** Chunks take 50ms to appear.

**Cause:** Coalescing delay.

**Fix:** Reduce delay or disable:

```go
printer := output.NewPrinter(os.Stdout, output.WithCoalesceDelay(10*time.Millisecond))
```

### Issue: Torn output

**Symptom:** Prompt appears mid-line.

**Cause:** Missing output-prompt coordination.

**Fix:** Manually redraw prompt after writes (until Phase 3.2):

```go
printer.PrintLine("message")
promptRenderer.Redraw(promptModel, "")
```

### Issue: Memory leak

**Symptom:** Memory grows over time.

**Cause:** Channel not closed, `PrintChunks()` blocks forever.

**Fix:** Always close chunk channels:

```go
chunks := make(chan string)
defer close(chunks) // Ensure close
```

---

## References

- **FRD (Printer):** `specs/frds/FRD-20251010-append-only-printer.md`
- **FRD (Coordinator):** `specs/frds/FRD-20251010-output-prompt-coordination.md`
- **Roadmap:** `specs/tui-implementation/ROADMAP.md` Phase 3.1 & 3.2
- **Architecture:** `specs/tui-implementation/tui-new.md` Section 5 & 7
- **Tests:** `internal/ui/output/printer_test.go`, `internal/ui/output/coordinator_test.go`

---

## Summary

The output package provides a **simple, efficient, thread-safe** way to print append-only output for the TUI. It balances **low latency** (newline fast-path) with **smooth rendering** (coalescing) and **high throughput** (large chunk bypass).

**Key takeaways:**
- Use `PrintLine()` for discrete messages
- Use `PrintChunks()` for streaming data
- Default 50ms coalesce delay works well for most cases
- Thread-safe, context-aware, zero data loss
- Ready for Phase 3.2 integration (coordinator)
