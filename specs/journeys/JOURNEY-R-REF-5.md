# JOURNEY-R-REF-5: Inline filterEnvironment and filesearch Aliases

**Roadmap Item:** R-REF-5
**Spec:** [specs/ref/SPEC.md](../ref/SPEC.md) — Cluster 1: String Wrappers & Duplicates

## Summary

Two changes:

1. Remove `filterEnvironment` wrapper in `internal/agent/environment.go` — 1 caller replaced with direct `execx.FilterEnvironment` call.
2. Remove `filesearch/matcher.go` and `filesearch/ignore.go` — type aliases and wrapper functions for `pathx.Matcher`, `pathx.Match`, `pathx.IgnoreHandler`, `pathx.NewMatcher`, `pathx.NewIgnoreHandler`. Callers within the `filesearch` package now use `pathx` directly.

## Acceptance Criteria

- [x] `filterEnvironment` deleted; caller uses `execx.FilterEnvironment` directly
- [x] Test updated to use `execx.FilterEnvironment` with package-level filter lists
- [x] `filesearch/matcher.go` deleted
- [x] `filesearch/ignore.go` deleted
- [x] `searcher.go` uses `pathx.Matcher`, `pathx.Match`, `pathx.NewMatcher`
- [x] `scanner.go` uses `pathx.IgnoreHandler`, `pathx.NewIgnoreHandler`
- [x] `doc.go` updated
- [x] `go build ./...` passes
- [x] `go test ./internal/agent/... ./internal/filesearch/...` passes
- [x] No new lint issues

## Implementation

- **Modified:** `internal/agent/environment.go` — inlined `filterEnvironment`
- **Modified:** `internal/agent/environment_test.go` — updated test to use `execx.FilterEnvironment`
- **Deleted:** `internal/filesearch/matcher.go`
- **Deleted:** `internal/filesearch/ignore.go`
- **Modified:** `internal/filesearch/searcher.go` — uses `pathx` directly
- **Modified:** `internal/filesearch/scanner.go` — uses `pathx` directly
- **Modified:** `internal/filesearch/doc.go` — updated docs
