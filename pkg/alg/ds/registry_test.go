package ds

// Journey: specs/journeys/JOURNEY-R-REF-25.md.

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRegistry_RegisterAndGet(t *testing.T) {
	t.Parallel()

	reg := NewRegistry[string, int](0)
	reg.Register("a", 1)

	require.Equal(t, 1, reg.Lookup("a"))
	require.Equal(t, 0, reg.Lookup("missing")) // Fallback.
	require.Equal(t, 1, reg.Count())
}

func TestRegistry_Fallback(t *testing.T) {
	t.Parallel()

	fallback := "default"
	reg := NewRegistry[int, string](fallback)

	require.Equal(t, fallback, reg.Lookup(42))
}

func TestRegistry_Overwrite(t *testing.T) {
	t.Parallel()

	reg := NewRegistry[string, int](0)
	reg.Register("key", 1)
	reg.Register("key", 2)

	require.Equal(t, 2, reg.Lookup("key"))
	require.Equal(t, 1, reg.Count())
}
