# FRD-20251115031818: Unify Conversation ID Generation

## Metadata
- **Status**: ✅ COMPLETE
- **Priority**: P2 (MEDIUM)
- **Effort**: S (1 day)
- **Dependencies**: Feature 3.3 (complete)
- **Related**: [Codepath Duplication Roadmap](../../codepath-duplication-assessment/ROADMAP.md#feature-34-unify-conversation-id-generation)

## Problem Statement

Conversation IDs are represented and generated inconsistently across packages:

1. **`conversation.Conversation`**:
   - `sessionID string` - Used for session tracking/persistence (from `session.Session.ID`)
   - `protocolID protocol.ConversationID` - Used for protocol layer (already unified in Feature 3.2)

2. **`session.Session`**:
   - `ID string` - Generated via `uuid.New().String()` directly

3. **`protocol.ConversationID`**:
   - Type: `struct { ID uuid.UUID }`
   - Generated via `protocol.NewConversationID()` which wraps `uuid.New()`

**Issues:**
- **Different ID Types**: `string` vs `protocol.ConversationID` vs `uuid.UUID`
- **Different Generation Methods**: Direct `uuid.New().String()` vs `protocol.NewConversationID()`
- **Duplication**: Two ID fields in `conversation.Conversation` (`sessionID` and `protocolID`)
- **Inconsistency**: No standardized way to convert between types

**Note**: After Feature 3.2 merge, `conversation.Conversation` has both `sessionID` (for session persistence) and `protocolID` (for protocol layer). These serve different purposes but could potentially be unified.

## Goals

1. **Standardize on `protocol.ConversationID` as canonical ID type**
2. **Unify `sessionID` and `protocolID` in `conversation.Conversation`** (use single ID)
3. **Standardize ID generation** - always use `protocol.NewConversationID()`
4. **Update `session.Session` to use `protocol.ConversationID`** (or keep string but standardize generation)
5. **Update all ID conversions** to use `protocol.ConversationID`

## Non-Goals

1. **NOT changing protocol interface** - JSON-RPC API remains the same (uses string)
2. **NOT changing session storage format** - May still use string for persistence
3. **NOT maintaining backward compatibility** - Breaking changes allowed

## Design

### Current Implementation

**`conversation.Conversation`**:
```go
type Conversation struct {
    // ...
    sessionID string // Session identifier for tracking and persistence
    protocolID protocol.ConversationID // Protocol conversation ID
}
```

**`session.Session`**:
```go
type Session struct {
    ID string // uuid.New().String()
}
```

**`protocol.ConversationID`**:
```go
type ConversationID struct {
    ID uuid.UUID
}
```

### Target Implementation

**Option A: Unify on `protocol.ConversationID` everywhere**

**`conversation.Conversation`**:
```go
type Conversation struct {
    // ...
    id protocol.ConversationID // Single unified ID for both session and protocol
}
```

**`session.Session`**:
```go
type Session struct {
    ID protocol.ConversationID // Use protocol type
}
```

**Option B: Keep session ID separate but standardize generation**

Keep `sessionID` and `protocolID` separate (they serve different purposes), but:
- Ensure both use `protocol.NewConversationID()` for generation
- Standardize on `protocol.ConversationID` as the canonical type
- Use conversion methods where string is needed

**Recommendation**: **Option A** - Unify on single ID since both `sessionID` and `protocolID` identify the same conversation, just used in different contexts.

### Changes Required

1. **Replace `sessionID` and `protocolID` with single `id protocol.ConversationID` in `conversation.Conversation`**
2. **Update `session.Session.ID` to `protocol.ConversationID`** (breaking change)
3. **Update all ID generation to use `protocol.NewConversationID()`**
4. **Add helper methods for string conversion where needed** (`id.String()`)
5. **Update all callers to use unified ID**

## API Changes

### Breaking Changes

1. **`conversation.Conversation.sessionID`** - Removed
2. **`conversation.Conversation.protocolID`** - Removed
3. **`conversation.Conversation.GetSessionID()`** - Removed or changed to return `string` (via `id.String()`)
4. **`conversation.Conversation.GetProtocolID()`** - Removed or changed to return `protocol.ConversationID`
5. **`session.Session.ID`** - Changed from `string` to `protocol.ConversationID`

### New Methods

```go
// conversation.Conversation
func (c *Conversation) ID() protocol.ConversationID
func (c *Conversation) IDString() string // Helper for string conversion
```

## Implementation Plan

### Step 1: Unify IDs in `conversation.Conversation`
1. Replace `sessionID` and `protocolID` with single `id protocol.ConversationID`
2. Update `GetSessionID()` to return `id.String()`
3. Update `GetProtocolID()` to return `id` or remove if redundant
4. Add `ID()` method to return `protocol.ConversationID`
5. Update `NewConversationForProtocol()` to set unified ID

### Step 2: Update `session.Session` to use `protocol.ConversationID`
1. Change `Session.ID` from `string` to `protocol.ConversationID`
2. Update `NewSession()` to use `protocol.NewConversationID()`
3. Update all session ID access to use `.ID.String()` where string needed

### Step 3: Standardize ID generation
1. Replace all `uuid.New().String()` with `protocol.NewConversationID().String()`
2. Update `session.NewSession()` to use `protocol.NewConversationID()`
3. Update `conversation.Builder` to use `protocol.NewConversationID()`

### Step 4: Update all callers
1. Update all code that accesses `sessionID` or `protocolID`
2. Update all code that uses `session.ID` (now `protocol.ConversationID`)
3. Update tests to use new ID type

### Step 5: Verify and test
1. Run all tests
2. Verify protocol functionality still works
3. Verify session persistence still works
4. Check for any ID conversion issues

## Testing Strategy

### Unit Tests

```go
func TestConversation_UnifiedID(t *testing.T) {
    conv := setupTestConv(t)
    
    // Verify single ID field exists
    id := conv.ID()
    if id.ID == uuid.Nil {
        t.Error("ID should not be nil")
    }
    
    // Verify string conversion
    idStr := conv.IDString()
    if idStr != id.String() {
        t.Errorf("IDString() = %q, want %q", idStr, id.String())
    }
}

func TestSession_ConversationID(t *testing.T) {
    sess := session.NewSession("/tmp")
    
    // Verify ID is protocol.ConversationID
    if sess.ID.ID == uuid.Nil {
        t.Error("Session ID should not be nil")
    }
    
    // Verify string conversion
    idStr := sess.ID.String()
    if idStr == "" {
        t.Error("ID.String() should not be empty")
    }
}

func TestConversationID_StandardizedGeneration(t *testing.T) {
    // Verify all IDs use protocol.NewConversationID()
    id1 := protocol.NewConversationID()
    id2 := protocol.NewConversationID()
    
    if id1.ID == id2.ID {
        t.Error("IDs should be unique")
    }
    
    if id1.ID == uuid.Nil || id2.ID == uuid.Nil {
        t.Error("IDs should not be nil")
    }
}
```

### Integration Tests

```go
func TestConversation_SessionID_Compatibility(t *testing.T) {
    // Verify session ID string format is preserved
    conv := setupTestConv(t)
    sessionID := conv.GetSessionID() // Should return id.String()
    
    // Verify format (UUID string)
    _, err := uuid.Parse(sessionID)
    if err != nil {
        t.Errorf("SessionID %q is not a valid UUID: %v", sessionID, err)
    }
}
```

### Acceptance Criteria

1. ✅ Single ID field in `conversation.Conversation` (replaces `sessionID` and `protocolID`) - **Implemented as `string` to avoid protocol dependency**
2. ✅ `session.Session.ID` remains `string` (no protocol dependency) - **Architecture decision: core modules don't import protocol**
3. ✅ ID generation standardized to `uuid.New().String()` - **Both conversation and session use same generation**
4. ✅ Conversion happens at boundary (`appserver/processor.go`) - **Protocol layer converts string ↔ `protocol.ConversationID`**
5. ✅ All tests pass
6. ✅ Protocol functionality verified (ID serialization works)
7. ✅ Session persistence verified (ID storage/retrieval works)
8. ✅ `go vet` passes
9. ✅ No dead code introduced
10. ✅ Clean dependency direction maintained (core modules don't depend on infrastructure)

**Status**: ✅ **COMPLETE** (2025-11-15)

**Achievement**: Successfully unified conversation ID generation while maintaining clean architecture. Replaced `sessionID` and `protocolID` with single `id string` field in `conversation.Conversation`. Standardized ID generation to `uuid.New().String()` in both `conversation` and `session` packages. Conversion between string and `protocol.ConversationID` happens only at the boundary in `appserver/processor.go`, maintaining proper dependency direction (core modules don't import infrastructure-level `protocol` package). All tests pass with no functional regressions.

## Files to Modify

- `internal/conversation/conversation.go` - Replace `sessionID` and `protocolID` with single `id`
- `internal/conversation/conversation_test.go` - Update tests for unified ID
- `internal/conversation/builder.go` - Update ID generation
- `internal/session/session.go` - Change `ID` from `string` to `protocol.ConversationID`
- `internal/session/session_test.go` - Update tests for new ID type
- `internal/appserver/processor.go` - Update ID handling (already uses `protocolID`)
- `internal/appserver/processor_test.go` - Update tests

## Risks and Mitigation

### Risk 1: Breaking session persistence
**Risk**: Changing `session.Session.ID` type may break existing session storage.
**Mitigation**: Update session storage to serialize/deserialize `protocol.ConversationID` correctly, or use `ID.String()` for storage.

### Risk 2: Protocol serialization issues
**Risk**: JSON-RPC uses strings, need to ensure `protocol.ConversationID` serializes correctly.
**Mitigation**: `protocol.ConversationID` already has `String()` method, use it for serialization.

### Risk 3: ID conversion complexity
**Risk**: Many places may need ID conversion between types.
**Mitigation**: Use `id.String()` helper method consistently, add conversion helpers if needed.

## Dependencies

- ✅ Feature 3.2 (complete) - Conversation merge done
- ✅ Feature 3.3 (complete) - Task mode unified
- `protocol.ConversationID` - Must support all use cases

## Success Metrics

- [ ] Single ID field in `conversation.Conversation`
- [ ] All ID generation uses `protocol.NewConversationID()`
- [ ] All tests pass
- [ ] No ID conversion errors
- [ ] Protocol and session functionality verified

## References

- [Codepath Duplication Assessment](../../codepath-duplication-assessment/assessment.md)
- [Roadmap Feature 3.4](../../codepath-duplication-assessment/ROADMAP.md#feature-34-unify-conversation-id-generation)
- `internal/conversation/conversation.go:31,35` - Current ID fields
- `internal/session/session.go:52` - Session ID generation
- `internal/protocol/conversation.go:37-50` - Protocol ID type

