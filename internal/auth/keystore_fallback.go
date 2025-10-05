//go:build !linux && !darwin && !windows

package auth

// newPlatformKeystore creates a fallback keystore for unsupported platforms.
//
// This implementation uses in-memory storage and is suitable for:
//   - BSD systems
//   - Development/testing environments
//   - Any platform without native secure storage
func newPlatformKeystore() Keystore {
	return newMemoryKeystore()
}
