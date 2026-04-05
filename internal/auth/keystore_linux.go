//go:build linux

package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/zalando/go-keyring"
)

// secretServiceTimeout is the maximum time to wait for D-Bus Secret Service availability check.
const secretServiceTimeout = 3 * time.Second

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
func (k *linuxKeystore) Get(ctx context.Context, key string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", fmt.Errorf("keyring get: %w", err)
	}

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
func (k *linuxKeystore) Set(ctx context.Context, key, value string) error {
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("keyring set: %w", err)
	}

	err := keyring.Set(serviceName, key, value)
	if err != nil {
		return fmt.Errorf("keyring set: %w", err)
	}

	return nil
}

// Delete removes a value from the keyring.
//
// This operation is idempotent - deleting a non-existent key succeeds.
func (k *linuxKeystore) Delete(ctx context.Context, key string) error {
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("keyring delete: %w", err)
	}

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
func (k *linuxKeystore) List(ctx context.Context) ([]string, error) {
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("keyring list: %w", err)
	}

	return nil, fmt.Errorf("list not supported on Linux Secret Service: %w", ErrNoKeystore)
}

// isSecretServiceAvailable checks if Secret Service is available.
//
// It attempts a simple operation to determine if the D-Bus service
// is accessible and working. A timeout prevents hanging when D-Bus
// is present but the Secret Service daemon is unresponsive.
func isSecretServiceAvailable() bool {
	resultCh := make(chan bool, 1)

	go func() {
		_, err := keyring.Get(serviceName, "spin-availability-check")
		resultCh <- err == nil || errors.Is(err, keyring.ErrNotFound)
	}()

	select {
	case available := <-resultCh:
		return available
	case <-time.After(secretServiceTimeout):
		return false
	}
}
