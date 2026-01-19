// Package errors provides structured error types for the Spin agent.
//
// This package defines error codes, structured error types, and utilities
// for creating and handling errors in a type-safe, consistent manner across
// the codebase.
package errors

import (
	"fmt"
)

// ErrorCode represents different error categories for type-safe error handling.
type ErrorCode string

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

// Error codes for different failure categories.
const (
	CodeValidation      ErrorCode = "validation"       // Invalid input or configuration
	CodeTimeout         ErrorCode = "timeout"          // Operation exceeded time limit
	CodeNotFound        ErrorCode = "not_found"        // Resource not found
	CodePermission      ErrorCode = "permission"       // Permission denied or unauthorized
	CodeLLM             ErrorCode = "llm"              // LLM provider error
	CodeToolExecution   ErrorCode = "tool_execution"   // Tool execution failed
	CodeApprovalDenied  ErrorCode = "approval_denied"  // User denied approval
	CodeCycle           ErrorCode = "cycle"            // Cycle or infinite loop detected
	CodeInternal        ErrorCode = "internal"         // Internal error
	CodeNetwork         ErrorCode = "network"          // Network or connection error
	CodeIO              ErrorCode = "io"               // File or I/O error
	CodePatch           ErrorCode = "patch"            // Patch application error
	CodeGit             ErrorCode = "git"              // Git operation error
	CodeContextMismatch ErrorCode = "context_mismatch" // Patch context not found
)

// Error represents a structured error with context and error code.
//
// Error implements the error interface and supports error wrapping via Unwrap().
// It includes operation context, error categorization, and optional underlying errors.
type Error struct {
	Code    ErrorCode // Error category code
	Op      string    // Operation: "Agent.Execute", "Tool.Execute", etc.
	Err     error     // Underlying error (optional)
	Message string    // Human-readable message
}

// Error returns the formatted error message.
//
// Format: "Op: Message: Err" if Err is present, otherwise "Op: Message".
func (e *Error) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %s: %v", e.Op, e.Message, e.Err)
	}
	return fmt.Sprintf("%s: %s", e.Op, e.Message)
}

// Unwrap returns the underlying error for error chain traversal.
func (e *Error) Unwrap() error {
	return e.Err
}

// GetCode returns the error category code.
// This method implements the SpinError interface.
func (e *Error) GetCode() ErrorCode {
	return e.Code
}

// Operation returns the operation that failed.
// This method implements the SpinError interface.
func (e *Error) Operation() string {
	return e.Op
}

// New creates a new structured error.
//
// Example:
//
//	err := errors.New(
//	    errors.CodeLLM,
//	    "Agent.Execute",
//	    "llm completion failed",
//	    underlyingErr,
//	)
func New(code ErrorCode, op string, message string, err error) *Error {
	return &Error{
		Code:    code,
		Op:      op,
		Err:     err,
		Message: message,
	}
}

// Newf creates a new structured error with formatted message.
//
// Example:
//
//	err := errors.Newf(
//	    errors.CodeValidation,
//	    "Agent.Execute",
//	    underlyingErr,
//	    "invalid max_turns: %d",
//	    maxTurns,
//	)
func Newf(code ErrorCode, op string, err error, format string, args ...interface{}) *Error {
	return &Error{
		Code:    code,
		Op:      op,
		Err:     err,
		Message: fmt.Sprintf(format, args...),
	}
}
