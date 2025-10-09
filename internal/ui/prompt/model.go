package prompt

// Model combines buffer and history into a single prompt state.
type Model struct {
	buffer  *Buffer
	history *History
}

// NewModel creates a new prompt model with the specified history limit.
func NewModel(historyLimit int) *Model {
	return &Model{
		buffer:  NewBuffer(),
		history: NewHistory(historyLimit),
	}
}

// Insert inserts a rune at the cursor position.
func (m *Model) Insert(r rune) {
	m.buffer.Insert(r)
}

// Backspace deletes the rune before the cursor.
func (m *Model) Backspace() bool {
	return m.buffer.Backspace()
}

// Delete deletes the rune at the cursor position.
func (m *Model) Delete() bool {
	return m.buffer.Delete()
}

// MoveLeft moves the cursor left by one position.
func (m *Model) MoveLeft() bool {
	return m.buffer.MoveLeft()
}

// MoveRight moves the cursor right by one position.
func (m *Model) MoveRight() bool {
	return m.buffer.MoveRight()
}

// MoveStart moves the cursor to the start of the buffer.
func (m *Model) MoveStart() {
	m.buffer.MoveStart()
}

// MoveEnd moves the cursor to the end of the buffer.
func (m *Model) MoveEnd() {
	m.buffer.MoveEnd()
}

// ClearLineLeft deletes from the start to the cursor (Ctrl-U).
func (m *Model) ClearLineLeft() {
	m.buffer.ClearLineLeft()
}

// ClearLineRight deletes from the cursor to the end (Ctrl-K).
func (m *Model) ClearLineRight() {
	m.buffer.ClearLineRight()
}

// DeleteWord deletes the previous word (Ctrl-W).
func (m *Model) DeleteWord() {
	m.buffer.DeleteWord()
}

// Clear resets the buffer to empty state.
func (m *Model) Clear() {
	m.buffer.Clear()
}

// PrevHistory loads the previous command from history into the buffer.
// Returns true if successful, false if already at oldest.
func (m *Model) PrevHistory() bool {
	currentDraft := m.buffer.Text()
	entry, ok := m.history.PrevHistory(currentDraft)
	if !ok {
		return false
	}
	m.buffer.SetText(entry)
	return true
}

// NextHistory loads the next command from history into the buffer.
// Returns true if successful, false if not navigating.
func (m *Model) NextHistory() bool {
	entry, ok := m.history.NextHistory()
	if !ok {
		return false
	}
	m.buffer.SetText(entry)
	return true
}

// Submit adds the current buffer to history, returns the text, and clears the buffer.
func (m *Model) Submit() string {
	text := m.buffer.Text()
	m.history.Submit(text)
	m.buffer.Clear()
	return text
}

// Text returns the current buffer text.
func (m *Model) Text() string {
	return m.buffer.Text()
}

// Cursor returns the current cursor position.
func (m *Model) Cursor() int {
	return m.buffer.Cursor()
}

// History returns the history manager (for testing/inspection).
func (m *Model) History() *History {
	return m.history
}
