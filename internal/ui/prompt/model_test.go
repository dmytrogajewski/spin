package prompt_test

import (
	"testing"

	"github.com/dmytrogajewski/spin/internal/ui/prompt"
)

func TestModel_BasicEditing(t *testing.T) {
	m := prompt.NewModel(10)

	// Type "hello"
	m.Insert('h')
	m.Insert('e')
	m.Insert('l')
	m.Insert('l')
	m.Insert('o')

	if got := m.Text(); got != "hello" {
		t.Errorf("Text() = %q, want 'hello'", got)
	}
	if got := m.Cursor(); got != 5 {
		t.Errorf("Cursor() = %d, want 5", got)
	}
}

func TestModel_Submit(t *testing.T) {
	m := prompt.NewModel(10)

	// Type and submit
	m.Insert('h')
	m.Insert('i')
	submitted := m.Submit()

	if submitted != "hi" {
		t.Errorf("Submit() = %q, want 'hi'", submitted)
	}

	// Buffer should be cleared
	if got := m.Text(); got != "" {
		t.Errorf("After Submit(), Text() = %q, want empty", got)
	}
	if got := m.Cursor(); got != 0 {
		t.Errorf("After Submit(), Cursor() = %d, want 0", got)
	}

	// History should have the entry
	entries := m.History().Entries()
	if len(entries) != 1 || entries[0] != "hi" {
		t.Errorf("After Submit(), History.Entries() = %v, want [hi]", entries)
	}
}

func TestModel_HistoryNavigation(t *testing.T) {
	m := prompt.NewModel(10)

	// Submit a few commands
	m.Insert('f')
	m.Insert('i')
	m.Insert('r')
	m.Insert('s')
	m.Insert('t')
	m.Submit()

	m.Insert('s')
	m.Insert('e')
	m.Insert('c')
	m.Insert('o')
	m.Insert('n')
	m.Insert('d')
	m.Submit()

	// Start typing new command
	m.Insert('t')
	m.Insert('h')

	// Navigate up (prev)
	ok := m.PrevHistory()
	if !ok {
		t.Errorf("PrevHistory() = false, want true")
	}
	if got := m.Text(); got != "second" {
		t.Errorf("After PrevHistory(), Text() = %q, want 'second'", got)
	}

	// Navigate up again
	ok = m.PrevHistory()
	if !ok {
		t.Errorf("Second PrevHistory() = false, want true")
	}
	if got := m.Text(); got != "first" {
		t.Errorf("After second PrevHistory(), Text() = %q, want 'first'", got)
	}

	// Navigate down (next)
	ok = m.NextHistory()
	if !ok {
		t.Errorf("NextHistory() = false, want true")
	}
	if got := m.Text(); got != "second" {
		t.Errorf("After NextHistory(), Text() = %q, want 'second'", got)
	}

	// Navigate down to draft
	ok = m.NextHistory()
	if !ok {
		t.Errorf("Second NextHistory() = false, want true")
	}
	if got := m.Text(); got != "th" {
		t.Errorf("After second NextHistory(), Text() = %q, want 'th' (draft)", got)
	}
}

func TestModel_EditHistoryEntry(t *testing.T) {
	m := prompt.NewModel(10)

	// Submit a command
	m.Insert('h')
	m.Insert('e')
	m.Insert('l')
	m.Insert('l')
	m.Insert('o')
	m.Submit()

	// Navigate to it
	m.PrevHistory()

	// Edit it
	m.MoveEnd()
	m.Insert(' ')
	m.Insert('w')
	m.Insert('o')
	m.Insert('r')
	m.Insert('l')
	m.Insert('d')

	// Submit edited version
	submitted := m.Submit()
	if submitted != "hello world" {
		t.Errorf("Submit() = %q, want 'hello world'", submitted)
	}

	// History should now have both
	entries := m.History().Entries()
	if len(entries) != 2 {
		t.Errorf("After edited submit, len(Entries()) = %d, want 2", len(entries))
	}
	if entries[0] != "hello world" || entries[1] != "hello" {
		t.Errorf("After edited submit, Entries() = %v, want [hello world, hello]", entries)
	}
}

func TestModel_AllBufferOperations(t *testing.T) {
	m := prompt.NewModel(10)

	// Test all buffer operations delegate correctly
	m.Insert('a')
	m.Backspace()
	m.Insert('h')
	m.Insert('e')
	m.Insert('l')
	m.Insert('l')
	m.Insert('o')

	m.MoveLeft()
	m.Delete()
	m.Insert('!')

	if got := m.Text(); got != "hell!" {
		t.Errorf("Text() = %q, want 'hell!'", got)
	}

	m.MoveStart()
	if got := m.Cursor(); got != 0 {
		t.Errorf("After MoveStart(), Cursor() = %d, want 0", got)
	}

	m.MoveEnd()
	if got := m.Cursor(); got != 5 {
		t.Errorf("After MoveEnd(), Cursor() = %d, want 5", got)
	}

	// Move to middle and clear left
	m.MoveLeft()
	m.MoveLeft()
	m.ClearLineLeft()
	if got := m.Text(); got != "l!" {
		t.Errorf("After ClearLineLeft(), Text() = %q, want 'l!'", got)
	}

	m.Clear()
	if got := m.Text(); got != "" {
		t.Errorf("After Clear(), Text() = %q, want empty", got)
	}
}

func TestModel_HistoryLimit(t *testing.T) {
	m := prompt.NewModel(3)

	// Submit 4 entries (exceeds limit)
	m.Insert('1')
	m.Submit()

	m.Insert('2')
	m.Submit()

	m.Insert('3')
	m.Submit()

	m.Insert('4')
	m.Submit()

	// Should only have 3 (newest)
	entries := m.History().Entries()
	if len(entries) != 3 {
		t.Errorf("len(Entries()) = %d, want 3 (limit)", len(entries))
	}
	if entries[0] != "4" || entries[1] != "3" || entries[2] != "2" {
		t.Errorf("Entries() = %v, want [4 3 2]", entries)
	}
}

func TestModel_UncoveredOperations(t *testing.T) {
	m := prompt.NewModel(10)

	// Test MoveRight
	m.Insert('a')
	m.Insert('b')
	m.MoveLeft()
	m.MoveLeft()
	m.MoveRight()
	if m.Cursor() != 1 {
		t.Errorf("After MoveRight, Cursor() = %d, want 1", m.Cursor())
	}

	// Test ClearLineRight
	m.ClearLineRight()
	if m.Text() != "a" {
		t.Errorf("After ClearLineRight, Text() = %q, want 'a'", m.Text())
	}

	// Test DeleteWord
	m.Insert(' ')
	m.Insert('w')
	m.Insert('o')
	m.Insert('r')
	m.Insert('d')
	m.DeleteWord()
	if m.Text() != "a " {
		t.Errorf("After DeleteWord, Text() = %q, want 'a '", m.Text())
	}
}
