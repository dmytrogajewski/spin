package textbuffer_test

import (
	"testing"

	"github.com/dmytrogajewski/spin/pkg/ui/textbuffer"
)

func TestBuffer_Insert(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		initial    string
		cursor     int
		insert     rune
		wantText   string
		wantCursor int
	}{
		{
			name:       "insert into empty buffer",
			initial:    "",
			cursor:     0,
			insert:     'a',
			wantText:   "a",
			wantCursor: 1,
		},
		{
			name:       "insert at start",
			initial:    "world",
			cursor:     0,
			insert:     'h',
			wantText:   "hworld",
			wantCursor: 1,
		},
		{
			name:       "insert at end",
			initial:    "hello",
			cursor:     5,
			insert:     '!',
			wantText:   "hello!",
			wantCursor: 6,
		},
		{
			name:       "insert in middle",
			initial:    "helo",
			cursor:     3,
			insert:     'l',
			wantText:   "hello",
			wantCursor: 4,
		},
		{
			name:       "insert emoji",
			initial:    "hello",
			cursor:     5,
			insert:     '😀',
			wantText:   "hello😀",
			wantCursor: 6,
		},
		{
			name:       "insert CJK character",
			initial:    "hello",
			cursor:     5,
			insert:     '世',
			wantText:   "hello世",
			wantCursor: 6,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			b := textbuffer.NewBuffer()
			b.SetText(tt.initial)
			b.SetCursor(tt.cursor)

			b.Insert(tt.insert)

			if got := b.Text(); got != tt.wantText {
				t.Errorf("Text() = %q, want %q", got, tt.wantText)
			}

			if got := b.Cursor(); got != tt.wantCursor {
				t.Errorf("Cursor() = %d, want %d", got, tt.wantCursor)
			}
		})
	}
}

// bufferEditCase is a test case for buffer editing operations that modify text.
type bufferEditCase struct {
	name       string
	initial    string
	cursor     int
	wantText   string
	wantCursor int
	wantOk     bool
}

// runBufferEditTests runs a set of buffer editing test cases against the given operation.
func runBufferEditTests(t *testing.T, cases []bufferEditCase, opName string, op func(*textbuffer.Buffer) bool) {
	t.Helper()

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			b := textbuffer.NewBuffer()
			b.SetText(tt.initial)
			b.SetCursor(tt.cursor)

			ok := op(b)

			if ok != tt.wantOk {
				t.Errorf("%s() = %v, want %v", opName, ok, tt.wantOk)
			}

			if got := b.Text(); got != tt.wantText {
				t.Errorf("Text() = %q, want %q", got, tt.wantText)
			}

			if got := b.Cursor(); got != tt.wantCursor {
				t.Errorf("Cursor() = %d, want %d", got, tt.wantCursor)
			}
		})
	}
}

func TestBuffer_Backspace(t *testing.T) {
	t.Parallel()

	runBufferEditTests(t, []bufferEditCase{
		{"backspace at start (no-op)", "hello", 0, "hello", 0, false},
		{"backspace at end", "hello", 5, "hell", 4, true},
		{"backspace in middle", "hello", 3, "helo", 2, true},
		{"backspace on empty buffer", "", 0, "", 0, false},
		{"backspace emoji", "hello\U0001f600", 6, "hello", 5, true},
	}, "Backspace", (*textbuffer.Buffer).Backspace)
}

func TestBuffer_Delete(t *testing.T) {
	t.Parallel()

	runBufferEditTests(t, []bufferEditCase{
		{"delete at end (no-op)", "hello", 5, "hello", 5, false},
		{"delete at start", "hello", 0, "ello", 0, true},
		{"delete in middle", "hello", 2, "helo", 2, true},
		{"delete on empty buffer", "", 0, "", 0, false},
		{"delete emoji", "hello\U0001f600world", 5, "helloworld", 5, true},
	}, "Delete", (*textbuffer.Buffer).Delete)
}

// bufferMoveCase is a test case for buffer cursor movement operations.
type bufferMoveCase struct {
	name       string
	initial    string
	cursor     int
	wantCursor int
	wantOk     bool
}

// runBufferMoveTests runs a set of cursor movement test cases against the given operation.
func runBufferMoveTests(t *testing.T, cases []bufferMoveCase, opName string, op func(*textbuffer.Buffer) bool) {
	t.Helper()

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			b := textbuffer.NewBuffer()
			b.SetText(tt.initial)
			b.SetCursor(tt.cursor)

			ok := op(b)

			if ok != tt.wantOk {
				t.Errorf("%s() = %v, want %v", opName, ok, tt.wantOk)
			}

			if got := b.Cursor(); got != tt.wantCursor {
				t.Errorf("Cursor() = %d, want %d", got, tt.wantCursor)
			}
		})
	}
}

func TestBuffer_MoveLeft(t *testing.T) {
	t.Parallel()

	runBufferMoveTests(t, []bufferMoveCase{
		{"move left at start (no-op)", "hello", 0, 0, false},
		{"move left at end", "hello", 5, 4, true},
		{"move left in middle", "hello", 3, 2, true},
		{"move left through emoji", "hello\U0001f600", 6, 5, true},
	}, "MoveLeft", (*textbuffer.Buffer).MoveLeft)
}

func TestBuffer_MoveRight(t *testing.T) {
	t.Parallel()

	runBufferMoveTests(t, []bufferMoveCase{
		{"move right at end (no-op)", "hello", 5, 5, false},
		{"move right at start", "hello", 0, 1, true},
		{"move right in middle", "hello", 2, 3, true},
		{"move right through emoji", "hello\U0001f600", 5, 6, true},
	}, "MoveRight", (*textbuffer.Buffer).MoveRight)
}

func TestBuffer_MoveStartEnd(t *testing.T) {
	t.Parallel()

	b := textbuffer.NewBuffer()
	b.SetText("hello world")
	b.SetCursor(5)

	// Move to start.
	b.MoveStart()

	if got := b.Cursor(); got != 0 {
		t.Errorf("After MoveStart(), Cursor() = %d, want 0", got)
	}

	// Move to end.
	b.MoveEnd()

	if got := b.Cursor(); got != 11 {
		t.Errorf("After MoveEnd(), Cursor() = %d, want 11", got)
	}
}

// bufferClearCase is a test case for buffer clear-line operations.
type bufferClearCase struct {
	name       string
	initial    string
	cursor     int
	wantText   string
	wantCursor int
}

// runBufferClearTests runs a set of clear-line test cases against the given operation.
func runBufferClearTests(t *testing.T, cases []bufferClearCase, opName string, op func(*textbuffer.Buffer)) {
	t.Helper()

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			b := textbuffer.NewBuffer()
			b.SetText(tt.initial)
			b.SetCursor(tt.cursor)

			op(b)

			if got := b.Text(); got != tt.wantText {
				t.Errorf("After %s: Text() = %q, want %q", opName, got, tt.wantText)
			}

			if got := b.Cursor(); got != tt.wantCursor {
				t.Errorf("After %s: Cursor() = %d, want %d", opName, got, tt.wantCursor)
			}
		})
	}
}

func TestBuffer_ClearLineLeft(t *testing.T) {
	t.Parallel()

	runBufferClearTests(t, []bufferClearCase{
		{"clear at start (no-op)", "hello world", 0, "hello world", 0},
		{"clear at end", "hello world", 11, "", 0},
		{"clear in middle", "hello world", 5, " world", 0},
	}, "ClearLineLeft", (*textbuffer.Buffer).ClearLineLeft)
}

func TestBuffer_ClearLineRight(t *testing.T) {
	t.Parallel()

	runBufferClearTests(t, []bufferClearCase{
		{"clear at end (no-op)", "hello world", 11, "hello world", 11},
		{"clear at start", "hello world", 0, "", 0},
		{"clear in middle", "hello world", 5, "hello", 5},
	}, "ClearLineRight", (*textbuffer.Buffer).ClearLineRight)
}

func TestBuffer_DeleteWord(t *testing.T) {
	t.Parallel()

	runBufferClearTests(t, []bufferClearCase{
		{"delete word at start (no-op)", "hello world", 0, "hello world", 0},
		{"delete word at end of first word", "hello world", 5, " world", 0},
		{"delete word in middle of second word", "hello world", 9, "hello ld", 6},
		{"delete word after space", "hello world", 6, "world", 0},
		{"delete word with punctuation", "hello-world", 11, "hello-", 6},
	}, "DeleteWord", (*textbuffer.Buffer).DeleteWord)
}

func TestBuffer_Clear(t *testing.T) {
	t.Parallel()

	b := textbuffer.NewBuffer()
	b.SetText("hello world")
	b.SetCursor(5)

	b.Clear()

	if got := b.Text(); got != "" {
		t.Errorf("After Clear(), Text() = %q, want empty", got)
	}

	if got := b.Cursor(); got != 0 {
		t.Errorf("After Clear(), Cursor() = %d, want 0", got)
	}
}

func TestBuffer_Len(t *testing.T) {
	t.Parallel()

	tests := []struct {
		text    string
		wantLen int
	}{
		{"", 0},
		{"hello", 5},
		{"hello😀", 6},
		{"世界", 2},
	}

	for _, tt := range tests {
		t.Run(tt.text, func(t *testing.T) {
			t.Parallel()

			b := textbuffer.NewBuffer()
			b.SetText(tt.text)

			if got := b.Len(); got != tt.wantLen {
				t.Errorf("Len() = %d, want %d", got, tt.wantLen)
			}
		})
	}
}

func TestBuffer_SetCursor(t *testing.T) {
	t.Parallel()

	b := textbuffer.NewBuffer()
	b.SetText("hello")

	// Set to negative (should clamp to 0).
	b.SetCursor(-1)

	if got := b.Cursor(); got != 0 {
		t.Errorf("SetCursor(-1) resulted in Cursor() = %d, want 0", got)
	}

	// Set beyond length (should clamp to len).
	b.SetCursor(100)

	if got := b.Cursor(); got != 5 {
		t.Errorf("SetCursor(100) resulted in Cursor() = %d, want 5", got)
	}

	// Set to valid position.
	b.SetCursor(3)

	if got := b.Cursor(); got != 3 {
		t.Errorf("SetCursor(3) resulted in Cursor() = %d, want 3", got)
	}
}

func TestBuffer_DeleteWord_Punctuation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		initial    string
		cursor     int
		wantText   string
		wantCursor int
	}{
		{
			name:       "delete punctuation word",
			initial:    "hello!!!",
			cursor:     8, // After all punctuation.
			wantText:   "hello",
			wantCursor: 5,
		},
		{
			name:       "delete mixed punctuation",
			initial:    "test@#$%",
			cursor:     7,       // After all punctuation.
			wantText:   "test%", // Only the last punctuation remains.
			wantCursor: 4,
		},
		{
			name:       "delete single punctuation",
			initial:    "word!",
			cursor:     5, // After '!'.
			wantText:   "word",
			wantCursor: 4,
		},
		{
			name:       "delete punctuation with spaces",
			initial:    "word !@#",
			cursor:     7,        // After '#'.
			wantText:   "word #", // Only the last punctuation remains.
			wantCursor: 5,
		},
		{
			name:       "delete only punctuation",
			initial:    "!!!",
			cursor:     3, // After all punctuation.
			wantText:   "",
			wantCursor: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			b := textbuffer.NewBuffer()
			for _, r := range tt.initial {
				b.Insert(r)
			}

			b.SetCursor(tt.cursor)

			b.DeleteWord()

			if got := b.Text(); got != tt.wantText {
				t.Errorf("DeleteWord() text = %q, want %q", got, tt.wantText)
			}

			if got := b.Cursor(); got != tt.wantCursor {
				t.Errorf("DeleteWord() cursor = %d, want %d", got, tt.wantCursor)
			}
		})
	}
}

func TestBuffer_GraphemeClusters_CombiningCharacters(t *testing.T) {
	t.Parallel()

	// "e" + combining acute accent (U+0301) forms a single grapheme "é".
	b := textbuffer.NewBuffer()
	b.Insert('e')
	b.Insert('\u0301') // Combining acute accent.

	if got := b.Len(); got != 1 {
		t.Errorf("Len() = %d, want 1 (combining char should merge)", got)
	}

	if got := b.Cursor(); got != 1 {
		t.Errorf("Cursor() = %d, want 1", got)
	}

	if got := b.Text(); got != "e\u0301" {
		t.Errorf("Text() = %q, want %q", got, "e\u0301")
	}

	// Backspace should delete entire grapheme cluster.
	b.Backspace()

	if got := b.Len(); got != 0 {
		t.Errorf("After Backspace, Len() = %d, want 0", got)
	}

	if got := b.Text(); got != "" {
		t.Errorf("After Backspace, Text() = %q, want empty", got)
	}
}

func TestBuffer_GraphemeClusters_SetText(t *testing.T) {
	t.Parallel()

	// Family emoji is a single grapheme cluster but multiple runes.
	b := textbuffer.NewBuffer()
	b.SetText("a\U0001F468\u200D\U0001F469\u200D\U0001F467b") // a + family emoji + b.

	if got := b.Len(); got != 3 {
		t.Errorf("Len() = %d, want 3 (a, family emoji, b)", got)
	}

	// Backspace should delete 'b' as a single grapheme.
	b.Backspace()

	if got := b.Len(); got != 2 {
		t.Errorf("After first Backspace, Len() = %d, want 2", got)
	}

	// Backspace should delete entire family emoji as one grapheme.
	b.Backspace()

	if got := b.Len(); got != 1 {
		t.Errorf("After second Backspace, Len() = %d, want 1", got)
	}

	if got := b.Text(); got != "a" {
		t.Errorf("After two Backspaces, Text() = %q, want %q", got, "a")
	}
}
