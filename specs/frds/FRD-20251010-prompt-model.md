# FRD-20251010: Prompt Model (Buffer & History)

## Metadata

- **ID**: FRD-20251010-prompt-model
- **Date**: 2025-10-10
- **Status**: Draft
- **Phase**: 2.1
- **Priority**: P0 (Critical path)
- **Complexity**: Medium
- **Related**: [ROADMAP.md](../tui-implementation/ROADMAP.md), [tui-new.md](../tui-implementation/tui-new.md)

## 1. Overview

Implement the single source of truth for prompt state: a pure state machine managing rune buffer with cursor position, command history with navigation, and all editing operations. This is a **pure data structure with no I/O**.

## 2. Goals

- Provide a robust, well-tested buffer abstraction for the prompt subsystem
- Support all common readline/libedit editing operations
- Maintain command history with navigation and draft preservation
- Handle Unicode correctly (grapheme clusters, wide characters, combining marks)
- Zero I/O: pure state transformation suitable for testing and rendering

## 3. Requirements

### 3.1 Functional Requirements

#### FR-1: Buffer State Management
- **FR-1.1**: Buffer stores text as `[]rune` slice with cursor index (0-based)
- **FR-1.2**: Cursor position always valid: `0 <= cursor <= len(runes)`
- **FR-1.3**: Support empty buffer state (cursor at 0)
- **FR-1.4**: No panics on boundary conditions

#### FR-2: Editing Operations
- **FR-2.1**: `Insert(r rune)` — insert at cursor, advance cursor
- **FR-2.2**: `Backspace()` — delete before cursor if not at start
- **FR-2.3**: `Delete()` — delete at cursor if not at end
- **FR-2.4**: `MoveLeft()` — move cursor left by 1 grapheme cluster
- **FR-2.5**: `MoveRight()` — move cursor right by 1 grapheme cluster
- **FR-2.6**: `MoveStart()` — move cursor to position 0
- **FR-2.7**: `MoveEnd()` — move cursor to end (len(runes))

#### FR-3: Kill-Line Operations
- **FR-3.1**: `ClearLineLeft()` (Ctrl-U) — delete from start to cursor, cursor to 0
- **FR-3.2**: `ClearLineRight()` (Ctrl-K) — delete from cursor to end
- **FR-3.3**: `DeleteWord()` (Ctrl-W) — delete previous word using Unicode word boundaries

#### FR-4: Command History
- **FR-4.1**: History stores up to N entries (configurable, default 1000)
- **FR-4.2**: Oldest entry dropped when limit exceeded (ring buffer behavior)
- **FR-4.3**: `PrevHistory()` — navigate to previous command
- **FR-4.4**: `NextHistory()` — navigate to next command
- **FR-4.5**: Preserve current uncommitted buffer as "draft" when navigating
- **FR-4.6**: Restore draft when navigating past most recent entry
- **FR-4.7**: `Submit()` — push current buffer to history, reset buffer

#### FR-5: Unicode Handling
- **FR-5.1**: Use `rivo/uniseg` for grapheme cluster boundaries
- **FR-5.2**: Cursor navigation respects grapheme clusters (e.g., emoji, CJK)
- **FR-5.3**: Word boundaries use Unicode word break rules
- **FR-5.4**: Handle combining marks correctly (treat as part of base char)

### 3.2 Non-Functional Requirements

#### NFR-1: Performance
- **NFR-1.1**: Insert operation O(n) amortized (slice growth)
- **NFR-1.2**: Navigation operations O(1) or O(k) where k = grapheme width
- **NFR-1.3**: History navigation O(1)

#### NFR-2: Quality
- **NFR-2.1**: Coverage ≥95% (critical path)
- **NFR-2.2**: All tests pass with `-race`
- **NFR-2.3**: Cyclomatic complexity ≤12 per function
- **NFR-2.4**: `make lint` clean, no dead code
- **NFR-2.5**: Godoc on all exports

#### NFR-3: Reliability
- **NFR-3.1**: No panics on edge cases: empty buffer, cursor at boundaries
- **NFR-3.2**: Cursor validity maintained after every operation
- **NFR-3.3**: History navigation preserves uncommitted draft

## 4. Design

### 4.1 Data Structures

```go
// Package prompt provides pure state management for prompt editing.
package prompt

// Buffer represents the editable text buffer with cursor position.
type Buffer struct {
    runes  []rune // text content
    cursor int    // cursor position (0-based, 0 <= cursor <= len(runes))
}

// History manages command history with ring buffer.
type History struct {
    entries []string // historical commands
    limit   int      // max entries
    pos     int      // current navigation position (-1 = not navigating)
    draft   string   // uncommitted input when navigating
}

// Model combines buffer and history into a single state.
type Model struct {
    buffer  *Buffer
    history *History
}
```

### 4.2 Core Operations

#### Buffer Operations
```go
func (b *Buffer) Insert(r rune)              // insert at cursor
func (b *Buffer) Backspace() bool            // delete before cursor, return success
func (b *Buffer) Delete() bool               // delete at cursor, return success
func (b *Buffer) MoveLeft() bool             // move left by grapheme, return success
func (b *Buffer) MoveRight() bool            // move right by grapheme, return success
func (b *Buffer) MoveStart()                 // move to start
func (b *Buffer) MoveEnd()                   // move to end
func (b *Buffer) ClearLineLeft()             // Ctrl-U
func (b *Buffer) ClearLineRight()            // Ctrl-K
func (b *Buffer) DeleteWord()                // Ctrl-W
func (b *Buffer) String() string             // get text
func (b *Buffer) Cursor() int                // get cursor position
func (b *Buffer) Len() int                   // get text length
func (b *Buffer) Clear()                     // reset to empty
```

#### History Operations
```go
func NewHistory(limit int) *History
func (h *History) PrevHistory(currentDraft string) (string, bool)   // navigate back
func (h *History) NextHistory() (string, bool)                      // navigate forward
func (h *History) Submit(line string)                               // add to history
func (h *History) Reset()                                           // reset navigation
func (h *History) Entries() []string                                // get all entries (newest first)
```

#### Model Operations
```go
func NewModel(historyLimit int) *Model
func (m *Model) Insert(r rune)
func (m *Model) Backspace() bool
func (m *Model) Delete() bool
// ... (delegate to buffer)
func (m *Model) PrevHistory() bool           // load previous command into buffer
func (m *Model) NextHistory() bool           // load next command into buffer
func (m *Model) Submit() string              // push to history, return submitted text
func (m *Model) Text() string                // get current buffer text
func (m *Model) Cursor() int                 // get cursor position
```

### 4.3 Unicode Handling Strategy

Use `rivo/uniseg` library:
- `GraphemeClusterCount(s string)` — count clusters
- `Step(s string)` iterator — iterate clusters
- For cursor navigation, find cluster boundaries

Word deletion strategy:
- Use `unicode.IsSpace`, `unicode.IsLetter`, `unicode.IsDigit`
- Delete backwards until word boundary (space to non-space transition)

### 4.4 History Navigation State Machine

States:
- **NotNavigating**: `pos = -1`, draft empty
- **Navigating**: `pos >= 0`, draft holds original uncommitted input

Transitions:
1. **PrevHistory()** while NotNavigating:
   - Save current buffer as draft
   - Set `pos = len(entries) - 1`
   - Load `entries[pos]` into buffer
2. **PrevHistory()** while Navigating:
   - Decrement `pos` if `pos > 0`
   - Load `entries[pos]` into buffer
3. **NextHistory()** while Navigating:
   - Increment `pos` if `pos < len(entries) - 1`
   - Load `entries[pos]` into buffer
   - If `pos` reaches end, restore draft and reset to NotNavigating
4. **Submit()**:
   - Add current buffer to entries
   - Clear draft
   - Reset to NotNavigating

## 5. Testing Strategy

### 5.1 Unit Tests

#### Buffer Tests (table-driven)
- Insert: empty buffer, at start, middle, end, Unicode (emoji, CJK)
- Backspace: at start (no-op), at end, middle, empty buffer
- Delete: at end (no-op), at start, middle, empty buffer
- Navigation: left/right at boundaries, through multi-byte runes
- Kill-line: Ctrl-U/K/W with various cursor positions
- Edge cases: empty buffer, cursor at 0, cursor at len

#### History Tests (table-driven)
- PrevHistory: empty history, single entry, multiple entries
- NextHistory: at oldest, at newest, restore draft
- Submit: add to history, overflow limit (ring buffer)
- Navigation: full cycle (prev, prev, next, next)
- Draft preservation: edit buffer, navigate away, navigate back

#### Model Integration Tests
- Combine buffer and history operations
- Submit adds to history correctly
- History navigation loads buffer correctly

### 5.2 Property-Based Tests (optional stretch)
- Cursor invariant: always `0 <= cursor <= len(runes)`
- Insert/delete roundtrip: insert N, delete N → empty buffer
- History invariant: `len(entries) <= limit`

### 5.3 Coverage Targets
- Buffer: ≥95%
- History: ≥95%
- Model: ≥95%
- Overall: ≥95%

## 6. Implementation Phases

### Phase 1: Buffer Implementation
1. Implement `Buffer` struct and constructor
2. Implement basic edit operations (Insert, Backspace, Delete)
3. Implement navigation (MoveLeft, MoveRight, MoveStart, MoveEnd)
4. Write unit tests for buffer

### Phase 2: Kill-Line Operations
1. Implement ClearLineLeft/Right
2. Implement DeleteWord with Unicode boundaries
3. Write unit tests for kill-line ops

### Phase 3: History Implementation
1. Implement `History` struct with ring buffer
2. Implement navigation (Prev, Next)
3. Implement draft preservation
4. Write unit tests for history

### Phase 4: Model Integration
1. Implement `Model` combining buffer + history
2. Wire Submit() to history + buffer reset
3. Wire Prev/NextHistory to buffer load
4. Write integration tests

### Phase 5: Unicode & Edge Cases
1. Add grapheme cluster handling (rivo/uniseg)
2. Test emoji, CJK, combining marks
3. Test all boundary conditions
4. Property-based tests (optional)

## 7. Acceptance Criteria

### AC-1: Buffer Operations
- ✅ All buffer operations work correctly
- ✅ Cursor always valid after operations
- ✅ Unicode handling correct (emoji, CJK, combining marks)
- ✅ No panics on edge cases

### AC-2: History Navigation
- ✅ PrevHistory navigates backwards through history
- ✅ NextHistory navigates forwards through history
- ✅ Draft preserved when navigating away from uncommitted input
- ✅ Draft restored when navigating past newest entry
- ✅ Ring buffer drops oldest when limit exceeded

### AC-3: Quality Gates
- ✅ All tests pass with `-race`
- ✅ Coverage ≥95%
- ✅ `make lint` clean
- ✅ Complexity ≤12 per function
- ✅ Godoc on all exports

### AC-4: Integration
- ✅ Model correctly combines buffer + history
- ✅ Submit adds to history and clears buffer
- ✅ Navigation loads history entries into buffer

## 8. Files to Create

```
internal/ui/prompt/
├── buffer.go          # Buffer implementation
├── buffer_test.go     # Buffer tests
├── history.go         # History implementation
├── history_test.go    # History tests
├── model.go           # Model combining buffer + history
├── model_test.go      # Model integration tests
└── doc.go             # Package documentation
```

## 9. Dependencies

### External
- `github.com/rivo/uniseg` — grapheme cluster handling

### Internal
- None (pure state, no I/O)

## 10. Migration & Compatibility

N/A — New implementation, no migration needed.

## 11. Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| Unicode edge cases break cursor | High | Comprehensive Unicode test suite with emoji, CJK, combining marks |
| History navigation complexity | Medium | Clear state machine design, table-driven tests |
| Performance with large buffers | Low | Use []rune (not string concat), benchmark if needed |

## 12. Open Questions

- **Q1**: Should history persist across sessions?
  - **A**: Out of scope for this phase. API designed to support future persistence (expose Entries(), allow Load()).

- **Q2**: Max history size?
  - **A**: Default 1000, configurable via constructor.

- **Q3**: Word deletion boundary rules?
  - **A**: Use Unicode word break algorithm (space, punctuation boundaries).

## 13. References

- [Roadmap: Phase 2.1](../tui-implementation/ROADMAP.md#21-prompt-model-buffer--history)
- [TUI Spec](../tui-implementation/tui-new.md)
- [rivo/uniseg](https://github.com/rivo/uniseg) — Grapheme clusters
- [Effective Go](https://go.dev/doc/effective_go)
- [GNU Readline](https://tiswww.case.edu/php/chet/readline/rluserman.html) — UX reference

## 14. Success Metrics

- All 10 tasks from roadmap completed
- All acceptance criteria met
- Coverage ≥95%
- Zero flake in tests
- Complexity ≤12
- Ready for Phase 2.2 (Renderer) integration

---

**Next Steps:**
1. Create package structure
2. Implement Buffer with tests (TDD)
3. Implement History with tests (TDD)
4. Implement Model integration
5. Run `uast parse | herr analyze`
6. Run `make lint`
7. Iterate to green
