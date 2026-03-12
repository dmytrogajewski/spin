// Package reflector provides reflection and insight generation.
package reflector

import (
	"errors"
	"fmt"
	"time"
)

const (
	defaultInsightConfidence = 0.5
	minInsightContentLen     = 50
	maxInsightContentLen     = 500
)

var (
	// ErrContentCannotBeEmpty is a sentinel error.
	ErrContentCannotBeEmpty = errors.New("content cannot be empty")
	// ErrContentTooShortMin50Chars is a sentinel error.
	ErrContentTooShortMin50Chars = errors.New("content too short (min 50 chars")
	// ErrContentTooLongMax500Chars is a sentinel error.
	ErrContentTooLongMax500Chars = errors.New("content too long (max 500 chars")
	// ErrConfidenceMustBeBetween00 is a sentinel error.
	ErrConfidenceMustBeBetween00 = errors.New("confidence must be between 0.0 and 1.0")
	// ErrInvalidCategory is a sentinel error.
	ErrInvalidCategory = errors.New("invalid category")
)

// InsightCategory classifies the type of insight.
type InsightCategory string

// CategorySuccessPattern defines a CategorySuccessPattern constant.
const (
	CategorySuccessPattern InsightCategory = "success_pattern"
	CategoryErrorMode      InsightCategory = "error_mode"
	CategoryOptimization   InsightCategory = "optimization"
	CategoryAntiPattern    InsightCategory = "anti_pattern"
)

// Insight represents an actionable lesson extracted from a trajectory.
type Insight struct {
	// Content is the actionable lesson.
	Content string

	// Source is the trajectory ID.
	Source string

	// Confidence is reliability score (0.0 to 1.0).
	Confidence float64

	// Category classifies the insight type.
	Category InsightCategory

	// Evidence are supporting quotes from trajectory.
	Evidence []string

	// Iteration is refinement round when created.
	Iteration int

	// CreatedAt is timestamp.
	CreatedAt time.Time
}

// NewInsight creates a new insight with default values.
func NewInsight(content string, category InsightCategory) *Insight {
	return &Insight{
		Content:    content,
		Category:   category,
		Confidence: defaultInsightConfidence,
		Evidence:   []string{},
		Iteration:  0,
		CreatedAt:  time.Now(),
	}
}

// Validate checks if the insight meets quality requirements.
func (i *Insight) Validate() error {
	if i.Content == "" {
		return ErrContentCannotBeEmpty
	}

	if len(i.Content) < minInsightContentLen {
		return fmt.Errorf("content too short (min 50 chars, got %d): %w", len(i.Content), ErrContentTooShortMin50Chars)
	}

	if len(i.Content) > maxInsightContentLen {
		return fmt.Errorf("content too long (max 500 chars, got %d): %w", len(i.Content), ErrContentTooLongMax500Chars)
	}

	if i.Confidence < 0.0 || i.Confidence > 1.0 {
		return fmt.Errorf("confidence must be between 0.0 and 1.0, got %.2f: %w", i.Confidence, ErrConfidenceMustBeBetween00)
	}

	if !isValidCategory(i.Category) {
		return fmt.Errorf("invalid category: %s: %w", i.Category, ErrInvalidCategory)
	}

	return nil
}

// isValidCategory checks if the category is a valid enum value.
func isValidCategory(c InsightCategory) bool {
	switch c {
	case CategorySuccessPattern, CategoryErrorMode, CategoryOptimization, CategoryAntiPattern:
		return true
	default:
		return false
	}
}
