package main

// Journey: specs/journeys/JOURNEY-025-parent-shutdown-cancels-children.md.

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/dmytrogajewski/spin/internal/agent/child"
)

func TestReapParentOrphans_RemovesStalePid(t *testing.T) {
	base := t.TempDir()
	t.Setenv("XDG_RUNTIME_DIR", base)

	const dead = 2147483646
	require.NoError(t, child.WritePidFile(child.RuntimeDir(), dead))
	reapParentOrphans()

	_, err := os.Stat(child.PidPath(child.RuntimeDir(), dead))
	require.ErrorIs(t, err, os.ErrNotExist)
}
