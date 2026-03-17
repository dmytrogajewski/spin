# JOURNEY-R8.4 — Browser Screenshot & Open Tools

**Status**: Done
**Roadmap**: specs/refactoring/opendev-gaps/ROADMAP.md → R-8.4

## User Journey

Agent generates an HTML page and wants to verify it visually. It calls `capture_web_screenshot` to take a headless screenshot, then the user reviews the image. Alternatively, the agent calls `open_browser` to open a URL in the user's default browser for interactive viewing.

## Phases

### Phase 1: ScreenshotTool (internal/tools/web_screenshot.go)
- Implements `tools.Tool` interface.
- Parameters: `url` (required), `width` (optional, default 1920), `height` (optional, default 1080), `full_page` (optional, default false).
- `ScreenshotCapturer` function type: `func(ctx, url, width, height, fullPage) (filePath string, err error)` — avoids ireturn.
- Returns file path to saved PNG.
- Read-only: safe in Plan Mode.
- 180s timeout via context (enforced by caller).

### Phase 2: OpenBrowserTool (internal/tools/web_browser.go)
- Implements `tools.Tool` interface.
- Parameters: `url` (required).
- `BrowserOpener` function type: `func(ctx, url) error` — avoids ireturn.
- Auto-converts local file paths to `file://` URIs.
- Read-only: safe in Plan Mode.
- Platform-native opening delegated to `BrowserOpener` implementation (adapter layer handles xdg-open/open/cmd).

## Friction Points

| Point | Risk | Mitigation |
|-------|------|------------|
| Chrome not installed | Screenshot fails | Return clear error "headless browser not available" |
| Large viewport | Memory/timeout | Cap viewport at 3840x2160 |
| Invalid URL | Tool error | Validate URL scheme (http/https/file) |
| Local file path | UX confusion | Auto-convert to file:// URI |
| ireturn linter | Flagged interface returns | Use function types (ScreenshotCapturer, BrowserOpener) |
| dupl linter | Duplicate test bodies | Table-driven tests for success/error and viewport cases |

## Design Decisions

1. **Function-type DI**: `ScreenshotCapturer` and `BrowserOpener` are named function types. Tests use closures; production code wraps chromedp/rod and platform-native commands.
2. **No chromedp/rod dependency in tools package**: The tools package only defines the function type. The actual headless browser integration lives in the adapter layer, injected at construction time.
3. **Read-only tools**: Both tools are safe in Plan Mode — no file mutations (screenshot writes to temp dir but is read-only from agent perspective), no approvals needed.
4. **Local path to file:// conversion**: `open_browser` converts absolute paths starting with `/` to `file:///` URIs for cross-platform compatibility.
5. **Viewport validation**: Width and height are capped at reasonable maximums to prevent resource exhaustion.

## DoD

- [x] `internal/tools/web_screenshot.go` — ScreenshotTool with ScreenshotCapturer function type
- [x] `internal/tools/web_browser.go` — OpenBrowserTool with BrowserOpener function type
- [x] Unit tests (>=90% coverage): 7 + 8 = 15 tests
- [x] `go vet` and `make lint` clean (0 issues)

## Implementation

- `internal/tools/web_screenshot.go` — `ScreenshotTool`, `ScreenshotCapturer` function type
- `internal/tools/web_browser.go` — `OpenBrowserTool`, `BrowserOpener` function type
- `internal/tools/web_screenshot_test.go` — 7 tests
- `internal/tools/web_browser_test.go` — 8 tests
- `.deadcode-whitelist` — Added 5 entries for Description/Schema/screenshotProperties
