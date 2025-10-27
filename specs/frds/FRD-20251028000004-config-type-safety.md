# FRD-20251028000004: Configuration Type Safety

## 1. Overview

### 1.1 Title
Configuration Helper Functions Type Safety

### 1.2 Purpose
Replace `interface{}` usage in configuration helper functions with idiomatic Go types (io.Writer, generics).

### 1.3 Scope
**In Scope:**
- `cmd/spin/config.go` - Replace inline interface with `io.Writer`, make functions generic
- `cmd/spin/mcp.go` - Convert `outputJSON` to generic
- `cmd/spin/config_test.go` - Update tests for new signatures

**Out of Scope:**
- `redactSensitiveValues` - Keep as-is (idiomatic recursive map manipulation)
- Backward compatibility - Not maintained per project requirements

### 1.4 Related Documents
- `specs/ifacesroadmap.md` - Phase 5.3
- `instructions/istr-implement.md` - Implementation guidelines

---

## 2. Problem Statement

### 2.1 Current Interface{} Usage

**Location 1: Inline interface for io.Writer**
```go
// cmd/spin/config.go:324
func printJSON(out interface{ Write([]byte) (int, error) }, data interface{}) error {
    encoder := json.NewEncoder(out)
    encoder.SetIndent("", "  ")
    return encoder.Encode(data)
}

// cmd/spin/config.go:331
func printYAML(out interface{ Write([]byte) (int, error) }, data interface{}) error {
    encoder := yaml.NewEncoder(out)
    encoder.SetIndent(2)
    defer encoder.Close()
    return encoder.Encode(data)
}
```

**Problem**: Using inline interface instead of stdlib `io.Writer`
- Non-idiomatic (io.Writer is the standard interface for this)
- Adds 2 interface{} occurrences unnecessarily

**Location 2: Generic data parameter**
```go
// cmd/spin/config.go:324
func printJSON(out interface{ Write([]byte) (int, error) }, data interface{}) error

// cmd/spin/config.go:331
func printYAML(out interface{ Write([]byte) (int, error) }, data interface{}) error

// cmd/spin/mcp.go:323
func outputJSON(data interface{}) error {
    encoder := json.NewEncoder(os.Stdout)
    encoder.SetIndent("", "  ")
    return encoder.Encode(data)
}
```

**Problem**: `data interface{}` can be replaced with generics
- Go 1.18+ supports generics
- Provides compile-time type safety
- Makes function signatures self-documenting

**Location 3: Recursive map redaction (KEEP AS-IS)**
```go
// cmd/spin/config.go:297
func redactSensitiveValues(m map[string]interface{}) {
    // ... recursive type assertions needed
    if nested, ok := v.(map[string]interface{}); ok {
        redactSensitiveValues(nested)
    }
}
```

**Decision**: Keep as `map[string]interface{}` because:
- This is idiomatic for recursive data manipulation
- Type assertions are necessary for recursion
- Making this generic would require reflection or constraint complexity
- This is an acceptable use case per "Keep As-Is" guidelines

### 2.2 Impact Analysis

**Current count**: 3 interface{} occurrences in config.go, 1 in mcp.go (4 total in production code)
**After changes**: 0 in production code
**Test file**: Keep test occurrences as-is (they test the redaction behavior)

---

## 3. Proposed Solution

### 3.1 Replace Inline Interface with io.Writer

```go
// BEFORE
func printJSON(out interface{ Write([]byte) (int, error) }, data interface{}) error

// AFTER
func printJSON[T any](out io.Writer, data T) error
```

**Benefits**:
- Uses standard library interface
- More idiomatic
- Generic type parameter for data

### 3.2 Convert to Generic Functions

```go
// config.go
func printJSON[T any](out io.Writer, data T) error {
    encoder := json.NewEncoder(out)
    encoder.SetIndent("", "  ")
    return encoder.Encode(data)
}

func printYAML[T any](out io.Writer, data T) error {
    encoder := yaml.NewEncoder(out)
    encoder.SetIndent(2)
    defer encoder.Close()
    return encoder.Encode(data)
}

// mcp.go
func outputJSON[T any](data T) error {
    encoder := json.NewEncoder(os.Stdout)
    encoder.SetIndent("", "  ")
    return encoder.Encode(data)
}
```

### 3.3 Keep redactSensitiveValues As-Is

```go
// UNCHANGED - idiomatic recursive map manipulation
func redactSensitiveValues(m map[string]interface{}) {
    // ... existing implementation
}
```

---

## 4. Implementation Plan

### 4.1 Update config.go Functions

**Changes to printJSON**:
1. Add `import "io"`
2. Change signature: `func printJSON[T any](out io.Writer, data T) error`
3. No implementation changes needed (already uses json.Encoder correctly)

**Changes to printYAML**:
1. Add `import "io"`
2. Change signature: `func printYAML[T any](out io.Writer, data T) error`
3. No implementation changes needed (already uses yaml.Encoder correctly)

**Estimated**: ~2 lines changed per function

### 4.2 Update mcp.go Function

**Changes to outputJSON**:
1. Change signature: `func outputJSON[T any](data T) error`
2. No implementation changes needed

**Estimated**: ~1 line changed

### 4.3 Update Callers

**In config.go**:
- Search for all `printJSON(` and `printYAML(` calls
- Verify type inference works (Go 1.18+ should infer T automatically)
- Update if explicit type parameters needed

**In mcp.go**:
- Search for all `outputJSON(` calls
- Verify type inference works

**Estimated**: Likely zero changes (type inference should work)

### 4.4 Update Tests

**config_test.go**:
- Verify existing tests still pass with new signatures
- Test data is already strongly typed (structs, maps)
- Type inference should handle everything

**Estimated**: Likely zero changes needed

---

## 5. Testing Strategy

### 5.1 Existing Tests

All existing tests should pass without modification:
- `TestConfigShow` - Uses printJSON with os.Stdout
- `TestConfigEdit` - Doesn't use print functions
- `TestConfigValidate` - Doesn't use print functions  
- `TestConfigList` - Uses printJSON
- `TestRedactSensitiveValues` - Tests map redaction (unchanged)

### 5.2 Type Inference Verification

Verify that type parameters are inferred correctly:
```go
// Should work without explicit [T]
printJSON(os.Stdout, configData)  // T inferred as type of configData
printYAML(os.Stdout, configData)  // T inferred as type of configData
outputJSON(serverList)            // T inferred as type of serverList
```

### 5.3 Manual Testing

```bash
# Test config commands
spin config show
spin config list
spin mcp list
```

---

## 6. Migration & Impact

### 6.1 Breaking Changes

**API Changes**:
- `printJSON` signature changes from `(interface{Write}, interface{})` to `[T any](io.Writer, T)`
- `printYAML` signature changes from `(interface{Write}, interface{})` to `[T any](io.Writer, T)`
- `outputJSON` signature changes from `(interface{})` to `[T any](T)`

**Impact**: NONE
- These are private functions (lowercase names)
- Only used within cmd/spin package
- Type inference means call sites don't need changes

### 6.2 Files Affected

- `cmd/spin/config.go` - 2 function signatures, add io import
- `cmd/spin/mcp.go` - 1 function signature
- `cmd/spin/config_test.go` - Verify tests pass (likely no changes)

### 6.3 Backward Compatibility

Not maintained per project requirement: "Do not maintain backward compatibility".

These are private package functions, so no external compatibility concerns.

---

## 7. Benefits

### 7.1 Code Quality

- **Idiomatic**: Uses `io.Writer` instead of inline interface
- **Type Safe**: Generic constraints provide compile-time safety
- **Self-Documenting**: Generic signature shows data can be any type
- **Standard**: Follows Go stdlib patterns (json.Encoder, yaml.Encoder already use generics internally)

### 7.2 Interface{} Elimination

- **Before**: 4 occurrences in production code (2 in config.go, 1 in mcp.go, plus inline interfaces)
- **After**: 0 occurrences in production code (redactSensitiveValues kept as idiomatic)
- **Reduction**: 100% of non-idiomatic usage

### 7.3 Maintainability

- Clearer function signatures
- Better IDE autocomplete and type checking
- Easier to understand what data can be passed

---

## 8. Risks & Mitigation

### 8.1 Risks

1. **Low**: Type inference might fail in some edge cases
   - **Mitigation**: Comprehensive testing of all call sites

2. **Low**: Generic syntax might be unfamiliar to some developers
   - **Mitigation**: This is standard Go 1.18+ syntax, widely adopted

### 8.2 Rollback Plan

Simple git revert if issues arise (unlikely given simple change and private functions).

---

## 9. Acceptance Criteria

- [ ] printJSON signature uses `io.Writer` and generic type parameter
- [ ] printYAML signature uses `io.Writer` and generic type parameter  
- [ ] outputJSON signature uses generic type parameter
- [ ] io package imported in config.go
- [ ] All existing tests pass
- [ ] No compilation errors
- [ ] `go vet ./cmd/spin/...` passes
- [ ] Manual testing of config commands works
- [ ] Interface{} count reduced by 4 in production code

---

## 10. Implementation Checklist

- [ ] Update printJSON signature in config.go
- [ ] Update printYAML signature in config.go
- [ ] Add io import to config.go
- [ ] Update outputJSON signature in mcp.go
- [ ] Run `go build ./cmd/spin/...`
- [ ] Run `go test ./cmd/spin/...`
- [ ] Run `go vet ./cmd/spin/...`
- [ ] Manual test: `spin config show`
- [ ] Manual test: `spin mcp list`
- [ ] Verify interface{} count reduced
- [ ] Update `specs/ifacesroadmap.md` - mark Phase 5.3 complete
- [ ] Commit with message: "feat(config): use io.Writer and generics (Phase 5.3)"

---

## Document Metadata

**Version**: 1.0
**Created**: 2025-10-28
**Author**: Claude (Rob Pike persona)
**Status**: Draft
**Related FRDs**: 
- FRD-20251028000003 (UI Commands)
- FRD-20251028000002 (UI Blocks)
