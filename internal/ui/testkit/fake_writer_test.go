package testkit

import (
	"strings"
	"sync"
	"testing"
	"time"
)

func TestFakeWriter_Write(t *testing.T) {
	w := NewFakeWriter()

	n, err := w.Write([]byte("hello"))
	if err != nil {
		t.Errorf("Write() error = %v", err)
	}
	if n != 5 {
		t.Errorf("Write() wrote %d bytes, want 5", n)
	}

	got := w.Snapshot()
	if got != "hello" {
		t.Errorf("Snapshot() = %q, want %q", got, "hello")
	}
}

func TestFakeWriter_WriteMultiple(t *testing.T) {
	w := NewFakeWriter()

	w.Write([]byte("hello "))
	w.Write([]byte("world"))

	got := w.Snapshot()
	if got != "hello world" {
		t.Errorf("Snapshot() = %q, want %q", got, "hello world")
	}
}

func TestFakeWriter_Snapshot(t *testing.T) {
	w := NewFakeWriter()

	// Empty snapshot
	if got := w.Snapshot(); got != "" {
		t.Errorf("Snapshot() on empty = %q, want empty string", got)
	}

	// After write
	w.Write([]byte("test"))
	if got := w.Snapshot(); got != "test" {
		t.Errorf("Snapshot() = %q, want %q", got, "test")
	}

	// Multiple snapshots don't mutate
	if got := w.Snapshot(); got != "test" {
		t.Errorf("Snapshot() second call = %q, want %q", got, "test")
	}
}

func TestFakeWriter_Reset(t *testing.T) {
	w := NewFakeWriter()

	w.Write([]byte("data"))
	w.Reset()

	if got := w.Snapshot(); got != "" {
		t.Errorf("Snapshot() after Reset() = %q, want empty", got)
	}

	// Write after reset
	w.Write([]byte("new"))
	if got := w.Snapshot(); got != "new" {
		t.Errorf("Snapshot() after reset + write = %q, want %q", got, "new")
	}
}

func TestFakeWriter_ContainsANSI(t *testing.T) {
	w := NewFakeWriter()

	tests := []struct {
		name    string
		content string
		seq     string
		want    bool
	}{
		{"exact match", "\x1b[31m", "\x1b[31m", true},
		{"substring", "hello\x1b[0mworld", "\x1b[0m", true},
		{"not present", "plain text", "\x1b[1m", false},
		{"empty", "", "\x1b[0m", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w.Reset()
			w.Write([]byte(tt.content))

			got := w.ContainsANSI(tt.seq)
			if got != tt.want {
				t.Errorf("ContainsANSI(%q) = %v, want %v", tt.seq, got, tt.want)
			}
		})
	}
}

func TestFakeWriter_WaitForContent_Immediate(t *testing.T) {
	w := NewFakeWriter()
	w.Write([]byte("hello world"))

	// Content already present - should return immediately
	found := w.WaitForContent("world", 100*time.Millisecond)
	if !found {
		t.Error("WaitForContent() = false, want true (content present)")
	}
}

func TestFakeWriter_WaitForContent_Timeout(t *testing.T) {
	w := NewFakeWriter()

	// Content never arrives - should timeout
	found := w.WaitForContent("missing", 50*time.Millisecond)
	if found {
		t.Error("WaitForContent() = true, want false (timeout)")
	}
}

func TestFakeWriter_WaitForContent_Arrival(t *testing.T) {
	w := NewFakeWriter()

	// Write content asynchronously after 50ms
	go func() {
		time.Sleep(50 * time.Millisecond)
		w.Write([]byte("delayed content"))
	}()

	// Wait for content (should arrive before timeout)
	found := w.WaitForContent("delayed", 200*time.Millisecond)
	if !found {
		t.Error("WaitForContent() = false, want true (content arrived)")
	}
}

func TestFakeWriter_Lines_Empty(t *testing.T) {
	w := NewFakeWriter()

	lines := w.Lines()
	if len(lines) != 0 {
		t.Errorf("Lines() on empty = %d lines, want 0", len(lines))
	}
}

func TestFakeWriter_Lines_Single(t *testing.T) {
	w := NewFakeWriter()
	w.Write([]byte("single line"))

	lines := w.Lines()
	if len(lines) != 1 {
		t.Errorf("Lines() = %d lines, want 1", len(lines))
	}
	if lines[0] != "single line" {
		t.Errorf("Lines()[0] = %q, want %q", lines[0], "single line")
	}
}

func TestFakeWriter_Lines_Multiple(t *testing.T) {
	w := NewFakeWriter()
	w.Write([]byte("line1\nline2\nline3"))

	lines := w.Lines()
	want := []string{"line1", "line2", "line3"}

	if len(lines) != len(want) {
		t.Errorf("Lines() = %d lines, want %d", len(lines), len(want))
	}

	for i, wantLine := range want {
		if i >= len(lines) {
			break
		}
		if lines[i] != wantLine {
			t.Errorf("Lines()[%d] = %q, want %q", i, lines[i], wantLine)
		}
	}
}

func TestFakeWriter_Lines_TrailingNewline(t *testing.T) {
	w := NewFakeWriter()
	w.Write([]byte("line1\nline2\n"))

	lines := w.Lines()
	// Split includes empty string after trailing newline
	want := []string{"line1", "line2", ""}

	if len(lines) != len(want) {
		t.Errorf("Lines() = %d lines, want %d", len(lines), len(want))
	}

	for i, wantLine := range want {
		if i >= len(lines) {
			break
		}
		if lines[i] != wantLine {
			t.Errorf("Lines()[%d] = %q, want %q", i, lines[i], wantLine)
		}
	}
}

func TestFakeWriter_StripANSI(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "no ANSI codes",
			input: "plain text",
			want:  "plain text",
		},
		{
			name:  "single code",
			input: "\x1b[31mred text\x1b[0m",
			want:  "red text",
		},
		{
			name:  "multiple codes",
			input: "\x1b[1m\x1b[31mbold red\x1b[0m normal",
			want:  "bold red normal",
		},
		{
			name:  "cursor codes",
			input: "hello\x1b[2Kworld\x1b[10Gtest",
			want:  "helloworldtest",
		},
		{
			name:  "empty",
			input: "",
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := NewFakeWriter()
			w.Write([]byte(tt.input))

			got := w.StripANSI()
			if got != tt.want {
				t.Errorf("StripANSI() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFakeWriter_Len(t *testing.T) {
	w := NewFakeWriter()

	if got := w.Len(); got != 0 {
		t.Errorf("Len() on empty = %d, want 0", got)
	}

	w.Write([]byte("12345"))
	if got := w.Len(); got != 5 {
		t.Errorf("Len() = %d, want 5", got)
	}

	w.Write([]byte("67890"))
	if got := w.Len(); got != 10 {
		t.Errorf("Len() after second write = %d, want 10", got)
	}

	w.Reset()
	if got := w.Len(); got != 0 {
		t.Errorf("Len() after Reset() = %d, want 0", got)
	}
}

func TestFakeWriter_ConcurrentWrites(t *testing.T) {
	w := NewFakeWriter()

	const goroutines = 10
	const writesPerGoroutine = 100

	var wg sync.WaitGroup
	wg.Add(goroutines)

	for i := 0; i < goroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < writesPerGoroutine; j++ {
				w.Write([]byte("x"))
			}
		}(i)
	}

	wg.Wait()

	// Verify total writes
	got := w.Len()
	want := goroutines * writesPerGoroutine
	if got != want {
		t.Errorf("Len() after concurrent writes = %d, want %d", got, want)
	}

	// Verify all 'x' characters
	snapshot := w.Snapshot()
	if count := strings.Count(snapshot, "x"); count != want {
		t.Errorf("Snapshot() contains %d 'x', want %d", count, want)
	}
}

func TestFakeWriter_ConcurrentReads(t *testing.T) {
	w := NewFakeWriter()
	w.Write([]byte("stable content"))

	const goroutines = 10
	var wg sync.WaitGroup
	wg.Add(goroutines)

	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				_ = w.Snapshot()
				_ = w.Lines()
				_ = w.StripANSI()
				_ = w.Len()
			}
		}()
	}

	wg.Wait()
}

func TestFakeWriter_ConcurrentWriteAndWait(t *testing.T) {
	w := NewFakeWriter()

	// Start waiter
	done := make(chan bool)
	go func() {
		found := w.WaitForContent("target", 500*time.Millisecond)
		done <- found
	}()

	// Write content after delay
	time.Sleep(50 * time.Millisecond)
	w.Write([]byte("this is the target content"))

	// Verify waiter found content
	found := <-done
	if !found {
		t.Error("WaitForContent() = false, want true")
	}
}
