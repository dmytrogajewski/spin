package curator

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dmytrogajewski/spin/internal/ace/reflector"
)

// TestConvertInsights_Single tests converting single insight.
func TestConvertInsights_Single(t *testing.T) {
	t.Parallel()

	insight := &reflector.Insight{
		Content:    "Always validate input parameters before processing them",
		Source:     "traj-123",
		Confidence: 0.85,
		Category:   reflector.CategorySuccessPattern,
		Evidence:   []string{"validation prevented error"},
	}

	bullets, err := ConvertInsights([]*reflector.Insight{insight})

	require.NoError(t, err)
	require.Len(t, bullets, 1)

	bullet := bullets[0]
	assert.Equal(t, "Always validate input parameters before processing them", bullet.Content)
	assert.Equal(t, 8, bullet.HelpfulCount) // 0.85 * 10 = 8.5 → 8
	assert.Equal(t, 0, bullet.HarmfulCount)

	// Check metadata tags.
	require.NotNil(t, bullet.Tags)
	assert.Equal(t, "success_pattern", bullet.Tags["category"])
	assert.Equal(t, "traj-123", bullet.Tags["source"])
	assert.Contains(t, bullet.Tags["evidence"], "validation prevented error")
}

// TestConvertInsights_Empty tests converting empty insight list.
func TestConvertInsights_Empty(t *testing.T) {
	t.Parallel()

	bullets, err := ConvertInsights([]*reflector.Insight{})

	require.NoError(t, err)
	assert.Empty(t, bullets)
}

// TestConvertInsights_NoEvidence tests insight without evidence.
func TestConvertInsights_NoEvidence(t *testing.T) {
	t.Parallel()

	insight := &reflector.Insight{
		Content:    "Always validate input parameters before processing them",
		Confidence: 0.85,
		Category:   reflector.CategorySuccessPattern,
	}

	bullets, err := ConvertInsights([]*reflector.Insight{insight})

	require.NoError(t, err)
	require.Len(t, bullets, 1)

	// Should not have evidence tag.
	_, hasEvidence := bullets[0].Tags["evidence"]
	assert.False(t, hasEvidence)
}

// TestConvertInsights_MultipleEvidence tests joining multiple evidence strings.
func TestConvertInsights_MultipleEvidence(t *testing.T) {
	t.Parallel()

	insight := &reflector.Insight{
		Content:    "Always validate input parameters before processing them",
		Confidence: 0.85,
		Category:   reflector.CategorySuccessPattern,
		Evidence:   []string{"first evidence", "second evidence", "third evidence"},
	}

	bullets, err := ConvertInsights([]*reflector.Insight{insight})

	require.NoError(t, err)
	require.Len(t, bullets, 1)

	// Check evidence is joined with semicolon.
	assert.Equal(t, "first evidence; second evidence; third evidence", bullets[0].Tags["evidence"])
}
