package main

// Journey: specs/journeys/JOURNEY-007-finish-the-hook-runner-contract.md.

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/dmytrogajewski/spin/internal/safety/hooks"
)

func TestACPHooksGlobalDir_Expanded(t *testing.T) {
	t.Parallel()

	dir := acpHooksGlobalDir()
	require.Equal(t, hooks.DefaultGlobalDir(), dir)
	require.NotContains(t, dir, "~", "ACP builder must pass an expanded global dir")

	home, err := os.UserHomeDir()
	require.NoError(t, err)
	require.Equal(t, filepath.Join(home, ".spin", "hooks"), dir)
}
