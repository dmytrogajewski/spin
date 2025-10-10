# FRD-20251010: Block Timeline UI Integration

**Phase:** 6.1
**Priority:** P1 (High priority)
**Complexity:** High
**Status:** Draft
**Created:** 2025-10-10
**Related Roadmap:** [ROADMAP.md](../tui-implementation/ROADMAP.md) Phase 6.1

---

## 1. Overview

Integrate block timeline rendering into PureTTY adapter. Extend the basic prompt-only TUI to render a scrollable timeline of blocks (EXECUTE, PLAN, READ, GREP, APPLY PATCH, SUMMARY, TESTING, NOTICE, ERROR) with navigation, filtering, and block actions.

This phase completes the transformation from basic text-based TUI to a rich block-based interface.

---

## 2. Goals

- **Timeline rendering** above prompt in viewport
- **Navigation** with keyboard (scroll, collapse/expand, focus)
- **Filtering** by block type, file, exit code, impact
- **Block actions**: copy, save, rerun, toggle fold
- **Event integration** with Core events (map events → blocks)
- **Performance**: smooth rendering with 1000+ blocks via virtualization

---

## 3. Non-Goals

- **Command palette** (deferred to Phase 6.2)
- **File preview popup** (deferred to Phase 6.3)
- **Theming system** (deferred to Phase 6.4)
- **Full Core integration** (deferred to Phase 7.4, this FRD implements adapter side only)
- **Approval UI** (deferred to Phase 7.4)

---

## 4. Requirements

### 4.1 Functional Requirements

**FR-1: Timeline State Management**
- PureTTY maintains `Timeline` instance
- Append blocks via `AppendBlock(block)`
- Update blocks via `UpdateBlock(blockID, block)`
- Delete blocks via `DeleteBlock(blockID)`

**FR-2: Viewport Rendering**
- Render visible blocks in viewport above prompt
- Viewport height = terminal height - (input bar + statusline + padding)
- Only visible blocks rendered (virtualization)
- Blocks render using `blocks.Renderer` per Phase 4.2

**FR-3: Navigation Keys**
- **Scroll**: `PgUp`/`PgDn`, `g`/`G` (top/bottom), `[`/`]` (prev/next block)
- **Focus**: Arrow keys (`↑`/`↓`) move focus between blocks
- **Collapse**: `Enter` toggles fold on focused block
- **Expand/Collapse All**: `zR`/`zM`

**FR-4: Block Actions**
- **Copy** (`y`): Copy focused block body to clipboard (if available)
- **Save** (`S`): Save focused block to file (prompt for filename)
- **Rerun** (`r`): Emit "rerun" event for EXECUTE blocks
- **Toggle Wrap** (`w`): Toggle line wrapping for block

**FR-5: Filtering**
- **Activate filter**: `/` enters filter mode
- **Filter syntax**: `type:EXECUTE file:foo.go exit:0 impact:high`
- **Clear filter**: `Esc` in filter mode
- **Filter UI**: Show active filter chips above timeline
- **Filter application**: Timeline renders only matching blocks

**FR-6: Mode Switching**
- **Timeline mode** (default): Keys navigate blocks
- **Input mode**: Keys go to prompt (triggered by typing)
- **Filter mode**: Keys edit filter string
- Mode indicator in statusline

**FR-7: Scroll Indicator**
- Show scroll position: `[Block 5/23] 21%`
- Show when timeline larger than viewport
- Position: right side of statusline

**FR-8: Event-Driven Block Creation**
- Listen to application events (not Core directly)
- Map events to block operations:
  - `AppendBlockEvent` → `timeline.Append(block)`
  - `UpdateBlockEvent` → `timeline.Update(id, block)`
  - `DeleteBlockEvent` → `timeline.Delete(id)`

---

### 4.2 Non-Functional Requirements

**NFR-1: Performance**
- Render 1000+ blocks without lag
- Viewport calculation <5ms (via Timeline's O(1) algorithm)
- Redraw only on state change (no flicker)

**NFR-2: Thread Safety**
- All timeline operations thread-safe
- Concurrent block append + navigation
- No data races (verified with `-race`)

**NFR-3: Testability**
- Unit tests with fake timeline
- Integration tests with fake keyboard events
- Navigation tests: scroll through 100 blocks

**NFR-4: Complexity**
- Keep per-function complexity ≤15
- Main event dispatcher may be higher (simple switch)

---

## 5. Design

### 5.1 Extended PureTTY Structure

```go
// internal/ui/adapters/puretty.go
type PureTTY struct {
	tty      *term.TTY
	model    *prompt.Model
	renderer *prompt.Renderer
	coord    *output.CoordinatedWriter
	out      io.Writer
	inputs   <-chan string

	// NEW: Timeline and block rendering
	timeline       *blocks.Timeline
	blockRenderer  *blocks.Renderer
	viewportHeight int
	mode           UIMode // timeline, input, filter
	filterInput    string

	mu      sync.Mutex
	running bool
	stopped bool
	cancel  context.CancelFunc
}

type UIMode int

const (
	ModeTimeline UIMode = iota // Navigate blocks
	ModeInput                  // Edit prompt
	ModeFilter                 // Edit filter
)
```

---

### 5.2 Constructor Updates

```go
// NewPureTTY creates adapter with timeline support.
func NewPureTTY(out io.Writer, opts ...PureTTYOption) (*PureTTY, error) {
	p := &PureTTY{
		out:  out,
		mode: ModeInput, // Start in input mode (for backward compat)
	}

	// ... existing setup ...

	// NEW: Create timeline and block renderer
	if p.timeline == nil {
		p.timeline = blocks.NewTimeline()
	}

	if p.blockRenderer == nil {
		w, _ := p.tty.Size()
		p.blockRenderer = blocks.NewRenderer(w)
	}

	return p, nil
}
```

**New Options**:
```go
func WithTimeline(timeline *blocks.Timeline) PureTTYOption
func WithBlockRenderer(r *blocks.Renderer) PureTTYOption
```

---

### 5.3 Viewport Calculation

```go
// calculateViewport computes visible block range.
func (u *PureTTY) calculateViewport() {
	_, h := u.tty.Size()

	// Reserve space for UI elements
	inputBarHeight := 2   // 2 rows (mode line + prompt)
	statusLineHeight := 1 // 1 row
	padding := 2          // Top + bottom padding

	u.viewportHeight = h - inputBarHeight - statusLineHeight - padding

	// Update timeline viewport
	u.timeline.SetViewportHeight(u.viewportHeight)
}
```

---

### 5.4 Rendering Loop

```go
// render redraws the entire UI: timeline + prompt.
func (u *PureTTY) render() {
	u.mu.Lock()
	defer u.mu.Unlock()

	// Clear previous render (optional: optimize with dirty tracking)
	// For now: append-only, so no clearing needed

	// Render visible blocks
	visible := u.timeline.GetVisibleBlocks()
	for _, block := range visible {
		rendered := u.blockRenderer.Render(block)
		// Write directly to output (bypass coordinator for timeline)
		fmt.Fprint(u.out, rendered)
	}

	// Render filter UI if active
	if u.mode == ModeFilter || u.timeline.GetFilter() != nil {
		u.renderFilterUI()
	}

	// Render prompt (via coordinator)
	u.coord.RedrawPrompt()
}
```

**Performance Note**: Initial implementation renders all visible blocks on each update. Future optimization: dirty tracking to redraw only changed blocks.

---

### 5.5 Key Handling (Extended)

```go
// handleKey dispatches key events based on current mode.
func (u *PureTTY) handleKey(key term.KeyEvent) {
	switch u.mode {
	case ModeTimeline:
		u.handleTimelineKey(key)
	case ModeInput:
		u.handleInputKey(key)
	case ModeFilter:
		u.handleFilterKey(key)
	}
}

// handleTimelineKey handles navigation and block actions.
func (u *PureTTY) handleTimelineKey(key term.KeyEvent) {
	switch key.Kind {
	case term.KeyPgUp:
		u.timeline.ScrollUp(u.viewportHeight)
		u.render()
	case term.KeyPgDn:
		u.timeline.ScrollDown(u.viewportHeight)
		u.render()
	case term.KeyRune:
		switch key.Rune {
		case 'g':
			u.timeline.ScrollToTop()
			u.render()
		case 'G':
			u.timeline.ScrollToBottom()
			u.render()
		case '[':
			u.timeline.PrevBlock()
			u.render()
		case ']':
			u.timeline.NextBlock()
			u.render()
		case 'y':
			u.handleCopyBlock()
		case 'S':
			u.handleSaveBlock()
		case 'r':
			u.handleRerunBlock()
		case 'w':
			u.handleToggleWrap()
		case '/':
			u.enterFilterMode()
		case ':':
			// Switch to input mode (for commands)
			u.mode = ModeInput
			u.render()
		default:
			// Any other char: switch to input mode, insert char
			u.mode = ModeInput
			u.model.Insert(key.Rune)
			u.render()
		}
	case term.KeyEnter:
		// Toggle fold on focused block
		focused := u.timeline.GetFocusedBlock()
		if focused != nil {
			u.timeline.ToggleFold(focused.ID)
			u.render()
		}
	case term.KeyEsc:
		// Exit timeline mode → input mode
		u.mode = ModeInput
		u.render()
	}
}

// handleInputKey delegates to existing prompt logic.
func (u *PureTTY) handleInputKey(key term.KeyEvent) {
	// Existing prompt logic from Phase 2.3
	// NEW: Esc switches to timeline mode
	if key.Kind == term.KeyEsc {
		u.mode = ModeTimeline
		u.render()
		return
	}

	// ... rest of prompt key handling ...
}

// handleFilterKey handles filter input editing.
func (u *PureTTY) handleFilterKey(key term.KeyEvent) {
	switch key.Kind {
	case term.KeyEsc:
		// Clear filter, return to timeline mode
		u.filterInput = ""
		u.timeline.SetFilter(nil)
		u.mode = ModeTimeline
		u.render()
	case term.KeyEnter:
		// Apply filter, return to timeline mode
		filter := u.parseFilter(u.filterInput)
		u.timeline.SetFilter(filter)
		u.mode = ModeTimeline
		u.render()
	case term.KeyBackspace:
		if len(u.filterInput) > 0 {
			u.filterInput = u.filterInput[:len(u.filterInput)-1]
			u.render()
		}
	case term.KeyRune:
		u.filterInput += string(key.Rune)
		u.render()
	}
}
```

---

### 5.6 Filter Parsing

```go
// parseFilter parses filter string into blocks.Filter.
// Syntax: "type:EXECUTE file:foo.go exit:0 impact:high"
func (u *PureTTY) parseFilter(input string) *blocks.Filter {
	filter := &blocks.Filter{}

	parts := strings.Fields(input)
	for _, part := range parts {
		kv := strings.SplitN(part, ":", 2)
		if len(kv) != 2 {
			continue
		}

		key, val := kv[0], kv[1]
		switch key {
		case "type":
			filter.Types = append(filter.Types, blocks.BlockType(val))
		case "file":
			filter.File = val
		case "exit":
			if code, err := strconv.Atoi(val); err == nil {
				filter.ExitCode = &code
			}
		case "impact":
			filter.Impact = val
		}
	}

	return filter
}
```

---

### 5.7 Block Actions

```go
// handleCopyBlock copies focused block body to clipboard.
func (u *PureTTY) handleCopyBlock() {
	block := u.timeline.GetFocusedBlock()
	if block == nil {
		return
	}

	// Copy to clipboard (platform-specific, use external cmd)
	// For now: just log (clipboard support is optional)
	u.coord.PrintLine(fmt.Sprintf("[Copied block %s]", block.ID))
}

// handleSaveBlock saves focused block to file.
func (u *PureTTY) handleSaveBlock() {
	block := u.timeline.GetFocusedBlock()
	if block == nil {
		return
	}

	// Prompt for filename (simple implementation: use block ID)
	filename := fmt.Sprintf("block_%s.txt", block.ID)

	// Write body to file
	if err := os.WriteFile(filename, []byte(block.Body), 0644); err != nil {
		u.coord.PrintLine(fmt.Sprintf("[Error saving: %v]", err))
		return
	}

	u.coord.PrintLine(fmt.Sprintf("[Saved to %s]", filename))
}

// handleRerunBlock emits rerun event for EXECUTE blocks.
func (u *PureTTY) handleRerunBlock() {
	block := u.timeline.GetFocusedBlock()
	if block == nil || block.Type != blocks.BlockTypeExecute {
		return
	}

	// Emit rerun event (to be consumed by application)
	// For now: just log
	u.coord.PrintLine(fmt.Sprintf("[Rerun requested for block %s]", block.ID))
}

// handleToggleWrap toggles line wrapping for block.
func (u *PureTTY) handleToggleWrap() {
	// TODO: Implement wrap toggle (requires renderer state)
	u.coord.PrintLine("[Toggle wrap not implemented yet]")
}
```

---

### 5.8 Filter UI Rendering

```go
// renderFilterUI renders filter input or active filter chips.
func (u *PureTTY) renderFilterUI() {
	if u.mode == ModeFilter {
		// Show filter input line
		fmt.Fprintf(u.out, "\r%s/ %s%s\r\n",
			term.ClearLine,
			u.filterInput,
			term.ShowCursor,
		)
	} else if f := u.timeline.GetFilter(); f != nil {
		// Show active filter chips
		chips := u.formatFilterChips(f)
		fmt.Fprintf(u.out, "\r%sFilter: %s%s\r\n",
			term.ClearLine,
			chips,
			term.HideCursor,
		)
	}
}

// formatFilterChips formats active filter as colored chips.
func (u *PureTTY) formatFilterChips(f *blocks.Filter) string {
	var chips []string

	for _, typ := range f.Types {
		chips = append(chips, fmt.Sprintf("[type:%s]", typ))
	}
	if f.File != "" {
		chips = append(chips, fmt.Sprintf("[file:%s]", f.File))
	}
	if f.ExitCode != nil {
		chips = append(chips, fmt.Sprintf("[exit:%d]", *f.ExitCode))
	}
	if f.Impact != "" {
		chips = append(chips, fmt.Sprintf("[impact:%s]", f.Impact))
	}

	return strings.Join(chips, " ")
}
```

---

### 5.9 Public API Extensions

```go
// AppendBlock appends a new block to timeline and re-renders.
func (u *PureTTY) AppendBlock(block *blocks.Block) error {
	u.mu.Lock()
	defer u.mu.Unlock()

	u.timeline.Append(block)
	u.render()
	return nil
}

// UpdateBlock updates an existing block and re-renders.
func (u *PureTTY) UpdateBlock(blockID string, block *blocks.Block) error {
	u.mu.Lock()
	defer u.mu.Unlock()

	u.timeline.Update(blockID, block)
	u.render()
	return nil
}

// DeleteBlock deletes a block and re-renders.
func (u *PureTTY) DeleteBlock(blockID string) error {
	u.mu.Lock()
	defer u.mu.Unlock()

	u.timeline.Delete(blockID)
	u.render()
	return nil
}

// SetMode switches UI mode (for testing or external control).
func (u *PureTTY) SetMode(mode UIMode) {
	u.mu.Lock()
	defer u.mu.Unlock()

	u.mode = mode
	u.render()
}
```

---

### 5.10 Updated UI Port Interface

```go
// internal/ui/ports/ui.go
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

	// NEW: Block timeline operations
	AppendBlock(block *blocks.Block) error
	UpdateBlock(blockID string, block *blocks.Block) error
	DeleteBlock(blockID string) error
}
```

---

## 6. Implementation Plan

### 6.1 Phase 1: Extend PureTTY Structure
1. Add `timeline`, `blockRenderer`, `viewportHeight`, `mode`, `filterInput` fields
2. Update constructor to create timeline and block renderer
3. Add new options: `WithTimeline`, `WithBlockRenderer`

### 6.2 Phase 2: Viewport Calculation
1. Implement `calculateViewport()`
2. Call on startup and resize
3. Update `handleResize()` to recalculate viewport

### 6.3 Phase 3: Rendering Logic
1. Implement `render()` that renders timeline + prompt
2. Implement `renderFilterUI()` for filter display
3. Update `Run()` to call `render()` after initial setup

### 6.4 Phase 4: Key Handling
1. Implement `handleKey()` mode dispatcher
2. Implement `handleTimelineKey()` for navigation
3. Update `handleInputKey()` to support mode switching
4. Implement `handleFilterKey()` for filter editing

### 6.5 Phase 5: Filter System
1. Implement `parseFilter()` string parser
2. Implement `formatFilterChips()` for UI display
3. Add `enterFilterMode()` helper

### 6.6 Phase 6: Block Actions
1. Implement `handleCopyBlock()` (stub with log)
2. Implement `handleSaveBlock()` (write to file)
3. Implement `handleRerunBlock()` (emit event)
4. Implement `handleToggleWrap()` (stub)

### 6.7 Phase 7: Public API
1. Implement `AppendBlock()`
2. Implement `UpdateBlock()`
3. Implement `DeleteBlock()`
4. Implement `SetMode()` for testing

### 6.8 Phase 8: Update UI Port Interface
1. Add block methods to `ports.UI`
2. Verify `PureTTY` satisfies extended interface

### 6.9 Phase 9: Tests
1. Unit tests for viewport calculation
2. Unit tests for filter parsing
3. Unit tests for key dispatching
4. Integration tests: navigate 100 blocks
5. Integration tests: filter by type/file/exit
6. Integration tests: block actions (copy, save, rerun)
7. Race detection tests

---

## 7. Testing Strategy

### 7.1 Unit Tests

**Test Cases**:

1. **TestCalculateViewport**: Verify viewport height calculation
2. **TestParseFilter**: Filter string parsing
   - `"type:EXECUTE"` → `Filter{Types: [EXECUTE]}`
   - `"file:foo.go exit:0"` → `Filter{File: "foo.go", ExitCode: &0}`
3. **TestFormatFilterChips**: Filter → chip string
4. **TestHandleTimelineKey**: Navigation keys
   - `PgUp` scrolls up
   - `g` scrolls to top
   - `[` focuses prev block
5. **TestHandleFilterKey**: Filter editing
   - Typing builds filter string
   - `Enter` applies filter
   - `Esc` clears filter
6. **TestModeSwitch**: Mode transitions
   - `Esc` in input mode → timeline mode
   - `/` in timeline mode → filter mode
   - Any char in timeline mode → input mode
7. **TestAppendBlock**: Block append + render
8. **TestUpdateBlock**: Block update + render
9. **TestDeleteBlock**: Block delete + render

---

### 7.2 Integration Tests

**Test Scenarios**:

1. **TestNavigate100Blocks**:
   - Append 100 blocks
   - Scroll with `PgDn` through all
   - Verify viewport shows correct blocks

2. **TestFilterByType**:
   - Append mixed blocks (EXECUTE, READ, PLAN)
   - Filter `/type:EXECUTE`
   - Verify only EXECUTE blocks visible

3. **TestFilterByFile**:
   - Append blocks with different files
   - Filter `/file:foo.go`
   - Verify only blocks matching file visible

4. **TestBlockActions**:
   - Focus block, press `y` → verify copy logged
   - Press `S` → verify file created
   - Press `r` → verify rerun logged

5. **TestResize**:
   - Start with 80x24
   - Resize to 120x40
   - Verify viewport recalculated, blocks re-rendered

6. **TestConcurrentBlockAppend**:
   - Goroutine appending blocks
   - Goroutine navigating timeline
   - Verify no races, no corruption

---

### 7.3 Performance Tests

**Benchmarks**:

1. **BenchmarkRender1000Blocks**:
   - 1000 blocks in timeline
   - Measure `render()` time
   - Target: <50ms (for 20fps)

2. **BenchmarkScroll1000Blocks**:
   - Scroll through 1000 blocks
   - Measure `ScrollDown()` + `GetVisibleBlocks()` time
   - Target: <5ms per scroll

---

## 8. Acceptance Criteria

- [ ] All tests pass with `-race`
- [ ] Coverage ≥85%
- [ ] `make lint` clean
- [ ] Complexity ≤15 per function (except event dispatchers)
- [ ] All navigation keys work (PgUp/PgDn, g/G, [/], Enter, zR/zM)
- [ ] Filter syntax parsed correctly
- [ ] Filter applied instantly, shows chip UI
- [ ] Block actions work (copy logs, save writes file, rerun logs)
- [ ] Mode switching: timeline ↔ input ↔ filter
- [ ] Scroll indicator shows position
- [ ] 1000 blocks render smoothly (<50ms per frame)
- [ ] No flicker during navigation
- [ ] Thread-safe concurrent operations

---

## 9. Quality Gates

**Tests**:
- ✅ All passing with `-race`
- ✅ Coverage ≥85%
- ✅ Benchmarks meet perf targets

**Code**:
- ✅ `make lint` passes (zero errors)
- ✅ Complexity ≤15 (average), ≤25 (max for dispatchers)
- ✅ No dead code
- ✅ Godoc on all exports

---

## 10. Dependencies

**Completed**:
- Phase 4.1: Block Types and Data Model ✅
- Phase 4.2: Block Rendering Rules ✅
- Phase 4.3: Block Timeline State Machine ✅
- Phase 5.1: PureTTY Adapter (basic) ✅

**Blocks**:
- None (all dependencies satisfied)

---

## 11. Open Questions

**Q1**: Should timeline mode be default, or input mode?
**A**: Start with **input mode** for backward compatibility. User can switch with Esc.

**Q2**: How to handle clipboard access (for copy action)?
**A**: Initial implementation: log only. Future enhancement: platform-specific clipboard (xclip, pbcopy, clip.exe).

**Q3**: Should filter support regex or just exact match?
**A**: Initial: exact/prefix match. Future: regex support via `~` prefix (e.g., `file:~.*\.go$`).

**Q4**: How to notify application of rerun events?
**A**: Deferred to Phase 7.4. For now: log message. Future: event channel or callback.

**Q5**: Should blocks auto-scroll to bottom (follow mode)?
**A**: Yes, when new block appended and timeline already at bottom. Add `ScrollToBottom()` in `AppendBlock()` if scroll position is at max.

---

## 12. Future Enhancements

### Phase 6.2: Command Palette
- `Ctrl-P` opens fuzzy command search
- Quick jump to blocks by ID/type

### Phase 6.3: File Preview
- `o` on filename:line anchors opens preview popup

### Phase 6.4: Theming
- Dark, Light, High-contrast themes
- Configurable via config file

### Phase 7.4: Core Event Integration
- Map `core.EventTypeToolCallStart` → `AppendBlock(EXECUTE)`
- Map `core.EventTypeToolCallProgress` → `UpdateBlock(stream)`
- Map `core.EventTypeStreamContent` → text accumulation
- Add approval UI for `EventCommandApproval`

---

## 13. References

- **Spec**: [tui-new.md](../tui-implementation/tui-new.md) Sections 5, 6.1, 7, 14
- **Roadmap**: [ROADMAP.md](../tui-implementation/ROADMAP.md) Phase 6.1
- **Dependencies**:
  - [FRD-20251010-block-types-data-model.md](./FRD-20251010-block-types-data-model.md)
  - [FRD-20251010-block-rendering-rules.md](./FRD-20251010-block-rendering-rules.md)
  - [FRD-20251010-block-timeline.md](./FRD-20251010-block-timeline.md)
  - [FRD-20251010-puretty-adapter.md](./FRD-20251010-puretty-adapter.md)
- **Docs**:
  - [docs/packages/core.md](../../docs/packages/core.md) - Event types
  - [docs/packages/ui-blocks.md](../../docs/packages/ui-blocks.md) - Block system

---

## 14. Sign-Off

**Author**: Claude (Spin Agent)
**Reviewers**: (pending)
**Approved**: (pending)
**Status**: Draft → Ready for Review

---

**Next Steps**:
1. **Review FRD** (step 5 of 14)
2. **Write tests first** (step 6)
3. **Implement to green** (step 7)
4. **Analyze with uast/herr** (step 8)
5. **Run make lint** (step 9)
6. **Iterate until clean** (step 10-11)
7. **Update ROADMAP** (step 12)
8. **Update docs/** (step 13)
