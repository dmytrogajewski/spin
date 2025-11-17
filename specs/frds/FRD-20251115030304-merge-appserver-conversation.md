# FRD-20251115030304: Merge `appserver.Conversation` into `conversation.Conversation`

## Metadata
- **Status**: COMPLETE
- **Priority**: P0 (CRITICAL)
- **Effort**: M (2 days)
- **Dependencies**: Feature 3.1 (complete)
- **Related**: [Codepath Duplication Roadmap](../../codepath-duplication-assessment/ROADMAP.md#feature-32-merge-appserverconversation-into-conversationconversation)

## Problem Statement

Two different conversation types exist with overlapping responsibilities:

1. **`conversation.Conversation`** (`internal/conversation/conversation.go:17-141`):
   - Uses `history.History` wrapper for message management
   - Task mode management (`SetTaskMode()`, `GetTaskMode()`)
   - Turn execution (`RunTurn()`)
   - Session tracking (`sessionID`)
   - Event emission via `EventEmitter`

2. **`appserver.Conversation`** (`internal/appserver/processor.go:52-59`):
   - Raw `[]message.Message` slice for history storage
   - Task mode tracking (`taskMode` field with mutex)
   - Protocol-level conversation tracking (`protocol.ConversationID`)
   - Cancellation context management (`cancel context.CancelFunc`)
   - Turn ID tracking (`TurnID string`)

**Issues:**
- **Duplication**: Both types manage conversation state, history, and task mode
- **Different History Models**: `conversation.Conversation` uses `history.History`, `appserver.Conversation` uses `[]message.Message`
- **Different Task Mode Handling**: Different validation and management logic
- **Different ID Types**: `conversation.Conversation` uses `string` (sessionID), `appserver.Conversation` uses `protocol.ConversationID`
- **Maintenance Burden**: Changes must be made in multiple places
- **Inconsistency Risk**: Different implementations may diverge

## Goals

1. **Merge `appserver.Conversation` into `conversation.Conversation`** to eliminate duplication
2. **Extend `conversation.Conversation` with protocol-specific fields** (ID, TurnID, cancel)
3. **Unify history management** - migrate from `[]message.Message` to `history.History`
4. **Unify task mode handling** - use `conversation.Conversation` validation
5. **Update Processor to use unified type** - replace all `appserver.Conversation` references
6. **Remove `appserver.Conversation` type** entirely

## Non-Goals

1. **NOT merging `session.Session`** - remains separate for persistent state
2. **NOT changing protocol interface** - Processor interface remains the same
3. **NOT breaking backward compatibility** - API behavior unchanged

## Design

### Current Implementation

**`appserver.Conversation`** (processor.go:50-57):
```go
type Conversation struct {
    ID       protocol.ConversationID
    TurnID   string
    History  []message.Message
    cancel   context.CancelFunc
    taskMode string
    mu       sync.RWMutex // protects taskMode access
}
```

**`conversation.Conversation`** (conversation.go:17-31):
```go
type Conversation struct {
    // Services (optional)
    gitService   *git.Service
    shellService *shell.Service
    mcpService   *mcp.Service

    // Core components
    agent     *agent.Agent
    history   *history.History
    emitter   *events.EventEmitter
    taskMode  string
    sessionID string
    workDir   string
}
```

### Target Implementation

**Extended `conversation.Conversation`**:
```go
type Conversation struct {
    // Services (optional)
    gitService   *git.Service
    shellService *shell.Service
    mcpService   *mcp.Service

    // Core components
    agent     *agent.Agent
    history   *history.History
    emitter   *events.EventEmitter
    taskMode  string
    sessionID string
    workDir   string

    // Protocol-specific fields (optional, for protocol use)
    protocolID protocol.ConversationID  // Protocol conversation ID
    turnID     string                   // Current turn ID
    cancel     context.CancelFunc       // Cancellation context
    protocolMu sync.RWMutex             // Protects protocol fields (turnID, cancel)
}
```

### Changes Required

1. **Extend `conversation.Conversation` struct** with protocol fields
2. **Add methods for protocol fields** (getters/setters with mutex protection)
3. **Update Processor** to use `conversation.Conversation`
4. **Handle history migration** - convert `[]message.Message` to `history.History`
5. **Remove `appserver.Conversation` type** definition

## API Changes

### New Fields on `conversation.Conversation`

```go
// Protocol-specific fields (optional, for protocol use)
protocolID protocol.ConversationID  // Protocol conversation ID
turnID     string                   // Current turn ID  
cancel     context.CancelFunc       // Cancellation context
protocolMu sync.RWMutex             // Protects protocol fields
```

### New Methods on `conversation.Conversation`

```go
// GetProtocolID returns the protocol conversation ID.
func (c *Conversation) GetProtocolID() protocol.ConversationID

// SetProtocolID sets the protocol conversation ID.
func (c *Conversation) SetProtocolID(id protocol.ConversationID)

// GetTurnID returns the current turn ID.
func (c *Conversation) GetTurnID() string

// SetTurnID sets the current turn ID (thread-safe).
func (c *Conversation) SetTurnID(turnID string)

// GetCancel returns the cancellation context function.
func (c *Conversation) GetCancel() context.CancelFunc

// SetCancel sets the cancellation context function (thread-safe).
func (c *Conversation) SetCancel(cancel context.CancelFunc)

// Cancel cancels the current turn (thread-safe).
func (c *Conversation) Cancel()
```

**Breaking Change**: Yes - `appserver.Conversation` type removed.

## Implementation Plan

### Step 1: Extend `conversation.Conversation` with protocol fields
1. Add protocol-specific fields to struct
2. Add mutex for protocol fields
3. Add getter/setter methods with thread-safety
4. Add unit tests for new methods

### Step 2: Update Processor to use `conversation.Conversation`
1. Change `Processor.conversations` map type: `map[string]*conversation.Conversation`
2. Update `HandleSendMessage()` to create/use `conversation.Conversation` via Builder
3. Update `CancelTurn()` to work with new type
4. Update `runTurn()` to work with new type

### Step 3: Handle history migration
1. Remove raw `[]message.Message` slice from old type
2. Use `conversation.Conversation.history` (already `history.History`)
3. Update history access patterns in Processor

### Step 4: Handle task mode migration
1. Remove duplicate task mode validation
2. Use `conversation.Conversation.SetTaskMode()` and `GetTaskMode()`
3. Replace `jsonrpc.ValidateTaskMode()` with `conversation` package validation

### Step 5: Remove `appserver.Conversation` type
1. Delete struct definition
2. Update all references in Processor
3. Update all tests
4. Verify no remaining references

### Step 6: Update tests
1. Update Processor tests to use `conversation.Conversation`
2. Update integration tests
3. Verify protocol functionality still works
4. Add tests for protocol-specific fields

## Testing Strategy

### Unit Tests

```go
func TestConversation_ProtocolFields(t *testing.T) {
    conv := setupTestConv(t)
    
    // Test protocol ID
    id := protocol.NewConversationID()
    conv.SetProtocolID(id)
    assert.Equal(t, id, conv.GetProtocolID())
    
    // Test turn ID (thread-safe)
    conv.SetTurnID("turn-123")
    assert.Equal(t, "turn-123", conv.GetTurnID())
    
    // Test cancel (thread-safe)
    ctx, cancel := context.WithCancel(context.Background())
    conv.SetCancel(cancel)
    assert.NotNil(t, conv.GetCancel())
    
    // Test cancel execution
    conv.Cancel()
    assert.Error(t, ctx.Err())
}

func TestConversation_ProtocolFields_ThreadSafety(t *testing.T) {
    conv := setupTestConv(t)
    
    // Concurrent access to protocol fields
    var wg sync.WaitGroup
    for i := 0; i < 100; i++ {
        wg.Add(1)
        go func(i int) {
            defer wg.Done()
            conv.SetTurnID(fmt.Sprintf("turn-%d", i))
            _ = conv.GetTurnID()
        }(i)
    }
    wg.Wait()
}
```

### Integration Tests

```go
func TestProcessor_UsesConversationType(t *testing.T) {
    processor := setupProcessor(t)
    
    params := jsonrpc.SendMessageParams{
        Message: "Hello",
    }
    
    result, err := processor.HandleSendMessage(context.Background(), params)
    assert.NoError(t, err)
    
    // Verify conversation is conversation.Conversation type
    processor.mu.RLock()
    conv, ok := processor.conversations[result.ConversationID]
    processor.mu.RUnlock()
    
    assert.True(t, ok)
    assert.IsType(t, (*conversation.Conversation)(nil), conv)
}

func TestProcessor_HistoryAccess(t *testing.T) {
    // Verify history is accessible via conversation.Conversation.history
    // Verify history is history.History type, not []message.Message
}
```

### Acceptance Criteria

1. ✅ `conversation.Conversation` extended with protocol fields (ID, TurnID, cancel)
2. ✅ Protocol field methods added with thread-safety
3. ✅ `Processor` uses `conversation.Conversation` instead of `appserver.Conversation`
4. ✅ `appserver.Conversation` type removed entirely
5. ✅ All Processor methods updated for new type
6. ✅ History migration complete (uses `history.History` via `GetHistoryMessages()` and `AddHistoryMessage()`)
7. ✅ Task mode migration complete (uses `conversation.SetTaskMode()` and `GetTaskMode()`)
8. ✅ All tests pass (unit, integration)
9. ✅ Protocol functionality verified (WebSocket JSON-RPC still works)
10. ✅ `go vet` passes
11. ✅ No dead code introduced

**Status**: ✅ **COMPLETE** (2025-11-15)

**Achievement**: Successfully merged `appserver.Conversation` into `conversation.Conversation`. Extended `conversation.Conversation` with protocol-specific fields (protocolID, turnID, cancel) and thread-safe methods. Updated Processor to use unified type, replacing all references. Removed `appserver.Conversation` type entirely. Migrated history management to use `history.History` via public methods. Migrated task mode handling to use `conversation.SetTaskMode()` and `GetTaskMode()`. All tests pass with no functional changes.

## Files to Modify

- `internal/conversation/conversation.go` - Add protocol-specific fields and methods
- `internal/conversation/conversation_test.go` - Add tests for protocol fields
- `internal/appserver/processor.go` - Replace `appserver.Conversation` with `conversation.Conversation`
- `internal/appserver/processor_test.go` - Update tests for new type
- `internal/appserver/processor_integration_test.go` - Update integration tests

## Risks and Mitigation

### Risk 1: Thread-safety issues
**Risk**: Protocol fields accessed concurrently without proper synchronization.
**Mitigation**: Use mutex protection for protocol fields, add concurrency tests.

### Risk 2: History migration breaks functionality
**Risk**: History access patterns differ between `[]message.Message` and `history.History`.
**Mitigation**: Use `history.History` methods consistently, add integration tests.

### Risk 3: Protocol functionality breaks
**Risk**: Protocol layer depends on specific conversation structure.
**Mitigation**: Maintain protocol field compatibility, add protocol integration tests.

### Risk 4: Builder complexity
**Risk**: Creating `conversation.Conversation` via Builder for protocol use may be complex.
**Mitigation**: Simplify Builder usage, create helper function for protocol conversations.

## Dependencies

- ✅ Feature 3.1 (analysis complete, Option A selected)
- `conversation.Builder` - Must support creating conversations for protocol use
- `history.History` - Must support all history access patterns

## Success Metrics

- [ ] Zero references to `appserver.Conversation` in codebase
- [ ] All Processor operations use `conversation.Conversation`
- [ ] All tests pass (unit, integration, protocol)
- [ ] Protocol functionality verified (WebSocket JSON-RPC works)
- [ ] No performance regression

## References

- [Conversation Unification Analysis](../../analysis/conversation-unification-analysis.md)
- [Codepath Duplication Assessment](../../codepath-duplication-assessment/assessment.md)
- [Roadmap Feature 3.2](../../codepath-duplication-assessment/ROADMAP.md#feature-32-merge-appserverconversation-into-conversationconversation)
- `internal/conversation/conversation.go` - Type 1 definition
- `internal/appserver/processor.go` - Type 2 definition and usage

