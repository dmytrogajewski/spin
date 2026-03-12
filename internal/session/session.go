// Package session provides session state management for conversations.
package session

import (
	"errors"
	"fmt"
	"slices"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/dmytrogajewski/spin/internal/state"
)

// CurrentSchemaVersion is the current session schema version for migrations.
const CurrentSchemaVersion = 1

// State is now unified with state.State for consistency.
// Use state.State instead of this type.
type State = state.State

// Session states - now using unified state constants.
const (
	StateActive    = state.StateIdle      // Session is active (idle, not running).
	StateRunning   = state.StateRunning   // Session has active execution.
	StatePaused    = state.StatePaused    // Session is paused.
	StateCompleted = state.StateCompleted // Session completed successfully.
	StateFailed    = state.StateFailed    // Session failed.
	StateCancelled = state.StateCancelled // Session canceled by user.
	StateArchived  = state.StateArchived  // Session archived.
)

// Session-specific state methods are now handled by state.UnifiedState.

// Session represents a persistent conversation session.
// Note: Conversation content (messages) is stored separately in history.History.
// Session only tracks metadata, state, and configuration.
type Session struct {
	ID        string       // Unique session identifier (UUID string, for storage).
	WorkDir   string       // Working directory for this session.
	CreatedAt time.Time    // Session creation timestamp.
	UpdatedAt time.Time    // Last update timestamp.
	Metadata  Metadata     // Session metadata.
	State     State        // Current session state.
	Version   int          // Schema version for migrations.
	mu        sync.RWMutex // Protects all fields.
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
	}
}

// IncrementTurnCount increments the turn counter and updates tokens used.
// This is called when a turn completes. The actual messages are stored in history.History.
func (s *Session) IncrementTurnCount(tokensUsed int) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.Metadata.TotalTurns++
	s.Metadata.TokensUsed += tokensUsed
	s.UpdatedAt = time.Now()
}

// UpdateMetadata updates session metadata using a callback function.
func (s *Session) UpdateMetadata(fn func(*Metadata)) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	fn(&s.Metadata)
	s.UpdatedAt = time.Now()

	return nil
}

// SetState updates session state with validation.
func (s *Session) SetState(state State) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Validate state transition.
	err := s.validateStateTransition(s.State, state)
	if err != nil {
		return err
	}

	s.State = state
	s.UpdatedAt = time.Now()

	return nil
}

// validateStateTransition checks if a state transition is valid.
func (s *Session) validateStateTransition(from, to State) error {
	// Archived is terminal - cannot transition from it.
	if from == StateArchived {
		return errors.New("cannot transition from archived state")
	}

	// Cannot transition back to active from terminal states.
	if to == StateActive && (from == StateCompleted || from == StateFailed || from == StateCancelled) {
		return fmt.Errorf("cannot transition from %s to active", from)
	}

	return nil
}

// AddTag adds a tag to the session.
func (s *Session) AddTag(tag string) error {
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
	s.mu.Lock()
	defer s.mu.Unlock()

	// Find and remove tag.
	for i, existingTag := range s.Metadata.Tags {
		if existingTag == tag {
			s.Metadata.Tags = append(s.Metadata.Tags[:i], s.Metadata.Tags[i+1:]...)
			s.UpdatedAt = time.Now()

			break
		}
	}

	return nil
}

// SetTitle updates the session title.
func (s *Session) SetTitle(title string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.Metadata.Title = title
	s.UpdatedAt = time.Now()

	return nil
}

// Validate checks session integrity.
func (s *Session) Validate() error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	errs := s.validateBasicFields()

	if len(errs) > 0 {
		return errors.Join(errs...)
	}

	return nil
}

// validateBasicFields validates basic session fields.
func (s *Session) validateBasicFields() []error {
	var errs []error

	// Validate ID.
	if s.ID == "" {
		errs = append(errs, errors.New("session ID is empty"))
	}

	_, err := uuid.Parse(s.ID)
	if err != nil {
		errs = append(errs, fmt.Errorf("session ID is not a valid UUID: %w", err))
	}

	// Validate WorkDir.
	if s.WorkDir == "" {
		errs = append(errs, errors.New("work directory is empty"))
	}

	// Validate timestamps.
	if s.UpdatedAt.Before(s.CreatedAt) {
		errs = append(errs, errors.New("updated_at is before created_at"))
	}

	// Validate state.
	if !isValidState(s.State) {
		errs = append(errs, fmt.Errorf("invalid state: %s", s.State))
	}

	return errs
}

// isValidState checks if a state value is valid.
func isValidState(state State) bool {
	switch state {
	case StateActive, StateCompleted, StateFailed, StateArchived, StateCancelled:
		return true
	default:
		return false
	}
}
