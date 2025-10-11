package testkit

import (
	"bytes"
	"regexp"
	"strings"
	"sync"
	"time"
)

// FakeWriter captures ANSI output for testing.
// Thread-safe implementation that buffers all writes and provides
// helpers for asserting output content.
type FakeWriter struct {
	buf     *bytes.Buffer
	mu      sync.Mutex
	cond    *sync.Cond // For WaitForContent blocking
	ansiRe  *regexp.Regexp
}

// NewFakeWriter creates a new fake writer for testing.
func NewFakeWriter() *FakeWriter {
	f := &FakeWriter{
		buf:    &bytes.Buffer{},
		ansiRe: regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`),
	}
	f.cond = sync.NewCond(&f.mu)
	return f
}

// Write implements io.Writer, appending data to the buffer.
func (f *FakeWriter) Write(p []byte) (int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	n, err := f.buf.Write(p)

	// Signal waiting goroutines that new content arrived
	f.cond.Broadcast()

	return n, err
}

// Snapshot returns the current buffer content as a string.
// Thread-safe.
func (f *FakeWriter) Snapshot() string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.buf.String()
}

// Reset clears the buffer.
// Thread-safe.
func (f *FakeWriter) Reset() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.buf.Reset()
}

// ContainsANSI checks if the output contains a specific ANSI escape sequence.
// Thread-safe.
func (f *FakeWriter) ContainsANSI(seq string) bool {
	return strings.Contains(f.Snapshot(), seq)
}

// WaitForContent blocks until the given substring appears in the output
// or the timeout expires. Returns true if content was found, false on timeout.
func (f *FakeWriter) WaitForContent(s string, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)

	f.mu.Lock()
	defer f.mu.Unlock()

	for {
		// Check if content already present
		if strings.Contains(f.buf.String(), s) {
			return true
		}

		// Check timeout
		remaining := time.Until(deadline)
		if remaining <= 0 {
			return false
		}

		// Wait with timeout using a goroutine
		// We can't use cond.Wait directly with timeout, so use channel approach
		waitDone := make(chan struct{})
		go func() {
			time.Sleep(remaining)
			f.cond.Broadcast() // Wake up if timeout
			close(waitDone)
		}()

		// Wait for signal (will be woken by Write or timeout)
		f.cond.Wait() // This unlocks mu, waits, then relocks mu

		// Check again after wake
		if strings.Contains(f.buf.String(), s) {
			return true
		}

		// Check if we timed out
		if time.Now().After(deadline) {
			return false
		}
	}
}

// Lines returns the output split by newlines.
// Empty lines are included. Thread-safe.
func (f *FakeWriter) Lines() []string {
	content := f.Snapshot()
	if content == "" {
		return []string{}
	}
	return strings.Split(content, "\n")
}

// StripANSI returns the output with all ANSI escape sequences removed.
// Thread-safe.
func (f *FakeWriter) StripANSI() string {
	content := f.Snapshot()
	return f.ansiRe.ReplaceAllString(content, "")
}

// Len returns the current buffer length in bytes.
// Thread-safe.
func (f *FakeWriter) Len() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.buf.Len()
}
