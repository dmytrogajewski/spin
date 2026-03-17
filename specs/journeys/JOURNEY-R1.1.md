# Journey R-1.1: Head-Tail Output Truncation

**Roadmap Item**: R-1.1
**Spec**: [SPEC.md](../refactoring/opendev-gaps/SPEC.md) Section 1, Stage 5
**Status**: In Progress

## Context

Large command outputs (e.g., `go test ./...` on a big project producing 200K+ characters) overwhelm the LLM context window. The current `ShellCommandTool` returns raw executor output without any truncation strategy. This means the agent loses the tail of output — where test failures, build errors, and final status lines typically appear.

## User Journey

### Persona
Developer using Spin to run tests, builds, or linters on a large codebase.

### Phases

| Phase | Action | Current Experience | Target Experience |
|-------|--------|--------------------|-------------------|
| Execute | Run `go test ./...` on 500-test project | Agent receives 200K chars, context overflows | Agent receives max 30K chars |
| Read head | Inspect imports, package list | Visible (first N chars preserved) | Visible (first 10K preserved) |
| Read tail | Inspect test failures/summary | **Lost** — flat truncation cuts the end | **Preserved** — last 10K chars kept |
| Read middle | Inspect passing tests detail | Visible but useless noise | Replaced with `[N characters omitted]` marker |
| Long lines | Minified JS or data blobs | Single 50K-char line consumes budget | Lines capped at 2000 chars |

### Friction Points (Current)
1. **Lost failures**: Test failure output is at the end; flat truncation loses it entirely.
2. **Wasted context**: Passing test output in the middle is noise but consumes full token budget.
3. **Minified files**: A single long line from bundled JS can consume thousands of tokens.

### Success Criteria
- Output ≤ `MaxOutputChars` (30,000) after truncation.
- First `HeadChars` (10,000) chars always preserved.
- Last `TailChars` (10,000) chars always preserved.
- Lines > `MaxLineChars` (2,000) are cut with a `... [truncated]` suffix.
- Omission marker shows exact character count: `... [N characters omitted] ...`.
- Inputs shorter than the limit pass through unchanged.
- Empty input returns empty output.
- Pure function with no side effects, no I/O, no allocations beyond the result string.

## Technical Design

### Package Location
`internal/tools/truncate.go` — pure functions, no dependencies beyond stdlib.

### Constants
```go
const (
    MaxOutputChars = 30_000
    HeadChars      = 10_000
    TailChars      = 10_000
    MaxLineChars   = 2_000
)
```

### Functions
```go
// TruncateHeadTail truncates s to at most maxTotal characters,
// preserving the first head and last tail characters with an
// omission marker between them.
func TruncateHeadTail(s string, maxTotal, head, tail int) string

// TruncateLines truncates individual lines exceeding maxLen.
func TruncateLines(s string, maxLen int) string

// TruncateOutput applies both line truncation and head-tail truncation
// using the default constants.
func TruncateOutput(s string) string
```

### Integration Point
`ShellCommandTool.buildSuccessResult()` and `buildErrorResult()` call `TruncateOutput()` on the combined output before returning `ToolResult`.

## Test Plan

| Test | Mutant Killed | Description |
|------|---------------|-------------|
| `TestTruncateHeadTail_UnderLimit` | "always truncate" | Input < max passes through unchanged |
| `TestTruncateHeadTail_ExactLimit` | "off-by-one at boundary" | Input == max passes through unchanged |
| `TestTruncateHeadTail_OverLimit` | "missing head or tail" | Preserves first 10K + last 10K |
| `TestTruncateHeadTail_OmissionMarker` | "wrong count" | Marker shows exact omitted char count |
| `TestTruncateHeadTail_EmptyInput` | "nil panic" | Empty string returns empty |
| `TestTruncateLines_UnderLimit` | "always truncate lines" | Short lines unchanged |
| `TestTruncateLines_LongLine` | "missing suffix" | Long line cut with suffix |
| `TestTruncateLines_MultipleLines` | "only first line" | All long lines truncated |
| `TestTruncateLines_EmptyInput` | "nil panic" | Empty returns empty |
| `TestTruncateOutput_Integration` | "stages not composed" | Both line + head-tail applied |
| `TestShellCommandTool_TruncatesOutput` | "not wired" | Tool result is truncated |

## Implementation

**Status**: Complete

### Files Created
- `internal/tools/truncate.go` — `TruncateHeadTail`, `TruncateLines`, `TruncateOutput` pure functions.
- `internal/tools/truncate_test.go` — 11 tests covering all success criteria.

### Files Modified
- `internal/tools/shell_command.go` — `buildSuccessResult()` and `buildErrorResult()` call `TruncateOutput()`.

### Roadmap
- [ROADMAP.md](../refactoring/opendev-gaps/ROADMAP.md) — R-1.1 marked Done.
