# JOURNEY-R-REF-11: Extract Generic Pattern Detection to pkg/alg/search

**Roadmap Item:** R-REF-11
**Spec:** [specs/ref/SPEC.md](../ref/SPEC.md) — Cluster 5: Pattern/Cycle Detection

## Summary

Add 2 generic pattern detection functions to `pkg/alg/search`:

1. **`DetectRepeat[T any](items []T, eq func(T, T) bool) bool`** — returns true if all items are equal according to eq. True for len <= 1 (vacuous truth).
2. **`DetectAlternating[T comparable](items []T) bool`** — returns true if items follow A→B→A→B pattern. Requires len >= 4.

## Design Decisions

- `DetectRepeat` takes `eq func(T, T) bool` rather than requiring `comparable`, since cycle detection compares by extracted keys (tool name, error string), not whole structs.
- `DetectAlternating` uses `comparable` since it compares items directly. Callers extract strings first.
- No `window` parameter on `DetectRepeat` — callers already slice with `collections.TailN`. Keep concerns separated.

## Acceptance Criteria

- [ ] 2 functions added to `pkg/alg/search/pattern.go`
- [ ] Table-driven tests
- [ ] `go test ./pkg/alg/search/...` passes
- [ ] No new lint issues

## Implementation

- **Created:** `pkg/alg/search/pattern.go`
- **Created:** `pkg/alg/search/pattern_test.go`
