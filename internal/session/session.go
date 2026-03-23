// Package session provides session state management for conversations.
package session

import (
	"errors"
	"fmt"
	"slices"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/dmytrogajewski/spin/pkg/apperr"
	"github.com/dmytrogajewski/spin/internal/state"
)

var (
	// ErrCannotTransitionFromArchivedState is a sentinel error.
	ErrCannotTransitionFromArchivedState = errors.New("cannot transition from archived state")
	// ErrCannotTransitionFromToActive is a sentinel error.
	ErrCannotTransitionFromToActive = errors.New("cannot transition from  to active")
	// ErrSessionIDIsEmpty is a sentinel error.
	ErrSessionIDIsEmpty = errors.New("session ID is empty")
	// ErrWorkDirectoryIsEmpty is a sentinel error.
	ErrWorkDirectoryIsEmpty = errors.New("work directory is empty")
	// ErrUpdatedAtIsBeforeCreatedAt is a sentinel error.
	ErrUpdatedAtIsBeforeCreatedAt = errors.New("updated_at is before created_at")
	// ErrInvalidState is a sentinel error.
	ErrInvalidState = errors.New("invalid state")
)

// CurrentSchemaVersion is the current session schema version for migrations.
const CurrentSchemaVersion = 1

// State is now unified with state.State for consistency.
// Use state.State instead of this type.
type State = state.State

// Session states - now using unified state constants.
const (
	// StateActive is exported.
	StateActive = state.StateIdle // Session is active (idle, not running).
	// StateRunning is exported.
	StateRunning = state.StateRunning // Session has active execution.
	// StatePaused is exported.
	StatePaused = state.StatePaused // Session is paused.
	// StateCompleted is exported.
	StateCompleted = state.StateCompleted // Session completed successfully.
	// StateFailed is exported.
	StateFailed = state.StateFailed // Session failed.
	// StateCancelled is exported.
	StateCancelled = state.StateCancelled // Session canceled by user.
	// StateArchived is exported.
	StateArchived = state.StateArchived // Session archived.
)

// Session-specific state methods are now handled by state.UnifiedState.

// Session represents a persistent conversation session.
// Note: Conversation content (messages) is stored separately in history.History.
// Session only tracks metadata, state, and configuration.
type Session struct {
	ID        string        // Unique session identifier (UUID string, for storage).
	WorkDir   string        // Working directory for this session.
	CreatedAt time.Time     // Session creation timestamp.
	UpdatedAt time.Time     // Last update timestamp.
	Metadata  Metadata      // Session metadata.
	State     State         // Current session state.
	Version   int           // Schema version for migrations.
	mu        *sync.RWMutex // Protects all fields.
}

// NewSession creates a new session with the given working directory.
// A unique session ID (UUID string) is automatically generated.
// The ID is a string for storage compatibility, but should be converted to protocol.ConversationID
// when used in conversation.Conversation.
func NewSession(workDir string) *Session {
	now := time.Now()

	return &Session{
		ID:        uuid.New().String(),
		WorkDir:   workDir,
		CreatedAt: now,
		UpdatedAt: now,
		Metadata:  Metadata{},
		State:     StateActive,
		Version:   CurrentSchemaVersion,
		mu:        &sync.RWMutex{},
	}
}

// ensureMu lazily initializes the mutex for deserialized sessions.
func (s *Session) ensureMu() {
	if s.mu == nil {
		s.mu = &sync.RWMutex{}
	}
}

// MigrateVersion applies schema version defaults for deserialized sessions.
// If Version is zero (absent in old JSON), it sets it to CurrentSchemaVersion.
func (s *Session) MigrateVersion() {
	if s.Version == 0 {
		s.Version = CurrentSchemaVersion
	}
}

// IncrementTurnCount increments the turn counter and updates tokens used.
// This is called when a turn completes. The actual messages are stored in history.History.
func (s *Session) IncrementTurnCount(tokensUsed int) {
	s.ensureMu()
	s.mu.Lock()
	defer s.mu.Unlock()

	s.Metadata.TotalTurns++
	s.Metadata.TokensUsed += tokensUsed
	s.UpdatedAt = time.Now()
}

// RecordLLMCall records cost metrics from a single LLM API call.
// It atomically updates input/output tokens, cost, call count, and TokensUsed.
func (s *Session) RecordLLMCall(inputTokens, outputTokens int, costUSD float64) {
	s.ensureMu()
	s.mu.Lock()
	defer s.mu.Unlock()

	s.Metadata.CostTracking.InputTokens += inputTokens
	s.Metadata.CostTracking.OutputTokens += outputTokens
	s.Metadata.CostTracking.TotalCostUSD += costUSD
	s.Metadata.CostTracking.APICallCount++
	s.Metadata.TokensUsed += inputTokens + outputTokens
	s.UpdatedAt = time.Now()
}

// UpdateMetadata updates session metadata using a callback function.
func (s *Session) UpdateMetadata(fn func(*Metadata)) error {
	s.ensureMu()
	s.mu.Lock()
	defer s.mu.Unlock()

	fn(&s.Metadata)
	s.UpdatedAt = time.Now()

	return nil
}

// SetState updates session state with validation.
func (s *Session) SetState(newState State) error {
	s.ensureMu()
	s.mu.Lock()
	defer s.mu.Unlock()

	// Validate state transition.
	err := s.validateStateTransition(s.State, newState)
	if err != nil {
		return err
	}

	s.State = newState
	s.UpdatedAt = time.Now()

	return nil
}

// validateStateTransition checks if a state transition is valid.
func (s *Session) validateStateTransition(from, to State) error {
	// Archived is terminal - cannot transition from it.
	if from == StateArchived {
		return ErrCannotTransitionFromArchivedState
	}

	// Cannot transition back to active from terminal states.
	if to == StateActive && (from == StateCompleted || from == StateFailed || from == StateCancelled) {
		return fmt.Errorf("cannot transition from %s to active: %w", from, ErrCannotTransitionFromToActive)
	}

	return nil
}

// AddTag adds a tag to the session.
func (s *Session) AddTag(tag string) error {
	s.ensureMu()
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check for duplicate.
	if slices.Contains(s.Metadata.Tags, tag) {
		return nil // Silently ignore duplicate.
	}

	s.Metadata.Tags = append(s.Metadata.Tags, tag)
	s.UpdatedAt = time.Now()

	return nil
}

// RemoveTag removes a tag from the session.
func (s *Session) RemoveTag(tag string) error {
	s.ensureMu()
	s.mu.Lock()
	defer s.mu.Unlock()

	before := len(s.Metadata.Tags)
	s.Metadata.Tags = slices.DeleteFunc(s.Metadata.Tags, func(existing string) bool {
		return existing == tag
	})

	if len(s.Metadata.Tags) != before {
		s.UpdatedAt = time.Now()
	}

	return nil
}

// SetTitle updates the session title.
func (s *Session) SetTitle(title string) error {
	s.ensureMu()
	s.mu.Lock()
	defer s.mu.Unlock()

	s.Metadata.Title = title
	s.UpdatedAt = time.Now()

	return nil
}

// Validate checks session integrity.
func (s *Session) Validate() error {
	s.ensureMu()

	s.mu.RLock()
	defer s.mu.RUnlock()

	var errs apperr.ErrorList

	s.validateBasicFields(&errs)

	return errs.Err()
}

// validateBasicFields validates basic session fields.
func (s *Session) validateBasicFields(errs *apperr.ErrorList) {
	// Validate ID.
	if s.ID == "" {
		errs.Add(ErrSessionIDIsEmpty)
	}

	_, err := uuid.Parse(s.ID)
	if err != nil {
		errs.Add(fmt.Errorf("session ID is not a valid UUID: %w", err))
	}

	// Validate WorkDir.
	if s.WorkDir == "" {
		errs.Add(ErrWorkDirectoryIsEmpty)
	}

	// Validate timestamps.
	if s.UpdatedAt.Before(s.CreatedAt) {
		errs.Add(ErrUpdatedAtIsBeforeCreatedAt)
	}

	// Validate state.
	if !isValidState(s.State) {
		errs.Add(fmt.Errorf("invalid state: %s: %w", s.State, ErrInvalidState))
	}
}

// isValidState checks if a state value is valid.
func isValidState(st State) bool {
	switch st {
	case StateActive, StateCompleted, StateFailed, StateArchived, StateCancelled:
		return true
	default:
		return false
	}
}
