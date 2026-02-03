package adapter

import "time"

// SignalType categorizes execution signals.
type SignalType string

const (
	SignalTypeTest    SignalType = "test"
	SignalTypeBuild   SignalType = "build"
	SignalTypeLint    SignalType = "lint"
	SignalTypeError   SignalType = "error"
	SignalTypeToolUse SignalType = "tool_use"
	SignalTypeUser    SignalType = "user"
)

// SignalOutcome indicates signal polarity.
type SignalOutcome string

const (
	OutcomeSuccess SignalOutcome = "success"
	OutcomeFailure SignalOutcome = "failure"
	OutcomeNeutral SignalOutcome = "neutral"
)

// ExecutionSignal represents a feedback event from agent execution.
type ExecutionSignal struct {
	SignalType SignalType
	Context    string
	Outcome    SignalOutcome
	Details    map[string]string
	Timestamp  time.Time
	SessionID  string
}

// Session tracks online learning state.
type Session struct {
	ID            string
	StartTime     time.Time
	SignalCount   int
	UpdateCount   int
	LastSignal    *ExecutionSignal
	RecentSignals []*ExecutionSignal
}

// AddSignal adds a signal to the session.
func (s *Session) AddSignal(signal *ExecutionSignal) {
	s.SignalCount++
	s.LastSignal = signal
	s.RecentSignals = append(s.RecentSignals, signal)
	s.RecentSignals = trimToLastN(s.RecentSignals, 10)
}

// trimToLastN keeps only the last n elements of a slice.
func trimToLastN(signals []*ExecutionSignal, n int) []*ExecutionSignal {
	if len(signals) <= n {
		return signals
	}
	return signals[len(signals)-n:]
}

// AdaptationAction describes what the adapter did.
type AdaptationAction string

const (
	ActionSkip     AdaptationAction = "skip"
	ActionReflect  AdaptationAction = "reflect"
	ActionQuickAdd AdaptationAction = "quick_add"
	ActionUpdate   AdaptationAction = "update"
)

// AdaptationResult describes the outcome of online adaptation.
type AdaptationResult struct {
	Action              AdaptationAction
	BulletsAdded        int
	BulletsUpdated      int
	LatencyMs           int64
	Reason              string
	RefinementTriggered bool
}
