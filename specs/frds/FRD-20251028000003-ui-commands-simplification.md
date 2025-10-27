# FRD-20251028000003: UI Commands Simplification

## 1. Overview

### 1.1 Title
UI Commands Dead Feature Removal

### 1.2 Purpose
Remove unused variadic arguments from Command interface to eliminate `interface{}` usage and deadcode.

### 1.3 Scope
**In Scope:**
- `internal/ui/overlay/command.go` - Remove variadic args from Command.Execute
- `internal/ui/overlay/command_test.go` - Update tests
- `internal/ui/overlay/palette_test.go` - Update integration tests
- `internal/ui/overlay/palette_renderer_test.go` - Update renderer tests

**Out of Scope:**
- Backward compatibility - Not maintained per project requirements
- CommandArgs struct - Not needed since args are never used

### 1.4 Related Documents
- `specs/ifacesroadmap.md` - Phase 5.2
- `instructions/istr-implement.md` - Implementation guidelines

---

## 2. Problem Statement

### 2.1 Current Interface{} Usage

The Command interface uses variadic `interface{}` args that are never actually utilized:

```go
type Command interface {
    Execute(ctx context.Context, args ...interface{}) error
}
```

**Analysis of actual usage**:
- 7 calls to Execute() found in codebase
- 6 calls pass NO arguments: `cmd.Execute(context.Background())`
- 1 test passes args but **never uses them**: `Execute(ctx, "arg1", 42, true)`
- NO production code uses arguments
- NO command implementations inspect args

**Conclusion**: This is dead design - a feature that was planned but never implemented.

### 2.2 Why It's a Problem

1. **Type Unsafe**: Accepts any types, provides no compile-time safety
2. **Dead Design**: Args are collected but never used anywhere
3. **Misleading API**: Suggests commands can take arguments when they can't
4. **Interface{} Count**: Contributes ~4 occurrences to elimination goal

### 2.3 Root Cause

The variadic args were likely added for future extensibility but:
- No actual need materialized
- Commands are simple actions without parameters
- Any parametrization happens through context or command-specific setup

---

## 3. Proposed Solution

### 3.1 Design Decision

**Remove variadic args entirely** instead of making them type-safe because:
1. They are never used in practice (deadcode)
2. Project requirement: "Do not introduce new deadcode"
3. Commands don't need runtime arguments
4. Simplifies interface and implementation

### 3.2 Updated Interface

```go
// Command represents an executable action in the palette.
type Command interface {
    Name() string
    Description() string
    Category() string
    Icon() rune
    Execute(ctx context.Context) error  // NO ARGS
}
```

### 3.3 Updated Implementation

```go
type simpleCommand struct {
    name        string
    description string
    category    string
    icon        rune
    exec        func(context.Context) error  // NO ARGS
}

func NewSimpleCommand(name, description, category string, icon rune, exec func(context.Context) error) Command {
    return &simpleCommand{
        name:        name,
        description: description,
        category:    category,
        icon:        icon,
        exec:        exec,
    }
}

func (c *simpleCommand) Execute(ctx context.Context) error {
    if c.exec == nil {
        return nil
    }
    return c.exec(ctx)
}
```

---

## 4. Implementation Plan

### 4.1 Update Command Interface (command.go)

**Changes**:
1. Change `Execute(ctx context.Context, args ...interface{}) error` to `Execute(ctx context.Context) error`
2. Change `exec func(context.Context, ...interface{}) error` to `exec func(context.Context) error`
3. Update `NewSimpleCommand` signature
4. Update `simpleCommand.Execute` implementation
5. Remove `args...` pass-through

**Estimated**: ~5 lines changed

### 4.2 Update Tests (command_test.go)

**Test Cases to Update**:
1. `TestSimpleCommand_Execute_NoFunc` - Already passes no args ✅
2. `TestSimpleCommand_Execute` - Already passes no args ✅
3. `TestSimpleCommand_Execute_WithArgs` - **REMOVE TEST** (tests dead feature)
4. `TestSimpleCommand_Execute_Error` - Already passes no args ✅
5. `TestSimpleCommand_Execute_ContextCancellation` - Already passes no args ✅

**Changes**:
- Remove `TestSimpleCommand_Execute_WithArgs` test (~15 lines)
- No changes needed to other tests

**Estimated**: ~15 lines removed

### 4.3 Update Integration Tests (palette_test.go)

**Changes**:
- Update command creation in `TestPalette_SelectAndExecute_NoCommands`
- Update command creation in tests if any use simpleCommand

**Estimated**: ~2-3 lines changed

### 4.4 Update Renderer Tests (palette_renderer_test.go)

**Changes**:
- Update command creation if any use simpleCommand

**Estimated**: ~1-2 lines changed

---

## 5. Testing Strategy

### 5.1 Unit Tests

```go
// TestCommand_Execute_Simple verifies command execution
func TestCommand_Execute_Simple(t *testing.T) {
    executed := false
    cmd := NewSimpleCommand("Test", "Test", "Test", 'T', func(ctx context.Context) error {
        executed = true
        return nil
    })
    
    err := cmd.Execute(context.Background())
    
    assert.NoError(t, err)
    assert.True(t, executed)
}

// TestCommand_Execute_Error verifies error propagation
func TestCommand_Execute_Error(t *testing.T) {
    cmd := NewSimpleCommand("Test", "Test", "Test", 'T', func(ctx context.Context) error {
        return errors.New("test error")
    })
    
    err := cmd.Execute(context.Background())
    
    assert.Error(t, err)
    assert.Equal(t, "test error", err.Error())
}
```

### 5.2 Coverage Target

- `command.go`: Maintain 100% coverage (simple code)
- Existing tests already cover all paths
- Removal of one test won't affect coverage of actual production code

---

## 6. Migration & Impact

### 6.1 Breaking Changes

**API Change**: Command.Execute signature changes from `(ctx, ...interface{})` to `(ctx)`.

**Impact**: NONE in production
- No production code passes arguments
- All existing calls already use zero args

### 6.2 Files Affected

- `internal/ui/overlay/command.go` (-5 chars in 3 lines, cleaner signature)
- `internal/ui/overlay/command_test.go` (-15 lines, test removal)
- `internal/ui/overlay/palette_test.go` (~2-3 lines updated if needed)
- `internal/ui/overlay/palette_renderer_test.go` (~1-2 lines updated if needed)

### 6.3 Backward Compatibility

Not maintained per project requirement: "Do not maintain backward compatibility".

---

## 7. Benefits

### 7.1 Code Quality

- **Simpler API**: Clear that commands don't take runtime arguments
- **Type Safety**: No `interface{}` usage
- **Less Code**: Removes ~20 lines total (test + signatures)
- **Clearer Intent**: Interface matches actual usage

### 7.2 Interface{} Elimination

- **Before**: 4 occurrences (interface definition + simpleCommand field + NewSimpleCommand param + Execute method)
- **After**: 0 occurrences
- **Reduction**: 100% in this file

### 7.3 Maintainability

- Future developers won't wonder how to pass command arguments
- No confusion about type assertions or argument parsing
- Matches actual usage pattern (zero-arg commands)

---

## 8. Risks & Mitigation

### 8.1 Risks

1. **Low**: Some external code might theoretically implement Command interface
   - **Mitigation**: This is internal UI code, no external users

2. **Low**: Future need for command arguments
   - **Mitigation**: Can add typed params to specific command implementations if needed

### 8.2 Rollback Plan

Simple git revert if issues arise (unlikely given no actual usage of args).

---

## 9. Acceptance Criteria

- [ ] Command.Execute signature has no variadic args
- [ ] All 4 test files compile successfully
- [ ] All existing tests pass (except removed WithArgs test)
- [ ] Zero `interface{}` in command.go
- [ ] `go vet ./internal/ui/overlay/...` passes
- [ ] `make lint` passes
- [ ] Total interface{} count reduced by 4

---

## 10. Implementation Checklist

- [ ] Update Command interface in command.go
- [ ] Update simpleCommand struct in command.go
- [ ] Update NewSimpleCommand in command.go
- [ ] Update simpleCommand.Execute in command.go
- [ ] Remove TestSimpleCommand_Execute_WithArgs from command_test.go
- [ ] Update command creation in palette_test.go (if needed)
- [ ] Update command creation in palette_renderer_test.go (if needed)
- [ ] Run `go test ./internal/ui/overlay/...`
- [ ] Run `go vet ./internal/ui/overlay/...`
- [ ] Run `make lint`
- [ ] Verify interface{} count reduced
- [ ] Update `specs/ifacesroadmap.md` - mark Phase 5.2 complete
- [ ] Commit with message: "feat(ui): remove unused command args (Phase 5.2)"

---

## Document Metadata

**Version**: 1.0
**Created**: 2025-10-28
**Author**: Claude (Rob Pike persona)
**Status**: Draft
**Related FRDs**: FRD-20251028000002 (UI Blocks - already complete)
