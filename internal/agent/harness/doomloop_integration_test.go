package harness_test

import (
	"context"
	"log/slog"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/dmytrogajewski/spin/internal/agent/harness"
	"github.com/dmytrogajewski/spin/internal/events"
	"github.com/dmytrogajewski/spin/internal/message"
	"github.com/dmytrogajewski/spin/internal/tools"
)

// Journey: specs/journeys/JOURNEY-0.1.md.

// doomLoopCaller returns the same tool call for the first N turns, then completes.
// When mixSuccessAt is > 0, turn #mixSuccessAt returns a *different* tool call
// to break the fingerprint repetition.
type doomLoopCaller struct {
	toolTurns    int
	mixSuccessAt int // 1-based turn to return a different tool call (0 = never).
	callCount    int
	repeatName   string
	repeatArgs   string
	successName  string
	successArgs  string
}

func (d *doomLoopCaller) Call(
	_ context.Context, _ []message.Message, _ []tools.ToolSchema, _ int,
) (string, []message.ToolCall, string, error) {
	d.callCount++

	if d.callCount > d.toolTurns {
		return testOutput, nil, "stop", nil
	}

	name := d.repeatName
	args := d.repeatArgs

	if d.mixSuccessAt > 0 && d.callCount == d.mixSuccessAt {
		name = d.successName
		args = d.successArgs
	}

	return "", []message.ToolCall{{
		ID:   "call_doom_int",
		Type: "function",
		Function: message.ToolCallFunction{
			Name:      name,
			Arguments: args,
		},
	}}, "", nil
}

// TestDoomLoopGuard_DetectsRepeatedFailures feeds the guard multiple consecutive
// identical tool calls through the executor and verifies the loop halts with
// FinishReasonGuard and emits a DoomLoopDetected event.
func TestDoomLoopGuard_DetectsRepeatedFailures(t *testing.T) {
	t.Parallel()

	emitter := events.NewEventEmitter(100)
	_, eventCh, err := emitter.Subscribe()
	require.NoError(t, err)

	guard := harness.NewDoomLoopGuard(harness.DefaultWindowSize, harness.DefaultThreshold)
	guard.SetEmitter(emitter)

	// Caller returns the same tool call every turn; enough turns to exceed threshold.
	caller := &doomLoopCaller{
		toolTurns:  harness.DefaultThreshold + 2,
		repeatName: "shell_command",
		repeatArgs: `{"command":"rm -rf /"}`,
	}

	exec, err := harness.NewExecutor(
		testSpec(harness.DefaultThreshold+5),
		caller,
		&stubDispatcher{},
		[]harness.Guard{guard},
		nil,
		slog.Default(),
		harness.WithEmitter(emitter),
	)
	require.NoError(t, err)

	resp, err := exec.Execute(t.Context(), testQuery, nil)

	require.NoError(t, err)
	require.Equal(t, harness.FinishReasonGuard, resp.FinishReason,
		"loop should halt via guard when doom-loop detected")

	// The guard should have halted at exactly the threshold call.
	require.Equal(t, harness.DefaultThreshold, caller.callCount,
		"caller should have been called exactly threshold times before guard halted")

	// Verify the warning message was injected into conversation messages.
	foundWarning := false

	for _, msg := range resp.Messages {
		if msg.Role == message.RoleSystem && containsSubstring(msg.Content, "Doom-loop detected") {
			foundWarning = true

			break
		}
	}

	require.True(t, foundWarning, "doom-loop warning message should appear in response messages")

	// Verify a DoomLoopDetected event was emitted.
	timeout := time.After(time.Second)

	select {
	case ev := <-eventCh:
		require.Equal(t, events.EventDoomLoopDetected, ev.Type)

		data, ok := ev.DoomLoopDetectedData()
		require.True(t, ok)
		require.Equal(t, "shell_command", data.ToolName)
		require.Equal(t, harness.DefaultThreshold, data.Count)
	case <-timeout:
		t.Fatal("timeout waiting for doom-loop detection event")
	}
}

// TestReminderInjector_InjectsAfterDoomLoopDetected verifies that when a doom-loop
// triggers and halts the loop, any reminder injector that was active has already
// injected its reminders into the conversation messages during earlier turns.
// This tests the integration of both DoomLoopGuard (as a Guard) and
// ReminderInjector working together through the Executor.
func TestReminderInjector_InjectsAfterDoomLoopDetected(t *testing.T) {
	t.Parallel()

	guard := harness.NewDoomLoopGuard(harness.DefaultWindowSize, harness.DefaultThreshold)

	// Reminder injector that always injects a reminder each turn.
	inj := &fakeReminderInjector{shouldInject: true}

	// Caller returns the same tool call every turn.
	caller := &doomLoopCaller{
		toolTurns:  harness.DefaultThreshold + 2,
		repeatName: "shell_command",
		repeatArgs: `{"command":"ls -la"}`,
	}

	exec, err := harness.NewExecutor(
		testSpec(harness.DefaultThreshold+5),
		caller,
		&stubDispatcher{},
		[]harness.Guard{guard},
		nil,
		slog.Default(),
		harness.WithReminderInjector(inj),
	)
	require.NoError(t, err)

	resp, err := exec.Execute(t.Context(), testQuery, nil)

	require.NoError(t, err)
	require.Equal(t, harness.FinishReasonGuard, resp.FinishReason,
		"loop should halt via guard when doom-loop detected")

	// The reminder injector should have been called on every turn that ran.
	// The loop runs threshold turns (turns 0..threshold-1); reminder runs
	// at the start of each turn before the LLM call.
	require.Equal(t, harness.DefaultThreshold, inj.injectCalled,
		"reminder injector should be called once per turn that executed")

	// Verify reminder content appears in the final messages.
	foundReminder := false

	for _, msg := range resp.Messages {
		if msg.Content == reminderContent {
			foundReminder = true

			break
		}
	}

	require.True(t, foundReminder,
		"reminder content should be present in conversation messages when doom-loop halts")
}

// TestDoomLoopGuard_ResetsOnSuccess feeds the guard repeated identical tool calls
// below the threshold, then a different (successful) tool call, then resumes the
// repeated calls. The different call evicts older fingerprints from the sliding
// window, so the guard should NOT trigger despite the total count of identical
// calls exceeding the threshold.
func TestDoomLoopGuard_ResetsOnSuccess(t *testing.T) {
	t.Parallel()

	// Use a small window so that one different call pushes out old fingerprints.
	const (
		windowSize = 4
		threshold  = 3
	)

	guard := harness.NewDoomLoopGuard(windowSize, threshold)

	// Pattern: 2 identical calls, then 1 different call (fills window slots),
	// then 2 more identical calls. Because window=4 and the different call
	// plus the 2 new identical ones push the first identical one out, the
	// max count should stay below threshold.
	//
	// We actually need windowSize different calls to fully evict. Use a caller
	// that mixes at the right spot.
	//
	// Strategy: 2 repeated, then windowSize different calls to flush, then 2 repeated.
	// Total tool turns = 2 + windowSize + 2 = 8. Threshold = 3 so the guard
	// should NOT fire since the window only ever contains 2 identical fingerprints.
	totalToolTurns := 2 + windowSize + 2

	caller := &multiPhaseResetCaller{
		repeatName:      "shell_command",
		repeatArgs:      `{"command":"rm -rf /"}`,
		repeatBeforeMix: 2,
		mixCount:        windowSize,
		repeatAfterMix:  2,
		totalToolTurns:  totalToolTurns,
	}

	exec, err := harness.NewExecutor(
		testSpec(totalToolTurns+5),
		caller,
		&stubDispatcher{},
		[]harness.Guard{guard},
		nil,
		slog.Default(),
	)
	require.NoError(t, err)

	resp, err := exec.Execute(t.Context(), testQuery, nil)

	require.NoError(t, err)
	require.Equal(t, harness.FinishReasonStop, resp.FinishReason,
		"loop should complete normally when different calls break the doom-loop pattern")

	// Caller should have been called for all tool turns + 1 completion turn.
	require.Equal(t, totalToolTurns+1, caller.callCount,
		"all turns should execute without guard halt")
}

// multiPhaseResetCaller returns tool calls in three phases:
// 1. repeatBeforeMix identical calls
// 2. mixCount different calls (each unique)
// 3. repeatAfterMix identical calls (same as phase 1)
// After totalToolTurns it returns completion.
type multiPhaseResetCaller struct {
	repeatName      string
	repeatArgs      string
	repeatBeforeMix int
	mixCount        int
	repeatAfterMix  int
	totalToolTurns  int
	callCount       int
}

func (m *multiPhaseResetCaller) Call(
	_ context.Context, _ []message.Message, _ []tools.ToolSchema, _ int,
) (string, []message.ToolCall, string, error) {
	m.callCount++

	if m.callCount > m.totalToolTurns {
		return testOutput, nil, "stop", nil
	}

	name := m.repeatName
	args := m.repeatArgs

	// Phase 2: different calls to flush the window.
	if m.callCount > m.repeatBeforeMix && m.callCount <= m.repeatBeforeMix+m.mixCount {
		idx := m.callCount - m.repeatBeforeMix
		name = "different_tool"
		args = `{"idx":"` + strconv.Itoa(idx) + `"}`
	}

	return "", []message.ToolCall{{
		ID:   "call_reset",
		Type: "function",
		Function: message.ToolCallFunction{
			Name:      name,
			Arguments: args,
		},
	}}, "", nil
}

// containsSubstring is a helper to check substring presence.
func containsSubstring(s, substr string) bool {
	return len(s) >= len(substr) && searchSubstring(s, substr)
}

func searchSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}

	return false
}
