# FRD-20251027000001: Protocol Layer Typed Messages

## Metadata

**ID**: FRD-20251027000001  
**Feature**: Protocol Layer - ParsedMessage Type Safety  
**Roadmap Phase**: Phase 3.1 - Protocol Layer (Priority: P1)  
**Author**: Claude (Rob Pike persona)  
**Created**: 2025-10-27  
**Status**: In Progress

## 1. Executive Summary

Replace `interface{}` return types in protocol message parsing with a type-safe `ParsedMessage` interface. This eliminates 9 occurrences of `interface{}` in `internal/protocol/protocol.go` and provides compile-time type safety for message handling throughout the protocol layer.

**Current Interface{} Count in protocol.go**: 9 occurrences  
**Target After Completion**: 0 occurrences  
**Impact**: Core protocol layer used by JSON-RPC server and all UI clients

## 2. Problem Statement

### Current Implementation Issues

1. **Type Uncertainty**: `ParseMessage()` returns `interface{}`, requiring runtime type assertions
2. **No Compile-Time Safety**: Callers must manually check types with type switches
3. **Error-Prone**: Easy to miss handling a message type
4. **Poor IDE Support**: No autocomplete for message-specific methods

### Example of Current Problem

```go
// Current implementation (protocol.go:152)
func ParseMessage(msg Message) (interface{}, error) {
    parser := getMessageParser(msg.Type)
    if parser == nil {
        return nil, fmt.Errorf("unknown message type: %s", msg.Type)
    }
    return parser(msg.Data)
}

// Usage requires type assertions
parsed, err := ParseMessage(msg)
if err != nil {
    return err
}

// Unsafe - runtime type assertion required
switch v := parsed.(type) {
case TurnStart:
    // handle
case AssistantDelta:
    // handle
// Easy to miss a type!
}
```

## 3. Requirements

### Functional Requirements

**FR1**: Define `ParsedMessage` interface with marker method  
**FR2**: All message types implement `ParsedMessage`  
**FR3**: `ParseMessage()` returns `ParsedMessage` instead of `interface{}`  
**FR4**: Type-safe message type checking via type switches  
**FR5**: Backward compatible JSON serialization

### Non-Functional Requirements

**NFR1**: 90%+ test coverage for new implementation  
**NFR2**: No performance regression in message parsing  
**NFR3**: Zero deadcode warnings from static analysis  
**NFR4**: All existing tests continue to pass  
**NFR5**: Documentation updated to reflect new types

## 4. Technical Design

### 4.1 ParsedMessage Interface

```go
// ParsedMessage is implemented by all protocol message types.
// The messageType() method is a marker that prevents external types
// from implementing this interface.
type ParsedMessage interface {
    messageType()
}
```

**Rationale**: 
- Sealed interface pattern (unexported method prevents external implementation)
- Enables compile-time verification of message types
- Compatible with type switches for pattern matching

### 4.2 Message Type Implementations

Each message type implements `ParsedMessage`:

```go
// TurnStart implements ParsedMessage
func (TurnStart) messageType() {}

// AssistantDelta implements ParsedMessage
func (AssistantDelta) messageType() {}

// ToolCallProposed implements ParsedMessage
func (ToolCallProposed) messageType() {}

// ToolCallExecuting implements ParsedMessage
func (ToolCallExecuting) messageType() {}

// ToolCallResult implements ParsedMessage
func (ToolCallResult) messageType() {}

// TurnComplete implements ParsedMessage
func (TurnComplete) messageType() {}

// StatusUpdate implements ParsedMessage
func (StatusUpdate) messageType() {}
```

### 4.3 Updated Parser Signatures

```go
// Before
type messageParser func([]byte) (interface{}, error)
func ParseMessage(msg Message) (interface{}, error)

// After
type messageParser func([]byte) (ParsedMessage, error)
func ParseMessage(msg Message) (ParsedMessage, error)
```

### 4.4 Usage Pattern

```go
// Type-safe usage with exhaustive type switch
parsed, err := ParseMessage(msg)
if err != nil {
    return err
}

switch v := parsed.(type) {
case TurnStart:
    // IDE autocompletes TurnStart fields
    handleTurnStart(v.TurnID, v.UserMessage)
case AssistantDelta:
    handleDelta(v.Delta, v.Reasoning)
case ToolCallProposed:
    handleToolCall(v.ToolCallID, v.ToolName, v.Arguments)
case ToolCallExecuting:
    handleExecuting(v.ToolCallID)
case ToolCallResult:
    handleResult(v.ToolCallID, v.Result)
case TurnComplete:
    handleComplete(v.TurnID, v.FinalMessage)
case StatusUpdate:
    handleStatus(v.Message, v.Level)
default:
    // Compiler warns if we add new message type and forget to handle it
    return fmt.Errorf("unhandled message type: %T", v)
}
```

## 5. Implementation Plan

### Phase 1: Define Interface and Implementations (Micro-TDD)

**Step 1**: Add `ParsedMessage` interface  
**Step 2**: Add `messageType()` method to `TurnStart`  
**Step 3**: Add `messageType()` to remaining message types  
**Step 4**: Write tests verifying interface implementation

### Phase 2: Update Parser Functions

**Step 5**: Update `messageParser` type definition  
**Step 6**: Update `parseTurnStart` signature  
**Step 7**: Update remaining parser functions  
**Step 8**: Update `ParseMessage()` signature  
**Step 9**: Write parser tests

### Phase 3: Verification

**Step 10**: Run `make test` - verify all tests pass  
**Step 11**: Run `uast parse` and `herr analyze`  
**Step 12**: Run `make lint` - fix any issues  
**Step 13**: Check test coverage (target: 90%+)

## 6. Testing Strategy

### Unit Tests

```go
func TestParsedMessageInterface(t *testing.T) {
    // Verify all message types implement ParsedMessage
    var _ ParsedMessage = TurnStart{}
    var _ ParsedMessage = AssistantDelta{}
    var _ ParsedMessage = ToolCallProposed{}
    var _ ParsedMessage = ToolCallExecuting{}
    var _ ParsedMessage = ToolCallResult{}
    var _ ParsedMessage = TurnComplete{}
    var _ ParsedMessage = StatusUpdate{}
}

func TestParseMessage_TypeSafety(t *testing.T) {
    tests := []struct {
        name     string
        message  Message
        wantType ParsedMessage
    }{
        {
            name: "TurnStart",
            message: NewTurnStartMessage(TurnStart{
                TurnID:      "turn-1",
                UserMessage: "Hello",
            }),
            wantType: TurnStart{},
        },
        // ... more test cases
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            parsed, err := ParseMessage(tt.message)
            if err != nil {
                t.Fatalf("ParseMessage() error = %v", err)
            }

            // Verify correct type returned
            switch tt.wantType.(type) {
            case TurnStart:
                if _, ok := parsed.(TurnStart); !ok {
                    t.Errorf("Expected TurnStart, got %T", parsed)
                }
            // ... more type checks
            }
        })
    }
}
```

### Coverage Targets

- `protocol.go`: 90%+ coverage
- All message types: 100% coverage
- All parser functions: 100% coverage

## 7. Migration Impact

### Breaking Changes

**None** - This is an internal change. External callers already use type assertions, which continue to work.

### Affected Components

1. **internal/protocol/jsonrpc/server.go** - Message handling
2. **internal/appserver/** - Protocol message processing
3. **cmd/spin-tui/** - UI message handlers

### Migration Path

No migration needed - type assertions work identically with `ParsedMessage` interface:

```go
// Before
parsed, _ := ParseMessage(msg)
if ts, ok := parsed.(TurnStart); ok {
    // ...
}

// After (identical code)
parsed, _ := ParseMessage(msg)
if ts, ok := parsed.(TurnStart); ok {
    // ...
}
```

## 8. Alternatives Considered

### Alternative 1: Generic ParseMessage[T ParsedMessage]

```go
func ParseMessage[T ParsedMessage](msg Message) (T, error)
```

**Rejected**: Caller must know type in advance, defeating purpose of dynamic message parsing.

### Alternative 2: Keep interface{}

**Rejected**: Violates roadmap goal of eliminating `interface{}` and provides no type safety.

### Alternative 3: Separate Parse Functions

```go
func ParseTurnStart(msg Message) (TurnStart, error)
func ParseAssistantDelta(msg Message) (AssistantDelta, error)
// ...
```

**Rejected**: Requires caller to map message type to function, duplicating `getMessageParser()` logic.

## 9. Success Criteria

### Acceptance Criteria

- [ ] All message types implement `ParsedMessage`
- [ ] `ParseMessage()` returns `ParsedMessage`
- [ ] All parser functions return `ParsedMessage`
- [ ] 90%+ test coverage achieved
- [ ] All existing tests pass
- [ ] Zero lint errors
- [ ] Zero deadcode warnings
- [ ] Documentation updated

### Verification Steps

1. Grep for `interface{}` in `internal/protocol/protocol.go` - expect 0 results
2. Run `make test` - all tests pass
3. Run `make lint` - no errors
4. Run `uast parse internal/protocol/protocol.go | herr analyze` - no warnings
5. Check `go test -cover ./internal/protocol` - ≥90% coverage

## 10. Documentation Updates

### Files to Update

1. **docs/packages/protocol.md** - Add section on `ParsedMessage` interface
2. **internal/protocol/protocol.go** - Add package-level documentation
3. **specs/ifacesroadmap.md** - Mark Phase 3.1 complete

### Documentation Content

```markdown
## Type-Safe Message Parsing

All protocol messages implement the `ParsedMessage` interface, providing
compile-time type safety:

\`\`\`go
parsed, err := protocol.ParseMessage(msg)
if err != nil {
    return err
}

switch v := parsed.(type) {
case protocol.TurnStart:
    // Type-safe access to TurnStart fields
case protocol.AssistantDelta:
    // Type-safe access to AssistantDelta fields
// ... handle all message types
}
\`\`\`

The `ParsedMessage` interface uses a sealed interface pattern (unexported
marker method) to prevent external types from implementing it.
```

## 11. Risks and Mitigations

| Risk | Impact | Probability | Mitigation |
|------|--------|-------------|------------|
| Breaking downstream code | High | Low | Comprehensive testing of appserver and TUI |
| Performance regression | Medium | Low | Benchmark parsing functions |
| Incomplete type handling | Medium | Medium | Exhaustive type switch tests |
| Lint failures | Low | Low | Run lint early and often |

## 12. References

- **Roadmap**: specs/ifacesroadmap.md - Phase 3.1
- **Instructions**: instructions/istr-implement.md - Micro-TDD workflow
- **Package Doc**: docs/packages/protocol.md
- **Related FRDs**: 
  - FRD-20251026-event-system-generics.md (similar pattern)
  - FRD-20251026000001-mcp-go-sdk-migration.md (SDK migration approach)

## 13. Approval

**Status**: In Progress  
**Approved By**: N/A (Self-executing agent following roadmap)  
**Implementation Start**: 2025-10-27
