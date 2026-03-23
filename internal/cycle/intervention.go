package cycle

import "time"

// Message represents a conversation message for interventions.
type Message interface {
	GetRole() string
	GetContent() string
	GetTimestamp() time.Time
}

// EventData represents the data payload for detection events.
type EventData map[string]any

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
