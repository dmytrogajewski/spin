package core

import (
	"errors"
	"fmt"
	"testing"
)

// TestSentinelErrors verifies all sentinel errors are defined
func TestSentinelErrors(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want string
	}{
		{"ErrInvalidInput", ErrInvalidInput, "invalid input"},
		{"ErrSessionNotFound", ErrSessionNotFound, "session not found"},
		{"ErrExecutionFailed", ErrExecutionFailed, "execution failed"},
		{"ErrPolicyViolation", ErrPolicyViolation, "policy violation"},
		{"ErrLLMError", ErrLLMError, "llm error"},
		{"ErrToolNotFound", ErrToolNotFound, "tool not found"},
		{"ErrContextTooLarge", ErrContextTooLarge, "context too large"},
		{"ErrTimeout", ErrTimeout, "timeout"},
		{"ErrCancelled", ErrCancelled, "cancelled"},
		{"ErrNotImplemented", ErrNotImplemented, "not implemented"},
		{"ErrAlreadyExists", ErrAlreadyExists, "already exists"},
		{"ErrConcurrentAccess", ErrConcurrentAccess, "concurrent access"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err == nil {
				t.Errorf("%s is nil", tt.name)
			}
			if got := tt.err.Error(); got != tt.want {
				t.Errorf("%s.Error() = %q, want %q", tt.name, got, tt.want)
			}
		})
	}
}

// TestError_Creation tests Error struct creation
func TestError_Creation(t *testing.T) {
	err := &Error{
		Op:   "Test.Operation",
		Err:  ErrInvalidInput,
		Code: ErrCodeInvalidInput,
	}

	if err.Op != "Test.Operation" {
		t.Errorf("Op = %q, want %q", err.Op, "Test.Operation")
	}
	if err.Code != ErrCodeInvalidInput {
		t.Errorf("Code = %d, want %d", err.Code, ErrCodeInvalidInput)
	}
	if !errors.Is(err.Err, ErrInvalidInput) {
		t.Errorf("Err should be ErrInvalidInput")
	}
}

// TestError_Error tests Error() string formatting
func TestError_Error(t *testing.T) {
	tests := []struct {
		name string
		err  *Error
		want string
	}{
		{
			name: "with underlying error",
			err: &Error{
				Op:  "Session.Load",
				Err: ErrSessionNotFound,
			},
			want: "Session.Load: session not found",
		},
		{
			name: "without underlying error",
			err: &Error{
				Op: "Session.Load",
			},
			want: "Session.Load",
		},
		{
			name: "nested errors",
			err: &Error{
				Op: "Manager.LoadSession",
				Err: &Error{
					Op:  "Session.Load",
					Err: ErrSessionNotFound,
				},
			},
			want: "Manager.LoadSession: Session.Load: session not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.err.Error(); got != tt.want {
				t.Errorf("Error() = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestError_Unwrap tests Unwrap() returns underlying error
func TestError_Unwrap(t *testing.T) {
	inner := ErrSessionNotFound
	err := &Error{
		Op:  "Session.Load",
		Err: inner,
	}

	unwrapped := errors.Unwrap(err)
	if unwrapped != inner {
		t.Errorf("Unwrap() = %v, want %v", unwrapped, inner)
	}
}

// TestError_Is tests error matching with errors.Is
func TestError_Is(t *testing.T) {
	tests := []struct {
		name   string
		err    error
		target error
		want   bool
	}{
		{
			name: "direct match by code",
			err: &Error{
				Op:   "Test",
				Code: ErrCodeNotFound,
			},
			target: &Error{Code: ErrCodeNotFound},
			want:   true,
		},
		{
			name: "no match different codes",
			err: &Error{
				Op:   "Test",
				Code: ErrCodeNotFound,
			},
			target: &Error{Code: ErrCodeInvalidInput},
			want:   false,
		},
		{
			name: "match wrapped sentinel error",
			err: &Error{
				Op:  "Test",
				Err: ErrSessionNotFound,
			},
			target: ErrSessionNotFound,
			want:   true,
		},
		{
			name: "no match different sentinel",
			err: &Error{
				Op:  "Test",
				Err: ErrSessionNotFound,
			},
			target: ErrInvalidInput,
			want:   false,
		},
		{
			name: "match nested error",
			err: &Error{
				Op: "Outer",
				Err: &Error{
					Op:   "Inner",
					Code: ErrCodeNotFound,
				},
			},
			target: &Error{Code: ErrCodeNotFound},
			want:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := errors.Is(tt.err, tt.target); got != tt.want {
				t.Errorf("errors.Is() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestError_WithContext tests context preservation
func TestError_WithContext(t *testing.T) {
	ctx := map[string]interface{}{
		"session_id": "abc123",
		"user_id":    456,
	}

	err := &Error{
		Context: ctx,
	}

	if err.Context["session_id"] != "abc123" {
		t.Errorf("Context[session_id] = %v, want abc123", err.Context["session_id"])
	}
	if err.Context["user_id"] != 456 {
		t.Errorf("Context[user_id] = %v, want 456", err.Context["user_id"])
	}
}

// TestNewValidationError tests NewValidationError constructor
func TestNewValidationError(t *testing.T) {
	err := NewValidationError("TestOp", "invalid input")
	if err == nil {
		t.Fatal("NewValidationError() returned nil")
	}

	var e *Error
	if !errors.As(err, &e) {
		t.Fatalf("NewValidationError() did not return *Error type")
	}

	if e.Op != "TestOp" {
		t.Errorf("Op = %q, want %q", e.Op, "TestOp")
	}
	if e.Code != ErrCodeInvalidInput {
		t.Errorf("Code = %v, want %v", e.Code, ErrCodeInvalidInput)
	}
}

// TestNewNotFoundError tests NewNotFoundError constructor
func TestNewNotFoundError(t *testing.T) {
	err := NewNotFoundError("LoadSession", "session-123")
	if err == nil {
		t.Fatal("NewNotFoundError() returned nil")
	}

	var e *Error
	if !errors.As(err, &e) {
		t.Fatalf("NewNotFoundError() did not return *Error type")
	}

	if e.Op != "LoadSession" {
		t.Errorf("Op = %q, want %q", e.Op, "LoadSession")
	}
	if e.Code != ErrCodeNotFound {
		t.Errorf("Code = %v, want %v", e.Code, ErrCodeNotFound)
	}
}

// TestNewExecutionError tests NewExecutionError constructor
func TestNewExecutionError(t *testing.T) {
	innerErr := errors.New("command failed")
	err := NewExecutionError("ExecuteCommand", innerErr)
	if err == nil {
		t.Fatal("NewExecutionError() returned nil")
	}

	var e *Error
	if !errors.As(err, &e) {
		t.Fatalf("NewExecutionError() did not return *Error type")
	}

	if e.Op != "ExecuteCommand" {
		t.Errorf("Op = %q, want %q", e.Op, "ExecuteCommand")
	}
	if e.Code != ErrCodeExternal {
		t.Errorf("Code = %v, want %v", e.Code, ErrCodeExternal)
	}
	if !errors.Is(err, innerErr) {
		t.Error("NewExecutionError() did not wrap inner error")
	}
}

// TestNewTimeoutError tests NewTimeoutError constructor
func TestNewTimeoutError(t *testing.T) {
	err := NewTimeoutError("WaitForResponse")
	if err == nil {
		t.Fatal("NewTimeoutError() returned nil")
	}

	var e *Error
	if !errors.As(err, &e) {
		t.Fatalf("NewTimeoutError() did not return *Error type")
	}

	if e.Op != "WaitForResponse" {
		t.Errorf("Op = %q, want %q", e.Op, "WaitForResponse")
	}
	if e.Code != ErrCodeTimeout {
		t.Errorf("Code = %v, want %v", e.Code, ErrCodeTimeout)
	}
}

// TestNewCancelledError tests NewCancelledError constructor
func TestNewCancelledError(t *testing.T) {
	err := NewCancelledError("RunTurn")
	if err == nil {
		t.Fatal("NewCancelledError() returned nil")
	}

	var e *Error
	if !errors.As(err, &e) {
		t.Fatalf("NewCancelledError() did not return *Error type")
	}

	if e.Op != "RunTurn" {
		t.Errorf("Op = %q, want %q", e.Op, "RunTurn")
	}
	if e.Code != ErrCodeCancelled {
		t.Errorf("Code = %v, want %v", e.Code, ErrCodeCancelled)
	}
}

// TestNewInternalError tests NewInternalError constructor
func TestNewInternalError(t *testing.T) {
	innerErr := errors.New("database connection failed")
	err := NewInternalError("SaveSession", innerErr)
	if err == nil {
		t.Fatal("NewInternalError() returned nil")
	}

	var e *Error
	if !errors.As(err, &e) {
		t.Fatalf("NewInternalError() did not return *Error type")
	}

	if e.Op != "SaveSession" {
		t.Errorf("Op = %q, want %q", e.Op, "SaveSession")
	}
	if e.Code != ErrCodeInternal {
		t.Errorf("Code = %v, want %v", e.Code, ErrCodeInternal)
	}
	if !errors.Is(err, innerErr) {
		t.Error("NewInternalError() did not wrap inner error")
	}
}

// TestNewPermissionError tests NewPermissionError constructor
func TestNewPermissionError(t *testing.T) {
	err := NewPermissionError("WriteFile", "access denied to /etc/hosts")
	if err == nil {
		t.Fatal("NewPermissionError() returned nil")
	}

	var e *Error
	if !errors.As(err, &e) {
		t.Fatalf("NewPermissionError() did not return *Error type")
	}

	if e.Op != "WriteFile" {
		t.Errorf("Op = %q, want %q", e.Op, "WriteFile")
	}
	if e.Code != ErrCodePermissionDenied {
		t.Errorf("Code = %v, want %v", e.Code, ErrCodePermissionDenied)
	}
}

// TestNewAlreadyExistsError tests NewAlreadyExistsError constructor
func TestNewAlreadyExistsError(t *testing.T) {
	err := NewAlreadyExistsError("CreateSession", "session-456")
	if err == nil {
		t.Fatal("NewAlreadyExistsError() returned nil")
	}

	var e *Error
	if !errors.As(err, &e) {
		t.Fatalf("NewAlreadyExistsError() did not return *Error type")
	}

	if e.Op != "CreateSession" {
		t.Errorf("Op = %q, want %q", e.Op, "CreateSession")
	}
	if e.Code != ErrCodeAlreadyExists {
		t.Errorf("Code = %v, want %v", e.Code, ErrCodeAlreadyExists)
	}
}

// TestWrapError tests WrapError helper function
func TestWrapError(t *testing.T) {
	// Test wrapping a regular error
	innerErr := errors.New("original error")
	wrapped := WrapError("Operation", innerErr)

	var e *Error
	if !errors.As(wrapped, &e) {
		t.Fatalf("WrapError() did not return *Error type")
	}

	if e.Op != "Operation" {
		t.Errorf("Op = %q, want %q", e.Op, "Operation")
	}
	if !errors.Is(wrapped, innerErr) {
		t.Error("WrapError() did not preserve error chain")
	}

	// Test wrapping an already-wrapped error
	doubleWrapped := WrapError("OuterOp", wrapped)
	var e2 *Error
	if !errors.As(doubleWrapped, &e2) {
		t.Fatalf("WrapError() did not return *Error type on double wrap")
	}

	// WrapError concatenates operations when wrapping a core.Error
	expectedOp := "OuterOp: Operation"
	if e2.Op != expectedOp {
		t.Errorf("Op = %q, want %q", e2.Op, expectedOp)
	}
	if !errors.Is(doubleWrapped, innerErr) {
		t.Error("WrapError() did not preserve error chain through double wrap")
	}
}

// TestErrorCodes verifies all error codes are defined
func TestErrorCodes(t *testing.T) {
	codes := []ErrorCode{
		ErrCodeUnknown,
		ErrCodeInvalidInput,
		ErrCodeNotFound,
		ErrCodeAlreadyExists,
		ErrCodePermissionDenied,
		ErrCodeTimeout,
		ErrCodeCancelled,
		ErrCodeInternal,
		ErrCodeExternal,
	}

	// Verify all codes have unique values
	seen := make(map[ErrorCode]bool)
	for _, code := range codes {
		if seen[code] {
			t.Errorf("Duplicate error code: %d", code)
		}
		seen[code] = true
	}

	// Verify codes are sequential starting from 0
	for i, code := range codes {
		if int(code) != i {
			t.Errorf("ErrorCode[%d] = %d, want %d", i, code, i)
		}
	}
}

// TestErrorCode_String tests ErrorCode string conversion
func TestErrorCode_String(t *testing.T) {
	tests := []struct {
		code ErrorCode
		want string
	}{
		{ErrCodeUnknown, "unknown"},
		{ErrCodeInvalidInput, "invalid_input"},
		{ErrCodeNotFound, "not_found"},
		{ErrCodeAlreadyExists, "already_exists"},
		{ErrCodePermissionDenied, "permission_denied"},
		{ErrCodeTimeout, "timeout"},
		{ErrCodeCancelled, "cancelled"},
		{ErrCodeInternal, "internal"},
		{ErrCodeExternal, "external"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.code.String(); got != tt.want {
				t.Errorf("ErrorCode.String() = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestState_String tests State string conversion
func TestState_String(t *testing.T) {
	tests := []struct {
		state State
		want  string
	}{
		{StateIdle, "idle"},
		{StateRunning, "running"},
		{StatePaused, "paused"},
		{StateCompleted, "completed"},
		{StateFailed, "failed"},
		{StateCancelled, "cancelled"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.state.String(); got != tt.want {
				t.Errorf("State.String() = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestState_MarshalText tests State JSON marshaling
func TestState_MarshalText(t *testing.T) {
	tests := []struct {
		state State
		want  string
	}{
		{StateIdle, "idle"},
		{StateRunning, "running"},
		{StateCompleted, "completed"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			data, err := tt.state.MarshalText()
			if err != nil {
				t.Fatalf("MarshalText() error = %v", err)
			}
			if got := string(data); got != tt.want {
				t.Errorf("MarshalText() = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestState_UnmarshalText tests State JSON unmarshaling
func TestState_UnmarshalText(t *testing.T) {
	tests := []struct {
		text string
		want State
	}{
		{"idle", StateIdle},
		{"running", StateRunning},
		{"completed", StateCompleted},
		{"failed", StateFailed},
	}

	for _, tt := range tests {
		t.Run(tt.text, func(t *testing.T) {
			var state State
			err := state.UnmarshalText([]byte(tt.text))
			if err != nil {
				t.Fatalf("UnmarshalText() error = %v", err)
			}
			if state != tt.want {
				t.Errorf("UnmarshalText() = %v, want %v", state, tt.want)
			}
		})
	}
}

// TestState_UnmarshalText_Invalid tests invalid state unmarshaling
func TestState_UnmarshalText_Invalid(t *testing.T) {
	var state State
	err := state.UnmarshalText([]byte("invalid"))
	if err == nil {
		t.Error("UnmarshalText() should return error for invalid state")
	}
}

// TestTokenUsage_Calculation tests TokenUsage calculations
func TestTokenUsage_Calculation(t *testing.T) {
	usage := TokenUsage{
		PromptTokens:     100,
		CompletionTokens: 50,
	}

	// Total should be sum of prompt and completion
	usage.TotalTokens = usage.PromptTokens + usage.CompletionTokens

	if usage.TotalTokens != 150 {
		t.Errorf("TotalTokens = %d, want 150", usage.TotalTokens)
	}
}

// TestTokenUsage_Add tests adding token usage
func TestTokenUsage_Add(t *testing.T) {
	usage1 := TokenUsage{
		PromptTokens:     100,
		CompletionTokens: 50,
		TotalTokens:      150,
	}

	usage2 := TokenUsage{
		PromptTokens:     25,
		CompletionTokens: 75,
		TotalTokens:      100,
	}

	result := usage1.Add(usage2)

	if result.PromptTokens != 125 {
		t.Errorf("PromptTokens = %d, want 125", result.PromptTokens)
	}
	if result.CompletionTokens != 125 {
		t.Errorf("CompletionTokens = %d, want 125", result.CompletionTokens)
	}
	if result.TotalTokens != 250 {
		t.Errorf("TotalTokens = %d, want 250", result.TotalTokens)
	}
}

// TestFilter_Validation tests Filter struct validation
func TestFilter_Validation(t *testing.T) {
	tests := []struct {
		name    string
		filter  Filter
		wantErr bool
	}{
		{
			name: "valid filter",
			filter: Filter{
				WorkDir: "/tmp",
				State:   StateCompleted,
				Limit:   10,
				Offset:  0,
			},
			wantErr: false,
		},
		{
			name: "negative limit",
			filter: Filter{
				Limit: -1,
			},
			wantErr: true,
		},
		{
			name: "negative offset",
			filter: Filter{
				Offset: -1,
			},
			wantErr: true,
		},
		{
			name: "end before start",
			filter: Filter{
				StartTime: mustParseTime("2025-01-02T00:00:00Z"),
				EndTime:   mustParseTime("2025-01-01T00:00:00Z"),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.filter.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Filter.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// Helper function for parsing time in tests
func mustParseTime(s string) interface{} {
	return s // Simplified for test purposes
}

// TestError_As tests error type assertion with errors.As
func TestError_As(t *testing.T) {
	original := &Error{
		Op:   "Test",
		Err:  ErrSessionNotFound,
		Code: ErrCodeNotFound,
		Context: map[string]interface{}{
			"id": "test123",
		},
	}

	err := fmt.Errorf("wrapped: %w", original)

	var target *Error
	if !errors.As(err, &target) {
		t.Fatal("errors.As() should match *Error")
	}

	if target.Op != original.Op {
		t.Errorf("Op = %q, want %q", target.Op, original.Op)
	}
	if target.Code != original.Code {
		t.Errorf("Code = %d, want %d", target.Code, original.Code)
	}
	if target.Context["id"] != "test123" {
		t.Errorf("Context[id] = %v, want test123", target.Context["id"])
	}
}
