package detection

import (
	"fmt"

	"github.com/dmytrogajewski/spin/internal/cycle"
)

// DetectionService handles cycle and pattern detection for agent behavior.
//
// This service centralizes detection logic that was previously embedded in Agent.
// It provides a clean interface for recording agent state snapshots and detecting
// problematic patterns such as repeated tool calls, similar responses, and error loops.
type DetectionService struct {
	cycleDetector   *cycle.Detector
	patternDetector *cycle.PatternDetector
}

// NewDetectionService creates a new detection service with the given detectors.
//
// Both cycleDetector and patternDetector can be nil. When nil, methods that
// require these dependencies will return appropriate errors.
func NewDetectionService(cycleDetector *cycle.Detector, patternDetector *cycle.PatternDetector) *DetectionService {
	return &DetectionService{
		cycleDetector:   cycleDetector,
		patternDetector: patternDetector,
	}
}

// RecordSnapshot records an agent state snapshot for cycle detection.
//
// This should be called after each agent turn to maintain the rolling window
// of recent behavior. If the cycle detector is not configured, this is a no-op.
func (s *DetectionService) RecordSnapshot(snapshot cycle.Snapshot) {
	if s.cycleDetector == nil {
		return
	}

	s.cycleDetector.Record(snapshot)
}

// CheckCycle analyzes the current history for cycle patterns.
//
// Returns the type of cycle detected (or CycleNone) along with confidence
// and details about the detection. Returns an error if the cycle detector
// is not configured.
//
// Common cycle types:
// - CycleSimilarResponses: Agent repeating similar responses
// - CycleRepeatedTool: Same tool called repeatedly
// - CycleSameError: Same error occurring multiple times
// - CycleOscillation: Agent alternating between two states
func (s *DetectionService) CheckCycle() (cycle.CycleResult, error) {
	if s.cycleDetector == nil {
		return cycle.CycleResult{Type: cycle.CycleNone}, fmt.Errorf("cycle detector not configured")
	}

	result, err := s.cycleDetector.Check()
	if err != nil {
		return cycle.CycleResult{Type: cycle.CycleNone}, fmt.Errorf("cycle detection failed: %w", err)
	}

	return result, nil
}

// DetectPattern detects advanced patterns in agent behavior.
//
// This is more sophisticated than basic cycle detection and can identify
// complex behavioral patterns. Returns an error if the pattern detector
// is not configured.
//
// Note: Pattern detection requires a sufficient history of snapshots
// to analyze. It may return an empty slice if there isn't enough data.
func (s *DetectionService) DetectPattern() ([]cycle.PatternResult, error) {
	if s.patternDetector == nil {
		return nil, fmt.Errorf("pattern detector not configured")
	}

	if s.cycleDetector == nil {
		return nil, fmt.Errorf("cycle detector required for pattern detection")
	}

	// Get current history for pattern analysis
	history := s.cycleDetector.GetHistory()

	results := s.patternDetector.AnalyzePatterns(history)
	return results, nil
}

// Reset clears the detection history.
//
// This is useful when starting a new conversation or when you want to
// reset the detection state. If the cycle detector is not configured,
// this is a no-op.
func (s *DetectionService) Reset() {
	if s.cycleDetector == nil {
		return
	}

	s.cycleDetector.Reset()
}

// GetHistory returns a copy of the current snapshot history.
//
// This is primarily for testing and debugging purposes. Returns an empty
// slice if the cycle detector is not configured.
func (s *DetectionService) GetHistory() []cycle.Snapshot {
	if s.cycleDetector == nil {
		return []cycle.Snapshot{}
	}

	return s.cycleDetector.GetHistory()
}
