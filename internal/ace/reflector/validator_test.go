package reflector

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestInsightValidator_New tests creating a new validator.
func TestInsightValidator_New(t *testing.T) {
	t.Parallel()

	validator := NewInsightValidator()

	require.NotNil(t, validator)
}

// validatorValidateCases contains test cases for validator validation.
var validatorValidateCases = []struct {
	name    string
	insight *Insight
	wantErr bool
	errMsg  string
}{
	{
		name:    "valid insight",
		insight: &Insight{
			Content: "Always validate input parameters before processing them in Go",
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

// TestInsightValidator_Validate tests insight validation with all rules.
func TestInsightValidator_Validate(t *testing.T) {
	t.Parallel()

	validator := NewInsightValidator()

	for _, tt := range validatorValidateCases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := validator.Validate(tt.insight)
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// TestInsightValidator_ValidateBatch tests batch validation.
func TestInsightValidator_ValidateBatch(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		insights []*Insight
		wantErr  bool
		errCount int
	}{
		{
			name: "all valid",
			insights: []*Insight{
				{
					Content:    "Always validate input parameters before processing them",
					Confidence: 0.8,
					Category:   CategorySuccessPattern,
				},
				{
					Content:    "Use errors.Is for error type checking in Go applications",
					Confidence: 0.9,
					Category:   CategorySuccessPattern,
				},
			},
			wantErr:  false,
			errCount: 0,
		},
		{
			name: "some invalid",
			insights: []*Insight{
				{
					Content:    "Always validate input parameters before processing them",
					Confidence: 0.8,
					Category:   CategorySuccessPattern,
				},
				{
					Content:    "short",
					Confidence: 0.9,
					Category:   CategorySuccessPattern,
				},
			},
			wantErr:  true,
			errCount: 1,
		},
		{
			name:     "empty slice",
			insights: []*Insight{},
			wantErr:  false,
			errCount: 0,
		},
	}

	validator := NewInsightValidator()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			errs := validator.ValidateBatch(tt.insights)
			if tt.wantErr {
				assert.Len(t, errs, tt.errCount)
			} else {
				assert.Empty(t, errs)
			}
		})
	}
}

// TestInsightValidator_FilterByQuality tests quality filtering.
func TestInsightValidator_FilterByQuality(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		insights      []*Insight
		minConfidence float64
		wantCount     int
	}{
		{
			name: "filter by confidence",
			insights: []*Insight{
				{
					Content:    "Always validate input parameters before processing them",
					Confidence: 0.9,
					Category:   CategorySuccessPattern,
				},
				{
					Content:    "Use errors.Is for error type checking in Go applications",
					Confidence: 0.4,
					Category:   CategorySuccessPattern,
				},
				{
					Content:    "Always use context.Context as the first parameter in functions",
					Confidence: 0.7,
					Category:   CategorySuccessPattern,
				},
			},
			minConfidence: 0.5,
			wantCount:     2,
		},
		{
			name: "no filtering needed",
			insights: []*Insight{
				{
					Content:    "Always validate input parameters before processing them",
					Confidence: 0.9,
					Category:   CategorySuccessPattern,
				},
			},
			minConfidence: 0.5,
			wantCount:     1,
		},
		{
			name:          "empty slice",
			insights:      []*Insight{},
			minConfidence: 0.5,
			wantCount:     0,
		},
	}

	validator := NewInsightValidator()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			filtered := validator.FilterByQuality(tt.insights, tt.minConfidence)
			assert.Len(t, filtered, tt.wantCount)

			// All filtered insights should meet minimum confidence.
			for _, insight := range filtered {
				assert.GreaterOrEqual(t, insight.Confidence, tt.minConfidence)
			}
		})
	}
}
