# FRD-20251010: Keyboard Event System

**Status:** ✅ Completed (2025-10-10)
**Priority:** P0 (Critical path)
**Phase:** 1.2
**Author:** Spin
**Date:** 2025-10-10

## 1. Overview

Implement keyboard event parsing that translates raw terminal input into structured key events for the TUI. This system handles single-byte keys, multi-byte escape sequences, bracketed paste mode, and UTF-8 multi-byte runes with proper context cancellation support.

## 2. Goals

* Parse raw terminal input into structured `KeyEvent` objects
* Support navigation keys (arrows, Home, End, PgUp, PgDn)
* Support editing keys (Backspace, Delete, Ctrl-U, Ctrl-K, Ctrl-W)
* Handle bracketed paste mode for safe multi-line input
* Correctly decode UTF-8 multi-byte runes
* Provide context-aware cancellation for clean shutdown
* Handle ambiguous ESC sequences with timeout

## 3. Non-Goals

* Prompt model/state management (Phase 2.1)
* Prompt rendering (Phase 2.2)
* Mouse event support (may be added later)
* Full readline emulation (only specified keys)

## 4. Requirements

### 4.1 Functional Requirements

**FR-1.2.1:** Key Event Types
* Define `KeyKind` enum covering all required key types:
  * Printable runes (letters, digits, symbols, Unicode)
  * Enter, Tab
  * Backspace, Delete
  * Arrow keys (Up, Down, Left, Right)
  * Home, End
  * Page Up, Page Down
  * Control keys: Ctrl-C, Ctrl-D, Ctrl-U, Ctrl-K, Ctrl-W, Ctrl-L
  * Escape
  * Function keys F1-F12 (for future use)
* Each `KeyEvent` contains: kind, rune (if printable), raw bytes

**FR-1.2.2:** ESC Sequence Parsing
* Parse single ESC as `KeyEscape` after timeout (default 100ms)
* Parse multi-byte sequences:
  * `\x1b[A` → Up
  * `\x1b[B` → Down
  * `\x1b[C` → Right
  * `\x1b[D` → Left
  * `\x1b[H` → Home
  * `\x1b[F` → End
  * `\x1b[3~` → Delete
  * `\x1b[5~` → Page Up
  * `\x1b[6~` → Page Down
  * `\x1b[1;2A` → Shift+Up (for future)
  * Function keys: `\x1bOP` (F1), `\x1bOQ` (F2), etc.
* Handle partial/incomplete sequences gracefully

**FR-1.2.3:** Bracketed Paste Mode
* Detect paste start: `\x1b[200~`
* Accumulate all bytes until paste end: `\x1b[201~`
* Emit single `KeyPaste` event with complete payload
* Support multi-KB payloads (up to 1MB reasonable limit)
* Preserve exact bytes (including newlines, tabs, special chars)

**FR-1.2.4:** UTF-8 Handling
* Correctly decode UTF-8 sequences (1-4 bytes)
* Emit `KeyRune` for each complete rune
* Handle emoji, CJK characters, combining marks
* Return error for invalid UTF-8 sequences

**FR-1.2.5:** Control Characters
* Parse single-byte control chars:
  * `\x03` → Ctrl-C
  * `\x04` → Ctrl-D
  * `\x15` → Ctrl-U (kill line left)
  * `\x0b` → Ctrl-K (kill line right)
  * `\x17` → Ctrl-W (delete word left)
  * `\x0c` → Ctrl-L (redraw)
  * `\x7f` or `\x08` → Backspace
  * `\x0d` or `\x0a` → Enter
  * `\t` → Tab

**FR-1.2.6:** Context Cancellation
* `ReadKeys` accepts `context.Context`
* Stop reading immediately on context cancellation
* Return `context.Canceled` error
* Close output channel on exit

### 4.2 Non-Functional Requirements

**NFR-1.2.1:** Performance
* Parse keys with <1ms latency for single-byte keys
* ESC sequence timeout configurable (default 100ms)
* No blocking reads that prevent cancellation
* Minimal allocations in hot path (<10 allocs/key)

**NFR-1.2.2:** Reliability
* Handle partial ESC sequences (disconnection mid-sequence)
* Never hang on invalid input
* Gracefully handle EOF (terminal closed)
* No panics on malformed input

**NFR-1.2.3:** Compatibility
* Work with xterm, VT100, VT220 escape sequences
* Support modern terminals: kitty, alacritty, iTerm2, gnome-terminal
* Degrade gracefully on terminals without bracketed paste

## 5. Design

### 5.1 Package Structure

```
internal/ui/term/
  keyboard.go         // KeyKind, KeyEvent, ReadKeys
  keyboard_test.go    // Table-driven tests
```

### 5.2 API Design

```go
package term

import (
    "context"
    "io"
    "time"
)

// KeyKind identifies the type of key event.
type KeyKind int

const (
    KeyRune KeyKind = iota
    KeyEnter
    KeyTab
    KeyBackspace
    KeyDelete
    KeyUp
    KeyDown
    KeyLeft
    KeyRight
    KeyHome
    KeyEnd
    KeyPgUp
    KeyPgDn
    KeyEscape
    KeyCtrlC
    KeyCtrlD
    KeyCtrlU    // kill line left
    KeyCtrlK    // kill line right
    KeyCtrlW    // delete word left
    KeyCtrlL    // redraw
    KeyPaste    // bracketed paste
    KeyF1
    KeyF2
    KeyF3
    KeyF4
    KeyF5
    KeyF6
    KeyF7
    KeyF8
    KeyF9
    KeyF10
    KeyF11
    KeyF12
    KeyUnknown  // fallback for unrecognized sequences
)

// KeyEvent represents a single keyboard event.
type KeyEvent struct {
    Kind  KeyKind
    Rune  rune   // valid when Kind == KeyRune
    Paste []byte // valid when Kind == KeyPaste
    Raw   []byte // raw input bytes (for debugging)
}

// KeyReaderConfig configures the key reader.
type KeyReaderConfig struct {
    EscTimeout time.Duration // timeout for disambiguating ESC (default 100ms)
}

// ReadKeys reads keyboard events from r and sends them to the returned channel.
// It stops when ctx is canceled or r returns EOF.
// The channel is closed when the function exits.
func ReadKeys(ctx context.Context, r io.Reader, cfg *KeyReaderConfig) (<-chan KeyEvent, error)
```

### 5.3 Parsing State Machine

```
State: Initial
  ├─ Read byte
  ├─ 0x1b (ESC) → State: ESCStart (start timeout)
  ├─ 0x03 → Emit(KeyCtrlC)
  ├─ 0x04 → Emit(KeyCtrlD)
  ├─ 0x7f/0x08 → Emit(KeyBackspace)
  ├─ 0x0d/0x0a → Emit(KeyEnter)
  ├─ 0x09 → Emit(KeyTab)
  ├─ 0x15 → Emit(KeyCtrlU)
  ├─ 0x0b → Emit(KeyCtrlK)
  ├─ 0x17 → Emit(KeyCtrlW)
  ├─ 0x0c → Emit(KeyCtrlL)
  ├─ UTF-8 start → State: UTF8Decode
  └─ ASCII printable → Emit(KeyRune)

State: ESCStart (timeout active)
  ├─ Timeout → Emit(KeyEscape), State: Initial
  ├─ '[' → State: CSI
  ├─ 'O' → State: SS3
  └─ Other → Emit(KeyEscape), re-process byte

State: CSI (Control Sequence Introducer)
  ├─ 'A' → Emit(KeyUp)
  ├─ 'B' → Emit(KeyDown)
  ├─ 'C' → Emit(KeyRight)
  ├─ 'D' → Emit(KeyLeft)
  ├─ 'H' → Emit(KeyHome)
  ├─ 'F' → Emit(KeyEnd)
  ├─ '2' → State: CSITilde (expect '~' for Delete)
  ├─ '3' → State: CSITilde (expect '~' for Delete)
  ├─ '5' → State: CSITilde (expect '~' for PgUp)
  ├─ '6' → State: CSITilde (expect '~' for PgDn)
  ├─ '200~' → State: BracketedPaste
  └─ Other → Emit(KeyUnknown)

State: SS3 (Single Shift 3, for function keys)
  ├─ 'P' → Emit(KeyF1)
  ├─ 'Q' → Emit(KeyF2)
  ├─ 'R' → Emit(KeyF3)
  ├─ 'S' → Emit(KeyF4)
  └─ Other → Emit(KeyUnknown)

State: BracketedPaste
  ├─ Read bytes until '\x1b[201~'
  └─ Emit(KeyPaste) with accumulated payload

State: UTF8Decode
  ├─ Read continuation bytes
  ├─ Decode rune
  └─ Emit(KeyRune)
```

### 5.4 Bracketed Paste Implementation

```go
// Enable bracketed paste mode (caller's responsibility, via ANSI sequence)
const EnableBracketedPaste = "\x1b[?2004h"
const DisableBracketedPaste = "\x1b[?2004l"

// Inside ReadKeys, when detecting \x1b[200~:
// 1. Allocate buffer (start with 4KB, grow as needed)
// 2. Read bytes until \x1b[201~ sequence
// 3. Emit KeyPaste with buffer contents
// 4. Limit max buffer size to 1MB to prevent DoS
```

### 5.5 Timeout Handling for ESC

Use a timer to disambiguate ESC alone vs ESC sequence:

```go
escTimer := time.NewTimer(cfg.EscTimeout)
select {
case b := <-readByte:
    escTimer.Stop()
    // process next byte in sequence
case <-escTimer.C:
    // emit KeyEscape, return to initial state
case <-ctx.Done():
    return ctx.Err()
}
```

## 6. Testing Strategy

### 6.1 Unit Tests

**keyboard_test.go:**

* **TestKeyKindStringer**: Verify String() method for debugging
* **TestReadKeys_SingleByteKeys**: Table test for ASCII, Enter, Backspace, Tab
* **TestReadKeys_ControlKeys**: Ctrl-C, Ctrl-D, Ctrl-U, Ctrl-K, Ctrl-W, Ctrl-L
* **TestReadKeys_ArrowKeys**: Up, Down, Left, Right
* **TestReadKeys_HomeEnd**: Home, End
* **TestReadKeys_PageKeys**: PgUp, PgDn
* **TestReadKeys_Delete**: Delete key
* **TestReadKeys_FunctionKeys**: F1-F12
* **TestReadKeys_EscapeAlone**: ESC with timeout → KeyEscape
* **TestReadKeys_EscapeSequence**: ESC followed by '[A' → KeyUp (no timeout)
* **TestReadKeys_UTF8**: Multi-byte runes (€, 你好, 🚀, emoji)
* **TestReadKeys_InvalidUTF8**: Malformed sequences → error or skip
* **TestReadKeys_BracketedPaste**: Small payload (10 bytes), large payload (10KB), multi-line
* **TestReadKeys_PartialSequence**: ESC[ with EOF → emit unknown
* **TestReadKeys_ContextCancel**: Cancel mid-read → channel closed, ctx.Err returned
* **TestReadKeys_EOF**: Reader returns EOF → channel closed
* **TestReadKeys_RapidInput**: 1000 keys in quick succession

### 6.2 Edge Cases

* Empty input (EOF immediately)
* Single ESC byte then EOF
* Partial CSI sequence: `\x1b[` then EOF
* Invalid CSI sequence: `\x1b[X` (unknown)
* Bracketed paste without end marker (EOF or timeout)
* Very long paste (>1MB) → error or truncate
* Context cancel during bracketed paste
* Context cancel during UTF-8 decode
* Zero timeout for ESC (immediate KeyEscape)

### 6.3 Benchmark Tests

* BenchmarkReadKeys_SingleByte: throughput for ASCII keys
* BenchmarkReadKeys_ArrowKeys: throughput for escape sequences
* BenchmarkReadKeys_BracketedPaste: throughput for large paste
* BenchmarkReadKeys_UTF8: throughput for multi-byte runes

## 7. Acceptance Criteria

* [x] All tests pass with `-race` ✅
* [x] Coverage: keyboard.go functions 71-100% (avg ~85%) ✅
* [x] `make lint` clean ✅
* [x] Complexity ≤15 per function (max: 13, avg: 4.7) ✅
* [x] Handles all keys from spec (Enter, BS, Del, Left, Right, Up, Down, Ctrl-C, Ctrl-D, Ctrl-U, Ctrl-K, Ctrl-W, Ctrl-L) ✅
* [x] Bracketed paste correctly assembles multi-KB payloads ✅
* [x] No blocking reads prevent context cancellation ✅
* [x] ESC timeout works correctly (ESC alone vs ESC sequence) ✅
* [x] UTF-8 decoding works for emoji, CJK, combining marks ✅
* [x] Partial sequences handled gracefully (no panic, no hang) ✅

**Final Metrics:**
- **Tests:** 65 test cases covering all key types, sequences, edge cases
- **Coverage:** keyboard.go functions range from 71.4% (parseTildeCSI) to 100% (parseByte)
- **Complexity:** max 13 (parseByte), avg 4.69 (well below target of 15)
- **Race conditions:** Zero detected with `-race` flag
- **Lint errors:** Zero

## 8. Dependencies

* `io` - Reader interface
* `context` - Cancellation
* `time` - Timeout for ESC disambiguation
* `unicode/utf8` - UTF-8 decoding
* Existing: `internal/ui/term` package (TTY from Phase 1.1)

## 9. Risks & Mitigations

| Risk | Mitigation |
|------|------------|
| Blocking reads prevent cancellation | Use goroutine with select on context + read channel |
| ESC timeout too short (false ESC) | Make configurable, default 100ms (tested value) |
| Bracketed paste DoS (huge payload) | Limit buffer to 1MB, return error if exceeded |
| Invalid UTF-8 crashes parser | Use utf8.DecodeRune, handle errors gracefully |
| Terminal doesn't support bracketed paste | Graceful: user pastes appear as rapid key events (acceptable) |
| Race between timeout and next byte | Use timer.Stop() correctly, drain channel if needed |

## 10. Timeline

* FRD writing: 0.5 day
* Implementation: 1.5 days
* Testing: 1 day
* Review/polish: 0.5 day
* **Total:** 3.5 days

## 11. Future Enhancements

* Mouse event support (for terminal that support it)
* Shift/Alt/Ctrl modifiers on arrow keys (e.g., Shift+Up for selection)
* Paste length indicator during bracketed paste
* Custom key binding configuration
* Support for additional terminal types (tmux, screen special sequences)

## 12. References

* [XTerm Control Sequences](https://invisible-island.net/xterm/ctlseqs/ctlseqs.html)
* [VT100 User Guide](https://vt100.net/docs/vt100-ug/)
* [Bracketed Paste Mode](https://cirw.in/blog/bracketed-paste)
* [UTF-8 Encoding](https://en.wikipedia.org/wiki/UTF-8)
* Roadmap: [specs/tui-implementation/ROADMAP.md](../tui-implementation/ROADMAP.md) Phase 1.2
* TUI Spec: [specs/tui-implementation/tui-new.md](../tui-implementation/tui-new.md) Section 3.2, 4.2
* Previous FRD: [FRD-20251009-tui-terminal-control.md](./FRD-20251009-tui-terminal-control.md)

## 13. Implementation Notes

### 13.1 Read Loop Pattern

Use non-blocking select pattern to enable cancellation:

```go
func ReadKeys(ctx context.Context, r io.Reader, cfg *KeyReaderConfig) (<-chan KeyEvent, error) {
    if cfg == nil {
        cfg = &KeyReaderConfig{EscTimeout: 100 * time.Millisecond}
    }

    ch := make(chan KeyEvent, 16) // buffered for burst input
    go func() {
        defer close(ch)

        buf := make([]byte, 1)
        for {
            select {
            case <-ctx.Done():
                return
            default:
                // Read with timeout or use SetReadDeadline
                n, err := r.Read(buf)
                if err != nil {
                    return // EOF or error
                }
                if n > 0 {
                    event := parseKeyByte(buf[0], r, cfg, ctx)
                    select {
                    case ch <- event:
                    case <-ctx.Done():
                        return
                    }
                }
            }
        }
    }()
    return ch, nil
}
```

### 13.2 Buffering Strategy

* Use buffered channel (size 16) to handle burst input
* Parser holds minimal state (current sequence accumulator)
* Bracketed paste uses growable byte buffer (start 4KB)

### 13.3 Error Handling

* Invalid UTF-8: emit KeyUnknown with raw bytes
* Unknown ESC sequence: emit KeyUnknown with raw bytes
* Bracketed paste too large: emit error, truncate, or skip
* EOF during sequence: emit partial as KeyUnknown

## 14. Security Considerations

* **DoS via large paste:** Limit bracketed paste buffer to 1MB
* **Terminal escape injection:** Return raw bytes for unknown sequences (caller validates)
* **UTF-8 overlong encoding:** Use standard library's utf8.DecodeRune (handles this)
* **Control character injection:** Emit control chars as structured events (caller decides handling)

## 15. Observability

* Add debug logging (via slog if available):
  * Log unknown escape sequences (helps debug terminal compatibility)
  * Log bracketed paste size
  * Log context cancellation
* Metrics (optional):
  * Count of each KeyKind
  * Bracketed paste frequency and size distribution
  * Unknown sequence frequency

---

**Status:** Ready for Implementation
