# FRD-20251027000004: Orchestration Metadata Type Safety

## Metadata
- **FRD ID**: FRD-20251027000004
- **Title**: Orchestration Metadata Type Safety Improvements
- **Status**: Draft
- **Created**: 2025-10-27
- **Author**: Claude (Rob Pike persona)
- **Related Documents**:
  - `specs/ifacesroadmap.md` - Phase 3.3
  - FRD-20251027000002 - JSON-RPC Layer Type Safety

## 1. Overview

### 1.1 Purpose
Eliminate `interface{}` usage in orchestration package metadata fields by converting to `json.RawMessage`.

### 1.2 Scope
**In Scope:**
- `internal/orchestration/turn.go` - Turn.Metadata field
- `internal/orchestration/plan.go` - Plan.Metadata field
- `internal/orchestration/orchestration_test.go` - Test updates

**Out of Scope:**
- Tool executor (already type-safe with `tools.ToolParameters`)
- Other orchestration types
- Backward compatibility (per project requirements)

### 1.3 Background
The orchestration package has two metadata fields using `map[string]interface{}`:
1. `Turn.Metadata` - Extensible metadata for turn tracking
2. `Plan.Metadata` - Additional context for execution plans

Neither field is currently accessed or manipulated - they exist for future extensibility and JSON serialization.

## 2. Current State Analysis

### 2.1 Current Interface{} Usage

**Location 1: Turn.Metadata**
```go
// internal/orchestration/turn.go:66
type Turn struct {
    // ... other fields
    Metadata map[string]interface{} `json:"metadata,omitempty"` // Extensible metadata
}
```

**Location 2: Plan.Metadata**
```go
// internal/orchestration/plan.go:108
type Plan struct {
    // ... other fields
    Metadata map[string]interface{} // Additional context
}
```

**Usage Analysis:**
```bash
# Searching for metadata access patterns
$ grep -r "\.Metadata\[" internal/orchestration/
# Result: No matches - metadata is never accessed!

# Metadata is only:
# 1. Declared in struct definitions
# 2. Initialized in tests: make(map[string]interface{})
# 3. Serialized to/from JSON
```

### 2.2 Problems with Current Approach

1. **Type Unsafety**: No compile-time guarantees about metadata contents
2. **No IDE Support**: Cannot autocomplete or validate metadata structure
3. **Runtime Errors**: Type assertions would fail at runtime if used
4. **Unused Complexity**: Fields exist but are never accessed - pure overhead

### 2.3 Why json.RawMessage is Perfect Here

Since metadata is:
- ✅ Never accessed directly in code
- ✅ Only used for JSON serialization
- ✅ Meant for extensibility
- ✅ Optional (omitempty)

Using `json.RawMessage`:
- ✅ Maintains JSON flexibility
- ✅ Eliminates `interface{}`
- ✅ Zero behavior change (transparent migration)
- ✅ Future-proof for when metadata IS needed

## 3. Design Decision

### 3.1 Selected Approach: json.RawMessage

**Rationale:**
1. **Transparent Migration**: Since metadata is never accessed, switching to `json.RawMessage` has zero behavioral impact
2. **Type Safety**: Eliminates `interface{}`
3. **JSON Compatible**: Still serializable/deserializable
4. **Consistency**: Matches pattern from Phase 3.2 (JSON-RPC layer)
5. **Future-Proof**: When metadata IS needed, consumers can unmarshal into specific types

**Alternative Considered: Remove Fields**
- ❌ Breaks API (even though unused)
- ❌ Loses extensibility
- ❌ Would require re-adding later if needed

**Alternative Considered: Define Specific Types**
- ❌ No use cases exist yet to define structure
- ❌ Over-engineering for unused feature
- ❌ Reduces flexibility

## 4. Implementation Plan

### 4.1 Update Turn.Metadata

```go
// BEFORE
type Turn struct {
    // ... other fields
    Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// AFTER
type Turn struct {
    // ... other fields
    Metadata json.RawMessage `json:"metadata,omitempty"`
}
```

### 4.2 Update Plan.Metadata

```go
// BEFORE
type Plan struct {
    // ... other fields
    Metadata map[string]interface{} // Additional context
}

// AFTER
type Plan struct {
    // ... other fields
    Metadata json.RawMessage `json:"metadata,omitempty"` // Additional context
}
```

**Note**: Added `omitempty` to Plan.Metadata for consistency with Turn.

### 4.3 Update Tests

```go
// BEFORE (orchestration_test.go:362)
Metadata: make(map[string]interface{}),

// AFTER
Metadata: nil, // or json.RawMessage(`{}`), or omit entirely
```

### 4.4 Add Helper Methods (Optional)

If/when metadata needs to be accessed:

```go
// ParseMetadata unmarshals metadata into a target struct
func (t *Turn) ParseMetadata(target interface{}) error {
    if len(t.Metadata) == 0 {
        return nil
    }
    return json.Unmarshal(t.Metadata, target)
}

// SetMetadata marshals a value into metadata
func (t *Turn) SetMetadata(value interface{}) error {
    data, err := json.Marshal(value)
    if err != nil {
        return err
    }
    t.Metadata = data
    return nil
}
```

**Decision**: Do NOT add these methods now (YAGNI). Add when actually needed.

## 5. Testing Strategy

### 5.1 Test Coverage

Since metadata is unused, tests only need to verify:
1. ✅ Structs marshal to JSON correctly
2. ✅ Structs unmarshal from JSON correctly
3. ✅ Empty/nil metadata is handled correctly

**Existing Coverage:**
- `turn.go`: Already has comprehensive state machine tests
- `plan.go`: Already has plan execution tests
- Metadata is incidental in existing tests

**New Tests Needed:**
```go
func TestTurn_Metadata_JSON(t *testing.T) {
    // Test nil metadata
    // Test empty metadata
    // Test metadata with JSON content
    // Test round-trip marshal/unmarshal
}

func TestPlan_Metadata_JSON(t *testing.T) {
    // Same tests as Turn
}
```

## 6. Migration Steps

1. ✅ Read and analyze current implementation
2. ✅ Write FRD
3. Update `turn.go` - Change Metadata type
4. Update `plan.go` - Change Metadata type
5. Update `orchestration_test.go` - Fix test initialization
6. Add metadata JSON tests
7. Run `go vet` and `go fmt`
8. Run full test suite
9. Update roadmap

## 7. Impact Analysis

### 7.1 Breaking Changes

**None!** Since metadata is never accessed, this is a pure internal refactoring.

### 7.2 Files Affected

- `internal/orchestration/turn.go` (+1 line, -1 line)
- `internal/orchestration/plan.go` (+1 line, -1 line)
- `internal/orchestration/orchestration_test.go` (+1 line, -1 line)
- `internal/orchestration/turn_test.go` (new tests: +30 lines)
- `internal/orchestration/plan_test.go` (new tests: +30 lines, if exists)

**Total Impact**: ~6 lines changed, ~60 lines added (tests)

### 7.3 Interface{} Eliminated

- Turn.Metadata: 1 occurrence
- Plan.Metadata: 1 occurrence
- Test initialization: 1 occurrence
- **Total: 3 occurrences**

## 8. Success Criteria

- [ ] No `interface{}` in Turn.Metadata
- [ ] No `interface{}` in Plan.Metadata
- [ ] All existing tests pass
- [ ] New metadata JSON tests pass
- [ ] Zero lint errors
- [ ] Documentation updated

## 9. Future Considerations

### 9.1 When Metadata IS Needed

If future code needs to access metadata:

```go
// Example: Store task mode in turn metadata
type TurnMetadata struct {
    TaskMode string `json:"task_mode,omitempty"`
}

// Set metadata
metadata := TurnMetadata{TaskMode: "review"}
turn.Metadata, _ = json.Marshal(metadata)

// Read metadata
var metadata TurnMetadata
json.Unmarshal(turn.Metadata, &metadata)
```

### 9.2 Alternative: Typed Metadata

If a common metadata structure emerges, consider:

```go
type TurnMetadata struct {
    TaskMode string                 `json:"task_mode,omitempty"`
    Tags     []string               `json:"tags,omitempty"`
    Custom   map[string]interface{} `json:"custom,omitempty"` // Still flexible
}

type Turn struct {
    // ...
    Metadata TurnMetadata `json:"metadata,omitempty"`
}
```

But for now: **YAGNI** (You Aren't Gonna Need It)

## 10. Conclusion

This is a **trivial, zero-risk migration**:
- ✅ No code uses metadata currently
- ✅ Pure type safety improvement
- ✅ Consistent with JSON-RPC layer (Phase 3.2)
- ✅ Future-proof for when metadata IS needed
- ✅ Eliminates 3 `interface{}` occurrences

**Recommendation**: Proceed with implementation.

---

## Appendix A: Roadmap Updates

**Phase 3.3 Orchestration:**
- [x] Analyze tool_executor.go - Already type-safe (uses tools.ToolParameters)
- [ ] Update Turn.Metadata to json.RawMessage
- [ ] Update Plan.Metadata to json.RawMessage
- [ ] Write tests
- [ ] Mark complete

**Note**: The roadmap mentions "parseToolArguments" but it already returns `tools.ToolParameters` (type-safe). No work needed there.

---

**Document Version**: 1.0
**Last Updated**: 2025-10-27
**Status**: Ready for Implementation
