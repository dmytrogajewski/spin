# JOURNEY-R8.2 — LSP Tool Handlers

**Status**: Done
**Roadmap**: specs/refactoring/opendev-gaps/ROADMAP.md → R-8.2

## User Journey

Agent needs to rename `processRequest` to `handleRequest` across the codebase. It calls `rename_symbol` which semantically renames all usages (not just text matches) across multiple files without touching strings or comments. Before that, the developer might use `find_symbol` to locate a function and `find_references` to see where it's used.

## Phases

### Phase 1: Matcher (internal/lsp/retriever.go)
- Concrete `Matcher` struct with `Match(name string) bool`.
- `ParseMatcher(pattern string) Matcher` factory — exact for plain names, prefix for `.` suffix, wildcard for `*`/`?` patterns.
- `FilterSymbols(symbols, matcher)` — filters symbol lists by matcher.
- Uses `strings.CutSuffix` (modernize) and `[filepath.Match]` (godoclint).

### Phase 2: FindSymbolTool
- Implements `tools.Tool` interface.
- Parameters: `name` (required), `file_path` (required — provides language context).
- Uses `DefinitionFinder` function type (no interface return — ireturn-clean).
- Read-only: safe in Plan Mode.

### Phase 3: FindReferencesTool
- Implements `tools.Tool` interface.
- Parameters: `file_path` (required), `line` (required), `character` (required).
- Uses `ReferenceFinder` function type, groups results by file.
- Uses `params.Has()` + `GetIntOr()` to avoid nilerr.
- Read-only: safe in Plan Mode.

### Phase 4: RenameSymbolTool
- Implements `tools.Tool` and `tools.ToolWithApproval`.
- Parameters: `file_path`, `line`, `character`, `new_name` (all required).
- Uses `SymbolRenamer` function type.
- `isValidIdentifier` via `go/token.IsIdentifier`.
- CheckApproval returns RiskHigh with reason.

### Phase 5: Event Type
- `EventLSPDiagnostics` (36) added to event enum.

## Friction Points

| Point | Risk | Mitigation |
|-------|------|------------|
| Language server not installed | Tool fails | Return clear error message, don't crash |
| Symbol not found | Empty results | Return "no symbols found" message |
| Rename across many files | High-risk operation | Require approval via ToolWithApproval |
| Unsupported language | .xyz files | Return ErrUnsupportedLanguage from DetectLanguage |
| ireturn linter | Flagged interface returns | Use function types instead of interfaces |
| nilerr linter | Error var in scope + nil return | Use Has() + GetIntOr() instead of GetInt() |

## Design Decisions

1. **Function-type DI**: `DefinitionFinder`, `ReferenceFinder`, `SymbolRenamer` are named function types — avoids ireturn issues that arise from interfaces. Tests use closures directly; production code wraps LSP Manager.
2. **Concrete Matcher**: Single `Matcher` struct with internal `matchMode` replaces `NameMatcher` interface + 3 concrete types. Eliminates ireturn on `ParseMatcher`.
3. **Has/GetIntOr pattern**: Avoids nilerr by never creating an error variable — `params.Has(key)` for existence, `params.GetIntOr(key, 0)` for value.
4. **RenameSymbolTool implements ToolWithApproval**: Multi-file rename is a high-risk operation.
5. **Read-only tools**: find_symbol and find_references are safe in Plan Mode.

## DoD

- [x] `internal/lsp/retriever.go` — Matcher struct, ParseMatcher, FilterSymbols
- [x] `internal/tools/find_symbol.go` — FindSymbolTool, DefinitionFinder function type
- [x] `internal/tools/find_references.go` — FindReferencesTool, ReferenceFinder function type
- [x] `internal/tools/rename_symbol.go` — RenameSymbolTool, SymbolRenamer, ToolWithApproval
- [x] `internal/events/event.go` — EventLSPDiagnostics (36)
- [x] Unit tests: 28 tests across 4 test files (≥90% coverage)
- [x] `go vet` and `make lint` clean (0 issues)

## Implementation

### Files Created/Modified
- `internal/lsp/retriever.go` — `Matcher` struct, `ParseMatcher()`, `FilterSymbols()`
- `internal/lsp/retriever_test.go` — 8 tests
- `internal/tools/find_symbol.go` — `FindSymbolTool`, `DefinitionFinder`
- `internal/tools/find_symbol_test.go` — 7 tests
- `internal/tools/find_references.go` — `FindReferencesTool`, `ReferenceFinder`
- `internal/tools/find_references_test.go` — 6 tests
- `internal/tools/rename_symbol.go` — `RenameSymbolTool`, `SymbolRenamer`, `ToolWithApproval`
- `internal/tools/rename_symbol_test.go` — 7 tests + compile-time check
- `internal/events/event.go` — Added `EventLSPDiagnostics` (36)
- `internal/events/event_test.go` — Added test case
- `.deadcode-whitelist` — Whitelisted Description/Schema methods
