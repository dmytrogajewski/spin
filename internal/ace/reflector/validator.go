package reflector

import (
	"errors"
	"fmt"
)

// ErrInsightCannotBeNil is a sentinel error.
var ErrInsightCannotBeNil = errors.New("insight cannot be nil")

// InsightValidator validates insight quality.
type InsightValidator struct {
	// Future: add configuration options.
}

// NewInsightValidator creates a new insight validator.
func NewInsightValidator() *InsightValidator {
	return &InsightValidator{}
}

// Validate checks if an insight meets quality requirements.
func (v *InsightValidator) Validate(insight *Insight) error {
	if insight == nil {
		return ErrInsightCannotBeNil
	}

	// Delegate to existing Validate method on Insight.
	return insight.Validate()
}

// ValidateBatch validates multiple insights and returns all errors.
func (v *InsightValidator) ValidateBatch(insights []*Insight) []error {
	var errs []error

	for i, insight := range insights {
		err := v.Validate(insight)
		if err != nil {
			errs = append(errs, fmt.Errorf("insight %d: %w", i, err))
		}
	}

	return errs
}

// FilterByQuality filters insights by minimum confidence threshold.
func (v *InsightValidator) FilterByQuality(insights []*Insight, minConfidence float64) []*Insight {
	filtered := make([]*Insight, 0, len(insights))

	for _, insight := range insights {
		if insight.Confidence >= minConfidence {
			filtered = append(filtered, insight)
		}
	}

	return filtered
}
