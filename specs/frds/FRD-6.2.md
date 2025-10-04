# FRD-6.2: Tool Call Processing

**Feature ID:** 6.2  
**Feature Name:** Tool Call Processing  
**Phase:** 6 - Agent Core  
**Priority:** P0 (Blocker)  
**Estimated Effort:** 12 hours  
**Actual Effort:** ~4 hours  
**Status:** ✅ Complete  

---

## Overview

Implement robust tool call handling that enables the agent to parse, validate, and execute tool calls from LLM responses. This feature completes the agent orchestration by adding the ability to process tool invocations, coordinate with the executor, handle approvals, and format results for the LLM.

## Problem Statement

The current agent implementation (Feature 6.1) has placeholder support for tool calls. To enable the agent to perform actual work (reading files, executing commands, searching code), we need:

- Tool call parsing from LLM responses
- Parameter extraction and validation
- Tool execution coordination with approval workflows
- Result formatting for LLM consumption
- Error handling and recovery
- Timeout management for long-running tools

Without proper tool call processing, the agent cannot interact with the system or accomplish practical tasks.

## Goals

1. **Tool Call Parsing:** Extract and validate tool calls from LLM responses
2. **Parameter Handling:** Extract and validate tool parameters (JSON arguments)
3. **Execution Coordination:** Coordinate tool execution with executor and validator
4. **Approval Workflow:** Integrate command approval for dangerous operations
5. **Result Formatting:** Format tool results for LLM consumption
6. **Error Recovery:** Handle tool failures gracefully
7. **Multi-Tool Support:** Execute multiple tool calls in sequence
8. **Event Integration:** Emit events for tool execution progress

## Non-Goals

- Full tool registry implementation (Phase 8.2)
- Streaming tool output (deferred to optimization phase)
- Concurrent/parallel tool execution (safety first, optimize later)
- Advanced caching of tool results (Phase 8.6)
- Custom tool plugins (Phase 8.2)

## Definition of Ready (DoR)

- [x] Feature 6.1 (Agent Orchestration) completed
- [x] Tool call format defined (OpenAI-compatible)
- [x] Tool result format defined
- [x] Executor available for command execution
- [x] Validator available for safety checks
- [x] Event system available for progress tracking

## Definition of Done (DoD)

### Code Implementation
- [ ] `ProcessToolCall()` fully implemented
- [ ] Tool call parsing from ToolCall structure
- [ ] Parameter extraction from JSON arguments
- [ ] Tool type detection (command, file_read, file_write, etc.)
- [ ] Command execution via executor
- [ ] Approval workflow integration
- [ ] Result formatting for LLM
- [ ] Error handling and wrapping
- [ ] Tool timeout enforcement
- [ ] Event emission for tool lifecycle

### Testing
- [ ] Unit tests for ProcessToolCall (>90% coverage)
- [ ] Parameter parsing tests
- [ ] Tool execution tests
- [ ] Approval workflow tests
- [ ] Error handling tests
- [ ] Timeout tests
- [ ] Integration tests with executor
- [ ] Multi-tool sequence tests
- [ ] Race detector clean

### Quality
- [ ] All tests passing
- [ ] Linter passing
- [ ] Code complexity ≤15
- [ ] Godoc comments complete
- [ ] Error handling follows patterns

### Documentation
- [ ] Tool call flow documented
- [ ] Example tool calls provided
- [ ] Error scenarios documented
- [ ] FRD updated with results

## Technical Design

### Tool Call Structure

The tool call structure follows OpenAI's function calling format:

```go
// ToolCall represents a tool invocation from the LLM
type ToolCall struct {
    ID       string           // Unique identifier
    Type     string           // "function"
    Function ToolCallFunction // Function details
}

// ToolCallFunction contains function call details
type ToolCallFunction struct {
    Name      string // Function name (e.g., "execute_command", "read_file")
    Arguments string // JSON string of arguments
}

// ToolResult represents execution result
type ToolResult struct {
    ID       string // Matches ToolCall.ID
    Success  bool   // Execution success
    Output   string // Tool output
    Error    error  // Error if failed
    ExitCode int    // Exit code (for commands)
}
```

### Supported Tool Types

For Phase 6.2, we'll support these core tool types:

1. **execute_command** - Run shell commands
   ```json
   {
     "command": "ls -la",
     "workdir": "/path/to/dir"
   }
   ```

2. **read_file** - Read file contents
   ```json
   {
     "path": "/path/to/file"
   }
   ```

3. **write_file** - Write file contents
   ```json
   {
     "path": "/path/to/file",
     "content": "file contents"
   }
   ```

4. **list_directory** - List directory contents
   ```json
   {
     "path": "/path/to/dir"
   }
   ```

### ProcessToolCall Implementation

```go
func (a *Agent) ProcessToolCall(ctx context.Context, call *ToolCall) (*ToolResult, error) {
    // 1. Validate tool call
    if err := a.validateToolCall(call); err != nil {
        return &ToolResult{
            ID:      call.ID,
            Success: false,
            Error:   err,
        }, err
    }

    // 2. Parse arguments
    args, err := a.parseToolArguments(call)
    if err != nil {
        return &ToolResult{
            ID:      call.ID,
            Success: false,
            Error:   err,
        }, err
    }

    // 3. Emit tool start event
    a.emitter.Emit(Event{
        Type: EventToolCallStart,
        Data: map[string]interface{}{
            "tool_id":   call.ID,
            "tool_name": call.Function.Name,
        },
    })

    // 4. Execute based on tool type
    var result *ToolResult
    switch call.Function.Name {
    case "execute_command":
        result, err = a.executeCommand(ctx, call.ID, args)
    case "read_file":
        result, err = a.readFile(ctx, call.ID, args)
    case "write_file":
        result, err = a.writeFile(ctx, call.ID, args)
    case "list_directory":
        result, err = a.listDirectory(ctx, call.ID, args)
    default:
        err = fmt.Errorf("unknown tool: %s", call.Function.Name)
        result = &ToolResult{
            ID:      call.ID,
            Success: false,
            Error:   err,
        }
    }

    // 5. Emit completion event
    a.emitter.Emit(Event{
        Type: EventToolCallComplete,
        Data: map[string]interface{}{
            "tool_id": call.ID,
            "success": result.Success,
        },
    })

    return result, err
}
```

### Helper Methods

```go
// validateToolCall validates the tool call structure
func (a *Agent) validateToolCall(call *ToolCall) error

// parseToolArguments extracts and parses JSON arguments
func (a *Agent) parseToolArguments(call *ToolCall) (map[string]interface{}, error)

// executeCommand executes a shell command
func (a *Agent) executeCommand(ctx context.Context, id string, args map[string]interface{}) (*ToolResult, error)

// readFile reads a file's contents
func (a *Agent) readFile(ctx context.Context, id string, args map[string]interface{}) (*ToolResult, error)

// writeFile writes content to a file
func (a *Agent) writeFile(ctx context.Context, id string, args map[string]interface{}) (*ToolResult, error)

// listDirectory lists directory contents
func (a *Agent) listDirectory(ctx context.Context, id string, args map[string]interface{}) (*ToolResult, error)

// formatToolResult formats result for LLM consumption
func (a *Agent) formatToolResult(result *ToolResult) string
```

### Command Execution Flow

```
FUNCTION executeCommand(ctx, id, args):
    1. Extract command from args
    2. Parse command into Command struct
    3. Check if needs approval (ShouldApprove)
    4. IF needs approval:
        a. Emit approval request event
        b. Wait for approval (with timeout)
        c. IF denied: return error result
    5. Execute via executor
    6. Capture output and exit code
    7. Format result
    8. Return ToolResult
```

### Error Handling

```go
// Tool execution errors should be captured but not fail the agent
func (a *Agent) executeCommand(ctx context.Context, id string, args map[string]interface{}) (*ToolResult, error) {
    // Parse command
    cmd, err := a.parseCommand(args)
    if err != nil {
        return &ToolResult{
            ID:      id,
            Success: false,
            Output:  "",
            Error:   fmt.Errorf("invalid command: %w", err),
        }, nil // Return nil error so agent continues
    }

    // Check approval
    if needsApproval, reason := a.ShouldApprove(cmd); needsApproval {
        // Emit approval request
        approved := a.requestApproval(ctx, cmd, reason)
        if !approved {
            return &ToolResult{
                ID:      id,
                Success: false,
                Output:  "",
                Error:   errors.New("command denied by user"),
            }, nil
        }
    }

    // Execute
    execResult, err := a.executor.Execute(ctx, cmd)
    if err != nil {
        return &ToolResult{
            ID:       id,
            Success:  false,
            Output:   execResult.Stderr,
            Error:    err,
            ExitCode: execResult.ExitCode,
        }, nil
    }

    return &ToolResult{
        ID:       id,
        Success:  execResult.ExitCode == 0,
        Output:   execResult.Stdout,
        ExitCode: execResult.ExitCode,
    }, nil
}
```

### Approval Request Handling

For Phase 6.2, we'll implement a synchronous approval mechanism:

```go
func (a *Agent) requestApproval(ctx context.Context, cmd *Command, reason string) bool {
    // Emit approval request event
    a.emitter.Emit(Event{
        Type: EventCommandApproval,
        Data: map[string]interface{}{
            "command": cmd.Raw,
            "reason":  reason,
        },
    })

    // For testing, we'll use a channel-based approach
    // In real implementation, this will integrate with UI
    if a.approvalHandler != nil {
        return a.approvalHandler(cmd, reason)
    }

    // Default to deny if no handler
    return false
}
```

## Implementation Plan

### Step 1: Tool Call Validation (1.5 hours)
1. Implement `validateToolCall()`
2. Check required fields (ID, Type, Function.Name)
3. Validate JSON arguments structure
4. Write validation tests

### Step 2: Argument Parsing (1.5 hours)
1. Implement `parseToolArguments()`
2. Parse JSON arguments string
3. Handle parsing errors
4. Write parsing tests

### Step 3: Command Execution (3 hours)
1. Implement `executeCommand()`
2. Parse command from arguments
3. Integrate with executor
4. Handle approval workflow
5. Format results
6. Write command execution tests

### Step 4: File Operations (2 hours)
1. Implement `readFile()`
2. Implement `writeFile()`
3. Implement `listDirectory()`
4. Add error handling
5. Write file operation tests

### Step 5: ProcessToolCall Integration (2 hours)
1. Complete `ProcessToolCall()` implementation
2. Add tool type routing
3. Add event emission
4. Write integration tests

### Step 6: Error Handling & Edge Cases (1 hour)
1. Add timeout handling
2. Add resource cleanup
3. Handle unknown tools
4. Write error scenario tests

### Step 7: Multi-Turn Agent Loop (1 hour)
1. Update Execute() to handle tool calls
2. Add tool result to message history
3. Continue loop until completion
4. Write multi-turn tests

## Test Plan

### Unit Tests

```go
// Test tool call validation
func TestAgent_validateToolCall(t *testing.T)
func TestAgent_validateToolCall_InvalidID(t *testing.T)
func TestAgent_validateToolCall_MissingFunction(t *testing.T)

// Test argument parsing
func TestAgent_parseToolArguments(t *testing.T)
func TestAgent_parseToolArguments_InvalidJSON(t *testing.T)
func TestAgent_parseToolArguments_EmptyArgs(t *testing.T)

// Test command execution
func TestAgent_executeCommand(t *testing.T)
func TestAgent_executeCommand_WithApproval(t *testing.T)
func TestAgent_executeCommand_Denied(t *testing.T)
func TestAgent_executeCommand_ExecutionError(t *testing.T)
func TestAgent_executeCommand_Timeout(t *testing.T)

// Test file operations
func TestAgent_readFile(t *testing.T)
func TestAgent_readFile_NotFound(t *testing.T)
func TestAgent_writeFile(t *testing.T)
func TestAgent_writeFile_PermissionDenied(t *testing.T)
func TestAgent_listDirectory(t *testing.T)

// Test ProcessToolCall
func TestAgent_ProcessToolCall_Command(t *testing.T)
func TestAgent_ProcessToolCall_ReadFile(t *testing.T)
func TestAgent_ProcessToolCall_WriteFile(t *testing.T)
func TestAgent_ProcessToolCall_UnknownTool(t *testing.T)
func TestAgent_ProcessToolCall_Events(t *testing.T)

// Test multi-turn with tools
func TestAgent_Execute_WithToolCalls(t *testing.T)
func TestAgent_Execute_MultipleTools(t *testing.T)
```

### Integration Tests

```go
func TestAgent_Integration_ToolExecution(t *testing.T)
func TestAgent_Integration_ApprovalWorkflow(t *testing.T)
func TestAgent_Integration_ErrorRecovery(t *testing.T)
```

### Test Coverage Targets
- **ProcessToolCall:** >90%
- **Helper methods:** >85%
- **Error paths:** 100%
- **Overall agent package:** >88%

## Dependencies

### Internal Dependencies
- `internal/core/agent.go` (Feature 6.1) ✅
- `internal/core/executor.go` (Feature 2.2) ✅
- `internal/core/validator.go` (Feature 2.1) ✅
- `internal/core/event.go` (Feature 4.1) ✅
- `internal/core/message.go` ✅

### External Dependencies
- `encoding/json` (stdlib)
- `os` (stdlib)
- `path/filepath` (stdlib)

## Risk Assessment

### High Risk
- **Approval workflow complexity:** Synchronous approval in async execution context
  - *Mitigation:* Use channels and timeouts, comprehensive testing

### Medium Risk
- **Tool execution safety:** Commands could be dangerous
  - *Mitigation:* Robust validation, approval for dangerous commands
  
- **Error propagation:** Tool failures should not crash agent
  - *Mitigation:* Careful error wrapping, return ToolResult with error info

### Low Risk
- **JSON parsing:** Well-understood problem
  - *Mitigation:* Standard library json package, validation

## Success Criteria

1. ✅ Tool calls are parsed correctly from LLM responses
2. ✅ Commands are executed via executor with proper safety checks
3. ✅ File operations work correctly (read, write, list)
4. ✅ Approval workflow functions for dangerous commands
5. ✅ Tool failures are handled gracefully
6. ✅ Events are emitted for tool lifecycle
7. ✅ >90% test coverage achieved
8. ✅ All tests passing with race detector clean
9. ✅ Multi-turn conversations work with tool calls
10. ✅ Code complexity ≤15

## Example Usage

### Execute Command

```go
call := &ToolCall{
    ID:   "call_123",
    Type: "function",
    Function: ToolCallFunction{
        Name:      "execute_command",
        Arguments: `{"command": "ls -la", "workdir": "/tmp"}`,
    },
}

result, err := agent.ProcessToolCall(ctx, call)
if err != nil {
    log.Printf("Tool execution failed: %v", err)
}

fmt.Printf("Success: %v\n", result.Success)
fmt.Printf("Output: %s\n", result.Output)
```

### Read File

```go
call := &ToolCall{
    ID:   "call_124",
    Type: "function",
    Function: ToolCallFunction{
        Name:      "read_file",
        Arguments: `{"path": "/path/to/file.go"}`,
    },
}

result, err := agent.ProcessToolCall(ctx, call)
fmt.Printf("File contents: %s\n", result.Output)
```

## Open Questions

1. **Q: Should we support parallel tool execution?**
   - A: No, sequential execution for safety and simplicity in Phase 6.2

2. **Q: How to handle approval timeouts?**
   - A: Use context with timeout, default to deny after timeout

3. **Q: What's the max tool output size?**
   - A: Limit to 10KB initially, truncate with warning

4. **Q: Should we cache tool results?**
   - A: No, defer to Phase 8.6 (Performance Optimization)

## Related Documents

- [FRD-6.1: Agent Orchestration](./FRD-6.1.md)
- [FRD-2.2: Command Executor](./FRD-2.2.md)
- [FRD-2.1: Command Validator](./FRD-2.1.md)
- [Core Module Spec](../core-module/spec.md)
- [ROADMAP](../core-module/ROADMAP.md)

## References

- [OpenAI Function Calling](https://platform.openai.com/docs/guides/function-calling)
- [Anthropic Claude Tools](https://docs.anthropic.com/claude/docs/tool-use)

---

**Created:** October 4, 2025  
**Author:** AI Development Agent  
**Reviewers:** Development Team  
**Last Updated:** October 4, 2025

