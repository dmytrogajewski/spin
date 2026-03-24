// Package syncmap provides a generic thread-safe map with lifecycle support.
package syncmap

import "sync"

// Map is a generic thread-safe map. It is safe for concurrent use by
// multiple goroutines without additional locking. The zero value is not
// usable; create instances with [New].
type Map[K comparable, V any] struct {
	data   map[K]V
	mu     sync.RWMutex
	closed bool
}

// New creates a new empty Map.
func New[K comparable, V any]() *Map[K, V] {
	return &Map[K, V]{
		data: make(map[K]V),
	}
}

// Set stores a key-value pair. If the map is closed, Set is a no-op.
func (m *Map[K, V]) Set(key K, value V) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.closed {
		return
	}

	m.data[key] = value
}

// Get retrieves the value for key. The second return value reports
// whether the key was found.
func (m *Map[K, V]) Get(key K) (V, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	val, ok := m.data[key]

	return val, ok
}

// GetOrCreate returns the existing value for key if present. Otherwise
// it calls create, stores the result, and returns it. The create function
// is called under the write lock — it must not call back into the Map.
func (m *Map[K, V]) GetOrCreate(key K, create func() V) V {
	val, getErr := m.GetOrCreateErr(key, func() (V, error) {
		return create(), nil
	})
	if getErr != nil {
		var zero V

		return zero
	}

	return val
}

// GetOrCreateErr is like [GetOrCreate] but the create function may return an error.
// If the key already exists, returns the existing value with nil error.
// If create returns an error, the key is not stored and the error is returned.
func (m *Map[K, V]) GetOrCreateErr(key K, create func() (V, error)) (V, error) {
	// Fast path: read lock.
	m.mu.RLock()

	if val, ok := m.data[key]; ok {
		m.mu.RUnlock()

		return val, nil
	}

	m.mu.RUnlock()

	// Slow path: write lock with double-check.
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.closed {
		var zero V

		return zero, nil
	}

	if val, ok := m.data[key]; ok {
		return val, nil
	}

	val, err := create()
	if err != nil {
		return val, err
	}

	m.data[key] = val

	return val, nil
}

// SetIfAbsent atomically stores the key-value pair only if the key does
// not already exist. Returns true if the value was stored.
func (m *Map[K, V]) SetIfAbsent(key K, value V) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.closed {
		return false
	}

	if _, ok := m.data[key]; ok {
		return false
	}

	m.data[key] = value

	return true
}

// SetIfPresent atomically updates the value for key only if the key
// already exists. Returns true if the value was updated.
func (m *Map[K, V]) SetIfPresent(key K, value V) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.closed {
		return false
	}

	if _, ok := m.data[key]; !ok {
		return false
	}

	m.data[key] = value

	return true
}

// Values returns a snapshot of all values.
func (m *Map[K, V]) Values() []V {
	m.mu.RLock()
	defer m.mu.RUnlock()

	values := make([]V, 0, len(m.data))
	for _, v := range m.data {
		values = append(values, v)
	}

	return values
}

// Delete removes the key from the map.
func (m *Map[K, V]) Delete(key K) {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.data, key)
}

// Pop atomically removes and returns the value for key. The second return
// value reports whether the key was found.
func (m *Map[K, V]) Pop(key K) (V, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	val, ok := m.data[key]
	if ok {
		delete(m.data, key)
	}

	return val, ok
}

// Len returns the number of entries.
func (m *Map[K, V]) Len() int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return len(m.data)
}

// Keys returns a snapshot of all keys.
func (m *Map[K, V]) Keys() []K {
	m.mu.RLock()
	defer m.mu.RUnlock()

	keys := make([]K, 0, len(m.data))
	for k := range m.data {
		keys = append(keys, k)
	}

	return keys
}

// Range calls fn for each key-value pair. If fn returns false, iteration
// stops. The callback must not modify the Map.
func (m *Map[K, V]) Range(fn func(key K, value V) bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for k, v := range m.data {
		if !fn(k, v) {
			return
		}
	}
}

// Clear removes all entries from the map.
func (m *Map[K, V]) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.data = make(map[K]V)
}

// Close marks the map as closed and removes all entries. The optional
// cleanup function is called once for each remaining value before removal.
// Close is idempotent — subsequent calls are no-ops.
func (m *Map[K, V]) Close(cleanup func(V)) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.closed {
		return
	}

	m.closed = true

	if cleanup != nil {
		for _, v := range m.data {
			cleanup(v)
		}
	}

	m.data = make(map[K]V)
}
