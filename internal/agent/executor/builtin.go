package executor

import (
	"context"
	"log/slog"

	"github.com/dmytrogajewski/spin/internal/events"
	gitpkg "github.com/dmytrogajewski/spin/internal/git"
	"github.com/dmytrogajewski/spin/internal/security"
	"github.com/dmytrogajewski/spin/internal/session"
	shellpkg "github.com/dmytrogajewski/spin/internal/shell"
	"github.com/dmytrogajewski/spin/internal/tools"
	"github.com/dmytrogajewski/spin/internal/tui"
	"github.com/dmytrogajewski/spin/internal/ui/ports"
)

const builtinToolCount = 4

// BuiltinRuntime implements Runtime for builtin/TUI/EXEC modes.
// Uses local execution, TUI notifications, and TUI approval dialogs.
type BuiltinRuntime struct {
	workDir         string
	emitter         *events.EventEmitter
	storage         session.Storage
	sessionID       string
	executor        CommandExecutor
	validator       *security.Validator
	shellService    *shellpkg.Service
	gitService      *gitpkg.Service
	mapper          *tui.Mapper
	approvalHandler security.ApprovalHandler
	logger          *slog.Logger
}

// BuiltinRuntimeConfig configures the builtin runtime.
type BuiltinRuntimeConfig struct {
	WorkDir         string
	Emitter         *events.EventEmitter
	Storage         session.Storage
	SessionID       string
	Executor        CommandExecutor
	Validator       *security.Validator
	ShellService    *shellpkg.Service
	GitService      *gitpkg.Service
	UI              ports.UI
	ApprovalHandler security.ApprovalHandler
	Logger          *slog.Logger
}

// NewBuiltinRuntime creates a new builtin runtime.
func NewBuiltinRuntime(cfg BuiltinRuntimeConfig) (*BuiltinRuntime, error) {
	if cfg.WorkDir == "" {
		return nil, ErrWorkdirIsRequired
	}

	if cfg.Emitter == nil {
		return nil, ErrEmitterIsRequired
	}

	logger := cfg.Logger
	if logger == nil {
		logger = slog.Default()
	}

	mapper := tui.NewMapper(cfg.UI)

	return &BuiltinRuntime{
		workDir:         cfg.WorkDir,
		emitter:         cfg.Emitter,
		storage:         cfg.Storage,
		sessionID:       cfg.SessionID,
		executor:        cfg.Executor,
		validator:       cfg.Validator,
		shellService:    cfg.ShellService,
		gitService:      cfg.GitService,
		mapper:          mapper,
		approvalHandler: cfg.ApprovalHandler,
		logger:          logger,
	}, nil
}

// RegisterTools registers builtin-specific tools.
func (r *BuiltinRuntime) RegisterTools(registry *tools.Registry) {
	// Read-only tools (shared, no runtime dependency).
	_ = registry.Register(tools.NewReadFileTool())
	_ = registry.Register(tools.NewWriteFileTool())
	_ = registry.Register(tools.NewListDirectoryTool())

	// Builtin-specific shell command tool (uses local executor).
	var (
		validatorAdapt tools.CommandValidator
		shellCtxAdapt  tools.ShellContext
		execAdapt      tools.CommandExecutor
	)

	if r.validator != nil {
		validatorAdapt = NewValidatorAdapter(r.validator)
	}

	if r.shellService != nil {
		shellCtxAdapt = NewShellContextAdapter(r.shellService.GetContext())
	}

	if r.executor != nil {
		execAdapt = &Adapter{executor: r.executor}
	}

	_ = registry.Register(tools.NewShellCommandTool(validatorAdapt, shellCtxAdapt, execAdapt))

	r.logger.Debug("registered builtin tools", "count", builtinToolCount)
}

// NotificationSender returns the TUI notification sender (TUIMapper).
func (r *BuiltinRuntime) NotificationSender() NotificationSender {
	return &builtinNotificationSender{mapper: r.mapper}
}

// ApprovalHandler returns the TUI approval handler.
func (r *BuiltinRuntime) ApprovalHandler() security.ApprovalHandler {
	return r.approvalHandler
}

// SessionStorage returns the session storage.
func (r *BuiltinRuntime) SessionStorage() session.Storage {
	return r.storage
}

// SessionID returns the current session ID.
func (r *BuiltinRuntime) SessionID() string {
	return r.sessionID
}

// SupportsTerminals returns false (builtin uses local execution).
func (r *BuiltinRuntime) SupportsTerminals() bool {
	return false
}

// TerminalClient returns nil (builtin doesn't use terminal protocol).
func (r *BuiltinRuntime) TerminalClient() TerminalClient {
	return nil
}

// Mapper returns the TUI mapper for event processing.
// This is exposed so TUI code can use it directly for mapping events.
func (r *BuiltinRuntime) Mapper() *tui.Mapper {
	return r.mapper
}

// builtinNotificationSender wraps TUIMapper to implement NotificationSender.
type builtinNotificationSender struct {
	mapper *tui.Mapper
}

// SendToolCallStart implements the SendToolCallStart operation.
func (s *builtinNotificationSender) SendToolCallStart(_ context.Context, _, _ string, _ tools.ToolParameters) error {
	// TUIMapper handles EventToolCallStart events, not direct notifications
	// This is called from event emission, handled by the mapper's MapEvent.
	return nil
}

// SendToolCallUpdate implements the SendToolCallUpdate operation.
func (s *builtinNotificationSender) SendToolCallUpdate(_ context.Context, _, _ string, _ any) error {
	// TUIMapper handles EventToolCallProgress events.
	return nil
}

// SendToolCallComplete implements the SendToolCallComplete operation.
func (s *builtinNotificationSender) SendToolCallComplete(_ context.Context, _ string, _ bool, _ string, _ error) error {
	// TUIMapper handles EventToolCallComplete events.
	return nil
}

// SendMessageChunk implements the SendMessageChunk operation.
func (s *builtinNotificationSender) SendMessageChunk(_ context.Context, _ string) error {
	// TUIMapper handles EventContentDelta events.
	return nil
}

// SendPlanUpdate implements the SendPlanUpdate operation.
func (s *builtinNotificationSender) SendPlanUpdate(_ context.Context, _ []PlanEntry) error {
	// Plan updates are handled via events.
	return nil
}
