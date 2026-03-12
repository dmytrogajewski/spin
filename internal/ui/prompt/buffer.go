package prompt

import (
	"unicode"
)

// Buffer represents the editable text buffer with cursor position.
// It stores text as a slice of runes and maintains a cursor index.
// The cursor position is always valid: 0 <= cursor <= len(runes).
type Buffer struct {
	runes  []rune // text content.
	cursor int    // cursor position (0-based).
}

// NewBuffer creates a new empty buffer.
func NewBuffer() *Buffer {
	return &Buffer{
		runes:  []rune{},
		cursor: 0,
	}
}

// Insert inserts a rune at the cursor position and advances the cursor.
func (b *Buffer) Insert(r rune) {
	// Insert at cursor position.
	b.runes = append(b.runes[:b.cursor], append([]rune{r}, b.runes[b.cursor:]...)...)
	b.cursor++
}

// Backspace deletes the rune before the cursor if possible.
// Returns true if a character was deleted, false otherwise.
func (b *Buffer) Backspace() bool {
	if b.cursor == 0 {
		return false
	}

	// Delete character before cursor.
	b.runes = append(b.runes[:b.cursor-1], b.runes[b.cursor:]...)
	b.cursor--

	return true
}

// Delete deletes the rune at the cursor position if possible.
// Returns true if a character was deleted, false otherwise.
func (b *Buffer) Delete() bool {
	if b.cursor >= len(b.runes) {
		return false
	}

	// Delete character at cursor.
	b.runes = append(b.runes[:b.cursor], b.runes[b.cursor+1:]...)

	return true
}

// MoveLeft moves the cursor left by one position.
// Returns true if the cursor moved, false if already at start.
func (b *Buffer) MoveLeft() bool {
	if b.cursor == 0 {
		return false
	}

	b.cursor--

	return true
}

// MoveRight moves the cursor right by one position.
// Returns true if the cursor moved, false if already at end.
func (b *Buffer) MoveRight() bool {
	if b.cursor >= len(b.runes) {
		return false
	}

	b.cursor++

	return true
}

// MoveStart moves the cursor to the start of the buffer.
func (b *Buffer) MoveStart() {
	b.cursor = 0
}

// MoveEnd moves the cursor to the end of the buffer.
func (b *Buffer) MoveEnd() {
	b.cursor = len(b.runes)
}

// ClearLineLeft deletes from the start of the line to the cursor (Ctrl-U).
// Moves cursor to position 0.
func (b *Buffer) ClearLineLeft() {
	if b.cursor == 0 {
		return
	}

	b.runes = b.runes[b.cursor:]
	b.cursor = 0
}

// ClearLineRight deletes from the cursor to the end of the line (Ctrl-K).
// Cursor position remains unchanged.
func (b *Buffer) ClearLineRight() {
	if b.cursor >= len(b.runes) {
		return
	}

	b.runes = b.runes[:b.cursor]
}

// DeleteWord deletes the previous word (Ctrl-W).
// Uses Unicode word boundaries (space, punctuation).
func (b *Buffer) DeleteWord() {
	if b.cursor == 0 {
		return
	}

	wordStart := b.findWordStart()
	b.deleteFromPosition(wordStart)
}

// findWordStart finds the start position of the word before the cursor.
func (b *Buffer) findWordStart() int {
	pos := b.cursor - 1
	pos = b.skipTrailingSpaces(pos)

	if pos < 0 {
		return pos
	}

	return b.skipWordCharacters(pos)
}

// skipTrailingSpaces skips trailing spaces before the word.
func (b *Buffer) skipTrailingSpaces(pos int) int {
	for pos >= 0 && unicode.IsSpace(b.runes[pos]) {
		pos--
	}

	return pos
}

// skipWordCharacters skips characters of the current word type.
func (b *Buffer) skipWordCharacters(pos int) int {
	isAlnum := unicode.IsLetter(b.runes[pos]) || unicode.IsDigit(b.runes[pos])

	if isAlnum {
		return b.skipAlphanumericCharacters(pos)
	}

	return b.skipPunctuationCharacters(pos)
}

// skipAlphanumericCharacters skips alphanumeric characters.
func (b *Buffer) skipAlphanumericCharacters(pos int) int {
	for pos >= 0 && (unicode.IsLetter(b.runes[pos]) || unicode.IsDigit(b.runes[pos])) {
		pos--
	}

	return pos
}

// skipPunctuationCharacters skips punctuation/symbol characters.
func (b *Buffer) skipPunctuationCharacters(pos int) int {
	for pos >= 0 && !unicode.IsSpace(b.runes[pos]) && !unicode.IsLetter(b.runes[pos]) && !unicode.IsDigit(b.runes[pos]) {
		pos--
	}

	return pos
}

// deleteFromPosition deletes characters from the given position to the cursor.
func (b *Buffer) deleteFromPosition(deleteFrom int) {
	deleteFrom++ // Adjust for 0-based indexing.
	b.runes = append(b.runes[:deleteFrom], b.runes[b.cursor:]...)
	b.cursor = deleteFrom
}

// Clear resets the buffer to empty state.
func (b *Buffer) Clear() {
	b.runes = []rune{}
	b.cursor = 0
}

// Text returns the current buffer text as a string.
func (b *Buffer) Text() string {
	return string(b.runes)
}

// Cursor returns the current cursor position.
func (b *Buffer) Cursor() int {
	return b.cursor
}

// Len returns the length of the buffer in runes.
func (b *Buffer) Len() int {
	return len(b.runes)
}

// SetText sets the buffer text and moves cursor to the end.
func (b *Buffer) SetText(s string) {
	b.runes = []rune(s)
	b.cursor = len(b.runes)
}

// SetCursor sets the cursor position.
// The position is clamped to valid range [0, len(runes)].
func (b *Buffer) SetCursor(pos int) {
	if pos < 0 {
		pos = 0
	}

	if pos > len(b.runes) {
		pos = len(b.runes)
	}

	b.cursor = pos
}
