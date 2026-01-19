package session

import "time"

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
