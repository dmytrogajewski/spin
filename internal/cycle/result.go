package cycle

import (
	"errors"
	"time"
)

const (
	// noneLabel is the display label for undetected/unknown cycle types.
	noneLabel = "none"
)

var (
	// ErrDetectorNotConfigured is a sentinel error.
	ErrDetectorNotConfigured = errors.New("cycle detector not configured")
	// ErrPatternDetectorNotConfigured is a sentinel error.
	ErrPatternDetectorNotConfigured = errors.New("pattern detector not configured")
	// ErrDetectorRequiredForPatternDetection is a sentinel error.
	ErrDetectorRequiredForPatternDetection = errors.New("cycle detector required for pattern detection")
)

// Type represents the type of cycle detected.
type Type int

const (
	// None indicates no cycle detected.
	None Type = iota

	// SimilarResponses indicates repeated similar responses.
	SimilarResponses

	// RepeatedTool indicates same tool called repeatedly.
	RepeatedTool

	// Oscillation indicates A→B→A→B oscillation pattern.
	Oscillation

	// SameError indicates repeated identical errors.
	SameError
)

// String returns the string representation of the cycle type.
func (ct Type) String() string {
	switch ct {
	case None:
		return noneLabel
	case SimilarResponses:
		return "similar_responses"
	case RepeatedTool:
		return "repeated_tool"
	case Oscillation:
		return "oscillation"
	case SameError:
		return "same_error"
	default:
		return noneLabel
	}
}

// Snapshot represents a point-in-time capture of agent state for cycle detection analysis.
type Snapshot struct {
	Turn      int
	Response  string
	ToolCalls []string
	Error     string
	Timestamp time.Time
}

// Result contains the result of cycle detection.
type Result struct {
	Type       Type
	Confidence float64
	Details    string
	Timestamp  time.Time
}
