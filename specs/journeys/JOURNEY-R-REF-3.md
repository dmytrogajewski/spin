# JOURNEY-R-REF-3: Inline Fuzzy String Helpers

**Roadmap Item:** R-REF-3
**Spec:** [specs/ref/SPEC.md](../ref/SPEC.md) — Cluster 1: String Wrappers & Duplicates

## Summary

Replace 2 of 3 private fuzzy helpers with `pkg/alg/stringsx` equivalents:
- `trimLines` (collapse.go) → `stringsx.TrimTrailingPerLine` — identical semantics
- `countNonBlankLines` (anchor.go) → `stringsx.CountLines` with non-blank predicate

**NOT replaced:** `collapseWhitespace` (whitespace.go) — semantics differ. Fuzzy version only collapses spaces/tabs, preserving newlines. `stringsx.CollapseWhitespace` also collapses newlines, which would break fuzzy matching.

## Semantic Analysis

| Function | Fuzzy behavior | stringsx behavior | Compatible? |
|----------|---------------|-------------------|-------------|
| `trimLines` | Split on `\n`, `TrimRight(line, " \t")`, join | Identical | Yes |
| `collapseWhitespace` | `[ \t]+` → single space (newlines preserved) | `strings.Fields` (newlines collapsed too) | **No** |
| `countNonBlankLines` | Count lines where `TrimSpace != ""` | `CountLines(input, predicate)` | Yes |

## Acceptance Criteria

- [x] `trimLines` removed from `collapse.go`; replaced with `stringsx.TrimTrailingPerLine`
- [x] `countNonBlankLines` removed from `anchor.go`; replaced with `stringsx.CountLines`
- [x] `collapseWhitespace` kept in `whitespace.go` (semantics differ)
- [x] `go test ./internal/tools/fuzzy/...` passes
- [x] `go test ./internal/tools/...` passes
- [x] No new lint issues

## Implementation

- **Modified:** `internal/tools/fuzzy/collapse.go` — replaced `trimLines` with `stringsx.TrimTrailingPerLine`, deleted function
- **Modified:** `internal/tools/fuzzy/anchor.go` — replaced `countNonBlankLines` with `stringsx.CountLines`, deleted function
