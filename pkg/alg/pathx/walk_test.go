package pathx

// Journey: specs/journeys/JOURNEY-R6.md.

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// sentinelFile is the marker file used in walk-up-find tests.
const sentinelFile = ".marker"

// dirPermissions is the permission mode for test directories.
const dirPermissions = 0o750

// filePermissions is the permission mode for test files.
const filePermissions = 0o600

func TestWalkUpFind(t *testing.T) {
	t.Parallel()

	t.Run("finds_marker_in_parent", func(t *testing.T) {
		t.Parallel()

		root := t.TempDir()
		child := filepath.Join(root, "a", "b")
		require.NoError(t, os.MkdirAll(child, dirPermissions))
		require.NoError(t, os.WriteFile(filepath.Join(root, sentinelFile), nil, filePermissions))

		hasSentinel := func(dir string) bool {
			_, err := os.Stat(filepath.Join(dir, sentinelFile))

			return err == nil
		}

		got, err := WalkUpFind(context.Background(), child, hasSentinel)
		require.NoError(t, err)
		require.Equal(t, root, got)
	})

	t.Run("finds_marker_in_start_dir", func(t *testing.T) {
		t.Parallel()

		root := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(root, sentinelFile), nil, filePermissions))

		hasSentinel := func(dir string) bool {
			_, err := os.Stat(filepath.Join(dir, sentinelFile))

			return err == nil
		}

		got, err := WalkUpFind(context.Background(), root, hasSentinel)
		require.NoError(t, err)
		require.Equal(t, root, got)
	})

	t.Run("not_found_returns_error", func(t *testing.T) {
		t.Parallel()

		root := t.TempDir()
		child := filepath.Join(root, "deep", "nested")
		require.NoError(t, os.MkdirAll(child, dirPermissions))

		neverMatch := func(_ string) bool { return false }

		_, err := WalkUpFind(context.Background(), child, neverMatch)
		require.ErrorIs(t, err, ErrNotFound)
	})

	t.Run("context_cancellation", func(t *testing.T) {
		t.Parallel()

		root := t.TempDir()
		child := filepath.Join(root, "a", "b", "c")
		require.NoError(t, os.MkdirAll(child, dirPermissions))

		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately.

		neverMatch := func(_ string) bool { return false }

		_, err := WalkUpFind(ctx, child, neverMatch)
		require.ErrorIs(t, err, context.Canceled)
	})
}
