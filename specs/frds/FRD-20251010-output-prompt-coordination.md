# FRD-20251010: Output-Prompt Coordination (Race-Free)

**Feature ID:** FRD-20251010-02
**Roadmap Phase:** 3.2 - Output-Prompt Coordination (Race-Free)
**Priority:** P0 (Critical path)
**Complexity:** Medium
**Status:** Draft
**Author:** Spin Agent
**Date:** 2025-10-10

---

## 1. Overview

Implement coordination between the output printer and prompt renderer to avoid race conditions when writing to stdout. Any stdout write must be followed by an automatic prompt redraw to keep the prompt at the bottom of the screen without torn output.

This completes the output subsystem (Phase 3) by ensuring atomic write-then-redraw operations, making the TUI robust for concurrent output from multiple sources.

---

## 2. Goals

### 2.1 Primary Goals

- **Atomic coordination**: Every output write followed by automatic prompt redraw
- **Race-free**: No torn output (partial line + prompt interleaved)
- **Thread-safe**: Support concurrent calls to print operations
- **Minimal interface**: Simple wrapper that "just works"
- **No deadlocks**: Careful lock ordering and duration

### 2.2 Non-Goals

- Event bus architecture (KISS: use mutex wrapper)
- Complex state machine (single lock is sufficient)
- Retry logic for write errors (fail-fast)
- ANSI cursor save/restore optimizations (future enhancement)

---

## 3. Architecture

### 3.1 Package Structure

```
/internal/ui/output/
  printer.go           # Existing Printer implementation
  coordinator.go       # NEW: CoordinatedWriter
  printer_test.go      # Existing tests
  coordinator_test.go  # NEW: Coordinator tests
```

### 3.2 Core Types

```go
// PromptRenderer is the interface required for prompt redrawing.
// Implemented by prompt.Renderer.
type PromptRenderer interface {
    Redraw(model PromptModel, status string) error
}

// PromptModel is the interface for accessing prompt state.
// Implemented by prompt.Model.
type PromptModel interface {
    Text() string
    Cursor() int
}

// CoordinatedWriter wraps a Printer and PromptRenderer to ensure
// atomic write-then-redraw operations.
type CoordinatedWriter struct {
    printer  *Printer
    renderer PromptRenderer
    model    PromptModel
    mu       sync.Mutex
}
```

### 3.3 Public API

```go
// NewCoordinatedWriter creates a new coordinator that wraps printer
// and automatically redraws the prompt after each write.
func NewCoordinatedWriter(
    printer *Printer,
    renderer PromptRenderer,
    model PromptModel,
) *CoordinatedWriter

// PrintLine writes a line and redraws the prompt atomically.
// Thread-safe.
func (c *CoordinatedWriter) PrintLine(s string) error

// PrintChunks streams chunks and redraws the prompt after each flush.
// Thread-safe. Blocks until channel closes or context cancels.
func (c *CoordinatedWriter) PrintChunks(ctx context.Context, chunks <-chan string) error

// SetStatus updates the prompt status text and redraws.
// Thread-safe.
func (c *CoordinatedWriter) SetStatus(status string) error

// RedrawPrompt manually triggers a prompt redraw.
// Thread-safe. Useful for window resize or explicit refresh.
func (c *CoordinatedWriter) RedrawPrompt() error
```

---

## 4. Detailed Design

### 4.1 Coordination Strategy

**Pattern:** Wrap every write operation with prompt redraw.

**Implementation:**
1. Acquire mutex lock
2. Call underlying `Printer` method (PrintLine/PrintChunks)
3. Call `renderer.Redraw(model, status)`
4. Release lock
5. Return any error (from print or redraw)

**Key insight:** Prompt renderer writes to the *same* `io.Writer` as the printer, so the lock guarantees atomic `[output]\n[prompt redraw]` sequences.

### 4.2 PrintLine Coordination

```go
func (c *CoordinatedWriter) PrintLine(s string) error {
    c.mu.Lock()
    defer c.mu.Unlock()

    // Write line via printer
    if err := c.printer.PrintLine(s); err != nil {
        return fmt.Errorf("print line: %w", err)
    }

    // Redraw prompt
    if err := c.renderer.Redraw(c.model, c.status); err != nil {
        return fmt.Errorf("redraw prompt: %w", err)
    }

    return nil
}
```

**Thread safety:**
- Lock held for entire operation (print + redraw)
- Multiple goroutines calling `PrintLine` are serialized
- No partial writes visible to user

### 4.3 PrintChunks Coordination

**Challenge:** Printer's `PrintChunks` is blocking and long-running. We cannot hold the lock for the entire duration (would block other prints).

**Solution:** Hook into printer's flush points:

```go
func (c *CoordinatedWriter) PrintChunks(ctx context.Context, chunks <-chan string) error {
    // Create internal channel for coordinated chunks
    coordChunks := make(chan string, cap(chunks))
    defer close(coordChunks)

    // Forward chunks and trigger redraws on flush
    errChan := make(chan error, 1)
    go func() {
        errChan <- c.printer.PrintChunks(ctx, coordChunks)
    }()

    // Process chunks and coordinate redraws
    for {
        select {
        case <-ctx.Done():
            c.mu.Lock()
            c.renderer.Redraw(c.model, c.status)
            c.mu.Unlock()
            return ctx.Err()

        case chunk, ok := <-chunks:
            if !ok {
                // Wait for printer to finish
                err := <-errChan
                c.mu.Lock()
                c.renderer.Redraw(c.model, c.status)
                c.mu.Unlock()
                return err
            }

            // Forward chunk
            coordChunks <- chunk

            // Redraw on newline (when printer flushes)
            if strings.Contains(chunk, "\n") {
                c.mu.Lock()
                c.renderer.Redraw(c.model, c.status)
                c.mu.Unlock()
            }
        }
    }
}
```

**Alternative (simpler):** Redraw after entire stream completes:

```go
func (c *CoordinatedWriter) PrintChunks(ctx context.Context, chunks <-chan string) error {
    // Let printer handle streaming (it has internal coordination)
    err := c.printer.PrintChunks(ctx, chunks)

    // Redraw prompt after stream completes
    c.mu.Lock()
    defer c.mu.Unlock()
    if rerr := c.renderer.Redraw(c.model, c.status); rerr != nil {
        if err == nil {
            err = rerr
        }
    }

    return err
}
```

**Decision:** Use simpler approach initially. Printer's internal coalescing already handles flicker. Redraw once at end is sufficient for MVP.

### 4.4 Status Management

```go
type CoordinatedWriter struct {
    // ...
    status string  // Current status text (protected by mu)
}

func (c *CoordinatedWriter) SetStatus(status string) error {
    c.mu.Lock()
    defer c.mu.Unlock()

    c.status = status
    return c.renderer.Redraw(c.model, c.status)
}
```

**Thread safety:**
- Status is private field, accessed only under lock
- Every status update triggers immediate redraw

### 4.5 Manual Redraw (for SIGWINCH)

```go
func (c *CoordinatedWriter) RedrawPrompt() error {
    c.mu.Lock()
    defer c.mu.Unlock()

    return c.renderer.Redraw(c.model, c.status)
}
```

**Use case:** Window resize (SIGWINCH) handler calls this to reflow prompt.

---

## 5. Testing Strategy

### 5.1 Unit Tests

**Test Coverage:**

1. **PrintLine:**
   - Single call → verify output + prompt redraw
   - Concurrent calls → verify serialization, no torn output
   - Printer error → error returned, no redraw called
   - Renderer error → printer succeeds, error returned

2. **PrintChunks:**
   - Single stream → verify output + final redraw
   - Context cancel → verify partial output + final redraw
   - Concurrent streams → verify serialization

3. **SetStatus:**
   - Update status → verify redraw with new status
   - Concurrent updates → verify serialization

4. **RedrawPrompt:**
   - Manual redraw → verify renderer called with current model/status

5. **Integration:**
   - Interleave PrintLine + PrintChunks + SetStatus → verify correct output order
   - Verify prompt always redrawn after output

### 5.2 Test Utilities

```go
// testkit/fake_renderer.go
type FakeRenderer struct {
    mu         sync.Mutex
    redraws    []RedrawCall
    redrawErr  error
}

type RedrawCall struct {
    Text   string
    Cursor int
    Status string
}

func (f *FakeRenderer) Redraw(model PromptModel, status string) error
func (f *FakeRenderer) Redraws() []RedrawCall
func (f *FakeRenderer) SetRedrawError(err error)
```

### 5.3 Race Detection

All tests must pass with `-race`:

```bash
go test -race ./internal/ui/output/...
```

**Specific race scenarios:**
- Concurrent `PrintLine` calls from multiple goroutines
- Concurrent `SetStatus` calls
- `PrintLine` + `SetStatus` interleaved
- `PrintChunks` + `PrintLine` concurrent

---

## 6. API Examples

### 6.1 Basic Setup

```go
// Create components
printer := output.NewPrinter(os.Stdout)
renderer := prompt.NewRenderer(os.Stdout, 80, "> ")
model := prompt.NewModel(100)

// Create coordinator
coord := output.NewCoordinatedWriter(printer, renderer, model)

// Now all prints automatically redraw prompt
coord.PrintLine("User: Hello!")
// Output:
// User: Hello!
// > [cursor]
```

### 6.2 Streaming with Coordination

```go
chunks := make(chan string, 10)
go func() {
    chunks <- "Thinking"
    chunks <- "..."
    chunks <- "\n"
    close(chunks)
}()

ctx := context.Background()
coord.PrintChunks(ctx, chunks)
// Output:
// Thinking...
// > [cursor]
```

### 6.3 Status Updates

```go
coord.SetStatus("typing...")
// Prompt redraws with status:
// > [cursor]                    typing...

time.Sleep(100 * time.Millisecond)
coord.SetStatus("")
// Prompt redraws without status:
// > [cursor]
```

### 6.4 Window Resize

```go
// On SIGWINCH handler:
func handleResize(coord *output.CoordinatedWriter, newWidth int) {
    renderer.SetWidth(newWidth)
    coord.RedrawPrompt()
}
```

---

## 7. Quality Gates

### 7.1 Code Quality

- ✅ `go test -race ./internal/ui/output/...` passes
- ✅ Coverage ≥90% (critical path: coordination logic)
- ✅ `make lint` clean (zero errors)
- ✅ Cyclomatic complexity ≤10 per function
- ✅ Godoc on all exports

### 7.2 Behavior

- ✅ No torn output (prompt never interleaved mid-line)
- ✅ Prompt always visible after output
- ✅ Thread-safe concurrent operations
- ✅ No deadlocks under load
- ✅ No goroutine leaks

### 7.3 Performance

- ✅ Coordination overhead < 10µs per operation
- ✅ No performance regression vs Phase 3.1
- ✅ Prompt redraw completes within 1ms (typical terminal)

---

## 8. Integration Points

### 8.1 Dependencies

- **Printer (`output.Printer`)**: Existing Phase 3.1 implementation
- **Renderer (`prompt.Renderer`)**: Existing Phase 2.2 implementation
- **Model (`prompt.Model`)**: Existing Phase 2.1 implementation

### 8.2 Future Integration (Phase 5.1)

- **PureTTY Adapter:**
  - Will create `CoordinatedWriter` instance
  - Map UI port methods to coordinator
  - Handle SIGWINCH → `coord.RedrawPrompt()`

---

## 9. Design Decisions

### 9.1 Mutex vs Event Bus

**Decision:** Use mutex wrapper (simple, proven)

**Rationale:**
- KISS principle: mutex is simpler than event bus
- Single lock sufficient for coordination
- No need for pub/sub complexity

**Alternative considered:** Event bus with "OutputFlushed" event
- **Rejected:** Over-engineering for this use case

### 9.2 Lock Duration

**Decision:** Hold lock for entire write+redraw operation

**Rationale:**
- Ensures atomicity (no torn output)
- Redraw is fast (~1ms), acceptable critical section
- Simplifies reasoning about correctness

**Alternative considered:** Fine-grained locking (separate locks for print/redraw)
- **Rejected:** Risk of torn output if not carefully ordered

### 9.3 PrintChunks Coordination

**Decision:** Redraw once after entire stream completes

**Rationale:**
- Printer's internal coalescing already reduces flicker
- Redrawing on every chunk adds overhead
- Final redraw is sufficient for correctness

**Future enhancement:** Hook into printer's flush events for real-time redraw

---

## 10. Open Questions

1. **ANSI cursor save/restore:** Should we optimize with `\x1b[s` and `\x1b[u`?
   - **Decision:** Not in MVP; current approach works on all terminals
   - **Future:** Measure if optimization is needed

2. **Error handling:** Should we retry on redraw errors?
   - **Decision:** No retry; fail-fast and return error to caller
   - **Rationale:** Terminal write errors are usually fatal (broken pipe, etc.)

3. **Status persistence:** Should status survive across multiple prints?
   - **Decision:** Yes, status persists until explicitly cleared with `SetStatus("")`

---

## 11. Success Metrics

- All tests pass with `-race` detector
- Coverage ≥90%
- Lint clean
- Complexity ≤10
- Manual testing: no visual glitches (torn output, missing prompt)
- Integration with Phase 5.1 (PureTTY adapter) works seamlessly

---

## 12. Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Deadlock with nested locks | High | Single lock; minimal critical section |
| Performance degradation | Medium | Measure overhead; optimize if needed |
| Redraw errors break prompt | Medium | Fail-fast; return error to caller |
| Race in status updates | High | Status protected by mutex |

---

## 13. Timeline

- **FRD Review:** 1 hour
- **Implementation:** 3 hours
- **Testing:** 3 hours
- **Iteration:** 1 hour
- **Total:** ~8 hours (1 day)

---

## 14. References

- **Roadmap:** `specs/tui-implementation/ROADMAP.md`, Phase 3.2
- **Architecture Spec:** `specs/tui-implementation/tui-new.md`, Section 7 (Prompt/Output Coordination)
- **AGENTS.md:** Quality gates, testing philosophy
- **Printer FRD:** `specs/frds/FRD-20251010-append-only-printer.md`
- **Renderer:** `/internal/ui/prompt/renderer.go`

---

## 15. Acceptance Criteria

Phase 3.2 is **complete** when:

- [ ] FRD written and aligned with roadmap
- [ ] `internal/ui/output/coordinator.go` implemented
- [ ] `internal/ui/output/coordinator_test.go` with ≥90% coverage
- [ ] All tests pass with `-race`
- [ ] `make lint` clean
- [ ] `uast parse | herr analyze` clean (complexity ≤10)
- [ ] Manual testing: no torn output
- [ ] ROADMAP.md updated (Phase 3.2 marked complete)
- [ ] Ready for Phase 5.1 integration (PureTTY adapter)

---

**Status:** Ready for implementation ✅
