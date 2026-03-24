# JOURNEY-R-REF-6: Inline capOutput Wrapper

**Roadmap Item:** R-REF-6
**Spec:** [specs/ref/SPEC.md](../ref/SPEC.md) — Cluster 1: String Wrappers & Duplicates

## Summary

`capOutput` truncates to maxChars and appends a suffix beyond the limit. `stringsx.TruncateWithEllipsis` truncates within the limit with `"..."`. Since semantics differ, added `stringsx.TruncateWithSuffix(input, maxLen, suffix)` which matches `capOutput` semantics exactly, then replaced the caller.

## Acceptance Criteria

- [x] `stringsx.TruncateWithSuffix` added with tests
- [x] `capOutput` removed from `web_fetch.go`
- [x] All tests pass
- [x] No new lint issues

## Implementation

- **Modified:** `pkg/alg/stringsx/extended.go` — added `TruncateWithSuffix`
- **Modified:** `pkg/alg/stringsx/extended_test.go` — added 4 test cases
- **Modified:** `internal/tools/web_fetch.go` — replaced `capOutput` with `stringsx.TruncateWithSuffix`
