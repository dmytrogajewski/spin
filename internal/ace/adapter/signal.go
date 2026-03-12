package adapter

import "time"

const maxRecentSignals = 50

// SignalType categorizes execution signals.
type SignalType string

const (
	// SignalTypeTest defines a SignalTypeTest constant.
	SignalTypeTest    SignalType = "test"
	// SignalTypeBuild defines a SignalTypeBuild constant.
	SignalTypeBuild   SignalType = "build"
	// SignalTypeLint represents a lint signal.
	SignalTypeLint    SignalType = "lint"
	// SignalTypeError represents an error signal.
	SignalTypeError   SignalType = "error"
	// SignalTypeToolUse represents a tool use signal.
	SignalTypeToolUse SignalType = "tool_use"
	// SignalTypeUser represents a user signal.
	SignalTypeUser    SignalType = "user"
)

// SignalOutcome indicates signal polarity.
type SignalOutcome string

const (
	// OutcomeSuccess defines a OutcomeSuccess constant.
	OutcomeSuccess SignalOutcome = "success"
	// OutcomeFailure defines a OutcomeFailure constant.
	OutcomeFailure SignalOutcome = "failure"
	// OutcomeNeutral represents a neutral outcome.
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
	s.RecentSignals = trimToLastN(s.RecentSignals, maxRecentSignals)
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
	// ActionSkip defines a ActionSkip constant.
	ActionSkip     AdaptationAction = "skip"
	// ActionReflect defines a ActionReflect constant.
	ActionReflect  AdaptationAction = "reflect"
	// ActionQuickAdd represents a quick add action.
	ActionQuickAdd AdaptationAction = "quick_add"
	// ActionUpdate represents an update action.
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
