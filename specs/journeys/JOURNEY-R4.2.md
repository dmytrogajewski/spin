# Journey R-4.2: Fuzzy Edit Chain (9-Pass Matching)

**Roadmap Item**: R-4.2
**Spec**: [SPEC.md](../refactoring/opendev-gaps/SPEC.md) Section 4
**Status**: In Progress

## Context

LLMs frequently produce `old_content` that drifts from the actual file content — extra whitespace, wrong indentation, escape sequence differences, CRLF/LF mismatches, or trailing spaces. A 9-pass fuzzy matching chain progressively relaxes matching to find the intended region, while preserving the original file formatting.

## User Journey

### Persona
Developer using Spin to edit files via the agent.

### Phases

| Phase | Action | Current Experience | Target Experience |
|-------|--------|--------------------|-------------------|
| Read | Agent reads `main.go` | Content returned | Content returned |
| Edit | Agent calls `edit_file` with drifted `old_content` | No `edit_file` tool; must use `write_file` (full overwrite) or `apply_patch` | Fuzzy chain matches despite drift, applies edit precisely |
| Feedback | Agent sees result | Full file overwritten | Unified diff + pass name that matched |
| Conflict | Two regions match | Silent first-match | Error: "ambiguous match — 2 occurrences found" |

### Success Criteria
- 9-pass chain: exact → whitespace → indent → escape → lineend → trim → collapse → anchor → partial.
- Each pass returns the **actual matched substring** from the file.
- `Find()` short-circuits on first successful pass.
- Ambiguous matches (>1 occurrence) are rejected.
- `EditFileTool` calls `tracker.AssertFresh()` before edit and `tracker.RecordRead()` after.
- Tool returns unified diff and pass name.
- Registered in `BuiltinRuntime.RegisterTools()`.

## Technical Design

### Package Location
- `internal/tools/fuzzy/` — matching chain package.
- `internal/tools/edit_file.go` — `EditFileTool`.

### Types
```go
// MatchResult holds the result of a fuzzy match.
type MatchResult struct {
    Start    int    // byte offset in file content
    End      int    // byte offset end (exclusive)
    Original string // actual matched substring from file
    PassName string // name of the pass that matched
}

// Pass attempts to find old content in file content.
type Pass struct {
    Name string
    Find func(fileContent, oldContent string) []MatchResult
}

// Chain runs passes in order, short-circuits on first match.
type Chain struct {
    passes []Pass
}
```

### Pass Descriptions
1. **Exact** — `strings.Index`, count occurrences.
2. **Whitespace** — collapse runs of spaces/tabs to single space, compare.
3. **Indent** — strip common leading whitespace per line, compare line-by-line.
4. **Escape** — normalize `\n`, `\t`, `\"`, `\\` literals to actual characters.
5. **LineEnd** — normalize `\r\n` → `\n`.
6. **Trim** — trim trailing whitespace per line.
7. **Collapse** — collapse consecutive blank lines to single blank line.
8. **Anchor** — use first+last non-blank lines as anchors, find matching region.
9. **Partial** — longest common substring above 60% threshold.

### EditFileTool
- Implements `Tool` + `ToolWithApproval`.
- Parameters: `path`, `old_content`, `new_content`.
- Flow: assert fresh → find match → verify uniqueness → replace → record read → return diff.

## Test Plan

| Test | Mutant Killed | Description |
|------|---------------|-------------|
| `TestExactPass_Match` | "exact skipped" | Exact match found |
| `TestExactPass_NoMatch` | "false match" | No match returns empty |
| `TestExactPass_AmbiguousMatch` | "ambiguity ignored" | Two matches returns both |
| `TestWhitespacePass_ExtraSpaces` | "whitespace not normalized" | Extra spaces matched |
| `TestIndentPass_DifferentIndentation` | "indent not stripped" | Different indent matched |
| `TestEscapePass_EscapeDifferences` | "escapes not normalized" | Escape sequences matched |
| `TestLineEndPass_CRLFvsLF` | "CRLF not normalized" | CRLF matched as LF |
| `TestTrimPass_TrailingSpaces` | "trailing not trimmed" | Trailing spaces matched |
| `TestCollapsePass_ExtraBlankLines` | "blanks not collapsed" | Extra blank lines matched |
| `TestAnchorPass_ContextAnchors` | "anchors missed" | Anchor lines find region |
| `TestPartialPass_SubstringMatch` | "partial missed" | Longest common substring found |
| `TestChain_ShortCircuitsOnExact` | "chain not short-circuit" | Exact match stops chain |
| `TestChain_FallsThroughToFuzzy` | "chain not progressive" | Falls through to fuzzy pass |
| `TestChain_AllPassesFail` | "nil not returned" | No match returns nil |
| `TestEditFileTool_SuccessfulEdit` | "edit not applied" | Edit applied correctly |
| `TestEditFileTool_StaleReadRejection` | "stale not checked" | Stale file rejected |
| `TestEditFileTool_AmbiguousMatchRejection` | "ambiguity allowed" | Ambiguous match rejected |
| `TestEditFileTool_ReturnsDiff` | "diff missing" | Output contains diff |

## Implementation

**Status**: Complete

### Files Created
- `internal/tools/fuzzy/doc.go` — package documentation.
- `internal/tools/fuzzy/chain.go` — `Chain`, `MatchResult`, `Pass`, `Find()`, `FindAll()`, `DefaultChain()`.
- `internal/tools/fuzzy/exact.go` — `ExactFind()` exact string matching.
- `internal/tools/fuzzy/whitespace.go` — `WhitespaceFind()` + `findByNormalized()` helper.
- `internal/tools/fuzzy/indent.go` — `IndentFind()` + `findLineSequence()` helper.
- `internal/tools/fuzzy/escape.go` — `EscapeFind()` escape sequence normalization.
- `internal/tools/fuzzy/lineend.go` — `LineEndFind()` CRLF normalization.
- `internal/tools/fuzzy/trim.go` — `TrimFind()` trailing whitespace removal.
- `internal/tools/fuzzy/collapse.go` — `CollapseFind()` blank line collapsing.
- `internal/tools/fuzzy/anchor.go` — `AnchorFind()` first+last line anchoring.
- `internal/tools/fuzzy/partial.go` — `PartialFind()` longest common substring (60% threshold).
- `internal/tools/fuzzy/chain_test.go` — 22 tests covering all passes and chain behavior.
- `internal/tools/edit_file.go` — `EditFileTool` implementing `Tool` + `ToolWithApproval`.
- `internal/tools/edit_file_test.go` — 10 tests covering edit, stale, ambiguous, diff, params, approval.

### Files Modified
- `internal/tools/tool.go` — added `NewEditFileTool()` to `BuiltinTools` slice.
- `internal/tools/tool_test.go` — updated count to 9, added `edit_file` to expected names.
- `internal/tools/registry_test.go` — updated count to 9, added `NewEditFileTool()` to manual registry.
- `internal/agent/executor/builtin.go` — `RegisterTools()` creates `EditFileTool` with shared tracker, count updated to 5.

### Roadmap
- [ROADMAP.md](../refactoring/opendev-gaps/ROADMAP.md) — R-4.2 marked Done.
