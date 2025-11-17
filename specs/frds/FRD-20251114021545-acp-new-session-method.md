# FRD: Implement NewSession Method

**ID**: FRD-20251114021545  
**Feature**: 3.1 - Implement NewSession Method  
**Status**: In Progress  
**Created**: 2025-11-14  
**Roadmap**: [specs/acp/ROADMAP.md](../../specs/acp/ROADMAP.md)

## Overview

Implement the `acp.Agent.NewSession()` method to create new sessions with working directory and MCP server connections.

## Goals

1. Implement `NewSession()` method
2. Parse ACP NewSessionRequest
3. Convert ACP MCP server configs to Spin format
4. Create session via `session.NewSession`
5. Connect MCP servers for the session
6. Return session ID in NewSessionResponse

## Requirements

### Functional Requirements

1. **Session Creation**
   - Extract working directory from `NewSessionRequest.Cwd`
   - Create session using `session.NewSession(workDir)`
   - Generate session ID (use session.ID)
   - Store session in session map

2. **MCP Server Connection**
   - Extract MCP server configurations from `NewSessionRequest.McpServers`
   - Convert ACP `McpServer` union type to Spin `MCPServerConfig`
   - Support stdio transport (required)
   - Support HTTP transport (if available, currently not supported)
   - Support SSE transport (if available, currently not supported)
   - Connect MCP servers via MCPManager

3. **Response Building**
   - Return session ID in `NewSessionResponse`
   - Optionally include mode state (if supported)
   - Optionally include model state (if supported)

4. **Error Handling**
   - Validate working directory (must be absolute path)
   - Handle invalid MCP server configurations
   - Handle MCP connection failures
   - Return appropriate errors

### Non-Functional Requirements

1. **Testing**
   - Unit tests for session creation
   - Tests for MCP server conversion
   - Tests for MCP server connection
   - Tests for error handling
   - Minimum 90% coverage

2. **Thread Safety**
   - Session map access must be thread-safe
   - MCP server connection must be thread-safe

## Design

### Method Signature

```go
func (a *SpinACPAgent) NewSession(ctx context.Context, req acp.NewSessionRequest) (acp.NewSessionResponse, error)
```

### Session Storage

Add session map to `SpinACPAgent`:
```go
type SpinACPAgent struct {
    // ... existing fields
    sessions map[acp.SessionId]*session.Session
    mu       sync.RWMutex // Protects sessions map
}
```

### MCP Server Conversion

```go
func convertACPMcpServerToSpin(acpServer acp.McpServer, name string) (mcp.MCPServerConfig, error) {
    config := mcp.MCPServerConfig{Name: name}
    
    if acpServer.Stdio != nil {
        config.Command = acpServer.Stdio.Command
        config.Args = acpServer.Stdio.Args
        config.Env = acpServer.Stdio.Env
        return config, nil
    }
    
    // HTTP and SSE not yet supported
    if acpServer.Http != nil {
        return config, fmt.Errorf("HTTP transport not yet supported")
    }
    
    if acpServer.Sse != nil {
        return config, fmt.Errorf("SSE transport not yet supported")
    }
    
    return config, fmt.Errorf("no transport specified")
}
```

### MCP Server Connection

Need to add method to MCPManager to connect servers dynamically:
```go
func (m *MCPManager) ConnectServer(ctx context.Context, config MCPServerConfig) error
```

Or use existing `connectServer` method if it can be made accessible.

## Implementation Plan

### Phase 1: Basic Session Creation (TDD)
1. Write test for successful session creation
2. Implement basic NewSession method
3. Create session and store in map
4. Return session ID

### Phase 2: MCP Server Conversion (TDD)
1. Write tests for MCP server conversion
2. Implement conversion function
3. Support stdio transport
4. Handle unsupported transports

### Phase 3: MCP Server Connection (TDD)
1. Write tests for MCP server connection
2. Add method to connect servers dynamically
3. Connect servers for session
4. Handle connection failures

### Phase 4: Error Handling (TDD)
1. Write tests for error cases
2. Implement error handling
3. Validate working directory
4. Validate MCP server configs

## Testing Strategy

### Unit Tests

- `TestSpinACPAgent_NewSession_Success` - Successful session creation
- `TestSpinACPAgent_NewSession_McpServers` - MCP server connection
- `TestSpinACPAgent_NewSession_InvalidCwd` - Invalid working directory
- `TestSpinACPAgent_NewSession_InvalidMcpServer` - Invalid MCP server config
- `TestSpinACPAgent_NewSession_McpServerConnectionFailure` - Connection failure handling

### Test Coverage

- Session creation: 100%
- MCP server conversion: 100%
- MCP server connection: 100%
- Error handling: 100%

## Acceptance Criteria

- [ ] `NewSession()` method implemented
- [ ] Sessions created correctly
- [ ] MCP servers connected (stdio transport)
- [ ] Session ID returned correctly
- [ ] Error handling implemented
- [ ] Unit tests written and passing
- [ ] Coverage ≥90%

## Risks and Mitigations

| Risk | Impact | Probability | Mitigation |
|------|--------|-------------|------------|
| MCP server connection failure | Medium | Medium | Handle errors gracefully, continue with session creation |
| Unsupported transport type | Low | Low | Return error for unsupported transports |
| Thread safety issues | High | Low | Use mutex for session map access |
| Session ID collision | Low | Very Low | Use UUID from session.NewSession |

## Dependencies

### External Dependencies
- `github.com/coder/acp-go-sdk` v0.6.3 (already added)

### Internal Dependencies
- `internal/session` - Session creation
- `internal/mcp` - MCP server management
- May need to add `ConnectServer` method to MCPManager

## References

- [ACP Roadmap](../../specs/acp/ROADMAP.md)
- [ACP Type Mapping](../../docs/packages/acp-type-mapping.md)
- [ACP SDK Integration](../../docs/packages/acp-sdk-integration.md)

## Notes

- MCP servers are currently shared globally, not per-session
- May need to modify MCPManager to support per-session connections
- HTTP and SSE transports not yet supported in MCPManager
- Session ID uses UUID from session.NewSession

