package generator

import (
	"time"

	"github.com/dmytrogajewski/spin/internal/ace/bullet"
	"github.com/dmytrogajewski/spin/internal/ace/feedback"
)

// BulletFeedback is re-exported from feedback package.
type BulletFeedback = feedback.BulletFeedback

// Trajectory is a complete execution trace.
type Trajectory struct {
	// ID is unique identifier
	ID string

	// Query is the input task
	Query string

	// RetrievedBullets are context bullets used
	RetrievedBullets []*bullet.Bullet

	// Steps are execution steps in order
	Steps []TrajectoryStep

	// Output is the final result
	Output string

	// Success indicates if task succeeded
	Success bool

	// BulletFeedback contains utility annotations
	BulletFeedback *BulletFeedback

	// Metadata contains additional info
	Metadata TrajectoryMetadata

	// CreatedAt is when trajectory was generated
	CreatedAt time.Time
}

// TrajectoryMetadata contains additional trajectory info.
type TrajectoryMetadata struct {
	Model       string
	Temperature float64
	MaxTokens   int
	TotalTokens int
	Duration    time.Duration
	Turns       int
}

// TrajectoryStep is a single reasoning or execution step.
type TrajectoryStep struct {
	// StepNumber is the step index (0-based)
	StepNumber int

	// Type is step type ("reasoning", "tool_call", "tool_result")
	Type string

	// Content is the step content
	Content string

	// Timestamp is when step occurred
	Timestamp time.Time
}
