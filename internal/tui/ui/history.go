package ui

// History manages input history with a ring buffer.
type History struct {
	items    []string
	maxSize  int
	position int    // -1 = at current input, 0+ = in history
	tempBuf  string // Temporary buffer for current input
}

// DefaultMaxHistory is the default maximum history size.
const DefaultMaxHistory = 100

// NewHistory creates a new history manager.
func NewHistory(maxSize int) *History {
	if maxSize <= 0 {
		maxSize = DefaultMaxHistory
	}

	return &History{
		items:    make([]string, 0, maxSize),
		maxSize:  maxSize,
		position: -1,
		tempBuf:  "",
	}
}

// Add adds an item to history (deduplicated).
func (h *History) Add(item string) {
	// Don't add empty strings
	if item == "" {
		return
	}

	// Don't add duplicates (remove old occurrence)
	for i, existing := range h.items {
		if existing == item {
			h.items = append(h.items[:i], h.items[i+1:]...)
			break
		}
	}

	// Add to end
	h.items = append(h.items, item)

	// Trim if exceeds max size
	if len(h.items) > h.maxSize {
		h.items = h.items[len(h.items)-h.maxSize:]
	}

	h.position = -1
}

// Previous returns the previous history item.
func (h *History) Previous() (string, bool) {
	if len(h.items) == 0 {
		return "", false
	}

	// First time: move from current to most recent
	if h.position == -1 {
		h.position = len(h.items) - 1
		return h.items[h.position], true
	}

	// Already in history: move backward
	if h.position > 0 {
		h.position--
		return h.items[h.position], true
	}

	// At oldest item
	return h.items[h.position], false
}

// Next returns the next history item.
func (h *History) Next() (string, bool) {
	if len(h.items) == 0 || h.position == -1 {
		return "", false
	}

	// Move forward
	if h.position < len(h.items)-1 {
		h.position++
		return h.items[h.position], true
	}

	// Back to current input (temp buffer)
	h.position = -1
	return h.tempBuf, true
}

// Reset resets the history position.
func (h *History) Reset() {
	h.position = -1
	h.tempBuf = ""
}

// SetTempBuffer stores the current input before navigation.
func (h *History) SetTempBuffer(value string) {
	h.tempBuf = value
}

// GetAll returns all history items.
func (h *History) GetAll() []string {
	return h.items
}

// Clear clears the history.
func (h *History) Clear() {
	h.items = make([]string, 0, h.maxSize)
	h.position = -1
	h.tempBuf = ""
}
