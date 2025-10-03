package session

import "time"

// Storage provides session persistence operations.
type Storage interface {
	// Save writes session to storage
	Save(s *Session) error

	// Load reads session from storage
	Load(id string) (*Session, error)

	// Delete removes session from storage
	Delete(id string) error

	// Exists checks if session exists
	Exists(id string) (bool, error)

	// List returns all session IDs with optional filter
	List(filter Filter) ([]string, error)

	// ListMetadata returns session metadata without loading full sessions
	ListMetadata(filter Filter) ([]*Metadata, error)
}

// Filter for session queries.
type Filter struct {
	State         *State     // Filter by state
	WorkDir       string     // Filter by working directory
	CreatedAfter  *time.Time // Filter by creation date (inclusive)
	CreatedBefore *time.Time // Filter by creation date (inclusive)
	Tags          []string   // Filter by tags (OR logic)
	Limit         int        // Limit results
	Offset        int        // Pagination offset
}
