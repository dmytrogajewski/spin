package auth

import "sync"

// memoryKeystore implements Keystore using in-memory storage.
// This is used as a fallback when platform-specific keystores are unavailable.
type memoryKeystore struct {
	data map[string]string
	mu   sync.RWMutex
}

// newMemoryKeystore creates a new in-memory keystore.
func newMemoryKeystore() Keystore {
	return &memoryKeystore{
		data: make(map[string]string),
	}
}

// Get retrieves a value by key.
func (m *memoryKeystore) Get(key string) (string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	value, exists := m.data[key]
	if !exists {
		return "", ErrNotFound
	}
	return value, nil
}

// Set stores a key-value pair.
func (m *memoryKeystore) Set(key, value string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.data[key] = value
	return nil
}

// Delete removes a key-value pair.
func (m *memoryKeystore) Delete(key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.data, key)
	return nil
}

// List returns all stored keys.
func (m *memoryKeystore) List() ([]string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	keys := make([]string, 0, len(m.data))
	for k := range m.data {
		keys = append(keys, k)
	}
	return keys, nil
}
