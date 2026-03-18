# JOURNEY-RT6 — Move Registry Validation to Package Functions

**Status**: Done
**Roadmap**: specs/refactoring/tools-cleanup/ROADMAP.md -> R-T6

## User Journey

Seven validation methods on `*Registry` never access any Registry field — they are pure functions. Converting them to package-level functions clarifies that validation has no side effects.

## DoD

- [x] All 7 methods converted to package-level functions.
- [x] All call sites updated (8 `r.` -> direct calls).
- [x] Tests pass unchanged.
- [x] `make lint` 0 issues.

## Implementation

### Files Modified
- `internal/tools/registry.go` — removed `(r *Registry)` receiver from 7 methods; updated 8 internal call sites from `r.method()` to `method()`.
