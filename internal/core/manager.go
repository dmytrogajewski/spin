package core

import (
	"context"
	"errors"
	"fmt"

	"github.com/dmytrogajewski/spin/internal/session"
	"github.com/dmytrogajewski/spin/internal/llm"
	"github.com/dmytrogajewski/spin/internal/tools"
)

// Manager coordinates conversation lifecycle and state management.
// It serves as the main entry point for creating and managing conversations.
type Manager struct {
	cfg          *Config
	llm          llm.Provider
	emitter      *EventEmitter
	storage      session.Storage
	toolRegistry *tools.Registry
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
	if workDir == "" {
		workDir = m.cfg.WorkDir
	}

	// Build core dependencies for an Agent
	validator := NewValidator()
	executor, err := NewExecutor(workDir)
	if err != nil {
		return nil, err
	}
	ctxEnv := &Environment{WorkDir: workDir}

	// Build agent with optional tool registry
	var agentOpts []AgentOption
	if m.toolRegistry != nil {
		agentOpts = append(agentOpts, WithToolRegistry(m.toolRegistry))
	}

	agent, err := NewAgent(m.llm, executor, validator, ctxEnv, m.emitter, agentOpts...)
	if err != nil {
		return nil, err
	}
	history := NewHistoryWithDefaults()
	_ = history.AddSystemMessage("You are a helpful AI coding assistant.")

	conv := NewConversation(agent, history, m.emitter)
	return conv, nil
}

// ResumeConversation loads an existing session by ID and creates a new Conversation
// with fully restored history. Returns an error if the session does not exist.
func (m *Manager) ResumeConversation(ctx context.Context, sessionID string) (*Conversation, error) {
	if sessionID == "" {
		return nil, errors.New("sessionID is required")
	}

	// Load full session from storage
	sess, err := m.storage.Load(sessionID)
	if err != nil {
		return nil, fmt.Errorf("load session: %w", err)
	}

	// Build agent with session workdir
	validator := NewValidator()
	executor, err := NewExecutor(sess.WorkDir)
	if err != nil {
		return nil, fmt.Errorf("create executor: %w", err)
	}
	ctxEnv := &Environment{WorkDir: sess.WorkDir}

	// Build agent with optional tool registry
	var agentOpts []AgentOption
	if m.toolRegistry != nil {
		agentOpts = append(agentOpts, WithToolRegistry(m.toolRegistry))
	}

	agent, err := NewAgent(m.llm, executor, validator, ctxEnv, m.emitter, agentOpts...)
	if err != nil {
		return nil, fmt.Errorf("create agent: %w", err)
	}

	// Restore history from session turns
	history := NewHistoryWithDefaults()
	for _, t := range sess.Turns {
		// Add user input
		if t.UserInput != "" {
			_ = history.AddUserMessage(t.UserInput)
		}
		// Add assistant response
		if t.AIResponse != "" {
			_ = history.AddAssistantMessage(t.AIResponse)
		}
		// Note: Tool messages not yet supported in turn history restoration
	}

	conv := NewConversation(agent, history, m.emitter)
	return conv, nil
}

// ListConversations returns session metadata with optional filtering.
// The filter parameter can be a session.Filter or nil for all sessions.
func (m *Manager) ListConversations(ctx context.Context, filter any) ([]*session.Metadata, error) {
	var f session.Filter
	if filter != nil {
		if sf, ok := filter.(session.Filter); ok {
			f = sf
		}
	}

	return m.storage.ListMetadata(f)
}

// ArchiveConversation marks a session as archived by updating its state and persisting.
func (m *Manager) ArchiveConversation(ctx context.Context, sessionID string) error {
	if sessionID == "" {
		return errors.New("sessionID is required")
	}

	// Load session
	sess, err := m.storage.Load(sessionID)
	if err != nil {
		return fmt.Errorf("load session: %w", err)
	}

	// Update state to archived
	archived := session.StateArchived
	sess.State = archived

	// Save back to storage
	if err := m.storage.Save(sess); err != nil {
		return fmt.Errorf("save archived session: %w", err)
	}

	return nil
}
