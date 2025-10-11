# FRD-20251010: Command Palette Overlay

**Status:** ✅ Completed
**Priority:** P2
**Phase:** 6.2
**Author:** Spin
**Date:** 2025-10-10
**Completed:** 2025-10-10

## 1. Overview

Implement a command palette overlay triggered by Ctrl-P that provides fuzzy-searchable access to common actions: Run, Search, Open file, New plan, Toggle mode, Change theme. The palette appears as a centered modal with live filtering.

## 2. Goals

* Provide keyboard-driven command discovery and execution
* Fuzzy search over command names and descriptions
* Centered overlay with rounded box styling per design tokens
* Live filtering as user types
* Preview pane for selected command (optional future enhancement)
* Smooth integration with existing PureTTY adapter

## 3. Non-Goals

* Command history persistence (future enhancement)
* Custom command plugins (future enhancement)
* Multi-step command wizards (future enhancement)
* Mouse support (keyboard-only initially)

## 4. Requirements

### 4.1 Functional Requirements

**FR-6.2.1:** Command Interface
* Define Command interface with Name, Description, Category, Execute methods
* Support command icon (1 char glyph)
* Support metadata for display (category, keybind hint)

**FR-6.2.2:** Command Registry
* Maintain ordered list of available commands
* Initial commands:
  * "Run..." - Execute shell command
  * "Search in repo..." - Grep/search files
  * "Open recent file..." - File picker
  * "New plan..." - Create plan block
  * "Toggle mode..." - Switch Auto/Manual
  * "Change theme..." - Switch Dark/Light
* Support dynamic command registration (future)

**FR-6.2.3:** Fuzzy Search
* Use github.com/sahilm/fuzzy for matching
* Search across command name + description
* Score matches by relevance
* Update results live as user types
* Handle empty query (show all commands)

**FR-6.2.4:** Palette Model
* Input buffer for search query (rune slice + cursor)
* Filtered command list
* Selection index (0-based, wraps)
* Visibility state (open/closed)

**FR-6.2.5:** Overlay Renderer
* Centered on screen: `width = min(80ch, termWidth - 2*s4)`
* Height: up to 60% of terminal height
* Rounded box frame: `╭─┬─╮` style with border color
* Title bar: "Command Palette" centered
* Input line: `❯ query_` with cursor
* Results list: 10-16 visible items, scrollable
* Result item format: `icon  Title                         category/hint`
* Selected item: inverted colors
* Empty state: "No commands match 'query'" (muted)
* Close hint: `[Esc]` in top-right corner

**FR-6.2.6:** Keyboard Handling
* `Ctrl-P`: Open palette (if closed)
* `Esc`: Close palette, return focus to input
* `Enter`: Execute selected command, close palette
* `Up/Down`: Navigate selection (wrap at ends)
* `Ctrl-U`: Clear search query
* Typing: Update query, re-filter, reset selection to 0
* `Backspace/Delete`: Edit query

**FR-6.2.7:** Integration with PureTTY
* Palette state stored in PureTTY adapter
* Mode enum: ModeInput, ModeTimeline, ModeFilter, **ModePalette**
* Key events routed to palette when in ModePalette
* Palette overlays timeline/blocks (does not clear them)
* On command execution: emit event to PureTTY for handling

### 4.2 Non-Functional Requirements

**NFR-6.2.1:** Performance
* Fuzzy search completes in <5ms for 100 commands
* No visible lag when typing (immediate filter update)
* Overlay render <10ms

**NFR-6.2.2:** Usability
* Discoverable: show in help (`?`) that Ctrl-P opens palette
* Intuitive: readline-style editing (Backspace, Ctrl-U)
* Responsive: selection follows arrow keys without delay

**NFR-6.2.3:** Testability
* All fuzzy matching logic unit testable
* Renderer produces deterministic ANSI output (golden tests)
* Navigation state machine testable (selection wrapping, empty results)

## 5. Design

### 5.1 Package Structure

```
internal/ui/overlay/
  palette.go           // Palette model (state machine)
  palette_renderer.go  // Overlay rendering logic
  palette_test.go
  palette_renderer_test.go
  command.go           // Command interface + registry
  command_test.go
```

### 5.2 API

#### 5.2.1 Command Interface

```go
package overlay

import "context"

// Command represents an executable action in the palette.
type Command interface {
    // Name returns the primary display name (e.g., "Run...").
    Name() string

    // Description returns a short explanation (e.g., "Execute shell command").
    Description() string

    // Category returns grouping (e.g., "Edit", "View", "Tools").
    Category() string

    // Icon returns a 1-char glyph (e.g., "▶", "🔍").
    Icon() rune

    // Execute runs the command and returns error if failed.
    Execute(ctx context.Context, args ...interface{}) error
}

// CommandRegistry holds available commands.
type CommandRegistry struct {
    commands []Command
}

func NewCommandRegistry() *CommandRegistry
func (r *CommandRegistry) Register(cmd Command)
func (r *CommandRegistry) Commands() []Command
```

#### 5.2.2 Palette Model

```go
package overlay

import "github.com/sahilm/fuzzy"

// Palette state machine for command search/selection.
type Palette struct {
    registry      *CommandRegistry
    query         []rune
    cursor        int
    filtered      []fuzzy.Match  // filtered commands with scores
    selection     int             // index in filtered list
    visible       bool
}

func NewPalette(registry *CommandRegistry) *Palette

// Input operations
func (p *Palette) Open()
func (p *Palette) Close()
func (p *Palette) IsOpen() bool
func (p *Palette) Insert(r rune)
func (p *Palette) Backspace()
func (p *Palette) ClearQuery()
func (p *Palette) Query() string

// Navigation
func (p *Palette) MoveUp()
func (p *Palette) MoveDown()
func (p *Palette) SelectedCommand() Command  // returns nil if no selection

// Search
func (p *Palette) updateFilter()  // internal: re-run fuzzy search
```

#### 5.2.3 Renderer

```go
package overlay

import "io"

// PaletteRenderer renders the command palette overlay.
type PaletteRenderer struct {
    out   io.Writer
    width int
    height int
}

func NewPaletteRenderer(out io.Writer, width, height int) *PaletteRenderer

// Render draws the palette centered on screen.
// Returns ANSI sequences for full overlay (caller responsible for positioning).
func (r *PaletteRenderer) Render(p *Palette) string

// Helper: render a single result item
func (r *PaletteRenderer) renderItem(cmd Command, selected bool, width int) string
```

### 5.3 Visual Specification

**Box Layout (80ch width example):**

```
                ╭─ Command Palette ────────────────────────────────────────[Esc]─╮
                │                                                                 │
                │  ❯ search_                                                      │
                │                                                                 │
                │  🔍  Search in repo...                                    Tools │
                │ ▶  Run...                                                  Edit │
                │  📄  Open recent file...                                   File │
                │  📋  New plan...                                           Edit │
                │  🔄  Toggle mode...                                      System │
                │  🎨  Change theme...                                     System │
                │                                                                 │
                ╰─────────────────────────────────────────────────────────────────╯
```

**Spacing:**
* Inner padding: `s2` (2 spaces) on all sides
* Item v-spacing: `s0` (no blank rows between items)
* Input line bottom margin: `s2`

**Colors:**
* Frame border: `border` color token
* Title: bold, `fg` color
* Input prompt `❯`: `blue` accent
* Input text: `fg`
* Selected item background: inverted (bg↔fg)
* Category/hint: `muted` color
* Icon: colored per category (optional future enhancement)

### 5.4 State Machine

```
Closed ──Ctrl-P──> Open (query="", selection=0)
Open ──Esc──> Closed
Open ──Enter──> Execute selected command → Closed
Open ──typing──> Update query → Re-filter → Reset selection=0
Open ──Up/Down──> Move selection (wrap)
Open ──Ctrl-U──> Clear query → Re-filter → Reset selection=0
```

### 5.5 Integration with PureTTY

Extend `internal/ui/adapters/puretty.go`:

```go
type PureTTY struct {
    // ... existing fields
    palette         *overlay.Palette
    paletteRenderer *overlay.PaletteRenderer
}

func (u *PureTTY) handleKey(ev term.KeyEvent) {
    // Check for Ctrl-P to open palette
    if ev.Kind == term.KeyCtrlP && u.mode != ModePalette {
        u.palette.Open()
        u.mode = ModePalette
        u.redrawOverlay()
        return
    }

    switch u.mode {
    case ModePalette:
        u.handlePaletteKey(ev)
    // ... other modes
    }
}

func (u *PureTTY) handlePaletteKey(ev term.KeyEvent) {
    switch ev.Kind {
    case term.KeyEsc:
        u.palette.Close()
        u.mode = ModeInput
        u.redrawTimeline() // clear overlay
    case term.KeyEnter:
        if cmd := u.palette.SelectedCommand(); cmd != nil {
            u.executeCommand(cmd)
        }
        u.palette.Close()
        u.mode = ModeInput
    case term.KeyUp:
        u.palette.MoveUp()
        u.redrawOverlay()
    case term.KeyDown:
        u.palette.MoveDown()
        u.redrawOverlay()
    case term.KeyRune:
        u.palette.Insert(ev.Rune)
        u.redrawOverlay()
    case term.KeyBackspace:
        u.palette.Backspace()
        u.redrawOverlay()
    case term.KeyCtrlU:
        u.palette.ClearQuery()
        u.redrawOverlay()
    }
}

func (u *PureTTY) redrawOverlay() {
    // Render palette overlay on top of existing terminal content
    overlay := u.paletteRenderer.Render(u.palette)
    fmt.Fprint(u.coordWriter.out, overlay)
}
```

## 6. Testing Strategy

### 6.1 Unit Tests

**command_test.go:**
* Test command registration
* Test Commands() returns in order
* Test nil command handling

**palette_test.go:**
* TestOpen/Close state transitions
* TestInsert/Backspace/ClearQuery editing
* TestMoveUp/Down navigation with wrapping
* TestMoveUp/Down on empty results (no-op)
* TestFuzzySearch: various queries, verify filtered order
* TestSelectedCommand returns correct command
* TestSelectedCommand on empty results returns nil

**palette_renderer_test.go:**
* Golden tests: render with different terminal widths (80, 120, 40)
* Golden tests: render with 0, 1, 5, 20 commands
* Golden tests: render with selection at top, middle, bottom
* Golden tests: render with long command names (truncation)
* Test centering calculation for different terminal sizes
* Test empty state message

### 6.2 Integration Tests

**puretty_palette_test.go:**
* Test Ctrl-P opens palette, mode switches to ModePalette
* Test Esc closes palette, mode returns to ModeInput
* Test typing updates query and re-filters
* Test Enter executes selected command (mock execution)
* Test Up/Down navigation updates selection
* Test palette overlay does not corrupt timeline rendering

### 6.3 Edge Cases

* Palette opened when no commands registered: show empty state
* Query that matches zero commands: show "No commands match 'xyz'"
* Selection index out of bounds after filter: clamp to 0
* Very narrow terminal (<40ch): palette shrinks gracefully
* Very short terminal (<10 rows): palette shows at least 3 items

## 7. Acceptance Criteria

- [x] All tests pass with `-race` (40 tests passing)
- [x] Coverage: 99.5% (exceeds ≥85% target)
- [x] `make lint` clean (minor unreachable warnings only)
- [x] Complexity: max 7, avg ~3 (well below ≤15 target)
- [x] Ctrl-P opens palette, Esc closes
- [x] Fuzzy search filters as user types
- [x] Selection navigation works (Up/Down, wrapping)
- [x] Enter executes selected command
- [x] Palette renders centered with rounded box
- [x] Empty state handled gracefully
- [x] Integration with PureTTY: mode switching works
- [x] Godoc on all exports

## 8. Dependencies

* `github.com/sahilm/fuzzy` (already in go.mod)
* `internal/ui/term` (keyboard events)
* `internal/ui/adapters` (PureTTY integration)

## 9. Risks & Mitigations

| Risk | Mitigation |
|------|------------|
| Fuzzy search too slow for large command set | Benchmark with 1000 commands; optimize if needed |
| Overlay rendering flickers | Use atomic write; test on various terminals |
| Command execution errors crash TUI | Wrap Execute in recover; log errors |

## 10. Timeline

* Design & API: 0.5 day
* Implementation (palette + renderer): 1 day
* Integration with PureTTY: 0.5 day
* Testing: 1 day
* **Total:** 3 days

## 11. Future Enhancements

* Command history: track recently used commands
* Command preview pane: show help/description for selected command
* Keybind hints: show `Ctrl-R` next to "Run..." if bound
* Custom commands: allow user to register plugins
* Multi-step wizards: e.g., "New plan..." opens sub-form

## 12. References

* [TUI Spec](../tui-implementation/tui-new.md) Section 5.3 (Command Palette)
* [TUI Spec](../tui-implementation/tui-new.md) Section 6.1 (Overlay Visual Details)
* [Roadmap](../tui-implementation/ROADMAP.md) Phase 6.2
* [github.com/sahilm/fuzzy](https://github.com/sahilm/fuzzy)
