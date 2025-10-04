# FRD-8.9: ExecuteCommandTool.Execute() Implementation

**Feature ID**: FRD-8.9
**Status**: In Progress
**Priority**: High
**Created**: 2025-10-04
**Related**: [missing.md](../refactoring/missing.md)

---

## Overview

Implement the `ExecuteCommandTool.Execute()` method to enable proper tool-based command execution without requiring delegation through `Agent.ProcessToolCall()`. This resolves the stub implementation and makes the tool fully functional as a standalone component.

## Current State

### Location
[internal/tools/builtin.go:261-267](../../internal/tools/builtin.go#L261)

### Current Implementation
```go
func (t *ExecuteCommandTool) Execute(ctx context.Context, params map[string]interface{}) (ToolResult, error) {
    // Note: This is a stub implementation
    // The actual implementation will delegate to core.Executor
    // This is implemented in agent.go as executeCommand()
    return ToolResult{
        Success: false,
        Error:   "execute_command must be called through Agent.ProcessToolCall",
    }, nil
}
```

### Current Architecture
- `ExecuteCommandTool` holds `executor` and `validator` as `interface{}` types to avoid circular dependency
- Actual command execution logic exists in `agent.go:executeCommand()` (lines 760-820)
- The tool is constructed with dependencies but doesn't use them
- Current approach delegates all execution to Agent, bypassing the Tool interface

## Problem Statement

The stub implementation has several issues:
1. **Incomplete abstraction**: Tool interface not fully implemented
2. **Tight coupling**: Requires Agent.ProcessToolCall() instead of direct Tool.Execute()
3. **Inconsistency**: Other tools (ReadFile, WriteFile, ListDirectory) are fully functional
4. **Testing limitations**: Can't test tool in isolation without Agent

## Requirements

### Functional Requirements

#### FR-1: Direct Command Execution
- `ExecuteCommandTool.Execute()` must execute commands without requiring Agent
- Must use injected `executor` and `validator` dependencies
- Must support all parameters: `command` (required), `workdir` (optional)

#### FR-2: Parameter Validation
- Validate `command` parameter is non-empty string
- Validate `workdir` parameter is string (if provided)
- Return proper ToolResult with error for invalid parameters

#### FR-3: Command Parsing
- Parse command string into program + arguments
- Handle empty commands gracefully
- Support complex command strings with multiple arguments

#### FR-4: Working Directory Handling
- Use `workdir` parameter if provided
- Fallback to executor's default working directory if not provided
- Validate working directory exists

#### FR-5: Command Execution
- Delegate actual execution to `executor.Execute()`
- Pass proper context for cancellation support
- Return execution results in ToolResult format

#### FR-6: Error Handling
- Distinguish between validation errors and execution errors
- Return ToolResult with Success=false for errors
- Include descriptive error messages in ToolResult.Error field

### Non-Functional Requirements

#### NFR-1: No Circular Dependencies
- Must not import `internal/core` package
- Continue using `interface{}` for executor and validator types
- Use type assertions to access required methods

#### NFR-2: Backward Compatibility
- Must maintain existing tool schema
- Must work with existing Agent.ProcessToolCall() flow
- Should not break existing tests

#### NFR-3: Type Safety
- Validate executor and validator are not nil
- Handle type assertion failures gracefully
- Return clear errors for missing or invalid dependencies

## Design

### Approach

The implementation will:
1. Add runtime type assertions to convert `interface{}` to concrete types
2. Replicate the core logic from `agent.go:executeCommand()`
3. Remove approval logic (that's Agent's responsibility, not Tool's)
4. Focus purely on command execution

### Type Assertions

```go
type executorInterface interface {
    Execute(ctx context.Context, cmd *Command, opts *ExecuteOptions) (*ExecuteResult, error)
}
```

We'll assert `t.executor` implements a minimal interface with required methods.

### Execution Flow

```
1. Validate dependencies (executor not nil)
2. Extract and validate parameters (command, workdir)
3. Parse command into Command struct
4. Type assert executor to get Execute() method
5. Call executor.Execute() with command
6. Convert ExecuteResult to ToolResult
7. Return ToolResult with success/error
```

### Removed Functionality

The following will NOT be included (Agent's responsibility):
- Command approval logic (`ShouldApprove()`, `requestApproval()`)
- Context integration (using agent's context)
- Event emission for tool execution

### Working Directory Logic

```go
// If workdir provided in params, use it
if workDir, ok := args["workdir"].(string); ok {
    cmd.WorkDir = workDir
} else {
    // Don't set WorkDir - let executor use its default
    cmd.WorkDir = ""
}
```

## Implementation Plan

### Phase 1: Define Interfaces (for type assertions)
```go
type commandExecutor interface {
    Execute(ctx context.Context, cmd interface{}, opts interface{}) (interface{}, error)
}
```

### Phase 2: Implement Validation
- Check executor is not nil
- Validate command parameter
- Validate workdir parameter (optional)

### Phase 3: Implement Command Parsing
- Extract command string
- Parse into program + args
- Create Command struct

### Phase 4: Implement Execution
- Type assert executor
- Call Execute method
- Handle execution errors

### Phase 5: Result Conversion
- Convert execution result to ToolResult
- Format output (stdout + stderr)
- Set Success flag based on exit code

## Testing Strategy

### Unit Tests

#### Test Cases
1. **TestExecuteCommandTool_NilExecutor**: Verify error when executor is nil
2. **TestExecuteCommandTool_InvalidCommand**: Test with empty/invalid command
3. **TestExecuteCommandTool_InvalidWorkdir**: Test with invalid workdir type
4. **TestExecuteCommandTool_SimpleCommand**: Test basic command execution (e.g., `echo hello`)
5. **TestExecuteCommandTool_CommandWithArgs**: Test command with multiple arguments
6. **TestExecuteCommandTool_WithWorkdir**: Test command with custom working directory
7. **TestExecuteCommandTool_CommandFailure**: Test handling of failed commands (non-zero exit)
8. **TestExecuteCommandTool_ExecutionError**: Test handling of execution errors

### Integration Tests
- Test ExecuteCommandTool through Agent.ProcessToolCall() (existing tests should pass)
- Verify backward compatibility with existing workflows

### Test Dependencies
```go
// Create mock executor for testing
type mockExecutor struct {
    executeFunc func(context.Context, *core.Command, *core.ExecuteOptions) (*core.ExecuteResult, error)
}
```

## Acceptance Criteria

- [ ] ExecuteCommandTool.Execute() successfully executes commands without Agent
- [ ] All unit tests pass (8 test cases minimum)
- [ ] Existing integration tests continue to pass
- [ ] No circular dependency between tools and core packages
- [ ] Error messages are clear and descriptive
- [ ] Working directory parameter works correctly
- [ ] Command parsing handles edge cases (empty, complex commands)
- [ ] Documentation updated to remove "stub implementation" note

## Migration Notes

### Before
```go
// Tool can only be used through Agent
result := agent.ProcessToolCall(ctx, toolCall)
```

### After
```go
// Tool can be used directly
tool := tools.NewExecuteCommandTool(executor, validator)
result, err := tool.Execute(ctx, params)

// Or through Agent (backward compatible)
result := agent.ProcessToolCall(ctx, toolCall)
```

## Open Questions

1. **Validator usage**: The current `agent.executeCommand()` doesn't use the validator. Should we implement validation in the tool, or keep validator for future use?
   - **Decision**: Keep validator field but don't use it yet. It's for future extensibility.

2. **Approval logic**: Should tools handle approval, or is that purely Agent's concern?
   - **Decision**: Approval is Agent's responsibility. Tools should focus on execution only.

3. **Context vs workdir**: How to handle default working directory when workdir not specified?
   - **Decision**: Don't set WorkDir if not provided - let executor use its default.

## References

- Current implementation: [internal/tools/builtin.go:213-268](../../internal/tools/builtin.go#L213)
- Agent's executeCommand: [internal/core/agent.go:760-820](../../internal/core/agent.go#L760)
- Missing tasks: [specs/refactoring/missing.md#L41-46](../refactoring/missing.md#L41)
- Executor interface: [internal/core/executor.go](../../internal/core/executor.go)

## Success Metrics

- 100% test coverage for ExecuteCommandTool.Execute()
- Zero regression in existing tests
- Tool can execute commands in isolation (without Agent)
- Clear error messages for all failure modes

---

**Implementation Status**: 🔨 In Progress
**Next Steps**: Implement Execute() method and write tests
