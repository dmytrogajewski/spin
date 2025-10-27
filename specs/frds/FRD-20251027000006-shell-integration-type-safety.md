# FRD-20251027000006: Shell Integration Type Safety

## Metadata
- **FRD ID**: FRD-20251027000006
- **Title**: Shell Integration Context Type Safety
- **Status**: Draft
- **Created**: 2025-10-27
- **Author**: Claude (Rob Pike persona)
- **Related Documents**: 
  - `specs/ifacesroadmap.md` - Phase 4.2
  - Similar pattern in `internal/git/integration.go`

## 1. Overview

### 1.1 Purpose
Replace `map[string]interface{}` in `GetContextInfo()` with a strongly-typed `ShellContextInfo` struct.

### 1.2 Scope
**In Scope:**
- `internal/shell/context.go` - Define ContextInfo and update GetContextInfo
- `internal/shell/operation_tool.go` - Update usage (formatting)
- `internal/manager/manager.go` - Update usage (type assertions)
- Test files - Already use `tools.ToolParameters` (type-safe)

**Out of Scope:**
- `internal/git/integration.go` - Similar pattern, but separate phase
- ShellOperationTool.Execute - Already uses `tools.ToolParameters` (type-safe)

### 1.3 Background
The `GetContextInfo()` method returns shell context as `map[string]interface{}`, but the actual structure is well-defined:
- `shell_enabled`: bool
- `shell`: string
- `shell_path`: string
- `shell_env`: map[string]string (optional)

This can be represented with a proper struct for compile-time type safety.

## 2. Current State Analysis

### 2.1 Current Interface{} Usage

**Location 1: GetContextInfo return type**
```go
// internal/shell/context.go:103
func (s *Context) GetContextInfo() map[string]interface{} {
    if !s.IsEnabled() {
        return map[string]interface{}{
            "shell_enabled": false,
        }
    }

    info := map[string]interface{}{
        "shell_enabled": true,
        "shell":         s.GetShell(),
        "shell_path":    s.GetShellPath(),
    }

    envVars := s.GetEnvironmentVars()
    if len(envVars) > 0 {
        info["shell_env"] = envVars
    }

    return info
}
```

**Location 2: Usage in operation_tool.go**
```go
// internal/shell/operation_tool.go:130
shellInfo := t.shellIntegration.GetContextInfo()
var output strings.Builder
output.WriteString("Shell Information:\n")
for key, value := range shellInfo {
    output.WriteString(fmt.Sprintf("%s: %v\n", key, value))
}
```

**Location 3: Usage in manager.go**
```go
// internal/manager/manager.go:194
shellInfo := m.shellIntegration.GetContextInfo()
for key, value := range shellInfo {
    if strValue, ok := value.(string); ok {
        env.Context[key] = strValue
    }
}
```

**Location 4: Test files**
```go
// internal/shell/operation_tool_test.go (10 occurrences)
params := map[string]interface{}{
    "operation": "get_shell_info",
}
```

### 2.2 Problems with Current Approach

1. **Type Unsafety**: Callers must use type assertions to access values
2. **No Documentation**: Structure is implicit, not explicit
3. **Error-Prone**: Easy to misspell keys ("shell_enable" vs "shell_enabled")
4. **No IDE Support**: Cannot autocomplete field names

### 2.3 Test Files Are Different

The test files use `map[string]interface{}` but they're **testing tool calls**, which go through `tools.ToolParameters` (already type-safe). These will be updated to use `tools.FromMap()` for clarity.

## 3. Design Decision

### 3.1 Define ShellContextInfo Struct

```go
// ShellContextInfo contains shell context information for the agent.
type ShellContextInfo struct {
    ShellEnabled bool              `json:"shell_enabled"`
    Shell        string            `json:"shell,omitempty"`
    ShellPath    string            `json:"shell_path,omitempty"`
    ShellEnv     map[string]string `json:"shell_env,omitempty"`
}
```

**Benefits:**
- ✅ Compile-time type safety
- ✅ Self-documenting structure
- ✅ IDE autocomplete support
- ✅ JSON serialization still works (for tool output)
- ✅ No runtime type assertions needed

### 3.2 Update GetContextInfo

```go
// BEFORE
func (s *Context) GetContextInfo() map[string]interface{} {
    if !s.IsEnabled() {
        return map[string]interface{}{
            "shell_enabled": false,
        }
    }
    info := map[string]interface{}{
        "shell_enabled": true,
        "shell":         s.GetShell(),
        "shell_path":    s.GetShellPath(),
    }
    envVars := s.GetEnvironmentVars()
    if len(envVars) > 0 {
        info["shell_env"] = envVars
    }
    return info
}

// AFTER
func (s *Context) GetContextInfo() ContextInfo {
    if !s.IsEnabled() {
        return ShellContextInfo{
            ShellEnabled: false,
        }
    }

    return ShellContextInfo{
        ShellEnabled: true,
        Shell:        s.GetShell(),
        ShellPath:    s.GetShellPath(),
        ShellEnv:     s.GetEnvironmentVars(), // Returns empty map if none, omitempty handles it
    }
}
```

## 4. Implementation Plan

### 4.1 Define ContextInfo (context.go)

Add struct definition before `GetContextInfo`:

```go
// ShellContextInfo contains shell context information for the agent.
type ShellContextInfo struct {
    ShellEnabled bool              `json:"shell_enabled"`
    Shell        string            `json:"shell,omitempty"` 
    ShellPath    string            `json:"shell_path,omitempty"`
    ShellEnv     map[string]string `json:"shell_env,omitempty"`
}
```

### 4.2 Update GetContextInfo (integration.go)

Replace implementation as shown in section 3.2.

### 4.3 Update operation_tool.go

```go
// BEFORE
shellInfo := t.shellIntegration.GetContextInfo()
var output strings.Builder
output.WriteString("Shell Information:\n")
for key, value := range shellInfo {
    output.WriteString(fmt.Sprintf("%s: %v\n", key, value))
}

// AFTER
shellInfo := t.shellIntegration.GetContextInfo()
var output strings.Builder
output.WriteString("Shell Information:\n")
output.WriteString(fmt.Sprintf("shell_enabled: %t\n", shellInfo.ShellEnabled))
if shellInfo.ShellEnabled {
    output.WriteString(fmt.Sprintf("shell: %s\n", shellInfo.Shell))
    output.WriteString(fmt.Sprintf("shell_path: %s\n", shellInfo.ShellPath))
    if len(shellInfo.ShellEnv) > 0 {
        output.WriteString("shell_env:\n")
        for k, v := range shellInfo.ShellEnv {
            output.WriteString(fmt.Sprintf("  %s=%s\n", k, v))
        }
    }
}
```

### 4.4 Update manager.go

```go
// BEFORE
shellInfo := m.shellIntegration.GetContextInfo()
for key, value := range shellInfo {
    if strValue, ok := value.(string); ok {
        env.Context[key] = strValue
    }
}

// AFTER
shellInfo := m.shellIntegration.GetContextInfo()
if shellInfo.ShellEnabled {
    env.Context["shell_enabled"] = "true"
    if shellInfo.Shell != "" {
        env.Context["shell"] = shellInfo.Shell
    }
    if shellInfo.ShellPath != "" {
        env.Context["shell_path"] = shellInfo.ShellPath
    }
    // Note: shell_env is a map, not included in Context (which is string->string)
} else {
    env.Context["shell_enabled"] = "false"
}
```

### 4.5 Update Test Files (operation_tool_test.go)

```go
// BEFORE
params := map[string]interface{}{
    "operation": "get_shell_info",
}

// AFTER
params, _ := tools.FromMap(map[string]interface{}{
    "operation": "get_shell_info",
})
```

This makes it clear that tests are using the type-safe `tools.ToolParameters` type.

## 5. Testing Strategy

### 5.1 Existing Tests

Update all test cases in `operation_tool_test.go` to use `tools.FromMap()`.

### 5.2 New Tests (if needed)

Add tests for `ShellContextInfo` JSON marshaling (optional):

```go
func TestShellContextInfo_JSON(t *testing.T) {
    info := ShellContextInfo{
        ShellEnabled: true,
        Shell:        "bash",
        ShellPath:    "/bin/bash",
        ShellEnv:     map[string]string{"PATH": "/usr/bin"},
    }
    
    data, err := json.Marshal(info)
    require.NoError(t, err)
    
    var decoded ShellContextInfo
    err = json.Unmarshal(data, &decoded)
    require.NoError(t, err)
    assert.Equal(t, info, decoded)
}
```

## 6. Impact Analysis

### 6.1 Breaking Changes

**None for external API** - this is all internal code.

### 6.2 Files Affected

- `internal/shell/context.go` (+10 lines struct, ~5 lines method)
- `internal/shell/operation_tool.go` (+10 lines, -5 lines)
- `internal/manager/manager.go` (+8 lines, -4 lines)
- `internal/shell/operation_tool_test.go` (+20 lines for `tools.FromMap`)

### 6.3 Interface{} Eliminated

- `GetContextInfo` return type: 1 occurrence
- `GetContextInfo` implementation: 2 occurrences (2 return statements)
- `operation_tool.go` usage: 1 map iteration
- `manager.go` usage: 1 map iteration with type assertion
- Test files: 10 occurrences (converted to `tools.FromMap`)
- **Total: ~15 interface{} uses eliminated**

## 7. Success Criteria

- [ ] `ShellContextInfo` struct defined
- [ ] `GetContextInfo()` returns `ShellContextInfo`
- [ ] `operation_tool.go` updated to use struct fields
- [ ] `manager.go` updated to use struct fields
- [ ] Test files updated to use `tools.FromMap`
- [ ] All tests pass
- [ ] Build succeeds
- [ ] Zero lint errors

## 8. Future Considerations

### 8.1 Similar Pattern in Git Integration

The `internal/git/integration.go` has a similar `GetContextInfo()` method. This can be addressed in a separate phase with the same pattern.

### 8.2 Consider Helper Methods

If formatting is common, add a helper:

```go
func (s ShellContextInfo) String() string {
    var b strings.Builder
    b.WriteString(fmt.Sprintf("shell_enabled: %t\n", s.ShellEnabled))
    if s.ShellEnabled {
        b.WriteString(fmt.Sprintf("shell: %s\n", s.Shell))
        b.WriteString(fmt.Sprintf("shell_path: %s\n", s.ShellPath))
        if len(s.ShellEnv) > 0 {
            b.WriteString("shell_env:\n")
            for k, v := range s.ShellEnv {
                b.WriteString(fmt.Sprintf("  %s=%s\n", k, v))
            }
        }
    }
    return b.String()
}
```

But YAGNI for now - only used in one place.

## 9. Conclusion

This is a straightforward type safety improvement:
- ✅ Replaces `map[string]interface{}` with proper struct
- ✅ Eliminates ~15 `interface{}` occurrences
- ✅ Improves code clarity and maintainability
- ✅ No breaking changes (internal code only)
- ✅ Pattern can be reused for git integration

**Recommendation**: Proceed with implementation.

---

**Document Version**: 1.0
**Last Updated**: 2025-10-27
**Status**: Ready for Implementation
