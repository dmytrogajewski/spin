package cycle

import "fmt"

// Service handles cycle and pattern detection for agent behavior.
type Service struct {
	tracker         Tracker
	patternDetector PatternAnalyzer
}

// NewService creates a new detection service with the given detectors.
func NewService(tracker Tracker, patternDetector PatternAnalyzer) *Service {
	return &Service{
		tracker:         tracker,
		patternDetector: patternDetector,
	}
}

// RecordSnapshot records an agent state snapshot for cycle detection.
func (s *Service) RecordSnapshot(snapshot Snapshot) {
	if s.tracker == nil {
		return
	}

	s.tracker.Record(snapshot)
}

// CheckCycle analyzes the current history for cycle patterns.
func (s *Service) CheckCycle() (Result, error) {
	if s.tracker == nil {
		return Result{Type: None}, ErrDetectorNotConfigured
	}

	result, err := s.tracker.Check()
	if err != nil {
		return Result{Type: None}, fmt.Errorf("cycle detection failed: %w", err)
	}

	return result, nil
}

// DetectPattern detects advanced patterns in agent behavior.
func (s *Service) DetectPattern() ([]PatternResult, error) {
	if s.patternDetector == nil {
		return nil, ErrPatternDetectorNotConfigured
	}

	if s.tracker == nil {
		return nil, ErrDetectorRequiredForPatternDetection
	}

	history := s.tracker.GetHistory()
	results := s.patternDetector.AnalyzePatterns(history)

	return results, nil
}

// Reset clears the detection history.
func (s *Service) Reset() {
	if s.tracker == nil {
		return
	}

	s.tracker.Reset()
}

// GetHistory returns a copy of the current snapshot history.
func (s *Service) GetHistory() []Snapshot {
	if s.tracker == nil {
		return []Snapshot{}
	}

	return s.tracker.GetHistory()
}
