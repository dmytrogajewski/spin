package adapter

import "time"

// ExecutionSignal represents a feedback event from agent execution
type ExecutionSignal struct {
	// SignalType indicates the kind of signal
	SignalType SignalType

	// Context provides the execution context
	Context string

	// Outcome indicates success or failure
	Outcome SignalOutcome

	// Details contains signal-specific information
	Details map[string]string

	// Timestamp of signal generation
	Timestamp time.Time

	// SessionID links signal to session
	SessionID string
}

// SignalType categorizes execution signals
type SignalType string

const (
	SignalTypeTest    SignalType = "test"     // Test execution result
	SignalTypeBuild   SignalType = "build"    // Build/compile result
	SignalTypeLint    SignalType = "lint"     // Lint/static analysis
	SignalTypeError   SignalType = "error"    // Runtime error
	SignalTypeToolUse SignalType = "tool_use" // Tool execution pattern
	SignalTypeUser    SignalType = "user"     // User correction/approval
)

// SignalOutcome indicates signal polarity
type SignalOutcome string

const (
	OutcomeSuccess SignalOutcome = "success"
	OutcomeFailure SignalOutcome = "failure"
	OutcomeNeutral SignalOutcome = "neutral"
)

// Session tracks online learning state
type Session struct {
	// ID uniquely identifies the session
	ID string

	// StartTime marks session creation
	StartTime time.Time

	// SignalCount tracks total signals processed
	SignalCount int

	// UpdateCount tracks playbook updates made
	UpdateCount int

	// LastSignal stores most recent signal
	LastSignal *ExecutionSignal

	// RecentSignals maintains sliding window (last 10)
	RecentSignals []*ExecutionSignal
}

// AddSignal adds a signal to the session
func (s *Session) AddSignal(signal *ExecutionSignal) {
	s.SignalCount++
	s.LastSignal = signal

	// Add to recent signals
	s.RecentSignals = append(s.RecentSignals, signal)

	// Keep only last 10 signals
	if len(s.RecentSignals) > 10 {
		s.RecentSignals = s.RecentSignals[len(s.RecentSignals)-10:]
	}
}

// AdaptationResult describes the outcome of online adaptation
type AdaptationResult struct {
	// Action taken (skipped, reflected, quick-added)
	Action AdaptationAction

	// BulletsAdded is count of new bullets
	BulletsAdded int

	// BulletsUpdated is count of modified bullets
	BulletsUpdated int

	// LatencyMs is processing time
	LatencyMs int64

	// Reason explains the adaptation decision
	Reason string

	// RefinementTriggered indicates if memory management ran
	RefinementTriggered bool
}

// AdaptationAction describes what the adapter did
type AdaptationAction string

const (
	ActionSkip     AdaptationAction = "skip"      // Signal ignored
	ActionReflect  AdaptationAction = "reflect"   // Full reflection cycle
	ActionQuickAdd AdaptationAction = "quick_add" // Direct bullet generation
	ActionUpdate   AdaptationAction = "update"    // Update existing bullets
)
