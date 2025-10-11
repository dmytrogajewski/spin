# FRD-20251011-230532: Pretty-Formatted Tool Call Display

**Status:** Draft
**Created:** 2025-10-11
**Author:** Spin Agent
**Priority:** P2 (UX Enhancement)

---

## 1. Problem Statement

### 1.1 Current State

Currently, when the agent executes tools (commands, file operations, etc.), the TUI displays them as blocks in the timeline. However, there's no immediate visual feedback showing:
- What tool was called
- What parameters were passed to the tool
- What the result was (exit code, line count, duration)

Users see blocks appear in the timeline but have to expand them or infer from the block headers what happened. This creates a disconnect between the agent's actions and the user's understanding.

### 1.2 Desired State

When a tool is called, the user should see a **compact, pretty-formatted summary** that immediately communicates:
1. **WHAT** tool was executed (with colored background tag)
2. **WITH WHAT** parameters (truncated, human-readable)
3. **WHAT HAPPENED** (result: exit code, output size, duration)

**Example format:**
```
 EXECUTE  (notify-send me smilet, impact: medium)
 ↳ Exit code: 0. Output: 2 lines.
```

```
 READ  (package.json, offset: 0, limit: 400)
 ↳ Read 62 lines.
```

This provides **at-a-glance status** without needing to open blocks, similar to how modern IDEs show test results inline.

### 1.3 Why This Matters

**User Benefits:**
- Immediate feedback on tool execution
- No need to expand blocks for basic status
- Easier to scan conversation history
- Clear separation between agent thinking and tool execution

**Technical Benefits:**
- Leverages existing event stream architecture
- No changes to core agent logic
- Pure presentation layer enhancement

---

## 2. Requirements

### 2.1 Functional Requirements

**FR-1: Tool Start Notification**
- MUST display formatted line when `EventToolCallStart` occurs
- MUST show tool name with colored background (using block colors)
- MUST show key parameters in human-readable form
- MUST truncate long parameters to fit on one line

**FR-2: Tool Complete Notification**
- MUST display formatted result line when `EventToolCallComplete` occurs
- MUST use arrow symbol (↳) to indicate continuation
- MUST show success indicator (exit code, output size, etc.)
- MUST be compact (1-2 lines total per tool call)

**FR-3: Tool-Specific Formatting**
- `execute_command`: Show command + exit code + duration
- `read_file`: Show file path + lines read
- `write_file`: Show file path + success/failure
- `grep`: Show pattern + match count
- `list_directory`: Show path + item count

**FR-4: Visual Design**
- MUST use existing block color scheme for consistency
- MUST align with TUI design tokens (spacing, colors)
- MUST be readable in both dark and light terminals
- MUST not break existing block rendering

### 2.2 Non-Functional Requirements

**NFR-1: Performance**
- MUST render in <1ms per tool call
- MUST not block event stream processing
- MUST not increase memory footprint

**NFR-2: Compatibility**
- MUST work with existing `CoordinatedWriter` system
- MUST preserve prompt positioning
- MUST work in all terminal emulators (80+ columns)

**NFR-3: Maintainability**
- MUST use existing UI primitives (blocks, colors, spacing)
- MUST be testable via unit tests
- MUST follow Spin coding patterns

---

## 3. Design

### 3.1 Architecture

**Component: ToolFormatter** (new)
- **Location:** `internal/ui/output/tool_formatter.go`
- **Responsibility:** Format tool call events into compact display strings
- **Dependencies:** `internal/core` (event types), `internal/ui/blocks` (colors)

**Integration Point:** `TUIMapper`
- When `EventToolCallStart` is received, call `ToolFormatter.FormatStart()`
- When `EventToolCallComplete` is received, call `ToolFormatter.FormatComplete()`
- Output formatted strings via `UI.PrintLine()`

### 3.2 Data Flow

```
EventToolCallStart
    ↓
TUIMapper.handleToolStart()
    ↓
ToolFormatter.FormatStart(data) → string
    ↓
UI.PrintLine(formatted string)
    ↓
[Block created and appended to timeline as usual]

EventToolCallComplete
    ↓
TUIMapper.handleToolComplete()
    ↓
ToolFormatter.FormatComplete(data) → string
    ↓
UI.PrintLine(formatted string)
    ↓
[Block updated in timeline as usual]
```

**Key Insight:** This is **additive** to the existing block system, not a replacement. Blocks still exist for detailed history, but these compact lines provide real-time feedback.

### 3.3 API Design

```go
package output

// ToolFormatter formats tool call events into compact display strings.
type ToolFormatter struct {
	width int // Terminal width for truncation
}

// NewToolFormatter creates a formatter with the given terminal width.
func NewToolFormatter(width int) *ToolFormatter

// FormatStart formats a tool call start event.
// Returns a formatted string like:
//   " EXECUTE  (go test ./..., impact: medium)"
func (f *ToolFormatter) FormatStart(data core.ToolCallStartData) string

// FormatComplete formats a tool call completion event.
// Returns a formatted string like:
//   " ↳ Exit code: 0. Output: 42 lines. Duration: 1.2s"
func (f *ToolFormatter) FormatComplete(data core.ToolCallCompleteData) string
```

### 3.4 Formatting Rules

**Start Line Format:**
```
<space><TAG><space><space>(<params>)
```

Where:
- `<TAG>` = Tool name with colored background (e.g., `\e[44m EXECUTE \e[0m`)
- `<params>` = Comma-separated key:value pairs, truncated to fit

**Complete Line Format:**
```
<space>↳<space><result>
```

Where:
- `↳` = Unicode arrow (U+21B3)
- `<result>` = Tool-specific summary (exit code, lines, duration, etc.)

**Examples:**

```go
// execute_command
FormatStart(...)  → " EXECUTE  (go test ./..., cwd: ., impact: medium)"
FormatComplete(...) → " ↳ Exit code: 0. Output: 15 lines. Duration: 1.2s"

// read_file
FormatStart(...)  → " READ  (package.json, offset: 0, limit: 400)"
FormatComplete(...) → " ↳ Read 62 lines."

// write_file
FormatStart(...)  → " WRITE  (main.go, 1250 bytes)"
FormatComplete(...) → " ↳ File written successfully."
```

### 3.5 Color Mapping

Reuse existing block color constants from `internal/ui/blocks/tokens.go`:

| Tool Type | Block Type | Color | ANSI Code |
|-----------|-----------|-------|-----------|
| `execute_command` | EXECUTE | Blue | `\e[44m` (bg) |
| `read_file` | READ | Cyan | `\e[46m` (bg) |
| `write_file` | APPLY_PATCH | Green | `\e[42m` (bg) |
| `grep` | GREP | Yellow | `\e[43m` (bg) |
| `list_directory` | EXECUTE | Blue | `\e[44m` (bg) |

**Implementation:**
```go
// Map tool name to block type, then use existing GetTagColor()
func (f *ToolFormatter) getToolColor(toolName string) string {
	blockType := mapToolToBlockType(toolName)
	return blocks.GetTagColor(blockType)
}
```

### 3.6 Parameter Truncation

**Goal:** Fit start line within terminal width (default 80 columns).

**Strategy:**
1. Reserve space for tag (12 chars) + padding (5 chars) = **17 chars fixed**
2. Remaining space: `width - 17` for parameters
3. Show most important parameters first
4. Truncate with `...` if over limit

**Priority by tool:**
- `execute_command`: command (required), cwd (optional), impact (optional)
- `read_file`: path (required), offset (optional), limit (optional)
- `write_file`: path (required), size (optional)
- `grep`: pattern (required), mode (optional), context (optional)

**Example:**
```go
// If width=80, params space=63 chars
params := []string{
    "command: go test -race -cover ./internal/...",
    "cwd: /home/user/project",
    "impact: high",
}
// Total length: 82 chars → truncate command to 40 chars
result := "command: go test -race -cover ./int..., cwd: /home/user/project, impact: high"
```

---

## 4. Implementation Plan

### 4.1 Phase 1: Core Formatter (2 hours)

**Files:**
- `internal/ui/output/tool_formatter.go` (new, ~200 lines)
- `internal/ui/output/tool_formatter_test.go` (new, ~300 lines)

**Tasks:**
1. Create `ToolFormatter` struct with width field
2. Implement `FormatStart()` with parameter extraction and truncation
3. Implement `FormatComplete()` with result formatting
4. Add color mapping via `mapToolToBlockType()`
5. Write table-driven tests for all tool types
6. Test truncation edge cases (very long commands, Unicode, etc.)

**Acceptance:**
- All tests pass with `-race`
- Coverage ≥90%
- No lint errors
- Complexity ≤10

### 4.2 Phase 2: TUIMapper Integration (1 hour)

**Files:**
- `internal/core/tui_mapper.go` (modify, ~30 lines added)
- `internal/core/tui_mapper_test.go` (modify, add 2 tests)

**Tasks:**
1. Add `ToolFormatter` field to `TUIMapper`
2. Initialize formatter in `NewTUIMapper()` with width=80
3. In `handleToolStart()`, call `FormatStart()` and `UI.PrintLine()`
4. In `handleToolComplete()`, call `FormatComplete()` and `UI.PrintLine()`
5. Add tests for formatter integration

**Acceptance:**
- Tool calls print formatted lines before block creation
- Prompt repositions correctly after each line
- Tests verify line format

### 4.3 Phase 3: E2E Testing (1 hour)

**Files:**
- `cmd/spin/tui_test.go` (new or modify)

**Tasks:**
1. Create integration test that starts TUI, sends prompt, captures output
2. Verify formatted lines appear in correct order
3. Test with multiple concurrent tool calls
4. Test with errors (non-zero exit codes)

**Acceptance:**
- E2E test passes with real LLM (or mock)
- Output matches expected format
- No race conditions

### 4.4 Phase 4: Documentation (30 min)

**Files:**
- `docs/packages/ui-output.md` (update)
- `docs/tui.md` (update "Tool Execution Feedback" section)

**Tasks:**
1. Document `ToolFormatter` API in ui-output.md
2. Add examples to tui.md showing formatted output
3. Update screenshots if applicable

---

## 5. Testing Strategy

### 5.1 Unit Tests

**File:** `internal/ui/output/tool_formatter_test.go`

**Test Cases:**
1. `TestFormatStart_ExecuteCommand` - basic command formatting
2. `TestFormatStart_ReadFile` - file path with offset/limit
3. `TestFormatStart_WriteFile` - file path with size
4. `TestFormatStart_Truncation` - long parameters
5. `TestFormatStart_Unicode` - Unicode in parameters
6. `TestFormatComplete_Success` - successful execution
7. `TestFormatComplete_Failure` - failed execution with error
8. `TestFormatComplete_NoOutput` - empty output
9. `TestColorMapping` - verify correct colors for each tool type
10. `TestWidthRespect` - verify output never exceeds terminal width

**Coverage Target:** ≥90%

### 5.2 Integration Tests

**File:** `internal/core/tui_mapper_test.go`

**Test Cases:**
1. `TestTUIMapper_ToolCallFormattedOutput` - verify PrintLine called with correct format
2. `TestTUIMapper_ToolCallBlockCreation` - verify block still created after line print

### 5.3 E2E Tests

**File:** `cmd/spin/tui_test.go` (or manual)

**Scenarios:**
1. Start TUI, send prompt "run ls -la", verify formatted output appears
2. Start TUI, send prompt "read package.json", verify file read formatted
3. Start TUI, send prompt "write hello.txt", verify file write formatted
4. Verify output is readable in 80-column terminal
5. Verify colors appear correctly (visual check)

### 5.4 Performance Tests

**Benchmark:**
```go
func BenchmarkFormatStart(b *testing.B) {
	formatter := NewToolFormatter(80)
	data := core.ToolCallStartData{
		ToolName: "execute_command",
		ToolID:   "tool_123",
		Parameters: types.ToolCallArguments{...},
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		formatter.FormatStart(data)
	}
}
```

**Target:** <1µs per call (negligible overhead)

---

## 6. Risks and Mitigations

### 6.1 Risk: Terminal Width Variability

**Problem:** Users have different terminal widths (80, 120, 200+ columns). Fixed formatting may break.

**Mitigation:**
- Make `ToolFormatter` accept width parameter
- TUI reads actual terminal width via `term.GetSize()`
- Update formatter width on `SIGWINCH` (resize signal)
- Always truncate safely with `...` indicator

### 6.2 Risk: Unicode Rendering Issues

**Problem:** Unicode arrow (↳) may not render in some terminals or fonts.

**Mitigation:**
- Use widely supported Unicode (U+21B3 is in most fonts)
- Add fallback to ASCII (`->`) if Unicode detection fails
- Document font requirements in troubleshooting guide

### 6.3 Risk: Color Blindness

**Problem:** Users with color blindness may not distinguish tool types.

**Mitigation:**
- Use distinct tag **text** in addition to color (`EXECUTE`, `READ`, etc.)
- Colors are supplementary, not required for understanding
- Future: Add option to disable colors or use high-contrast mode

### 6.4 Risk: Performance Overhead

**Problem:** Formatting on every tool call might slow down event processing.

**Mitigation:**
- Formatting is pure string manipulation (~1µs)
- No blocking IO, no allocations beyond string concat
- Benchmark to verify <1% overhead
- If needed, add config flag to disable formatting

---

## 7. Alternatives Considered

### 7.1 Alternative A: Inline Block Headers

**Description:** Show formatted summary in block header instead of separate line.

**Example:**
```
│ ▐EXECUTE▌  go test ./... (exit: 0, 15 lines, 1.2s)
│   === RUN TestFoo
│   --- PASS: TestFoo (0.00s)
```

**Pros:**
- No additional lines
- Integrated with existing blocks

**Cons:**
- No immediate feedback (block appears only after tool completes)
- Header gets cluttered with metadata
- Harder to scan for tool results

**Decision:** Rejected. Real-time feedback is more valuable.

### 7.2 Alternative B: Status Bar

**Description:** Show tool execution in a dedicated status bar at the bottom.

**Example:**
```
│ Timeline...
│
> _                                   [Running: go test ./...]
```

**Pros:**
- Doesn't clutter timeline
- Always visible

**Cons:**
- Requires status bar implementation (not yet in TUI)
- Transient status (disappears after tool completes)
- No history of tool calls

**Decision:** Rejected. Adds complexity, loses history. Consider for Phase 8.

### 7.3 Alternative C: Rich Notifications

**Description:** Use terminal notifications (BEL, OSC 9) for tool completion.

**Pros:**
- Native OS notifications
- Works even if terminal is backgrounded

**Cons:**
- Not all terminals support notifications
- Annoying for frequent tool calls
- No in-app history

**Decision:** Rejected. Too invasive. Consider as opt-in feature later.

---

## 8. Success Metrics

### 8.1 Quantitative Metrics

- **Test Coverage:** ≥90% for `tool_formatter.go`
- **Performance:** <1µs per `FormatStart()` call
- **Lint Errors:** 0
- **Cyclomatic Complexity:** ≤10 per function

### 8.2 Qualitative Metrics

- **User Feedback:** "I can see what the agent is doing in real-time"
- **Dogfooding:** Spin developers find formatted output useful during development
- **Visual Quality:** Output is readable, well-aligned, and visually distinct

### 8.3 Acceptance Criteria

- [ ] All unit tests pass with `-race`
- [ ] Integration tests verify PrintLine called correctly
- [ ] E2E test shows formatted output for all tool types
- [ ] Documentation updated with examples
- [ ] `make lint` passes (zero errors)
- [ ] Manual QA in 80, 120, and 200-column terminals
- [ ] No regression in existing TUI features (blocks still render correctly)

---

## 9. Timeline

**Total Estimate:** 4.5 hours

- Phase 1 (Core Formatter): 2 hours
- Phase 2 (Integration): 1 hour
- Phase 3 (E2E Testing): 1 hour
- Phase 4 (Documentation): 30 minutes

**Target Completion:** 2025-10-12 (1 day)

---

## 10. Open Questions

1. **Q:** Should we support custom formatting templates?
   **A:** No, not in v1. Keep it simple and consistent. Consider in Phase 8 (Polish).

2. **Q:** Should formatted lines be persistent (logged to session file)?
   **A:** No, they're ephemeral UI feedback. Blocks contain the persistent data.

3. **Q:** What about nested tool calls (tools calling tools)?
   **A:** Indent continuation line by 2 spaces for nested calls. Scope: v2.

4. **Q:** Should we show progress for long-running tools?
   **A:** Not in v1. Use `EventToolCallProgress` for future progress bars. Scope: Phase 8.

---

## 11. References

- **UI Blocks:** [docs/packages/ui-blocks.md](../../docs/packages/ui-blocks.md)
- **UI Output:** [docs/packages/ui-output.md](../../docs/packages/ui-output.md)
- **Core Events:** [internal/core/event.go](../../internal/core/event.go)
- **TUI Mapper:** [internal/core/tui_mapper.go](../../internal/core/tui_mapper.go)
- **AGENTS.md:** [AGENTS.md](../../AGENTS.md) (workflow)

---

**FRD Status:** Draft → Ready for Implementation
**Next Steps:** Review FRD, proceed to Phase 1 implementation
