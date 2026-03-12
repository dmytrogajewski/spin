// Package detection provides pattern detection services.
package detection

import (
	"context"
	"errors"
	"fmt"
	"time"
)

const escalateSeverity = 3

var (
	// ErrCycleDetectorNotConfigured is a sentinel error.
	ErrCycleDetectorNotConfigured = errors.New("cycle detector not configured")
	// ErrPatternDetectorNotConfigured is a sentinel error.
	ErrPatternDetectorNotConfigured = errors.New("pattern detector not configured")
	// ErrCycleDetectorRequiredForPatternDetection is a sentinel error.
	ErrCycleDetectorRequiredForPatternDetection = errors.New("cycle detector required for pattern detection")
)

// CycleType represents the type of cycle detected.
type CycleType int

const (
	// CycleNone indicates no cycle detected.
	CycleNone CycleType = iota

	// CycleSimilarResponses indicates repeated similar responses.
	CycleSimilarResponses

	// CycleRepeatedTool indicates same tool called repeatedly.
	CycleRepeatedTool

	// CycleOscillation indicates A→B→A→B oscillation pattern.
	CycleOscillation

	// CycleSameError indicates repeated identical errors.
	CycleSameError
)

// String returns the string representation of the cycle type.
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

// Snapshot represents a point-in-time capture of agent state for cycle detection analysis.
type Snapshot struct {
	Turn      int
	Response  string
	ToolCalls []string
	Error     string
	Timestamp time.Time
}

// CycleResult contains the result of cycle detection.
type CycleResult struct {
	Type       CycleType
	Confidence float64
	Details    string
	Timestamp  time.Time
}

// Message represents a conversation message for interventions.
type Message interface {
	GetRole() string
	GetContent() string
	GetTimestamp() time.Time
}

// EventEmitter defines the interface for emitting events.
type EventEmitter interface {
	Emit(event Event)
}

// Event defines the interface for events.
type Event interface {
	GetType() string
	GetTimestamp() time.Time
	GetData() any
}

// EventData represents the data payload for detection events.
// This is a strongly-typed alternative to interface{} for detection-specific events.
type EventData map[string]any

// Intervention defines the interface for cycle-breaking strategies.
type Intervention interface {
	Apply(ctx context.Context, messages []Message) ([]Message, error)
	Name() string
	Description() string
	Severity() int
}

// ReflectionIntervention implements a soft intervention.
type ReflectionIntervention struct{}

// Apply implements the Apply operation.
func (i *ReflectionIntervention) Apply(_ context.Context, messages []Message) ([]Message, error) {
	reflectionMsg := &message{
		role: "user",
		content: "I notice you may be repeating similar responses or approaches. " +
			"Let's take a step back and try a different perspective. " +
			"What other angles or methods could we explore for this task?",
		timestamp: time.Now(),
	}

	return append(messages, reflectionMsg), nil
}

// Name returns the intervention name.
func (i *ReflectionIntervention) Name() string { return "Reflection" }

// Description returns a description of the intervention.
func (i *ReflectionIntervention) Description() string {
	return "Injects a reflection prompt to help the agent recognize repetitive patterns and consider alternative approaches"
}

// Severity returns the intervention severity level.
func (i *ReflectionIntervention) Severity() int { return 1 }

// EscalateIntervention implements a hard intervention.
type EscalateIntervention struct {
	Emitter EventEmitter
}

// Apply implements the Apply operation.
func (i *EscalateIntervention) Apply(_ context.Context, messages []Message) ([]Message, error) {
	if i.Emitter != nil {
		i.Emitter.Emit(&event{
			eventType: "turn_paused",
			timestamp: time.Now(),
			data: EventData{
				"status":  "paused",
				"message": "Agent appears stuck in a reasoning cycle. Please provide guidance or restart the conversation.",
			},
		})
	}

	return messages, nil
}

// Name returns the intervention name.
func (i *EscalateIntervention) Name() string { return "User Escalation" }

// Description returns a description of the intervention.
func (i *EscalateIntervention) Description() string {
	return "Pauses agent execution and requests user guidance when automated interventions fail to break the cycle"
}

// Severity returns the intervention severity level.
func (i *EscalateIntervention) Severity() int { return escalateSeverity }

// message implements the Message interface.
type message struct {
	role      string
	content   string
	timestamp time.Time
}

// GetRole returns the message role.
func (m *message) GetRole() string { return m.role }

// GetContent returns the message content.
func (m *message) GetContent() string { return m.content }

// GetTimestamp returns the message timestamp.
func (m *message) GetTimestamp() time.Time { return m.timestamp }

// event implements the Event interface.
type event struct {
	eventType string
	timestamp time.Time
	data      EventData
}

// GetType returns the event type.
func (e *event) GetType() string { return e.eventType }

// GetTimestamp returns the event timestamp.
func (e *event) GetTimestamp() time.Time { return e.timestamp }

// GetData returns the event data.
func (e *event) GetData() any { return e.data }

// CycleDetector defines the interface for cycle detection implementations.
type CycleDetector interface {
	Record(snapshot Snapshot)
	Check() (CycleResult, error)
	GetHistory() []Snapshot
	Reset()
}

// PatternDetector defines the interface for pattern detection implementations.
type PatternDetector interface {
	AnalyzePatterns(history []Snapshot) []PatternResult
}

// PatternResult represents the result of pattern detection.
type PatternResult struct {
	Type       string
	Confidence float64
	Details    string
}

// Service handles cycle and pattern detection for agent behavior.
type Service struct {
	cycleDetector   CycleDetector
	patternDetector PatternDetector
}

// NewService creates a new detection service with the given detectors.
func NewService(cycleDetector CycleDetector, patternDetector PatternDetector) *Service {
	return &Service{
		cycleDetector:   cycleDetector,
		patternDetector: patternDetector,
	}
}

// RecordSnapshot records an agent state snapshot for cycle detection.
func (s *Service) RecordSnapshot(snapshot Snapshot) {
	if s.cycleDetector == nil {
		return
	}

	s.cycleDetector.Record(snapshot)
}

// CheckCycle analyzes the current history for cycle patterns.
func (s *Service) CheckCycle() (CycleResult, error) {
	if s.cycleDetector == nil {
		return CycleResult{Type: CycleNone}, ErrCycleDetectorNotConfigured
	}

	result, err := s.cycleDetector.Check()
	if err != nil {
		return CycleResult{Type: CycleNone}, fmt.Errorf("cycle detection failed: %w", err)
	}

	return result, nil
}

// DetectPattern detects advanced patterns in agent behavior.
func (s *Service) DetectPattern() ([]PatternResult, error) {
	if s.patternDetector == nil {
		return nil, ErrPatternDetectorNotConfigured
	}

	if s.cycleDetector == nil {
		return nil, ErrCycleDetectorRequiredForPatternDetection
	}

	history := s.cycleDetector.GetHistory()
	results := s.patternDetector.AnalyzePatterns(history)

	return results, nil
}

// Reset clears the detection history.
func (s *Service) Reset() {
	if s.cycleDetector == nil {
		return
	}

	s.cycleDetector.Reset()
}

// GetHistory returns a copy of the current snapshot history.
func (s *Service) GetHistory() []Snapshot {
	if s.cycleDetector == nil {
		return []Snapshot{}
	}

	return s.cycleDetector.GetHistory()
}
