// Package textbuffer provides a text editing buffer with cursor management.
package textbuffer

import (
	"strings"
	"unicode"

	"github.com/dmytrogajewski/spin/pkg/ui/textwidth"
)

// Buffer represents the editable text buffer with cursor position.
// It stores text as a slice of grapheme clusters and maintains a cursor index.
// The cursor position is always valid: 0 <= cursor <= len(graphemes).
type Buffer struct {
	graphemes []string // text content as grapheme clusters.
	cursor    int      // cursor position (0-based, in grapheme clusters).
}

// NewBuffer creates a new empty buffer.
func NewBuffer() *Buffer {
	return &Buffer{
		graphemes: []string{},
		cursor:    0,
	}
}

// firstRune returns the first rune of a grapheme cluster.
func firstRune(grapheme string) rune {
	for _, r := range grapheme {
		return r
	}

	return 0
}

// Insert inserts a rune at the cursor position and advances the cursor.
// If the rune is a combining character, it merges with the preceding grapheme.
func (b *Buffer) Insert(r rune) {
	s := string(r)

	// Try to combine with previous grapheme cluster (for combining characters).
	if b.cursor > 0 {
		combined := b.graphemes[b.cursor-1] + s
		clusters := textwidth.ExtractGraphemes(combined)

		if len(clusters) == 1 {
			// Rune combines with previous grapheme (e.g., combining accent).
			b.graphemes[b.cursor-1] = clusters[0]

			return
		}
	}

	// Insert as new grapheme cluster.
	b.graphemes = append(b.graphemes[:b.cursor], append([]string{s}, b.graphemes[b.cursor:]...)...)
	b.cursor++
}

// Backspace deletes the grapheme cluster before the cursor if possible.
// Returns true if a cluster was deleted, false otherwise.
func (b *Buffer) Backspace() bool {
	if b.cursor == 0 {
		return false
	}

	// Delete grapheme cluster before cursor.
	b.graphemes = append(b.graphemes[:b.cursor-1], b.graphemes[b.cursor:]...)
	b.cursor--

	return true
}

// Delete deletes the grapheme cluster at the cursor position if possible.
// Returns true if a cluster was deleted, false otherwise.
func (b *Buffer) Delete() bool {
	if b.cursor >= len(b.graphemes) {
		return false
	}

	// Delete grapheme cluster at cursor.
	b.graphemes = append(b.graphemes[:b.cursor], b.graphemes[b.cursor+1:]...)

	return true
}

// MoveLeft moves the cursor left by one grapheme cluster.
// Returns true if the cursor moved, false if already at start.
func (b *Buffer) MoveLeft() bool {
	if b.cursor == 0 {
		return false
	}

	b.cursor--

	return true
}

// MoveRight moves the cursor right by one grapheme cluster.
// Returns true if the cursor moved, false if already at end.
func (b *Buffer) MoveRight() bool {
	if b.cursor >= len(b.graphemes) {
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
	b.cursor = len(b.graphemes)
}

// ClearLineLeft deletes from the start of the line to the cursor (Ctrl-U).
// Moves cursor to position 0.
func (b *Buffer) ClearLineLeft() {
	if b.cursor == 0 {
		return
	}

	b.graphemes = b.graphemes[b.cursor:]
	b.cursor = 0
}

// ClearLineRight deletes from the cursor to the end of the line (Ctrl-K).
// Cursor position remains unchanged.
func (b *Buffer) ClearLineRight() {
	if b.cursor >= len(b.graphemes) {
		return
	}

	b.graphemes = b.graphemes[:b.cursor]
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
	for pos >= 0 && unicode.IsSpace(firstRune(b.graphemes[pos])) {
		pos--
	}

	return pos
}

// skipWordCharacters skips characters of the current word type.
func (b *Buffer) skipWordCharacters(pos int) int {
	r := firstRune(b.graphemes[pos])
	isAlnum := unicode.IsLetter(r) || unicode.IsDigit(r)

	if isAlnum {
		return b.skipAlphanumericCharacters(pos)
	}

	return b.skipPunctuationCharacters(pos)
}

// skipAlphanumericCharacters skips alphanumeric characters.
func (b *Buffer) skipAlphanumericCharacters(pos int) int {
	for pos >= 0 {
		r := firstRune(b.graphemes[pos])
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) {
			break
		}

		pos--
	}

	return pos
}

// skipPunctuationCharacters skips punctuation/symbol characters.
func (b *Buffer) skipPunctuationCharacters(pos int) int {
	for pos >= 0 {
		r := firstRune(b.graphemes[pos])
		if unicode.IsSpace(r) || unicode.IsLetter(r) || unicode.IsDigit(r) {
			break
		}

		pos--
	}

	return pos
}

// deleteFromPosition deletes characters from the given position to the cursor.
func (b *Buffer) deleteFromPosition(deleteFrom int) {
	deleteFrom++ // Adjust for 0-based indexing.
	b.graphemes = append(b.graphemes[:deleteFrom], b.graphemes[b.cursor:]...)
	b.cursor = deleteFrom
}

// Clear resets the buffer to empty state.
func (b *Buffer) Clear() {
	b.graphemes = []string{}
	b.cursor = 0
}

// Text returns the current buffer text as a string.
func (b *Buffer) Text() string {
	return strings.Join(b.graphemes, "")
}

// Cursor returns the current cursor position (in grapheme clusters).
func (b *Buffer) Cursor() int {
	return b.cursor
}

// Len returns the length of the buffer in grapheme clusters.
func (b *Buffer) Len() int {
	return len(b.graphemes)
}

// SetText sets the buffer text and moves cursor to the end.
func (b *Buffer) SetText(s string) {
	b.graphemes = textwidth.ExtractGraphemes(s)
	b.cursor = len(b.graphemes)
}

// SetCursor sets the cursor position.
// The position is clamped to valid range [0, len(graphemes)].
func (b *Buffer) SetCursor(pos int) {
	if pos < 0 {
		pos = 0
	}

	if pos > len(b.graphemes) {
		pos = len(b.graphemes)
	}

	b.cursor = pos
}
