package syncutil

import "sync"

// LazyInit provides thread-safe lazy initialization of a value.
// The init function is called exactly once, on the first call to Get.
type LazyInit[T any] struct {
	once sync.Once
	init func() (T, error)
	val  T
	err  error
}

// NewLazyInit creates a new LazyInit with the given initializer function.
func NewLazyInit[T any](init func() (T, error)) *LazyInit[T] {
	return &LazyInit[T]{init: init}
}

// Get returns the lazily initialized value. The init function is called
// at most once. Subsequent calls return the cached result.
func (l *LazyInit[T]) Get() (T, error) {
	l.once.Do(func() {
		l.val, l.err = l.init()
	})

	return l.val, l.err
}
