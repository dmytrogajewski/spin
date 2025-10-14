package core

import (
	"context"
	"errors"
	"fmt"

	"github.com/dmytrogajewski/spin/internal/core/history"
	"github.com/dmytrogajewski/spin/internal/core/history/compress"
	"github.com/dmytrogajewski/spin/internal/core/task"
	"github.com/dmytrogajewski/spin/internal/llm"
	"github.com/dmytrogajewski/spin/internal/session"
	"github.com/dmytrogajewski/spin/internal/tools"
)

// Manager coordinates conversation lifecycle and state management.
// It serves as the main entry point for creating and managing conversations.
type Manager struct {
	cfg             *Config
	llm             llm.Provider
	emitter         *EventEmitter
	storage         session.Storage
	toolRegistry    *tools.Registry
	taskRegistry    *task.Registry // Task registry for all conversations
	mcpManager      *MCPManager
	approvalHandler ApprovalHandler
}

// Functional options
type ManagerOption func(*Manager) error

// WithLLM sets the LLM provider for the manager
func WithLLM(provider llm.Provider) ManagerOption {
	return func(m *Manager) error {
		m.llm = provider
		return nil
	}
}

// WithEmitter sets the shared event emitter
func WithEmitter(e *EventEmitter) ManagerOption {
	return func(m *Manager) error {
		m.emitter = e
		return nil
	}
}

// WithStorage sets the session storage backend
func WithStorage(s session.Storage) ManagerOption {
	return func(m *Manager) error {
		m.storage = s
		return nil
	}
}

// WithManagerToolRegistry sets a custom tool registry for all conversations created by this manager
func WithManagerToolRegistry(registry *tools.Registry) ManagerOption {
	return func(m *Manager) error {
		m.toolRegistry = registry
		return nil
	}
}

// WithMCPManager sets the MCP manager for connecting to external MCP servers
func WithMCPManager(mcpMgr *MCPManager) ManagerOption {
	return func(m *Manager) error {
		m.mcpManager = mcpMgr
		return nil
	}
}

// WithManagerApprovalHandler sets the approval handler for all agents created by this manager
func WithManagerApprovalHandler(handler ApprovalHandler) ManagerOption {
	return func(m *Manager) error {
		m.approvalHandler = handler
		return nil
	}
}

// WithManagerTaskRegistry sets a custom task registry for all agents created by this manager.
// The task registry defines available task modes (e.g., regular, review, compact, planning)
// and their configurations. If not provided, agents will use their default task registry.
//
// Example:
//
//	registry := task.NewRegistry()
//	registry.Register("custom", task.NewCompact())
//	registry.SetDefault("custom")
//	mgr := NewManager(cfg, WithManagerTaskRegistry(registry))
func WithManagerTaskRegistry(registry *task.Registry) ManagerOption {
	return func(m *Manager) error {
		if registry == nil {
			return errors.New("task registry cannot be nil")
		}
		m.taskRegistry = registry
		return nil
	}
}

// NewManager creates a new Manager
func NewManager(cfg *Config, opts ...ManagerOption) (*Manager, error) {
	if cfg == nil {
		return nil, errors.New("config cannot be nil")
	}
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	m := &Manager{cfg: cfg}
	for _, opt := range opts {
		if err := opt(m); err != nil {
			return nil, err
		}
	}
	if m.llm == nil {
		// Fallback mock for tests if not provided
		m.llm = llm.NewMockProvider("default")
	}
	if m.emitter == nil {
		m.emitter = NewEventEmitter(DefaultEventBufferSize)
	}
	if m.storage == nil {
		// Initialize default file storage
		fs, err := session.NewFileStorage(cfg.SessionDir)
		if err != nil {
			return nil, fmt.Errorf("initialize storage: %w", err)
		}
		m.storage = fs
	}
	return m, nil
}

// NewConversation starts a new conversation for the given workDir
func (m *Manager) NewConversation(ctx context.Context, workDir string) (*Conversation, error) {
	logger := withContext(ctx)
	logger.Info("creating new conversation", "work_dir", workDir)

	if workDir == "" {
		workDir = m.cfg.WorkDir
	}

	// Build core dependencies for an Agent
	validator := NewValidator()
	executor, err := NewExecutor(workDir)
	if err != nil {
		logger.Error("failed to create executor", "error", err, "work_dir", workDir)
		return nil, err
	}
	ctxEnv := &Environment{WorkDir: workDir}

	// Build agent with optional tool registry, task registry, and approval
	var agentOpts []AgentOption
	// Enable approval for dangerous commands
	agentOpts = append(agentOpts, WithRequireApproval(true))
	// Set approval handler if configured
	if m.approvalHandler != nil {
		agentOpts = append(agentOpts, WithApprovalHandler(m.approvalHandler))
	}
	if m.toolRegistry != nil {
		agentOpts = append(agentOpts, WithToolRegistry(m.toolRegistry))
		logger.Debug("using custom tool registry", "tool_count", len(m.toolRegistry.ListSchemas()))
	}
	// Pass task registry if configured
	if m.taskRegistry != nil {
		agentOpts = append(agentOpts, WithTaskRegistry(m.taskRegistry))
		logger.Debug("using custom task registry", "task_count", len(m.taskRegistry.List()))
	}

	agent, err := NewAgent(m.llm, executor, validator, ctxEnv, m.emitter, agentOpts...)
	if err != nil {
		logger.Error("failed to create agent", "error", err)
		return nil, err
	}

	// Create history with LLM summarization + hybrid fallback (if LLM available)
	hist := NewHistoryWithDefaults()

	if m.llm != nil {
		// Use composite compressor: LLM summarization (primary) + hybrid (fallback)
		adapter := history.NewLLMProviderAdapter(m.llm)
		compressor := compress.NewDefaultLLMWithHybridFallback(adapter)
		hist.SetCompressor(compressor)
	}

	// Set event emitter for compression notifications
	hist.SetEventEmitter(m.emitter)

	_ = hist.AddSystemMessage("You are a helpful AI coding assistant.")

	conv := NewConversation(agent, hist, m.emitter)
	logger.Info("conversation created successfully")
	return conv, nil
}

// NewConversationWithTask creates a new conversation starting in a specific task mode.
// This is a convenience method that calls NewConversation and then SetTaskMode.
//
// Parameters:
//   - ctx: Context for the operation
//   - workDir: Working directory for the conversation (empty string uses config default)
//   - taskName: Name of the task mode ("regular", "review", "compact", "planning")
//
// Returns:
//   - *Conversation: The new conversation in the specified mode
//   - error: If conversation creation fails or task mode is invalid
//
// Example:
//
//	conv, err := mgr.NewConversationWithTask(ctx, "/path/to/project", "review")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	// Conversation is now in review mode (read-only)
func (m *Manager) NewConversationWithTask(ctx context.Context, workDir string, taskName string) (*Conversation, error) {
	logger := withContext(ctx)
	logger.Info("creating new conversation with task mode", "work_dir", workDir, "task_mode", taskName)

	// Validate task name before creating conversation
	if taskName == "" {
		return nil, errors.New("task name cannot be empty")
	}

	// Create conversation
	conv, err := m.NewConversation(ctx, workDir)
	if err != nil {
		return nil, err
	}

	// Set task mode
	if err := conv.SetTaskMode(taskName); err != nil {
		logger.Error("failed to set task mode", "error", err, "task_mode", taskName)
		return nil, fmt.Errorf("set task mode: %w", err)
	}

	logger.Info("conversation created successfully with task mode", "task_mode", taskName)
	return conv, nil
}

// ResumeConversation loads an existing session by ID and creates a new Conversation
// with fully restored history. Returns an error if the session does not exist.
func (m *Manager) ResumeConversation(ctx context.Context, sessionID string) (*Conversation, error) {
	if sessionID == "" {
		return nil, errors.New("sessionID is required")
	}

	// Add session ID to context for logging
	ctx = WithSessionID(ctx, sessionID)
	logger := withContext(ctx)
	logger.Info("resuming conversation", "session_id", sessionID)

	// Load full session from storage
	sess, err := m.storage.Load(sessionID)
	if err != nil {
		logger.Error("failed to load session", "error", err, "session_id", sessionID)
		return nil, fmt.Errorf("load session: %w", err)
	}

	logger.Debug("session loaded", "work_dir", sess.WorkDir, "turn_count", len(sess.Turns))

	// Build agent with session workdir
	validator := NewValidator()
	executor, err := NewExecutor(sess.WorkDir)
	if err != nil {
		logger.Error("failed to create executor", "error", err, "work_dir", sess.WorkDir)
		return nil, fmt.Errorf("create executor: %w", err)
	}
	ctxEnv := &Environment{WorkDir: sess.WorkDir}

	// Build agent with optional tool registry, task registry, and approval
	var agentOpts []AgentOption
	// Enable approval for dangerous commands
	agentOpts = append(agentOpts, WithRequireApproval(true))
	// Set approval handler if configured
	if m.approvalHandler != nil {
		agentOpts = append(agentOpts, WithApprovalHandler(m.approvalHandler))
	}
	if m.toolRegistry != nil {
		agentOpts = append(agentOpts, WithToolRegistry(m.toolRegistry))
	}
	// Pass task registry if configured
	if m.taskRegistry != nil {
		agentOpts = append(agentOpts, WithTaskRegistry(m.taskRegistry))
		logger.Debug("using custom task registry for resumed conversation", "task_count", len(m.taskRegistry.List()))
	}

	agent, err := NewAgent(m.llm, executor, validator, ctxEnv, m.emitter, agentOpts...)
	if err != nil {
		logger.Error("failed to create agent", "error", err)
		return nil, fmt.Errorf("create agent: %w", err)
	}

	// Restore history from session turns with LLM summarization support
	hist := NewHistoryWithDefaults()

	if m.llm != nil {
		// Use composite compressor: LLM summarization (primary) + hybrid (fallback)
		adapter := history.NewLLMProviderAdapter(m.llm)
		compressor := compress.NewDefaultLLMWithHybridFallback(adapter)
		hist.SetCompressor(compressor)
	}

	// Set event emitter for compression notifications
	hist.SetEventEmitter(m.emitter)

	for _, t := range sess.Turns {
		// Add user input
		if t.UserInput != "" {
			_ = hist.AddUserMessage(t.UserInput)
		}
		// Add assistant response
		if t.AIResponse != "" {
			_ = hist.AddAssistantMessage(t.AIResponse)
		}
		// Note: Tool messages not yet supported in turn history restoration
	}

	conv := NewConversation(agent, hist, m.emitter)
	logger.Info("conversation resumed successfully", "session_id", sessionID, "history_messages", len(hist.Messages()))
	return conv, nil
}

// ListConversations returns session metadata with optional filtering.
// Pass nil filter to retrieve all sessions, or provide a *session.Filter for filtering.
func (m *Manager) ListConversations(ctx context.Context, filter *session.Filter) ([]*session.Metadata, error) {
	logger := withContext(ctx)
	logger.Debug("listing conversations")

	// Use an empty filter if nil is provided
	f := session.Filter{}
	if filter != nil {
		f = *filter
	}

	metadata, err := m.storage.ListMetadata(f)
	if err != nil {
		logger.Error("failed to list conversations", "error", err)
		return nil, err
	}

	logger.Info("conversations listed", "count", len(metadata))
	return metadata, nil
}

// ArchiveConversation marks a session as archived by updating its state and persisting.
func (m *Manager) ArchiveConversation(ctx context.Context, sessionID string) error {
	logger := withContext(ctx)
	logger.Info("archiving conversation", "session_id", sessionID)

	if sessionID == "" {
		return errors.New("sessionID is required")
	}

	// Load session
	sess, err := m.storage.Load(sessionID)
	if err != nil {
		logger.Error("failed to load session for archival", "error", err, "session_id", sessionID)
		return fmt.Errorf("load session: %w", err)
	}

	// Update state to archived
	archived := session.StateArchived
	sess.State = archived

	// Save back to storage
	if err := m.storage.Save(sess); err != nil {
		logger.Error("failed to save archived session", "error", err, "session_id", sessionID)
		return fmt.Errorf("save archived session: %w", err)
	}

	logger.Info("conversation archived successfully", "session_id", sessionID)
	return nil
}

// MCPManager returns the MCP manager if configured, or nil.
func (m *Manager) MCPManager() *MCPManager {
	return m.mcpManager
}
