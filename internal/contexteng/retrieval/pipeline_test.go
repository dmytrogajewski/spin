package retrieval

// Journey: specs/journeys/JOURNEY-7.5.md.

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testSourceA = "source_a"
	testSourceB = "source_b"
)

// errSourceFailed is a sentinel error for test source failures.
var errSourceFailed = errors.New("source failed")

// stubSource returns pre-configured fragments.
type stubSource struct {
	name      string
	fragments []Fragment
	err       error
	called    int
}

func (s *stubSource) Name() string { return s.name }

func (s *stubSource) Retrieve(
	_ context.Context, _ Request,
) ([]Fragment, error) {
	s.called++

	return s.fragments, s.err
}

// TestNewPipeline_Empty verifies empty pipeline creation.
// Kills mutant: nil sources slice would panic on Assemble.
func TestNewPipeline_Empty(t *testing.T) {
	t.Parallel()

	p := NewPipeline()

	assert.NotNil(t, p)

	result := p.Assemble(t.Context(), Request{})
	assert.Empty(t, result.Fragments)
}

// TestPipeline_SingleSource verifies single source retrieval.
// Kills mutant: not calling source would return empty result.
func TestPipeline_SingleSource(t *testing.T) {
	t.Parallel()

	src := &stubSource{
		name: testSourceA,
		fragments: []Fragment{
			{Source: testSourceA, Content: "fragment 1"},
		},
	}

	p := NewPipeline(src)
	result := p.Assemble(t.Context(), Request{Turn: 1, Query: "test"})

	require.Len(t, result.Fragments, 1)
	assert.Equal(t, testSourceA, result.Fragments[0].Source)
	assert.Equal(t, "fragment 1", result.Fragments[0].Content)
	assert.Equal(t, 1, src.called)
}

// TestPipeline_MultipleSources verifies fragments from multiple sources.
// Kills mutant: only querying first source would miss subsequent fragments.
func TestPipeline_MultipleSources(t *testing.T) {
	t.Parallel()

	srcA := &stubSource{
		name:      testSourceA,
		fragments: []Fragment{{Source: testSourceA, Content: "A1"}},
	}
	srcB := &stubSource{
		name:      testSourceB,
		fragments: []Fragment{{Source: testSourceB, Content: "B1"}},
	}

	p := NewPipeline(srcA, srcB)
	result := p.Assemble(t.Context(), Request{})

	require.Len(t, result.Fragments, 2)
	assert.Equal(t, "A1", result.Fragments[0].Content)
	assert.Equal(t, "B1", result.Fragments[1].Content)
}

// TestPipeline_SourceError_Skipped verifies failing sources are skipped.
// Kills mutant: propagating error would abort entire pipeline.
func TestPipeline_SourceError_Skipped(t *testing.T) {
	t.Parallel()

	failing := &stubSource{
		name: testSourceA,
		err:  errSourceFailed,
	}
	working := &stubSource{
		name:      testSourceB,
		fragments: []Fragment{{Source: testSourceB, Content: "ok"}},
	}

	p := NewPipeline(failing, working)
	result := p.Assemble(t.Context(), Request{})

	require.Len(t, result.Fragments, 1)
	assert.Equal(t, "ok", result.Fragments[0].Content)
	assert.Equal(t, 1, failing.called)
	assert.Equal(t, 1, working.called)
}

// TestPipeline_EmptySource verifies source returning nil fragments.
// Kills mutant: appending nil slice would not grow result.
func TestPipeline_EmptySource(t *testing.T) {
	t.Parallel()

	src := &stubSource{name: testSourceA, fragments: nil}
	p := NewPipeline(src)

	result := p.Assemble(t.Context(), Request{})

	assert.Empty(t, result.Fragments)
	assert.Equal(t, 1, src.called)
}

// TestPipeline_PreservesOrder verifies fragment ordering matches source registration.
// Kills mutant: randomizing order would break deterministic output.
func TestPipeline_PreservesOrder(t *testing.T) {
	t.Parallel()

	srcA := &stubSource{
		name: testSourceA,
		fragments: []Fragment{
			{Source: testSourceA, Content: "first"},
			{Source: testSourceA, Content: "second"},
		},
	}
	srcB := &stubSource{
		name:      testSourceB,
		fragments: []Fragment{{Source: testSourceB, Content: "third"}},
	}

	p := NewPipeline(srcA, srcB)
	result := p.Assemble(t.Context(), Request{})

	require.Len(t, result.Fragments, 3)
	assert.Equal(t, "first", result.Fragments[0].Content)
	assert.Equal(t, "second", result.Fragments[1].Content)
	assert.Equal(t, "third", result.Fragments[2].Content)
}

// TestPipeline_AllSourcesFail verifies graceful handling when all sources fail.
// Kills mutant: not initializing result would cause nil pointer.
func TestPipeline_AllSourcesFail(t *testing.T) {
	t.Parallel()

	srcA := &stubSource{name: testSourceA, err: errSourceFailed}
	srcB := &stubSource{name: testSourceB, err: errSourceFailed}

	p := NewPipeline(srcA, srcB)
	result := p.Assemble(t.Context(), Request{})

	assert.Empty(t, result.Fragments)
}
