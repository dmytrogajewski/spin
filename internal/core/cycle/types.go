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
)

// CycleType represents the type of cycle detected
type CycleType int

const (
	// CycleNone indicates no cycle detected
	CycleNone CycleType = iota

	// CycleSimilarResponses indicates repeated similar responses
	CycleSimilarResponses

	// CycleRepeatedTool indicates same tool called repeatedly
	CycleRepeatedTool

	// CycleOscillation indicates A→B→A→B oscillation pattern
	CycleOscillation

	// CycleSameError indicates repeated identical errors
	CycleSameError
)

// String returns the string representation of the cycle type
func (ct CycleType) String() string {
	switch ct {
	case CycleSimilarResponses:
		return "similar_responses"
	case CycleRepeatedTool:
		return "repeated_tool"
	case CycleOscillation:
		return "oscillation"
	case CycleSameError:
		return "same_error"
	default:
		return "none"
	}
}

// Snapshot represents a point-in-time capture of agent state
// for cycle detection analysis
type Snapshot struct {
	// Turn is the agent turn number when this snapshot was taken
	Turn int

	// Response is the LLM response content (if any)
	Response string

	// ToolCalls are the names of tools called in this turn
	ToolCalls []string

	// Error is any error that occurred in this turn (if any)
	Error string

	// Timestamp when this snapshot was created
	Timestamp time.Time
}

// Config contains configuration for cycle detection
type Config struct {
	// WindowSize is the number of snapshots to compare for pattern detection (default: 3)
	WindowSize int

	// SimilarityThresh is the threshold for response similarity detection (default: 0.8)
	SimilarityThresh float64

	// ToolRepeatLimit is the max identical tool calls before triggering cycle (default: 3)
	ToolRepeatLimit int

	// ErrorRepeatLimit is the max identical errors before triggering cycle (default: 3)
	ErrorRepeatLimit int

	// Enabled controls whether cycle detection is active (default: true)
	Enabled bool
}

// DefaultConfig returns sensible defaults for cycle detection
func DefaultConfig() Config {
	return Config{
		WindowSize:       3,
		SimilarityThresh: 0.8,
		ToolRepeatLimit:  3,
		ErrorRepeatLimit: 3,
		Enabled:          true,
	}
}

// CycleResult contains the result of cycle detection
type CycleResult struct {
	// Type is the type of cycle detected (or CycleNone)
	Type CycleType

	// Confidence is a value 0.0-1.0 indicating detection confidence
	Confidence float64

	// Details provides additional information about the cycle
	Details string

	// Timestamp when the cycle was detected
	Timestamp time.Time
}

// InterventionType represents the type of intervention applied
type InterventionType int

const (
	// InterventionNone indicates no intervention needed
	InterventionNone InterventionType = iota

	// InterventionReflection uses reflection prompts for early cycles
	InterventionReflection

	// InterventionSummarize compresses context for mid-stage cycles
	InterventionSummarize

	// InterventionEscalate pauses agent and requests user guidance
	InterventionEscalate
)

// String returns the string representation of the intervention type
func (it InterventionType) String() string {
	switch it {
	case InterventionReflection:
		return "reflection"
	case InterventionSummarize:
		return "summarize"
	case InterventionEscalate:
		return "escalate"
	default:
		return "none"
	}
}

// InterventionResult contains the result of applying an intervention
type InterventionResult struct {
	// Type is the type of intervention applied
	Type InterventionType

	// Success indicates whether the intervention was applied successfully
	Success bool

	// Message describes what happened during the intervention
	Message string

	// Timestamp when the intervention was applied
	Timestamp time.Time
}
