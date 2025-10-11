# FRD-20251010: E2E TUI Tests

**Status:** In Progress
**Priority:** P0 (Critical path)
**Phase:** 7.1
**Author:** Spin
**Date:** 2025-10-10

## 1. Overview

Implement end-to-end tests for the full TUI lifecycle: startup, user input, output streaming, block rendering, navigation, and shutdown. Use hermetic test infrastructure (fake terminal, scripted input, captured output) to enable deterministic, fast, and reliable testing without real terminal dependencies.

## 2. Goals

* **Full lifecycle coverage**: Test complete user journeys from Run() to Stop()
* **Hermetic execution**: No real terminal I/O, no flaky timing issues
* **Deterministic results**: Reproducible ANSI output verification
* **Fast execution**: Complete E2E suite in <10s
* **Black-box testing**: Test through public UI interface (ports.UI)
* **High coverage**: ≥85% overall UI package coverage
* **Zero goroutine leaks**: Verify clean shutdown
* **Thread safety**: All tests pass with `-race`

## 3. Non-Goals

* Real terminal emulator testing (defer to manual QA Phase 8.2)
* Visual regression testing (screenshots)
* Performance benchmarking (defer to Phase 7.2)
* Integration with real LLM providers (stub/mock)
* Browser-based TUI testing

## 4. Requirements

### 4.1 Functional Requirements

**FR-7.1.1: Test Infrastructure**

* **FakeWriter**: Captures all ANSI bytes written to stdout
  * Buffer accumulation with thread-safe access
  * ANSI sequence parsing helpers (cursor pos, colors, content)
  * Snapshot helpers for golden tests

* **FakeKeyboard**: Injects scripted key sequences
  * Queue of KeyEvent with timing control
  * Blocking/non-blocking injection modes
  * Paste simulation (bracketed paste)

* **FakeTTY**: Simulates terminal dimensions and signals
  * Configurable width/height
  * SIGWINCH simulation (resize events)
  * Enter/Exit raw mode tracking
  * OnResize callback verification

**FR-7.1.2: E2E Test Scenarios**

1. **Basic Input/Output**
   * Start UI, send "hello", press Enter
   * Verify input appears on RequestInput() channel
   * Verify prompt redraws after submission

2. **Streaming Chunks**
   * Stream 100 chunks via PrintChunks()
   * Verify prompt stays at bottom (no torn output)
   * Verify all chunks appear in output

3. **Block Navigation**
   * Append 20 blocks to timeline
   * Simulate PgUp/PgDn navigation
   * Verify visible blocks change correctly
   * Simulate [/] block focus navigation
   * Verify focused block updates

4. **Block Collapse/Expand**
   * Append expanded block
   * Press Enter on block (toggle fold)
   * Verify block collapses (renders header only)
   * Press Enter again
   * Verify block expands

5. **Filtering**
   * Append mixed block types (EXECUTE, PLAN, ERROR)
   * Press `/` to enter filter mode
   * Type "type:execute"
   * Verify only EXECUTE blocks visible
   * Press Esc to clear filter
   * Verify all blocks visible again

6. **Terminal Resize**
   * Start UI with 80×24 terminal
   * Trigger resize to 120×40
   * Verify OnResize callback invoked
   * Verify renderer width updated
   * Verify redraw with new width

7. **Graceful Shutdown: Ctrl-C**
   * Start UI
   * Send Ctrl-C
   * Verify Run() exits cleanly
   * Verify Stop() called (TTY exits raw mode)
   * Verify no goroutine leaks

8. **Graceful Shutdown: Context Cancel**
   * Start UI with cancelable context
   * Cancel context
   * Verify Run() returns context.Canceled
   * Verify Stop() called
   * Verify no goroutine leaks

9. **Graceful Shutdown: Ctrl-D (EOF)**
   * Start UI
   * Send Ctrl-D on empty buffer
   * Verify Run() exits cleanly
   * Verify Stop() called

10. **Large Payload**
    * Stream 10,000 lines via PrintChunks()
    * Verify no goroutine hang
    * Verify complete output captured
    * Verify memory stable (no leaks)

11. **Concurrent Operations**
    * Concurrently: PrintLine(), stream chunks, inject keys, resize
    * Verify no race conditions (run with `-race`)
    * Verify no torn output
    * Verify no deadlocks

### 4.2 Non-Functional Requirements

**NFR-7.1.1: Performance**
* E2E test suite completes in <10s total
* Individual tests complete in <1s (except large payload: <3s)
* No flaky timing issues (use synchronization, not sleep)

**NFR-7.1.2: Determinism**
* All tests produce identical results on repeated runs
* No random seed variation
* No wall-clock time dependencies

**NFR-7.1.3: Isolation**
* Each test creates fresh UI instance
* No shared state between tests
* No environmental dependencies (real TTY, env vars)

**NFR-7.1.4: Maintainability**
* Test names read like requirements (e.g., "TestE2E_InputSubmit_PromptsRedraw")
* Clear setup/act/assert structure
* Reusable test helpers in testkit package
* Golden files for ANSI snapshot verification

## 5. Design

### 5.1 Package Structure

```
internal/ui/
  e2e_test.go              // E2E test scenarios
  testkit/
    fake_writer.go         // ANSI output capture
    fake_writer_test.go
    fake_keyboard.go       // Scripted key injection
    fake_keyboard_test.go
    fake_tty.go            // Terminal simulation
    fake_tty_test.go
    helpers.go             // Test utilities (wait helpers, assertions)
    helpers_test.go
```

### 5.2 API

#### 5.2.1 FakeWriter

```go
package testkit

import "bytes"

// FakeWriter captures ANSI output for testing
type FakeWriter struct {
    buf   *bytes.Buffer
    mu    sync.Mutex
    // Optional: parse ANSI sequences for structured assertions
}

func NewFakeWriter() *FakeWriter

// Write implements io.Writer
func (f *FakeWriter) Write(p []byte) (int, error)

// Snapshot returns current output as string
func (f *FakeWriter) Snapshot() string

// Reset clears buffer
func (f *FakeWriter) Reset()

// ContainsANSI checks if output contains ANSI sequence
func (f *FakeWriter) ContainsANSI(seq string) bool

// WaitForContent blocks until substring appears or timeout
func (f *FakeWriter) WaitForContent(s string, timeout time.Duration) bool

// Lines returns output split by newlines
func (f *FakeWriter) Lines() []string

// StripANSI returns output with ANSI codes removed
func (f *FakeWriter) StripANSI() string
```

#### 5.2.2 FakeKeyboard

```go
package testkit

import "github.com/dmytrogajewski/spin/internal/ui/term"

// FakeKeyboard injects scripted key events
type FakeKeyboard struct {
    events chan term.KeyEvent
    mu     sync.Mutex
}

func NewFakeKeyboard() *FakeKeyboard

// Events returns read channel for term.ReadKeys()
func (f *FakeKeyboard) Events() <-chan term.KeyEvent

// InjectKey queues a single key event
func (f *FakeKeyboard) InjectKey(kind term.KeyKind, r rune)

// InjectString queues key events for each rune in string
func (f *FakeKeyboard) InjectString(s string)

// InjectEnter queues KeyEnter
func (f *FakeKeyboard) InjectEnter()

// InjectCtrlC queues KeyCtrlC
func (f *FakeKeyboard) InjectCtrlC()

// InjectPaste queues bracketed paste sequence
func (f *FakeKeyboard) InjectPaste(text string)

// Close closes event channel (signals EOF)
func (f *FakeKeyboard) Close()
```

#### 5.2.3 FakeTTY

```go
package testkit

import "github.com/dmytrogajewski/spin/internal/ui/term"

// FakeTTY simulates terminal for testing
type FakeTTY struct {
    width, height int
    entered       bool
    exited        bool
    resizeCB      func(w, h int)
    mu            sync.Mutex
}

func NewFakeTTY(width, height int) *FakeTTY

// Enter simulates entering raw mode
func (f *FakeTTY) Enter() error

// Exit simulates exiting raw mode
func (f *FakeTTY) Exit() error

// Size returns terminal dimensions
func (f *FakeTTY) Size() (int, int)

// OnResize registers resize callback
func (f *FakeTTY) OnResize(cb func(w, h int))

// SetSize updates dimensions and triggers resize callback
func (f *FakeTTY) SetSize(w, h int)

// IsEntered checks if Enter() was called
func (f *FakeTTY) IsEntered() bool

// IsExited checks if Exit() was called
func (f *FakeTTY) IsExited() bool
```

### 5.3 Test Structure Pattern

Each E2E test follows this pattern:

```go
func TestE2E_ScenarioName(t *testing.T) {
    // Setup: Create fake components
    writer := testkit.NewFakeWriter()
    keyboard := testkit.NewFakeKeyboard()
    fakeTTY := testkit.NewFakeTTY(80, 24)

    // Create UI with fake components
    ui, err := adapters.NewPureTTY(writer,
        adapters.WithTTY(fakeTTY),
        adapters.WithKeyboard(keyboard),
    )
    require.NoError(t, err)

    // Act: Run UI in background
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()

    runErr := make(chan error, 1)
    go func() {
        runErr <- ui.Run(ctx)
    }()

    // Wait for startup
    testkit.WaitForStartup(t, ui, 100*time.Millisecond)

    // Perform actions
    keyboard.InjectString("hello")
    keyboard.InjectEnter()

    // Assert: Verify outputs
    line := testkit.WaitForInput(t, ui.RequestInput(), 200*time.Millisecond)
    assert.Equal(t, "hello", line)

    // Verify output contains expected content
    assert.True(t, writer.WaitForContent("hello", 200*time.Millisecond))

    // Cleanup: Shutdown
    cancel()
    err = testkit.WaitForShutdown(t, runErr, 1*time.Second)
    assert.ErrorIs(t, err, context.Canceled)
    assert.True(t, fakeTTY.IsExited())
}
```

### 5.4 Test Helpers

```go
package testkit

import "testing"

// WaitForStartup waits for UI to be ready
func WaitForStartup(t *testing.T, ui ports.UI, timeout time.Duration) {
    // Wait for TTY.Enter() to be called
}

// WaitForInput waits for line on input channel
func WaitForInput(t *testing.T, ch <-chan string, timeout time.Duration) string

// WaitForShutdown waits for Run() to exit
func WaitForShutdown(t *testing.T, errCh <-chan error, timeout time.Duration) error

// AssertNoGoroutineLeak verifies goroutine count stable
func AssertNoGoroutineLeak(t *testing.T, before, after int)

// AssertANSISequence checks for specific ANSI code
func AssertANSISequence(t *testing.T, output string, seq string)
```

## 6. Testing Strategy

### 6.1 Unit Tests for Testkit

Each fake component has its own unit tests:

**testkit/fake_writer_test.go:**
* TestWrite appends to buffer
* TestSnapshot returns current content
* TestReset clears buffer
* TestWaitForContent blocks/unblocks correctly
* TestContainsANSI detects sequences
* TestStripANSI removes escape codes
* TestConcurrentWrite (race detector)

**testkit/fake_keyboard_test.go:**
* TestInjectKey queues event
* TestInjectString queues multiple events
* TestInjectPaste generates bracketed paste
* TestClose closes channel
* TestConcurrentInject (race detector)

**testkit/fake_tty_test.go:**
* TestEnterExit updates flags
* TestSize returns dimensions
* TestSetSize triggers resize callback
* TestOnResize registers callback
* TestConcurrentResize (race detector)

### 6.2 E2E Tests

**internal/ui/e2e_test.go:**

```go
func TestE2E_InputSubmit_PromptsRedraw(t *testing.T)
func TestE2E_StreamingChunks_PromptAtBottom(t *testing.T)
func TestE2E_BlockNavigation_ScrollsCorrectly(t *testing.T)
func TestE2E_BlockToggleFold_CollapsesExpands(t *testing.T)
func TestE2E_FilterBlocks_ShowsOnlyMatching(t *testing.T)
func TestE2E_TerminalResize_RedrawsWithNewWidth(t *testing.T)
func TestE2E_ShutdownCtrlC_ExitsCleanly(t *testing.T)
func TestE2E_ShutdownContextCancel_ExitsCleanly(t *testing.T)
func TestE2E_ShutdownCtrlD_ExitsOnEOF(t *testing.T)
func TestE2E_LargePayload_StreamsWithoutHang(t *testing.T)
func TestE2E_ConcurrentOperations_NoRaceConditions(t *testing.T)
```

### 6.3 Integration Strategy

**Existing components tested:**
* `internal/ui/term` (Phase 1)
* `internal/ui/prompt` (Phase 2)
* `internal/ui/output` (Phase 3)
* `internal/ui/blocks` (Phase 4)
* `internal/ui/adapters` (Phase 5)
* `internal/ui/overlay` (Phase 6.2)

**E2E tests verify:**
* Integration of all components
* Real user workflows
* Race-free concurrency
* Clean resource management

### 6.4 Edge Cases

* Empty input submission
* Rapid key sequences (stress test)
* Resize during streaming
* Cancel during block navigation
* Concurrent PrintLine + PrintChunks
* Very long lines (>1000 chars)
* Unicode input (emoji, CJK)
* Bracketed paste of 100KB text

## 7. Acceptance Criteria

- [ ] All E2E tests pass with `-race`
- [ ] Coverage ≥85% overall for internal/ui package
- [ ] E2E suite completes in <10s
- [ ] `make lint` clean
- [ ] Complexity ≤15 for test helpers
- [ ] Zero flake: 100 consecutive runs pass
- [ ] No goroutine leaks (verified with runtime.NumGoroutine)
- [ ] Tests document user flows (descriptive names)
- [ ] Testkit package has ≥90% unit test coverage
- [ ] Golden files for ANSI snapshot tests (if applicable)

## 8. Dependencies

* `github.com/stretchr/testify/assert` (assertions)
* `github.com/stretchr/testify/require` (test failures)
* `internal/ui/term` (KeyEvent types)
* `internal/ui/ports` (UI interface)
* `internal/ui/adapters` (PureTTY implementation)

## 9. Risks & Mitigations

| Risk | Mitigation |
|------|------------|
| Timing-dependent flake | Use synchronization primitives (channels, waitgroups), not sleep |
| Goroutine leaks | Verify NumGoroutine before/after each test |
| ANSI parsing complexity | Use simple substring checks initially, defer complex parsing |
| Test maintenance burden | Extract reusable helpers in testkit |
| False positives | Golden tests with deterministic output |

## 10. Timeline

* Testkit infrastructure: 1 day
* E2E test scenarios (11 tests): 1.5 days
* Edge case coverage: 0.5 day
* Golden tests and polish: 0.5 day
* **Total:** 3.5 days

## 11. Future Enhancements

### Phase 7.2: Performance Benchmarks
* Add benchmark tests for throughput and latency
* Verify <16ms render time for 60fps

### Phase 8.2: Manual Terminal Testing
* Real terminal compatibility (xterm, kitty, alacritty, iTerm2, Windows Terminal)
* SSH session testing
* tmux/screen testing

### Optional: Visual Regression
* Snapshot testing with terminal recordings (asciinema)
* Diff rendering for visual changes

## 12. References

* [AGENTS.md](../../AGENTS.md) - E2E testing philosophy
* [TUI Spec](../tui-implementation/tui-new.md) - Feature specification
* [Roadmap](../tui-implementation/ROADMAP.md) - Phase 7.1
* [Existing Integration Tests](../../internal/ui/term/tty_integration_test.go) - PTY-based pattern
* [Existing Fake Patterns](../../internal/ui/prompt/loop_test.go) - FakeRenderer example

---

**Last Updated:** 2025-10-10
**Status:** In Progress
