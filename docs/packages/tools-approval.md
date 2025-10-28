# Tool Approval System

## Overview

The universal tool approval system allows any tool to declare its approval requirements based on risk levels. This replaces the previous shell-only approval system with a flexible, type-safe approach.

## Architecture

### Core Types (`internal/tools/approval.go`)

#### RiskLevel
```go
type RiskLevel int

const (
    RiskSafe     RiskLevel = iota // No approval needed
    RiskLow                        // Read operations
    RiskMedium                     // Single file modifications
    RiskHigh                       // Multiple file modifications or patches
    RiskCritical                   // System file modifications
)
```

#### ApprovalNeeds
```go
type ApprovalNeeds struct {
    Required bool      // Whether approval is needed
    Risk     RiskLevel // Risk level of the operation
    Reason   string    // Human-readable explanation
}
```

#### ToolWithApproval Interface
```go
type ToolWithApproval interface {
    Tool
    CheckApproval(params ToolParameters) ApprovalNeeds
}
```

## Tool Implementations

### write_file Tool

Risk classification:
- **RiskCritical**: Writing to system paths (`/etc/`, `/sys/`, `/usr/`)
- **RiskHigh**: Writing executable files (`.sh`, `.go`, `.py`, `.rb`, `.pl`, `.js`, `.ts`)
- **RiskMedium**: Writing any other file

Example:
```go
tool := NewWriteFileTool()
params, _ := tools.FromMap(map[string]interface{}{
    "path": "/etc/passwd",
    "content": "...",
})

needs := tool.CheckApproval(params)
// needs.Required = true
// needs.Risk = RiskCritical
// needs.Reason = "Writing to system path: /etc/passwd"
```

### apply_patch Tool

Risk classification:
- **RiskHigh**: All patch operations (can modify multiple files)
- **RiskSafe**: Empty or invalid patch

Example:
```go
tool := NewApplyPatchTool("/workspace")
params, _ := tools.FromMap(map[string]interface{}{
    "patch_text": "*** a/file.go\n...",
})

needs := tool.CheckApproval(params)
// needs.Required = true
// needs.Risk = RiskHigh
// needs.Reason = "Applying patch can modify multiple files"
```

## Orchestration Integration

The `ToolExecutor` automatically checks approval requirements before executing tools:

```go
// In ToolExecutor.Execute()
if toolWithApproval, ok := tool.(tools.ToolWithApproval); ok {
    needs := toolWithApproval.CheckApproval(args)
    if needs.Required {
        return &ToolResult{
            ID:      call.ID,
            Success: false,
            Error:   fmt.Errorf("approval required: %s (risk: %s)", needs.Reason, needs.Risk),
        }, nil
    }
}
```

## Test Coverage

- `approval.go` core types: **85.7%**
- `write_file.CheckApproval()`: **91.7%**
- `apply_patch.CheckApproval()`: **100%**
- `ToolExecutor` approval check: **100%**

## Usage

### Implementing Approval in a New Tool

1. Implement the `ToolWithApproval` interface:
```go
func (t *MyTool) CheckApproval(params ToolParameters) ApprovalNeeds {
    // Extract parameters
    path, err := params.GetString("path")
    if err != nil || path == "" {
        return ApprovalNeeds{Required: false, Risk: RiskSafe}
    }
    
    // Check conditions and return appropriate risk level
    if isSystemPath(path) {
        return ApprovalNeeds{
            Required: true,
            Risk:     RiskCritical,
            Reason:   fmt.Sprintf("Operating on system path: %s", path),
        }
    }
    
    return ApprovalNeeds{
        Required: true,
        Risk:     RiskMedium,
        Reason:   fmt.Sprintf("Operating on: %s", path),
    }
}
```

2. Write tests following TDD:
```go
func TestMyTool_CheckApproval(t *testing.T) {
    tool := NewMyTool()
    params, _ := tools.FromMap(map[string]interface{}{
        "path": "/etc/config",
    })
    
    needs := tool.CheckApproval(params)
    
    if !needs.Required {
        t.Error("should require approval")
    }
    if needs.Risk != RiskCritical {
        t.Errorf("got risk %v, want RiskCritical", needs.Risk)
    }
}
```

3. The `ToolExecutor` will automatically enforce approval requirements.

## Design Decisions

1. **Interface composition**: `ToolWithApproval` extends `Tool` interface
2. **Risk-based classification**: 5 levels from Safe to Critical
3. **Tool declares needs**: Each tool knows its own approval requirements
4. **Orchestration enforces**: `ToolExecutor` checks before execution
5. **Type-safe**: No empty interfaces or `interface{}`
6. **Minimal**: No backward compatibility, no dead code

## Future Work

- Integration with `ApprovalService` for actual user approval flow
- Event emission for approval requests
- Policy-based approval (e.g., auto-approve RiskLow for certain users)
- Approval history and audit trail
