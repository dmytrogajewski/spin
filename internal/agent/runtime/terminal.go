package runtime

import (
	"context"
)

// TerminalClient provides terminal protocol operations.
// Used by ACP runtime to execute commands via client's terminal protocol.
type TerminalClient interface {
	// Create creates a new terminal and executes a command.
	// Returns the terminal ID.
	Create(ctx context.Context, cmd string, args []string, env []EnvVar, cwd string, limit int) (string, error)

	// WaitForExit waits for the terminal command to complete.
	// Returns the exit code and signal (if any).
	WaitForExit(ctx context.Context, terminalID string) (int, *string, error)

	// GetOutput retrieves the current terminal output.
	// Returns output, truncated flag, and exit status (if completed).
	GetOutput(ctx context.Context, terminalID string) (string, bool, *ExitStatus, error)

	// Release releases terminal resources.
	Release(ctx context.Context, terminalID string) error
}

// EnvVar represents an environment variable.
type EnvVar struct {
	Name  string
	Value string
}

// ExitStatus represents terminal exit status.
type ExitStatus struct {
	ExitCode *int
	Signal   *string
}

// Context key for session ID
type contextKey string

const sessionIDKey contextKey = "acp_session_id"

// ContextWithSessionID returns a context with the session ID.
func ContextWithSessionID(ctx context.Context, sessionID string) context.Context {
	return context.WithValue(ctx, sessionIDKey, sessionID)
}

// GetSessionIDFromContext returns the session ID from the context.
func GetSessionIDFromContext(ctx context.Context) string {
	if val, ok := ctx.Value(sessionIDKey).(string); ok {
		return val
	}
	return ""
}

// FilesystemClient provides filesystem operations via ACP protocol.
type FilesystemClient interface {
	// ReadTextFile reads a text file.
	// line and limit are optional (nil means not specified).
	ReadTextFile(ctx context.Context, path string, line *int, limit *int) (string, error)

	// WriteTextFile writes a text file.
	WriteTextFile(ctx context.Context, path, content string) error
}
