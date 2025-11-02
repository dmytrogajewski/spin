package reflector

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestInsight_New tests creating a new insight
func TestInsight_New(t *testing.T) {
	insight := NewInsight("Always validate input parameters", CategorySuccessPattern)

	require.NotNil(t, insight)
	assert.Equal(t, "Always validate input parameters", insight.Content)
	assert.Equal(t, CategorySuccessPattern, insight.Category)
	assert.False(t, insight.CreatedAt.IsZero())
}

// TestInsight_Validate tests insight validation
func TestInsight_Validate(t *testing.T) {
	tests := []struct {
		name    string
		insight *Insight
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid insight",
			insight: &Insight{
				Content:    "Always validate input parameters before processing them",
				Confidence: 0.8,
				Category:   CategorySuccessPattern,
			},
			wantErr: false,
		},
		{
			name: "empty content",
			insight: &Insight{
				Content:    "",
				Confidence: 0.8,
				Category:   CategorySuccessPattern,
			},
			wantErr: true,
			errMsg:  "content cannot be empty",
		},
		{
			name: "content too short",
			insight: &Insight{
				Content:    "short",
				Confidence: 0.8,
				Category:   CategorySuccessPattern,
			},
			wantErr: true,
			errMsg:  "content too short",
		},
		{
			name: "content too long",
			insight: &Insight{
				Content:    strings.Repeat("x", 501),
				Confidence: 0.8,
				Category:   CategorySuccessPattern,
			},
			wantErr: true,
			errMsg:  "content too long",
		},
		{
			name: "confidence negative",
			insight: &Insight{
				Content:    "Always validate input parameters before processing them",
				Confidence: -0.1,
				Category:   CategorySuccessPattern,
			},
			wantErr: true,
			errMsg:  "confidence",
		},
		{
			name: "confidence too high",
			insight: &Insight{
				Content:    "Always validate input parameters before processing them",
				Confidence: 1.5,
				Category:   CategorySuccessPattern,
			},
			wantErr: true,
			errMsg:  "confidence",
		},
		{
			name: "invalid category",
			insight: &Insight{
				Content:    "Always validate input parameters before processing them",
				Confidence: 0.8,
				Category:   "invalid_category",
			},
			wantErr: true,
			errMsg:  "category",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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
