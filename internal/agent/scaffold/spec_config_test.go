package scaffold

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// Journey: specs/journeys/JOURNEY-1.2.md.

// TestThinkingLevel_Constants tests that ThinkingLevel constants have expected values.
// Kills mutant: changing constant order would make this test fail.
func TestThinkingLevel_Constants(t *testing.T) {
	t.Parallel()

	assert.Equal(t, ThinkingOff, ThinkingLevel(0))
	assert.Equal(t, ThinkingLow, ThinkingLevel(1))
	assert.Equal(t, ThinkingHigh, ThinkingLevel(2))
}
