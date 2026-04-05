# JOURNEY-R-REF-2: Inline Path Wrapper

**Roadmap Item:** R-REF-2
**Spec:** [specs/ref/SPEC.md](../ref/SPEC.md) — Cluster 1: String Wrappers & Duplicates

## Summary

Remove the `resolvePath` wrapper function from `internal/tools/path.go` and replace 4 callers with direct `pathx.ResolvePath` calls. Delete `path_test.go` (tests `pathx.ResolvePath` behavior, already covered in `pkg/alg/pathx/`).

## User Journey

**Actor:** Developer maintaining the `tools` package.

### Phase 1: Discovery
- Developer encounters `resolvePath(path, workDir)` in `path.go`.
- It's a single-line wrapper around `pathx.ResolvePath(workDir, path)` with swapped arguments.
- **Friction:** Hides which library function is used; argument order swap is confusing.

### Phase 2: Resolution
- Replace 4 call sites with `pathx.ResolvePath(t.workDir, path)`.
- Add `pathx` import to 4 tool files.
- Delete `path.go` and `path_test.go`.

### Phase 3: Validation
- `go build ./...` passes.
- `go test ./internal/tools/...` passes.
- No new lint issues.

## Acceptance Criteria

- [x] `resolvePath` deleted from `path.go`
- [x] `path.go` deleted
- [x] `path_test.go` deleted
- [x] 4 tool files call `pathx.ResolvePath` directly
- [x] `go build ./...` passes
- [x] `go test ./internal/tools/...` passes

## Implementation

- **Deleted:** `internal/tools/path.go`
- **Deleted:** `internal/tools/path_test.go`
- **Modified:** `internal/tools/read_file.go` — added `pathx` import, replaced call
- **Modified:** `internal/tools/edit_file.go` — added `pathx` import, replaced call
- **Modified:** `internal/tools/write_file.go` — added `pathx` import, replaced call
- **Modified:** `internal/tools/list_directory.go` — added `pathx` import, replaced call
