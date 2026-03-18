# JOURNEY-CTX-2.5 — SmitheryRegistry.Close Cleans Up Loaded Servers

**Status**: Done
**Roadmap**: specs/ctx/ROADMAP.md -> 2.5
**Spec**: specs/ctx/SPEC.md -> CTX-006

## User Journey

User loads multiple Smithery MCP servers dynamically via `LoadServer`. On shutdown, `Close()` only closes the static client. All dynamically loaded `RemoteRegistry` instances leak connections and goroutines. After this change, `Close()` iterates all loaded servers and closes each one.

## Design Decisions

1. **errors.Join for multi-error**: Collect close errors from all servers and the static client, return joined error.
2. **Clear the map**: After closing, clear `loadedServers` to prevent double-close on repeat calls.

## DoD

- [x] `Close()` iterates loadedServers and calls `Close()` on each.
- [x] Errors from individual closes are collected and returned via `errors.Join`.
- [x] Map cleared after close to prevent double-close.
- [x] `go vet ./...` clean.
- [x] `make lint` clean.
- [x] All tests pass.

## Implementation

### Files Modified
- `internal/mcp/registry_smithery.go` — `Close()` now iterates `loadedServers`, closes each `RemoteRegistry`, collects errors via `errors.Join`, and clears the map. Added `errors` import.
