# FRD-20251026000001: MCP Manager Migration to mcp-go SDK

**Feature ID**: FRD-20251026000001  
**Title**: Migrate MCP Manager to mark3labs/mcp-go SDK  
**Status**: In Progress  
**Created**: 2025-10-26  
**Roadmap Phase**: Phase 2.4 - MCP Manager  
**Priority**: P1

---

## 1. Overview

### 1.1 Purpose

Migrate the MCP (Model Context Protocol) manager from custom client implementation to the official `github.com/mark3labs/mcp-go` SDK to reduce code complexity, improve maintainability, and leverage battle-tested protocol handling.

### 1.2 Background

Current implementation (`internal/mcp/`):
- **Custom protocol implementation**: ~800+ lines of custom JSON-RPC and MCP protocol handling
- **Custom client**: `internal/mcp/client/` with stdio transport
- **Custom types**: `internal/mcp/types/` mirroring MCP protocol types
- **Interface{} usage**: `CallTool` arguments use `map[string]interface{}`

Problems:
- High maintenance burden keeping protocol in sync with MCP spec
- Custom JSON-RPC implementation increases surface area for bugs
- Duplication of effort (SDK already exists and is maintained)
- Interface{} usage violates roadmap goal of type-safe APIs

### 1.3 Goals

1. **Reduce Code Complexity**: Replace ~800 lines of custom code with SDK
2. **Eliminate interface{}**: Replace `map[string]interface{}` with type-safe `json.RawMessage`
3. **Improve Maintainability**: Leverage community-maintained SDK
4. **No Backward Compatibility**: Clean cutover (as per instructions)

### 1.4 Non-Goals

- Backward compatibility with old MCP types
- Support for transports other than stdio (keep current scope)
- Custom protocol extensions

---

## 2. Architecture

### 2.1 Current Architecture

```
internal/mcp/
├── manager.go                  # MCPManager, MCPTool, MCPToolWrapper
├── client/
│   ├── client.go              # Client interface
│   ├── stdio.go               # Custom stdio transport (~200 lines)
│   ├── error.go               # Error types
│   └── client_test.go
└── types/
    ├── protocol.go            # MCP core types
    ├── request.go             # Request types
    ├── response.go            # Response types
    ├── capabilities.go        # Capability types
    └── types_test.go
```

**Interface{} Usage** (Phase 2.4 Target):
- `manager.go:CallTool(ctx, toolName, arguments map[string]interface{})`
- `manager.go:MCPToolWrapper.Execute()` converts ToolParameters → map

### 2.2 Target Architecture

```
internal/mcp/
└── manager.go                  # MCPManager using mcp-go SDK
```

**Changes**:
1. **Delete** `internal/mcp/client/` - replaced by `github.com/mark3labs/mcp-go/client`
2. **Delete** `internal/mcp/types/` - replaced by `github.com/mark3labs/mcp-go/mcp`
3. **Simplify** `manager.go`:
   - Use `client.NewStdioMCPClient()` from SDK
   - Use `mcp.CallToolRequest` types
   - Replace `map[string]interface{}` with `json.RawMessage`
   - Direct mapping to SDK types

### 2.3 Type Mapping

| Current Type | SDK Type | Notes |
|-------------|----------|-------|
| `types.InitializeRequest` | `mcp.InitializeRequest` | Direct replacement |
| `types.InitializeResponse` | `mcp.InitializeResult` | SDK uses "Result" suffix |
| `types.Tool` | `mcp.Tool` | Compatible |
| `types.ListToolsResponse` | `mcp.ListToolsResult` | SDK uses "Result" suffix |
| `types.CallToolResponse` | `mcp.CallToolResult` | SDK uses "Result" suffix |
| `map[string]interface{}` | `json.RawMessage` | **Key change** - type-safe arguments |
| `client.Client` | `client.Client` (SDK) | Interface-compatible |

---

## 3. Detailed Design

### 3.1 MCPManager Refactoring

#### 3.1.1 Dependencies

```go
import (
    "github.com/mark3labs/mcp-go/client"
    "github.com/mark3labs/mcp-go/mcp"
    "github.com/dmytrogajewski/spin/internal/tools"
)
```

#### 3.1.2 Client Creation

**Before (Custom)**:
```go
// internal/mcp/client/stdio.go
mcpClient, err := client.NewStdioClient(client.Config{
    Command: serverConfig.Command,
    Args:    serverConfig.Args,
    Env:     serverConfig.Env,
    Timeout: 30 * time.Second,
})
```

**After (SDK)**:
```go
// Using mcp-go SDK
env := make([]string, 0, len(serverConfig.Env))
for k, v := range serverConfig.Env {
    env = append(env, fmt.Sprintf("%s=%s", k, v))
}

mcpClient := client.NewStdioMCPClient(
    serverConfig.Command,
    env,
    serverConfig.Args...,
)

// Must call Start() before use
if err := mcpClient.Start(ctx); err != nil {
    return err
}
```

#### 3.1.3 Initialization

**Before**:
```go
initReq := types.InitializeRequest{
    ProtocolVersion: "2024-11-05",
    Capabilities: types.ClientCapabilities{
        Tools: &types.ToolsCapability{ListChanged: true},
    },
    ClientInfo: types.Implementation{
        Name:    "spin",
        Version: "0.1.0",
    },
}
initResp, err := mcpClient.Initialize(ctx, initReq)
```

**After**:
```go
initReq := mcp.InitializeRequest{
    ProtocolVersion: "2024-11-05",
    Capabilities: mcp.ClientCapabilities{
        Tools: &mcp.ToolsCapability{ListChanged: true},
    },
    ClientInfo: mcp.Implementation{
        Name:    "spin",
        Version: "0.1.0",
    },
}
initResp, err := mcpClient.Initialize(ctx, initReq)
```

#### 3.1.4 Tool Listing

**Before**:
```go
toolsResp, err := mcpClient.ListTools(ctx)
for _, tool := range toolsResp.Tools {
    // register tool
}
```

**After**:
```go
// SDK requires explicit request struct
listReq := mcp.ListToolsRequest{}
toolsResp, err := mcpClient.ListTools(ctx, listReq)
for _, tool := range toolsResp.Tools {
    // register tool
}
```

#### 3.1.5 Tool Invocation (KEY CHANGE)

**Before (interface{})**:
```go
func (m *MCPManager) CallTool(
    ctx context.Context,
    toolName string,
    arguments map[string]interface{}, // ❌ interface{}
) (tools.ToolResult, error) {
    argsJSON, err := json.Marshal(arguments)
    if err != nil {
        return tools.ToolResult{}, err
    }
    resp, err := mcpTool.Client.CallTool(ctx, mcpTool.Tool.Name, argsJSON)
    // ...
}
```

**After (json.RawMessage)** ✅:
```go
func (m *MCPManager) CallTool(
    ctx context.Context,
    toolName string,
    arguments json.RawMessage, // ✅ Type-safe
) (tools.ToolResult, error) {
    req := mcp.CallToolRequest{
        Name:      mcpTool.Tool.Name,
        Arguments: arguments,
    }
    resp, err := mcpTool.Client.CallTool(ctx, req)
    // ...
}
```

#### 3.1.6 MCPToolWrapper Integration

**Before**:
```go
func (w *MCPToolWrapper) Execute(
    ctx context.Context,
    params tools.ToolParameters,
) (tools.ToolResult, error) {
    paramsMap := params.ToMap() // Convert to map[string]interface{}
    return w.manager.CallTool(ctx, w.name, paramsMap)
}
```

**After**:
```go
func (w *MCPToolWrapper) Execute(
    ctx context.Context,
    params tools.ToolParameters,
) (tools.ToolResult, error) {
    // ToolParameters already stores json.RawMessage internally
    argsJSON, err := json.Marshal(params.ToMap())
    if err != nil {
        return tools.ToolResult{
            Success: false,
            Error:   fmt.Sprintf("marshal arguments: %v", err),
        }, nil
    }
    return w.manager.CallTool(ctx, w.name, argsJSON)
}
```

### 3.2 Error Handling

**SDK Errors**:
- SDK returns standard Go errors
- Check for OAuth errors: `client.IsOAuthAuthorizationRequiredError(err)`
- Transport errors wrapped by SDK

**Our Error Mapping**:
```go
func mapSDKError(err error) tools.ToolResult {
    return tools.ToolResult{
        Success: false,
        Error:   err.Error(),
    }
}
```

### 3.3 Client Lifecycle

**SDK Requirements**:
1. Create client: `client.NewStdioMCPClient(...)`
2. Start transport: `client.Start(ctx)` ← **Must call before use**
3. Initialize: `client.Initialize(ctx, req)`
4. Use client: `ListTools()`, `CallTool()`, etc.
5. Close: `client.Close()`

**Manager Responsibilities**:
- Call `Start()` in `connectServer()`
- Call `Close()` in manager's `Close()`
- Track client state (started, initialized)

---

## 4. Implementation Plan (TDD Micro-Steps)

### 4.1 Phase 1: Add SDK Dependency

**Goal**: Add mcp-go SDK to project

**Steps**:
1. Run: `go get github.com/mark3labs/mcp-go@latest`
2. Verify: `go mod tidy`
3. Check version in `go.mod`

**Test**: Build succeeds with new dependency

### 4.2 Phase 2: Update Manager Signature (RED→GREEN)

**Micro-Step 1**: Update CallTool signature

**Test-RED**:
```go
// manager_test.go
func TestCallTool_AcceptsJSONRawMessage(t *testing.T) {
    // Create manager with mock client
    mgr := NewMCPManager(...)
    
    // Raw JSON arguments
    args := json.RawMessage(`{"path": "/test"}`)
    
    result, err := mgr.CallTool(ctx, "test_tool", args)
    require.NoError(t, err)
    assert.True(t, result.Success)
}
```

**Code-GREEN**:
```go
// manager.go
func (m *MCPManager) CallTool(
    ctx context.Context,
    toolName string,
    arguments json.RawMessage, // Changed from map[string]interface{}
) (tools.ToolResult, error) {
    // Implementation updated to use json.RawMessage
}
```

**Reflect**:
- Breaks MCPToolWrapper.Execute() - expected, fix next
- Signature now type-safe ✓
- No accidental behavior added ✓

### 4.3 Phase 3: Replace Client Creation (RED→GREEN)

**Micro-Step 2**: Use SDK client instead of custom

**Test-RED**:
```go
func TestConnectServer_UsesSDKClient(t *testing.T) {
    mgr := NewMCPManager(...)
    cfg := MCPServerConfig{
        Name:    "test",
        Command: "echo",
        Args:    []string{"hello"},
        Env:     map[string]string{"TEST": "1"},
    }
    
    err := mgr.connectServer(ctx, cfg)
    require.NoError(t, err)
    
    // Verify client is SDK client type
    client := mgr.clients["test"]
    _, ok := client.(client.Client) // SDK type
    assert.True(t, ok)
}
```

**Code-GREEN**:
```go
func (m *MCPManager) createClient(config MCPServerConfig) (client.Client, error) {
    env := make([]string, 0, len(config.Env))
    for k, v := range config.Env {
        env = append(env, fmt.Sprintf("%s=%s", k, v))
    }
    
    return client.NewStdioMCPClient(
        config.Command,
        env,
        config.Args...,
    ), nil
}
```

### 4.4 Phase 4: Update Initialization (RED→GREEN)

**Micro-Step 3**: Use SDK types for initialization

**Test-RED**:
```go
func TestInitialize_UsesSDKTypes(t *testing.T) {
    // Test uses mcp.InitializeRequest (SDK type)
}
```

**Code-GREEN**:
```go
// Replace types.InitializeRequest with mcp.InitializeRequest
initReq := mcp.InitializeRequest{
    ProtocolVersion: "2024-11-05",
    Capabilities: mcp.ClientCapabilities{...},
    ClientInfo: mcp.Implementation{...},
}
```

### 4.5 Phase 5: Update Tool Operations (RED→GREEN)

**Micro-Step 4**: Use SDK for ListTools

**Micro-Step 5**: Use SDK for CallTool

### 4.6 Phase 6: Delete Custom Code

**Files to Delete**:
- `internal/mcp/client/client.go`
- `internal/mcp/client/stdio.go`
- `internal/mcp/client/error.go`
- `internal/mcp/client/client_test.go`
- `internal/mcp/types/` (entire directory)

**Test**: All tests pass without deleted files

---

## 5. Testing Strategy

### 5.1 Test Coverage Target

**Goal**: 90%+ coverage for `internal/mcp/manager.go`

### 5.2 Test Structure

```
internal/mcp/
├── manager.go
└── manager_test.go  (NEW - comprehensive tests)
```

### 5.3 Test Cases

#### 5.3.1 Unit Tests (with mocks)

1. **Manager Creation**
   - `TestNewMCPManager`
   - `TestMCPManager_Initialize_Disabled`
   - `TestMCPManager_Initialize_NoServers`

2. **Server Connection**
   - `TestConnectServer_Success`
   - `TestConnectServer_InvalidCommand`
   - `TestConnectServer_InitializeFails`
   - `TestConnectServer_ListToolsFails`

3. **Tool Operations**
   - `TestCallTool_Success`
   - `TestCallTool_NotFound`
   - `TestCallTool_InvalidArguments`
   - `TestCallTool_SDKError`

4. **Tool Wrapper**
   - `TestMCPToolWrapper_Execute`
   - `TestMCPToolWrapper_Schema`
   - `TestGetTools_ReturnsAllTools`

5. **Lifecycle**
   - `TestClose_ClosesAllClients`
   - `TestClose_Idempotent`
   - `TestIsConnected`
   - `TestGetConnectedServers`

#### 5.3.2 Integration Tests (optional)

- Test with real `echo` command (simple stdio server)
- Verify actual MCP protocol flow

### 5.4 Mocking Strategy

**Mock SDK Client**:
```go
type mockMCPClient struct {
    startFunc      func(context.Context) error
    initializeFunc func(context.Context, mcp.InitializeRequest) (*mcp.InitializeResult, error)
    listToolsFunc  func(context.Context, mcp.ListToolsRequest) (*mcp.ListToolsResult, error)
    callToolFunc   func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error)
    closeFunc      func() error
}
```

**Inject mock via factory**:
```go
type clientFactory func(MCPServerConfig) (client.Client, error)

type MCPManager struct {
    // ...
    clientFactory clientFactory // for testing
}
```

---

## 6. Migration Impact

### 6.1 Breaking Changes

**Public API Changes**:
```go
// BEFORE
func (m *MCPManager) CallTool(
    ctx context.Context,
    toolName string,
    arguments map[string]interface{}, // ❌
) (tools.ToolResult, error)

// AFTER
func (m *MCPManager) CallTool(
    ctx context.Context,
    toolName string,
    arguments json.RawMessage, // ✅
) (tools.ToolResult, error)
```

**Impact**: Any code calling `MCPManager.CallTool()` directly needs update
- **Search**: `grep -r "\.CallTool(" --include="*.go"`
- **Expected**: Only `MCPToolWrapper.Execute()` calls it

### 6.2 Dependency Changes

**Added**:
- `github.com/mark3labs/mcp-go` (latest version)

**Removed**:
- All custom `internal/mcp/client/` code
- All custom `internal/mcp/types/` code

**Code Reduction**:
- Before: ~800 lines (manager + client + types)
- After: ~400 lines (manager only)
- **Reduction**: ~50% less code

### 6.3 Files Deleted

```
internal/mcp/client/client.go          (~80 lines)
internal/mcp/client/stdio.go           (~200 lines)
internal/mcp/client/error.go           (~40 lines)
internal/mcp/client/client_test.go     (~150 lines)
internal/mcp/types/protocol.go         (~100 lines)
internal/mcp/types/request.go          (~80 lines)
internal/mcp/types/response.go         (~80 lines)
internal/mcp/types/capabilities.go     (~60 lines)
internal/mcp/types/types_test.go       (~30 lines)
```

**Total**: ~820 lines deleted

---

## 7. Success Criteria

### 7.1 Functional Requirements

- ✅ MCP manager connects to configured servers
- ✅ Tools are discovered and registered
- ✅ Tool invocation works with `json.RawMessage` arguments
- ✅ Multiple servers supported concurrently
- ✅ Clean shutdown closes all connections

### 7.2 Code Quality

- ✅ No `interface{}` usage (replaced with `json.RawMessage`)
- ✅ 90%+ test coverage for `manager.go`
- ✅ Zero deadcode warnings
- ✅ `make lint` passes without errors
- ✅ All tests pass

### 7.3 Documentation

- ✅ Update `docs/packages/mcp.md` with SDK references
- ✅ Update roadmap Phase 2.4 to completed
- ✅ FRD document created

### 7.4 Code Reduction

- ✅ ~400 lines of custom code removed
- ✅ `internal/mcp/client/` deleted
- ✅ `internal/mcp/types/` deleted

---

## 8. Risks and Mitigations

### 8.1 SDK Maturity

**Risk**: mcp-go SDK is relatively new (2025)

**Mitigation**:
- SDK is used by 1,266 projects (per pkg.go.dev)
- Maintained by mark3labs (reputable)
- Can fork if needed (open source)

### 8.2 Protocol Compatibility

**Risk**: SDK may not support all MCP features we use

**Mitigation**:
- We only use: Initialize, ListTools, CallTool
- These are core MCP features, fully supported
- SDK supports Resources and Prompts (unused for now)

### 8.3 Performance

**Risk**: SDK overhead compared to custom implementation

**Mitigation**:
- MCP is not a hot path (tool discovery is startup)
- SDK handles stdio transport efficiently
- JSON marshaling is same overhead

### 8.4 Testing Complexity

**Risk**: Mocking SDK client may be harder

**Mitigation**:
- SDK client implements clear interface
- Use factory pattern for injecting mocks
- Integration tests with real `echo` command

---

## 9. Future Enhancements (Out of Scope)

1. **SSE Transport**: SDK supports HTTP/SSE for remote MCP servers
2. **OAuth Support**: SDK has built-in OAuth handling
3. **Resource APIs**: Implement ListResources, ReadResource
4. **Prompt APIs**: Implement ListPrompts, GetPrompt
5. **Notifications**: Handle server notifications (resource updates)
6. **Pagination**: Use SDK's pagination helpers for large tool lists

---

## 10. References

- **MCP Specification**: https://modelcontextprotocol.io/specification
- **mcp-go SDK**: https://github.com/mark3labs/mcp-go
- **SDK Documentation**: https://pkg.go.dev/github.com/mark3labs/mcp-go
- **Roadmap Phase 2.4**: `specs/ifacesroadmap.md#2.4`
- **Current MCP Implementation**: `internal/mcp/`

---

## 11. Approval

**Author**: Claude (AI Assistant)  
**Reviewer**: (Pending)  
**Status**: Ready for Implementation  
**Date**: 2025-10-26
