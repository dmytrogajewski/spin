# FRD: Message System - Typed Structures

**Feature ID**: Phase 1.3 - Message System  
**Priority**: P0  
**Status**: In Progress  
**Created**: 2025-10-26  
**Author**: Claude (Rob Pike persona)  
**Related**: [Empty Interface Elimination Roadmap](../ifacesroadmap.md#13-message-system-priority-p0)

---

## Executive Summary

Replace `interface{}` usage in `internal/message/message.go` with strongly-typed structures for ToolCall, FunctionCall, and Metadata. This provides compile-time type safety, better IDE support, and eliminates runtime type assertions.

**Scope**:
- `internal/message/message.go` - Define ToolCall and FunctionCall structs, update Metadata type

**Changes**:
1. Replace `ToolCalls []interface{}` with `ToolCalls []ToolCall`
2. Create `ToolCall` struct matching orchestration.ToolCall format
3. Create `FunctionCall` struct for function details
4. Replace `Metadata map[string]interface{}` with `Metadata` type alias `map[string]string`

---

## Current State

**File: internal/message/message.go**

```go
type Message struct {
    // ...
    ToolCalls []interface{} `json:"tool_calls,omitempty"`  // ← interface{}
    // ...
    Metadata map[string]interface{} `json:"metadata,omitempty"`  // ← interface{}
}
```

**Issues**:
1. `ToolCalls []interface{}` requires runtime type assertions
2. `Metadata map[string]interface{}` allows non-string values (inconsistent)
3. No type safety or IDE autocomplete
4. No tests exist for message package

---

## Desired State

**Strongly-typed structures:**

```go
// ToolCall represents a tool invocation from an assistant message
type ToolCall struct {
    ID       string       `json:"id"`
    Type     string       `json:"type"`
    Function FunctionCall `json:"function"`
}

// FunctionCall contains function invocation details
type FunctionCall struct {
    Name      string `json:"name"`
    Arguments string `json:"arguments"`
}

// Metadata stores string key-value metadata
type Metadata map[string]string

type Message struct {
    // ...
    ToolCalls []ToolCall `json:"tool_calls,omitempty"`  // ← Typed
    // ...
    Metadata Metadata `json:"metadata,omitempty"`  // ← Typed
}
```

---

## Requirements

### FR1: Define ToolCall Struct
**Priority**: P0

**Description**: Create strongly-typed ToolCall struct

**Acceptance Criteria**:
- Struct has `ID`, `Type`, and `Function` fields
- Matches OpenAI/orchestration.ToolCall format
- JSON tags for serialization
- Comprehensive tests (90%+ coverage)

### FR2: Define FunctionCall Struct
**Priority**: P0

**Description**: Create Function struct for function details

**Acceptance Criteria**:
- Struct has `Name` and `Arguments` fields
- Arguments is JSON string (matches LLM API format)
- JSON tags for serialization
- Tests verify JSON marshaling/unmarshaling

### FR3: Define Metadata Type
**Priority**: P0

**Description**: Create Metadata type alias for string-only values

**Acceptance Criteria**:
- Type alias: `type Metadata map[string]string`
- Enforces string values only
- Compatible with existing usage
- Tests verify type safety

### FR4: Update Message Struct
**Priority**: P0

**Description**: Update Message to use new types

**Acceptance Criteria**:
- `ToolCalls []ToolCall` (not `[]interface{}`)
- `Metadata Metadata` (not `map[string]interface{}`)
- All existing functionality preserved
- JSON serialization works correctly

---

## Implementation Plan

### Phase 1: Define Types (30 min)
1. Create `ToolCall` struct
2. Create `FunctionCall` struct  
3. Create `Metadata` type alias
4. Write structure tests

### Phase 2: Update Message (30 min)
1. Update `Message.ToolCalls` field
2. Update `Message.Metadata` field
3. Write integration tests
4. Verify JSON serialization

### Phase 3: Testing (30 min)
1. Unit tests for each type
2. JSON marshaling/unmarshaling tests
3. Edge case tests
4. Achieve 90%+ coverage

**Total Time**: ~90 minutes

---

## Test Strategy

### Unit Tests

1. **ToolCall Creation**
```go
func TestToolCall_Creation(t *testing.T) {
    tc := ToolCall{
        ID:   "call_1",
        Type: "function",
        Function: FunctionCall{
            Name:      "read_file",
            Arguments: `{"path": "test.go"}`,
        },
    }
    assert.Equal(t, "call_1", tc.ID)
    assert.Equal(t, "read_file", tc.Function.Name)
}
```

2. **JSON Marshaling**
```go
func TestToolCall_JSONMarshaling(t *testing.T) {
    tc := ToolCall{...}
    data, err := json.Marshal(tc)
    assert.NoError(t, err)
    
    var decoded ToolCall
    err = json.Unmarshal(data, &decoded)
    assert.NoError(t, err)
    assert.Equal(t, tc, decoded)
}
```

3. **Metadata Type Safety**
```go
func TestMetadata_StringOnly(t *testing.T) {
    meta := Metadata{
        "key1": "value1",
        "key2": "value2",
    }
    assert.Equal(t, "value1", meta["key1"])
}
```

4. **Message with ToolCalls**
```go
func TestMessage_WithToolCalls(t *testing.T) {
    msg := Message{
        Role: RoleAssistant,
        ToolCalls: []ToolCall{
            {ID: "call_1", Type: "function", Function: FunctionCall{...}},
        },
    }
    assert.Len(t, msg.ToolCalls, 1)
    assert.Equal(t, "call_1", msg.ToolCalls[0].ID)
}
```

---

## Benefits

✅ **Compile-Time Safety**: Type errors caught at compile time  
✅ **IDE Autocomplete**: Full type information for fields  
✅ **No Type Assertions**: Direct field access  
✅ **Clear Intent**: Types self-document structure  
✅ **Consistency**: Matches orchestration.ToolCall format  

---

## Risks

| Risk | Impact | Mitigation |
|------|--------|------------|
| Breaking changes | Low | Message is internal package |
| JSON compatibility | Low | Maintain same JSON structure |
| Existing usage | Medium | Search and update all usages |

---

## Success Metrics

- ✅ Zero `interface{}` in Message struct
- ✅ 90%+ test coverage for message package
- ✅ All tests pass
- ✅ go vet clean
- ✅ JSON serialization preserved

---

## References

- [orchestration/toolcall.go](../../internal/orchestration/toolcall.go) - Reference implementation
- [Empty Interface Elimination Roadmap](../ifacesroadmap.md)
- [Phase 1.2 Event System](./FRD-20251026-event-system-generics.md)

---

**Status**: Ready for Implementation  
**Next**: Write tests using TDD approach
