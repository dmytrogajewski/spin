# FRD-20251010: PureTTY Adapter (Ports Interface)

**Phase:** 5.1
**Priority:** P0 (Critical path)
**Complexity:** Medium
**Status:** Draft
**Created:** 2025-10-10
**Related Roadmap:** [ROADMAP.md](../tui-implementation/ROADMAP.md) Phase 5.1

---

## 1. Overview

Implement the `UI` port interface using PureTTY adapter that composes TTY, Prompt, Output, and Timeline subsystems into a unified interface for the Spin application. This adapter provides a complete native-scrollback TUI without alt-screen buffer.

---

## 2. Goals

- **Compose subsystems** into a unified `UI` port interface
- **Lifecycle management**: Run() blocks, Stop() restores terminal
- **Event-driven architecture**: React to keyboard events, SIGWINCH, context cancellation
- **Thread-safe operations**: Concurrent output, input, and status updates
- **Clean shutdown**: No goroutine leaks, terminal restored to original state
- **Testable**: Fake TTY, fake writer, fake keyboard for unit tests

---

## 3. Non-Goals

- **Block rendering integration** (deferred to Phase 6.1)
- **BubbleteaHybrid adapter** (deferred to Phase 5.2)
- **Command palette** (deferred to Phase 6.2)
- **File preview** (deferred to Phase 6.3)
- **Core event integration** (deferred to Phase 7.4)

---

## 4. Requirements

### 4.1 Functional Requirements

**FR-1: Port Interface Implementation**
- Implement `UI` port interface from `internal/ui/ports/ui.go`
- All methods must be thread-safe
- Lifecycle methods: `Run(ctx)`, `Stop()`
- Output methods: `PrintLine(s)`, `PrintChunks(ctx, ch)`
- Prompt methods: `SetStatus(s)`, `RequestInput() <-chan string`

**FR-2: Component Composition**
- Compose `TTY` (terminal control)
- Compose `Renderer` + `Model` (prompt system)
- Compose `Printer` (output system)
- Compose `CoordinatedWriter` (output-prompt coordination)

**FR-3: Lifecycle Management**
- `Run(ctx)` enters raw mode, hides cursor, starts event loop
- `Run(ctx)` blocks until context cancel, Ctrl-C, or Ctrl-D
- `Stop()` exits raw mode, shows cursor, cleans up goroutines
- `Stop()` is idempotent (safe to call multiple times)

**FR-4: Event Loop**
- Select on:
  - Keyboard events (from TTY reader)
  - Submitted lines (from prompt loop)
  - Context cancellation
  - SIGWINCH (terminal resize)
- Dispatch events to appropriate handlers

**FR-5: Keyboard Event Handling**
- Map `KeyEvent` to prompt model operations
- Handle Ctrl-C, Ctrl-D for graceful shutdown
- Handle resize events (update model width, redraw)

**FR-6: Output Operations**
- `PrintLine(s)`: delegate to `CoordinatedWriter.PrintLine(s)`
- `PrintChunks(ctx, ch)`: delegate to `CoordinatedWriter.PrintChunks(ctx, ch)`
- Both operations automatically redraw prompt (via coordinator)

**FR-7: Status Updates**
- `SetStatus(s)`: update model status, delegate to `CoordinatedWriter.SetStatus(s)`
- Status appears right-aligned in prompt line

**FR-8: Input Handling**
- `RequestInput()` returns channel from prompt loop
- Channel emits submitted lines (on Enter key)
- Channel closed on shutdown

---

### 4.2 Non-Functional Requirements

**NFR-1: Thread Safety**
- All public methods callable from multiple goroutines
- No data races (verified with `go test -race`)

**NFR-2: Clean Shutdown**
- No goroutine leaks (verified with integration tests)
- Terminal restored even on panic (via defer)
- Channels closed properly

**NFR-3: Testability**
- Accept injectable dependencies (TTY, renderer, model, printer)
- Support fake implementations for unit tests
- Integration tests with real PTY

**NFR-4: Performance**
- Event loop latency <10ms (keyboard to screen)
- No blocking operations in critical path
- Minimal allocations in hot loop

---

## 5. Design

### 5.1 Port Interface

```go
// internal/ui/ports/ui.go
package ports

import "context"

// UI port defines the interface for TUI implementations.
type UI interface {
    // Lifecycle
    Run(ctx context.Context) error
    Stop() error

    // Output (append-only)
    PrintLine(line string) error
    PrintChunks(ctx context.Context, chunks <-chan string) error

    // Prompt control
    SetStatus(text string) error
    RequestInput() <-chan string
}
```

---

### 5.2 PureTTY Adapter Structure

```go
// internal/ui/adapters/puretty.go
package adapters

import (
    "context"
    "io"
    "sync"

    "github.com/dmytrogajewski/spin/internal/ui/output"
    "github.com/dmytrogajewski/spin/internal/ui/prompt"
    "github.com/dmytrogajewski/spin/internal/ui/term"
)

// PureTTY implements ports.UI using native terminal control.
type PureTTY struct {
    tty    *term.TTY
    model  *prompt.Model
    coord  *output.CoordinatedWriter
    keys   <-chan term.KeyEvent
    inputs <-chan string

    mu      sync.Mutex
    running bool
    stopped bool
}

// Options for PureTTY creation
type PureTTYOption func(*PureTTY)

func WithTTY(tty *term.TTY) PureTTYOption
func WithModel(model *prompt.Model) PureTTYOption
func WithCoordinator(coord *output.CoordinatedWriter) PureTTYOption

// NewPureTTY creates a new PureTTY adapter.
func NewPureTTY(out io.Writer, opts ...PureTTYOption) (*PureTTY, error)
```

**Default behavior** (when no options provided):
- Create TTY with stdin/stdout
- Create Model with default settings (100-entry history, "> " prefix)
- Create Renderer with stdout
- Create Printer with stdout
- Create CoordinatedWriter wrapping printer + renderer + model

---

### 5.3 Run Loop

```go
func (u *PureTTY) Run(ctx context.Context) error {
    // 1. Enter raw mode, hide cursor
    if err := u.tty.Enter(); err != nil {
        return fmt.Errorf("enter raw mode: %w", err)
    }
    defer u.tty.Exit()
    defer fmt.Fprint(u.tty.Out(), term.ShowCursor)

    // 2. Start keyboard reader
    keys := make(chan term.KeyEvent, 100)
    go u.tty.ReadKeys(ctx, keys)

    // 3. Start prompt loop
    inputs := u.startPromptLoop(ctx, keys)

    // 4. Setup SIGWINCH handler
    u.tty.OnResize(func(w, h int) {
        u.handleResize(w, h)
    })

    // 5. Initial prompt draw
    u.coord.RedrawPrompt()

    // 6. Event loop
    for {
        select {
        case line, ok := <-inputs:
            if !ok {
                return nil // Clean shutdown
            }
            u.handleSubmittedLine(line)

        case <-ctx.Done():
            return ctx.Err()
        }
    }
}
```

**Prompt Loop** (runs in background goroutine):
```go
func (u *PureTTY) startPromptLoop(ctx context.Context, keys <-chan term.KeyEvent) <-chan string {
    loop := prompt.NewLoop(u.coord, u.model, keys)
    return loop.Run(ctx)
}
```

---

### 5.4 Stop

```go
func (u *PureTTY) Stop() error {
    u.mu.Lock()
    defer u.mu.Unlock()

    if u.stopped {
        return nil // Already stopped
    }

    u.stopped = true

    // TTY cleanup handled by Run's defer
    // Goroutines cleaned up by context cancel
    return nil
}
```

---

### 5.5 Output Operations

```go
func (u *PureTTY) PrintLine(line string) error {
    return u.coord.PrintLine(line)
}

func (u *PureTTY) PrintChunks(ctx context.Context, chunks <-chan string) error {
    return u.coord.PrintChunks(ctx, chunks)
}
```

---

### 5.6 Status & Input

```go
func (u *PureTTY) SetStatus(text string) error {
    return u.coord.SetStatus(text)
}

func (u *PureTTY) RequestInput() <-chan string {
    return u.inputs
}
```

---

### 5.7 Resize Handling

```go
func (u *PureTTY) handleResize(w, h int) {
    u.model.Width = w
    u.coord.RedrawPrompt()
}
```

---

### 5.8 Submitted Line Handling

```go
func (u *PureTTY) handleSubmittedLine(line string) {
    // Echo user input to transcript
    u.coord.PrintLine(u.model.Prefix + line)
}
```

---

## 6. Implementation Plan

### 6.1 Phase 1: Port Definition
1. Create `internal/ui/ports/ui.go` with `UI` interface
2. Add godoc comments

### 6.2 Phase 2: Adapter Skeleton
1. Create `internal/ui/adapters/puretty.go`
2. Implement `PureTTY` struct with fields
3. Implement constructor `NewPureTTY()`
4. Implement options pattern

### 6.3 Phase 3: Lifecycle
1. Implement `Run(ctx)` with event loop
2. Implement `Stop()` with cleanup
3. Add defer guards for terminal restoration

### 6.4 Phase 4: Event Handlers
1. Implement `startPromptLoop()`
2. Implement `handleResize()`
3. Implement `handleSubmittedLine()`

### 6.5 Phase 5: Public API
1. Implement `PrintLine()`
2. Implement `PrintChunks()`
3. Implement `SetStatus()`
4. Implement `RequestInput()`

### 6.6 Phase 6: Tests
1. Unit tests with fake TTY/writer/keyboard
2. Integration tests with real PTY (if feasible)
3. Lifecycle tests (start, stop, restart)
4. Shutdown tests (context cancel, Ctrl-C, Ctrl-D)
5. Resize tests (SIGWINCH during operation)
6. Race detection tests

---

## 7. Testing Strategy

### 7.1 Unit Tests

**Fake Implementations:**
```go
// internal/ui/adapters/testkit/fake_tty.go
type FakeTTY struct {
    width, height int
    entered       bool
    events        chan term.KeyEvent
}

func (f *FakeTTY) Enter() error
func (f *FakeTTY) Exit() error
func (f *FakeTTY) Size() (w, h int)
func (f *FakeTTY) OnResize(cb func(w, h int))
func (f *FakeTTY) ReadKeys(ctx context.Context, out chan<- term.KeyEvent) error
func (f *FakeTTY) InjectKey(key term.KeyEvent)
func (f *FakeTTY) InjectResize(w, h int)
```

**Test Cases:**

1. **TestNewPureTTY**: Constructor with options
2. **TestRun_Basic**: Start event loop, receive input, shutdown
3. **TestRun_ContextCancel**: Graceful shutdown on context cancel
4. **TestRun_CtrlC**: Shutdown on Ctrl-C
5. **TestRun_CtrlD**: Shutdown on Ctrl-D
6. **TestStop_Idempotent**: Multiple Stop() calls safe
7. **TestPrintLine**: Output echoed correctly
8. **TestPrintChunks**: Streaming chunks
9. **TestSetStatus**: Status update triggers redraw
10. **TestRequestInput**: Channel returns submitted lines
11. **TestResize**: SIGWINCH updates width, triggers redraw
12. **TestConcurrent**: PrintLine + PrintChunks + SetStatus concurrent
13. **TestCleanShutdown**: No goroutine leaks

---

### 7.2 Integration Tests

**With Real PTY** (if feasible):
```go
func TestPureTTY_Integration(t *testing.T) {
    pty, tty, err := pty.Open()
    if err != nil {
        t.Skip("PTY not available")
    }
    defer pty.Close()

    ui, _ := adapters.NewPureTTY(tty)
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()

    go ui.Run(ctx)

    // Inject keys via PTY master
    pty.Write([]byte("hello\n"))

    // Read from RequestInput channel
    inputs := ui.RequestInput()
    select {
    case line := <-inputs:
        assert.Equal(t, "hello", line)
    case <-time.After(1 * time.Second):
        t.Fatal("timeout waiting for input")
    }
}
```

---

## 8. Acceptance Criteria

- [ ] All tests pass with `-race`
- [ ] Coverage ≥85%
- [ ] `make lint` clean
- [ ] Complexity ≤15 per function
- [ ] UI port interface fully implemented
- [ ] Run() blocks until context cancel or quit
- [ ] Stop() restores terminal (cursor visible, cooked mode)
- [ ] No goroutine leaks on shutdown (verified)
- [ ] Resize events correctly update prompt width
- [ ] Submitted lines appear on RequestInput() channel
- [ ] PrintLine echoes to transcript + redraws prompt
- [ ] PrintChunks streams correctly + redraws prompt after
- [ ] SetStatus updates right-aligned status
- [ ] Thread-safe concurrent operations (verified with -race)

---

## 9. Quality Gates

**Tests:**
- ✅ All passing with `-race`
- ✅ Coverage ≥85%
- ✅ No goroutine leaks

**Code:**
- ✅ `make lint` passes (zero errors)
- ✅ Complexity ≤15 per function
- ✅ No dead code
- ✅ Godoc on all exports

---

## 10. Dependencies

**Completed:**
- Phase 1.1: Terminal Control (TTY) ✅
- Phase 1.2: Keyboard Events ✅
- Phase 2.1: Prompt Model ✅
- Phase 2.2: Prompt Renderer ✅
- Phase 2.3: Input Loop ✅
- Phase 3.1: Append-Only Printer ✅
- Phase 3.2: Output-Prompt Coordination ✅

**Not Required Yet:**
- Phase 4.1-4.3: Block system (optional for basic TUI)

---

## 11. Open Questions

**Q1:** Should `Run()` echo submitted lines automatically, or leave that to the caller?
**A:** Echo automatically in `handleSubmittedLine()` for Factory Droid feel (user sees input in transcript).

**Q2:** How to handle errors from `PrintLine`/`PrintChunks` in concurrent scenarios?
**A:** Return errors to caller; caller decides to log, retry, or fail.

**Q3:** Should `RequestInput()` return a new channel each call or the same channel?
**A:** Same channel (single source of truth). Cached in `u.inputs` field.

**Q4:** How to test SIGWINCH without real terminal?
**A:** Fake TTY provides `InjectResize(w, h)` that triggers `OnResize` callback.

---

## 12. Future Enhancements

### Phase 6.1: Block Timeline Integration
- Add `Timeline` field to `PureTTY`
- Render blocks in viewport above prompt
- Navigate blocks with keys

### Phase 7.4: Core Event Integration
- Map `core.Event` stream to UI operations
- `EventTypeStreamContent` → `PrintChunks()`
- `EventTypeSystemNotice` → `PrintLine()`

---

## 13. References

- **Spec:** [tui-new.md](../tui-implementation/tui-new.md) Section 6.1
- **Roadmap:** [ROADMAP.md](../tui-implementation/ROADMAP.md) Phase 5.1
- **Port Pattern:** [Architecture Overview](../architecture-overview.md)
- **Dependencies:**
  - [FRD-20251009-tui-terminal-control.md](./FRD-20251009-tui-terminal-control.md)
  - [FRD-20251010-keyboard-events.md](./FRD-20251010-keyboard-events.md)
  - [FRD-20251010-prompt-model.md](./FRD-20251010-prompt-model.md)
  - [FRD-20251010-prompt-renderer.md](./FRD-20251010-prompt-renderer.md)
  - [FRD-20251010-input-loop.md](./FRD-20251010-input-loop.md)
  - [FRD-20251010-append-only-printer.md](./FRD-20251010-append-only-printer.md)
  - [FRD-20251010-output-prompt-coordination.md](./FRD-20251010-output-prompt-coordination.md)

---

## 14. Sign-Off

**Author:** Claude (Spin Agent)
**Reviewers:** (pending)
**Approved:** (pending)
**Status:** Draft → Ready for Implementation

---

**Next Steps:**
1. Review FRD
2. Write tests (TDD)
3. Implement adapter
4. Analyze with `uast parse | herr analyze`
5. Run `make lint`
6. Fix any issues
7. Update ROADMAP.md
8. Update docs/
