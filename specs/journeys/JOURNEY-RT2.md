# Journey R-T2: Deduplicate URL Validation

**Roadmap Item**: R-T2
**Spec**: [SPEC.md](../refactoring/tools-cleanup/SPEC.md) Section F-3
**Status**: Done

## Context

`web_fetch.go` defined `isValidURL(rawURL string) bool` and
`web_screenshot.go` defined `isScreenshotURL(rawURL string) bool`.
Both checked `strings.HasPrefix(rawURL, httpScheme) || strings.HasPrefix(rawURL, httpsScheme)`.
Identical implementations with different names.

## User Journey

### Persona
Developer adding a new web tool that needs URL validation.

### Phases

| Phase | Action | Current Experience | Target Experience |
|-------|--------|--------------------|-------------------|
| Discover | Search for URL validation | Two identical functions | One: `isValidURL` |
| Reuse | Call the validator | Must choose between copies | One clear choice |
| Maintain | Update URL scheme logic | Two functions to update | One function |

### Friction Points (Resolved)
1. **Name confusion**: eliminated — `isScreenshotURL` deleted.
2. **Discovery cost**: eliminated — single canonical `isValidURL`.

## Implementation

### Files Modified
| File | Change |
|------|--------|
| `internal/tools/web_screenshot.go` | Deleted `isScreenshotURL` function; caller changed to `isValidURL`; removed unused `strings` import |

## Tests

- All existing `web_screenshot_test.go` and `web_fetch_test.go` tests pass unchanged
- `go vet ./internal/tools/...` clean
- No new lint issues
