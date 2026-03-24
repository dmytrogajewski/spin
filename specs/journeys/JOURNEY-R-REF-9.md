# JOURNEY-R-REF-9: Add Filter, Clamp, ValidateAll to collections

**Roadmap Item:** R-REF-9
**Spec:** [specs/ref/SPEC.md](../ref/SPEC.md) — Cluster 7: Generic Collection Operations

## Summary

Add 3 generic utility functions to `pkg/alg/collections`:

1. **`Filter[T any](items []T, pred func(T) bool) []T`** — returns elements where predicate is true. Returns nil for nil/empty input.
2. **`Clamp[T cmp.Ordered](val, lo, hi T) T`** — clamps val to [lo, hi] range. Uses `cmp.Ordered` constraint (Go 1.21+).
3. **`ValidateAll[T any](items []T, validate func(T) error) error`** — runs validate on each item, collects non-nil errors, returns `errors.Join`.

## Design Decisions

- `Filter` returns nil (not empty slice) for nil input — consistent with `TailN` pattern.
- `Clamp` uses `cmp.Ordered` from stdlib (not `constraints.Ordered` from x/exp) for zero-dependency.
- `ValidateAll` uses `errors.Join` (Go 1.20+) — no custom error types needed.
- All functions are in `extended.go` to match existing pattern.

## Acceptance Criteria

- [x] 3 functions added to `pkg/alg/collections/extended.go`
- [x] Table-driven tests with edge cases (15 tests, 100% coverage)
- [x] `go test ./pkg/alg/collections/...` passes
- [x] `go vet ./...` clean
- [x] `make lint` — no new issues

## Implementation

- **Modified:** `pkg/alg/collections/extended.go` — added Filter, Clamp, ValidateAll
- **Modified:** `pkg/alg/collections/genutil_test.go` — added tests
