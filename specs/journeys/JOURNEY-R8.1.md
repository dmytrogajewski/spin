# JOURNEY-R8.1 — LSP Server Lifecycle Manager

**Status**: Done
**Roadmap**: specs/refactoring/opendev-gaps/ROADMAP.md → R-8.1

## User Journey

Developer asks "find all usages of HandleRequest". The agent calls `find_symbol` tool. The LSP manager detects `.go` file extension, lazily starts `gopls`, sends a `textDocument/definition` JSON-RPC request, and returns the symbol location. On subsequent requests for `.go` files, the running `gopls` instance is reused. If `gopls` crashes, the manager detects it and restarts on the next request.

## Phases

### Phase 1: Types
- `SymbolKind` enum: Function, Method, Variable, Constant, Type, Interface, Package, Field, Property.
- `Location` struct: URI, Range (start/end line+character).
- `Symbol` struct: Name, Kind, Location, ContainerName.
- `Reference` struct: Location, IsDefinition flag.
- `TextEdit` struct: Range, NewText.
- `WorkspaceEdit` struct: map of URI → []TextEdit.
- `Diagnostic` struct: Range, Severity, Source, Message.
- `DiagnosticSeverity` enum: Error, Warning, Information, Hint.

### Phase 2: Language Detection
- `LanguageConfig` struct: ID, Extensions, ServerCommand, ServerArgs, RootMarkers.
- `DetectLanguage(filePath)` returns LanguageConfig by file extension.
- 8+ language mappings: Go, Python, TypeScript, JavaScript, Rust, Java, C/C++, Ruby.
- Unknown extensions return `ErrUnsupportedLanguage`.

### Phase 3: JSON-RPC 2.0 Transport
- `Transport` interface: `Send(ctx, method, params) (json.RawMessage, error)`, `Notify(ctx, method, params) error`, `Close() error`.
- `StdioTransport` wraps `exec.Cmd` stdin/stdout with Content-Length framing.
- Two goroutines: writer (serializes requests), reader (demuxes responses by ID).
- Atomic request IDs via `sync/atomic`.
- Per-request timeout via context deadline.

### Phase 4: Server Lifecycle
- `Server` struct: transport, language config, initialized flag, root URI.
- `Initialize(ctx, rootURI)` sends `initialize` request, then `initialized` notification.
- `FindDefinition(ctx, uri, line, char)` sends `textDocument/definition`.
- `FindReferences(ctx, uri, line, char)` sends `textDocument/references`.
- `Rename(ctx, uri, line, char, newName)` sends `textDocument/rename`.
- `DidOpen(ctx, uri, languageID, text)` sends `textDocument/didOpen`.
- `DidChange(ctx, uri, version, text)` sends `textDocument/didChange`.
- `Shutdown(ctx)` sends `shutdown` then `exit` notification.
- `IsAlive()` checks if process is still running.

### Phase 5: Manager
- `Manager` struct: servers map (language → *Server), mu sync.RWMutex, rootURI.
- `ForFile(ctx, filePath)` — detect language, lazy-start with double-check locking.
- `Close()` — shutdown all servers gracefully.
- If language server binary not found in PATH, return `ErrServerNotFound`.

### Phase 6: Response Cache
- Two-level cache keyed by content hash (MD5).
- L1: raw JSON-RPC responses (fastest, avoids re-parsing).
- L2: processed symbols (avoids re-extraction).
- Cache invalidation on content change (different hash).
- Thread-safe via `sync.RWMutex`.

## Friction Points

| Point | Risk | Mitigation |
|-------|------|------------|
| Server not installed | gopls/rust-analyzer missing | Return descriptive `ErrServerNotFound`, don't crash |
| Server crash | Process dies mid-request | `IsAlive()` check before reuse; manager restarts on next `ForFile()` |
| Slow initialization | LSP handshake takes seconds | Initialize only on first use (lazy) |
| Response timeout | Server hangs | Per-request context with deadline |
| Concurrent requests | Multiple goroutines call same server | Transport serializes writes; response demux by ID |
| Content-Length framing | Off-by-one in header parsing | Strict `Content-Length: N\r\n\r\n` parsing with tests |

## Design Decisions

1. **Transport interface**: Decouples protocol from process management; enables mock transport in tests.
2. **Lazy start with double-check locking**: Avoids starting servers that are never needed.
3. **One server per language**: Matches LSP design — each server handles one language.
4. **Content-Length framing**: Standard LSP wire format over stdio.
5. **Atomic request IDs**: Lock-free, monotonically increasing.
6. **Two-level cache**: L1 avoids JSON unmarshal; L2 avoids symbol extraction. Both keyed by content hash.

## DoD

- [x] `internal/lsp/types.go` — LSP types: Symbol, Reference, WorkspaceEdit, Diagnostic, etc.
- [x] `internal/lsp/languages.go` — LanguageConfig, DetectLanguage(), 9 language mappings
- [x] `internal/lsp/transport.go` — Transport interface, StdioTransport with JSON-RPC 2.0
- [x] `internal/lsp/server.go` — Server lifecycle: Initialize, FindDefinition, FindReferences, Rename
- [x] `internal/lsp/manager.go` — Manager with ForFile() lazy start, Close()
- [x] `internal/lsp/cache.go` — Two-level response cache with content hash invalidation
- [x] Descriptive error when language server binary not found (`ErrServerNotFound`)
- [x] 43 unit tests with mock transport, race-detector clean
- [x] `go vet` and `make lint` clean

## Implementation

### Files Created
- `internal/lsp/types.go` — Types, enums, sentinel errors
- `internal/lsp/languages.go` — Language detection with extension index
- `internal/lsp/transport.go` — StdioTransport with Content-Length framing
- `internal/lsp/server.go` — Server lifecycle with map[string]any params
- `internal/lsp/manager.go` — Manager with lazy start and crash restart
- `internal/lsp/cache.go` — Two-level cache with MD5 content hash
- `internal/lsp/types_test.go` — Enum string tests
- `internal/lsp/languages_test.go` — Language detection tests
- `internal/lsp/transport_test.go` — Transport tests with mock JSON-RPC
- `internal/lsp/server_test.go` — Server tests with mock transport
- `internal/lsp/manager_test.go` — Manager tests with mock factory
- `internal/lsp/cache_test.go` — Cache L1/L2 tests

### Files Modified
- `specs/refactoring/opendev-gaps/ROADMAP.md` — R-8.1 marked Done
