# FRD-20260120-003: Approval Logic Centralization Review

**Created:** 2026-01-20
**Author:** Architecture Analysis
**Status:** COMPLETED (No Changes Required)

## Summary

This FRD documents the review of roadmap item 2.5 "Scattered Approval Logic" and confirms that the centralization work has already been completed through previous refactoring efforts.

## Original Problem Statement

The roadmap identified approval decision-making spread across multiple locations:
- `internal/security/validator.go:145` - `NeedsApproval()`
- `internal/tools/write_file.go:77` - `CheckApproval()`
- `internal/security/approval.go:58` - `RequestApproval()`
- `cmd/spin/approval_handlers.go` - Policy store building

The proposed solution was:
```go
type ApprovalDecisionService interface {
    ShouldApprove(ctx context.Context, op Operation) (ApprovalDecision, error)
    RequestApproval(ctx context.Context, op Operation) (ApprovalResponse, error)
}
```

## Analysis

### Current Architecture

After thorough code analysis, the approval logic is now well-centralized:

#### 1. SecurityService (`internal/security/security.go`)

Provides the central coordination for all security operations:

```go
type SecurityService struct {
    validator       *Validator
    approvalService *ApprovalService
}

// ValidateAndApprove combines validation + approval into single call
func (s *SecurityService) ValidateAndApprove(ctx context.Context, cmd *Command, workDir string) (bool, error)

// NeedsApproval checks if command requires approval
func (s *SecurityService) NeedsApproval(cmd *Command) bool

// RequestApproval delegates to ApprovalService
func (s *SecurityService) RequestApproval(ctx context.Context, operation Operation) (bool, error)
```

#### 2. ApprovalService (`internal/security/approval.go`)

Handles the complete approval workflow with event emission and policy persistence:

```go
type ApprovalService struct {
    handler           ApprovalHandler
    emitter           *events.EventEmitter
    validator         *Validator
    store             PolicyStore
    sessionDefaultTTL time.Duration
    globalDefaultTTL  time.Duration
}

// RequestApproval handles full approval flow with policy short-circuit
func (s *ApprovalService) RequestApproval(ctx context.Context, operation Operation) (reqID string, approved bool, err error)
```

#### 3. Tool-Level Approval (`internal/tools/approval.go`)

Tools implement the `ToolWithApproval` interface for self-assessment:

```go
type ToolWithApproval interface {
    Tool
    CheckApproval(params ToolParameters) ApprovalNeeds
}

type ApprovalNeeds struct {
    Required bool
    Risk     RiskLevel
    Reason   string
}
```

#### 4. ToolRuntime Integration (`internal/agent/tool_runtime.go`)

Unified tool execution with approval:

```go
if toolWithApproval, ok := tool.(tools.ToolWithApproval); ok {
    needs := toolWithApproval.CheckApproval(args)
    if needs.Required {
        operation := security.NewOperationWithToolCallID(cmd, needs.Reason, t.workDir, call.ID)
        _, approved, err := t.approvalService.RequestApproval(ctx, operation)
        // ...
    }
}
```

### Comparison with Proposed Solution

| Proposed Interface | Current Implementation |
|-------------------|----------------------|
| `ShouldApprove(op) (ApprovalDecision, error)` | `SecurityService.ValidateAndApprove()` - validates and returns approval decision |
| `RequestApproval(op) (ApprovalResponse, error)` | `ApprovalService.RequestApproval()` - full approval flow with events |

The current implementation actually provides **more functionality** than the proposed interface:
- Policy persistence with TTL
- Event emission for UI/protocol updates
- Command modification support
- Session and global scope management

### Why No Further Changes Needed

1. **Single Source of Truth**: `ApprovalService` is the canonical approval handler
2. **Clear Delegation**: `SecurityService` coordinates validation + approval
3. **Tool Self-Assessment**: `ToolWithApproval` interface allows tools to specify needs
4. **Unified Execution**: `ToolRuntime` and `Agent` use centralized services
5. **Handler Abstraction**: `ApprovalHandler` type allows different implementations (TUI, ACP, auto-approve)

### Code Paths Verified

- `internal/agent/agent.go:561` - Uses `security.NeedsApproval()`
- `internal/agent/tool_runtime.go:87,105` - Uses `ApprovalService.RequestApproval()`
- `internal/protocol/acp/agent.go:1365` - Uses `ApprovalService.RequestApproval()`
- All paths converge to centralized services

## Conclusion

The roadmap item 2.5 "Scattered Approval Logic" has been **addressed through incremental refactoring**. The approval system now follows a clean, centralized architecture:

```
┌─────────────────────────────────────────────────────────────────┐
│                    Consumer Code                                 │
│  (Agent, ToolRuntime, ACP Protocol)                             │
└────────────────────────────┬────────────────────────────────────┘
                             │
                             ▼
┌─────────────────────────────────────────────────────────────────┐
│                   SecurityService                                │
│  - ValidateCommand()                                            │
│  - NeedsApproval()                                              │
│  - ValidateAndApprove() ← combines validation + approval        │
│  - RequestApproval()                                            │
└────────────────────────────┬────────────────────────────────────┘
                             │
              ┌──────────────┼──────────────┐
              │              │              │
              ▼              ▼              ▼
┌─────────────────┐ ┌─────────────────┐ ┌─────────────────┐
│    Validator    │ │ ApprovalService │ │  PolicyStore    │
│  - Classify()   │ │ - RequestApproval│ │  - Get()        │
│  - NeedsApproval│ │ - Events        │ │  - Save()       │
└─────────────────┘ └─────────────────┘ └─────────────────┘
```

## Recommendation

Mark roadmap item 2.5 as **COMPLETED** with resolution note documenting that centralization was achieved through:
- FRD-20251115022244-executor-security-service.md
- FRD-20251115022813-shell-tool-security-service.md  
- FRD-20251115023502-remove-should-approve.md
- FRD-20251028000006-universal-tool-approval.md

No additional code changes required.
