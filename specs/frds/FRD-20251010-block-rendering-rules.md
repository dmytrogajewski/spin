# FRD-20251010-02: Block Rendering Rules

**Feature:** Block rendering system for TUI timeline
**Priority:** P1 (Critical for Phase 4)
**Date:** 2025-10-10
**Author:** Spin Agent
**Roadmap:** Phase 4.2 (Block Rendering Rules)
**Spec Reference:** [specs/tui-implementation/tui-new.md](../tui-implementation/tui-new.md) Section 3.3, Section 0 (Design Tokens)

---

## 1. Goals & Scope

### 1.1 Primary Goals

Implement rendering logic for all block types following the visual specification:

1. **Block Renderer Infrastructure**: Core renderer that orchestrates header, body, footer rendering
2. **Header Rendering**: Tag pills, title/meta, right-aligned chips with truncation
3. **Body Rendering**: Type-specific rendering (diffs, code, lists, transcripts)
4. **Footer Rendering**: Outcome chips, state labels
5. **Design Token System**: Colors, spacing, typography per spec
6. **Diff Rendering**: Unified format with red/green lines, hunk headers
7. **Code Rendering**: Line numbers, basic syntax highlighting
8. **List Rendering**: Bullets (•, ✓, ◦), strikethrough for done items

### 1.2 Out of Scope

- Timeline state machine (Phase 4.3)
- Navigation and interaction (Phase 6.1)
- Advanced syntax highlighting (future enhancement)
- Theming system (Phase 6.4)
- File preview popup (Phase 6.3)

---

## 2. Technical Design

### 2.1 Package Structure

```
internal/ui/blocks/
├── renderer.go       # Core block renderer
├── renderer_test.go  # Renderer tests (golden tests)
├── header.go         # Header rendering
├── header_test.go    # Header tests
├── body.go           # Body rendering dispatcher
├── body_test.go      # Body tests
├── footer.go         # Footer rendering
├── footer_test.go    # Footer tests
├── diff.go           # Diff rendering
├── diff_test.go      # Diff tests
├── code.go           # Code rendering with line numbers
├── code_test.go      # Code tests
├── list.go           # List/checklist rendering
├── list_test.go      # List tests
├── tokens.go         # Design tokens (colors, spacing)
├── tokens_test.go    # Token tests
```

### 2.2 Design Tokens

Per spec Section 0, implement design tokens for consistent styling:

```go
// Spacing scale (in terminal cells)
const (
    S0 = 0
    S1 = 1
    S2 = 2
    S3 = 3
    S4 = 4
    S6 = 6
    S8 = 8
    S12 = 12
)

// ANSI color codes (256-color)
type Color string

const (
    ColorFg     Color = "\x1b[38;5;252m" // Default foreground
    ColorBg     Color = "\x1b[48;5;233m" // Default background
    ColorMuted  Color = "\x1b[38;5;244m" // Dim text
    ColorBorder Color = "\x1b[38;5;238m" // Borders/separators
    ColorShadow Color = "\x1b[38;5;235m" // Very dim

    // Accents
    ColorBlue    Color = "\x1b[38;5;39m"  // EXECUTE, TESTING
    ColorGreen   Color = "\x1b[38;5;42m"  // APPLY_PATCH, success
    ColorYellow  Color = "\x1b[38;5;221m" // GREP, warnings
    ColorRed     Color = "\x1b[38;5;203m" // ERROR
    ColorMagenta Color = "\x1b[38;5;170m" // PLAN
    ColorCyan    Color = "\x1b[38;5;51m"  // READ, SUMMARY

    ColorReset   Color = "\x1b[0m"
    ColorBold    Color = "\x1b[1m"
    ColorDim     Color = "\x1b[2m"
)

// Tag color map
var TagColors = map[BlockType]Color{
    BlockTypeExecute:    ColorBlue,
    BlockTypePlan:       ColorMagenta,
    BlockTypeRead:       ColorCyan,
    BlockTypeGrep:       ColorYellow,
    BlockTypeApplyPatch: ColorGreen,
    BlockTypeSummary:    ColorCyan,
    BlockTypeTesting:    ColorBlue,
    BlockTypeNotice:     ColorMuted,
    BlockTypeError:      ColorRed,
}
```

### 2.3 Core Renderer

```go
// Renderer renders blocks to ANSI terminal output.
type Renderer struct {
    width int // Terminal width in columns
}

// NewRenderer creates a new block renderer.
func NewRenderer(width int) *Renderer

// Render renders a complete block to a string.
// Returns ANSI-formatted string suitable for terminal output.
func (r *Renderer) Render(b *Block) (string, error)

// RenderHeader renders only the block header.
func (r *Renderer) RenderHeader(b *Block) string

// RenderBody renders only the block body based on type.
func (r *Renderer) RenderBody(b *Block) (string, error)

// RenderFooter renders only the block footer.
func (r *Renderer) RenderFooter(b *Block) string

// SetWidth updates the terminal width (for resize).
func (r *Renderer) SetWidth(width int)
```

### 2.4 Header Rendering

Per spec Section 2.1:

**Layout:**
```
[AccentBar] s2 [TagPill] s2 [Title/Meta (truncated)] s3 [RightMeta]
```

**Components:**
- **AccentBar**: 1ch wide vertical bar in tag color
- **TagPill**: `▐EXECUTE▌` style with tag color
- **Title/Meta**: Primary text bold, secondary dim in parentheses
- **RightMeta**: Right-aligned chips, truncated if space insufficient

```go
// HeaderOptions configures header rendering.
type HeaderOptions struct {
    ShowAccentBar bool
    TruncateTitle bool
    MaxRightMeta  int // Max right-aligned chips (0 = all)
}

// RenderHeaderWithOptions renders header with custom options.
func (r *Renderer) RenderHeaderWithOptions(b *Block, opts HeaderOptions) string

// midEllipsize truncates a string with mid ellipsis (60/40 split).
func midEllipsize(s string, maxWidth int) string
```

### 2.5 Body Rendering

Dispatcher based on block type:

```go
// BodyRenderer is the interface for type-specific body renderers.
type BodyRenderer interface {
    Render(b *Block, width int) (string, error)
}

// Body renderers for each type
type TranscriptRenderer struct{}  // EXECUTE, NOTICE
type CodeRenderer struct{}        // READ
type DiffRenderer struct{}        // APPLY_PATCH
type ListRenderer struct{}        // PLAN, SUMMARY, TESTING
type ErrorRenderer struct{}       // ERROR (specialized transcript)
```

### 2.6 Diff Rendering (APPLY_PATCH)

Per spec Section 2.2.C:

**Features:**
- Unified diff format (`@@ -a,b +c,d @@`)
- Red lines for removals (`-`)
- Green lines for additions (`+`)
- Muted lines for context (no marker)
- Hunk headers in border color
- Per-hunk stats line `(+12/-3)` right-aligned

```go
// DiffRenderer renders unified diffs.
type DiffRenderer struct{}

// Render parses unified diff and applies color/formatting.
func (dr *DiffRenderer) Render(b *Block, width int) (string, error)

// ParseDiff parses unified diff into structured hunks.
func ParseDiff(diff string) ([]*DiffHunk, error)

// DiffHunk represents a single diff hunk.
type DiffHunk struct {
    Header      string      // @@ -a,b +c,d @@
    Lines       []DiffLine
    LinesAdded  int
    LinesRemoved int
}

// DiffLine represents a single line in a diff.
type DiffLine struct {
    Type    DiffLineType // Added, Removed, Context
    Content string
}

type DiffLineType int
const (
    DiffLineContext DiffLineType = iota
    DiffLineAdded
    DiffLineRemoved
)
```

### 2.7 Code Rendering (READ)

Per spec Section 2.2.B:

**Features:**
- Line numbers in gutter (3-6ch width based on max line number)
- Gutter format: `│NNN `
- Basic syntax highlighting (optional initial implementation)
- No soft wrap by default
- File header subline (muted): `internal/tui/ui/input.go  offset:0  limit:120`

```go
// CodeRenderer renders code with line numbers.
type CodeRenderer struct {
    ShowLineNumbers bool
    Highlight       bool // Enable basic syntax highlighting
}

// Render formats code with gutter and optional highlighting.
func (cr *CodeRenderer) Render(b *Block, width int) (string, error)

// gutterWidth calculates gutter width for given line count.
func gutterWidth(lineCount int) int

// highlightLine applies basic lexical highlighting (keywords, comments, strings).
func highlightLine(line string, lang string) string
```

### 2.8 List Rendering (PLAN, SUMMARY, TESTING)

Per spec Section 2.2.D:

**Features:**
- Bullet styles: `•` (pending), `✓` (done), `◦` (skipped)
- Strikethrough for done text (use `~text~` if terminal lacks strikethrough)
- Indent: bullet at `s2`, text at `s4`
- Vertical spacing: `s0` between items, `s1` before nested group

```go
// ListRenderer renders bulleted lists.
type ListRenderer struct{}

// Render parses body as list items and formats with bullets.
func (lr *ListRenderer) Render(b *Block, width int) (string, error)

// ListItem represents a single list item.
type ListItem struct {
    State   ItemState // Pending, Done, Skipped
    Text    string
    Indent  int // Nesting level (0 = top level)
}

type ItemState int
const (
    ItemStatePending ItemState = iota
    ItemStateDone
    ItemStateSkipped
)

// ParseList parses body into list items.
// Format: "• text", "✓ text", "◦ text", or markdown "- [ ] text", "- [x] text"
func ParseList(body string) ([]ListItem, error)
```

### 2.9 Footer Rendering

Per spec Section 2.3:

**Layout:**
```
[LeftChips] ... [RightLabels]
```

**Left chips** (outcome): `[exit: 0] [out: 54 lines] [dur: 4.2s]`
**Right labels** (state): `[cached] [streaming] [compressed]` in muted

```go
// RenderFooter extracts metadata and formats footer chips.
func (r *Renderer) RenderFooter(b *Block) string

// Chip helpers
func exitCodeChip(code int) string
func durationChip(ms int64) string
func linesChip(count int) string
func impactChip(impact string) string
func stateLabel(label string) string
```

### 2.10 Truncation and Wrapping

Per spec Section 10:

**Header truncation:**
- Mid-ellipsize: `left…right` (60/40 split)
- Preserve beginning and end of title

**Body wrapping:**
- Default: no wrap, horizontal scroll indicator `↔` at right margin
- Optional wrap mode (future): soft wrap with continuation prefix `⋮`

```go
// midEllipsize truncates string with middle ellipsis.
// Example: "very long filename.go" → "very long…name.go" (60/40)
func midEllipsize(s string, maxWidth int) string

// Truncation preserves readability for file paths and commands.
```

---

## 3. Rendering Rules Summary

### 3.1 EXECUTE Block

**Header:**
- Tag: `▐EXECUTE▌` (blue)
- Meta: `(cmd: "go test", cwd: "./", timeout: 600s)`
- Right chips: `[impact: medium]`

**Body:**
- Transcript renderer
- Plain stdout/stderr text
- No line numbers
- Collapsible if > 200 lines

**Footer:**
- `[exit: 0] [out: 54 lines] [dur: 4.2s]`
- Success: `✓` prefix
- Error: `●` prefix (red)

### 3.2 PLAN Block

**Header:**
- Tag: `▐PLAN▌` (magenta)
- Meta: `Updated: 3 total (0 pending, 0 in progress, 3 completed)`

**Body:**
- List renderer
- Bullets: `•` pending, `✓` done, `◦` skipped
- Strikethrough for done items

**Footer:**
- (optional) `[last_updated: timestamp]`

### 3.3 READ Block

**Header:**
- Tag: `▐READ▌` (cyan)
- Meta: `(file: internal/tui/ui/input.go, offset: 0, limit: 120)`

**Body:**
- Code renderer
- Line numbers in gutter
- File header subline: `internal/tui/ui/input.go  offset:0  limit:120`
- Optional basic syntax highlighting

**Footer:**
- `[lines: 120]`

### 3.4 GREP Block

**Header:**
- Tag: `▐GREP▌` (yellow)
- Meta: `(pattern: "error", mode: content, context: 2)`

**Body:**
- Code renderer (with filename:line anchors)
- Matches underlined or highlighted
- Format: `filename:line: matched content`

**Footer:**
- `[matches: 12] [files: 3]`

### 3.5 APPLY_PATCH Block

**Header:**
- Tag: `▐APPLY_PATCH▌` (green)
- Meta: `(file: internal/ui/blocks/renderer.go)`

**Body:**
- Diff renderer
- Unified format with hunk headers
- Red `-` lines, green `+` lines
- Per-hunk stats `(+12/-3)`

**Footer:**
- Success: `✓ Succeeded. File edited. (+1 added)`
- Error: `● Failed: [error message]`

### 3.6 SUMMARY Block

**Header:**
- Tag: `▐SUMMARY▌` (cyan)
- No additional meta

**Body:**
- List renderer (for bullet points)
- Or plain text (for paragraphs)

**Footer:**
- (none)

### 3.7 TESTING Block

**Header:**
- Tag: `▐TESTING▌` (blue)
- Meta: `(suites: 5, failed: 2)`

**Body:**
- List renderer
- Commands with status indicators
- Failed suites in yellow/red with reasons

**Footer:**
- `[passed: 3] [failed: 2] [duration: 12.3s]`

### 3.8 NOTICE Block

**Header:**
- Tag: `▐NOTICE▌` (muted)
- Meta: `(type: system)`

**Body:**
- Transcript renderer (plain text)
- Muted color
- Example: "Conversation history has been compressed…"

**Footer:**
- (none)

### 3.9 ERROR Block

**Header:**
- Tag: `▐ERROR▌` (red)
- Meta: `(cause: "command timeout")`
- Prefix: `●` (red)

**Body:**
- Error renderer (specialized transcript)
- First line bold (error message)
- Subsequent lines (stack trace) muted
- Collapsible if > 20 lines

**Footer:**
- `[exit: 1]` or `[timeout]`

---

## 4. Testing Strategy

### 4.1 Unit Tests

**Renderer tests:**
- NewRenderer creates with correct defaults
- SetWidth updates width correctly

**Header tests:**
- Tag pill renders with correct color
- Title truncates correctly (mid-ellipsize)
- Right meta truncates when space insufficient
- Accent bar renders correctly

**Body tests:**
- Dispatcher selects correct renderer per type
- Each renderer handles empty body gracefully
- Long bodies handle truncation correctly

**Footer tests:**
- Chip formatting correct for all metadata types
- State labels render correctly
- Left/right alignment works

**Diff tests:**
- Parses unified diff correctly
- Colors lines correctly (red/green/context)
- Hunk headers render correctly
- Stats line calculates correctly

**Code tests:**
- Line numbers align correctly (3-6ch gutter)
- File header renders correctly
- Empty code blocks handled
- Very long lines handled

**List tests:**
- Parses markdown lists correctly
- Bullet styles correct (•/✓/◦)
- Strikethrough renders correctly
- Indentation correct

### 4.2 Golden Tests

For each block type, golden tests compare rendered output to expected ANSI:

```go
func TestRenderExecuteBlock_Golden(t *testing.T) {
    block := blocks.NewBlock(blocks.BlockTypeExecute)
    // ... setup block ...

    renderer := NewRenderer(80)
    got := renderer.Render(block)

    golden.Assert(t, got, "execute_block_basic.golden")
}
```

**Golden files:**
- `execute_block_basic.golden`
- `plan_block_with_completed.golden`
- `read_block_with_code.golden`
- `grep_block_with_matches.golden`
- `patch_block_success.golden`
- `patch_block_failure.golden`
- `error_block_with_trace.golden`
- etc.

### 4.3 Edge Case Tests

**Truncation:**
- Very long titles (test mid-ellipsize)
- Very long right meta (test truncation)
- Header with no space for right meta

**Wrapping:**
- Code lines exceeding terminal width
- Diff lines exceeding width
- List items exceeding width

**Empty/Invalid:**
- Empty block body
- Invalid diff format
- Invalid list format
- Missing metadata

**Wide Characters:**
- Emoji in titles
- CJK characters in code
- Combining marks

### 4.4 Coverage Target

- Overall: ≥85%
- Critical rendering paths: ≥90%
- Diff/code/list parsers: ≥95%

---

## 5. Examples

### 5.1 Render EXECUTE Block

```go
block := blocks.NewBlock(blocks.BlockTypeExecute)
block.Title = "Run tests"
block.Body = "=== RUN TestFoo\n--- PASS: TestFoo (0.00s)\nPASS\n"

meta := &blocks.ExecuteMeta{
    Command:    "go test -race ./...",
    CWD:        "/home/user/project",
    TimeoutSec: 600,
    Impact:     "medium",
    ExitCode:   ptr.Int(0),
    DurationMS: ptr.Int64(4200),
    LinesOut:   ptr.Int(54),
}
blocks.SetExecuteMeta(block, meta)

renderer := blocks.NewRenderer(80)
output, err := renderer.Render(block)
if err != nil {
    log.Fatal(err)
}

fmt.Print(output)
```

**Output:**
```
│ ▐EXECUTE▌  Run tests (cmd: "go test -race ./...", cwd: "./", timeout: 600s)  [impact: medium]
│   === RUN TestFoo
│   --- PASS: TestFoo (0.00s)
│   PASS
│ ✓ [exit: 0] [out: 54 lines] [dur: 4.2s]
```

### 5.2 Render PATCH Block

```go
block := blocks.NewBlock(blocks.BlockTypeApplyPatch)
block.Body = `@@ -42,6 +42,7 @@ func main() {
 func process() {
     fmt.Println("processing")
+    log.Info("started")
 }
`

meta := &blocks.PatchMeta{
    File:       "main.go",
    Succeeded:  true,
    LinesAdded: ptr.Int(1),
}
blocks.SetPatchMeta(block, meta)

renderer := blocks.NewRenderer(80)
output, _ := renderer.Render(block)
fmt.Print(output)
```

**Output:**
```
│ ▐APPLY_PATCH▌  main.go
│   @@ -42,6 +42,7 @@ func main()
│    func process() {
│        fmt.Println("processing")
│   +    log.Info("started")
│    }
│ ✓ Succeeded. File edited. (+1 added)                                    (+1/-0)
```

---

## 6. Quality Gates

### 6.1 Definition of Done

- [ ] All tests pass with `-race`
- [ ] Coverage ≥85%
- [ ] `make lint` clean
- [ ] Complexity ≤15 per function
- [ ] All block types render per spec
- [ ] Colors match tag color map
- [ ] Spacing matches design tokens
- [ ] Diff rendering shows red/green/context correctly
- [ ] Code rendering shows line numbers correctly
- [ ] List rendering handles all bullet types
- [ ] Golden tests verify exact output
- [ ] Godoc on all exports
- [ ] No dead code

### 6.2 Acceptance Criteria

**Block Rendering:**
- All 9 block types render correctly
- Header, body, footer components work independently
- Truncation preserves readability
- ANSI color codes apply correctly

**Diff Rendering:**
- Unified diff format parsed correctly
- Colors: red (`-`), green (`+`), muted (context)
- Hunk headers in border color
- Stats line calculates correctly

**Code Rendering:**
- Line numbers align (dynamic gutter width)
- File header subline renders
- Long lines handled (no wrap by default)

**List Rendering:**
- Markdown lists parsed
- Bullet styles correct (•/✓/◦)
- Strikethrough works
- Indentation correct

---

## 7. Dependencies

### 7.1 External Dependencies

- `fmt`, `strings`, `strconv` (stdlib)
- `github.com/rivo/uniseg` (for grapheme width calculation)

### 7.2 Internal Dependencies

- `internal/ui/blocks` (types, model)

---

## 8. Risks & Mitigations

### 8.1 Risk: Complex Diff Parsing

**Scenario:** Diff parsing fails on edge cases
**Mitigation:** Comprehensive test suite, graceful degradation (show raw diff)
**Test:** Fuzzing with real-world diffs

### 8.2 Risk: Wide Character Rendering

**Scenario:** CJK/emoji break alignment
**Mitigation:** Use `rivo/uniseg` for accurate width
**Test:** Wide character test cases

### 8.3 Risk: Performance on Large Blocks

**Scenario:** Rendering 10k line blocks lags
**Mitigation:** Virtualization (Phase 4.3), lazy rendering
**Test:** Benchmark with large blocks

---

## 9. Future Enhancements

### 9.1 Advanced Syntax Highlighting

Use tree-sitter or chroma library for accurate highlighting:

```go
func (cr *CodeRenderer) HighlightWithTreeSitter(code, lang string) string
```

### 9.2 Inline Diff (Word-Level)

Highlight changed words within lines:

```go
func intralineDiff(oldLine, newLine string) ([]DiffSpan, error)
```

### 9.3 Soft Wrap Mode

Optional soft wrapping with continuation indicator:

```go
type WrapMode int
const (
    WrapNone WrapMode = iota
    WrapSoft // Continuation prefix ⋮
)
```

---

## 10. References

- **Spec:** [specs/tui-implementation/tui-new.md](../tui-implementation/tui-new.md) Section 0, 3.3
- **Roadmap:** [specs/tui-implementation/ROADMAP.md](../tui-implementation/ROADMAP.md) Phase 4.2
- **Phase 4.1 FRD:** [FRD-20251010-block-types-data-model.md](FRD-20251010-block-types-data-model.md)
- **AGENTS.md:** Working loop steps 1-14

---

## 11. Changelog

**2025-10-10:** Initial FRD
**Author:** Spin Agent
**Status:** Ready for Implementation
