// Package cycle provides automatic detection of agent reasoning loops
// and intelligent intervention strategies to break cycles and maintain productivity.
//
// The package implements multiple detection methods:
//   - Response similarity (Jaccard similarity of text)
//   - Repeated tool calls
//   - State oscillation patterns (A→B→A→B)
//   - Error repetition
//
// Interventions are applied based on cycle severity:
//   - Soft: Reflection prompts for early cycles
//   - Medium: Context summarization for mid-stage cycles
//   - Hard: User escalation for persistent cycles
package cycle

import (
	"time"

	"github.com/dmytrogajewski/spin/internal/detection"
)

// Type aliases to detection package - cycle uses detection types directly.
type (
	CycleType   = detection.CycleType
	Snapshot    = detection.Snapshot
	CycleResult = detection.CycleResult
)

// Re-export detection constants for backward compatibility.
const (
	CycleNone             = detection.CycleNone
	CycleSimilarResponses = detection.CycleSimilarResponses
	CycleRepeatedTool     = detection.CycleRepeatedTool
	CycleOscillation      = detection.CycleOscillation
	CycleSameError        = detection.CycleSameError
)

// Config contains configuration for cycle detection.
type Config struct {
	// WindowSize is the number of snapshots to compare for pattern detection (default: 10).
	WindowSize int

	// SimilarityThresh is the threshold for response similarity detection (default: 0.8).
	SimilarityThresh float64

	// ToolRepeatLimit is the max identical tool calls before triggering cycle (default: 3).
	ToolRepeatLimit int

	// ErrorRepeatLimit is the max identical errors before triggering cycle (default: 3).
	ErrorRepeatLimit int

	// Enabled controls whether cycle detection is active (default: true).
	Enabled bool
}

// InterventionType represents the type of intervention applied.
type InterventionType int

const (
	// InterventionNone indicates no intervention needed.
	InterventionNone InterventionType = iota

	// InterventionReflection uses reflection prompts for early cycles.
	InterventionReflection

	// InterventionSummarize compresses context for mid-stage cycles.
	InterventionSummarize

	// InterventionEscalate pauses agent and requests user guidance.
	InterventionEscalate
)

// InterventionResult contains the result of applying an intervention.
type InterventionResult struct {
	// Type is the type of intervention applied.
	Type InterventionType

	// Success indicates whether the intervention was applied successfully.
	Success bool

	// Message describes what happened during the intervention.
	Message string

	// Timestamp when the intervention was applied.
	Timestamp time.Time
}
