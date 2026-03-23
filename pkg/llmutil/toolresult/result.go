// Package toolresult provides LLM tool result types and schema definitions
// for use across LLM tool protocol implementations.
package toolresult

import "fmt"

// Result represents the result of executing a tool.
type Result struct {
	// ID is the unique identifier for this tool call.
	ID string `json:"id,omitempty"`

	// Success indicates whether the tool execution succeeded.
	Success bool `json:"success"`

	// Output contains the tool's output message for the LLM.
	Output string `json:"output"`

	// Error contains an error message if the tool failed (for JSON serialization).
	Error string `json:"error,omitempty"`

	// Err contains the actual error if the tool failed.
	// This field is not serialized to JSON.
	Err error `json:"-"`

	// ExitCode contains the exit code for command-based tools.
	ExitCode int `json:"exit_code,omitempty"`

	// Metadata contains additional tool-specific data.
	Metadata map[string]any `json:"metadata,omitempty"`
}

// NewResult creates a successful tool result with the given output.
func NewResult(output string) Result {
	return Result{
		Success: true,
		Output:  output,
	}
}

// NewError creates a failed tool result from an error.
func NewError(err error) Result {
	return Result{
		Success: false,
		Err:     err,
		Error:   err.Error(),
	}
}

// ErrToResult converts an error into a failed Result with a formatted message.
func ErrToResult(format string, err error) (Result, error) {
	return Result{
		Success: false,
		Error:   fmt.Sprintf(format, err),
	}, nil
}

// NewErrorWithID creates a failed tool result with ID from an error.
func NewErrorWithID(id string, err error) Result {
	return Result{
		ID:      id,
		Success: false,
		Err:     err,
		Error:   err.Error(),
	}
}

// WithID returns a copy of the result with the given ID.
func (r Result) WithID(id string) Result {
	r.ID = id

	return r
}

// WithExitCode returns a copy of the result with the given exit code.
func (r Result) WithExitCode(code int) Result {
	r.ExitCode = code

	return r
}

// WithMetadata returns a copy of the result with the given metadata.
func (r Result) WithMetadata(metadata map[string]any) Result {
	r.Metadata = metadata

	return r
}

// GetErr returns the error if present, or nil.
func (r Result) GetErr() error {
	return r.Err
}

// String returns a string representation of the result.
func (r Result) String() string {
	if r.Success {
		return r.Output
	}

	if r.Err != nil {
		return r.Err.Error()
	}

	return r.Error
}
