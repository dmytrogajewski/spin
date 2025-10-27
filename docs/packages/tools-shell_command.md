# shell_command Tool Documentation

## Overview

**Name:** `shell_command`  
**Status:** Active (replaces deprecated `execute_command` and `shell_operation`)  
**Purpose:** Unified tool for shell command execution with security approval and shell environment introspection

## Description

The `shell_command` tool provides comprehensive shell interaction capabilities with built-in security features. It consolidates command execution, environment inspection, shell detection, and command validation into a single, secure interface.

## Integration

- **Executor**: `agent.Executor` - Provides secure command execution with approval flow
- **Validator**: `security.Validator` - Classifies commands as Safe/Dangerous/Critical
- **Shell Integration**: `shell.Integration` (optional) - Provides shell-specific features

## Operations

### 1. execute
Execute shell commands with automatic security approval for dangerous operations.

**Parameters:**
- `operation`: "execute" (required)
- `command`: Command string (required)
- `working_directory`: Working directory path (optional)
- `timeout`: Timeout in seconds, default 30s (optional)

**Security:** All commands go through validator classification. Dangerous/Critical commands trigger user approval.

**Example:**
```json
{
  "operation": "execute",
  "command": "git status",
  "working_directory": "/path/to/repo",
  "timeout": 60.0
}
```

### 2. get_environment
List shell environment variables.

**Parameters:**
- `operation`: "get_environment" (required)

**Security:** Safe operation, no approval needed.

**Example:**
```json
{
  "operation": "get_environment"
}
```

**Output:**
```
Environment Variables:
PATH=/usr/bin:/bin
HOME=/home/user
SHELL=/bin/bash
```

### 3. get_shell_info
Get shell type, path, and configuration.

**Parameters:**
- `operation`: "get_shell_info" (required)

**Security:** Safe operation, no approval needed.

**Example:**
```json
{
  "operation": "get_shell_info"
}
```

**Output:**
```
Shell Information:
shell_enabled: true
shell: bash
shell_path: /bin/bash
```

### 4. detect_shell
Check if command requires shell execution (pipes, redirects, variables, etc.).

**Parameters:**
- `operation`: "detect_shell" (required)
- `command`: Command to check (required)

**Security:** Safe operation, no approval needed.

**Example:**
```json
{
  "operation": "detect_shell",
  "command": "ls | grep test"
}
```

**Output:**
```
Is shell command: true
```

### 5. validate
Pre-validate command classification without executing.

**Parameters:**
- `operation`: "validate" (required)
- `command`: Command to validate (required)

**Security:** Safe operation, no approval needed. Useful for agents to check before execution.

**Example:**
```json
{
  "operation": "validate",
  "command": "rm -rf /tmp/test"
}
```

**Output:**
```
Command Validation Result:
Classification: Dangerous
Needs Approval: true
Reason: modifies filesystem
```

## Security Model

### Command Classification

**Safe Commands** (no approval):
- File reading: `cat`, `head`, `tail`, `less`
- Navigation: `ls`, `pwd`, `cd`
- Information: `echo`, `env`, `which`

**Dangerous Commands** (approval required):
- File modification: `rm`, `mv`, `cp`, `chmod`, `chown`
- Process control: `kill`, `killall`
- Network: `curl`, `wget` with write operations

**Critical Commands** (approval required + warning):
- Privilege escalation: `sudo`, `su`
- Disk operations: `dd`, `mkfs`, `fdisk`
- System control: `reboot`, `shutdown`, `systemctl`

### Approval Flow

1. Agent requests command execution
2. Validator classifies command
3. If Dangerous/Critical: User approval requested
4. User can approve, deny, or modify command
5. Modified commands are re-validated
6. Approved commands execute
7. Events emitted for audit trail

### No Bypass Guarantee

Unlike the deprecated `shell_operation` tool which bypassed approval, `shell_command` ensures ALL command execution goes through the security layer. Agents cannot bypass approval by choosing different operations.

## Timeout Behavior

- **Default**: 30 seconds for execute operation
- **Per-Command**: Override via `timeout` parameter
- **Context Precedence**: Calling context timeout takes precedence if shorter
- **Error Handling**: Timeout errors include detailed context information

## Code Examples

### Go Implementation

```go
import "github.com/dmytrogajewski/spin/internal/tools"

// Create tool with all integrations
tool := tools.NewShellCommandTool(executor, validator, shellIntegration)

// Execute with approval
params, _ := tools.FromMap(map[string]interface{}{
    "operation": "execute",
    "command": "npm install",
    "timeout": 120.0,
})
result, err := tool.Execute(ctx, params)

// Get environment
params, _ = tools.FromMap(map[string]interface{}{
    "operation": "get_environment",
})
result, err = tool.Execute(ctx, params)

// Validate before execution
params, _ = tools.FromMap(map[string]interface{}{
    "operation": "validate",
    "command": "sudo reboot",
})
result, err = tool.Execute(ctx, params)
if result.Success {
    // Check if approval needed before proposing to agent
}
```

### Tool Registration

```go
// In manager.go
registry := tools.NewRegistry()
_ = registry.Register(tools.NewShellCommandTool(executor, validator, shellIntegration))
```

## Migration from Old Tools

### From execute_command

**Before:**
```json
{
  "command": "git status",
  "working_directory": "/repo",
  "timeout": 60.0
}
```

**After:**
```json
{
  "operation": "execute",
  "command": "git status",
  "working_directory": "/repo",
  "timeout": 60.0
}
```

### From shell_operation

**Before:**
```json
{
  "operation": "execute_command",
  "command": "ls"
}
```

**After:**
```json
{
  "operation": "execute",
  "command": "ls"
}
```

**Important:** Old `shell_operation` did NOT have security approval. New `shell_command` DOES have approval for all executions.

## Event Emissions

- `EventCommandApproval` - Approval requested
- `EventCommandApproved` - User approved command
- `EventCommandDenied` - User denied command

## Error Handling

### Common Errors

**Missing Operation:**
```
Error: operation parameter is required
```

**Unknown Operation:**
```
Error: unknown operation: invalid_op
```

**Missing Command:**
```
Error: command parameter is required for execute operation
```

**Executor Not Configured:**
```
Error: executor not configured
```

**Validator Not Configured:**
```
Error: validator not configured
```

### Timeout Errors

```
Error: context deadline exceeded
Error: shell command timed out after 30s: sleep 60
```

## Testing

See `internal/tools/shell_command_test.go` for comprehensive test examples covering all operations and edge cases.

## Related Documentation

- Security System: `internal/security/`
- Shell Integration: `internal/shell/`
- Executor: `internal/agent/executor.go`
- FRD: `specs/frds/FRD-20251028000001-unified-shell-command-tool.md`
- Architecture Analysis: `docs/tool-architecture-analysis.md`
