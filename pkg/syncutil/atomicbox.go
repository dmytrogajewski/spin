// Package syncutil provides generic thread-safety primitives
// that eliminate common RWMutex boilerplate across the codebase.
package syncutil

import "sync"

// AtomicBox is a generic RWMutex-protected value wrapper.
// It provides safe concurrent read/write access to a value of type T.
type AtomicBox[T any] struct {
	mu    sync.RWMutex
	value T
}

// NewAtomicBox creates a new AtomicBox with the given initial value.
func NewAtomicBox[T any](initial T) *AtomicBox[T] {
	return &AtomicBox[T]{value: initial}
}

// Read returns the current value under a read lock.
func (b *AtomicBox[T]) Read() T {
	b.mu.RLock()
	defer b.mu.RUnlock()

	return b.value
}

// Write sets the value under a write lock.
func (b *AtomicBox[T]) Write(val T) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.value = val
}

// Update applies the function to the current value under a write lock.
// The function receives the current value and returns the new value.
func (b *AtomicBox[T]) Update(fn func(T) T) T {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.value = fn(b.value)

	return b.value
}

// ReadWith executes the function under a read lock and returns its result.
func (b *AtomicBox[T]) ReadWith(fn func(T) T) T {
	b.mu.RLock()
	defer b.mu.RUnlock()

	return fn(b.value)
}
