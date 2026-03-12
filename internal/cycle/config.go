package cycle

import (
	"time"

	"github.com/dmytrogajewski/spin/internal/detection"
)

// Type is an alias to detection.CycleType.
type Type = detection.CycleType

// Snapshot is an alias to detection.Snapshot.
type Snapshot = detection.Snapshot

// Result is an alias to detection.CycleResult.
type Result = detection.CycleResult

// Re-export detection constants for backward compatibility.
const (
	// CycleNone is exported.
	CycleNone = detection.CycleNone
	// CycleSimilarResponses is exported.
	CycleSimilarResponses = detection.CycleSimilarResponses
	// CycleRepeatedTool is exported.
	CycleRepeatedTool = detection.CycleRepeatedTool
	// CycleOscillation is exported.
	CycleOscillation = detection.CycleOscillation
	// CycleSameError is exported.
	CycleSameError = detection.CycleSameError
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
