package testkit

import (
	"bytes"
	"sync"
)

// SafeBuffer is a thread-safe wrapper around bytes.Buffer.
//
// bytes.Buffer is NOT thread-safe for concurrent writes and reads.
// This wrapper adds mutex protection for testing scenarios where
// multiple goroutines write output concurrently.
type SafeBuffer struct {
	buf *bytes.Buffer
	mu  sync.RWMutex
}

// NewSafeBuffer creates a new thread-safe buffer.
func NewSafeBuffer() *SafeBuffer {
	return &SafeBuffer{
		buf: &bytes.Buffer{},
	}
}

// Write writes to the buffer (thread-safe).
func (s *SafeBuffer) Write(p []byte) (n int, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.buf.Write(p)
}

// WriteString writes a string to the buffer (thread-safe).
func (s *SafeBuffer) WriteString(str string) (n int, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.buf.WriteString(str)
}

// String returns the buffer contents (thread-safe).
func (s *SafeBuffer) String() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.buf.String()
}

// Reset clears the buffer (thread-safe).
func (s *SafeBuffer) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.buf.Reset()
}

// Len returns the buffer length (thread-safe).
func (s *SafeBuffer) Len() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.buf.Len()
}
