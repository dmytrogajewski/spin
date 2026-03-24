# JOURNEY-R-REF-8: Migrate Cycle Detection to collections.TailN

**Roadmap Item:** R-REF-8
**Spec:** [specs/ref/SPEC.md](../ref/SPEC.md) — Cluster 7: Generic Collection Operations

## Summary

Replace 3 private rolling-window helper functions in `internal/cycle/detector.go` with `collections.TailN` / `collections.TailNOrAll`:

- `getRecentSnapshots()` → `collections.TailNOrAll(d.history, d.config.WindowSize)` (WindowSize=0 means "all")
- `getRecentSnapshotsForToolCheck()` → `collections.TailN(d.history, d.config.ToolRepeatLimit)`
- `getRecentSnapshotsForErrorCheck()` → `collections.TailN(d.history, d.config.ErrorRepeatLimit)`

**Note:** `TailN` returns a copy (not a view). All usages are read-only so this is safe. `TailNOrAll` is used for the first case because `WindowSize=0` means "check all history".

## Acceptance Criteria

- [x] 3 helper function bodies replaced with `collections.TailN/TailNOrAll`
- [x] `go build ./...` passes
- [x] `go test ./internal/cycle/...` passes
- [x] No new lint issues

## Implementation

- **Modified:** `internal/cycle/detector.go` — simplified 3 functions to one-liners using `collections`
