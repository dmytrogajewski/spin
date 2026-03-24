# JOURNEY-R-REF-1: Inline Truncation Wrappers

**Roadmap Item:** R-REF-1
**Spec:** [specs/ref/SPEC.md](../ref/SPEC.md) — Cluster 1: String Wrappers & Duplicates

## Summary

Remove the two pure pass-through wrapper functions `TruncateHeadTail` and `TruncateLines` from `internal/tools/truncate.go`. These are 1:1 delegations to `pkg/alg/stringsx` with zero added logic. Keep `TruncateOutput` which encapsulates domain constants and two-step composition.

## User Journey

**Actor:** Developer maintaining the `tools` package.

### Phase 1: Discovery
- Developer sees `TruncateHeadTail` and `TruncateLines` in `truncate.go`.
- Both are single-line pass-throughs to `stringsx`.
- **Friction:** Misleading — suggests tools-specific truncation logic exists when it doesn't.

### Phase 2: Resolution
- Delete the two pure wrappers.
- Delete their 9 test functions (they test `stringsx` behavior, not tools behavior).
- Keep `TruncateOutput` (domain composition with 5 constants).
- Keep all constants (they encode output policy).

### Phase 3: Validation
- `go build ./...` passes — no callers of the deleted functions.
- `go test ./internal/tools/...` passes — remaining tests cover `TruncateOutput`.
- `make lint` passes — no dead code introduced.

## UX Assessment

| Aspect | Before | After |
|--------|--------|-------|
| Exported API surface | 3 functions + 5 constants | 1 function + 5 constants |
| Indirection layers | 2 pure wrappers | 0 pure wrappers |
| Test coverage | Tests re-test stringsx | Tests cover composition only |

## Acceptance Criteria

- [ ] `TruncateHeadTail` deleted from `truncate.go`
- [ ] `TruncateLines` deleted from `truncate.go`
- [ ] 9 test functions for deleted wrappers removed
- [ ] `TruncateOutput` and its 2 tests preserved
- [ ] All 5 constants preserved
- [ ] `go build ./...` passes
- [ ] `go test ./internal/tools/...` passes
- [ ] `make lint` passes

## Implementation

- **Modified:** `internal/tools/truncate.go` — removed 2 exported functions
- **Modified:** `internal/tools/truncate_test.go` — removed 9 wrapper tests
