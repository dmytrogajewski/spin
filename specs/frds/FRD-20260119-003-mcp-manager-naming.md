# FRD-20260119-003: MCP Manager Naming Consolidation

**Created:** 2026-01-19  
**Author:** Architecture Refactoring  
**Status:** Completed  
**Priority:** P1 (High)

## 1. Summary

Rename two conflicting `MCPManager` types to clearly distinguish their purposes:
- `internal/mcp/manager.go:MCPManager` → `MCPServerManager` (runtime server management)
- `internal/config/mcp_manager.go:MCPManager` → `MCPConfigStore` (configuration management)

## 2. Problem Statement

Two `MCPManager` types exist in the codebase serving completely different purposes:

| Location | Current Name | Purpose |
|----------|--------------|---------|
| `internal/mcp/manager.go:38` | `MCPManager` | Runtime MCP server connection management, tool registration, and tool execution |
| `internal/config/mcp_manager.go:19` | `MCPManager` | Configuration management for MCP servers (list, add, remove from config file) |

### Impact of Current State

1. **Confusing imports:** When both packages are imported, one must be aliased
2. **Naming collisions:** IDE autocomplete shows two `MCPManager` types
3. **Unclear responsibilities:** New developers cannot determine which type to use from name alone
4. **Maintenance burden:** Documentation must clarify which `MCPManager` is referenced

## 3. Proposed Solution

### 3.1 Rename Runtime Manager

**File:** `internal/mcp/manager.go`

```go
// Before
type MCPManager struct { ... }
func NewMCPManager(config *Config, logger *slog.Logger) *MCPManager

// After  
type MCPServerManager struct { ... }
func NewMCPServerManager(config *Config, logger *slog.Logger) *MCPServerManager
```

**Rationale:** The runtime manager manages MCP server connections (connect, initialize, call tools, close). "ServerManager" accurately describes managing multiple server instances.

### 3.2 Rename Config Manager

**File:** `internal/config/mcp_manager.go`

```go
// Before
type MCPManager struct { ... }
func NewMCPManager(loader *LoaderV2) *MCPManager

// After
type MCPConfigStore struct { ... }
func NewMCPConfigStore(loader *LoaderV2) *MCPConfigStore
```

**Rationale:** The config manager provides CRUD operations for MCP server configurations (list, get, add, remove). "ConfigStore" follows the pattern used elsewhere (e.g., `PolicyStore`) and accurately describes persistent configuration storage.

## 4. Files Affected

### 4.1 Primary Changes

| File | Change |
|------|--------|
| `internal/mcp/manager.go` | Rename `MCPManager` → `MCPServerManager`, rename constructor |
| `internal/config/mcp_manager.go` | Rename `MCPManager` → `MCPConfigStore`, rename constructor |

### 4.2 Usage Updates (mcp.MCPManager → mcp.MCPServerManager)

| File | Occurrences |
|------|-------------|
| `internal/mcp/service.go` | 2 |
| `internal/mcp/manager_test.go` | 2 |
| `internal/protocol/acp/agent.go` | 4 |
| `internal/protocol/acp/agent_test.go` | 7 |
| `internal/protocol/acp/initialize_test.go` | 6 |
| `internal/protocol/acp/session_mode_test.go` | 6 |
| `internal/protocol/acp/request_permission_test.go` | 8 |
| `internal/protocol/acp/cancel_test.go` | 4 |
| `internal/protocol/acp/command_integration_test.go` | 2 |
| `internal/protocol/acp/commands_test.go` | 2 |
| `internal/protocol/acp/user_message_test.go` | 1 |
| `internal/protocol/acp/notifications_integration_test.go` | 5 |
| `internal/protocol/acp/load_session_test.go` | 5 |
| `internal/protocol/acp/plan_notifications_test.go` | 1 |
| `internal/protocol/acp/plan_integration_test.go` | 2 |
| `internal/protocol/acp/prompt_test.go` | 6 |
| `internal/protocol/acp/new_session_test.go` | 5 |
| `cmd/spin/acp.go` | 1 |
| `tests/compliance/protocol_methods_test.go` | 1 |

### 4.3 Usage Updates (config.MCPManager → config.MCPConfigStore)

| File | Occurrences |
|------|-------------|
| `cmd/spin/mcp.go` | 4 |

### 4.4 Error Constant Update

| File | Change |
|------|--------|
| `internal/protocol/acp/agent.go` | Rename `ErrNilMCPManager` → `ErrNilMCPServerManager` |

## 5. Acceptance Criteria

1. **No duplicate type names:** No two types named `MCPManager` exist in codebase
2. **Clear naming:** Type names clearly indicate their purpose (server management vs config storage)
3. **All tests pass:** No regression in functionality
4. **No lint errors:** `make lint` passes
5. **No dead code:** All renamed types are used
6. **Documentation updated:** Architecture documentation reflects new names

## 6. Implementation Steps

### Phase 1: MCPServerManager (internal/mcp/)

1. Rename `MCPManager` struct to `MCPServerManager`
2. Rename `NewMCPManager` function to `NewMCPServerManager`
3. Update all method receivers from `*MCPManager` to `*MCPServerManager`
4. Update `MCPToolWrapper.manager` field type
5. Update `internal/mcp/service.go` to use new names
6. Update `internal/mcp/manager_test.go` to use new names

### Phase 2: MCPConfigStore (internal/config/)

1. Rename `MCPManager` struct to `MCPConfigStore`
2. Rename `NewMCPManager` function to `NewMCPConfigStore`
3. Update all method receivers from `*MCPManager` to `*MCPConfigStore`
4. Update `cmd/spin/mcp.go` to use new names

### Phase 3: Protocol Package Updates

1. Update `internal/protocol/acp/agent.go`:
   - Rename `ErrNilMCPManager` to `ErrNilMCPServerManager`
   - Update field type and constructor parameter
2. Update all test files in `internal/protocol/acp/`

### Phase 4: Command and Test Updates

1. Update `cmd/spin/acp.go`
2. Update `tests/compliance/protocol_methods_test.go`

### Phase 5: Verification

1. Run `make lint`
2. Run `go test ./...`
3. Run `uast parse {file} | herr analyze` on changed files

## 7. Risk Assessment

| Risk | Mitigation |
|------|------------|
| Breaking existing code | Compile-time errors will catch all usages |
| Missing usages | Use grep to find all occurrences before/after |
| Documentation drift | Update docs in same PR |

## 8. Non-Goals

- Changing the internal implementation of either manager
- Adding new functionality
- Changing method signatures (beyond receiver type)
- Refactoring the package structure

## 9. Testing Strategy

### Unit Tests
- Existing tests will be renamed but logic unchanged
- All tests must pass after rename

### Integration Tests  
- No new tests required (rename only)
- Existing integration tests validate functionality

### Verification Commands
```bash
# Ensure no MCPManager remains (except in specs/docs)
grep -r "MCPManager" --include="*.go" | grep -v "_test.go" | grep -v "specs/"

# Run all tests
go test ./...

# Run linter
make lint
```

## 10. References

- ROADMAP.md Section 2.1: Duplicate MCP Manager Naming
- `internal/mcp/manager.go` - Runtime manager implementation
- `internal/config/mcp_manager.go` - Config manager implementation
