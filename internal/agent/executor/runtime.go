package executor

import (
	"context"

	"github.com/dmytrogajewski/spin/internal/safety"
	"github.com/dmytrogajewski/spin/internal/session"
	"github.com/dmytrogajewski/spin/internal/tools"
)

// CommandExecutor is an interface for executing commands.
// This breaks the import cycle between runtime and agent packages.
type CommandExecutor interface {
	Execute(ctx context.Context, cmd *safety.Command, opts any) (*CommandResult, error)
}

// CommandResult represents the result of command execution.
type CommandResult struct {
	Command     *safety.Command
	Stdout      string
	Stderr      string
	ExitCode    int
	Duration    any // Avoid importing time.Duration.
	StartedAt   any
	CompletedAt any
	Error       error
	Truncated   bool
}

// ToolRegistrar provides tool registration capability.
// Implementations register runtime-specific tools to the registry.
type ToolRegistrar interface {
	// RegisterTools registers runtime-specific tools to the registry.
	// Each runtime provides its own tool implementations (e.g., ACP shell command
	// uses terminal protocol, builtin shell command uses local executor).
	RegisterTools(registry *tools.Registry)
}

// NotificationProvider provides notification sending capability.
// Implementations convert internal events to runtime-specific notifications.
type NotificationProvider interface {
	// NotificationSender returns the notification sender for this runtime.
	// Converts internal events to runtime-specific notifications (ACP protocol,
	// TUI events, etc.).
	NotificationSender() NotificationSender
}

// ApprovalProvider provides approval handling capability.
// Implementations handle user approval requests in a runtime-specific way.
type ApprovalProvider interface {
	// ApprovalHandler returns the approval handler for this runtime.
	// Handles user approval requests in a runtime-specific way (ACP request_permission,
	// TUI dialogs, etc.).
	ApprovalHandler() safety.ApprovalHandler
}

// SessionProvider provides session management capability.
// Implementations provide session storage and identification.
type SessionProvider interface {
	// SessionStorage returns the session storage for persistence.
	// Both runtimes can use the same storage format.
	SessionStorage() session.Storage

	// SessionID returns the current session ID.
	// ACP runtime uses protocol session ID, builtin generates one per conversation.
	SessionID() string
}

// TerminalProvider provides terminal capability.
// Implementations provide terminal support checking and client access.
type TerminalProvider interface {
	// SupportsTerminals returns whether this runtime supports terminal protocol.
	// ACP runtime supports terminals via client, builtin uses local execution.
	SupportsTerminals() bool

	// TerminalClient returns the terminal client if supported, nil otherwise.
	// Only ACP runtime provides a terminal client.
	TerminalClient() TerminalClient
}

// Runtime combines all runtime capabilities.
// Each runtime (ACP, Builtin) provides its own implementation of tools,
// notifications, approvals, and terminal capabilities.
// Consumers should prefer using specific interfaces when possible.
type Runtime interface {
	ToolRegistrar
	NotificationProvider
	ApprovalProvider
	SessionProvider
	TerminalProvider
}
