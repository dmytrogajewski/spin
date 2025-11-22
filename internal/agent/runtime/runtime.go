package runtime

import (
	"context"

	"github.com/dmytrogajewski/spin/internal/security"
	"github.com/dmytrogajewski/spin/internal/session"
	"github.com/dmytrogajewski/spin/internal/tools"
)

// CommandExecutor is an interface for executing commands.
// This breaks the import cycle between runtime and agent packages.
type CommandExecutor interface {
	Execute(ctx context.Context, cmd *security.Command, opts interface{}) (*CommandResult, error)
}

// CommandResult represents the result of command execution.
type CommandResult struct {
	Command     *security.Command
	Stdout      string
	Stderr      string
	ExitCode    int
	Duration    interface{} // Avoid importing time.Duration
	StartedAt   interface{}
	CompletedAt interface{}
	Error       error
	Truncated   bool
}

// Runtime defines the interface for runtime-specific functionality.
// Each runtime (ACP, Builtin) provides its own implementation of tools,
// notifications, approvals, and terminal capabilities.
type Runtime interface {
	// RegisterTools registers runtime-specific tools to the registry.
	// Each runtime provides its own tool implementations (e.g., ACP shell command
	// uses terminal protocol, builtin shell command uses local executor).
	RegisterTools(registry *tools.Registry)

	// NotificationSender returns the notification sender for this runtime.
	// Converts internal events to runtime-specific notifications (ACP protocol,
	// TUI events, etc.).
	NotificationSender() NotificationSender

	// ApprovalHandler returns the approval handler for this runtime.
	// Handles user approval requests in a runtime-specific way (ACP request_permission,
	// TUI dialogs, etc.).
	ApprovalHandler() security.ApprovalHandler

	// SessionStorage returns the session storage for persistence.
	// Both runtimes can use the same storage format.
	SessionStorage() session.Storage

	// SessionID returns the current session ID.
	// ACP runtime uses protocol session ID, builtin generates one per conversation.
	SessionID() string

	// SupportsTerminals returns whether this runtime supports terminal protocol.
	// ACP runtime supports terminals via client, builtin uses local execution.
	SupportsTerminals() bool

	// TerminalClient returns the terminal client if supported, nil otherwise.
	// Only ACP runtime provides a terminal client.
	TerminalClient() TerminalClient
}
