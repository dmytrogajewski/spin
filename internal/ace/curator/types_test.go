package curator

import (
	"testing"

	"github.com/dmytrogajewski/spin/internal/ace/reflector"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMergeRequest_Creation tests creating a merge request
func TestMergeRequest_Creation(t *testing.T) {
	insights := []*reflector.Insight{
		reflector.NewInsight("Test insight", reflector.CategorySuccessPattern),
	}

	req := &MergeRequest{
		Insights:            insights,
		SimilarityThreshold: 0.85,
	}

	require.NotNil(t, req)
	assert.Equal(t, 1, len(req.Insights))
	assert.Equal(t, 0.85, req.SimilarityThreshold)
}

// TestMergeResult_Empty tests empty merge result
func TestMergeResult_Empty(t *testing.T) {
	result := &MergeResult{
		Added:   0,
		Skipped: 0,
		Updated: 0,
	}

	require.NotNil(t, result)
	assert.Equal(t, 0, result.Added)
	assert.Equal(t, 0, result.Skipped)
}
