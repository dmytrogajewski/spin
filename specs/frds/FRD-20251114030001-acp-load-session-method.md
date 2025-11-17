# FRD: ACP LoadSession Method Implementation

**ID**: FRD-20251114030001
**Status**: Implementation
**Created**: 2025-11-14
**Roadmap**: [P3.2] Implement LoadSession Method (Optional)
**Phase**: 3 - Session Management Implementation
**Priority**: MEDIUM (Optional)
**Estimated Effort**: 3-4 hours

## Overview

Implement the `acp.Agent.LoadSession()` method to load existing sessions from storage, restore session state, reconnect MCP servers, and prepare for conversation history replay.

## Problem Statement

The ACP agent currently only supports creating new sessions via `NewSession()`. We need to support loading existing sessions from persistent storage to enable session resumption and conversation history replay.

## Goals

1. Implement `LoadSession()` method that loads sessions from storage
2. Validate session exists in storage
3. Restore session to sessions map
4. Reconnect MCP servers if provided
5. Prepare conversation history replay structure (actual replay deferred to Feature 4.2)

## Non-Goals

- Full conversation history replay via notifications (Feature 4.2)
- Session migration/upgrade logic
- Complex session validation beyond basic checks

## Design

### 1. Method Signature

```go
func (a *SpinACPAgent) LoadSession(ctx context.Context, req acp.LoadSessionRequest) (acp.LoadSessionResponse, error)
```

### 2. Session Storage Integration

The `SpinACPAgent` needs access to session storage. Since storage is optional, we'll:
- Add optional `storage session.Storage` field to `SpinACPAgent`
- If storage is nil, return error indicating session persistence not available
- If storage is provided, use it to load sessions

**Storage Integration**:
```go
type SpinACPAgent struct {
    // ... existing fields ...
    storage session.Storage // Optional session storage
}
```

### 3. LoadSession Flow

1. **Validate storage available**: Check if `a.storage != nil`
2. **Load session from storage**: Call `storage.Load(string(req.SessionId))`
3. **Validate session**: Ensure session is valid
4. **Store in sessions map**: Add to `a.sessions` map
5. **Reconnect MCP servers**: If `req.McpServers` provided, connect them
6. **Return response**: Return `LoadSessionResponse` with optional Models/Modes

### 4. MCP Server Reconnection

Similar to `NewSession`, validate and convert MCP servers:
- Validate MCP server configurations synchronously
- Connect servers asynchronously (non-blocking)
- Log errors but don't fail session loading

### 5. Conversation History Replay (Deferred)

The roadmap mentions replaying conversation history via `connection.SendUpdate()`. However:
- `connection.SendUpdate()` is not available yet (Feature 4.2)
- For now, we'll load the session and store it
- History replay will be implemented in Feature 4.2 when notification infrastructure is ready

**Future Enhancement** (Feature 4.2):
```go
// For each turn in session.Turns:
//   - Send user_message_chunk for UserInput
//   - Send tool_call notifications for ToolCalls
//   - Send tool_call_update for ToolResults
//   - Send agent_message_chunk for AIResponse
```

### 6. Error Handling

- Session not found in storage → return error
- Invalid session data → return error
- MCP server connection failures → log but don't fail
- Storage unavailable → return error indicating persistence not configured

## Implementation Steps

1. **Add storage field to SpinACPAgent** (optional)
2. **Update NewSpinACPAgent constructor** to accept optional storage
3. **Implement LoadSession() method**:
   - Validate storage available
   - Load session from storage
   - Validate session
   - Store in sessions map
   - Reconnect MCP servers
   - Return response
4. **Write comprehensive tests**

## Testing Strategy

### Unit Tests

1. **Storage Validation**:
   - Storage nil → error
   - Storage available → success

2. **Session Loading**:
   - Valid session ID → loads successfully
   - Invalid session ID → error
   - Corrupted session data → error

3. **Session Storage**:
   - Loaded session stored in sessions map
   - Session ID matches

4. **MCP Server Reconnection**:
   - MCP servers provided → reconnected
   - MCP connection failures → logged but don't fail

5. **Error Handling**:
   - All error cases handled correctly

### Integration Tests

1. **Full Load Flow**:
   - Create and save session
   - Load session
   - Verify session restored
   - Verify MCP servers reconnected

## Acceptance Criteria

- ✅ `LoadSession()` method implemented
- ✅ Session loaded from storage correctly
- ✅ Session stored in sessions map
- ✅ MCP servers reconnected (if provided)
- ✅ Error handling for missing/invalid sessions
- ✅ Unit tests (≥90% coverage)
- ✅ Integration tests
- ✅ No lint errors
- ✅ No deadcode

## Dependencies

- Feature 3.1 (NewSession) - ✅ Completed
- Session storage infrastructure - ✅ Exists (`session.Storage`, `session.FileStorage`)
- SDK types (`acp.LoadSessionRequest`, `acp.LoadSessionResponse`)
- MCP manager (already integrated)

## References

- [ACP Roadmap](../../specs/acp/ROADMAP.md#feature-32-implement-loadsession-method-optional)
- [ACP Type Mapping](../../docs/packages/acp-type-mapping.md)
- [ACP SDK Integration](../../docs/packages/acp-sdk-integration.md)
- [ACP Specification](../../specs/acp/specification.md)

## Notes

- Conversation history replay via notifications is deferred to Feature 4.2 (requires `connection.SendUpdate()`)
- Session storage is optional - if not provided, LoadSession will return an error
- MCP server reconnection follows the same pattern as NewSession (async, non-blocking)

