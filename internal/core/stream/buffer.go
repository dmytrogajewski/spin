package stream

import (
	"sync"
)

const (
	// DefaultBufferCapacity is the default buffer size (4KB)
	DefaultBufferCapacity = 4096
	// MinBufferCapacity is the minimum allowed buffer size
	MinBufferCapacity = 256
	// MaxBufferCapacity is the maximum allowed buffer size (1MB)
	MaxBufferCapacity = 1 << 20
)

// Buffer provides a thread-safe ring buffer for stream data.
// It efficiently manages fixed-capacity buffering with wrap-around support.
type Buffer struct {
	data     []byte
	capacity int
	size     int
	readPos  int
	writePos int
	mu       sync.RWMutex
	full     bool
}

// NewBuffer creates a new buffer with specified capacity.
// If capacity is <= 0, DefaultBufferCapacity is used.
// Capacity is clamped to MaxBufferCapacity if exceeded.
func NewBuffer(capacity int) *Buffer {
	if capacity <= 0 {
		capacity = DefaultBufferCapacity
	}
	if capacity > MaxBufferCapacity {
		capacity = MaxBufferCapacity
	}

	return &Buffer{
		data:     make([]byte, capacity),
		capacity: capacity,
		size:     0,
		readPos:  0,
		writePos: 0,
		full:     false,
	}
}

// Write writes data to the buffer.
// Returns number of bytes written and ErrBufferFull if buffer cannot accommodate all data.
// Partial writes are performed when buffer space is insufficient.
func (b *Buffer) Write(data []byte) (int, error) {
	if len(data) == 0 {
		return 0, nil
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	if b.full {
		return 0, ErrBufferFull
	}

	// Calculate available space
	available := b.capacity - b.size
	toWrite := len(data)
	if toWrite > available {
		toWrite = available
	}

	// Write data (may wrap around)
	written := 0
	for written < toWrite {
		// Write until end of buffer or all data written
		chunk := toWrite - written
		spaceToEnd := b.capacity - b.writePos
		if chunk > spaceToEnd {
			chunk = spaceToEnd
		}

		copy(b.data[b.writePos:b.writePos+chunk], data[written:written+chunk])
		written += chunk
		b.writePos = (b.writePos + chunk) % b.capacity
		b.size += chunk
	}

	// Check if buffer is now full
	if b.size == b.capacity {
		b.full = true
	}

	// Return error if not all data was written
	if written < len(data) {
		return written, ErrBufferFull
	}

	return written, nil
}

// Read reads up to len(p) bytes from buffer.
// Returns number of bytes read and ErrBufferEmpty if buffer is empty.
func (b *Buffer) Read(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	if b.size == 0 && !b.full {
		return 0, ErrBufferEmpty
	}

	// Calculate how much to read
	toRead := len(p)
	if toRead > b.size {
		toRead = b.size
	}

	// Read data (may wrap around)
	read := 0
	for read < toRead {
		// Read until end of buffer or all data read
		chunk := toRead - read
		dataToEnd := b.capacity - b.readPos
		if chunk > dataToEnd {
			chunk = dataToEnd
		}

		copy(p[read:read+chunk], b.data[b.readPos:b.readPos+chunk])
		read += chunk
		b.readPos = (b.readPos + chunk) % b.capacity
		b.size -= chunk
	}

	// Buffer is no longer full after reading
	b.full = false

	return read, nil
}

// Available returns number of bytes available to read.
func (b *Buffer) Available() int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.size
}

// Capacity returns total buffer capacity.
func (b *Buffer) Capacity() int {
	return b.capacity
}

// Reset clears the buffer.
func (b *Buffer) Reset() {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.size = 0
	b.readPos = 0
	b.writePos = 0
	b.full = false
}

// IsFull returns true if buffer is at capacity.
func (b *Buffer) IsFull() bool {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.full
}

// IsEmpty returns true if buffer is empty.
func (b *Buffer) IsEmpty() bool {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.size == 0 && !b.full
}
