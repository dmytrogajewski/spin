# Journey R-T4: Extract Path Resolution Helper

**Roadmap Item**: R-T4
**Spec**: [SPEC.md](../refactoring/tools-cleanup/SPEC.md) Section F-5
**Status**: Done

## Context

Four tools duplicated the same 3-line path resolution pattern:
```go
if !filepath.IsAbs(path) && t.workDir != "" {
    path = filepath.Join(t.workDir, path)
}
```

## User Journey

### Persona
Developer adding path security validation (e.g., preventing path traversal).

### Phases

| Phase | Action | Current Experience | Target Experience |
|-------|--------|--------------------|-------------------|
| Find | Locate path resolution logic | 4 copies in 4 files | One `resolvePath` function |
| Modify | Add security check | Must update 4 files | Update once in `path.go` |
| Test | Verify behavior | Hope all 4 copies agree | Unit test on `resolvePath` |

## Implementation

### Files Created
| File | Purpose |
|------|---------|
| `internal/tools/path.go` | `resolvePath(path, workDir string) string` helper |
| `internal/tools/path_test.go` | Table-driven tests: absolute, relative, empty workDir, empty path, both empty |

### Files Modified
| File | Change |
|------|--------|
| `internal/tools/read_file.go` | Inline logic → `resolvePath`; removed unused `path/filepath` import |
| `internal/tools/write_file.go` | Inline logic → `resolvePath` |
| `internal/tools/edit_file.go` | Inline logic → `resolvePath`; removed unused `path/filepath` import |
| `internal/tools/list_directory.go` | Inline logic → `resolvePath`; removed unused `path/filepath` import |

## Tests

- New: `TestResolvePath` with 5 table-driven cases (100% coverage of `resolvePath`)
- All existing tool tests pass unchanged
- `go vet ./internal/tools/...` clean
