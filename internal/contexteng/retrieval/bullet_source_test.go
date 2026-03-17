package retrieval

// Journey: specs/journeys/JOURNEY-7.5.md.

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dmytrogajewski/spin/internal/ace/bullet"
	"github.com/dmytrogajewski/spin/internal/ace/trajectory"
)

// TestBulletSource_Name verifies the source identifier.
// Kills mutant: wrong name would break pipeline source matching.
func TestBulletSource_Name(t *testing.T) {
	t.Parallel()

	src := NewBulletSource()
	assert.Equal(t, "ace_bullets", src.Name())
}

// TestBulletSource_NilTrajectory verifies nil trajectory returns empty.
// Kills mutant: nil trajectory dereference would panic.
func TestBulletSource_NilTrajectory(t *testing.T) {
	t.Parallel()

	src := NewBulletSource()
	frags, err := src.Retrieve(t.Context(), Request{})

	require.NoError(t, err)
	assert.Empty(t, frags)
}

// TestBulletSource_NoBullets verifies empty cache returns empty.
// Kills mutant: creating fragments from empty list would be wasteful.
func TestBulletSource_NoBullets(t *testing.T) {
	t.Parallel()

	trajCtx := trajectory.NewContext("test")
	src := NewBulletSource()

	frags, err := src.Retrieve(t.Context(), Request{
		TrajectoryCtx: trajCtx,
	})

	require.NoError(t, err)
	assert.Empty(t, frags)
}

// TestBulletSource_WithBullets verifies bullets are converted to fragments.
// Kills mutant: not extracting content would return empty fragments.
func TestBulletSource_WithBullets(t *testing.T) {
	t.Parallel()

	trajCtx := trajectory.NewContext("test")
	trajCtx.RecordRetrieval(
		trajectory.RetrievalEvent{Turn: 0},
		[]*bullet.Bullet{
			{ID: "B1", Content: "use structured logging"},
			{ID: "B2", Content: "prefer table-driven tests"},
		},
	)

	src := NewBulletSource()

	frags, err := src.Retrieve(t.Context(), Request{
		TrajectoryCtx: trajCtx,
	})

	require.NoError(t, err)
	require.Len(t, frags, 2)
	assert.Equal(t, "ace_bullets", frags[0].Source)
	assert.Equal(t, "use structured logging", frags[0].Content)
	assert.Equal(t, "prefer table-driven tests", frags[1].Content)
}

// TestBulletSource_ExpiredBullets verifies expired bullets are excluded.
// Kills mutant: ignoring TTL would return stale bullets.
func TestBulletSource_ExpiredBullets(t *testing.T) {
	t.Parallel()

	trajCtx := trajectory.NewContext("test")
	trajCtx.RecordRetrieval(
		trajectory.RetrievalEvent{Turn: 0},
		[]*bullet.Bullet{{ID: "B1", Content: "old bullet"}},
	)

	// Advance past TTL (default is 10).
	trajCtx.CurrentTurn = 15

	src := NewBulletSource()

	frags, err := src.Retrieve(t.Context(), Request{
		TrajectoryCtx: trajCtx,
	})

	require.NoError(t, err)
	assert.Empty(t, frags)
}

// TestBulletSource_MixedFreshAndExpired verifies only fresh bullets returned.
// Kills mutant: returning all bullets regardless of TTL would be incorrect.
func TestBulletSource_MixedFreshAndExpired(t *testing.T) {
	t.Parallel()

	trajCtx := trajectory.NewContext("test")
	trajCtx.RecordRetrieval(
		trajectory.RetrievalEvent{Turn: 0},
		[]*bullet.Bullet{{ID: "B1", Content: "old"}},
	)
	trajCtx.CurrentTurn = 15
	trajCtx.RecordRetrieval(
		trajectory.RetrievalEvent{Turn: 15},
		[]*bullet.Bullet{{ID: "B2", Content: "fresh"}},
	)

	src := NewBulletSource()

	frags, err := src.Retrieve(t.Context(), Request{
		TrajectoryCtx: trajCtx,
	})

	require.NoError(t, err)
	require.Len(t, frags, 1)
	assert.Equal(t, "fresh", frags[0].Content)
}
