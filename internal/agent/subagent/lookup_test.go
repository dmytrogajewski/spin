package subagent

// Journey: specs/journeys/JOURNEY-017-local-a2a-server-process.md.

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLookup_Explorer(t *testing.T) {
	t.Parallel()

	spec, err := Lookup(NameExplorer)
	require.NoError(t, err)
	require.Equal(t, NameExplorer, spec.Name)
	require.True(t, spec.HasTool("read_file"))
}

func TestLookup_Unknown(t *testing.T) {
	t.Parallel()

	_, err := Lookup("nope")
	require.ErrorIs(t, err, ErrSpecNotFound)
}
