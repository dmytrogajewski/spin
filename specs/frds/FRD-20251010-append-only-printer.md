# FRD-20251010: Append-Only Printer with Streaming

**Feature ID:** FRD-20251010
**Roadmap Phase:** 3.1 - Append-Only Printer with Streaming
**Priority:** P0 (Critical path)
**Complexity:** Low
**Status:** Draft
**Author:** Spin Agent
**Date:** 2025-10-10

---

## 1. Overview

Implement an append-only output printer for the TUI that supports both immediate line printing and streaming chunks with optional coalescing to reduce flicker. This component is responsible for writing chat transcript to stdout without ever clearing or repainting the viewport.

---

## 2. Goals

### 2.1 Primary Goals

- **Append-only output**: Write lines to stdout with real newlines, preserving terminal scrollback
- **Streaming support**: Handle incremental chunks from LLM responses efficiently
- **Flicker reduction**: Optional chunk coalescing to reduce redraw frequency
- **Thread-safe**: Support concurrent writes from multiple goroutines
- **Performance**: Handle 1k+ lines/sec without lag

### 2.2 Non-Goals

- Full-screen rendering (that's alt-screen TUI territory)
- ANSI color/styling (future enhancement)
- Block system integration (Phase 4)
- Virtualization (Phase 4.3)

---

## 3. Architecture

### 3.1 Package Structure

```
/internal/ui/output/
  printer.go           # Printer implementation
  printer_test.go      # Unit tests
```

### 3.2 Core Types

```go
// Printer handles append-only output to stdout for chat transcript.
// It supports both immediate line printing and streaming chunks with
// optional coalescing to reduce flicker.
type Printer struct {
    out           io.Writer
    mu            sync.Mutex
    coalesceDelay time.Duration  // Default: 50ms
}

// PrinterOption is a functional option for configuring the Printer.
type PrinterOption func(*Printer)
```

### 3.3 Public API

```go
// NewPrinter creates a new Printer with optional configuration.
func NewPrinter(out io.Writer, opts ...PrinterOption) *Printer

// WithCoalesceDelay sets the coalescing delay for streaming chunks.
func WithCoalesceDelay(d time.Duration) PrinterOption

// PrintLine writes a line immediately with a newline.
// Thread-safe.
func (p *Printer) PrintLine(s string) error

// PrintChunks streams chunks from a channel with optional coalescing.
// Flushes immediately on newline or after coalesceDelay.
// Blocks until channel closes or context is canceled.
// Thread-safe.
func (p *Printer) PrintChunks(ctx context.Context, chunks <-chan string) error
```

---

## 4. Detailed Design

### 4.1 PrintLine Behavior

**Purpose**: Immediate line output for discrete messages (user input, system notices, etc.)

**Implementation:**
1. Acquire mutex lock
2. Write string + `\n` to output writer
3. Flush if writer supports `Flush()` interface
4. Release lock
5. Return any I/O error

**Edge Cases:**
- Empty string: Write newline only
- Multi-line string: Write as-is (caller responsibility to split)
- Writer error: Return immediately, no retry

### 4.2 PrintChunks Behavior

**Purpose**: Streaming output for LLM responses, minimizing flicker

**Implementation:**
1. Create internal buffer (`strings.Builder`)
2. Start timer with `coalesceDelay` (default 50ms)
3. Loop:
   - Select on chunk channel, timer, and context
   - On chunk received:
     - Append to buffer
     - If chunk contains `\n`, flush immediately (fast-path)
   - On timer tick:
     - Flush buffer if non-empty
     - Reset timer
   - On channel close:
     - Flush remaining buffer
     - Return nil
   - On context cancel:
     - Flush remaining buffer
     - Return `ctx.Err()`

**Flush operation:**
1. Acquire mutex lock
2. Write buffer contents to output writer
3. Clear buffer
4. Release lock

**Edge Cases:**
- Channel closed immediately: No-op
- Context canceled mid-stream: Flush partial content, return `ctx.Err()`
- Timer fires before first chunk: No-op (buffer empty)
- Large chunks (>10KB): Write immediately without buffering
- Newline detection: Scan for `\n` anywhere in chunk (not just trailing)

### 4.3 Thread Safety

**Guarantees:**
- Multiple goroutines can call `PrintLine` concurrently
- Multiple goroutines can call `PrintChunks` concurrently
- `PrintLine` and `PrintChunks` can run concurrently
- Writes are atomic (no torn output)

**Mechanism:**
- Single `sync.Mutex` protects all writes to `out`
- Lock held only during write operation (minimal critical section)
- Channel operations outside lock (no deadlock risk)

### 4.4 Performance Considerations

**Coalescing Strategy:**
- **Small chunks (<1KB)**: Buffer for up to `coalesceDelay` (50ms default)
- **Large chunks (≥10KB)**: Write immediately (bypass buffering)
- **Newline detection**: Flush on any `\n` to keep prompt responsive

**Benchmarks (target):**
- `PrintLine`: <1µs per call (excluding I/O)
- `PrintChunks`: Handle 1000 chunks/sec with <5ms p99 latency
- Memory: O(1) per `Printer` instance (no unbounded growth)

---

## 5. Testing Strategy

### 5.1 Unit Tests

**Test Coverage:**

1. **PrintLine:**
   - Write single line → verify output
   - Write empty line → verify single `\n`
   - Write multiple lines concurrently → verify all present
   - Writer error → return error

2. **PrintChunks:**
   - Single chunk → verify output
   - Multiple chunks → verify coalescing
   - Chunk with newline → verify immediate flush
   - Channel close → verify final flush
   - Context cancel → verify partial flush + error
   - Empty channel → no output
   - Large chunk (>10KB) → immediate write

3. **Concurrency:**
   - Concurrent `PrintLine` calls → no race, all written
   - Concurrent `PrintChunks` calls → no race, all written
   - Interleaved `PrintLine` + `PrintChunks` → atomic writes

4. **Options:**
   - `WithCoalesceDelay(100ms)` → verify delay applied
   - `WithCoalesceDelay(0)` → immediate writes

### 5.2 Test Utilities

```go
// testkit/fake_writer.go
type FakeWriter struct {
    mu      sync.Mutex
    buffer  bytes.Buffer
    writeErr error
}

func (f *FakeWriter) Write(p []byte) (int, error)
func (f *FakeWriter) String() string
func (f *FakeWriter) SetWriteError(err error)
```

### 5.3 Benchmarks

```go
func BenchmarkPrintLine(b *testing.B)
func BenchmarkPrintChunksSmall(b *testing.B)   // 100 chunks, 10 bytes each
func BenchmarkPrintChunksLarge(b *testing.B)   // 10 chunks, 10KB each
func BenchmarkConcurrentWrites(b *testing.B)   // Parallel PrintLine
```

---

## 6. API Examples

### 6.1 Basic Usage

```go
// Create printer with default settings (50ms coalesce)
printer := output.NewPrinter(os.Stdout)

// Print discrete lines
printer.PrintLine("User: Hello, AI!")
printer.PrintLine("Assistant: Hi! How can I help?")

// Stream LLM response
chunks := make(chan string, 10)
go func() {
    chunks <- "I can help"
    chunks <- " you with"
    chunks <- " coding tasks.\n"
    close(chunks)
}()

ctx := context.Background()
printer.PrintChunks(ctx, chunks)
```

### 6.2 Custom Coalesce Delay

```go
// Faster updates (10ms coalesce)
printer := output.NewPrinter(os.Stdout,
    output.WithCoalesceDelay(10 * time.Millisecond))

// No coalescing (immediate writes)
printer := output.NewPrinter(os.Stdout,
    output.WithCoalesceDelay(0))
```

### 6.3 Context Cancellation

```go
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

chunks := streamLLMResponse() // slow infinite stream
err := printer.PrintChunks(ctx, chunks)
if err == context.DeadlineExceeded {
    printer.PrintLine("\n[Stream timeout]")
}
```

---

## 7. Quality Gates

### 7.1 Code Quality

- ✅ `go test -race ./internal/ui/output/...` passes
- ✅ Coverage ≥90% (critical path: PrintLine, PrintChunks)
- ✅ `make lint` clean (zero errors)
- ✅ Cyclomatic complexity ≤8 per function
- ✅ Godoc on all exports

### 7.2 Performance

- ✅ `PrintLine` < 1µs (excluding I/O)
- ✅ `PrintChunks` handles 1000 chunks/sec
- ✅ No goroutine leaks (verify with pprof)
- ✅ No race conditions (`-race` detector clean)

### 7.3 Behavior

- ✅ Immediate flush on newline (no lag)
- ✅ Coalescing reduces write syscalls (measure with strace)
- ✅ Thread-safe concurrent writes
- ✅ Context cancellation works correctly

---

## 8. Integration Points

### 8.1 Current Dependencies

- **Standard Library:**
  - `io.Writer` interface (output abstraction)
  - `context.Context` (cancellation)
  - `sync.Mutex` (thread safety)
  - `time.Timer` (coalescing)

### 8.2 Future Integration (Phase 3.2)

- **Output-Prompt Coordination:**
  - `Printer` will be wrapped by `CoordinatedWriter`
  - After each write, prompt will be redrawn
  - Shared mutex prevents torn output

### 8.3 Future Integration (Phase 5.1)

- **PureTTY Adapter:**
  - Adapter will own `Printer` instance
  - Map UI port `PrintLine`/`PrintChunks` to `Printer` methods
  - Connect LLM stream events → `PrintChunks`

---

## 9. Open Questions

1. **Coalesce delay default:** 50ms feels right for smooth UX, but should it be configurable per-call?
   - **Decision:** Global setting via `WithCoalesceDelay`, can be overridden in future if needed

2. **Large chunk threshold:** 10KB cutoff for immediate write?
   - **Decision:** Start with 10KB, measure in practice, tune if needed

3. **Newline handling:** Should we normalize `\r\n` → `\n`?
   - **Decision:** No normalization; pass-through as-is (caller responsibility)

4. **Writer interface:** Should we support `io.StringWriter` optimization?
   - **Decision:** Yes, check for `io.StringWriter` and use `WriteString()` when available

---

## 10. Success Metrics

- All tests pass with `-race` detector
- Coverage ≥90%
- Lint clean
- Complexity ≤8
- Benchmarks meet targets (1µs, 1k chunks/sec)
- Integration with Phase 3.2 (Coordinator) works seamlessly
- No performance regressions in TUI demo

---

## 11. Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Coalescing causes lag | High | Fast-path flush on newline; configurable delay |
| Race conditions in concurrent writes | High | Comprehensive `-race` tests; mutex on all writes |
| Deadlock with coordinator (future) | Medium | Minimal lock duration; channel ops outside lock |
| Memory leak in chunk buffering | Medium | Buffer reset after flush; bounded buffer size |

---

## 12. Timeline

- **FRD Review:** 1 hour
- **Implementation:** 2 hours
- **Testing:** 2 hours
- **Iteration:** 1 hour
- **Total:** ~6 hours (half day)

---

## 13. References

- **Roadmap:** `specs/tui-implementation/ROADMAP.md`, Phase 3.1
- **Architecture Spec:** `specs/tui-implementation/tui-new.md`, Section 5 (Output System)
- **AGENTS.md:** Quality gates, testing philosophy
- **Similar Patterns:** `/internal/ui/prompt/renderer.go` (single-line redraw)

---

## 14. Acceptance Criteria

Phase 3.1 is **complete** when:

- [x] FRD written and aligned with roadmap
- [ ] `internal/ui/output/printer.go` implemented
- [ ] `internal/ui/output/printer_test.go` with ≥90% coverage
- [ ] All tests pass with `-race`
- [ ] `make lint` clean
- [ ] `uast parse | herr analyze` clean (complexity ≤8)
- [ ] Benchmarks demonstrate performance targets
- [ ] ROADMAP.md updated (Phase 3.1 marked complete)
- [ ] Ready for Phase 3.2 integration

---

**Status:** Ready for implementation ✅
