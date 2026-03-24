# JOURNEY-R-REF-12: Migrate Cycle Detection to Use Generic Patterns

**Roadmap Item:** R-REF-12
**Spec:** [specs/ref/SPEC.md](../ref/SPEC.md) — Cluster 5: Pattern/Cycle Detection

## Summary

Refactor `cycle/detector.go` and `cycle/patterns.go` to delegate to `search.DetectRepeat` and `search.DetectAlternating`:

- `allToolsAreSame` → `search.DetectRepeat(recent, snapshotsSameTool)`
- `allErrorsAreSame` → `search.DetectRepeat(recent, func(...) { a.Error == b.Error })`
- `detectOscillatingTools` → extract tool names + `search.DetectAlternating(toolNames)`

Removed `snapshotUsesTool` method (replaced by standalone `snapshotsSameTool` function).

## Acceptance Criteria

- [x] `detector.go` uses `search.DetectRepeat` for both tool and error checks
- [x] `patterns.go` uses `search.DetectAlternating`
- [x] `snapshotUsesTool` removed, replaced by `snapshotsSameTool`
- [x] `go test ./internal/cycle/...` passes
- [x] No new lint issues

## Implementation

- **Modified:** `internal/cycle/detector.go` — `allToolsAreSame` and `allErrorsAreSame` delegate to `search.DetectRepeat`
- **Modified:** `internal/cycle/patterns.go` — `detectOscillatingTools` delegates to `search.DetectAlternating`
