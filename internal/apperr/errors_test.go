package apperr

import (
	"errors"
	"fmt"
	"testing"
)

var (
	errConnectionTimeout = errors.New("connection timeout")
	errUnderlyingError = errors.New("underlying error")
	errUnderlying = errors.New("underlying")
	errUnderlying2 = errors.New("underlying")
	errSentinelError = errors.New("sentinel error")
	errOtherError = errors.New("other error")
	errRootCause = errors.New("root cause")
	errConnectionRefused = errors.New("connection refused")
	errDiskFull = errors.New("disk full")
	errUnderlyingCause = errors.New("underlying cause")
)

func TestError_Error(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  *Error
		want string
	}{
		{
			name: "error with underlying error",
			err: &Error{
				Code:    CodeLLM,
				Op:      "Agent.Execute",
				Message: "llm completion failed",
				Err:     errConnectionTimeout,
			},
			want: "Agent.Execute: llm completion failed: connection timeout",
		},
		{
			name: "error without underlying error",
			err: &Error{
				Code:    CodeValidation,
				Op:      "Agent.Execute",
				Message: "invalid input",
			},
			want: "Agent.Execute: invalid input",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := tt.err.Error(); got != tt.want {
				t.Errorf("Error.Error() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestError_Unwrap(t *testing.T) {
	t.Parallel()

	underlying := errUnderlyingError
	err := &Error{
		Code:    CodeInternal,
		Op:      "Test.Operation",
		Message: "test error",
		Err:     underlying,
	}

	got := err.Unwrap()
	if !errors.Is(got, underlying) {
		t.Errorf("Error.Unwrap() = %v, want %v", got, underlying)
	}
}

func TestNew(t *testing.T) {
	t.Parallel()

	underlying := errUnderlying
	err := New(CodeTimeout, "Test.Op", "timeout occurred", underlying)

	if err.Code != CodeTimeout {
		t.Errorf("New().Code = %v, want %v", err.Code, CodeTimeout)
	}

	if err.Op != "Test.Op" {
		t.Errorf("New().Op = %v, want %v", err.Op, "Test.Op")
	}

	if err.Message != "timeout occurred" {
		t.Errorf("New().Message = %v, want %v", err.Message, "timeout occurred")
	}

	if !errors.Is(err.Err, underlying) {
		t.Errorf("New().Err = %v, want %v", err.Err, underlying)
	}
}

func TestNewf(t *testing.T) {
	t.Parallel()

	underlying := errUnderlying2
	err := Newf(CodeValidation, "Test.Op", underlying, "invalid value: %d", 42)

	if err.Code != CodeValidation {
		t.Errorf("Newf().Code = %v, want %v", err.Code, CodeValidation)
	}

	if err.Message != "invalid value: 42" {
		t.Errorf("Newf().Message = %v, want %v", err.Message, "invalid value: 42")
	}

	if !errors.Is(err.Err, underlying) {
		t.Errorf("Newf().Err = %v, want %v", err.Err, underlying)
	}
}

func TestIs(t *testing.T) {
	t.Parallel()

	sentinel := errSentinelError
	wrapped := New(CodeInternal, "Test.Op", "wrapped error", sentinel)

	if !errors.Is(wrapped, sentinel) {
		t.Error("errors.Is() should find sentinel error in chain")
	}

	other := errOtherError
	if errors.Is(wrapped, other) {
		t.Error("errors.Is() should not match different error")
	}
}

func TestAs(t *testing.T) {
	t.Parallel()

	inner := New(CodeTimeout, "Inner.Op", "inner timeout", nil)
	outer := New(CodeInternal, "Outer.Op", "outer error", inner)

	var e *Error
	if !errors.As(outer, &e) {
		t.Fatal("errors.As() should find *Error in chain")
	}

	if e.Code != CodeInternal {
		t.Errorf("errors.As() found error with Code = %v, want %v", e.Code, CodeInternal)
	}
}

func TestErrorCodes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		code ErrorCode
		want string
	}{
		{CodeValidation, "validation"},
		{CodeTimeout, "timeout"},
		{CodeNotFound, "not_found"},
		{CodePermission, "permission"},
		{CodeLLM, "llm"},
		{CodeToolExecution, "tool_execution"},
		{CodeApprovalDenied, "approval_denied"},
		{CodeCycle, "cycle"},
		{CodeInternal, "internal"},
		{CodeNetwork, "network"},
		{CodeIO, "io"},
		// New error codes for patch/git operations.
		{CodePatch, "patch"},
		{CodeGit, "git"},
		{CodeContextMismatch, "context_mismatch"},
	}

	for _, tt := range tests {
		t.Run(string(tt.code), func(t *testing.T) {
			t.Parallel()

			if string(tt.code) != tt.want {
				t.Errorf("ErrorCode = %q, want %q", tt.code, tt.want)
			}
		})
	}
}

func TestErrorChaining(t *testing.T) {
	t.Parallel()

	// Create error chain: root -> middle -> top.
	root := errRootCause
	middle := New(CodeNetwork, "Middle.Op", "network error", root)
	top := New(CodeInternal, "Top.Op", "operation failed", middle)

	// Test that we can find the root error.
	if !errors.Is(top, root) {
		t.Error("errors.Is() should find root error through chain")
	}

	// Test that we can extract structured error info.
	var structErr *Error
	if !errors.As(top, &structErr) {
		t.Fatal("errors.As() should find *Error in chain")
	}

	// Should get the top-level error.
	if structErr.Code != CodeInternal {
		t.Errorf("errors.As() extracted Code = %v, want %v", structErr.Code, CodeInternal)
	}

	// Unwrap to get middle error.
	unwrapped := structErr.Unwrap()
	if unwrapped == nil {
		t.Fatal("Unwrap() should return middle error")
	}

	var middleErr *Error
	if !errors.As(unwrapped, &middleErr) {
		t.Fatal("Middle error should be *Error")
	}

	if middleErr.Code != CodeNetwork {
		t.Errorf("Middle error Code = %v, want %v", middleErr.Code, CodeNetwork)
	}
}

func ExampleNew() {
	// Create a structured error wrapping an underlying error.
	underlying := errConnectionRefused
	err := New(CodeNetwork, "Client.Connect", "failed to connect to server", underlying)

	fmt.Println(err)
	// Output: Client.Connect: failed to connect to server: connection refused
}

func ExampleNewf() {
	// Create a structured error with formatted message.
	err := Newf(CodeValidation, "Config.Validate", nil, "invalid timeout: %d seconds", -5)

	fmt.Println(err)
	// Output: Config.Validate: invalid timeout: -5 seconds
}

func ExampleAs() {
	// Create an error chain.
	underlying := errDiskFull
	err := New(CodeIO, "File.Write", "failed to write file", underlying)

	// Extract structured error information.
	var structErr *Error
	if errors.As(err, &structErr) {
		fmt.Printf("Code: %s, Op: %s\n", structErr.Code, structErr.Op)
	}
	// Output: Code: io, Op: File.Write
}

// TestSpinErrorInterface verifies that Error implements SpinError interface.
func TestSpinErrorInterface(t *testing.T) {
	t.Parallel()

	err := New(CodeLLM, "Agent.Execute", "llm failed", nil)

	// Verify Error implements SpinError.
	var spinErr SpinError = err

	_ = spinErr // Compile-time check.

	// Test GetCode() method.
	if got := err.GetCode(); got != CodeLLM {
		t.Errorf("GetCode() = %v, want %v", got, CodeLLM)
	}
}

// TestSpinError_Operation verifies Operation() returns correct operation.
func TestSpinError_Operation(t *testing.T) {
	t.Parallel()

	const expectedOp = "Tool.ReadFile"

	err := New(CodeIO, expectedOp, "read failed", nil)

	if got := err.Operation(); got != expectedOp {
		t.Errorf("Operation() = %v, want %v", got, expectedOp)
	}
}

// TestSpinError_UnwrapMethod verifies Unwrap() returns underlying error.
func TestSpinError_UnwrapMethod(t *testing.T) {
	t.Parallel()

	underlying := errUnderlyingCause
	err := New(CodeInternal, "Test.Op", "test error", underlying)

	got := err.Unwrap()
	if !errors.Is(got, underlying) {
		t.Errorf("Unwrap() = %v, want %v", got, underlying)
	}
}

// TestSpinError_NilUnderlying verifies Unwrap() returns nil when no underlying error.
func TestSpinError_NilUnderlying(t *testing.T) {
	t.Parallel()

	err := New(CodeValidation, "Test.Op", "validation failed", nil)

	got := err.Unwrap()
	if got != nil {
		t.Errorf("Unwrap() = %v, want nil", got)
	}
}
