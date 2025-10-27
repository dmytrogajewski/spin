# Tool Architecture Analysis: execute_command vs shell_operation

## Executive Summary

This document analyzes the architectural differences between `execute_command` and `shell_operation` tools, their security models, and provides recommendations for consolidation.

**Key Findings:**
1. **Duplication**: Both tools execute shell commands with ~80% overlapping functionality
2. **Security Model Inconsistency**: `execute_command` has approval integration, `shell_operation` does not
3. **Package Location**: Tools are split between `internal/tools/` (builtin) and `internal/shell/` (shell context specific)
4. **Recommendation**: Consolidate into single tool with rich functionality in `internal/tools/`

---

## 1. Tool Comparison

### 1.1 execute_command Tool

**Location**: `internal/tools/execute_command.go`  
**Registration**: `internal/manager/manager.go:374` (always registered as builtin)

**Capabilities**:
- ✅ Execute shell commands
- ✅ Working directory support
- ✅ Timeout control (parameter-based)
- ✅ Reflection-based executor integration
- ✅ Command parsing (strings.Fields)
- ❌ NO environment variable inspection
- ❌ NO shell detection
- ❌ NO shell info queries

**Security Integration**:
```go
// internal/tools/execute_command.go:18
func NewExecuteCommandTool(executor, validator interface{}) *ExecuteCommandTool {
    return &ExecuteCommandTool{
        executor:  executor,  // agent.Executor - has approval service
        validator: validator, // security.Validator - unused currently
    }
}
```

**Approval Flow**:
- Executor has embedded `ApprovalService` (`internal/agent/executor.go`)
- Validator classifies commands as Safe/Dangerous/Critical
- Dangerous/Critical commands trigger approval request
- Approval flow: Executor → ApprovalService → ApprovalHandler → User

**Parameters**:
```json
{
  "command": "string (required)",
  "working_directory": "string (optional)",
  "timeout": "number (optional, seconds)"
}
```

### 1.2 shell_operation Tool

**Location**: `internal/shell/operation_tool.go`  
**Registration**: `internal/manager/manager.go:279` (conditional - only if Shell context enabled)

**Capabilities**:
- ✅ Execute shell commands (`execute_command` operation)
- ✅ Working directory support
- ✅ Timeout control (parameter-based)
- ✅ Get environment variables (`get_environment` operation)
- ✅ Get shell info (`get_shell_info` operation)
- ✅ Check if command requires shell (`is_shell_command` operation)
- ✅ Shell detection and initialization

**Security Integration**:
```go
// internal/shell/operation_tool.go:18
func NewShellOperationTool(shellContext *Context) *ShellOperationTool {
    return &ShellOperationTool{
        shellContext: shellContext, // NO validator, NO approval service
    }
}
```

**Approval Flow**:
- ⚠️ **NONE** - No approval integration at all
- Commands execute directly via `Context.ExecuteShellCommand`
- No security classification
- No validation

**Parameters**:
```json
{
  "operation": "execute_command|get_environment|get_shell_info|is_shell_command (required)",
  "command": "string (required for execute_command/is_shell_command)",
  "args": "array (optional)",
  "working_directory": "string (optional)",
  "timeout": "number (optional, seconds)"
}
```

---

## 2. Security Model Deep Dive

### 2.1 How Security Works for execute_command

**Classification System** (`internal/security/validator.go`):
```go
type CommandClassification int

const (
    CommandSafe     CommandClassification = iota // ls, pwd, echo
    CommandDangerous                              // rm, chmod, kill
    CommandCritical                               // sudo, dd, mkfs
)
```

**Approval Service Flow** (`internal/security/approval.go`):
```
1. Agent calls execute_command tool
2. ExecuteCommandTool.Execute() calls executor.Execute()
3. Executor.Execute() checks if approval needed:
   - Validator.Classify(command) → Safe/Dangerous/Critical
   - If Dangerous/Critical: approvalService.RequestApproval()
4. ApprovalService emits EventCommandApproval
5. ApprovalHandler (TUI) shows prompt to user
6. User approves/denies/modifies command
7. ApprovalService validates modified command
8. Command executes or fails
```

**Event Emissions**:
- `EventCommandApproval` - Approval requested
- `EventCommandApproved` - User approved
- `EventCommandDenied` - User denied

**Command Modification**:
Users can modify dangerous commands:
```
Original: rm -rf /tmp/dangerous
Modified: rm -rf /tmp/safe/specific-file
```
Modified command is re-validated before execution.

### 2.2 How Security Works for shell_operation

**Current State**: ⚠️ **NO SECURITY**

```go
// internal/shell/operation_tool.go:100
result, err := t.shellContext.ExecuteShellCommand(cmdCtx, command)
// Direct execution - no classification, no approval, no validation
```

**Problems**:
1. **Bypass Risk**: Agent can use `shell_operation` to bypass approval for dangerous commands
2. **No Audit Trail**: No approval events emitted
3. **Inconsistent UX**: User sees approvals for some commands, not others
4. **Security Hole**: Critical commands (sudo, rm -rf) execute without approval

**Example Attack Scenario**:
```
Agent thinks: "execute_command requires approval, let me use shell_operation instead"
Agent calls: shell_operation("execute_command", command="sudo rm -rf /")
Result: Executes without approval ❌
```

---

## 3. Why Two Tools Exist (Historical Context)

### 3.1 Package Organization Rationale

**Hypothesis**: Separation by concern
- `internal/tools/` - Core builtins (file operations, commands)
- `internal/shell/` - Shell-specific context (detection, environment)

**Problem**: This creates artificial boundary where none should exist.

### 3.2 Functionality Split

**execute_command** focuses on:
- Command execution with security
- Integration with agent.Executor
- Part of core tool set

**shell_operation** focuses on:
- Shell introspection (get_environment, get_shell_info)
- Shell detection (is_shell_command)
- Command execution (duplicate!)

**Reality**: The split is arbitrary. Shell introspection could be separate tools:
- `get_environment` tool
- `get_shell_info` tool  
- `detect_shell_command` tool

These don't need to bundle command execution.

---

## 4. Why Not Use One Tool with More Rich Functionality?

### 4.1 Proposed Unified Tool Design

**Name**: `shell_command` (replaces both)  
**Location**: `internal/tools/shell_command.go`  
**Package**: Builtin tools

**Capabilities** (superset of both):
```json
{
  "operations": [
    "execute",           // Execute command (with approval)
    "get_environment",   // List environment variables
    "get_shell_info",    // Get shell type, path, config
    "detect_shell",      // Check if command needs shell
    "validate"           // Pre-validate command classification
  ]
}
```

**Security Integration**:
```go
func NewShellCommandTool(
    executor *agent.Executor,      // For execution + approval
    validator *security.Validator,  // For classification
    shellInfo *shell.Context,   // For introspection (optional)
) *ShellCommandTool
```

**Benefits**:
1. ✅ Single tool for all shell operations
2. ✅ Consistent security for all executions
3. ✅ No bypass opportunities
4. ✅ Complete audit trail
5. ✅ Rich introspection capabilities
6. ✅ Simpler agent prompts

### 4.2 Migration Path

**Phase 1: Add Unified Tool**
```go
// internal/tools/shell_command.go
type ShellCommandTool struct {
    executor  *agent.Executor
    validator *security.Validator
    shellInfo *shell.Context // optional
}

func (t *ShellCommandTool) Execute(ctx context.Context, params tools.ToolParameters) (tools.ToolResult, error) {
    operation := params.GetStringOr("operation", "execute")
    
    switch operation {
    case "execute":
        return t.executeCommand(ctx, params) // With approval
    case "get_environment":
        return t.getEnvironment()
    case "get_shell_info":
        return t.getShellInfo()
    case "detect_shell":
        return t.detectShell(params)
    case "validate":
        return t.validateCommand(params)
    }
}
```

**Phase 2: Deprecate Old Tools**
- Keep `execute_command` as alias (backwards compat)
- Keep `shell_operation` as alias (backwards compat)
- Add deprecation warnings in logs

**Phase 3: Remove Old Tools**
- Delete `execute_command.go`
- Delete `shell/operation_tool.go`
- Update all references

### 4.3 Alternative: Fix shell_operation Security

If consolidation is too risky, minimum fix:

```go
// internal/shell/operation_tool.go
type ShellOperationTool struct {
    shellContext *Context
    executor         *agent.Executor      // ADD THIS
    validator        *security.Validator  // ADD THIS
}

func (t *ShellOperationTool) Execute(ctx context.Context, params tools.ToolParameters) (tools.ToolResult, error) {
    operation, _ := params.GetString("operation")
    
    switch operation {
    case "execute_command":
        // CHANGE: Use executor instead of direct shell execution
        return t.executeWithApproval(ctx, params) // Calls executor.Execute()
    case "get_environment":
        return t.getEnvironment() // Safe, no approval needed
    // ...
    }
}
```

**Pros**: Fixes security hole  
**Cons**: Still have two tools, still confusing

---

## 5. Recommendation

### 5.1 Short Term (Immediate Fix)

**Priority: P0 - Security Hole**

Add approval integration to `shell_operation`:
1. Add `executor` and `validator` to `ShellOperationTool`
2. Route `execute_command` operation through `executor.Execute()`
3. Keep introspection operations as-is (safe)
4. Add tests for approval flow

**Impact**: Minimal code change, fixes security bypass

### 5.2 Long Term (Architectural Cleanup)

**Priority: P1 - Technical Debt**

Consolidate into unified tool:
1. Create `internal/tools/shell_command.go` with all functionality
2. Add approval integration (from execute_command)
3. Add introspection operations (from shell_operation)
4. Deprecate old tools with aliases
5. Update documentation and agent prompts

**Impact**: Cleaner architecture, better UX, no duplicate code

### 5.3 Rationale for Consolidation

**Principle**: One tool per domain concept

Shell command execution is ONE concept:
- Execution (with security)
- Introspection (environment, shell info)
- Validation (classification, detection)

Current split violates Single Responsibility:
- `execute_command`: "I execute commands (with security)"
- `shell_operation`: "I execute commands (without security) AND introspect shell"

Better design:
- `shell_command`: "I manage all shell interactions (with security)"

---

## 6. Security Model Best Practices

### 6.1 Current Security Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                          Agent                               │
│  (decides which tool to call based on LLM reasoning)        │
└────────────┬─────────────────────────────┬──────────────────┘
             │                             │
             │                             │
    ┌────────▼─────────┐         ┌────────▼──────────────┐
    │ execute_command  │         │  shell_operation      │
    │  (with approval) │         │  (NO approval) ❌     │
    └────────┬─────────┘         └────────┬──────────────┘
             │                             │
             │                             │
    ┌────────▼──────────┐         ┌────────▼──────────────┐
    │ agent.Executor    │         │ shell.Context         │
    │ + ApprovalService │         │ (direct execution)    │
    │ + Validator       │         │                       │
    └───────────────────┘         └───────────────────────┘
```

**Security Gap**: Agent can choose `shell_operation` to bypass approval.

### 6.2 Recommended Security Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                          Agent                               │
│  (all tools route through security layer)                   │
└────────────────────────────┬────────────────────────────────┘
                             │
                             │
                  ┌──────────▼───────────┐
                  │   shell_command      │
                  │ (unified interface)  │
                  └──────────┬───────────┘
                             │
                  ┌──────────▼───────────┐
                  │  Security Layer      │
                  │  - Validator         │
                  │  - ApprovalService   │
                  │  - Executor          │
                  └──────────┬───────────┘
                             │
              ┌──────────────┴──────────────┐
              │                             │
     ┌────────▼─────────┐         ┌────────▼──────────────┐
     │ Command Execute  │         │ Shell Introspection   │
     │ (with approval)  │         │ (safe, read-only)     │
     └──────────────────┘         └───────────────────────┘
```

**Key Principles**:
1. **Single Entry Point**: All command execution goes through one secure path
2. **Defense in Depth**: Multiple security layers (classification → validation → approval)
3. **No Bypass**: No way to execute commands without security checks
4. **Audit Trail**: All operations emit events for monitoring
5. **Least Privilege**: Introspection operations don't need approval

---

## 7. Implementation Checklist

### 7.1 Quick Fix (shell_operation security)

- [ ] Add `executor` field to `ShellOperationTool`
- [ ] Add `validator` field to `ShellOperationTool`
- [ ] Update `NewShellOperationTool` signature
- [ ] Route `execute_command` operation through executor
- [ ] Update `manager.go` to pass executor and validator
- [ ] Add approval flow tests
- [ ] Update documentation
- [ ] Create FRD for security fix

**Estimated Effort**: 2-3 hours

### 7.2 Consolidation (unified tool)

- [ ] Design unified `ShellCommandTool` API
- [ ] Implement in `internal/tools/shell_command.go`
- [ ] Migrate execute_command functionality
- [ ] Migrate shell_operation functionality
- [ ] Add comprehensive tests (90%+ coverage)
- [ ] Create deprecation aliases
- [ ] Update manager.go registration
- [ ] Update agent prompts
- [ ] Update documentation
- [ ] Create migration guide
- [ ] Remove old tools (after deprecation period)

**Estimated Effort**: 1-2 days

---

## 8. Appendix: Code References

### A.1 Approval Flow Code

**ApprovalService.RequestApproval** (`internal/security/approval.go:53-105`):
- Generates unique request ID
- Emits approval request event
- Invokes approval handler (blocks for user input)
- Validates response
- Handles command modification
- Emits approved/denied event

**Executor.Execute** (`internal/agent/executor.go:150+`):
- Validates command
- Classifies command (Validator.Classify)
- Requests approval if dangerous/critical
- Executes command
- Returns result

### A.2 Registration Code

**execute_command** (`internal/manager/manager.go:374`):
```go
_ = registry.Register(tools.NewExecuteCommandTool(executor, validator))
```

**shell_operation** (`internal/manager/manager.go:279`):
```go
shellTool := shell.NewShellOperationTool(m.shellContext)
if err := m.toolRegistry.Register(shellTool); err != nil {
    logger.Warn("failed to register Shell operation tool", "error", err)
}
```

### A.3 Tool Interface

All tools implement (`internal/tools/tool.go`):
```go
type Tool interface {
    Name() string
    Description() string
    Schema() ToolSchema
    Execute(ctx context.Context, params ToolParameters) (ToolResult, error)
}
```

---

## 9. Conclusion

The current dual-tool architecture has a **critical security vulnerability** where `shell_operation` bypasses the approval system. This must be fixed immediately.

Long-term, consolidating into a single `shell_command` tool provides:
- Better security (single secure path)
- Better UX (consistent behavior)
- Better code quality (no duplication)
- Better maintainability (one tool to maintain)

**Recommended Action**:
1. **Immediate**: Fix shell_operation security (P0)
2. **Next Sprint**: Design unified tool (P1)
3. **Following Sprint**: Implement and migrate (P1)

---

**Document Metadata**:
- **Author**: Analysis by Claude Code
- **Date**: 2025-10-28
- **Version**: 1.0
- **Status**: Recommendation - Awaiting Review
