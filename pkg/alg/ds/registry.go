package ds

import "sync"

// Registry is a thread-safe type→strategy map.
// Register strategies at startup, then look them up concurrently.
type Registry[Key comparable, Val any] struct {
	mu       sync.RWMutex
	entries  map[Key]Val
	fallback Val
}

// NewRegistry creates a new empty registry with an optional fallback value.
func NewRegistry[Key comparable, Val any](fallback Val) *Registry[Key, Val] {
	return &Registry[Key, Val]{
		entries:  make(map[Key]Val),
		fallback: fallback,
	}
}

// Register adds a value for the given key.
func (r *Registry[Key, Val]) Register(key Key, val Val) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.entries[key] = val
}

// Lookup returns the value for key, or the fallback if not found.
func (r *Registry[Key, Val]) Lookup(key Key) Val {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if val, ok := r.entries[key]; ok {
		return val
	}

	return r.fallback
}

// Count returns the number of registered entries.
func (r *Registry[Key, Val]) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return len(r.entries)
}
