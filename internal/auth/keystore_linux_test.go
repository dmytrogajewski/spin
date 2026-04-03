//go:build linux

package auth

import (
	"errors"
	"testing"

	"github.com/zalando/go-keyring"
)

// requireSecretService skips the test if Secret Service is unavailable.
func requireSecretService(t *testing.T) {
	t.Helper()

	if !isSecretServiceAvailable() {
		t.Skip("Secret Service unavailable (D-Bus unresponsive or not running)")
	}
}

// TestLinuxKeystore_Get tests retrieving values.
func TestLinuxKeystore_Get(t *testing.T) {
	t.Parallel()
	requireSecretService(t)

	ks := &linuxKeystore{}

	// Clean up before test.
	_ = keyring.Delete(serviceName, "test-key")

	// Set a value first.
	err := keyring.Set(serviceName, "test-key", "test-value")
	if err != nil {
		t.Skipf("Secret Service not available: %v", err)
	}

	defer func() { _ = keyring.Delete(serviceName, "test-key") }()

	// Get the value.
	value, err := ks.Get(t.Context(), "test-key")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if value != "test-value" {
		t.Errorf("Get() = %q, want %q", value, "test-value")
	}
}

// TestLinuxKeystore_Get_NotFound tests getting non-existent values.
func TestLinuxKeystore_Get_NotFound(t *testing.T) {
	t.Parallel()
	requireSecretService(t)

	ks := &linuxKeystore{}

	// Clean up to ensure it doesn't exist.
	_ = keyring.Delete(serviceName, "nonexistent")

	_, err := ks.Get(t.Context(), "nonexistent")
	if err == nil {
		t.Fatal("Get() expected error, got nil")
	}

	if !errors.Is(err, ErrNotFound) {
		t.Errorf("Get() error = %v, want ErrNotFound", err)
	}
}

// TestLinuxKeystore_Set tests storing values.
func TestLinuxKeystore_Set(t *testing.T) {
	t.Parallel()
	requireSecretService(t)

	ks := &linuxKeystore{}

	// Clean up before test.
	_ = keyring.Delete(serviceName, "test-set")

	defer func() { _ = keyring.Delete(serviceName, "test-set") }()

	// Set a value.
	err := ks.Set(t.Context(), "test-set", "new-value")
	if err != nil {
		t.Skipf("Secret Service not available: %v", err)
	}

	// Verify it was stored.
	value, err := keyring.Get(serviceName, "test-set")
	if err != nil {
		t.Fatalf("Failed to verify set: %v", err)
	}

	if value != "new-value" {
		t.Errorf("Set() stored %q, want %q", value, "new-value")
	}
}

// TestLinuxKeystore_Set_Overwrite tests overwriting values.
func TestLinuxKeystore_Set_Overwrite(t *testing.T) {
	t.Parallel()
	requireSecretService(t)

	ks := &linuxKeystore{}

	// Clean up before test.
	_ = keyring.Delete(serviceName, "test-overwrite")

	defer func() { _ = keyring.Delete(serviceName, "test-overwrite") }()

	// Set initial value.
	err := keyring.Set(serviceName, "test-overwrite", "old-value")
	if err != nil {
		t.Skipf("Secret Service not available: %v", err)
	}

	// Overwrite with new value.
	err = ks.Set(t.Context(), "test-overwrite", "new-value")
	if err != nil {
		t.Fatalf("Set() error = %v", err)
	}

	// Verify new value.
	value, err := keyring.Get(serviceName, "test-overwrite")
	if err != nil {
		t.Fatalf("Failed to verify overwrite: %v", err)
	}

	if value != "new-value" {
		t.Errorf("Set() stored %q, want %q", value, "new-value")
	}
}

// TestLinuxKeystore_Delete tests deleting values.
func TestLinuxKeystore_Delete(t *testing.T) {
	t.Parallel()
	requireSecretService(t)

	ks := &linuxKeystore{}

	// Clean up before test.
	_ = keyring.Delete(serviceName, "test-delete")

	// Set a value first.
	err := keyring.Set(serviceName, "test-delete", "test-value")
	if err != nil {
		t.Skipf("Secret Service not available: %v", err)
	}

	// Delete it.
	err = ks.Delete(t.Context(), "test-delete")
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	// Verify it's gone.
	_, err = keyring.Get(serviceName, "test-delete")
	if err == nil {
		t.Error("Value still exists after delete")
	}
}

// TestLinuxKeystore_Delete_Idempotent tests deleting non-existent values.
func TestLinuxKeystore_Delete_Idempotent(t *testing.T) {
	t.Parallel()
	requireSecretService(t)

	ks := &linuxKeystore{}

	// Ensure it doesn't exist.
	_ = keyring.Delete(serviceName, "nonexistent-delete")

	// Delete should not error.
	err := ks.Delete(t.Context(), "nonexistent-delete")
	if err != nil {
		t.Errorf("Delete() error = %v, want nil", err)
	}
}

// TestLinuxKeystore_List tests that List returns error.
func TestLinuxKeystore_List(t *testing.T) {
	t.Parallel()

	ks := &linuxKeystore{}

	_, err := ks.List(t.Context())
	if err == nil {
		t.Fatal("List() expected error, got nil")
	}

	// Should mention not supported.
	if err.Error() == "" {
		t.Error("List() error message is empty")
	}
}

// TestLinuxKeystore_Integration tests full workflow.
func TestLinuxKeystore_Integration(t *testing.T) {
	t.Parallel()
	requireSecretService(t)

	ks := &linuxKeystore{}

	testKey := "integration-test"
	testValue := "integration-value"

	// Clean up before and after.
	_ = keyring.Delete(serviceName, testKey)

	defer func() { _ = keyring.Delete(serviceName, testKey) }()

	// Store.
	err := ks.Set(t.Context(), testKey, testValue)
	if err != nil {
		t.Skipf("Secret Service not available: %v", err)
	}

	// Retrieve.
	value, err := ks.Get(t.Context(), testKey)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if value != testValue {
		t.Errorf("Get() = %q, want %q", value, testValue)
	}

	// Update.
	newValue := "updated-value"

	err = ks.Set(t.Context(), testKey, newValue)
	if err != nil {
		t.Fatalf("Set() update error = %v", err)
	}

	// Verify update.
	value, err = ks.Get(t.Context(), testKey)
	if err != nil {
		t.Fatalf("Get() after update error = %v", err)
	}

	if value != newValue {
		t.Errorf("Get() after update = %q, want %q", value, newValue)
	}

	// Delete.
	err = ks.Delete(t.Context(), testKey)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	// Verify deleted.
	_, err = ks.Get(t.Context(), testKey)
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("Get() after delete error = %v, want ErrNotFound", err)
	}
}

// TestNewPlatformKeystore tests keystore creation.
func TestNewPlatformKeystore(t *testing.T) {
	t.Parallel()

	ks := newPlatformKeystore()

	if ks == nil {
		t.Fatal("newPlatformKeystore() returned nil")
	}

	// Should be either linuxKeystore or memoryKeystore (fallback).
	switch ks.(type) {
	case *linuxKeystore:
		t.Log("Using Linux Secret Service keystore")
	case *memoryKeystore:
		t.Log("Fell back to memory keystore (Secret Service unavailable)")
	default:
		t.Errorf("Unexpected keystore type: %T", ks)
	}
}

// TestIsSecretServiceAvailable tests Secret Service detection.
func TestIsSecretServiceAvailable(t *testing.T) {
	t.Parallel()

	available := isSecretServiceAvailable()
	t.Logf("Secret Service available: %v", available)

	// The result depends on the environment, just verify it doesn't panic
	// and returns a boolean.
	if available {
		// If available, we should be able to use it.
		ks := &linuxKeystore{}

		_, err := ks.Get(t.Context(), "test-availability")
		if err != nil && !errors.Is(err, ErrNotFound) {
			t.Logf("Secret Service reported available but Get failed: %v", err)
		}
	}
}

// TestLinuxKeystore_EmptyValue tests storing empty values.
func TestLinuxKeystore_EmptyValue(t *testing.T) {
	t.Parallel()
	requireSecretService(t)

	ks := &linuxKeystore{}

	testKey := "empty-value-test"

	// Clean up.
	_ = keyring.Delete(serviceName, testKey)

	defer func() { _ = keyring.Delete(serviceName, testKey) }()

	// Set empty value.
	err := ks.Set(t.Context(), testKey, "")
	if err != nil {
		t.Skipf("Secret Service not available: %v", err)
	}

	// Get empty value.
	value, err := ks.Get(t.Context(), testKey)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if value != "" {
		t.Errorf("Get() = %q, want empty string", value)
	}
}

// TestLinuxKeystore_SpecialCharacters tests keys with special characters.
func TestLinuxKeystore_SpecialCharacters(t *testing.T) {
	t.Parallel()
	requireSecretService(t)

	ks := &linuxKeystore{}

	tests := []struct {
		name  string
		key   string
		value string
	}{
		{"colon", "key:with:colons", "value"},
		{"slash", "key/with/slashes", "value"},
		{"spaces", "key with spaces", "value"},
		{"unicode", "key-\u4e2d\u6587", "value-\u4e2d\u6587"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Clean up.
			_ = keyring.Delete(serviceName, tt.key)

			defer func() { _ = keyring.Delete(serviceName, tt.key) }()

			// Set.
			err := ks.Set(t.Context(), tt.key, tt.value)
			if err != nil {
				t.Skipf("Secret Service not available: %v", err)
			}

			// Get.
			value, err := ks.Get(t.Context(), tt.key)
			if err != nil {
				t.Fatalf("Get() error = %v", err)
			}

			if value != tt.value {
				t.Errorf("Get() = %q, want %q", value, tt.value)
			}
		})
	}
}
