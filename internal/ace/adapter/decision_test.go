package adapter

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDecideAction_TestFailure(t *testing.T) {
	signal := ExecutionSignal{
		SignalType: SignalTypeTest,
		Outcome:    OutcomeFailure,
		Context:    "Test failed",
	}

	action, reason := decideAction(signal)

	assert.Equal(t, ActionReflect, action)
	assert.Contains(t, reason, "Test failure")
}

func TestDecideAction_BuildFailure(t *testing.T) {
	signal := ExecutionSignal{
		SignalType: SignalTypeBuild,
		Outcome:    OutcomeFailure,
		Context:    "Build failed",
	}

	action, reason := decideAction(signal)

	assert.Equal(t, ActionQuickAdd, action)
	assert.Contains(t, reason, "Build error")
}

func TestDecideAction_ErrorFailure(t *testing.T) {
	signal := ExecutionSignal{
		SignalType: SignalTypeError,
		Outcome:    OutcomeFailure,
		Context:    "Runtime error",
	}

	action, reason := decideAction(signal)

	assert.Equal(t, ActionQuickAdd, action)
	assert.Contains(t, reason, "error")
}

func TestDecideAction_LintFailure(t *testing.T) {
	signal := ExecutionSignal{
		SignalType: SignalTypeLint,
		Outcome:    OutcomeFailure,
		Context:    "Lint error",
	}

	action, reason := decideAction(signal)

	assert.Equal(t, ActionQuickAdd, action)
	assert.Contains(t, reason, "Lint")
}

func TestDecideAction_UserCorrection(t *testing.T) {
	signal := ExecutionSignal{
		SignalType: SignalTypeUser,
		Outcome:    OutcomeFailure,
		Context:    "User correction",
	}

	action, reason := decideAction(signal)

	assert.Equal(t, ActionReflect, action)
	assert.Contains(t, reason, "User")
}

func TestDecideAction_ToolUseSuccess(t *testing.T) {
	signal := ExecutionSignal{
		SignalType: SignalTypeToolUse,
		Outcome:    OutcomeSuccess,
		Context:    "Tool executed successfully",
	}

	action, reason := decideAction(signal)

	assert.Equal(t, ActionSkip, action)
	assert.Contains(t, reason, "success")
}

func TestDecideAction_TestSuccess(t *testing.T) {
	signal := ExecutionSignal{
		SignalType: SignalTypeTest,
		Outcome:    OutcomeSuccess,
		Context:    "Test passed",
	}

	action, reason := decideAction(signal)

	assert.Equal(t, ActionSkip, action)
	assert.Contains(t, reason, "success")
}

func TestDecideAction_NeutralOutcome(t *testing.T) {
	signal := ExecutionSignal{
		SignalType: SignalTypeTest,
		Outcome:    OutcomeNeutral,
		Context:    "Neutral result",
	}

	action, reason := decideAction(signal)

	assert.Equal(t, ActionSkip, action)
	assert.Contains(t, reason, "low-priority")
}
