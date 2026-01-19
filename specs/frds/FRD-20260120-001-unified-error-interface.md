# FRD-20260120-001: Unified Error Interface

**Created:** 2026-01-20  
**Author:** Architecture Refactoring  
**Status:** Implemented  
**Priority:** P1 (High)  
**Related Roadmap Item:** 2.2 Inconsistent Error Types

## Problem Statement

The Spin codebase currently has three different error type patterns without a unified approach:

1. **`internal/errors/errors.go:32`** - Generic `Error` struct with `Code`, `Op`, `Err`, `Message` fields
2. **`internal/git/errors.go:27`** - `PatchError` struct with `Message`, `FilePath`, `Line`, `Reason` fields
3. **`internal/patchapply/applier.go:78`** - Another `Error` struct with `Op`, `Path`, `Line`, `Err`, `Context` fields

### Impact

- No consistent error handling across packages
- Difficult to wrap/unwrap errors uniformly
- Type assertions required for error inspection
- No standardized way to extract error codes, operations, or context
- Inconsistent error messages and formatting

## Solution

Create a unified `SpinError` interface in `internal/errors/` that all structured errors implement, enabling consistent error handling throughout the codebase.

## Design

### SpinError Interface

```go
// SpinError is the interface that all structured Spin errors implement.
// It extends the standard error interface with methods for error inspection.
type SpinError interface {
    error
    
    // GetCode returns the error category code.
    GetCode() ErrorCode
    
    // Operation returns the operation that failed (e.g., "Agent.Execute", "Tool.ReadFile").
    Operation() string
    
    // Unwrap returns the underlying error for error chain traversal.
    Unwrap() error
}
```

Note: The interface uses `GetCode()` instead of `Code()` to avoid name collision with the existing `Error.Code` field.

### Updated Error Struct

The existing `Error` struct in `internal/errors/errors.go` implements `SpinError`:

```go
type Error struct {
    Code    ErrorCode // Error category code
    Op      string    // Operation: "Agent.Execute", "Tool.Execute", etc.
    Err     error     // Underlying error (optional)
    Message string    // Human-readable message
}

// GetCode returns the error category code.
func (e *Error) GetCode() ErrorCode {
    return e.Code
}

// Operation returns the operation that failed.
func (e *Error) Operation() string {
    return e.Op
}

// Unwrap returns the underlying error for error chain traversal.
func (e *Error) Unwrap() error {
    return e.Err
}
```

### Extended Error Codes

New error codes to support all use cases:

```go
const (
    // Existing codes
    CodeValidation     ErrorCode = "validation"
    CodeTimeout        ErrorCode = "timeout"
    CodeNotFound       ErrorCode = "not_found"
    CodePermission     ErrorCode = "permission"
    CodeLLM            ErrorCode = "llm"
    CodeToolExecution  ErrorCode = "tool_execution"
    CodeApprovalDenied ErrorCode = "approval_denied"
    CodeCycle          ErrorCode = "cycle"
    CodeInternal       ErrorCode = "internal"
    CodeNetwork        ErrorCode = "network"
    CodeIO             ErrorCode = "io"
    
    // New codes for patch/git operations
    CodePatch          ErrorCode = "patch"          // Patch application error
    CodeGit            ErrorCode = "git"            // Git operation error
    CodeContextMismatch ErrorCode = "context_mismatch" // Patch context not found
)
```

### Migration Strategy

#### Phase 1: Add Interface (This FRD) - IMPLEMENTED

1. Add `SpinError` interface to `internal/errors/errors.go`
2. Update `Error` struct to implement `SpinError` with `GetCode()` and `Operation()` methods
3. Add new error codes (`CodePatch`, `CodeGit`, `CodeContextMismatch`)
4. Ensure backward compatibility - existing `Error` struct fields remain public

Note: Helper functions (`IsCode`, `GetCode`, `GetOperation`, `AsSpinError`) were not included in Phase 1 as they would be dead code until Phase 2/3 integration.

#### Phase 2: Migrate git.PatchError (Future)

Convert `git.PatchError` to use the unified interface:
- Option A: Make it implement `SpinError` interface
- Option B: Replace with `errors.Error` using `CodePatch`

#### Phase 3: Migrate patchapply.Error (Future)

Convert `patchapply.Error` to use the unified interface:
- Option A: Make it implement `SpinError` interface  
- Option B: Replace with `errors.Error` using `CodePatch`

## Acceptance Criteria

1. [x] `SpinError` interface defined with `GetCode()`, `Operation()`, `Unwrap()` methods
2. [x] `Error` struct implements `SpinError` interface
3. [x] New error codes `CodePatch`, `CodeGit`, `CodeContextMismatch` added
4. [x] All existing tests pass
5. [x] New tests achieve >= 90% coverage for new code
6. [x] `make test` passes with zero errors
7. [x] Backward compatibility maintained - existing code using `Error` struct continues to work

## Files Changed

- `internal/errors/errors.go` - Add interface, methods, new error codes
- `internal/errors/errors_test.go` - Add tests for new functionality

## Test Cases

### Unit Tests

1. **SpinError interface compliance**
   - `Error` struct implements `SpinError`
   - `GetCode()` returns correct code
   - `Operation()` returns correct operation
   - `Unwrap()` returns underlying error

2. **Error chain traversal**
   - Nested wrapped errors traversed correctly
   - `errors.Is` works with SpinError
   - `errors.As` works with SpinError

3. **New error codes**
   - `CodePatch` has correct string value
   - `CodeGit` has correct string value
   - `CodeContextMismatch` has correct string value

## Non-Goals

- Full migration of `git.PatchError` (future work)
- Full migration of `patchapply.Error` (future work)
- Breaking changes to existing API
- Removing existing error struct fields

## Risks

- **Low:** Adding interface methods to existing struct may cause issues if struct is embedded elsewhere
  - Mitigation: Search codebase for embeddings, ensure no conflicts
  
- **Low:** New error codes may overlap semantically with existing codes
  - Mitigation: Clear documentation of each code's purpose

## References

- ROADMAP.md Section 2.2: Inconsistent Error Types
- Go Error Handling Best Practices: https://go.dev/blog/go1.13-errors
- Effective Go Error Handling: https://go.dev/doc/effective_go#errors
