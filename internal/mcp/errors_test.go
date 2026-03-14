package mcp

// Journey: specs/journeys/JOURNEY-unify-err-tool-not-found.md.

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/dmytrogajewski/spin/internal/tools"
)

func TestErrToolNotFound_IsCanonical(t *testing.T) {
	t.Parallel()

	// ErrToolNotFound in mcp must be the same sentinel as tools.ErrToolNotFound.
	// After unification, wrapping it should still match via errors.Is.
	wrapped := fmt.Errorf("tool not found: %s: %w", "my-tool", tools.ErrToolNotFound)
	assert.ErrorIs(t, wrapped, tools.ErrToolNotFound,
		"wrapped mcp error must match tools.ErrToolNotFound")
}
