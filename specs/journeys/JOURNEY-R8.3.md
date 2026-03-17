# JOURNEY-R8.3 — Web Fetch & Search Tools

**Status**: Done
**Roadmap**: specs/refactoring/opendev-gaps/ROADMAP.md → R-8.3

## User Journey

Agent needs to check the API docs for a library. It calls `web_search` to find the right page, then `fetch_url` to read the content. The HTML is converted to markdown for context efficiency. Both tools are read-only and safe in Plan Mode.

## Phases

### Phase 1: HTML Converter (internal/tools/html_convert.go)
- `HTMLConverter` function type: `func(htmlContent []byte) string`.
- `ConvertHTML` default implementation using `golang.org/x/net/html`.
- Preserves structure: headings (# / ## / ###), paragraphs, lists (- items), links ([text](url)), code blocks.
- Strips scripts, styles, nav, footer, header elements.
- Collapses whitespace, trims output.

### Phase 2: FetchURLTool (internal/tools/web_fetch.go)
- Implements `tools.Tool` interface.
- Parameters: `url` (required).
- `PageFetcher` function type for HTTP GET — avoids ireturn.
- `FetchResponse` concrete struct: `StatusCode`, `ContentType`, `Body`.
- URL validation: must start with `http://` or `https://`.
- Content-Type check: blocks non-text downloads (binary, images, etc.).
- HTML → markdown via `HTMLConverter`.
- Plain text / JSON returned as-is.
- Output capped at 50,000 chars.
- Timeout: 30s via context.
- User-Agent: `SpinBot/1.0`.
- Read-only: safe in Plan Mode.

### Phase 3: WebSearchTool (internal/tools/web_search.go)
- Implements `tools.Tool` interface.
- Parameters: `query` (required), `domain` (optional filter).
- `WebSearcher` function type: `func(ctx, query, maxResults) ([]SearchResult, error)`.
- `SearchResult` concrete struct: `Title`, `URL`, `Snippet`.
- Formats results as numbered list with title, URL, and snippet.
- Returns up to 10 results.
- Read-only: safe in Plan Mode.

## Friction Points

| Point | Risk | Mitigation |
|-------|------|------------|
| Non-text Content-Type | Tool returns binary gibberish | Block non-text MIME types |
| Large HTML pages | Context overflow | Cap output at 50,000 chars |
| Invalid/malformed URLs | HTTP error | Validate URL scheme before fetch |
| Network timeout | Tool hangs | 30s context timeout |
| Search returns no results | Empty output | Return "No results found" message |
| ireturn linter | Flagged interface returns | Use function types (PageFetcher, WebSearcher) |
| forbidigo linter | fmt.Print forbidden | Use fmt.Fprintf to string builders only |
| revive confusing-naming | `truncateOutput` vs `TruncateOutput` | Renamed to `capOutput` |
| dupl linter | Duplicate test bodies | Table-driven tests |
| cyclop/gocyclo | `writeElementNode` complexity | Split into `shouldSkipElement` + `headingPrefix` + `writeNonHeadingElement` |

## Design Decisions

1. **Function-type DI**: `PageFetcher`, `HTMLConverter`, `WebSearcher` are named function types. Tests use closures; production code wraps `http.Client` and search backends.
2. **FetchResponse struct**: Concrete return type from `PageFetcher` — carries status code, content type, and body. No interface returns.
3. **SearchResult struct**: Concrete type with exported fields. No interface needed.
4. **HTML converter as function type**: Decouples conversion strategy from tool. Default uses `golang.org/x/net/html` tree walker.
5. **Read-only tools**: Both tools are safe in Plan Mode — no file mutations, no approvals needed.
6. **No external dependencies**: Uses `golang.org/x/net/html` (already an indirect dependency) and stdlib `net/http`. No new third-party libraries.
7. **Complexity management**: `writeElementNode` split into three functions to stay under cyclop/gocyclo limit of 15.

## DoD

- [x] `internal/tools/html_convert.go` — HTMLConverter function type, ConvertHTML implementation
- [x] `internal/tools/web_fetch.go` — FetchURLTool with PageFetcher, FetchResponse
- [x] `internal/tools/web_search.go` — WebSearchTool with WebSearcher, SearchResult
- [x] Unit tests (>=90% coverage): 15 + 8 + 9 = 32 tests
- [x] `go vet` and `make lint` clean (0 issues)

## Implementation

- `internal/tools/html_convert.go` — `HTMLConverter` function type, `ConvertHTML` tree walker
- `internal/tools/web_fetch.go` — `FetchURLTool`, `PageFetcher` function type, `FetchResponse`
- `internal/tools/web_search.go` — `WebSearchTool`, `WebSearcher` function type, `SearchResult`
- `internal/tools/html_convert_test.go` — 15 tests
- `internal/tools/web_fetch_test.go` — 8 tests
- `internal/tools/web_search_test.go` — 9 tests
- `.deadcode-whitelist` — Added 4 entries for Description/Schema methods
