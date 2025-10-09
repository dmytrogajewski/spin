# UI Blocks Package

**Package:** `github.com/dmytrogajewski/spin/internal/ui/blocks`
**Purpose:** Block data model and rendering system for TUI timeline

---

## Overview

The blocks package provides the data model and rendering infrastructure for the TUI block system. Blocks represent discrete units of information in the timeline (EXECUTE commands, PLAN updates, diffs, code snippets, etc.) with type-specific metadata and visual rendering.

---

## Core Components

### Block Data Model

**Types:** `Block`, `BlockType`, `FoldState`, `Severity`

Blocks are the fundamental unit of the TUI timeline, containing:
- **Type**: Category (EXECUTE, PLAN, READ, GREP, APPLY_PATCH, SUMMARY, TESTING, NOTICE, ERROR)
- **Metadata**: Type-specific structured data
- **Body**: Renderable content (logs, code, diffs, lists)
- **State**: Expanded/collapsed, severity level

**Implementation:** [model.go](../../internal/ui/blocks/model.go), [types.go](../../internal/ui/blocks/types.go), [metadata.go](../../internal/ui/blocks/metadata.go)

**FRD:** [FRD-20251010-block-types-data-model.md](../../specs/frds/FRD-20251010-block-types-data-model.md)

---

### Block Renderer

**Types:** `Renderer`

Renders blocks to ANSI terminal output following the TUI specification:
- **Headers**: Tag pills with accent colors, title/meta, right-aligned chips
- **Bodies**: Type-specific rendering (diffs, code, lists, transcripts)
- **Footers**: Outcome chips and state labels
- **Design Tokens**: Consistent spacing and color palette

**Implementation:** [renderer.go](../../internal/ui/blocks/renderer.go), [tokens.go](../../internal/ui/blocks/tokens.go)

**FRD:** [FRD-20251010-block-rendering-rules.md](../../specs/frds/FRD-20251010-block-rendering-rules.md)

---

## API

### Creating Blocks

```go
import "github.com/dmytrogajewski/spin/internal/ui/blocks"

// Create a new EXECUTE block
block := blocks.NewBlock(blocks.BlockTypeExecute)
block.Title = "Run tests"
block.Body = "=== RUN TestFoo\n--- PASS: TestFoo (0.00s)\nPASS"

// Set metadata
meta := &blocks.ExecuteMeta{
    Command:    "go test -race ./...",
    CWD:        "/home/user/project",
    Impact:     "medium",
    ExitCode:   ptr.Int(0),
    DurationMS: ptr.Int64(4200),
    LinesOut:   ptr.Int(54),
}
blocks.SetExecuteMeta(block, meta)

// Validate
if err := block.Validate(); err != nil {
    log.Fatal(err)
}
```

### Rendering Blocks

```go
// Create renderer with terminal width
renderer := blocks.NewRenderer(80)

// Render complete block
output, err := renderer.Render(block)
if err != nil {
    log.Fatal(err)
}

fmt.Print(output)
// Output:
// │ ▐EXECUTE▌  Run tests (cmd: "go test -race ./...", cwd: "./", timeout: 600s)  [impact: medium]
// │   === RUN TestFoo
// │   --- PASS: TestFoo (0.00s)
// │   PASS
// │ ✓ [exit: 0] [out: 54 lines] [dur: 4.2s]
```

### Rendering Components

```go
// Render individual components
header := renderer.RenderHeader(block)
body, _ := renderer.RenderBody(block)
footer := renderer.RenderFooter(block)

// Update width on resize
renderer.SetWidth(120)
```

---

## Block Types

### EXECUTE

**Purpose:** Shell command execution

**Metadata:**
- Command, CWD, timeout, impact level
- Exit code, duration, output line count

**Rendering:**
- Transcript body (plain text)
- Success/failure indicator in footer

**Example:**
```
│ ▐EXECUTE▌  go test ./... (cwd: "./")  [impact: medium]
│   === RUN TestFoo
│   --- PASS: TestFoo (0.00s)
│ ✓ [exit: 0] [out: 3 lines] [dur: 0.1s]
```

---

### PLAN

**Purpose:** Task checklist

**Metadata:**
- Total, pending, in-progress, completed counts

**Rendering:**
- Bullet list with status indicators (•/✓/◦)
- Strikethrough for completed items

**Example:**
```
│ ▐PLAN▌  Updated: 3 total (0 pending, 0 in progress, 3 completed)
│   ✓ Task 1 (completed)
│   • Task 2 (pending)
│   ◦ Task 3 (skipped)
```

---

### READ

**Purpose:** File content preview

**Metadata:**
- File path, offset, limit

**Rendering:**
- Code body with line numbers
- Dynamic gutter width (3-6ch)

**Example:**
```
│ ▐READ▌  (file: main.go)
│   │1  package main
│   │2
│   │3  func main() {
│   │4      fmt.Println("hello")
│   │5  }
```

---

### GREP

**Purpose:** Search results

**Metadata:**
- Pattern, mode, context lines

**Rendering:**
- Code body with line numbers
- Filename:line anchors

---

### APPLY_PATCH

**Purpose:** File modification result

**Metadata:**
- File path, success status
- Lines added/removed, error message

**Rendering:**
- Unified diff format
- Red/green lines, hunk headers
- Success/failure footer

**Example:**
```
│ ▐APPLY_PATCH▌  (file: main.go)
│   @@ -42,6 +42,7 @@ func main() {
│    func process() {
│        fmt.Println("processing")
│   +    log.Info("started")
│    }
│ ✓ Succeeded. File edited. (+1 added)
```

---

### SUMMARY

**Purpose:** Human-readable changeset summary

**Rendering:**
- List or plain text body

---

### TESTING

**Purpose:** Test execution results

**Metadata:**
- Suite counts, pass/fail status

**Rendering:**
- List of test commands with status

---

### NOTICE

**Purpose:** System messages

**Rendering:**
- Plain text, muted color
- Example: "Conversation history compressed"

---

### ERROR

**Purpose:** Error messages

**Rendering:**
- First line bold (error message)
- Subsequent lines dim (stack trace)
- Red accent

**Example:**
```
│ ▐ERROR▌  Command failed
│   ● Error: exit status 1
│   Stack trace line 1
│   Stack trace line 2
│ [exit: 1]
```

---

## Design Tokens

### Spacing

```go
S0 = 0   // No gap
S1 = 1   // Minimal spacing
S2 = 2   // Typical margin
S3 = 3   // Component gap
S4 = 4   // Indent level
S6 = 6   // Medium gap
S8 = 8   // Large gap
S12 = 12 // Extra large gap
```

### Colors

**Palette:**
- `ColorBlue` - EXECUTE, TESTING
- `ColorMagenta` - PLAN
- `ColorCyan` - READ, SUMMARY
- `ColorYellow` - GREP, warnings
- `ColorGreen` - APPLY_PATCH, success
- `ColorRed` - ERROR
- `ColorMuted` - NOTICE, secondary text

**Usage:**
```go
tagColor := blocks.GetTagColor(blocks.BlockTypeExecute)
// Returns: ColorBlue
```

---

## Testing

### Unit Tests

```bash
go test ./internal/ui/blocks/...
go test -race ./internal/ui/blocks/...  # With race detector
go test -cover ./internal/ui/blocks/... # Coverage report
```

**Coverage:** 90.5%

**Test categories:**
- Block model: creation, validation, metadata
- Renderer: all block types, edge cases, truncation
- Tokens: color mapping, spacing constants

---

## Metrics

**Phase 4.1 (Data Model):**
- 35 tests, 85.0% coverage
- Max complexity: 3
- Zero lint errors

**Phase 4.2 (Renderer):**
- 38 tests, 90.5% coverage
- Max complexity: 9, avg: 3.14
- Zero lint errors

---

## Timeline

**Types:** `Timeline`, `Viewport`, `Filter`

Manages an ordered collection of blocks with viewport virtualization and filtering support.

**Implementation:** [timeline.go](../../internal/ui/blocks/timeline.go)

**FRD:** [FRD-20251010-block-timeline.md](../../specs/frds/FRD-20251010-block-timeline.md)

### Creating a Timeline

```go
timeline := blocks.NewTimeline()
timeline.SetViewportHeight(20)

// Add blocks
block := NewBlock(blocks.BlockTypeExecute)
block.Title = "Run tests"
timeline.Append(block)
```

### Viewport Management

```go
// Get visible blocks in viewport
visible := timeline.GetVisibleBlocks()

// Scroll operations
timeline.ScrollDown(5)
timeline.ScrollToTop()
timeline.ScrollToBottom()
timeline.ScrollToBlock("blk_123")

// Get viewport state
viewport := timeline.GetViewport()
// Viewport{Start: 0, End: 20, Height: 20}
```

### Filtering

```go
// Filter by block type
filter := &blocks.Filter{
    Types: []blocks.BlockType{blocks.BlockTypeExecute},
}
timeline.SetFilter(filter)

// Filter by exit code
exitCode := 1
filter := &blocks.Filter{
    ExitCode: &exitCode,
}
timeline.SetFilter(filter)

// Combined filters (AND logic)
filter := &blocks.Filter{
    Types:    []blocks.BlockType{blocks.BlockTypeExecute},
    ExitCode: &exitCode,
    Impact:   "high",
}
timeline.SetFilter(filter)

// Clear filter
timeline.ClearFilter()

// Get visible blocks returns filtered + viewport-sliced results
visible := timeline.GetVisibleBlocks()
```

### Focus Navigation

```go
// Focus a specific block
timeline.FocusBlock("blk_123")

// Get focused block
focused, err := timeline.GetFocusedBlock()

// Navigate between blocks
timeline.NextBlock()  // Move to next
timeline.PrevBlock()  // Move to previous
```

### Collapse/Expand

```go
// Toggle single block
timeline.ToggleFold("blk_123")

// Expand/collapse all
timeline.ExpandAll()
timeline.CollapseAll()
```

### Performance

- **Supports 1000+ blocks** without performance degradation
- **Viewport virtualization**: Only visible blocks rendered
- **Filter caching**: Filters evaluated on-demand
- **O(1) operations**: Append, viewport calculation
- **O(n) operations**: Filter (acceptable for 1000s of blocks)

**Metrics:**
- 36 tests, all passing with `-race`
- 89.7% coverage
- Max complexity: 10
- Zero race conditions

---

## Future Enhancements

### Advanced Syntax Highlighting

Tree-sitter or chroma integration for accurate code highlighting.

### Inline Diff

Word-level diff highlighting for changed lines.

### Soft Wrap Mode

Optional soft wrapping with continuation indicator.

---

## References

- **Spec:** [tui-new.md](../../specs/tui-implementation/tui-new.md)
- **Roadmap:** [ROADMAP.md](../../specs/tui-implementation/ROADMAP.md) Phase 4
- **FRDs:**
  - [FRD-20251010-block-types-data-model.md](../../specs/frds/FRD-20251010-block-types-data-model.md)
  - [FRD-20251010-block-rendering-rules.md](../../specs/frds/FRD-20251010-block-rendering-rules.md)

---

**Last Updated:** 2025-10-10
**Status:** ✅ Phase 4.1 & 4.2 Complete
