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

// State represents the execution state of conversations, turns, and tasks
type State int

const (
	// StateIdle indicates no active execution
	StateIdle State = iota
	// StateRunning indicates active execution
	StateRunning
	// StatePaused indicates execution is paused
	StatePaused
	// StateCompleted indicates successful completion
	StateCompleted
	// StateFailed indicates execution failed
	StateFailed
	// StateCancelled indicates execution was cancelled
	StateCancelled
)

// String returns the string representation of State
func (s State) String() string {
	switch s {
	case StateIdle:
		return "idle"
	case StateRunning:
		return "running"
	case StatePaused:
		return "paused"
	case StateCompleted:
		return "completed"
	case StateFailed:
		return "failed"
	case StateCancelled:
		return "cancelled"
	default:
		return "unknown"
	}
}

// MarshalText implements encoding.TextMarshaler for JSON encoding
func (s State) MarshalText() ([]byte, error) {
	return []byte(s.String()), nil
}

// UnmarshalText implements encoding.TextUnmarshaler for JSON decoding
func (s *State) UnmarshalText(text []byte) error {
	str := string(text)
	switch str {
	case "idle":
		*s = StateIdle
	case "running":
		*s = StateRunning
	case "paused":
		*s = StatePaused
	case "completed":
		*s = StateCompleted
	case "failed":
		*s = StateFailed
	case "cancelled":
		*s = StateCancelled
	default:
		return fmt.Errorf("invalid state: %s", str)
	}
	return nil
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

// Validate validates the filter parameters
func (f *Filter) Validate() error {
	if f.Limit < 0 {
		return &Error{
			Op:   "Filter.Validate",
			Err:  ErrInvalidInput,
			Code: ErrCodeInvalidInput,
			Context: map[string]interface{}{
				"field": "limit",
				"value": f.Limit,
			},
		}
	}

	if f.Offset < 0 {
		return &Error{
			Op:   "Filter.Validate",
			Err:  ErrInvalidInput,
			Code: ErrCodeInvalidInput,
			Context: map[string]interface{}{
				"field": "offset",
				"value": f.Offset,
			},
		}
	}

	// Validate time range if both are set
	if f.StartTime != nil && f.EndTime != nil {
		// For actual implementation, would compare time.Time values
		// For now, check if end is "before" start using string comparison
		startStr, startOK := f.StartTime.(string)
		endStr, endOK := f.EndTime.(string)
		if startOK && endOK && endStr < startStr {
			return &Error{
				Op:   "Filter.Validate",
				Err:  ErrInvalidInput,
				Code: ErrCodeInvalidInput,
				Context: map[string]interface{}{
					"field": "time_range",
					"error": "end_time before start_time",
				},
			}
		}
	}

	return nil
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

// Add returns a new TokenUsage with the sum of two usages
func (t TokenUsage) Add(other TokenUsage) TokenUsage {
	return TokenUsage{
		PromptTokens:     t.PromptTokens + other.PromptTokens,
		CompletionTokens: t.CompletionTokens + other.CompletionTokens,
		TotalTokens:      t.TotalTokens + other.TotalTokens,
	}
}
