# FRD-8.12: Strict Parameter Validation for Tool Registry

**Status**: Draft
**Created**: 2025-10-04
**Component**: `internal/tools`
**Priority**: Low
**Complexity**: Low

---

## 1. Overview

This FRD defines the policy for handling unknown parameters in tool execution. Currently, the `Registry.validateParams()` method silently accepts unknown parameters (parameters not defined in the tool's schema). This document establishes a strict validation policy that rejects unknown parameters to prevent errors and improve tool call reliability.

---

## 2. Background

### Current Behavior

In [registry.go:116-119](../../internal/tools/registry.go#L116), the `validateParams()` method has the following behavior:

```go
for name, value := range params {
    propDef, exists := paramSchema.Properties[name]
    if !exists {
        // Unknown parameter - could be allowed or rejected depending on policy
        // For now, we'll allow unknown parameters
        continue
    }
    // ... type validation ...
}
```

### Problem Statement

Allowing unknown parameters can lead to:
1. **Silent failures**: LLM sends wrong parameter names, tool doesn't receive expected data
2. **API misuse**: Tool calls with typos or outdated parameter names go undetected
3. **Debugging difficulty**: No feedback when parameter names are incorrect
4. **Schema drift**: Tools evolve but callers aren't informed of invalid parameters

### Design Decision

**Adopt strict validation**: Reject unknown parameters with a clear error message.

**Rationale**:
- **Fail-fast principle**: Errors should be caught early and reported clearly
- **API contract enforcement**: Tool schemas define explicit contracts
- **LLM feedback**: Clear errors help LLMs self-correct in multi-turn conversations
- **Production safety**: Prevents silent data loss or incorrect tool behavior

---

## 3. Requirements

### Functional Requirements

**FR-8.12.1**: Unknown Parameter Rejection
The registry MUST reject tool calls containing parameters not defined in the tool's schema.

**FR-8.12.2**: Clear Error Messages
Error messages MUST specify:
- Which parameter is unknown
- The tool name
- Available parameter names (for debugging)

**FR-8.12.3**: Backward Compatibility
This change modifies validation behavior and MAY break existing code that passes unknown parameters.

### Non-Functional Requirements

**NFR-8.12.1**: Performance
Parameter validation overhead MUST remain negligible (<1% of total execution time).

**NFR-8.12.2**: Error Quality
Error messages MUST be actionable and help developers/LLMs correct the issue.

---

## 4. Design

### Implementation Approach

Modify `validateParams()` in [registry.go:114-120](../../internal/tools/registry.go#L114) to reject unknown parameters:

```go
// Check parameter types
for name, value := range params {
    propDef, exists := paramSchema.Properties[name]
    if !exists {
        // Build list of valid parameter names for helpful error message
        validParams := make([]string, 0, len(paramSchema.Properties))
        for pname := range paramSchema.Properties {
            validParams = append(validParams, pname)
        }
        return fmt.Errorf("%w: unknown parameter %q (valid parameters: %v)",
            ErrInvalidParameters, name, validParams)
    }

    if !r.validateType(value, propDef.Type) {
        return fmt.Errorf("%w: parameter %s has wrong type (expected %s)",
            ErrInvalidParameters, name, propDef.Type)
    }

    // Check enum values
    if len(propDef.Enum) > 0 {
        if err := r.validateEnum(value, propDef.Enum); err != nil {
            return fmt.Errorf("%w: parameter %s %v", ErrInvalidParameters, name, err)
        }
    }
}
```

### Error Message Format

Example error:
```
invalid parameters: unknown parameter "fliename" (valid parameters: [filename path mode])
```

This helps identify typos (e.g., "fliename" vs "filename").

### Alternative Considered: Lenient Mode

**Option**: Add a configuration flag for strict vs. lenient validation.

**Rejected because**:
- Adds complexity for minimal benefit
- Strict validation is the correct default for production systems
- If flexibility is needed, individual tools can use variadic parameters or `map[string]interface{}`

---

## 5. Test Strategy

### Test Cases

**TC-8.12.1**: Reject Unknown Parameter
- **Given**: Tool with schema defining `param1` and `param2`
- **When**: Execute with `{param1: "value", unknown_param: "value"}`
- **Then**: Return error containing "unknown parameter \"unknown_param\""

**TC-8.12.2**: Accept All Known Parameters
- **Given**: Tool with schema defining `param1` and `param2`
- **When**: Execute with `{param1: "value", param2: "value"}`
- **Then**: Execute successfully

**TC-8.12.3**: Helpful Error Message
- **Given**: Tool with schema defining `filename`, `path`, `mode`
- **When**: Execute with `{fliename: "test.txt"}`  (typo)
- **Then**: Error message lists all valid parameter names

**TC-8.12.4**: Empty Parameters Valid
- **Given**: Tool with no required parameters
- **When**: Execute with `{}`
- **Then**: Execute successfully

**TC-8.12.5**: Required Parameter Check Still Works
- **Given**: Tool with required parameter `filename`
- **When**: Execute with `{}`
- **Then**: Return error for missing required parameter (existing behavior)

---

## 6. Implementation Plan

### Phase 1: Test Development
1. Add test cases to `registry_test.go`:
   - `TestRegistryExecute_UnknownParameter`
   - `TestRegistryExecute_UnknownParameter_ErrorMessage`
   - `TestRegistryExecute_AllKnownParameters`

### Phase 2: Implementation
1. Modify `validateParams()` in `registry.go`
2. Update error message to include valid parameter names
3. Remove TODO comment at line 117-118

### Phase 3: Validation
1. Run all existing tests to verify backward compatibility
2. Run new tests to verify strict validation
3. Analyze code with `uast parse` and `herr analyze` if needed

### Phase 4: Documentation
1. Update missing.md to mark task as completed
2. Document breaking change (if needed for migration guide)

---

## 7. Acceptance Criteria

- [ ] All new test cases pass
- [ ] All existing tests pass (or are updated appropriately)
- [ ] Error messages clearly indicate unknown parameter and list valid ones
- [ ] Code analysis (uast/herr) shows no issues
- [ ] Task marked complete in missing.md

---

## 8. Future Considerations

### Possible Extensions

1. **Parameter name suggestions**: Use Levenshtein distance to suggest correct names
   ```
   unknown parameter "fliename" (did you mean "filename"?)
   ```

2. **Deprecation warnings**: Allow parameters marked as deprecated with warnings
   ```go
   Properties: map[string]PropertyDefinition{
       "old_param": {Type: "string", Deprecated: true},
   }
   ```

3. **Additional properties**: Support JSON Schema `additionalProperties: false` explicitly

---

## 9. References

- [JSON Schema Validation Specification](https://json-schema.org/draft/2020-12/json-schema-validation.html)
- [OpenAPI Parameter Object](https://swagger.io/specification/#parameter-object)
- Related code: [registry.go:102-136](../../internal/tools/registry.go#L102)
- Related tests: [registry_test.go:232-374](../../internal/tools/registry_test.go#L232)

---

## 10. Changelog

| Date | Author | Change |
|------|--------|--------|
| 2025-10-04 | System | Initial FRD creation |
