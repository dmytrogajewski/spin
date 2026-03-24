# JOURNEY-R-REF-4: Inline TUI Mapper Helpers

**Roadmap Item:** R-REF-4
**Spec:** [specs/ref/SPEC.md](../ref/SPEC.md) — Cluster 1: String Wrappers & Duplicates

## Summary

Remove `extractString` and `extractIntValue` private helpers from `mapper.go`. Replace 12 call sites with direct `data.Parameters.GetStringOr` / `data.Parameters.GetIntOr` calls. Also removed unused `tools` import.

## Acceptance Criteria

- [x] `extractString` removed
- [x] `extractIntValue` removed
- [x] 12 call sites replaced with direct method calls
- [x] Unused `tools` import removed
- [x] `go build ./...` passes
- [x] `go test ./internal/tui/...` passes
- [x] No new lint issues

## Implementation

- **Modified:** `internal/tui/mapper.go` — replaced 12 call sites, deleted 2 functions, removed unused import
