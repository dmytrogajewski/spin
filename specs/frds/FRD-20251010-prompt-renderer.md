# FRD-20251010-2: Prompt Renderer (One-Line Redraw)

## Metadata

- **ID**: FRD-20251010-prompt-renderer
- **Date**: 2025-10-10
- **Status**: Draft
- **Phase**: 2.2
- **Priority**: P0 (Critical path)
- **Complexity**: Medium
- **Related**: [ROADMAP.md](../tui-implementation/ROADMAP.md), [tui-new.md](../tui-implementation/tui-new.md), [FRD-20251010-prompt-model.md](./FRD-20251010-prompt-model.md)

## 1. Overview

Implement single-line prompt rendering with ANSI escape sequences. The renderer must correctly redraw the prompt line with prefix, buffer content, cursor positioning, and optional right-aligned status, while measuring grapheme widths correctly for accurate cursor placement.

This is a **pure rendering layer** with no state management (state lives in `prompt.Model`).

## 2. Goals

- Render prompt line efficiently using ANSI escape sequences
- Position cursor accurately accounting for wide characters and grapheme clusters
- Support optional right-aligned status text
- Implement horizontal scrolling for lines exceeding terminal width
- Handle zero-width characters (combining marks) correctly
- Provide testable, deterministic output for golden tests

## 3. Requirements

### 3.1 Functional Requirements

#### FR-1: Basic Rendering
- **FR-1.1**: Render format: `\r` + `ClearLine` + content + cursor positioning
- **FR-1.2**: Content structure: `prefix` + `buffer_text` + optional `status`
- **FR-1.3**: Prefix is configurable (default `"> "`)
- **FR-1.4**: All rendering to a single `io.Writer`

#### FR-2: Cursor Positioning
- **FR-2.1**: Use `rivo/uniseg.StringWidth()` for accurate width calculation
- **FR-2.2**: Cursor position = prefix_width + width_of_text_before_cursor
- **FR-2.3**: Use ANSI `MoveCursorToCol(n)` for positioning (1-indexed)
- **FR-2.4**: Handle wide characters (CJK, emoji) correctly (count as 2 cells)
- **FR-2.5**: Handle combining marks correctly (count as 0 cells)

#### FR-3: Right-Aligned Status
- **FR-3.1**: Status text rendered right-aligned if space available
- **FR-3.2**: Minimum gap between buffer and status: 3 cells
- **FR-3.3**: If insufficient space, status truncates from left with `…` ellipsis
- **FR-3.4**: If no space at all, status omitted entirely

#### FR-4: Horizontal Scrolling
- **FR-4.1**: If line exceeds terminal width, implement scroll window around cursor
- **FR-4.2**: Show left ellipsis `…` if content scrolled past start
- **FR-4.3**: Show right ellipsis `…` if content continues past end
- **FR-4.4**: Keep cursor in visible area (prefer center, but allow edges)
- **FR-4.5**: Scroll window width = `terminal_width - prefix_width - status_width - 2` (for ellipses)

#### FR-5: Edge Cases
- **FR-5.1**: Empty buffer renders prefix only
- **FR-5.2**: Cursor at end renders correctly (no trailing space)
- **FR-5.3**: Very long lines scroll correctly
- **FR-5.4**: Very narrow terminals (< 20 cols) degrade gracefully

### 3.2 Non-Functional Requirements

#### NFR-1: Performance
- **NFR-1.1**: Redraw operations O(n) where n = buffer length
- **NFR-1.2**: Width calculations use `uniseg.StringWidth()` (single pass)
- **NFR-1.3**: No allocations beyond necessary string building

#### NFR-2: Quality
- **NFR-2.1**: Coverage ≥90%
- **NFR-2.2**: All tests pass with `-race`
- **NFR-2.3**: Cyclomatic complexity ≤12 per function
- **NFR-2.4**: `make lint` clean, no dead code
- **NFR-2.5**: Godoc on all exports

#### NFR-3: Testability
- **NFR-3.1**: Deterministic output for given inputs
- **NFR-3.2**: Golden tests verify exact ANSI output
- **NFR-3.3**: No I/O beyond the provided writer

## 4. Design

### 4.1 Data Structure

```go
// Package prompt provides pure state management and rendering for prompt editing.
package prompt

import "io"

// Renderer renders a prompt model to a terminal using ANSI escape sequences.
// It handles cursor positioning, wide characters, and optional status text.
type Renderer struct {
    out    io.Writer // output destination
    width  int       // terminal width in cells
    prefix string    // prompt prefix (e.g., "> ")
}

// NewRenderer creates a new renderer with the specified output writer,
// terminal width, and prompt prefix.
func NewRenderer(out io.Writer, width int, prefix string) *Renderer

// Redraw renders the prompt model to the output.
// It emits: \r + ClearLine + prefix + buffer + cursor positioning + status.
func (r *Renderer) Redraw(model *Model, status string) error

// SetWidth updates the terminal width (call on SIGWINCH).
func (r *Renderer) SetWidth(width int)

// SetPrefix updates the prompt prefix.
func (r *Renderer) SetPrefix(prefix string)
```

### 4.2 Rendering Algorithm

```
1. Calculate widths:
   - prefix_width = StringWidth(prefix)
   - buffer_text = model.Text()
   - buffer_width = StringWidth(buffer_text)
   - cursor_col = prefix_width + StringWidth(buffer_text[:model.Cursor()])
   - status_width = StringWidth(status)

2. Determine visible content:
   - available_width = terminal_width - prefix_width
   - If buffer_width <= available_width:
       - visible_buffer = buffer_text (no scroll)
       - scroll_offset = 0
   - Else:
       - Calculate scroll window around cursor
       - visible_buffer = truncated with ellipses
       - Adjust cursor_col for scroll

3. Determine status rendering:
   - If status non-empty:
       - required_space = buffer_width + 3 + status_width
       - If required_space <= available_width:
           - Render right-aligned status
       - Else if available_width - buffer_width >= 3:
           - Truncate status from left
       - Else:
           - Omit status

4. Emit ANSI sequence:
   - Write: "\r" + ClearLine
   - Write: prefix
   - Write: visible_buffer
   - If status fits:
       - Calculate padding: terminal_width - prefix_width - buffer_width - status_width
       - Write: padding spaces + status
   - Write: MoveCursorToCol(cursor_col + 1)  // 1-indexed
```

### 4.3 Horizontal Scrolling Strategy

When `buffer_width > available_width`:
- Define `scroll_window_width = available_width - 2` (reserve for `…`)
- Calculate `cursor_offset_in_buffer` = visual width of `buffer_text[:cursor]`
- Center scroll window around cursor:
  - `scroll_start = max(0, cursor_offset_in_buffer - scroll_window_width/2)`
  - `scroll_end = scroll_start + scroll_window_width`
- Adjust for boundaries:
  - If `scroll_end > buffer_width`: shift window left
  - If `scroll_start == 0`: no left ellipsis
  - If `scroll_end >= buffer_width`: no right ellipsis
- Extract visible slice using grapheme boundaries (use `uniseg.GraphemeClusterWidth`)
- Prepend `…` if `scroll_start > 0`
- Append `…` if `scroll_end < buffer_width`
- Adjust `cursor_col` to account for scroll offset and ellipsis

### 4.4 Unicode Width Calculation

Use `github.com/rivo/uniseg`:
```go
import "github.com/rivo/uniseg"

// Calculate display width
width := uniseg.StringWidth(text)

// Iterate grapheme clusters
graphemes := uniseg.NewGraphemes(text)
for graphemes.Next() {
    cluster := graphemes.Str()
    clusterWidth := uniseg.StringWidth(cluster)
    // ...
}
```

## 5. Testing Strategy

### 5.1 Unit Tests

#### Renderer Tests (table-driven)

**Test Cases:**
1. **Empty buffer**: renders prefix only, cursor at position 0
2. **ASCII text**: simple buffer "hello", cursor at positions [0, 3, 5]
3. **Wide characters**: emoji "Hello 👋", CJK "你好世界"
4. **Combining marks**: "e\u0301" (é with combining acute)
5. **Cursor positioning**: verify exact ANSI cursor move for each position
6. **Right-aligned status**: buffer "test", status "typing", verify spacing
7. **Status truncation**: long status, narrow terminal, verify ellipsis
8. **Status omission**: very narrow terminal, status omitted
9. **Horizontal scroll**: buffer 200 chars, terminal 80, cursor at various positions
10. **Scroll ellipses**: verify `…` at correct positions
11. **Very narrow terminal**: width 20, verify degradation
12. **Very long line**: buffer 1000 chars, verify no panic

### 5.2 Golden Tests

Create expected ANSI output for each test case, compare bytes exactly.

Example:
```go
func TestRenderer_Redraw_Golden(t *testing.T) {
    tests := []struct {
        name       string
        prefix     string
        bufferText string
        cursor     int
        status     string
        width      int
        want       string
    }{
        {
            name:       "empty buffer",
            prefix:     "> ",
            bufferText: "",
            cursor:     0,
            status:     "",
            width:      80,
            want:       "\r\x1b[2K> \x1b[3G",
        },
        {
            name:       "simple text",
            prefix:     "> ",
            bufferText: "hello",
            cursor:     2,
            status:     "",
            width:      80,
            want:       "\r\x1b[2K> hello\x1b[5G",
        },
        // ... more cases
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            var buf bytes.Buffer
            r := NewRenderer(&buf, tt.width, tt.prefix)
            model := NewModel(100)
            model.SetText(tt.bufferText)
            model.SetCursor(tt.cursor)
            r.Redraw(model, tt.status)
            if got := buf.String(); got != tt.want {
                t.Errorf("Redraw() output mismatch\ngot:  %q\nwant: %q", got, tt.want)
            }
        })
    }
}
```

### 5.3 Edge Case Tests

- **Wide char at cursor**: emoji at cursor position
- **Combining mark at cursor**: "e\u0301" with cursor in middle
- **Zero-width terminal**: width 0 (panic protection)
- **Negative width**: should not panic
- **Nil writer**: should not panic (or return error)

### 5.4 Coverage Targets

- **Renderer**: ≥90%
- **Overall**: ≥90%

## 6. Implementation Phases

### Phase 1: Basic Renderer
1. Implement `Renderer` struct and constructor
2. Implement basic `Redraw()` with prefix + buffer
3. Write tests for simple ASCII cases

### Phase 2: Cursor Positioning
1. Implement accurate cursor positioning using `uniseg.StringWidth()`
2. Write tests for wide characters (emoji, CJK)
3. Write tests for combining marks

### Phase 3: Right-Aligned Status
1. Implement status rendering with padding calculation
2. Implement status truncation logic
3. Write tests for status rendering, truncation, omission

### Phase 4: Horizontal Scrolling
1. Implement scroll window calculation
2. Implement ellipsis insertion
3. Adjust cursor column for scroll offset
4. Write tests for long lines with scrolling

### Phase 5: Edge Cases & Polish
1. Handle very narrow terminals
2. Handle very long lines (1000+ chars)
3. Boundary tests (empty, cursor at 0, cursor at end)
4. Golden tests for all cases

## 7. Acceptance Criteria

### AC-1: Rendering Correctness
- ✅ Prompt renders correctly for empty buffer
- ✅ Prompt renders correctly for ASCII text
- ✅ Cursor positioned correctly for all cursor positions
- ✅ Wide characters (emoji, CJK) render with correct cursor
- ✅ Combining marks handled correctly (zero-width)

### AC-2: Status Rendering
- ✅ Right-aligned status renders when space available
- ✅ Status truncates with `…` when space limited
- ✅ Status omitted when insufficient space
- ✅ Minimum 3-cell gap between buffer and status

### AC-3: Horizontal Scrolling
- ✅ Long lines scroll correctly
- ✅ Left ellipsis `…` appears when scrolled past start
- ✅ Right ellipsis `…` appears when content continues
- ✅ Cursor stays in visible area
- ✅ Scroll window centers on cursor

### AC-4: Edge Cases
- ✅ Very narrow terminals degrade gracefully
- ✅ Very long lines don't panic
- ✅ Empty buffer, cursor at 0, cursor at end all work
- ✅ No panics on boundary conditions

### AC-5: Quality Gates
- ✅ All tests pass with `-race`
- ✅ Coverage ≥90%
- ✅ `make lint` clean
- ✅ Complexity ≤12 per function
- ✅ Godoc on all exports

## 8. Files to Create

```
internal/ui/prompt/
├── renderer.go       # Renderer implementation
├── renderer_test.go  # Renderer tests (golden tests)
└── doc.go            # Package documentation (update)
```

## 9. Dependencies

### External
- `github.com/rivo/uniseg` — grapheme cluster width calculation

### Internal
- `internal/ui/prompt` — Model, Buffer
- `internal/ui/term` — ANSI escape sequences (ClearLine, MoveCursorToCol)

## 10. Migration & Compatibility

N/A — New implementation, no migration needed.

## 11. Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| Cursor positioning broken by wide chars | High | Comprehensive tests with emoji, CJK, combining marks; use `uniseg.StringWidth()` |
| Horizontal scrolling complex | Medium | Clear algorithm, step-by-step tests, visual verification |
| Terminal width edge cases | Medium | Boundary tests for narrow (20), normal (80), wide (200) |
| ANSI escape sequence compatibility | Low | Use standard sequences, test on multiple terminals |

## 12. Open Questions

- **Q1**: Should we cache width calculations?
  - **A**: No. Premature optimization. Profile first if needed.

- **Q2**: How to handle right-to-left text (RTL)?
  - **A**: Out of scope. LTR only for now. API designed to allow future RTL support.

- **Q3**: Should status be styled (colored)?
  - **A**: Out of scope for this phase. API allows future styling (pass styled string).

## 13. References

- [Roadmap: Phase 2.2](../tui-implementation/ROADMAP.md#22-prompt-renderer-one-line-redraw)
- [TUI Spec](../tui-implementation/tui-new.md)
- [FRD-20251010-prompt-model.md](./FRD-20251010-prompt-model.md)
- [rivo/uniseg](https://github.com/rivo/uniseg) — Grapheme clusters and width
- [ANSI Escape Codes](https://en.wikipedia.org/wiki/ANSI_escape_code)
- [Effective Go](https://go.dev/doc/effective_go)

## 14. Success Metrics

- All 9 tasks from roadmap completed
- All acceptance criteria met
- Coverage ≥90%
- Zero flake in tests
- Complexity ≤12
- Golden tests verify exact output
- Ready for Phase 2.3 (Input Loop) integration

---

**Next Steps:**
1. Create `renderer.go` and `renderer_test.go`
2. Implement basic renderer with tests (TDD)
3. Add cursor positioning with Unicode handling
4. Add status rendering
5. Add horizontal scrolling
6. Run `uast parse | herr analyze`
7. Run `make lint`
8. Iterate to green
