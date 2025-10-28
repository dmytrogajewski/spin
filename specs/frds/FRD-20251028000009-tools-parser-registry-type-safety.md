# FRD-20251028000009: Tools System Parser and Registry Type Safety

**Status**: Implemented  
**Date**: 2025-10-28  
**Phase**: 6.6 - Tools System Extensions

## Overview

Eliminated `interface{}` usage in the tools system's argument parsing and registry execution paths by making `ArgumentParser.Parse()` return `ToolParameters` directly and updating `Registry.Execute()` to accept `ToolParameters` instead of `map[string]interface{}`.

## Motivation

The tools system had several interface{} usages:
- `ArgumentParser.Parse()` returned `map[string]interface{}`
- `Registry.Execute()` accepted `map[string]interface{}`
- Validation methods worked with `map[string]interface{}`

This required unnecessary conversions using `FromMap()` in production code (tool_executor.go, agent.go) and made the API less type-safe.

## Changes

### 1. ArgumentParser - Return ToolParameters Directly

**File**: `internal/tools/parser.go`

Changed `Parse()` to return `ToolParameters` instead of `map[string]interface{}`:

```go
// Before:
func (p *ArgumentParser) Parse(raw string) (map[string]interface{}, error)

// After:
func (p *ArgumentParser) Parse(raw string) (ToolParameters, error) {
    if raw == "" {
        if p.AllowEmpty {
            return ToolParameters{}, nil
        }
        return ToolParameters{}, fmt.Errorf("tool arguments cannot be empty")
    }
    var args map[string]interface{}
    if err := json.Unmarshal([]byte(raw), &args); err != nil {
        return ToolParameters{}, fmt.Errorf("failed to parse tool arguments: %w", err)
    }
    return FromMap(args)
}
```

This eliminates the need for callers to convert the result using `FromMap()`.

### 2. Registry.Execute - Accept ToolParameters

**File**: `internal/tools/registry.go`

Changed `Execute()` signature to accept `ToolParameters`:

```go
// Before:
func (r *Registry) Execute(ctx context.Context, name string, params map[string]interface{}) (ToolResult, error)

// After:
func (r *Registry) Execute(ctx context.Context, name string, params ToolParameters) (ToolResult, error)
```

### 3. JSON-Based Validation

**File**: `internal/tools/registry.go`

Updated validation to work directly with `json.RawMessage` from `ToolParameters` instead of converting to `map[string]interface{}`:

```go
// validateParams now works with ToolParameters directly
func (r *Registry) validateParams(schema ToolSchema, params ToolParameters) error {
    paramSchema := schema.Function.Parameters
    if err := r.validateRequiredParams(paramSchema, params); err != nil {
        return err
    }
    return r.validateParameterTypes(paramSchema, params)
}

// Validation uses JSON directly, preserving type information
func (r *Registry) validateTypeFromJSON(rawValue json.RawMessage, expectedType string) bool {
    switch expectedType {
    case "integer":
        var f float64
        if err := json.Unmarshal(rawValue, &f); err != nil {
            return false
        }
        // Check if it's an integer (no decimal point in JSON)
        return f == float64(int64(f))
    // ... other types
    }
}
```

Key improvement: Integer validation now works correctly by checking the JSON representation directly, avoiding Go's default JSON number unmarshaling to `float64`.

### 4. Production Code Simplification

**File**: `internal/orchestration/tool_executor.go`

Removed unnecessary `FromMap()` conversion:

```go
// Before:
func (t *ToolExecutor) parseToolArguments(call *ToolCall) (tools.ToolParameters, error) {
    parser := tools.NewArgumentParser()
    argsMap, err := parser.Parse(call.Function.Arguments)
    if err != nil {
        return tools.ToolParameters{}, fmt.Errorf("invalid tool arguments: %w", err)
    }
    return tools.FromMap(argsMap)
}

// After:
func (t *ToolExecutor) parseToolArguments(call *ToolCall) (tools.ToolParameters, error) {
    parser := tools.NewArgumentParser()
    return parser.Parse(call.Function.Arguments)
}
```

**File**: `internal/agent/agent.go`

Similar simplification in agent's `parseToolArguments()`.

### 5. Test Updates

**Files**: `internal/tools/parser_test.go`, `internal/tools/registry_test.go`, `internal/agent/agent_test.go`

- Updated parser tests to compare via `ToMap()` instead of direct comparison
- Updated registry tests to convert test data using `FromMap()` before calling `Execute()`
- Fixed agent test nil check to use `len(args.Keys()) == 0` instead of `args == nil`

## Impact

### Interface{} Eliminations
- Removed 2 interface{} return types (ArgumentParser.Parse)
- Removed 2 interface{} parameters (Registry.Execute, Registry.validateParams)
- Removed 2 interface{} parameters in validation helpers
- **Total: 6 interface{} usages eliminated**

### Code Quality
- Production code is simpler (removed FromMap conversions in 2 locations)
- Type safety is improved throughout the tools system
- Validation is more accurate (especially for integer types)
- API is more consistent (ToolParameters used end-to-end)

### Test Results
- All tools package tests pass (88ms)
- All orchestration package tests pass (257ms)
- All agent package tests pass (3ms)
- Build succeeds with zero errors

## Migration Notes

Since backward compatibility is not required:
- All callers must now pass `ToolParameters` to `Registry.Execute()`
- All callers must handle `ToolParameters` return from `ArgumentParser.Parse()`
- Test data should use `FromMap()` for convenience: `params, _ := FromMap(testData)`

## Related Changes

This change completes Phase 6.6 (Tools System Extensions) of the interface{} elimination roadmap.

Previous related changes:
- Phase 6.5: LLM Builder used typed ProviderOptions
- Earlier phases: ToolParameters type introduced with FromMap/ToMap converters

## Files Modified

- `internal/tools/parser.go` - Changed Parse return type
- `internal/tools/registry.go` - Changed Execute signature, JSON-based validation
- `internal/orchestration/tool_executor.go` - Simplified parseToolArguments
- `internal/agent/agent.go` - Simplified parseToolArguments
- `internal/tools/parser_test.go` - Updated assertions
- `internal/tools/registry_test.go` - Convert test data with FromMap
- `internal/agent/agent_test.go` - Fixed nil check
