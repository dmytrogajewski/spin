# JOURNEY-4.1: LSP Integration

## Overview

| Field | Value |
|-------|-------|
| Journey ID | 4.1 |
| Title | Wire LSP Manager and Symbol Tools |
| User Story | As a developer, the agent navigates code via find_symbol, find_references, rename_symbol backed by language servers. |
| Paper Section | 2.4 — LSP tools |
| Roadmap Item | JOURNEY-4.1 (42 functions) |

## Phases

### Phase 1: Discovery
- `lsp.Manager` manages per-language server instances with lazy init
- `lsp.Server` wraps JSON-RPC transport for LSP protocol
- 3 tool implementations: FindSymbolTool, FindReferencesTool, RenameSymbolTool
- All have unit tests. Never wired.

### Phase 2: Integration
- Create `lsp.NewManager()` in builder.go (with stub factory — real LSP server factory is complex)
- Register 3 LSP tools in `registerIntegrationTools()` with function deps from Manager
- Store Manager on Builder for lifecycle management

## Implementation

### Files Created
- `internal/lsp/factory.go` — `DefaultServerFactory` creates LSP servers via StdioTransport

### Files Modified
- `internal/conversation/builder.go` — Create `lsp.NewManager()` in Build(), pass to Conversation
- `internal/conversation/tools.go` — `registerLSPTools()` with DefinitionFinder/ReferenceFinder/SymbolRenamer closures
- `internal/conversation/conversation.go` — Added `lspManager` field, Close calls Manager.Close
- `internal/lsp/server.go` — Added `cache *Cache` field, `SearchSymbols()` method, DidOpen/DidChange/cache integration
- `internal/lsp/manager.go` — Use `Server.Language()`, `SetAlive(false)` on crash detection
