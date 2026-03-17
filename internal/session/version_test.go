package session

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// Journey: specs/journeys/JOURNEY-1.6.md.

// TestMigrateVersion_ZeroDefaultsToV1 tests that MigrateVersion sets
// version to CurrentSchemaVersion when it is zero (old format).
// Kills mutant: skipping migration would leave version at 0.
func TestMigrateVersion_ZeroDefaultsToV1(t *testing.T) {
	t.Parallel()

	sess := &Session{Version: 0}
	sess.MigrateVersion()

	assert.Equal(t, CurrentSchemaVersion, sess.Version)
}

// TestMigrateVersion_PreservesExistingVersion tests that MigrateVersion
// does not overwrite a non-zero version.
// Kills mutant: always overwriting would change existing versions.
func TestMigrateVersion_PreservesExistingVersion(t *testing.T) {
	t.Parallel()

	const existingVersion = 2

	sess := &Session{Version: existingVersion}
	sess.MigrateVersion()

	assert.Equal(t, existingVersion, sess.Version)
}

// TestNewSession_SetsVersion tests that NewSession sets the current
// schema version, ensuring new sessions always have a version.
// Kills mutant: removing version assignment would leave it at 0.
func TestNewSession_SetsVersion(t *testing.T) {
	t.Parallel()

	sess := NewSession(testWorkDir)

	assert.Equal(t, CurrentSchemaVersion, sess.Version)
}
