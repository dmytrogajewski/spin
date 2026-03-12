package reflector

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestInsight_New tests creating a new insight.
func TestInsight_New(t *testing.T) {
	t.Parallel()

	insight := NewInsight("Always validate input parameters", CategorySuccessPattern)

	require.NotNil(t, insight)
	assert.Equal(t, "Always validate input parameters", insight.Content)
	assert.Equal(t, CategorySuccessPattern, insight.Category)
	assert.False(t, insight.CreatedAt.IsZero())
}

// insightValidateCases contains test cases for insight validation.
var insightValidateCases = []struct {
	name    string
	insight *Insight
	wantErr bool
	errMsg  string
}{
	{
		name:    "valid insight",
		insight: &Insight{
			Content: "Always validate input parameters before processing them",
			Confidence: 0.8, Category: CategorySuccessPattern,
		},
	},
	{
		name: "empty content", wantErr: true, errMsg: "content cannot be empty",
		insight: &Insight{Content: "", Confidence: 0.8, Category: CategorySuccessPattern},
	},
	{
		name: "content too short", wantErr: true, errMsg: "content too short",
		insight: &Insight{Content: "short", Confidence: 0.8, Category: CategorySuccessPattern},
	},
	{
		name: "content too long", wantErr: true, errMsg: "content too long",
		insight: &Insight{Content: strings.Repeat("x", 501), Confidence: 0.8, Category: CategorySuccessPattern},
	},
	{
		name: "confidence negative", wantErr: true, errMsg: "confidence",
		insight: &Insight{
			Content: "Always validate input parameters before processing them",
			Confidence: -0.1, Category: CategorySuccessPattern,
		},
	},
	{
		name: "confidence too high", wantErr: true, errMsg: "confidence",
		insight: &Insight{
			Content: "Always validate input parameters before processing them",
			Confidence: 1.5, Category: CategorySuccessPattern,
		},
	},
	{
		name: "invalid category", wantErr: true, errMsg: "category",
		insight: &Insight{
			Content: "Always validate input parameters before processing them",
			Confidence: 0.8, Category: "invalid_category",
		},
	},
}

// TestInsight_Validate tests insight validation.
func TestInsight_Validate(t *testing.T) {
	t.Parallel()

	for _, tt := range insightValidateCases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := tt.insight.Validate()
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
