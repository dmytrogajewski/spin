package testkit

import (
	"testing"
	"time"

	"github.com/dmytrogajewski/spin/internal/ui/term"
)

func TestFakeKeyboard_InjectKey(t *testing.T) {
	t.Parallel()
	kb := NewFakeKeyboard()
	defer kb.Close()

	kb.InjectKey(term.KeyRune, 'a')

	select {
	case event := <-kb.Events():
		if event.Kind != term.KeyRune {
			t.Errorf("event.Kind = %v, want KeyRune", event.Kind)
		}

		if event.Rune != 'a' {
			t.Errorf("event.Rune = %c, want 'a'", event.Rune)
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("timeout waiting for key event")
	}
}

func TestFakeKeyboard_InjectString(t *testing.T) {
	t.Parallel()
	kb := NewFakeKeyboard()
	defer kb.Close()

	kb.InjectString("hello")

	expected := "hello"
	for _, r := range expected {
		select {
		case event := <-kb.Events():
			if event.Kind != term.KeyRune {
				t.Errorf("event.Kind = %v, want KeyRune", event.Kind)
			}

			if event.Rune != r {
				t.Errorf("event.Rune = %c, want %c", event.Rune, r)
			}
		case <-time.After(100 * time.Millisecond):
			t.Fatalf("timeout waiting for key event for rune %c", r)
		}
	}
}

func TestFakeKeyboard_InjectEnter(t *testing.T) {
	t.Parallel()
	kb := NewFakeKeyboard()
	defer kb.Close()

	kb.InjectEnter()

	select {
	case event := <-kb.Events():
		if event.Kind != term.KeyEnter {
			t.Errorf("event.Kind = %v, want KeyEnter", event.Kind)
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("timeout waiting for Enter event")
	}
}

func TestFakeKeyboard_InjectCtrlC(t *testing.T) {
	t.Parallel()
	kb := NewFakeKeyboard()
	defer kb.Close()

	kb.InjectCtrlC()

	select {
	case event := <-kb.Events():
		if event.Kind != term.KeyCtrlC {
			t.Errorf("event.Kind = %v, want KeyCtrlC", event.Kind)
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("timeout waiting for Ctrl-C event")
	}
}

func TestFakeKeyboard_InjectCtrlD(t *testing.T) {
	t.Parallel()
	kb := NewFakeKeyboard()
	defer kb.Close()

	kb.InjectCtrlD()

	select {
	case event := <-kb.Events():
		if event.Kind != term.KeyCtrlD {
			t.Errorf("event.Kind = %v, want KeyCtrlD", event.Kind)
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("timeout waiting for Ctrl-D event")
	}
}

func TestFakeKeyboard_InjectPaste(t *testing.T) {
	t.Parallel()
	kb := NewFakeKeyboard()
	defer kb.Close()

	text := "pasted text"
	kb.InjectPaste(text)

	select {
	case event := <-kb.Events():
		if event.Kind != term.KeyPaste {
			t.Errorf("event.Kind = %v, want KeyPaste", event.Kind)
		}

		if string(event.Paste) != text {
			t.Errorf("event.Paste = %q, want %q", string(event.Paste), text)
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("timeout waiting for Paste event")
	}
}

func TestFakeKeyboard_Close(t *testing.T) {
	t.Parallel()
	kb := NewFakeKeyboard()

	kb.InjectKey(term.KeyRune, 'a')

	// Read the event.
	<-kb.Events()

	// Close should close the channel.
	kb.Close()

	// Channel should be closed.
	select {
	case _, ok := <-kb.Events():
		if ok {
			t.Error("channel should be closed after Close()")
		}
	default:
		t.Error("channel should be closed after Close()")
	}

	// Injecting after close should be safe (no-op).
	kb.InjectKey(term.KeyRune, 'b')
}
