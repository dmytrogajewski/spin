# JOURNEY-unify-err-tool-not-found: Unify ErrToolNotFound Triplicate

## Roadmap Link
- Source roadmap: specs/ref/ROADMAP.md
- Feature: 1.2 — Unify ErrToolNotFound Triplicate
- Cluster: 6 (SPEC.md) | LIST.md findings 40, 42

## 1. Journey

When **a developer working on spin's tool execution pipeline** I want to **have a single canonical `ErrToolNotFound` sentinel error** so I can **use `errors.Is()` consistently across tools, MCP, and agent packages without worrying about three separate definitions with identical messages**.

## 2. CJM

Three packages independently define `ErrToolNotFound = errors.New("tool not found")`:
- `internal/tools/tool.go` (canonical — used by the tools registry)
- `internal/mcp/errors.go` (used by local, remote, and smithery registries)
- `internal/agent/tool_runtime.go` (used by the tool runtime executor)

All three have identical error messages. The duplication means `errors.Is(err, tools.ErrToolNotFound)` would fail to match errors wrapped around `mcp.ErrToolNotFound` or `agent.ErrToolNotFound`, even though they semantically represent the same condition.

### Phase 1: Discovery

**User Intent:** Understand which packages define and use `ErrToolNotFound`.

**Actions:** Grep for `ErrToolNotFound` across the codebase.

**Pain / Risk:**
1. Three definitions create semantic ambiguity — which one should callers check against?
2. `errors.Is()` checks can silently fail across package boundaries.

**Success Signal:** All three definitions identified with their usage sites.

### Phase 2: Unification

**User Intent:** Consolidate to a single sentinel in `internal/tools`.

**Actions:**
1. Remove `ErrToolNotFound` from `internal/mcp/errors.go`.
2. Remove `ErrToolNotFound` from `internal/agent/tool_runtime.go`.
3. Update `mcp` call-sites to use `tools.ErrToolNotFound` (already imports `tools`).
4. Update `agent` call-site to use `tools.ErrToolNotFound` (already imports `tools`).

**Pain / Risk:**
1. Import cycles — verified: both `mcp` and `agent` already import `tools`. No risk.
2. External callers referencing `mcp.ErrToolNotFound` or `agent.ErrToolNotFound` — verified: none exist.

**Success Signal:** Only `tools.ErrToolNotFound` exists; `mcp` and `agent` reference it.

### Phase 3: Verification

**User Intent:** Confirm no regressions.

**Actions:**
1. `go vet ./...` passes.
2. `go test ./internal/tools/... ./internal/mcp/... ./internal/agent/...` passes.
3. `grep -rn 'ErrToolNotFound' --include='*.go'` shows only `tools.ErrToolNotFound` definition and references.

**Pain / Risk:**
1. Test files might reference `agent.ErrToolNotFound` or `mcp.ErrToolNotFound` — verified: none do.

**Success Signal:** All tests green, single definition confirmed.

### Friction and Opportunity

| Friction | Phase | Opportunity |
|----------|-------|-------------|
| Three identical sentinels confuse `errors.Is()` | Discovery | Single source of truth |
| No external references to remove | Unification | Clean, safe migration |

### North Star Summary

A single `tools.ErrToolNotFound` sentinel serves as the canonical "tool not found" error across the entire spin codebase. Both `mcp` and `agent` packages reference this single sentinel, enabling correct `errors.Is()` behavior across package boundaries.

## 3. UX Implementation and Assessment

### Time to First Value
- [x] Immediate — removes semantic ambiguity in error checking

### Production-Ready Defaults
- [x] Single sentinel follows Go best practices for sentinel errors

### Golden Path Quality
- [x] `errors.Is(err, tools.ErrToolNotFound)` works regardless of which package produced the error

### Error Quality
- [x] Consistent error identity across the tool execution pipeline

### Failure Safety
- [x] No API changes — internal sentinel replacement only
- [x] Reversible via version control

## 4. Tests

### TC-01: tools.ErrToolNotFound is the only definition

**Given** the codebase has been updated.
**When** grepping for `ErrToolNotFound.*=.*errors.New` across all non-test `.go` files.
**Then** exactly 1 match exists, in `internal/tools/tool.go`.

### TC-02: mcp registries wrap tools.ErrToolNotFound

**Given** a tool lookup fails in an MCP registry.
**When** the returned error is checked with `errors.Is(err, tools.ErrToolNotFound)`.
**Then** it returns true.

### TC-03: agent tool runtime wraps tools.ErrToolNotFound

**Given** a tool call references a non-existent tool.
**When** the returned error is checked with `errors.Is(err, tools.ErrToolNotFound)`.
**Then** it returns true.

### TC-04: all affected package tests pass

**Given** the unification is complete.
**When** `go test ./internal/tools/... ./internal/mcp/... ./internal/agent/...` is run.
**Then** all tests pass with exit code 0.

## Implementation

- Modified: `internal/mcp/errors.go` — removed `ErrToolNotFound` definition
- Modified: `internal/mcp/registry_local.go` — uses `tools.ErrToolNotFound`
- Modified: `internal/mcp/registry_remote.go` — uses `tools.ErrToolNotFound`
- Modified: `internal/mcp/registry_smithery.go` — uses `tools.ErrToolNotFound`
- Modified: `internal/agent/tool_runtime.go` — removed `ErrToolNotFound`, uses `tools.ErrToolNotFound`
- Unchanged: `internal/tools/tool.go` — canonical definition remains
