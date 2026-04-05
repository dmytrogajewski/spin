package harness

import (
	"github.com/dmytrogajewski/spin/internal/ace/trajectory"
	"github.com/dmytrogajewski/spin/internal/message"
)

// IterationContext carries per-query mutable state across loop iterations.
type IterationContext struct {
	// Turn is the current iteration number (zero-based).
	Turn int

	// Messages is the conversation history for this execution.
	Messages []message.Message

	// NudgeCount tracks consecutive error-recovery nudges injected.
	NudgeCount int

	// GuardFlags stores one-shot flags for guard interventions.
	GuardFlags map[string]bool

	// TrajectoryCtx is the progressive execution context for this query.
	// Created once per Execute() call; shared across all middleware hooks.
	TrajectoryCtx *trajectory.Context
}

// NewIterationContext creates an IterationContext from initial messages.
func NewIterationContext(messages []message.Message) *IterationContext {
	return &IterationContext{
		Messages:   messages,
		GuardFlags: make(map[string]bool),
	}
}
