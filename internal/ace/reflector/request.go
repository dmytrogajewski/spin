package reflector

import (
	"time"

	"github.com/dmytrogajewski/spin/internal/ace/generator"
)

// ReflectionRequest contains parameters for reflection analysis.
type ReflectionRequest struct {
	// Trajectories to analyze
	Trajectories []*generator.Trajectory

	// MaxIterations for refinement (default 3)
	MaxIterations int

	// MinConfidence threshold (default 0.5)
	MinConfidence float64

	// Model to use for LLM calls
	Model string

	// Temperature for LLM (default 0.3 for consistency)
	Temperature float64
}

// ReflectionResponse contains the results of reflection analysis.
type ReflectionResponse struct {
	// Insights extracted from trajectories
	Insights []*Insight

	// Iterations performed during refinement
	Iterations int

	// TotalTokens used by LLM
	TotalTokens int

	// Duration of reflection process
	Duration time.Duration
}
