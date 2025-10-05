package auth

import "errors"

// Keystore provides platform-agnostic credential storage.
//
// Implementations may use platform-specific secure storage:
//   - macOS: Keychain
//   - Linux: Secret Service (libsecret)
//   - Windows: Credential Manager
//
// Implementations must be safe for concurrent use.
type Keystore interface {
	// Get retrieves a value by key.
	//
	// Returns ErrNotFound if the key doesn't exist.
	Get(key string) (string, error)

	// Set stores a key-value pair.
	//
	// If the key already exists, the value is overwritten.
	Set(key, value string) error

	// Delete removes a key-value pair.
	//
	// This operation is idempotent - deleting a non-existent key succeeds.
	Delete(key string) error

	// List returns all stored keys.
	//
	// Returns an empty slice if no keys are stored.
	List() ([]string, error)
}

var (
	// ErrNotFound indicates the requested key was not found
	ErrNotFound = errors.New("key not found")

	// ErrNoKeystore indicates no keystore is available
	ErrNoKeystore = errors.New("no keystore available")
)

// NewKeystore creates a new platform-specific keystore.
//
// On macOS, this returns a Keychain-backed keystore.
// On Linux, this returns a Secret Service-backed keystore.
// On Windows, this returns a Credential Manager-backed keystore.
// On unsupported platforms, this returns an in-memory keystore.
//
// The returned keystore is safe for concurrent use.
func NewKeystore() Keystore {
	return newPlatformKeystore()
}

// newPlatformKeystore is implemented in platform-specific files:
//   - keystore_linux.go (Linux)
//   - keystore_darwin.go (macOS)
//   - keystore_windows.go (Windows)
//   - keystore_fallback.go (other platforms)
