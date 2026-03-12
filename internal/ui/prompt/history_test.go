package prompt_test

import (
	"testing"

	"github.com/dmytrogajewski/spin/internal/ui/prompt"
)

func TestHistory_Submit(t *testing.T) {
	h := prompt.NewHistory(3)

	// Submit first entry.
	h.Submit("first")

	entries := h.Entries()
	if len(entries) != 1 || entries[0] != "first" {
		t.Errorf("After first submit, Entries() = %v, want [first]", entries)
	}

	// Submit second entry.
	h.Submit("second")

	entries = h.Entries()
	if len(entries) != 2 || entries[0] != "second" || entries[1] != "first" {
		t.Errorf("After second submit, Entries() = %v, want [second first]", entries)
	}

	// Submit third entry.
	h.Submit("third")

	entries = h.Entries()
	if len(entries) != 3 {
		t.Errorf("After third submit, len(Entries()) = %d, want 3", len(entries))
	}

	// Submit fourth entry (should drop oldest).
	h.Submit("fourth")

	entries = h.Entries()
	if len(entries) != 3 {
		t.Errorf("After fourth submit, len(Entries()) = %d, want 3 (limit)", len(entries))
	}

	if entries[0] != "fourth" || entries[1] != "third" || entries[2] != "second" {
		t.Errorf("After fourth submit, Entries() = %v, want [fourth third second]", entries)
	}
}

func TestHistory_PrevHistory_Empty(t *testing.T) {
	h := prompt.NewHistory(10)

	entry, ok := h.PrevHistory("draft")
	if ok {
		t.Errorf("PrevHistory on empty history returned ok=true, want false")
	}

	if entry != "" {
		t.Errorf("PrevHistory on empty history returned entry=%q, want empty", entry)
	}
}

func TestHistory_PrevHistory_SingleEntry(t *testing.T) {
	h := prompt.NewHistory(10)
	h.Submit("first")

	// First prev should return the entry.
	entry, ok := h.PrevHistory("draft")
	if !ok {
		t.Errorf("First PrevHistory returned ok=false, want true")
	}

	if entry != "first" {
		t.Errorf("First PrevHistory returned entry=%q, want 'first'", entry)
	}

	// Second prev should return false (already at oldest).
	entry, ok = h.PrevHistory("draft")
	if ok {
		t.Errorf("Second PrevHistory returned ok=true, want false (at oldest)")
	}
}

func TestHistory_PrevHistory_MultipleEntries(t *testing.T) {
	h := prompt.NewHistory(10)
	h.Submit("first")
	h.Submit("second")
	h.Submit("third")

	// Navigate backwards.
	entry, ok := h.PrevHistory("draft")
	if !ok || entry != "third" {
		t.Errorf("First PrevHistory = (%q, %v), want ('third', true)", entry, ok)
	}

	entry, ok = h.PrevHistory("draft")
	if !ok || entry != "second" {
		t.Errorf("Second PrevHistory = (%q, %v), want ('second', true)", entry, ok)
	}

	entry, ok = h.PrevHistory("draft")
	if !ok || entry != "first" {
		t.Errorf("Third PrevHistory = (%q, %v), want ('first', true)", entry, ok)
	}

	// At oldest, should return false.
	entry, ok = h.PrevHistory("draft")
	if ok {
		t.Errorf("Fourth PrevHistory returned ok=true, want false (at oldest)")
	}
}

func TestHistory_NextHistory(t *testing.T) {
	h := prompt.NewHistory(10)
	h.Submit("first")
	h.Submit("second")
	h.Submit("third")

	// Navigate back.
	h.PrevHistory("draft")
	h.PrevHistory("draft")
	h.PrevHistory("draft")

	// Navigate forward.
	entry, ok := h.NextHistory()
	if !ok || entry != "second" {
		t.Errorf("First NextHistory = (%q, %v), want ('second', true)", entry, ok)
	}

	entry, ok = h.NextHistory()
	if !ok || entry != "third" {
		t.Errorf("Second NextHistory = (%q, %v), want ('third', true)", entry, ok)
	}

	// At newest, should return draft.
	entry, ok = h.NextHistory()
	if !ok || entry != "draft" {
		t.Errorf("Third NextHistory = (%q, %v), want ('draft', true) (restore draft)", entry, ok)
	}

	// Beyond newest, should return false.
	entry, ok = h.NextHistory()
	if ok {
		t.Errorf("Fourth NextHistory returned ok=true, want false (beyond newest)")
	}
}

func TestHistory_DraftPreservation(t *testing.T) {
	h := prompt.NewHistory(10)
	h.Submit("first")
	h.Submit("second")

	// Start navigating with a draft.
	entry, ok := h.PrevHistory("my draft text")
	if !ok || entry != "second" {
		t.Errorf("PrevHistory = (%q, %v), want ('second', true)", entry, ok)
	}

	// Navigate to oldest.
	entry, ok = h.PrevHistory("my draft text")
	if !ok || entry != "first" {
		t.Errorf("PrevHistory = (%q, %v), want ('first', true)", entry, ok)
	}

	// Navigate back to draft.
	entry, ok = h.NextHistory()
	if !ok || entry != "second" {
		t.Errorf("NextHistory = (%q, %v), want ('second', true)", entry, ok)
	}

	entry, ok = h.NextHistory()
	if !ok || entry != "my draft text" {
		t.Errorf("NextHistory = (%q, %v), want ('my draft text', true)", entry, ok)
	}
}

func TestHistory_Reset(t *testing.T) {
	h := prompt.NewHistory(10)
	h.Submit("first")
	h.Submit("second")

	// Navigate.
	h.PrevHistory("draft")

	// Reset should clear navigation state.
	h.Reset()

	// Next PrevHistory should start from newest again.
	entry, ok := h.PrevHistory("new draft")
	if !ok || entry != "second" {
		t.Errorf("After Reset, PrevHistory = (%q, %v), want ('second', true)", entry, ok)
	}
}

func TestHistory_SubmitDuringNavigation(t *testing.T) {
	h := prompt.NewHistory(10)
	h.Submit("first")
	h.Submit("second")

	// Navigate.
	h.PrevHistory("draft")

	// Submit should add new entry and reset navigation.
	h.Submit("new entry")

	entries := h.Entries()
	if len(entries) != 3 {
		t.Errorf("After submit during nav, len(Entries()) = %d, want 3", len(entries))
	}

	if entries[0] != "new entry" {
		t.Errorf("Newest entry = %q, want 'new entry'", entries[0])
	}

	// Next PrevHistory should start from newest.
	entry, ok := h.PrevHistory("another draft")
	if !ok || entry != "new entry" {
		t.Errorf("After submit, PrevHistory = (%q, %v), want ('new entry', true)", entry, ok)
	}
}

func TestHistory_EmptyStringSubmit(t *testing.T) {
	h := prompt.NewHistory(10)

	// Submit empty string (should be allowed, readline does this).
	h.Submit("")

	entries := h.Entries()
	if len(entries) != 1 || entries[0] != "" {
		t.Errorf("After submitting empty, Entries() = %v, want ['']", entries)
	}
}

func TestHistory_DuplicateEntries(t *testing.T) {
	h := prompt.NewHistory(10)

	// Submit duplicate (should be allowed, both stored).
	h.Submit("duplicate")
	h.Submit("duplicate")

	entries := h.Entries()
	if len(entries) != 2 {
		t.Errorf("After two duplicates, len(Entries()) = %d, want 2", len(entries))
	}
}
