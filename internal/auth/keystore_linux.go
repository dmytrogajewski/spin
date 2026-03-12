//go:build linux

package auth

import (
	"errors"
	"fmt"

	"github.com/zalando/go-keyring"
)

const (
	// serviceName is the service name used for keyring operations.
	serviceName = "spin"
)

// linuxKeystore implements Keystore using Linux Secret Service.
//
// It uses the freedesktop.org Secret Service API via go-keyring,
// which supports GNOME Keyring, KWallet, and other compatible backends.
type linuxKeystore struct {
	// Empty - go-keyring is stateless.
}

// newPlatformKeystore creates a platform-specific keystore for Linux.
//
// It tests if Secret Service is available and falls back to memory storage
// if the service is unavailable (e.g., headless systems, missing D-Bus).
func newPlatformKeystore() Keystore {
	if !isSecretServiceAvailable() {
		// Fallback to memory keystore when Secret Service is unavailable.
		return newMemoryKeystore()
	}

	return &linuxKeystore{}
}

// Get retrieves a value from the keyring.
//
// Returns ErrNotFound if the key doesn't exist.
func (k *linuxKeystore) Get(key string) (string, error) {
	value, err := keyring.Get(serviceName, key)
	if err != nil {
		if errors.Is(err, keyring.ErrNotFound) {
			return "", ErrNotFound
		}

		return "", fmt.Errorf("keyring get: %w", err)
	}

	return value, nil
}

// Set stores a value in the keyring.
//
// If the key already exists, it is overwritten.
func (k *linuxKeystore) Set(key, value string) error {
	err := keyring.Set(serviceName, key, value)
	if err != nil {
		return fmt.Errorf("keyring set: %w", err)
	}

	return nil
}

// Delete removes a value from the keyring.
//
// This operation is idempotent - deleting a non-existent key succeeds.
func (k *linuxKeystore) Delete(key string) error {
	err := keyring.Delete(serviceName, key)
	if err != nil && !errors.Is(err, keyring.ErrNotFound) {
		return fmt.Errorf("keyring delete: %w", err)
	}

	return nil
}

// List is not supported by the Secret Service API.
//
// The freedesktop.org Secret Service specification does not provide
// a way to list all keys for a service. Use specific key names instead.
func (k *linuxKeystore) List() ([]string, error) {
	return nil, fmt.Errorf("list not supported on Linux Secret Service: %w", ErrNoKeystore)
}

// isSecretServiceAvailable checks if Secret Service is available.
//
// It attempts a simple operation to determine if the D-Bus service
// is accessible and working.
func isSecretServiceAvailable() bool {
	// Try to get a non-existent key
	// If we get ErrNotFound, the service is available
	// If we get any other error, assume unavailable.
	_, err := keyring.Get(serviceName, "spin-availability-check")
	if err != nil && !errors.Is(err, keyring.ErrNotFound) {
		return false
	}

	return true
}
