package runtime

import (
	"context"
	"fmt"
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
	mapper          *tui.TUIMapper
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
		return nil, fmt.Errorf("workDir is required")
	}
	if cfg.Emitter == nil {
		return nil, fmt.Errorf("emitter is required")
	}

	logger := cfg.Logger
	if logger == nil {
		logger = slog.Default()
	}

	mapper := tui.NewTUIMapper(cfg.UI)

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
	// Read-only tools (shared, no runtime dependency)
	registry.Register(tools.NewReadFileTool())
	registry.Register(tools.NewWriteFileTool())
	registry.Register(tools.NewListDirectoryTool())

	// Builtin-specific shell command tool (uses local executor)
	var validatorAdapt tools.CommandValidator
	var shellCtxAdapt tools.ShellContext
	var execAdapt tools.CommandExecutor

	if r.validator != nil {
		validatorAdapt = NewValidatorAdapter(r.validator)
	}
	if r.shellService != nil {
		shellCtxAdapt = NewShellContextAdapter(r.shellService.GetContext())
	}
	if r.executor != nil {
		execAdapt = &ExecutorAdapter{executor: r.executor}
	}

	registry.Register(tools.NewShellCommandTool(validatorAdapt, shellCtxAdapt, execAdapt))

	r.logger.Debug("registered builtin tools", "count", 4)
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
func (r *BuiltinRuntime) Mapper() *tui.TUIMapper {
	return r.mapper
}

// builtinNotificationSender wraps TUIMapper to implement NotificationSender.
type builtinNotificationSender struct {
	mapper *tui.TUIMapper
}

func (s *builtinNotificationSender) SendToolCallStart(ctx context.Context, toolID, toolName string, params tools.ToolParameters) error {
	// TUIMapper handles EventToolCallStart events, not direct notifications
	// This is called from event emission, handled by the mapper's MapEvent
	return nil
}

func (s *builtinNotificationSender) SendToolCallUpdate(ctx context.Context, toolID string, status string, content interface{}) error {
	// TUIMapper handles EventToolCallProgress events
	return nil
}

func (s *builtinNotificationSender) SendToolCallComplete(ctx context.Context, toolID string, success bool, output string, err error) error {
	// TUIMapper handles EventToolCallComplete events
	return nil
}

func (s *builtinNotificationSender) SendMessageChunk(ctx context.Context, content string) error {
	// TUIMapper handles EventContentDelta events
	return nil
}

func (s *builtinNotificationSender) SendPlanUpdate(ctx context.Context, entries []PlanEntry) error {
	// Plan updates are handled via events
	return nil
}
