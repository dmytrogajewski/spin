package conversation

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"

	"github.com/dmytrogajewski/spin/internal/history"
	"github.com/dmytrogajewski/spin/internal/session"
)

var (
	ErrConversationFactoryIsRequired = errors.New("conversation factory is required")
	ErrSessionIDCannotBeEmpty = errors.New("session ID cannot be empty")
	ErrConversationNotFound = errors.New("conversation not found")
	ErrHistoryStorageNotConfigured = errors.New("history storage not configured")
	ErrSessionNotFound = errors.New("session not found")
	ErrHistoryStorageNotConfigured2 = errors.New("history storage not configured")
	ErrConversationNotFound2 = errors.New("conversation not found")
	ErrErrorsClosingConversations = errors.New("errors closing conversations")
	ErrConversationNotFound3 = errors.New("conversation not found")
	ErrConversationNotFound4 = errors.New("conversation not found")
)

// Factory creates new Conversation instances.
// The factory receives the session ID and working directory, and returns
// a fully configured Conversation or an error.
type Factory func(ctx context.Context, sessionID string, workDir string) (*Conversation, error)

// Manager manages multiple concurrent conversations.
// This is useful for protocols like ACP that support multiple sessions.
// For single-session modes (TUI, exec), use Conversation directly.
type Manager struct {
	conversations map[string]*Conversation
	factory       Factory
	storage       session.Storage
	histStorage   history.Storage
	logger        *slog.Logger
	mu            sync.RWMutex
}

// ManagerConfig contains configuration for creating a Manager.
type ManagerConfig struct {
	// Factory creates new Conversation instances (required).
	Factory Factory

	// Storage for session persistence (optional).
	Storage session.Storage

	// HistoryStorage for history persistence (optional).
	HistoryStorage history.Storage

	// Logger for debug output (optional, uses slog.Default() if nil).
	Logger *slog.Logger
}

// NewManager creates a new conversation manager.
func NewManager(cfg ManagerConfig) (*Manager, error) {
	if cfg.Factory == nil {
		return nil, ErrConversationFactoryIsRequired
	}

	logger := cfg.Logger
	if logger == nil {
		logger = slog.Default()
	}

	return &Manager{
		conversations: make(map[string]*Conversation),
		factory:       cfg.Factory,
		storage:       cfg.Storage,
		histStorage:   cfg.HistoryStorage,
		logger:        logger,
	}, nil
}

// GetOrCreate returns an existing conversation or creates a new one.
// If the conversation already exists, it is returned.
// Otherwise, the factory is called to create a new conversation.
func (m *Manager) GetOrCreate(ctx context.Context, sessionID string, workDir string) (*Conversation, error) {
	if sessionID == "" {
		return nil, ErrSessionIDCannotBeEmpty
	}

	// Fast path: check if conversation exists.
	m.mu.RLock()

	if conv, ok := m.conversations[sessionID]; ok {
		m.mu.RUnlock()

		return conv, nil
	}

	m.mu.RUnlock()

	// Slow path: create new conversation.
	m.mu.Lock()
	defer m.mu.Unlock()

	// Double-check after acquiring write lock.
	if conv, ok := m.conversations[sessionID]; ok {
		return conv, nil
	}

	// Create new conversation via factory.
	conv, err := m.factory(ctx, sessionID, workDir)
	if err != nil {
		return nil, fmt.Errorf("create conversation: %w", err)
	}

	// Set the ID to match the session ID.
	conv.SetID(sessionID)

	m.conversations[sessionID] = conv
	m.logger.InfoContext(ctx, "conversation created", "session_id", sessionID, "work_dir", workDir)

	return conv, nil
}

// Get returns a conversation by session ID, or nil if not found.
func (m *Manager) Get(sessionID string) (*Conversation, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	conv, ok := m.conversations[sessionID]

	return conv, ok
}

// Remove removes and closes a conversation.
// Returns an error if the conversation doesn't exist or close fails.
func (m *Manager) Remove(ctx context.Context, sessionID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	conv, ok := m.conversations[sessionID]
	if !ok {
return fmt.Errorf("conversation not found: %s: %w", sessionID, ErrConversationNotFound)
	}

	// Close the conversation.
	err := conv.Close()
	if err != nil {
		m.logger.WarnContext(ctx, "error closing conversation", "session_id", sessionID, "error", err)
	}

	delete(m.conversations, sessionID)
	m.logger.InfoContext(ctx, "conversation removed", "session_id", sessionID)

	return nil
}

// Cancel cancels the active turn for a specific session.
func (m *Manager) Cancel(ctx context.Context, sessionID string) {
	m.mu.RLock()
	conv, ok := m.conversations[sessionID]
	m.mu.RUnlock()

	if ok {
		conv.Cancel()
		m.logger.DebugContext(ctx, "conversation canceled", "session_id", sessionID)
	}
}

// Load loads a conversation from storage.
// This creates a new conversation and restores its history from storage.
func (m *Manager) Load(ctx context.Context, sessionID string, workDir string) (*Conversation, error) {
	if m.histStorage == nil {
		return nil, ErrHistoryStorageNotConfigured
	}

	// Check if history exists.
	exists, err := m.histStorage.Exists(sessionID)
	if err != nil {
		return nil, fmt.Errorf("check history exists: %w", err)
	}

	if !exists {
return nil, fmt.Errorf("session not found: %s: %w", sessionID, ErrSessionNotFound)
	}

	// Create conversation via factory.
	conv, err := m.GetOrCreate(ctx, sessionID, workDir)
	if err != nil {
		return nil, fmt.Errorf("create conversation: %w", err)
	}

	// Load history into the conversation.
	hist := conv.history
	err = hist.Load(m.histStorage, sessionID)
	if err != nil {
		// Remove the conversation if loading fails.
		_ = m.Remove(ctx, sessionID)

		return nil, fmt.Errorf("load history: %w", err)
	}

	m.logger.InfoContext(ctx, "conversation loaded from storage", "session_id", sessionID)

	return conv, nil
}

// Save persists a conversation's history to storage.
func (m *Manager) Save(ctx context.Context, sessionID string) error {
	if m.histStorage == nil {
		return ErrHistoryStorageNotConfigured2
	}

	m.mu.RLock()
	conv, ok := m.conversations[sessionID]
	m.mu.RUnlock()

	if !ok {
return fmt.Errorf("conversation not found: %s: %w", sessionID, ErrConversationNotFound2)
	}

	err := conv.history.Save(m.histStorage, sessionID)
	if err != nil {
		return fmt.Errorf("save history: %w", err)
	}

	m.logger.DebugContext(ctx, "conversation saved", "session_id", sessionID)

	return nil
}

// List returns all active session IDs.
func (m *Manager) List() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	ids := make([]string, 0, len(m.conversations))
	for id := range m.conversations {
		ids = append(ids, id)
	}

	return ids
}

// Count returns the number of active conversations.
func (m *Manager) Count() int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return len(m.conversations)
}

// Close closes all conversations and cleans up resources.
func (m *Manager) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	var errs []error

	for id, conv := range m.conversations {
		err := conv.Close()
		if err != nil {
			errs = append(errs, fmt.Errorf("close %s: %w", id, err))
		}
	}

	m.conversations = make(map[string]*Conversation)

	if len(errs) > 0 {
return fmt.Errorf("errors closing conversations: %v: %w", errs, ErrErrorsClosingConversations)
	}

	return nil
}

// RunTurn executes a turn on a specific session.
// This is a convenience method that gets or creates the conversation and runs the turn.
func (m *Manager) RunTurn(ctx context.Context, sessionID string, workDir string, input string) error {
	conv, err := m.GetOrCreate(ctx, sessionID, workDir)
	if err != nil {
		return err
	}

	return conv.RunTurn(ctx, input)
}

// SetTaskMode sets the task mode for a specific session.
func (m *Manager) SetTaskMode(sessionID string, mode string) error {
	m.mu.RLock()
	conv, ok := m.conversations[sessionID]
	m.mu.RUnlock()

	if !ok {
return fmt.Errorf("conversation not found: %s: %w", sessionID, ErrConversationNotFound3)
	}

	return conv.SetTaskMode(mode)
}

// GetTaskMode returns the task mode for a specific session.
func (m *Manager) GetTaskMode(sessionID string) (string, error) {
	m.mu.RLock()
	conv, ok := m.conversations[sessionID]
	m.mu.RUnlock()

	if !ok {
return "", fmt.Errorf("conversation not found: %s: %w", sessionID, ErrConversationNotFound4)
	}

	return conv.GetTaskMode(), nil
}
