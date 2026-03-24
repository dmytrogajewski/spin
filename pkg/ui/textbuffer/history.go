package textbuffer

// History manages command history with ring buffer and navigation.
type History struct {
	entries []string // historical commands (newest first).
	limit   int      // max entries.
	pos     int      // current navigation position (-1 = not navigating).
	draft   string   // uncommitted input when navigating.
}

// NewHistory creates a new history with the specified limit.
// Limit is clamped to a minimum of 1.
func NewHistory(limit int) *History {
	if limit < 1 {
		limit = 1
	}

	return &History{
		entries: []string{},
		limit:   limit,
		pos:     -1,
		draft:   "",
	}
}

// Submit adds a command to history and resets navigation state.
func (h *History) Submit(line string) {
	// Add to front of entries (newest first).
	h.entries = append([]string{line}, h.entries...)

	// Enforce limit (drop oldest).
	if len(h.entries) > h.limit {
		h.entries = h.entries[:h.limit]
	}

	// Reset navigation state.
	h.pos = -1
	h.draft = ""
}

// PrevHistory navigates to the previous (older) command.
// Returns the entry and true if successful, empty string and false if at oldest.
// The currentDraft parameter is saved when starting navigation.
func (h *History) PrevHistory(currentDraft string) (string, bool) {
	if len(h.entries) == 0 {
		return "", false
	}

	if h.pos == -1 {
		// Starting navigation, save draft.
		h.draft = currentDraft
		h.pos = 0

		return h.entries[0], true
	}

	// Already navigating, try to go older.
	if h.pos >= len(h.entries)-1 {
		// Already at oldest.
		return "", false
	}

	h.pos++

	return h.entries[h.pos], true
}

// NextHistory navigates to the next (newer) command.
// Returns the entry and true if successful.
// When reaching beyond the newest entry, returns the saved draft.
func (h *History) NextHistory() (string, bool) {
	if h.pos == -1 {
		// Not navigating.
		return "", false
	}

	if h.pos == 0 {
		// At newest entry, return draft and reset.
		draft := h.draft
		h.pos = -1
		h.draft = ""

		return draft, true
	}

	// Go newer.
	h.pos--

	return h.entries[h.pos], true
}

// Reset clears the navigation state without modifying history.
func (h *History) Reset() {
	h.pos = -1
	h.draft = ""
}

// Entries returns a copy of all history entries (newest first).
func (h *History) Entries() []string {
	result := make([]string, len(h.entries))
	copy(result, h.entries)

	return result
}
