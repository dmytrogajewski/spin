# JOURNEY-4.2: Web Tools

## Overview

| Field | Value |
|-------|-------|
| Journey ID | 4.2 |
| Title | Wire Web Tools (fetch, search, browser, screenshot) |
| User Story | As a developer, the agent can fetch URLs, search the web, open a browser, and take screenshots. |
| Paper Section | 2.4 — WebToolHandler |
| Roadmap Item | JOURNEY-4.2 (33 functions) |

## Phases

### Phase 1: Discovery
- 4 web tools fully implemented with function dependencies
- `html_convert.go` provides HTML-to-text conversion (18 functions)
- All have unit tests. Never wired.

### Phase 2: Integration
- Register all 4 tools in `registerIntegrationTools()`
- Provide HTTP-based PageFetcher with ConvertHTML
- Provide exec-based BrowserOpener
- Stub WebSearcher and ScreenshotCapturer (require external services)

## Implementation

### Files Modified
- `internal/conversation/tools.go` — Added `registerWebTools()` with HTTP PageFetcher, ConvertHTML, exec BrowserOpener, stub WebSearcher/ScreenshotCapturer
