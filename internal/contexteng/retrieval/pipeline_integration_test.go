package retrieval_test

// Journey: specs/journeys/JOURNEY-3.2.md.

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dmytrogajewski/spin/internal/ace/bullet"
	"github.com/dmytrogajewski/spin/internal/ace/trajectory"
	"github.com/dmytrogajewski/spin/internal/contexteng/retrieval"
)

// fakeSource is a simple Source implementation for integration tests.
type fakeSource struct {
	name      string
	fragments []retrieval.Fragment
	err       error
}

func (f *fakeSource) Name() string { return f.name }

func (f *fakeSource) Retrieve(
	_ context.Context, _ retrieval.Request,
) ([]retrieval.Fragment, error) {
	return f.fragments, f.err
}

// TestPipeline_AssemblesFragmentsFromMultipleSources verifies that a pipeline
// with multiple heterogeneous sources collects fragments from all of them
// into a single result, preserving source attribution.
func TestPipeline_AssemblesFragmentsFromMultipleSources(t *testing.T) {
	t.Parallel()

	srcA := &fakeSource{
		name: "config",
		fragments: []retrieval.Fragment{
			{Source: "config", Content: "max_retries=3"},
			{Source: "config", Content: "timeout=30s"},
		},
	}
	srcB := &fakeSource{
		name: "history",
		fragments: []retrieval.Fragment{
			{Source: "history", Content: "user asked about auth"},
		},
	}
	srcC := &fakeSource{
		name: "embeddings",
		fragments: []retrieval.Fragment{
			{Source: "embeddings", Content: "related function: validateToken"},
		},
	}

	p := retrieval.NewPipeline(srcA, srcB, srcC)
	result := p.Assemble(t.Context(), retrieval.Request{Turn: 1, Query: "how does auth work"})

	require.Len(t, result.Fragments, 4)

	// Verify fragments from all three sources are present.
	sources := make(map[string]int)
	for _, f := range result.Fragments {
		sources[f.Source]++
	}

	assert.Equal(t, 2, sources["config"])
	assert.Equal(t, 1, sources["history"])
	assert.Equal(t, 1, sources["embeddings"])

	// Verify ordering: srcA fragments first, then srcB, then srcC.
	assert.Equal(t, "max_retries=3", result.Fragments[0].Content)
	assert.Equal(t, "timeout=30s", result.Fragments[1].Content)
	assert.Equal(t, "user asked about auth", result.Fragments[2].Content)
	assert.Equal(t, "related function: validateToken", result.Fragments[3].Content)
}

// TestPipeline_BulletSourceIntegration verifies that a real BulletSource
// wired into the pipeline produces fragments whose content matches the
// active bullets from the trajectory context.
func TestPipeline_BulletSourceIntegration(t *testing.T) {
	t.Parallel()

	// Set up trajectory context with active bullets.
	trajCtx := trajectory.NewContext("integration test")
	trajCtx.CurrentTurn = 3
	trajCtx.RecordRetrieval(
		trajectory.RetrievalEvent{Turn: 2},
		[]*bullet.Bullet{
			{ID: "B1", Content: "always validate input before DB writes"},
			{ID: "B2", Content: "use structured logging for errors"},
		},
	)

	bulletSrc := retrieval.NewBulletSource()
	p := retrieval.NewPipeline(bulletSrc)

	result := p.Assemble(t.Context(), retrieval.Request{
		Turn:          3,
		Query:         "write a new endpoint",
		TrajectoryCtx: trajCtx,
	})

	require.Len(t, result.Fragments, 2)

	// Both fragments should come from the bullet source.
	for _, f := range result.Fragments {
		assert.Equal(t, "ace_bullets", f.Source)
	}

	// Collect content to verify bullet text is present (order may be sorted by ID).
	contents := make(map[string]bool)
	for _, f := range result.Fragments {
		contents[f.Content] = true
	}

	assert.True(t, contents["always validate input before DB writes"],
		"expected bullet B1 content in fragments")
	assert.True(t, contents["use structured logging for errors"],
		"expected bullet B2 content in fragments")
}

// TestPipeline_EmptySources_ReturnsEmptyFragments verifies that a pipeline
// with sources that all return no fragments does not crash and produces
// an empty result.
func TestPipeline_EmptySources_ReturnsEmptyFragments(t *testing.T) {
	t.Parallel()

	emptySrc := &fakeSource{
		name:      "empty",
		fragments: nil,
	}

	// BulletSource with no trajectory context returns nothing.
	bulletSrc := retrieval.NewBulletSource()

	p := retrieval.NewPipeline(emptySrc, bulletSrc)
	result := p.Assemble(t.Context(), retrieval.Request{Turn: 1, Query: "anything"})

	require.Empty(t, result.Fragments)
}
