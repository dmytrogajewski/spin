package exec

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

// ExitCode represents program exit codes.
type ExitCode int

const (
	// ExitSuccess indicates successful execution.
	ExitSuccess ExitCode = 0
	// ExitGeneralError indicates a general error.
	ExitGeneralError ExitCode = 1
	// ExitAuthError indicates authentication failure.
	ExitAuthError ExitCode = 2
	// ExitTaskFailed indicates task execution failed.
	ExitTaskFailed ExitCode = 3
	// ExitTimeout indicates execution timeout.
	ExitTimeout ExitCode = 4
	// ExitUserCancel indicates user cancellation (SIGINT).
	ExitUserCancel ExitCode = 5
)

// AuthError represents an authentication error.
type AuthError struct {
	msg string
}

func (e *AuthError) Error() string {
	return e.msg
}

// TaskFailedError represents a task execution failure.
type TaskFailedError struct {
	msg string
}

func (e *TaskFailedError) Error() string {
	return e.msg
}

// exitCodeFromError determines the appropriate exit code from an error.
func exitCodeFromError(err error) ExitCode {
	if err == nil {
		return ExitSuccess
	}

	// Check for context errors
	if errors.Is(err, context.DeadlineExceeded) {
		return ExitTimeout
	}
	if errors.Is(err, context.Canceled) {
		return ExitUserCancel
	}

	// Check for custom error types
	var authErr *AuthError
	if errors.As(err, &authErr) {
		return ExitAuthError
	}

	var taskErr *TaskFailedError
	if errors.As(err, &taskErr) {
		return ExitTaskFailed
	}

	// Default to general error
	return ExitGeneralError
}

// formatError formats an error with its full error chain.
func formatError(err error) string {
	var b strings.Builder
	b.WriteString("Error: ")
	b.WriteString(err.Error())

	// Unwrap error chain
	unwrapped := errors.Unwrap(err)
	if unwrapped != nil {
		b.WriteString("\nCaused by:")
		i := 0
		for unwrapped != nil {
			fmt.Fprintf(&b, "\n  %d: %v", i, unwrapped)
			unwrapped = errors.Unwrap(unwrapped)
			i++
		}
	}

	return b.String()
}
