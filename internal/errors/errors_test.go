package errors

import (
	"errors"
	"fmt"
	"testing"
)

func TestError_Error(t *testing.T) {
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
				Err:     errors.New("connection timeout"),
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
			if got := tt.err.Error(); got != tt.want {
				t.Errorf("Error.Error() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestError_Unwrap(t *testing.T) {
	underlying := errors.New("underlying error")
	err := &Error{
		Code:    CodeInternal,
		Op:      "Test.Operation",
		Message: "test error",
		Err:     underlying,
	}

	if got := err.Unwrap(); got != underlying {
		t.Errorf("Error.Unwrap() = %v, want %v", got, underlying)
	}
}

func TestNew(t *testing.T) {
	underlying := errors.New("underlying")
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
	if err.Err != underlying {
		t.Errorf("New().Err = %v, want %v", err.Err, underlying)
	}
}

func TestNewf(t *testing.T) {
	underlying := errors.New("underlying")
	err := Newf(CodeValidation, "Test.Op", underlying, "invalid value: %d", 42)

	if err.Code != CodeValidation {
		t.Errorf("Newf().Code = %v, want %v", err.Code, CodeValidation)
	}
	if err.Message != "invalid value: 42" {
		t.Errorf("Newf().Message = %v, want %v", err.Message, "invalid value: 42")
	}
	if err.Err != underlying {
		t.Errorf("Newf().Err = %v, want %v", err.Err, underlying)
	}
}

func TestIs(t *testing.T) {
	sentinel := errors.New("sentinel error")
	wrapped := New(CodeInternal, "Test.Op", "wrapped error", sentinel)

	if !errors.Is(wrapped, sentinel) {
		t.Error("errors.Is() should find sentinel error in chain")
	}

	other := errors.New("other error")
	if errors.Is(wrapped, other) {
		t.Error("errors.Is() should not match different error")
	}
}

func TestAs(t *testing.T) {
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
	}

	for _, tt := range tests {
		t.Run(string(tt.code), func(t *testing.T) {
			if string(tt.code) != tt.want {
				t.Errorf("ErrorCode = %q, want %q", tt.code, tt.want)
			}
		})
	}
}

func TestErrorChaining(t *testing.T) {
	// Create error chain: root -> middle -> top
	root := errors.New("root cause")
	middle := New(CodeNetwork, "Middle.Op", "network error", root)
	top := New(CodeInternal, "Top.Op", "operation failed", middle)

	// Test that we can find the root error
	if !errors.Is(top, root) {
		t.Error("errors.Is() should find root error through chain")
	}

	// Test that we can extract structured error info
	var structErr *Error
	if !errors.As(top, &structErr) {
		t.Fatal("errors.As() should find *Error in chain")
	}

	// Should get the top-level error
	if structErr.Code != CodeInternal {
		t.Errorf("errors.As() extracted Code = %v, want %v", structErr.Code, CodeInternal)
	}

	// Unwrap to get middle error
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
	// Create a structured error wrapping an underlying error
	underlying := fmt.Errorf("connection refused")
	err := New(CodeNetwork, "Client.Connect", "failed to connect to server", underlying)

	fmt.Println(err)
	// Output: Client.Connect: failed to connect to server: connection refused
}

func ExampleNewf() {
	// Create a structured error with formatted message
	err := Newf(CodeValidation, "Config.Validate", nil, "invalid timeout: %d seconds", -5)

	fmt.Println(err)
	// Output: Config.Validate: invalid timeout: -5 seconds
}

func ExampleAs() {
	// Create an error chain
	underlying := fmt.Errorf("disk full")
	err := New(CodeIO, "File.Write", "failed to write file", underlying)

	// Extract structured error information
	var structErr *Error
	if errors.As(err, &structErr) {
		fmt.Printf("Code: %s, Op: %s\n", structErr.Code, structErr.Op)
	}
	// Output: Code: io, Op: File.Write
}
