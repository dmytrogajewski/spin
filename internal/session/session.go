// Package session provides session state management for conversations.
package session

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/dmytrogajewski/spin/internal/orchestration"
	"github.com/dmytrogajewski/spin/internal/state"
	"github.com/google/uuid"
)

// CurrentSchemaVersion is the current session schema version for migrations.
const CurrentSchemaVersion = 1

// State is now unified with state.State for consistency.
// Use state.State instead of this type.
type State = state.State

// Session states - now using unified state constants.
const (
	StateActive    = state.StateIdle      // Session is active (idle, not running)
	StateRunning   = state.StateRunning   // Session has active execution
	StatePaused    = state.StatePaused    // Session is paused
	StateCompleted = state.StateCompleted // Session completed successfully
	StateFailed    = state.StateFailed    // Session failed
	StateCancelled = state.StateCancelled // Session cancelled by user
	StateArchived  = state.StateArchived  // Session archived
)

// Session-specific state methods are now handled by state.UnifiedState.

// Session represents a persistent conversation session.
type Session struct {
	ID        string                // Unique session identifier (UUID)
	WorkDir   string                // Working directory for this session
	CreatedAt time.Time             // Session creation timestamp
	UpdatedAt time.Time             // Last update timestamp
	Turns     []*orchestration.Turn // Conversation turns
	Metadata  Metadata              // Session metadata
	State     State                 // Current session state
	Version   int                   // Schema version for migrations
	mu        sync.RWMutex          // Protects all fields
}

// NewSession creates a new session with the given working directory.
// A unique session ID (UUID) is automatically generated.
func NewSession(workDir string) *Session {
	now := time.Now()
	return &Session{
		ID:        uuid.New().String(),
		WorkDir:   workDir,
		CreatedAt: now,
		UpdatedAt: now,
		Turns:     make([]*orchestration.Turn, 0),
		Metadata:  Metadata{},
		State:     StateActive,
		Version:   CurrentSchemaVersion,
	}
}

// AddTurn appends a turn to the session.
func (s *Session) AddTurn(t *orchestration.Turn) error {
	if t == nil {
		return errors.New("turn cannot be nil")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.Turns = append(s.Turns, t)
	s.UpdatedAt = time.Now()
	s.Metadata.TotalTurns++
	s.Metadata.TokensUsed += t.Tokens.TotalTokens

	return nil
}

// GetTurn retrieves a turn by ID.
func (s *Session) GetTurn(turnID string) (*orchestration.Turn, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, t := range s.Turns {
		if t.ID == turnID {
			return t, nil
		}
	}

	return nil, fmt.Errorf("turn not found: %s", turnID)
}

// LastTurn returns the most recent turn.
func (s *Session) LastTurn() *orchestration.Turn {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if len(s.Turns) == 0 {
		return nil
	}

	return s.Turns[len(s.Turns)-1]
}

// TurnCount returns the number of turns.
func (s *Session) TurnCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return len(s.Turns)
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

	// Validate state transition
	if err := s.validateStateTransition(s.State, state); err != nil {
		return err
	}

	s.State = state
	s.UpdatedAt = time.Now()

	return nil
}

// validateStateTransition checks if a state transition is valid.
func (s *Session) validateStateTransition(from, to State) error {
	// Archived is terminal - cannot transition from it
	if from == StateArchived {
		return fmt.Errorf("cannot transition from archived state")
	}

	// Cannot transition back to active from terminal states
	if to == StateActive && (from == StateCompleted || from == StateFailed || from == StateCancelled) {
		return fmt.Errorf("cannot transition from %s to active", from)
	}

	return nil
}

// AddTag adds a tag to the session.
func (s *Session) AddTag(tag string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check for duplicate
	for _, existingTag := range s.Metadata.Tags {
		if existingTag == tag {
			return nil // Silently ignore duplicate
		}
	}

	s.Metadata.Tags = append(s.Metadata.Tags, tag)
	s.UpdatedAt = time.Now()

	return nil
}

// RemoveTag removes a tag from the session.
func (s *Session) RemoveTag(tag string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Find and remove tag
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

	var errs []error

	errs = append(errs, s.validateBasicFields()...)
	errs = append(errs, s.validateTurns()...)
	errs = append(errs, s.validateMetadata()...)

	if len(errs) > 0 {
		return errors.Join(errs...)
	}

	return nil
}

// validateBasicFields validates basic session fields.
func (s *Session) validateBasicFields() []error {
	var errs []error

	// Validate ID
	if s.ID == "" {
		errs = append(errs, errors.New("session ID is empty"))
	} else if _, err := uuid.Parse(s.ID); err != nil {
		errs = append(errs, fmt.Errorf("session ID is not a valid UUID: %w", err))
	}

	// Validate WorkDir
	if s.WorkDir == "" {
		errs = append(errs, errors.New("work directory is empty"))
	}

	// Validate timestamps
	if s.UpdatedAt.Before(s.CreatedAt) {
		errs = append(errs, errors.New("UpdatedAt is before CreatedAt"))
	}

	// Validate state
	if !isValidState(s.State) {
		errs = append(errs, fmt.Errorf("invalid state: %s", s.State))
	}

	return errs
}

// validateTurns validates turn-related fields.
func (s *Session) validateTurns() []error {
	var errs []error

	// Check for duplicate turn IDs
	turnIDs := make(map[string]bool)
	for _, t := range s.Turns {
		if turnIDs[t.ID] {
			errs = append(errs, fmt.Errorf("duplicate turn ID: %s", t.ID))
		}
		turnIDs[t.ID] = true
	}

	return errs
}

// validateMetadata validates metadata consistency.
func (s *Session) validateMetadata() []error {
	var errs []error

	// Validate turn count consistency
	if s.Metadata.TotalTurns != len(s.Turns) {
		errs = append(errs, fmt.Errorf("metadata turn count (%d) does not match actual turns (%d)",
			s.Metadata.TotalTurns, len(s.Turns)))
	}

	// Validate token count consistency
	actualTokens := s.calculateActualTokens()
	if s.Metadata.TokensUsed != actualTokens {
		errs = append(errs, fmt.Errorf("metadata tokens used (%d) does not match actual (%d)",
			s.Metadata.TokensUsed, actualTokens))
	}

	return errs
}

// calculateActualTokens calculates the total tokens from all turns.
func (s *Session) calculateActualTokens() int {
	actualTokens := 0
	for _, t := range s.Turns {
		actualTokens += t.Tokens.TotalTokens
	}
	return actualTokens
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
