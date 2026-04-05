// Package reminder provides context-aware system reminder injection
// for the harness loop. Detectors monitor conversation state and
// inject corrective reminders when problematic patterns are identified.
package reminder

import "github.com/dmytrogajewski/spin/internal/message"

// CheckContext provides conversation state for detector evaluation.
type CheckContext struct {
	// Turn is the current loop iteration (zero-based).
	Turn int

	// Messages is the conversation history.
	Messages []message.Message

	// LastToolFailed indicates whether the most recent tool call failed.
	LastToolFailed bool

	// ConsecutiveReads counts consecutive read-type tool operations.
	ConsecutiveReads int

	// HasIncompleteTodos indicates unfinished work items were detected.
	HasIncompleteTodos bool

	// LastAssistantEmpty indicates the last assistant message had empty content.
	LastAssistantEmpty bool

	// LastToolDenied indicates the agent retried a tool call that was denied by approval.
	LastToolDenied bool

	// AllTodosComplete indicates all work items have been completed.
	AllTodosComplete bool

	// PlanApprovedNotExecuted indicates a plan was approved but remains unexecuted.
	PlanApprovedNotExecuted bool

	// HasUnprocessedSubagentResults indicates subagent results were returned but not processed.
	HasUnprocessedSubagentResults bool
}

// Detector evaluates a specific conversation pattern and determines
// whether a reminder should be injected.
type Detector interface {
	// Name returns the unique identifier for this detector.
	Name() string

	// Check evaluates whether the detector's condition is met.
	Check(ctx CheckContext) bool

	// MaxFires returns the maximum number of times this detector can fire per query.
	MaxFires() int
}
