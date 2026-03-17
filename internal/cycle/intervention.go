package cycle

import (
	"context"
	"time"
)

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
type EventData map[string]any

// Intervention defines the interface for cycle-breaking strategies.
type Intervention interface {
	Apply(ctx context.Context, messages []Message) ([]Message, error)
	Name() string
}

// Tracker defines the interface for cycle detection implementations.
type Tracker interface {
	Record(snapshot Snapshot)
	Check() (Result, error)
	GetHistory() []Snapshot
	Reset()
}

// PatternAnalyzer defines the interface for pattern analysis implementations.
type PatternAnalyzer interface {
	AnalyzePatterns(history []Snapshot) []PatternResult
}
