package core

import (
	"errors"
	"fmt"
)

// Sentinel errors - common error conditions
var (
	// ErrInvalidInput indicates invalid input validation
	ErrInvalidInput = errors.New("invalid input")

	// ErrSessionNotFound indicates session lookup failure
	ErrSessionNotFound = errors.New("session not found")

	// ErrExecutionFailed indicates command execution failure
	ErrExecutionFailed = errors.New("execution failed")

	// ErrPolicyViolation indicates security policy violation
	ErrPolicyViolation = errors.New("policy violation")

	// ErrLLMError indicates LLM provider error
	ErrLLMError = errors.New("llm error")

	// ErrToolNotFound indicates tool registry lookup failure
	ErrToolNotFound = errors.New("tool not found")

	// ErrContextTooLarge indicates context exceeds size limits
	ErrContextTooLarge = errors.New("context too large")

	// ErrTimeout indicates operation timeout
	ErrTimeout = errors.New("timeout")

	// ErrCancelled indicates operation was cancelled
	ErrCancelled = errors.New("cancelled")

	// ErrNotImplemented indicates feature not yet implemented
	ErrNotImplemented = errors.New("not implemented")

	// ErrAlreadyExists indicates resource already exists
	ErrAlreadyExists = errors.New("already exists")

	// ErrConcurrentAccess indicates concurrent modification detected
	ErrConcurrentAccess = errors.New("concurrent access")
)

// ErrorCode represents machine-readable error codes
type ErrorCode int

const (
	// ErrCodeUnknown indicates an unknown error
	ErrCodeUnknown ErrorCode = iota
	// ErrCodeInvalidInput indicates invalid input
	ErrCodeInvalidInput
	// ErrCodeNotFound indicates resource not found
	ErrCodeNotFound
	// ErrCodeAlreadyExists indicates resource already exists
	ErrCodeAlreadyExists
	// ErrCodePermissionDenied indicates permission denied
	ErrCodePermissionDenied
	// ErrCodeTimeout indicates operation timeout
	ErrCodeTimeout
	// ErrCodeCancelled indicates operation cancelled
	ErrCodeCancelled
	// ErrCodeInternal indicates internal error
	ErrCodeInternal
	// ErrCodeExternal indicates external system error
	ErrCodeExternal
)

// String returns the string representation of ErrorCode
func (e ErrorCode) String() string {
	switch e {
	case ErrCodeUnknown:
		return "unknown"
	case ErrCodeInvalidInput:
		return "invalid_input"
	case ErrCodeNotFound:
		return "not_found"
	case ErrCodeAlreadyExists:
		return "already_exists"
	case ErrCodePermissionDenied:
		return "permission_denied"
	case ErrCodeTimeout:
		return "timeout"
	case ErrCodeCancelled:
		return "cancelled"
	case ErrCodeInternal:
		return "internal"
	case ErrCodeExternal:
		return "external"
	default:
		return "unknown"
	}
}

// Error represents a core package error with rich context
type Error struct {
	// Context contains additional context data
	Context map[string]interface{}

	// Err is the underlying error (for wrapping)
	Err error

	// Op is the operation that failed (e.g., "Manager.NewConversation")
	Op string

	// Code is the machine-readable error code
	Code ErrorCode
}

// Error implements the error interface
func (e *Error) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Op, e.Err)
	}
	return e.Op
}

// Unwrap returns the underlying error for errors.Is/As support
func (e *Error) Unwrap() error {
	return e.Err
}

// Is implements error matching for errors.Is
func (e *Error) Is(target error) bool {
	t, ok := target.(*Error)
	if !ok {
		return false
	}
	// Match by error code if both have codes
	if e.Code != ErrCodeUnknown && t.Code != ErrCodeUnknown {
		return e.Code == t.Code
	}
	return false
}

// NOTE: State type has been moved to state.go for better organization
// and to serve as the unified state management type across the entire package.

// Error Constructor Helpers
// These functions provide a consistent way to create errors with proper
// error codes and operation context throughout the codebase.

// NewValidationError creates an error for validation failures.
func NewValidationError(op, message string) error {
	return &Error{
		Op:   op,
		Code: ErrCodeInvalidInput,
		Err:  fmt.Errorf("%s", message),
	}
}

// Filter provides filtering options for listing conversations and sessions
type Filter struct {
	// StartTime filters by creation time (after this time)
	StartTime interface{} // Using interface{} for simplified test

	// EndTime filters by creation time (before this time)
	EndTime interface{} // Using interface{} for simplified test

	// WorkDir filters by working directory
	WorkDir string

	// Limit is the maximum number of results to return
	Limit int

	// Offset is the number of results to skip
	Offset int

	// State filters by execution state
	State State
}

// TokenUsage tracks token consumption for LLM API calls
type TokenUsage struct {
	// PromptTokens is the number of tokens in the prompt
	PromptTokens int

	// CompletionTokens is the number of tokens in the completion
	CompletionTokens int

	// TotalTokens is the total number of tokens used
	TotalTokens int
}
