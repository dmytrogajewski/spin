package testkit

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"
)

const writerTickInterval = 10 * time.Millisecond

// FakeWriter captures all output written to it for testing.
// It provides thread-safe access and ANSI sequence helpers.
type FakeWriter struct {
	mu  sync.RWMutex
	buf bytes.Buffer
}

// NewFakeWriter creates a new fake writer.
func NewFakeWriter() *FakeWriter {
	return &FakeWriter{}
}

// Write implements [io.Writer].
func (f *FakeWriter) Write(p []byte) (int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	n, err := f.buf.Write(p)
	if err != nil {
		return n, fmt.Errorf("fake writer write: %w", err)
	}

	return n, nil
}

// Snapshot returns the current output as a string.
func (f *FakeWriter) Snapshot() string {
	f.mu.RLock()
	defer f.mu.RUnlock()

	return f.buf.String()
}

// Reset clears the buffer.
func (f *FakeWriter) Reset() {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.buf.Reset()
}

// Contains checks if the output contains the given substring.
func (f *FakeWriter) Contains(s string) bool {
	f.mu.RLock()
	defer f.mu.RUnlock()

	return strings.Contains(f.buf.String(), s)
}

// ContainsANSI checks if the output contains the given ANSI sequence.
func (f *FakeWriter) ContainsANSI(seq string) bool {
	f.mu.RLock()
	defer f.mu.RUnlock()

	return strings.Contains(f.buf.String(), seq)
}

// StripANSI returns the output with ANSI escape sequences removed.
func (f *FakeWriter) StripANSI() string {
	f.mu.RLock()
	defer f.mu.RUnlock()

	content := f.buf.String()
	// Remove ANSI escape sequences.
	ansiRegex := regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)

	return ansiRegex.ReplaceAllString(content, "")
}

// Lines returns the output split by newlines.
func (f *FakeWriter) Lines() []string {
	f.mu.RLock()
	defer f.mu.RUnlock()

	return strings.Split(f.buf.String(), "\n")
}

// WaitForContent blocks until the output contains the given substring or timeout.
// Returns true if content was found, false on timeout.
func (f *FakeWriter) WaitForContent(s string, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)

	ticker := time.NewTicker(writerTickInterval)
	defer ticker.Stop()

	for time.Now().Before(deadline) {
		if f.Contains(s) {
			return true
		}

		<-ticker.C
	}

	return false
}

// Len returns the current buffer length.
func (f *FakeWriter) Len() int {
	f.mu.RLock()
	defer f.mu.RUnlock()

	return f.buf.Len()
}
