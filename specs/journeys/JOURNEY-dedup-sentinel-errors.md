# Journey: Collapse Duplicate Sentinel Errors

**Roadmap Item:** 1.1 — Collapse Duplicate Sentinel Errors
**Cluster:** 3 (SPEC.md) | **Priority:** P0

## Problem

The `err113` linter requires sentinel errors to be package-level variables defined with `errors.New()`. When the same error message is needed at multiple return sites within a package, developers created numbered duplicates (`ErrFoo2`, `ErrFoo3`, …) to satisfy the linter. This leads to 55 unnecessary error variables across 16 packages.

## User Journey

**As a** spin developer,
**I want** each error message to have exactly one sentinel variable,
**So that** I can use `errors.Is()` reliably and reduce maintenance burden.

### Phases

1. **Discovery** — Scan all `Err.*[2-9]` variables in non-test `.go` files
2. **Analysis** — For each duplicate, identify the canonical (un-numbered) variable and all call-sites
3. **Migration** — Replace each duplicate usage with the canonical sentinel, adding `fmt.Errorf("context: %w", ErrCanonical)` where call-site context is needed
4. **Verification** — Run `go vet`, `make lint`, `go test ./...`

### Friction Points

- `err113` linter requires sentinel errors — cannot use bare `errors.New()` in return statements
- Some call-sites return the sentinel directly; others wrap it — must preserve wrapping semantics
- Test files may reference the numbered variables by name — must update those too

### Fix Pattern

**Before:**
```go
var (
    ErrKeyCannotBeEmpty  = errors.New("key cannot be empty")
    ErrKeyCannotBeEmpty2 = errors.New("key cannot be empty")
    ErrKeyCannotBeEmpty3 = errors.New("key cannot be empty")
)

func (s *Store) Get(key string) error {
    if key == "" { return ErrKeyCannotBeEmpty }
}
func (s *Store) Set(key string) error {
    if key == "" { return ErrKeyCannotBeEmpty2 }
}
func (s *Store) Delete(key string) error {
    if key == "" { return ErrKeyCannotBeEmpty3 }
}
```

**After:**
```go
var ErrKeyCannotBeEmpty = errors.New("key cannot be empty")

func (s *Store) Get(key string) error {
    if key == "" { return ErrKeyCannotBeEmpty }
}
func (s *Store) Set(key string) error {
    if key == "" { return ErrKeyCannotBeEmpty }
}
func (s *Store) Delete(key string) error {
    if key == "" { return ErrKeyCannotBeEmpty }
}
```

### Scope

55 duplicate error variables across 16 packages (excluding test files):

| Package | Duplicates | Canonical Error |
|---------|-----------|-----------------|
| `internal/git` | 14 | `ErrNoGitStatusAvailable`, `ErrNotAGitRepository` |
| `internal/ui/blocks` | 6 | `ErrFileIsRequired`, `ErrMetadataIsEmpty` |
| `internal/patchapply` | 6 | `ErrInvalidPath`, `ErrInvalidNewPath` |
| `internal/ace/curator` | 4 | `ErrDeltaApplierNotInitialized` |
| `internal/tools` | 4 | `ErrParameterNotFound` |
| `internal/conversation` | 4 | `ErrConversationNotFound`, `ErrHistoryStorageNotConfigured` |
| `internal/protocol/acp` | 3 | `ErrSessionNotFound` |
| `internal/memory` | 3 | `ErrNoPersistentStoreConfigured` |
| `internal/storage` | 3 | `ErrKeyCannotBeEmpty` |
| `cmd/spin/config.go` | 2 | `ErrValidationFailed`, `ErrNoConfigFile` |
| `internal/ace/adapter` | 2 | `ErrSessionNotFound` |
| `internal/agent/executor` | 2 | `ErrSessionWorkingDirectoryNotSet`, `ErrPathIsOutsideTheAllowedWorkspace` |
| `internal/llm/factory` | 1 | `ErrAuthenticationRequiredFor` |
| `internal/ace/playbook` | 1 | `ErrBulletCannotBeNil` |
| `internal/auth` | 1 | `ErrUnknownCredentialType` |
| `internal/shell` | 1 | `ErrExecutionFailed` |

### Tests

- All existing tests must continue to pass unchanged
- No new tests needed — this is a pure refactoring (behavior-preserving)
- Verification: `grep -rn 'Err.*[2-9] =' --include='*.go' | grep -v _test.go` returns 0 matches

## Implementation

**Status:** Complete

**Files modified (16 packages, ~55 duplicate sentinel variables removed):**
- `internal/git/integration.go` — Removed 14 duplicates (`ErrNoGitStatusAvailable2-6`, `ErrNotAGitRepository2-9`)
- `internal/ui/blocks/metadata.go` — Removed 6 duplicates (`ErrFileIsRequired2`, `ErrMetadataIsEmpty2-6`)
- `internal/patchapply/parser.go` — Removed 6 duplicates (`ErrInvalidPath2-6`, `ErrInvalidNewPath2`)
- `internal/ace/curator/curator.go` — Removed 4 duplicates (`ErrDeltaApplierNotInitialized2-5`)
- `internal/tools/parameters.go` — Removed 4 duplicates (`ErrParameterNotFound2-5`)
- `internal/conversation/manager.go` — Removed 4 duplicates (`ErrHistoryStorageNotConfigured2`, `ErrConversationNotFound2-4`)
- `internal/protocol/acp/agent.go` — Removed 3 duplicates (`ErrSessionNotFound2-4`)
- `internal/memory/handoff.go` — Removed 3 duplicates (`ErrNoPersistentStoreConfigured2-4`)
- `internal/storage/store.go` — Removed 3 duplicates (`ErrKeyCannotBeEmpty2-4`)
- `cmd/spin/config.go` — Removed 2 duplicates (`ErrValidationFailed2`, `ErrNoConfigFile2`)
- `internal/ace/adapter/adapter.go` — Removed 2 duplicates (`ErrSessionNotFound2-3`)
- `internal/agent/executor/acp_tools.go` — Removed 2 duplicates (`ErrSessionWorkingDirectoryNotSet2`, `ErrPathIsOutsideTheAllowedWorkspace2`)
- `internal/llm/factory/factory.go` — Removed 1 duplicate (`ErrAuthenticationRequiredFor2`)
- `internal/ace/playbook/playbook.go` — Removed 1 duplicate (`ErrBulletCannotBeNil2`)
- `internal/auth/auth.go` — Removed 1 duplicate (`ErrUnknownCredentialType2`)
- `internal/shell/context.go` — Removed 1 duplicate (`ErrExecutionFailed2`)

**Verification:**
- `go vet ./...` — Clean
- `make lint` — 0 issues
- `make deadcode` — No dead code found
- `go test ./...` — All tests pass
- `grep -rn 'Err.*[2-9] =' --include='*.go' | grep -v _test.go` — 0 matches
