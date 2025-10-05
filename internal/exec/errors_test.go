package exec

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
)

func TestExitCodeFromError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want ExitCode
	}{
		{
			name: "nil error",
			err:  nil,
			want: ExitSuccess,
		},
		{
			name: "context timeout",
			err:  context.DeadlineExceeded,
			want: ExitTimeout,
		},
		{
			name: "context cancelled",
			err:  context.Canceled,
			want: ExitUserCancel,
		},
		{
			name: "auth error",
			err:  &AuthError{msg: "invalid credentials"},
			want: ExitAuthError,
		},
		{
			name: "task failed error",
			err:  &TaskFailedError{msg: "tests failed"},
			want: ExitTaskFailed,
		},
		{
			name: "general error",
			err:  errors.New("something went wrong"),
			want: ExitGeneralError,
		},
		{
			name: "wrapped timeout",
			err:  fmt.Errorf("execution failed: %w", context.DeadlineExceeded),
			want: ExitTimeout,
		},
		{
			name: "wrapped auth error",
			err:  fmt.Errorf("login failed: %w", &AuthError{msg: "bad token"}),
			want: ExitAuthError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := exitCodeFromError(tt.err)
			if got != tt.want {
				t.Errorf("exitCodeFromError() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFormatError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		contains []string
	}{
		{
			name:     "simple error",
			err:      errors.New("simple error"),
			contains: []string{"Error:", "simple error"},
		},
		{
			name:     "wrapped error",
			err:      fmt.Errorf("outer: %w", errors.New("inner")),
			contains: []string{"Error:", "outer", "Caused by:", "inner"},
		},
		{
			name:     "double wrapped",
			err:      fmt.Errorf("level1: %w", fmt.Errorf("level2: %w", errors.New("root"))),
			contains: []string{"Error:", "level1", "Caused by:", "level2", "root"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatError(tt.err)
			for _, want := range tt.contains {
				if !strings.Contains(got, want) {
					t.Errorf("formatError() missing %q in output:\n%s", want, got)
				}
			}
		})
	}
}

func TestAuthError(t *testing.T) {
	err := &AuthError{msg: "authentication failed"}
	if err.Error() != "authentication failed" {
		t.Errorf("AuthError.Error() = %v, want %v", err.Error(), "authentication failed")
	}
}

func TestTaskFailedError(t *testing.T) {
	err := &TaskFailedError{msg: "task execution failed"}
	if err.Error() != "task execution failed" {
		t.Errorf("TaskFailedError.Error() = %v, want %v", err.Error(), "task execution failed")
	}
}

func TestErrorUnwrap(t *testing.T) {
	inner := errors.New("inner error")
	wrapped := fmt.Errorf("outer: %w", inner)

	exitCode := exitCodeFromError(wrapped)
	if exitCode != ExitGeneralError {
		t.Errorf("exitCodeFromError() for wrapped error = %v, want %v", exitCode, ExitGeneralError)
	}

	// Test with timeout
	timeoutWrapped := fmt.Errorf("task timed out: %w", context.DeadlineExceeded)
	exitCode = exitCodeFromError(timeoutWrapped)
	if exitCode != ExitTimeout {
		t.Errorf("exitCodeFromError() for wrapped timeout = %v, want %v", exitCode, ExitTimeout)
	}
}

func TestExitCodeConstants(t *testing.T) {
	// Verify exit code values
	if ExitSuccess != 0 {
		t.Errorf("ExitSuccess = %d, want 0", ExitSuccess)
	}
	if ExitGeneralError != 1 {
		t.Errorf("ExitGeneralError = %d, want 1", ExitGeneralError)
	}
	if ExitAuthError != 2 {
		t.Errorf("ExitAuthError = %d, want 2", ExitAuthError)
	}
	if ExitTaskFailed != 3 {
		t.Errorf("ExitTaskFailed = %d, want 3", ExitTaskFailed)
	}
	if ExitTimeout != 4 {
		t.Errorf("ExitTimeout = %d, want 4", ExitTimeout)
	}
	if ExitUserCancel != 5 {
		t.Errorf("ExitUserCancel = %d, want 5", ExitUserCancel)
	}
}
