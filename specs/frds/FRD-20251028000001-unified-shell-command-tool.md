# FRD-20251028000001: Unified Shell Command Tool

## Metadata
- **Status**: Draft
- **Author**: Rob Pike (Claude)
- **Created**: 2025-10-28
- **Updated**: 2025-10-28
- **Related**: 
  - `docs/tool-architecture-analysis.md` - Architecture analysis
  - `specs/ifacesroadmap.md` - Empty interface elimination roadmap

## Problem Statement

Currently, spin has two separate tools for shell command execution:
1. **execute_command** (`internal/tools/execute_command.go`) - Has approval integration
2. **shell_operation** (`internal/shell/operation_tool.go`) - NO approval integration ⚠️

### Critical Issues

**Security Vulnerability (P0)**:
- `shell_operation` bypasses the approval system
- Agent can execute dangerous commands (`rm -rf`, `sudo`) without user approval
- No audit trail for commands executed via `shell_operation`
- Inconsistent security model creates confusion

**Code Duplication**:
- ~80% overlapping functionality between both tools
- Command execution logic duplicated
- Timeout handling duplicated
- Working directory handling duplicated

**Architectural Inconsistency**:
- Arbitrary split between `internal/tools/` and `internal/shell/`
- Different tool capabilities without clear rationale
- Agent must choose between tools with different security models

## Goals

### Primary Goals
1. **Fix Security Hole**: Ensure ALL command execution goes through approval system
2. **Eliminate Duplication**: Single implementation for shell operations
3. **Rich Functionality**: Combine execution + introspection capabilities
4. **Consistent UX**: One tool for all shell interactions

### Non-Goals
- Backward compatibility (per project requirements)
- Gradual migration with deprecation period
- Supporting old tool names

## Proposed Solution

### Unified Tool Design

Create single `shell_command` tool in `internal/tools/shell_command.go` that:
1. Executes commands **with security approval**
2. Provides shell introspection (environment, info)
3. Validates commands before execution
4. Maintains complete audit trail

### Tool Capabilities

```json
{
  "name": "shell_command",
  "operations": [
    "execute",           // Execute command (with approval for dangerous commands)
    "get_environment",   // List environment variables
    "get_shell_info",    // Get shell type, path, configuration
    "detect_shell",      // Check if command requires shell syntax
    "validate"           // Pre-validate command classification
  ]
}
```

### Security Model

**All command execution flows through**:
```
Command → Validator.Classify → ApprovalService (if dangerous) → Executor.Execute
```

**Safe operations** (no approval needed):
- `get_environment` - Read-only introspection
- `get_shell_info` - Read-only introspection
- `detect_shell` - Read-only validation
- `validate` - Read-only classification

**Dangerous operations** (approval required):
- `execute` with dangerous/critical commands (based on Validator classification)

## Implementation Design

### Type Structure

```go
// ShellCommandTool provides unified shell command execution and introspection.
type ShellCommandTool struct {
    executor  *agent.Executor       // For secure command execution
    validator *security.Validator   // For command classification
    shellInfo *shell.Integration    // For shell introspection (optional)
}

// NewShellCommandTool creates the unified shell command tool.
func NewShellCommandTool(
    executor *agent.Executor,
    validator *security.Validator,
    shellInfo *shell.Integration, // optional - can be nil
) *ShellCommandTool

// Execute handles all shell operations.
func (t *ShellCommandTool) Execute(
    ctx context.Context,
    params tools.ToolParameters,
) (tools.ToolResult, error)
```

### Operation Implementations

#### 1. execute Operation

**Parameters**:
```json
{
  "operation": "execute",
  "command": "ls -la /tmp",
  "working_directory": "/home/user",
  "timeout": 30.0
}
```

**Flow**:
1. Extract command string
2. Parse into `security.Command` struct
3. Call `executor.Execute(ctx, cmd, options)`
4. Executor internally:
   - Validates command
   - Classifies with validator (Safe/Dangerous/Critical)
   - Requests approval if needed
   - Executes command
5. Return result with stdout/stderr

**Security**: Full approval integration via executor

#### 2. get_environment Operation

**Parameters**:
```json
{
  "operation": "get_environment"
}
```

**Flow**:
1. Call `shellInfo.GetEnvironmentVars()` if available
2. Otherwise use `os.Environ()`
3. Format as key=value pairs
4. Return as string output

**Security**: Safe, read-only - no approval needed

#### 3. get_shell_info Operation

**Parameters**:
```json
{
  "operation": "get_shell_info"
}
```

**Flow**:
1. Call `shellInfo.GetContextInfo()` if available
2. Return structured info:
   - Shell enabled
   - Shell type (bash, zsh, etc)
   - Shell path
   - Environment variables
3. If shellInfo unavailable, return basic info from `os.Getenv("SHELL")`

**Security**: Safe, read-only - no approval needed

#### 4. detect_shell Operation

**Parameters**:
```json
{
  "operation": "detect_shell",
  "command": "ls | grep test"
}
```

**Flow**:
1. Call `shellInfo.IsShellCommand(command)` if available
2. Otherwise use simple heuristics:
   - Contains pipes: `|`
   - Contains redirects: `>`, `<`, `>>`
   - Contains shell variables: `$`
   - Contains shell builtins: `cd`, `export`, `source`
3. Return boolean result

**Security**: Safe, read-only - no approval needed

#### 5. validate Operation

**Parameters**:
```json
{
  "operation": "validate",
  "command": "rm -rf /tmp/test"
}
```

**Flow**:
1. Parse command into `security.Command`
2. Call `validator.Classify(command)`
3. Return classification result:
   - Classification: Safe/Dangerous/Critical
   - Reason: Why classified this way
   - Needs approval: boolean

**Security**: Safe, read-only - no approval needed

### Tool Schema

```go
func (t *ShellCommandTool) Schema() tools.ToolSchema {
    return tools.ToolSchema{
        Type: "function",
        Function: tools.FunctionSchema{
            Name:        "shell_command",
            Description: "Execute shell commands (with security approval) and inspect shell environment",
            Parameters: tools.ParameterSchema{
                Type: "object",
                Properties: map[string]tools.PropertyDefinition{
                    "operation": {
                        Type:        "string",
                        Description: "Operation: execute, get_environment, get_shell_info, detect_shell, validate",
                        Enum:        []string{"execute", "get_environment", "get_shell_info", "detect_shell", "validate"},
                    },
                    "command": {
                        Type:        "string",
                        Description: "Shell command (required for execute, detect_shell, validate)",
                    },
                    "working_directory": {
                        Type:        "string",
                        Description: "Working directory (optional, for execute operation)",
                    },
                    "timeout": {
                        Type:        "number",
                        Description: "Timeout in seconds (optional, for execute operation, defaults to 30s)",
                    },
                },
                Required: []string{"operation"},
            },
        },
    }
}
```

### Idiomatic Go Interface Design

The tool uses **small, focused interfaces** defined at the point of use (consumer-side), following idiomatic Go principles:

#### Interface Definitions

```go
// commandExecutor defines what we need from agent.Executor.
// The actual agent.Executor satisfies this interface automatically.
type commandExecutor interface {
	Execute(ctx context.Context, cmd interface{}, opts interface{}) (interface{}, error)
}

// commandValidator defines what we need from security.Validator.
// The actual security.Validator satisfies this interface automatically.
type commandValidator interface {
	Classify(cmd interface{}) (interface{}, error)
}

// shellIntegration defines what we need from shell.Integration.
// The actual shell.Integration satisfies this interface automatically.
type shellIntegration interface {
	GetEnvironmentVars() map[string]string
	GetContextInfo() interface{}
	IsShellCommand(command string) bool
}
```

**Key Principles**:
1. **Define interfaces where they're needed** - Interfaces in `shell_command.go`, not in the packages that implement them
2. **Small and focused** - Each interface defines only the methods actually used
3. **Automatic satisfaction** - Actual types (`agent.Executor`, `security.Validator`, `shell.Integration`) automatically satisfy these interfaces
4. **No reflection** - Uses compile-time type safety via interfaces
5. **Import cycle avoidance** - By using interfaces, we avoid direct imports of concrete types

#### Result Access Pattern

For accessing fields from returned interface{} values, we use **adapter interfaces**:

```go
// resultProvider defines interface for accessing command execution results.
type resultProvider interface {
	GetStdout() string
	GetStderr() string
	GetExitCode() int
}

// validationProvider defines interface for accessing validation results.
type validationProvider interface {
	GetClassification() int
	GetReason() string
}
```

**Adapter Implementation**:
```go
// resultAdapter wraps objects to satisfy resultProvider interface
type resultAdapter struct {
	obj interface{}
}

func (a *resultAdapter) GetStdout() string {
	type resultStruct struct {
		Stdout   string
		Stderr   string
		ExitCode int
	}
	if v, ok := a.obj.(*resultStruct); ok {
		return v.Stdout
	}
	return ""
}
```

**Benefits**:
- Works with both real types (agent.Result) and test mocks
- No reflection required
- Type-safe field access
- Minimal adapter overhead
- Test mocks implement getter methods directly

#### Avoiding Common Anti-Patterns

**❌ DON'T**: Use empty `interface{}` with reflection
```go
// Anti-pattern: requires reflection
type Tool struct {
    executor interface{}
}

func (t *Tool) execute() {
    // Would need reflection to call methods
    reflect.ValueOf(t.executor).MethodByName("Execute").Call(...)
}
```

**✅ DO**: Define small interfaces at point of use
```go
// Idiomatic: uses interfaces and type assertions
type commandExecutor interface {
    Execute(ctx context.Context, cmd interface{}, opts interface{}) (interface{}, error)
}

type Tool struct {
    executor commandExecutor
}

func (t *Tool) execute() {
    t.executor.Execute(...) // Direct method call, no reflection
}
```

## Migration Strategy

### Phase 1: Implement Unified Tool

1. Create `internal/tools/shell_command.go`
2. Implement all operations with TDD
3. Write comprehensive tests (90%+ coverage)
4. Register in manager.go

### Phase 2: Remove Old Tools (No Backward Compatibility)

1. Unregister `execute_command` from manager.go
2. Unregister `shell_operation` from manager.go
3. Delete `internal/tools/execute_command.go`
4. Delete `internal/tools/execute_command_test.go`
5. Delete `internal/shell/operation_tool.go`
6. Delete `internal/shell/operation_tool_test.go`

### Phase 3: Update References

1. Update documentation in `docs/packages/tools.md`
2. Update AGENTS.md if tool usage examples exist
3. Update any FRDs referencing old tools

## Testing Strategy

### Unit Tests (90%+ coverage target)

**Test Coverage Areas**:

1. **execute Operation**:
   - Successful command execution
   - Command with working directory
   - Command with timeout
   - Command requiring approval (dangerous)
   - Command requiring approval (critical)
   - Command approval denied
   - Command with modified approval
   - Command timeout error
   - Command execution failure
   - Invalid command parameter
   - Nil executor error

2. **get_environment Operation**:
   - With shell integration available
   - Without shell integration (fallback)
   - Empty environment

3. **get_shell_info Operation**:
   - With shell integration available
   - Without shell integration (fallback)
   - Shell integration disabled

4. **detect_shell Operation**:
   - Commands with pipes
   - Commands with redirects
   - Commands with variables
   - Commands with builtins
   - Simple commands (no shell needed)
   - With shell integration available
   - Without shell integration (fallback)

5. **validate Operation**:
   - Safe command classification
   - Dangerous command classification
   - Critical command classification
   - Invalid command syntax
   - Nil validator error

6. **Error Cases**:
   - Missing operation parameter
   - Unknown operation
   - Missing required parameters per operation
   - Nil dependencies

### Integration Tests

1. Full approval flow with mock ApprovalHandler
2. Command execution with real shell (conditional on CI)
3. Environment introspection accuracy
4. Shell detection accuracy

### Test Utilities

Reuse existing test infrastructure:
- Mock executors from `execute_command_test.go`
- Mock shell integration patterns from `operation_tool_test.go`
- Approval flow patterns from `internal/agent/executor_test.go`

## Code Organization

### File Structure

```
internal/tools/
├── shell_command.go          # NEW: Unified tool implementation
├── shell_command_test.go     # NEW: Comprehensive tests
├── execute_command.go        # DELETE: Old tool
├── execute_command_test.go   # DELETE: Old tests
└── ...other tools...

internal/shell/
├── integration.go            # KEEP: Core shell integration
├── integration_test.go       # KEEP: Integration tests
├── operation_tool.go         # DELETE: Old tool wrapper
└── operation_tool_test.go    # DELETE: Old tests
```

### Registration in Manager

**Before**:
```go
// internal/manager/manager.go:374
_ = registry.Register(tools.NewExecuteCommandTool(executor, validator))

// internal/manager/manager.go:279
shellTool := shell.NewShellOperationTool(m.shellIntegration)
_ = m.toolRegistry.Register(shellTool)
```

**After**:
```go
// internal/manager/manager.go
shellCmd := tools.NewShellCommandTool(executor, validator, m.shellIntegration)
_ = registry.Register(shellCmd)
```

## Security Considerations

### Approval Flow

**All command execution** (`execute` operation) goes through:
1. **Validation**: `validator.Classify(command)`
2. **Approval**: `approvalService.RequestApproval()` if dangerous/critical
3. **Execution**: `executor.Execute()` with approval tracking
4. **Audit**: Events emitted for all stages

**Introspection operations** (read-only):
- No approval needed
- No security risk
- Can query freely

### Threat Model

**Threats Mitigated**:
- ✅ Agent bypassing approval via `shell_operation`
- ✅ Dangerous commands executed without user consent
- ✅ Missing audit trail for shell operations

**Threats Remaining** (out of scope):
- Command injection via malformed parameters (handled by executor)
- Resource exhaustion (handled by timeout + executor limits)
- Privilege escalation (handled by OS security)

## Performance Considerations

**No performance regression expected**:
- Same executor backend (agent.Executor)
- Same shell integration backend (shell.Integration)
- Unified code path eliminates duplicate parsing

**Potential improvements**:
- Command parsing done once (vs twice with two tools)
- Reduced tool registry size (one tool vs two)

## Documentation Updates

### Files to Update

1. **docs/packages/tools.md**:
   - Remove `execute_command` section
   - Remove `shell_operation` section
   - Add `shell_command` section with all operations

2. **docs/packages/core.md**:
   - Update tool integration examples
   - Update security flow diagrams

3. **AGENTS.md** (if applicable):
   - Update tool usage examples
   - Update prompts referencing old tools

4. **docs/tool-architecture-analysis.md**:
   - Add "Implemented" status
   - Link to this FRD

## Risks and Mitigations

### Risk: Breaking Existing Conversations

**Impact**: Medium  
**Likelihood**: High (per requirements: no backward compatibility)  
**Mitigation**: Acceptable - clean cutover as requested

### Risk: Test Coverage Gaps

**Impact**: High (security tool)  
**Likelihood**: Low (90%+ coverage requirement)  
**Mitigation**: Comprehensive test suite with approval flow tests

### Risk: Missing Edge Cases

**Impact**: Medium  
**Likelihood**: Medium  
**Mitigation**: Reuse test cases from both old tools

## Success Criteria

### Functional Requirements
- ✅ All 5 operations implemented and tested
- ✅ Security approval works for `execute` operation
- ✅ Introspection operations work with/without shell integration
- ✅ All edge cases covered with tests

### Quality Requirements
- ✅ 90%+ test coverage
- ✅ Zero lint errors
- ✅ Zero deadcode
- ✅ All tests pass

### Security Requirements
- ✅ No command execution without approval check
- ✅ All dangerous commands trigger approval
- ✅ Complete audit trail via events
- ✅ Command modification validated

## Timeline

**Total Estimate**: 4-6 hours

- Phase 1 (Implementation): 2-3 hours
  - FRD: 30 min (done)
  - Test scaffolding: 30 min
  - TDD cycles: 2 hours
  
- Phase 2 (Cleanup): 1 hour
  - Remove old tools
  - Update registrations
  - Verify builds
  
- Phase 3 (Documentation): 1 hour
  - Update docs
  - Update AGENTS.md
  - Verify all references

## Appendix

### A. Command Classification Reference

From `internal/security/validator.go`:

**Safe Commands** (no approval):
- File reading: `cat`, `head`, `tail`, `less`
- Navigation: `ls`, `pwd`, `cd`
- Information: `echo`, `env`, `which`

**Dangerous Commands** (approval required):
- File modification: `rm`, `mv`, `cp`, `chmod`, `chown`
- Process control: `kill`, `killall`
- Network: `curl`, `wget` with `-O`

**Critical Commands** (approval required + warning):
- Privilege escalation: `sudo`, `su`
- Disk operations: `dd`, `mkfs`, `fdisk`
- System control: `reboot`, `shutdown`, `systemctl`

### B. Event Types Reference

From `internal/events/event.go`:

- `EventCommandApproval` - Approval requested
- `EventCommandApproved` - User approved command
- `EventCommandDenied` - User denied command

### C. Example Agent Prompts

**Before** (confusing):
```
You have two tools for commands:
- execute_command: For safe commands or commands needing approval
- shell_operation: For shell introspection and commands
```

**After** (clear):
```
You have shell_command tool with operations:
- execute: Run commands (approval for dangerous ones)
- get_environment: View environment variables
- get_shell_info: View shell configuration
- detect_shell: Check if command needs shell
- validate: Pre-check command safety
```

---

## References

- Tool Architecture Analysis: `docs/tool-architecture-analysis.md`
- Empty Interface Roadmap: `specs/ifacesroadmap.md`
- Security System: `internal/security/`
- Executor Implementation: `internal/agent/executor.go`
- Shell Integration: `internal/shell/integration.go`
