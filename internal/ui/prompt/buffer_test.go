package prompt_test

import (
	"testing"

	"github.com/dmytrogajewski/spin/internal/ui/prompt"
)

func TestBuffer_Insert(t *testing.T) {
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
			b := prompt.NewBuffer()
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

func TestBuffer_Backspace(t *testing.T) {
	tests := []struct {
		name       string
		initial    string
		cursor     int
		wantText   string
		wantCursor int
		wantOk     bool
	}{
		{
			name:       "backspace at start (no-op)",
			initial:    "hello",
			cursor:     0,
			wantText:   "hello",
			wantCursor: 0,
			wantOk:     false,
		},
		{
			name:       "backspace at end",
			initial:    "hello",
			cursor:     5,
			wantText:   "hell",
			wantCursor: 4,
			wantOk:     true,
		},
		{
			name:       "backspace in middle",
			initial:    "hello",
			cursor:     3,
			wantText:   "helo",
			wantCursor: 2,
			wantOk:     true,
		},
		{
			name:       "backspace on empty buffer",
			initial:    "",
			cursor:     0,
			wantText:   "",
			wantCursor: 0,
			wantOk:     false,
		},
		{
			name:       "backspace emoji",
			initial:    "hello😀",
			cursor:     6,
			wantText:   "hello",
			wantCursor: 5,
			wantOk:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := prompt.NewBuffer()
			b.SetText(tt.initial)
			b.SetCursor(tt.cursor)

			ok := b.Backspace()

			if ok != tt.wantOk {
				t.Errorf("Backspace() = %v, want %v", ok, tt.wantOk)
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

func TestBuffer_Delete(t *testing.T) {
	tests := []struct {
		name       string
		initial    string
		cursor     int
		wantText   string
		wantCursor int
		wantOk     bool
	}{
		{
			name:       "delete at end (no-op)",
			initial:    "hello",
			cursor:     5,
			wantText:   "hello",
			wantCursor: 5,
			wantOk:     false,
		},
		{
			name:       "delete at start",
			initial:    "hello",
			cursor:     0,
			wantText:   "ello",
			wantCursor: 0,
			wantOk:     true,
		},
		{
			name:       "delete in middle",
			initial:    "hello",
			cursor:     2,
			wantText:   "helo",
			wantCursor: 2,
			wantOk:     true,
		},
		{
			name:       "delete on empty buffer",
			initial:    "",
			cursor:     0,
			wantText:   "",
			wantCursor: 0,
			wantOk:     false,
		},
		{
			name:       "delete emoji",
			initial:    "hello😀world",
			cursor:     5,
			wantText:   "helloworld",
			wantCursor: 5,
			wantOk:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := prompt.NewBuffer()
			b.SetText(tt.initial)
			b.SetCursor(tt.cursor)

			ok := b.Delete()

			if ok != tt.wantOk {
				t.Errorf("Delete() = %v, want %v", ok, tt.wantOk)
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

func TestBuffer_MoveLeft(t *testing.T) {
	tests := []struct {
		name       string
		initial    string
		cursor     int
		wantCursor int
		wantOk     bool
	}{
		{
			name:       "move left at start (no-op)",
			initial:    "hello",
			cursor:     0,
			wantCursor: 0,
			wantOk:     false,
		},
		{
			name:       "move left at end",
			initial:    "hello",
			cursor:     5,
			wantCursor: 4,
			wantOk:     true,
		},
		{
			name:       "move left in middle",
			initial:    "hello",
			cursor:     3,
			wantCursor: 2,
			wantOk:     true,
		},
		{
			name:       "move left through emoji",
			initial:    "hello😀",
			cursor:     6,
			wantCursor: 5,
			wantOk:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := prompt.NewBuffer()
			b.SetText(tt.initial)
			b.SetCursor(tt.cursor)

			ok := b.MoveLeft()

			if ok != tt.wantOk {
				t.Errorf("MoveLeft() = %v, want %v", ok, tt.wantOk)
			}
			if got := b.Cursor(); got != tt.wantCursor {
				t.Errorf("Cursor() = %d, want %d", got, tt.wantCursor)
			}
		})
	}
}

func TestBuffer_MoveRight(t *testing.T) {
	tests := []struct {
		name       string
		initial    string
		cursor     int
		wantCursor int
		wantOk     bool
	}{
		{
			name:       "move right at end (no-op)",
			initial:    "hello",
			cursor:     5,
			wantCursor: 5,
			wantOk:     false,
		},
		{
			name:       "move right at start",
			initial:    "hello",
			cursor:     0,
			wantCursor: 1,
			wantOk:     true,
		},
		{
			name:       "move right in middle",
			initial:    "hello",
			cursor:     2,
			wantCursor: 3,
			wantOk:     true,
		},
		{
			name:       "move right through emoji",
			initial:    "hello😀",
			cursor:     5,
			wantCursor: 6,
			wantOk:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := prompt.NewBuffer()
			b.SetText(tt.initial)
			b.SetCursor(tt.cursor)

			ok := b.MoveRight()

			if ok != tt.wantOk {
				t.Errorf("MoveRight() = %v, want %v", ok, tt.wantOk)
			}
			if got := b.Cursor(); got != tt.wantCursor {
				t.Errorf("Cursor() = %d, want %d", got, tt.wantCursor)
			}
		})
	}
}

func TestBuffer_MoveStartEnd(t *testing.T) {
	b := prompt.NewBuffer()
	b.SetText("hello world")
	b.SetCursor(5)

	// Move to start
	b.MoveStart()
	if got := b.Cursor(); got != 0 {
		t.Errorf("After MoveStart(), Cursor() = %d, want 0", got)
	}

	// Move to end
	b.MoveEnd()
	if got := b.Cursor(); got != 11 {
		t.Errorf("After MoveEnd(), Cursor() = %d, want 11", got)
	}
}

func TestBuffer_ClearLineLeft(t *testing.T) {
	tests := []struct {
		name       string
		initial    string
		cursor     int
		wantText   string
		wantCursor int
	}{
		{
			name:       "clear at start (no-op)",
			initial:    "hello world",
			cursor:     0,
			wantText:   "hello world",
			wantCursor: 0,
		},
		{
			name:       "clear at end",
			initial:    "hello world",
			cursor:     11,
			wantText:   "",
			wantCursor: 0,
		},
		{
			name:       "clear in middle",
			initial:    "hello world",
			cursor:     5,
			wantText:   " world",
			wantCursor: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := prompt.NewBuffer()
			b.SetText(tt.initial)
			b.SetCursor(tt.cursor)

			b.ClearLineLeft()

			if got := b.Text(); got != tt.wantText {
				t.Errorf("Text() = %q, want %q", got, tt.wantText)
			}
			if got := b.Cursor(); got != tt.wantCursor {
				t.Errorf("Cursor() = %d, want %d", got, tt.wantCursor)
			}
		})
	}
}

func TestBuffer_ClearLineRight(t *testing.T) {
	tests := []struct {
		name       string
		initial    string
		cursor     int
		wantText   string
		wantCursor int
	}{
		{
			name:       "clear at end (no-op)",
			initial:    "hello world",
			cursor:     11,
			wantText:   "hello world",
			wantCursor: 11,
		},
		{
			name:       "clear at start",
			initial:    "hello world",
			cursor:     0,
			wantText:   "",
			wantCursor: 0,
		},
		{
			name:       "clear in middle",
			initial:    "hello world",
			cursor:     5,
			wantText:   "hello",
			wantCursor: 5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := prompt.NewBuffer()
			b.SetText(tt.initial)
			b.SetCursor(tt.cursor)

			b.ClearLineRight()

			if got := b.Text(); got != tt.wantText {
				t.Errorf("Text() = %q, want %q", got, tt.wantText)
			}
			if got := b.Cursor(); got != tt.wantCursor {
				t.Errorf("Cursor() = %d, want %d", got, tt.wantCursor)
			}
		})
	}
}

func TestBuffer_DeleteWord(t *testing.T) {
	tests := []struct {
		name       string
		initial    string
		cursor     int
		wantText   string
		wantCursor int
	}{
		{
			name:       "delete word at start (no-op)",
			initial:    "hello world",
			cursor:     0,
			wantText:   "hello world",
			wantCursor: 0,
		},
		{
			name:       "delete word at end of first word",
			initial:    "hello world",
			cursor:     5,
			wantText:   " world",
			wantCursor: 0,
		},
		{
			name:       "delete word in middle of second word",
			initial:    "hello world",
			cursor:     9,
			wantText:   "hello ld",
			wantCursor: 6,
		},
		{
			name:       "delete word after space",
			initial:    "hello world",
			cursor:     6,
			wantText:   "world",
			wantCursor: 0,
		},
		{
			name:       "delete word with punctuation",
			initial:    "hello-world",
			cursor:     11,
			wantText:   "hello-",
			wantCursor: 6,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := prompt.NewBuffer()
			b.SetText(tt.initial)
			b.SetCursor(tt.cursor)

			b.DeleteWord()

			if got := b.Text(); got != tt.wantText {
				t.Errorf("Text() = %q, want %q", got, tt.wantText)
			}
			if got := b.Cursor(); got != tt.wantCursor {
				t.Errorf("Cursor() = %d, want %d", got, tt.wantCursor)
			}
		})
	}
}

func TestBuffer_Clear(t *testing.T) {
	b := prompt.NewBuffer()
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
			b := prompt.NewBuffer()
			b.SetText(tt.text)

			if got := b.Len(); got != tt.wantLen {
				t.Errorf("Len() = %d, want %d", got, tt.wantLen)
			}
		})
	}
}

func TestBuffer_SetCursor(t *testing.T) {
	b := prompt.NewBuffer()
	b.SetText("hello")

	// Set to negative (should clamp to 0)
	b.SetCursor(-1)
	if got := b.Cursor(); got != 0 {
		t.Errorf("SetCursor(-1) resulted in Cursor() = %d, want 0", got)
	}

	// Set beyond length (should clamp to len)
	b.SetCursor(100)
	if got := b.Cursor(); got != 5 {
		t.Errorf("SetCursor(100) resulted in Cursor() = %d, want 5", got)
	}

	// Set to valid position
	b.SetCursor(3)
	if got := b.Cursor(); got != 3 {
		t.Errorf("SetCursor(3) resulted in Cursor() = %d, want 3", got)
	}
}
