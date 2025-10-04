package client

import (
	"errors"
	"fmt"
)

// Sentinel errors for common MCP client failures.
var (
	// ErrSpawnFailed indicates the MCP server process could not be started
	ErrSpawnFailed = errors.New("failed to spawn MCP server process")

	// ErrProtocolError indicates an invalid JSON-RPC message
	ErrProtocolError = errors.New("invalid JSON-RPC message")

	// ErrVersionMismatch indicates incompatible protocol versions
	ErrVersionMismatch = errors.New("incompatible protocol version")

	// ErrToolFailed indicates tool execution failed on the server
	ErrToolFailed = errors.New("tool execution failed")

	// ErrTimeout indicates a request exceeded the timeout duration
	ErrTimeout = errors.New("request timeout exceeded")

	// ErrConnectionClosed indicates the connection was closed
	ErrConnectionClosed = errors.New("connection closed")

	// ErrInvalidResponse indicates an unexpected response format
	ErrInvalidResponse = errors.New("invalid response format")
)

// Error wraps MCP client errors with additional context.
type Error struct {
	// Op is the operation that failed
	Op string

	// Err is the underlying error
	Err error
}

// Error implements the error interface.
func (e *Error) Error() string {
	return fmt.Sprintf("mcp client: %s: %v", e.Op, e.Err)
}

// Unwrap returns the underlying error.
func (e *Error) Unwrap() error {
	return e.Err
}

// Is implements error matching for Error types.
func (e *Error) Is(target error) bool {
	t, ok := target.(*Error)
	if !ok {
		return false
	}
	return e.Op == t.Op && errors.Is(e.Err, t.Err)
}
