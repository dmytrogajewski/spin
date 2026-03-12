// Package executor provides agent runtime execution.
package executor

import (
	"context"
	"errors"
	"log/slog"

	acpsdk "github.com/coder/acp-go-sdk"

	"github.com/dmytrogajewski/spin/internal/events"
	gitpkg "github.com/dmytrogajewski/spin/internal/git"
	"github.com/dmytrogajewski/spin/internal/security"
	"github.com/dmytrogajewski/spin/internal/session"
	shellpkg "github.com/dmytrogajewski/spin/internal/shell"
	"github.com/dmytrogajewski/spin/internal/tools"
)

var (
	// ErrWorkdirIsRequired is a sentinel error.
	ErrWorkdirIsRequired = errors.New("workDir is required")
	// ErrEmitterIsRequired is a sentinel error.
	ErrEmitterIsRequired = errors.New("emitter is required")
)

// ACPAgentInterface defines the interface for ACP agent functionality needed by runtime.
// This breaks the import cycle between runtime and protocol/acp packages.
type ACPAgentInterface interface {
	// GetClientCapabilities returns the client capabilities stored after Initialize.
	GetClientCapabilities() *acpsdk.ClientCapabilities
}

// ACPRuntime implements Runtime for ACP protocol mode.
// Uses ACP terminal protocol, ACP notifications, and ACP approval dialogs.
type ACPRuntime struct {
	workDir          string
	emitter          *events.EventEmitter
	storage          session.Storage
	sessionID        string
	acpAgent         ACPAgentInterface // Use interface to avoid import cycle.
	approvalHandler  security.ApprovalHandler
	clientCaps       *acpsdk.ClientCapabilities
	shellService     *shellpkg.Service
	gitService       *gitpkg.Service
	logger           *slog.Logger
	executor         tools.CommandExecutor
	validator        tools.CommandValidator
	terminalClient   TerminalClient
	filesystemClient FilesystemClient
}

// ACPConfig configures the ACP runtime.
type ACPConfig struct {
	WorkDir          string
	Emitter          *events.EventEmitter
	Storage          session.Storage
	SessionID        string
	ACPAgent         ACPAgentInterface
	ApprovalHandler  security.ApprovalHandler
	ClientCaps       *acpsdk.ClientCapabilities
	ShellService     *shellpkg.Service
	GitService       *gitpkg.Service
	Logger           *slog.Logger
	Executor         tools.CommandExecutor
	Validator        tools.CommandValidator
	TerminalClient   TerminalClient
	FilesystemClient FilesystemClient
}

// NewACP creates a new ACP runtime.
func NewACP(cfg ACPConfig) (*ACPRuntime, error) {
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

	return &ACPRuntime{
		workDir:          cfg.WorkDir,
		emitter:          cfg.Emitter,
		storage:          cfg.Storage,
		sessionID:        cfg.SessionID,
		acpAgent:         cfg.ACPAgent,
		approvalHandler:  cfg.ApprovalHandler,
		clientCaps:       cfg.ClientCaps,
		shellService:     cfg.ShellService,
		gitService:       cfg.GitService,
		logger:           logger,
		executor:         cfg.Executor,
		validator:        cfg.Validator,
		terminalClient:   cfg.TerminalClient,
		filesystemClient: cfg.FilesystemClient,
	}, nil
}

// RegisterTools registers ACP-specific tools.
// ACP provides native methods (fs/read_text_file, fs/write_text_file, terminal/create)
// so we do NOT register builtin tools. Instead, we register ACP-native tool wrappers
// that expose ACP protocol methods as tools to the LLM.
func (r *ACPRuntime) RegisterTools(registry *tools.Registry) {
	// Register ACP terminal tool (uses terminal/create protocol)
	// Note: Terminal client may not be set yet when this is called, but the tool
	// will check availability at execution time.
	terminalTool := NewACPTerminalTool(r)
	_ = registry.RegisterOrReplace(terminalTool)

	r.logger.Debug("registered ACP terminal tool")

	// Register ACP filesystem tools (uses fs/read_text_file and fs/write_text_file protocol)
	// Note: Filesystem client may not be set yet when this is called, but the tools
	// will check availability at execution time.
	if r.clientCaps != nil {
		if r.clientCaps.Fs.ReadTextFile {
			readTool := NewACPReadFileTool(r)
			_ = registry.RegisterOrReplace(readTool)

			r.logger.Debug("registered ACP read_file tool")
		}

		if r.clientCaps.Fs.WriteTextFile {
			writeTool := NewACPWriteFileTool(r)
			_ = registry.RegisterOrReplace(writeTool)

			r.logger.Debug("registered ACP write_file tool")
		}
	}

	r.logger.Debug("registered ACP tools", "count", len(registry.List()))
}

// NotificationSender returns the ACP notification sender.
func (r *ACPRuntime) NotificationSender() NotificationSender {
	return &acpNotificationSender{
		acpAgent:  r.acpAgent,
		sessionID: r.sessionID,
		emitter:   r.emitter,
	}
}

// ApprovalHandler returns the ACP approval handler.
func (r *ACPRuntime) ApprovalHandler() security.ApprovalHandler {
	// Return a wrapper that delegates to the current handler.
	return func(ctx context.Context, req security.ApprovalRequest) security.ApprovalResponse {
		handler := r.approvalHandler
		if handler == nil {
			// Fallback: auto-approve.
			return security.ApprovalResponse{Approved: true}
		}

		return handler(ctx, req)
	}
}

// SetExecutor updates the executor in the runtime.
func (r *ACPRuntime) SetExecutor(executor tools.CommandExecutor) {
	r.executor = executor
}

// SetValidator updates the validator in the runtime.
func (r *ACPRuntime) SetValidator(validator tools.CommandValidator) {
	r.validator = validator
}

// SetACPAgent updates the ACP agent in the runtime.
func (r *ACPRuntime) SetACPAgent(acpAgent ACPAgentInterface) {
	r.acpAgent = acpAgent
}

// SetApprovalHandler updates the approval handler in the runtime.
func (r *ACPRuntime) SetApprovalHandler(handler security.ApprovalHandler) {
	r.approvalHandler = handler
}

// SessionStorage returns the session storage.
func (r *ACPRuntime) SessionStorage() session.Storage {
	return r.storage
}

// SessionID returns the current session ID.
func (r *ACPRuntime) SessionID() string {
	return r.sessionID
}

// SupportsTerminals returns true if client supports terminals.
func (r *ACPRuntime) SupportsTerminals() bool {
	// Check from ACP agent first (most up-to-date).
	if r.acpAgent != nil {
		if caps := r.acpAgent.GetClientCapabilities(); caps != nil {
			return caps.Terminal
		}
	}
	// Fall back to stored capabilities.
	return r.clientCaps != nil && r.clientCaps.Terminal
}

// TerminalClient returns the ACP terminal client.
func (r *ACPRuntime) TerminalClient() TerminalClient {
	return r.terminalClient
}

// SetTerminalClient updates the terminal client in the runtime.
func (r *ACPRuntime) SetTerminalClient(terminalClient TerminalClient) {
	r.terminalClient = terminalClient
}

// SetFilesystemClient updates the filesystem client in the runtime.
func (r *ACPRuntime) SetFilesystemClient(filesystemClient FilesystemClient) {
	r.filesystemClient = filesystemClient
}

// SetClientCapabilities updates the client capabilities in the runtime.
func (r *ACPRuntime) SetClientCapabilities(caps *acpsdk.ClientCapabilities) {
	r.clientCaps = caps
}

// acpNotificationSender wraps ACP notification conversion to implement NotificationSender.
type acpNotificationSender struct {
	acpAgent  ACPAgentInterface
	sessionID string
	emitter   *events.EventEmitter
}

// SendToolCallStart implements the SendToolCallStart operation.
func (s *acpNotificationSender) SendToolCallStart(_ context.Context, _, _ string, _ tools.ToolParameters) error {
	// ACP notifications are sent via event emission, which is handled by the ACP agent
	// This is called from event emission, handled by the agent's event processing.
	return nil
}

// SendToolCallUpdate implements the SendToolCallUpdate operation.
func (s *acpNotificationSender) SendToolCallUpdate(_ context.Context, _, _ string, _ any) error {
	// ACP notifications are sent via event emission.
	return nil
}

// SendToolCallComplete implements the SendToolCallComplete operation.
func (s *acpNotificationSender) SendToolCallComplete(_ context.Context, _ string, _ bool, _ string, _ error) error {
	// ACP notifications are sent via event emission.
	return nil
}

// SendMessageChunk implements the SendMessageChunk operation.
func (s *acpNotificationSender) SendMessageChunk(_ context.Context, _ string) error {
	// ACP notifications are sent via event emission.
	return nil
}

// SendPlanUpdate implements the SendPlanUpdate operation.
func (s *acpNotificationSender) SendPlanUpdate(_ context.Context, _ []PlanEntry) error {
	// ACP notifications are sent via event emission.
	return nil
}
