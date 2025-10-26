# FRD: LLM Base Types - JSON Schema Parameters

**Feature ID**: Phase 2.1 - LLM Base Types  
**Priority**: P0  
**Status**: In Progress  
**Created**: 2025-10-26  
**Author**: Claude (Rob Pike persona)  
**Related**: [Empty Interface Elimination Roadmap](../ifacesroadmap.md#21-llm-base-types-priority-p0)

---

## Executive Summary

Replace `Function.Parameters interface{}` with `json.RawMessage` to enable delayed JSON parsing while maintaining type safety. This follows the roadmap guidelines for using `json.RawMessage` for JSON schema parameters that vary by provider.

**Scope**:
- `internal/llm/completion.go` - Update Function.Parameters field

**Change**:
- Replace `Parameters interface{}` with `Parameters json.RawMessage`

---

## Current State

**File: internal/llm/completion.go (Line 160)**

```go
// Function represents a function definition.
type Function struct {
    Name        string
    Description string
    Parameters  interface{} // ← interface{}
}
```

**Issue**: 
- `interface{}` allows any type, reducing type safety
- JSON schema parameters should use `json.RawMessage` for delayed parsing
- Different providers may have different schema formats

---

## Desired State

```go
// Function represents a function definition.
type Function struct {
    Name        string
    Description string
    Parameters  json.RawMessage // ← Delayed JSON parsing
}
```

**Benefits**:
- Type-safe: `json.RawMessage` is specifically for JSON data
- Delayed parsing: Schema validated by each provider
- Standard pattern: Matches Go idioms for JSON schemas

---

## Requirements

### FR1: Update Function.Parameters Type
**Priority**: P0

**Description**: Change Parameters from `interface{}` to `json.RawMessage`

**Acceptance Criteria**:
- Function.Parameters is `json.RawMessage`
- JSON marshaling/unmarshaling works correctly
- Existing tests pass
- No breaking changes to providers

### FR2: Verify Provider Compatibility
**Priority**: P0

**Description**: Ensure OpenAI and Ollama providers still work

**Acceptance Criteria**:
- OpenAI provider tests pass
- Ollama provider tests pass
- Tool schemas serialize correctly

### FR3: Test Coverage
**Priority**: P0

**Description**: Maintain test coverage

**Acceptance Criteria**:
- Existing tests updated
- JSON marshaling tests added
- Coverage maintained at 90%+

---

## Implementation Plan

### Step 1: Update Type (5 min)
1. Change `Parameters interface{}` to `Parameters json.RawMessage`
2. Add `encoding/json` import if needed

### Step 2: Test Compatibility (10 min)
1. Run existing LLM tests
2. Run provider tests (OpenAI, Ollama)
3. Fix any type issues

### Step 3: Add Tests (15 min)
1. Test JSON marshaling
2. Test schema handling
3. Verify round-trip serialization

**Total Time**: ~30 minutes

---

## Testing Strategy

### Unit Tests

```go
func TestFunction_JSONMarshaling(t *testing.T) {
    schema := json.RawMessage(`{"type":"object","properties":{"path":{"type":"string"}}}`)
    fn := Function{
        Name: "read_file",
        Description: "Reads a file",
        Parameters: schema,
    }
    
    // Marshal
    data, err := json.Marshal(fn)
    assert.NoError(t, err)
    
    // Unmarshal
    var decoded Function
    err = json.Unmarshal(data, &decoded)
    assert.NoError(t, err)
    assert.Equal(t, fn.Name, decoded.Name)
    assert.JSONEq(t, string(schema), string(decoded.Parameters))
}
```

---

## Risks

| Risk | Impact | Mitigation |
|------|--------|------------|
| Provider incompatibility | Medium | Test all providers |
| JSON parsing issues | Low | json.RawMessage handles this |
| Breaking changes | Low | Internal type, no external API |

---

## Success Metrics

- ✅ Zero `interface{}` in Function struct
- ✅ All provider tests pass
- ✅ JSON serialization works correctly
- ✅ No performance regression

---

## References

- [Roadmap: Phase 2.1](../ifacesroadmap.md#21-llm-base-types-priority-p0)
- [Roadmap: Use json.RawMessage](../ifacesroadmap.md#-use-jsonrawmessage)
- Go encoding/json documentation

---

**Status**: Ready for Implementation  
**Estimated Time**: 30 minutes
