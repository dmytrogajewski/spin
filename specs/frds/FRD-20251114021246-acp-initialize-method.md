# FRD: Implement Initialize Method

**ID**: FRD-20251114021246  
**Feature**: 2.2 - Implement Initialize Method  
**Status**: In Progress  
**Created**: 2025-11-14  
**Roadmap**: [specs/acp/ROADMAP.md](../../specs/acp/ROADMAP.md)

## Overview

Implement the `acp.Agent.Initialize()` method to establish connection and negotiate capabilities between the ACP client and Spin agent.

## Goals

1. Implement `Initialize()` method
2. Negotiate protocol version
3. Build and advertise agent capabilities
4. Store client capabilities
5. Exchange client/agent info

## Requirements

### Functional Requirements

1. **Protocol Version Negotiation**
   - Accept protocol version from client
   - Support protocol version 1
   - Return negotiated version (or latest supported if client version not supported)

2. **Agent Capabilities Advertisement**
   - `loadSession`: Based on session persistence support
   - `promptCapabilities`: 
     - `image`: true (Spin supports images)
     - `audio`: false (not supported)
     - `embeddedContext`: true (can handle embedded resources)
   - `mcpCapabilities`:
     - `stdio`: true (required, always supported)
     - `http`: Based on MCP manager support
     - `sse`: Based on MCP manager support

3. **Client Capabilities Storage**
   - Store client capabilities for later use
   - Accessible for capability checks in other methods

4. **Info Exchange**
   - Return agent info (name: "spin", version from build)
   - Accept client info (optional)

5. **Authentication Methods**
   - Return empty list initially (no auth required)

### Non-Functional Requirements

1. **Testing**
   - Unit tests for capability negotiation
   - Tests for protocol version handling
   - Tests for client capability storage
   - Minimum 90% coverage

2. **Error Handling**
   - Handle unsupported protocol versions
   - Validate request structure

## Design

### Method Signature

```go
func (a *SpinACPAgent) Initialize(ctx context.Context, req acp.InitializeRequest) (acp.InitializeResponse, error)
```

### Capability Building

```go
func (a *SpinACPAgent) buildAgentCapabilities() acp.AgentCapabilities {
    return acp.AgentCapabilities{
        LoadSession: hasSessionPersistence(),
        PromptCapabilities: acp.PromptCapabilities{
            Image:          true,
            Audio:          false,
            EmbeddedContext: true,
        },
        McpCapabilities: acp.McpCapabilities{
            Http: hasMcpHttpSupport(),
            Sse:  hasMcpSseSupport(),
        },
    }
}
```

### Client Capabilities Storage

Add field to `SpinACPAgent`:
```go
type SpinACPAgent struct {
    // ... existing fields
    clientCaps *acp.ClientCapabilities // Stored after Initialize
}
```

### Protocol Version Negotiation

```go
func (a *SpinACPAgent) negotiateProtocolVersion(clientVersion acp.ProtocolVersion) acp.ProtocolVersion {
    if clientVersion == acp.ProtocolVersionNumber {
        return acp.ProtocolVersionNumber
    }
    // Return latest supported (currently only version 1)
    return acp.ProtocolVersionNumber
}
```

## Implementation Plan

### Phase 1: Basic Implementation (TDD)
1. Write test for successful initialization
2. Implement basic Initialize method
3. Return protocol version and capabilities

### Phase 2: Capability Building (TDD)
1. Write tests for capability detection
2. Implement capability building logic
3. Test all capability combinations

### Phase 3: Client Capabilities Storage (TDD)
1. Write test for storing client capabilities
2. Add storage field
3. Store capabilities in Initialize

### Phase 4: Error Handling (TDD)
1. Write tests for error cases
2. Implement error handling
3. Verify error messages

## Testing Strategy

### Unit Tests

- `TestSpinACPAgent_Initialize_Success` - Successful initialization
- `TestSpinACPAgent_Initialize_ProtocolVersion` - Version negotiation
- `TestSpinACPAgent_Initialize_AgentCapabilities` - Capability advertisement
- `TestSpinACPAgent_Initialize_ClientCapabilitiesStorage` - Storage verification
- `TestSpinACPAgent_Initialize_AgentInfo` - Info exchange

### Test Coverage

- Protocol version negotiation: 100%
- Capability building: 100%
- Client capability storage: 100%
- Error handling: 100%

## Acceptance Criteria

- [ ] `Initialize()` method implemented
- [ ] Protocol version negotiated correctly
- [ ] Agent capabilities advertised based on Spin features
- [ ] Client capabilities stored for later use
- [ ] Agent info returned correctly
- [ ] Unit tests written and passing
- [ ] Coverage ≥90%

## Risks and Mitigations

| Risk | Impact | Probability | Mitigation |
|------|--------|-------------|------------|
| Incorrect capability detection | Medium | Low | Write tests for each capability |
| Protocol version mismatch | Low | Low | Support only version 1, return error for others |
| Client capabilities not stored | Medium | Low | Add explicit storage test |

## Dependencies

### External Dependencies
- `github.com/coder/acp-go-sdk` v0.6.3 (already added)

### Internal Dependencies
- `internal/session` - Check session persistence
- `internal/mcp` - Check MCP transport support
- `internal/version` - Get Spin version

## References

- [ACP Roadmap](../../specs/acp/ROADMAP.md)
- [ACP Type Mapping](../../docs/packages/acp-type-mapping.md)
- [ACP SDK Integration](../../docs/packages/acp-sdk-integration.md)

## Notes

- Session persistence: Check if `session.Storage` interface exists
- MCP HTTP/SSE: Check MCP manager capabilities
- Version: Use build-time version or constant

