package testkit

import (
	"sync"
	"testing"
	"time"

	"github.com/dmytrogajewski/spin/internal/ui/term"
)

func TestFakeKeyboard_InjectKey(t *testing.T) {
	kb := NewFakeKeyboard()
	defer kb.Close()

	kb.InjectKey(term.KeyRune, 'a')

	select {
	case ev := <-kb.Events():
		if ev.Kind != term.KeyRune || ev.Rune != 'a' {
			t.Errorf("Events() = %+v, want KeyRune 'a'", ev)
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("Events() timeout, want event")
	}
}

func TestFakeKeyboard_InjectString(t *testing.T) {
	kb := NewFakeKeyboard()
	defer kb.Close()

	kb.InjectString("hello")

	want := []rune{'h', 'e', 'l', 'l', 'o'}
	for i, wantRune := range want {
		select {
		case ev := <-kb.Events():
			if ev.Kind != term.KeyRune || ev.Rune != wantRune {
				t.Errorf("Events()[%d] = %+v, want KeyRune %q", i, ev, wantRune)
			}
		case <-time.After(100 * time.Millisecond):
			t.Errorf("Events() timeout at index %d", i)
		}
	}
}

func TestFakeKeyboard_InjectEnter(t *testing.T) {
	kb := NewFakeKeyboard()
	defer kb.Close()

	kb.InjectEnter()

	select {
	case ev := <-kb.Events():
		if ev.Kind != term.KeyEnter {
			t.Errorf("Events() = %+v, want KeyEnter", ev)
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("Events() timeout")
	}
}

func TestFakeKeyboard_InjectCtrlC(t *testing.T) {
	kb := NewFakeKeyboard()
	defer kb.Close()

	kb.InjectCtrlC()

	select {
	case ev := <-kb.Events():
		if ev.Kind != term.KeyCtrlC {
			t.Errorf("Events() = %+v, want KeyCtrlC", ev)
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("Events() timeout")
	}
}

func TestFakeKeyboard_InjectCtrlD(t *testing.T) {
	kb := NewFakeKeyboard()
	defer kb.Close()

	kb.InjectCtrlD()

	select {
	case ev := <-kb.Events():
		if ev.Kind != term.KeyCtrlD {
			t.Errorf("Events() = %+v, want KeyCtrlD", ev)
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("Events() timeout")
	}
}

func TestFakeKeyboard_InjectArrowKeys(t *testing.T) {
	tests := []struct {
		name   string
		inject func(*FakeKeyboard)
		want   term.KeyKind
	}{
		{"up", (*FakeKeyboard).InjectUp, term.KeyUp},
		{"down", (*FakeKeyboard).InjectDown, term.KeyDown},
		{"left", (*FakeKeyboard).InjectLeft, term.KeyLeft},
		{"right", (*FakeKeyboard).InjectRight, term.KeyRight},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			kb := NewFakeKeyboard()
			defer kb.Close()

			tt.inject(kb)

			select {
			case ev := <-kb.Events():
				if ev.Kind != tt.want {
					t.Errorf("Events() = %+v, want %v", ev, tt.want)
				}
			case <-time.After(100 * time.Millisecond):
				t.Error("Events() timeout")
			}
		})
	}
}

func TestFakeKeyboard_InjectPageUpDown(t *testing.T) {
	kb := NewFakeKeyboard()
	defer kb.Close()

	kb.InjectPageUp()
	kb.InjectPageDown()

	// Verify PageUp
	select {
	case ev := <-kb.Events():
		if ev.Kind != term.KeyPgUp {
			t.Errorf("Events() = %+v, want KeyPgUp", ev)
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("Events() timeout for PageUp")
	}

	// Verify PageDown
	select {
	case ev := <-kb.Events():
		if ev.Kind != term.KeyPgDn {
			t.Errorf("Events() = %+v, want KeyPgDn", ev)
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("Events() timeout for PageDown")
	}
}

func TestFakeKeyboard_InjectPaste(t *testing.T) {
	kb := NewFakeKeyboard()
	defer kb.Close()

	kb.InjectPaste("paste text")

	// Verify KeyPaste
	ev := <-kb.Events()
	if ev.Kind != term.KeyPaste {
		t.Errorf("Events()[0] = %+v, want KeyPaste", ev)
	}

	// Verify text runes
	text := "paste text"
	for i, r := range text {
		ev := <-kb.Events()
		if ev.Kind != term.KeyRune || ev.Rune != r {
			t.Errorf("Events()[%d] = %+v, want KeyRune %q", i+1, ev, r)
		}
	}
}

func TestFakeKeyboard_Close(t *testing.T) {
	kb := NewFakeKeyboard()

	kb.Close()

	// Verify channel closed
	select {
	case _, ok := <-kb.Events():
		if ok {
			t.Error("Events() still open after Close()")
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("Events() did not close")
	}

	// Verify IsClosed
	if !kb.IsClosed() {
		t.Error("IsClosed() = false, want true")
	}
}

func TestFakeKeyboard_CloseIdempotent(t *testing.T) {
	kb := NewFakeKeyboard()

	// Close multiple times - should not panic
	kb.Close()
	kb.Close()
	kb.Close()

	if !kb.IsClosed() {
		t.Error("IsClosed() = false after multiple Close()")
	}
}

func TestFakeKeyboard_InjectAfterClose(t *testing.T) {
	kb := NewFakeKeyboard()
	kb.Close()

	// InjectKey after close should not panic or block
	kb.InjectKey(term.KeyRune, 'x')

	// Channel should remain closed, no new events
	select {
	case _, ok := <-kb.Events():
		if ok {
			t.Error("Events() returned value after Close() + InjectKey()")
		}
	case <-time.After(50 * time.Millisecond):
		t.Error("Events() blocked after Close()")
	}
}

func TestFakeKeyboard_ConcurrentInject(t *testing.T) {
	kb := NewFakeKeyboard()
	defer kb.Close()

	const goroutines = 10
	const eventsPerGoroutine = 100

	var wg sync.WaitGroup
	wg.Add(goroutines)

	// Concurrent injectors
	for i := 0; i < goroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < eventsPerGoroutine; j++ {
				kb.InjectKey(term.KeyRune, 'a')
			}
		}(i)
	}

	// Concurrent consumer
	received := 0
	done := make(chan struct{})
	go func() {
		for range kb.Events() {
			received++
			if received == goroutines*eventsPerGoroutine {
				close(done)
				return
			}
		}
	}()

	wg.Wait()
	kb.Close()

	select {
	case <-done:
		// Success
	case <-time.After(2 * time.Second):
		t.Errorf("received %d events, want %d", received, goroutines*eventsPerGoroutine)
	}
}

func TestFakeKeyboard_BufferSize(t *testing.T) {
	kb := NewFakeKeyboard()
	defer kb.Close()

	// Inject many events without consuming (tests buffer capacity)
	for i := 0; i < 500; i++ {
		kb.InjectKey(term.KeyRune, 'x')
	}

	// Verify all events buffered
	count := 0
	timeout := time.After(1 * time.Second)
	for count < 500 {
		select {
		case <-kb.Events():
			count++
		case <-timeout:
			t.Errorf("received %d events, want 500 (buffer overflow?)", count)
			return
		}
	}
}

func TestFakeKeyboard_MultipleHelpers(t *testing.T) {
	kb := NewFakeKeyboard()
	defer kb.Close()

	// Use multiple inject helpers
	kb.InjectString("hello")
	kb.InjectEnter()
	kb.InjectCtrlU()
	kb.InjectBackspace()
	kb.InjectEscape()

	want := []term.KeyKind{
		term.KeyRune, term.KeyRune, term.KeyRune, term.KeyRune, term.KeyRune, // "hello"
		term.KeyEnter,
		term.KeyCtrlU,
		term.KeyBackspace,
		term.KeyEscape,
	}

	for i, wantKind := range want {
		select {
		case ev := <-kb.Events():
			if ev.Kind != wantKind {
				t.Errorf("Events()[%d] = %v, want %v", i, ev.Kind, wantKind)
			}
		case <-time.After(100 * time.Millisecond):
			t.Errorf("Events() timeout at index %d", i)
			return
		}
	}
}
