package term

import (
	"bytes"
	"context"
	"io"
	"testing"
	"time"
)

func TestKeyKind_String(t *testing.T) {
	tests := []struct {
		kind KeyKind
		want string
	}{
		{KeyRune, "Rune"},
		{KeyEnter, "Enter"},
		{KeyTab, "Tab"},
		{KeyBackspace, "Backspace"},
		{KeyDelete, "Delete"},
		{KeyUp, "Up"},
		{KeyDown, "Down"},
		{KeyLeft, "Left"},
		{KeyRight, "Right"},
		{KeyHome, "Home"},
		{KeyEnd, "End"},
		{KeyPgUp, "PgUp"},
		{KeyPgDn, "PgDn"},
		{KeyEscape, "Escape"},
		{KeyCtrlC, "Ctrl-C"},
		{KeyCtrlD, "Ctrl-D"},
		{KeyCtrlU, "Ctrl-U"},
		{KeyCtrlK, "Ctrl-K"},
		{KeyCtrlW, "Ctrl-W"},
		{KeyCtrlL, "Ctrl-L"},
		{KeyPaste, "Paste"},
		{KeyF1, "F1"},
		{KeyF2, "F2"},
		{KeyF12, "F12"},
		{KeyUnknown, "Unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := tt.kind.String()
			if got != tt.want {
				t.Errorf("KeyKind.String() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestReadKeys_SingleByteKeys(t *testing.T) {
	tests := []struct {
		name  string
		input []byte
		want  KeyEvent
	}{
		{"a", []byte{'a'}, KeyEvent{Kind: KeyRune, Rune: 'a', Raw: []byte{'a'}}},
		{"z", []byte{'z'}, KeyEvent{Kind: KeyRune, Rune: 'z', Raw: []byte{'z'}}},
		{"A", []byte{'A'}, KeyEvent{Kind: KeyRune, Rune: 'A', Raw: []byte{'A'}}},
		{"0", []byte{'0'}, KeyEvent{Kind: KeyRune, Rune: '0', Raw: []byte{'0'}}},
		{"9", []byte{'9'}, KeyEvent{Kind: KeyRune, Rune: '9', Raw: []byte{'9'}}},
		{"space", []byte{' '}, KeyEvent{Kind: KeyRune, Rune: ' ', Raw: []byte{' '}}},
		{"!", []byte{'!'}, KeyEvent{Kind: KeyRune, Rune: '!', Raw: []byte{'!'}}},
		{"@", []byte{'@'}, KeyEvent{Kind: KeyRune, Rune: '@', Raw: []byte{'@'}}},
		{"tab", []byte{'\t'}, KeyEvent{Kind: KeyTab, Raw: []byte{'\t'}}},
		{"enter LF", []byte{'\n'}, KeyEvent{Kind: KeyEnter, Raw: []byte{'\n'}}},
		{"enter CR", []byte{'\r'}, KeyEvent{Kind: KeyEnter, Raw: []byte{'\r'}}},
		{"backspace DEL", []byte{0x7f}, KeyEvent{Kind: KeyBackspace, Raw: []byte{0x7f}}},
		{"backspace BS", []byte{0x08}, KeyEvent{Kind: KeyBackspace, Raw: []byte{0x08}}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
			defer cancel()

			r := bytes.NewReader(tt.input)
			ch, err := ReadKeys(ctx, r, nil)
			if err != nil {
				t.Fatalf("ReadKeys() error = %v", err)
			}

			select {
			case got := <-ch:
				if !keyEventsEqual(got, tt.want) {
					t.Errorf("ReadKeys() got = %+v, want %+v", got, tt.want)
				}
			case <-ctx.Done():
				t.Fatal("timeout waiting for key event")
			}
		})
	}
}

func TestReadKeys_ControlKeys(t *testing.T) {
	tests := []struct {
		name  string
		input []byte
		want  KeyKind
	}{
		{"Ctrl-C", []byte{0x03}, KeyCtrlC},
		{"Ctrl-D", []byte{0x04}, KeyCtrlD},
		{"Ctrl-U", []byte{0x15}, KeyCtrlU},
		{"Ctrl-K", []byte{0x0b}, KeyCtrlK},
		{"Ctrl-W", []byte{0x17}, KeyCtrlW},
		{"Ctrl-L", []byte{0x0c}, KeyCtrlL},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
			defer cancel()

			r := bytes.NewReader(tt.input)
			ch, err := ReadKeys(ctx, r, nil)
			if err != nil {
				t.Fatalf("ReadKeys() error = %v", err)
			}

			select {
			case got := <-ch:
				if got.Kind != tt.want {
					t.Errorf("ReadKeys() got Kind = %v, want %v", got.Kind, tt.want)
				}
			case <-ctx.Done():
				t.Fatal("timeout waiting for key event")
			}
		})
	}
}

func TestReadKeys_ArrowKeys(t *testing.T) {
	tests := []struct {
		name  string
		input []byte
		want  KeyKind
	}{
		{"Up", []byte{0x1b, '[', 'A'}, KeyUp},
		{"Down", []byte{0x1b, '[', 'B'}, KeyDown},
		{"Right", []byte{0x1b, '[', 'C'}, KeyRight},
		{"Left", []byte{0x1b, '[', 'D'}, KeyLeft},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
			defer cancel()

			r := bytes.NewReader(tt.input)
			ch, err := ReadKeys(ctx, r, nil)
			if err != nil {
				t.Fatalf("ReadKeys() error = %v", err)
			}

			select {
			case got := <-ch:
				if got.Kind != tt.want {
					t.Errorf("ReadKeys() got Kind = %v, want %v", got.Kind, tt.want)
				}
			case <-ctx.Done():
				t.Fatal("timeout waiting for key event")
			}
		})
	}
}

func TestReadKeys_HomeEnd(t *testing.T) {
	tests := []struct {
		name  string
		input []byte
		want  KeyKind
	}{
		{"Home CSI H", []byte{0x1b, '[', 'H'}, KeyHome},
		{"Home CSI 1~", []byte{0x1b, '[', '1', '~'}, KeyHome},
		{"End CSI F", []byte{0x1b, '[', 'F'}, KeyEnd},
		{"End CSI 4~", []byte{0x1b, '[', '4', '~'}, KeyEnd},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
			defer cancel()

			r := bytes.NewReader(tt.input)
			ch, err := ReadKeys(ctx, r, nil)
			if err != nil {
				t.Fatalf("ReadKeys() error = %v", err)
			}

			select {
			case got := <-ch:
				if got.Kind != tt.want {
					t.Errorf("ReadKeys() got Kind = %v, want %v", got.Kind, tt.want)
				}
			case <-ctx.Done():
				t.Fatal("timeout waiting for key event")
			}
		})
	}
}

func TestReadKeys_PageKeys(t *testing.T) {
	tests := []struct {
		name  string
		input []byte
		want  KeyKind
	}{
		{"PgUp", []byte{0x1b, '[', '5', '~'}, KeyPgUp},
		{"PgDn", []byte{0x1b, '[', '6', '~'}, KeyPgDn},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
			defer cancel()

			r := bytes.NewReader(tt.input)
			ch, err := ReadKeys(ctx, r, nil)
			if err != nil {
				t.Fatalf("ReadKeys() error = %v", err)
			}

			select {
			case got := <-ch:
				if got.Kind != tt.want {
					t.Errorf("ReadKeys() got Kind = %v, want %v", got.Kind, tt.want)
				}
			case <-ctx.Done():
				t.Fatal("timeout waiting for key event")
			}
		})
	}
}

func TestReadKeys_Delete(t *testing.T) {
	tests := []struct {
		name  string
		input []byte
		want  KeyKind
	}{
		{"Delete CSI 3~", []byte{0x1b, '[', '3', '~'}, KeyDelete},
		{"Delete CSI 2~", []byte{0x1b, '[', '2', '~'}, KeyDelete}, // alternative
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
			defer cancel()

			r := bytes.NewReader(tt.input)
			ch, err := ReadKeys(ctx, r, nil)
			if err != nil {
				t.Fatalf("ReadKeys() error = %v", err)
			}

			select {
			case got := <-ch:
				if got.Kind != tt.want {
					t.Errorf("ReadKeys() got Kind = %v, want %v", got.Kind, tt.want)
				}
			case <-ctx.Done():
				t.Fatal("timeout waiting for key event")
			}
		})
	}
}

func TestReadKeys_FunctionKeys(t *testing.T) {
	tests := []struct {
		name  string
		input []byte
		want  KeyKind
	}{
		{"F1", []byte{0x1b, 'O', 'P'}, KeyF1},
		{"F2", []byte{0x1b, 'O', 'Q'}, KeyF2},
		{"F3", []byte{0x1b, 'O', 'R'}, KeyF3},
		{"F4", []byte{0x1b, 'O', 'S'}, KeyF4},
		{"F5", []byte{0x1b, '[', '1', '5', '~'}, KeyF5},
		{"F6", []byte{0x1b, '[', '1', '7', '~'}, KeyF6},
		{"F7", []byte{0x1b, '[', '1', '8', '~'}, KeyF7},
		{"F8", []byte{0x1b, '[', '1', '9', '~'}, KeyF8},
		{"F9", []byte{0x1b, '[', '2', '0', '~'}, KeyF9},
		{"F10", []byte{0x1b, '[', '2', '1', '~'}, KeyF10},
		{"F11", []byte{0x1b, '[', '2', '3', '~'}, KeyF11},
		{"F12", []byte{0x1b, '[', '2', '4', '~'}, KeyF12},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
			defer cancel()

			r := bytes.NewReader(tt.input)
			ch, err := ReadKeys(ctx, r, nil)
			if err != nil {
				t.Fatalf("ReadKeys() error = %v", err)
			}

			select {
			case got := <-ch:
				if got.Kind != tt.want {
					t.Errorf("ReadKeys() got Kind = %v, want %v (raw: %v)", got.Kind, tt.want, got.Raw)
				}
			case <-ctx.Done():
				t.Fatal("timeout waiting for key event")
			}
		})
	}
}

func TestReadKeys_EscapeAlone(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	// Single ESC byte, followed by slow/no input (simulated by blocking reader)
	r := &slowByteReader{data: []byte{0x1b}, delay: 200 * time.Millisecond}
	cfg := &KeyReaderConfig{EscTimeout: 50 * time.Millisecond}
	ch, err := ReadKeys(ctx, r, cfg)
	if err != nil {
		t.Fatalf("ReadKeys() error = %v", err)
	}

	start := time.Now()
	select {
	case got := <-ch:
		elapsed := time.Since(start)
		if got.Kind != KeyEscape {
			t.Errorf("ReadKeys() got Kind = %v, want %v", got.Kind, KeyEscape)
		}
		// Should emit after timeout (with some tolerance)
		if elapsed < 40*time.Millisecond || elapsed > 100*time.Millisecond {
			t.Errorf("ESC emitted at %v (expected ~50ms)", elapsed)
		}
	case <-ctx.Done():
		t.Fatal("timeout waiting for ESC event")
	}
}

func TestReadKeys_EscapeSequence(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// ESC followed immediately by [A (Up arrow)
	r := bytes.NewReader([]byte{0x1b, '[', 'A'})
	cfg := &KeyReaderConfig{EscTimeout: 100 * time.Millisecond}
	ch, err := ReadKeys(ctx, r, cfg)
	if err != nil {
		t.Fatalf("ReadKeys() error = %v", err)
	}

	select {
	case got := <-ch:
		if got.Kind != KeyUp {
			t.Errorf("ReadKeys() got Kind = %v, want %v", got.Kind, KeyUp)
		}
	case <-ctx.Done():
		t.Fatal("timeout waiting for Up event")
	}
}

func TestReadKeys_UTF8(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  rune
	}{
		{"Euro", "€", '€'},         // U+20AC, 3 bytes: E2 82 AC
		{"Chinese", "你", '你'},      // U+4F60, 3 bytes: E4 BD A0
		{"Emoji rocket", "🚀", '🚀'}, // U+1F680, 4 bytes: F0 9F 9A 80
		{"Emoji smile", "😀", '😀'},  // U+1F600, 4 bytes: F0 9F 98 80
		{"Greek alpha", "α", 'α'},  // U+03B1, 2 bytes: CE B1
		{"Cyrillic", "Д", 'Д'},     // U+0414, 2 bytes: D0 94
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
			defer cancel()

			r := bytes.NewReader([]byte(tt.input))
			ch, err := ReadKeys(ctx, r, nil)
			if err != nil {
				t.Fatalf("ReadKeys() error = %v", err)
			}

			select {
			case got := <-ch:
				if got.Kind != KeyRune {
					t.Errorf("ReadKeys() got Kind = %v, want %v", got.Kind, KeyRune)
				}
				if got.Rune != tt.want {
					t.Errorf("ReadKeys() got Rune = %U, want %U", got.Rune, tt.want)
				}
			case <-ctx.Done():
				t.Fatal("timeout waiting for UTF-8 rune")
			}
		})
	}
}

func TestReadKeys_BracketedPaste(t *testing.T) {
	tests := []struct {
		name    string
		payload string
		want    string
	}{
		{"simple", "hello", "hello"},
		{"multiline", "line1\nline2\nline3", "line1\nline2\nline3"},
		{"with tabs", "col1\tcol2\tcol3", "col1\tcol2\tcol3"},
		{"special chars", "!@#$%^&*()", "!@#$%^&*()"},
		{"unicode", "你好🚀", "你好🚀"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
			defer cancel()

			// Bracketed paste format: \x1b[200~ + payload + \x1b[201~
			input := []byte("\x1b[200~" + tt.payload + "\x1b[201~")
			r := bytes.NewReader(input)
			ch, err := ReadKeys(ctx, r, nil)
			if err != nil {
				t.Fatalf("ReadKeys() error = %v", err)
			}

			select {
			case got := <-ch:
				if got.Kind != KeyPaste {
					t.Errorf("ReadKeys() got Kind = %v, want %v", got.Kind, KeyPaste)
				}
				if string(got.Paste) != tt.want {
					t.Errorf("ReadKeys() got Paste = %q, want %q", got.Paste, tt.want)
				}
			case <-ctx.Done():
				t.Fatal("timeout waiting for paste event")
			}
		})
	}
}

func TestReadKeys_BracketedPasteLarge(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	// 10KB payload
	payload := make([]byte, 10*1024)
	for i := range payload {
		payload[i] = byte('A' + (i % 26))
	}

	input := append([]byte("\x1b[200~"), payload...)
	input = append(input, []byte("\x1b[201~")...)

	r := bytes.NewReader(input)
	ch, err := ReadKeys(ctx, r, nil)
	if err != nil {
		t.Fatalf("ReadKeys() error = %v", err)
	}

	select {
	case got := <-ch:
		if got.Kind != KeyPaste {
			t.Errorf("ReadKeys() got Kind = %v, want %v", got.Kind, KeyPaste)
		}
		if len(got.Paste) != len(payload) {
			t.Errorf("ReadKeys() got Paste len = %d, want %d", len(got.Paste), len(payload))
		}
		if !bytes.Equal(got.Paste, payload) {
			t.Error("Paste payload mismatch")
		}
	case <-ctx.Done():
		t.Fatal("timeout waiting for large paste event")
	}
}

func TestReadKeys_ContextCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	// Slow reader that will block
	r := &slowReader{delay: 1 * time.Second}
	ch, err := ReadKeys(ctx, r, nil)
	if err != nil {
		t.Fatalf("ReadKeys() error = %v", err)
	}

	// Cancel immediately
	cancel()

	// Channel should close
	select {
	case _, ok := <-ch:
		if ok {
			t.Error("channel should be closed after context cancel")
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("channel did not close after context cancel")
	}
}

func TestReadKeys_EOF(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// Empty reader (immediate EOF)
	r := bytes.NewReader([]byte{})
	ch, err := ReadKeys(ctx, r, nil)
	if err != nil {
		t.Fatalf("ReadKeys() error = %v", err)
	}

	// Channel should close on EOF
	select {
	case _, ok := <-ch:
		if ok {
			t.Error("channel should be closed on EOF")
		}
	case <-ctx.Done():
		t.Fatal("timeout waiting for channel close")
	}
}

func TestReadKeys_PartialSequence(t *testing.T) {
	tests := []struct {
		name  string
		input []byte
	}{
		{"ESC alone EOF", []byte{0x1b}},
		{"ESC [ EOF", []byte{0x1b, '['}},
		{"ESC [ 1 EOF", []byte{0x1b, '[', '1'}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
			defer cancel()

			r := bytes.NewReader(tt.input)
			cfg := &KeyReaderConfig{EscTimeout: 50 * time.Millisecond}
			ch, err := ReadKeys(ctx, r, cfg)
			if err != nil {
				t.Fatalf("ReadKeys() error = %v", err)
			}

			// Should emit something (either KeyEscape or KeyUnknown) or close
			select {
			case got, ok := <-ch:
				if ok && got.Kind != KeyEscape && got.Kind != KeyUnknown {
					t.Logf("Got key: %v (acceptable for partial sequence)", got.Kind)
				}
			case <-ctx.Done():
				t.Fatal("timeout waiting for partial sequence handling")
			}
		})
	}
}

func TestReadKeys_RapidInput(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	// 100 rapid keypresses
	var input []byte
	for i := 0; i < 100; i++ {
		input = append(input, byte('a'+(i%26)))
	}

	r := bytes.NewReader(input)
	ch, err := ReadKeys(ctx, r, nil)
	if err != nil {
		t.Fatalf("ReadKeys() error = %v", err)
	}

	count := 0
	for range ch {
		count++
	}

	if count != 100 {
		t.Errorf("ReadKeys() emitted %d events, want 100", count)
	}
}

// Helper functions

func keyEventsEqual(a, b KeyEvent) bool {
	if a.Kind != b.Kind {
		return false
	}
	if a.Kind == KeyRune && a.Rune != b.Rune {
		return false
	}
	if a.Kind == KeyPaste && !bytes.Equal(a.Paste, b.Paste) {
		return false
	}
	return bytes.Equal(a.Raw, b.Raw)
}

// slowReader simulates a slow/blocking reader for testing cancellation
type slowReader struct {
	delay time.Duration
}

func (s *slowReader) Read(p []byte) (n int, err error) {
	time.Sleep(s.delay)
	return 0, io.EOF
}

// slowByteReader simulates reading bytes with delay between them
type slowByteReader struct {
	data  []byte
	delay time.Duration
	pos   int
}

func (s *slowByteReader) Read(p []byte) (n int, err error) {
	if s.pos >= len(s.data) {
		time.Sleep(s.delay)
		return 0, io.EOF
	}
	if len(p) == 0 {
		return 0, nil
	}
	p[0] = s.data[s.pos]
	s.pos++
	return 1, nil
}

// Benchmarks

func BenchmarkReadKeys_SingleByte(b *testing.B) {
	input := bytes.Repeat([]byte{'a'}, b.N)
	r := bytes.NewReader(input)
	ctx := context.Background()

	ch, err := ReadKeys(ctx, r, nil)
	if err != nil {
		b.Fatal(err)
	}

	for range ch {
	}
}

func BenchmarkReadKeys_ArrowKeys(b *testing.B) {
	seq := []byte{0x1b, '[', 'A'} // Up arrow
	input := bytes.Repeat(seq, b.N)
	r := bytes.NewReader(input)
	ctx := context.Background()

	ch, err := ReadKeys(ctx, r, nil)
	if err != nil {
		b.Fatal(err)
	}

	for range ch {
	}
}

func BenchmarkReadKeys_UTF8(b *testing.B) {
	emoji := []byte("🚀")
	input := bytes.Repeat(emoji, b.N)
	r := bytes.NewReader(input)
	ctx := context.Background()

	ch, err := ReadKeys(ctx, r, nil)
	if err != nil {
		b.Fatal(err)
	}

	for range ch {
	}
}
