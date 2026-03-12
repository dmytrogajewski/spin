package testkit

import (
	"testing"
	"time"
)

func TestFakeWriter_Write(t *testing.T) {
	w := NewFakeWriter()

	n, err := w.Write([]byte("hello"))
	if err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	if n != 5 {
		t.Errorf("Write() n = %d, want 5", n)
	}

	if w.Snapshot() != "hello" {
		t.Errorf("Snapshot() = %q, want %q", w.Snapshot(), "hello")
	}
}

func TestFakeWriter_Snapshot(t *testing.T) {
	w := NewFakeWriter()

	w.Write([]byte("test"))

	snapshot := w.Snapshot()
	if snapshot != "test" {
		t.Errorf("Snapshot() = %q, want %q", snapshot, "test")
	}
}

func TestFakeWriter_Reset(t *testing.T) {
	w := NewFakeWriter()

	w.Write([]byte("test"))
	w.Reset()

	if w.Snapshot() != "" {
		t.Errorf("Snapshot() after Reset() = %q, want empty", w.Snapshot())
	}
}

func TestFakeWriter_Contains(t *testing.T) {
	w := NewFakeWriter()

	w.Write([]byte("hello world"))

	if !w.Contains("hello") {
		t.Error("Contains(\"hello\") = false, want true")
	}

	if !w.Contains("world") {
		t.Error("Contains(\"world\") = false, want true")
	}

	if w.Contains("missing") {
		t.Error("Contains(\"missing\") = true, want false")
	}
}

func TestFakeWriter_ContainsANSI(t *testing.T) {
	w := NewFakeWriter()

	w.Write([]byte("\x1b[31mred\x1b[0m"))

	if !w.ContainsANSI("\x1b[31m") {
		t.Error("ContainsANSI(\"\\x1b[31m\") = false, want true")
	}

	if !w.ContainsANSI("\x1b[0m") {
		t.Error("ContainsANSI(\"\\x1b[0m\") = false, want true")
	}
}

func TestFakeWriter_StripANSI(t *testing.T) {
	w := NewFakeWriter()

	w.Write([]byte("\x1b[31mred\x1b[0m text"))

	stripped := w.StripANSI()
	if stripped != "red text" {
		t.Errorf("StripANSI() = %q, want %q", stripped, "red text")
	}
}

func TestFakeWriter_Lines(t *testing.T) {
	w := NewFakeWriter()

	w.Write([]byte("line1\nline2\nline3"))

	lines := w.Lines()
	if len(lines) != 3 {
		t.Errorf("Lines() len = %d, want 3", len(lines))
	}

	if lines[0] != "line1" {
		t.Errorf("Lines()[0] = %q, want %q", lines[0], "line1")
	}

	if lines[1] != "line2" {
		t.Errorf("Lines()[1] = %q, want %q", lines[1], "line2")
	}
}

func TestFakeWriter_WaitForContent(t *testing.T) {
	w := NewFakeWriter()

	// Start goroutine that writes after delay.
	go func() {
		time.Sleep(50 * time.Millisecond)
		w.Write([]byte("delayed content"))
	}()

	// WaitForContent should find it.
	if !w.WaitForContent("delayed", 200*time.Millisecond) {
		t.Error("WaitForContent() = false, want true")
	}
}

func TestFakeWriter_WaitForContent_Timeout(t *testing.T) {
	w := NewFakeWriter()

	// WaitForContent should timeout if content never appears.
	if w.WaitForContent("missing", 50*time.Millisecond) {
		t.Error("WaitForContent() = true, want false (timeout)")
	}
}

func TestFakeWriter_ConcurrentWrite(t *testing.T) {
	w := NewFakeWriter()

	// Write concurrently.
	done := make(chan bool)

	for i := range 10 {
		go func(n int) {
			w.Write([]byte("test"))

			done <- true
		}(i)
	}

	// Wait for all writes.
	for range 10 {
		<-done
	}

	// Should have written 10 times.
	if w.Len() != 40 { // 4 bytes * 10.
		t.Errorf("Len() = %d, want 40", w.Len())
	}
}
