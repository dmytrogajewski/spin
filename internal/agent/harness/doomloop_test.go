package harness_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dmytrogajewski/spin/internal/agent/harness"
	"github.com/dmytrogajewski/spin/internal/events"
	"github.com/dmytrogajewski/spin/internal/message"
)

// Journey: specs/journeys/JOURNEY-2.2.md.

const (
	doomToolName = "shell_command"
	doomToolArgs = `{"command":"ls -la"}`
)

func makeDoomToolCalls(name, args string) []message.ToolCall {
	return []message.ToolCall{{
		ID:   "call_doom",
		Type: "function",
		Function: message.ToolCallFunction{
			Name:      name,
			Arguments: args,
		},
	}}
}

// TestDoomLoopGuard_ThresholdTriggersHalt verifies that reaching the threshold halts.
// Kills mutant: not checking threshold would allow infinite loops.
func TestDoomLoopGuard_ThresholdTriggersHalt(t *testing.T) {
	t.Parallel()

	guard := harness.NewDoomLoopGuard(harness.DefaultWindowSize, harness.DefaultThreshold)
	calls := makeDoomToolCalls(doomToolName, doomToolArgs)

	// First two calls should not halt.
	for range harness.DefaultThreshold - 1 {
		injected, halt, err := guard.Check(t.Context(), harness.NewIterationContext(nil), "", calls)
		require.NoError(t, err)
		assert.False(t, halt)
		assert.Empty(t, injected)
	}

	// Third call triggers halt.
	injected, halt, err := guard.Check(t.Context(), harness.NewIterationContext(nil), "", calls)

	require.NoError(t, err)
	assert.True(t, halt, "should halt at threshold")
	require.Len(t, injected, 1)
	assert.Contains(t, injected[0].Content, "Doom-loop detected")
	assert.Equal(t, message.RoleSystem, injected[0].Role)
}

// TestDoomLoopGuard_BelowThresholdNoHalt verifies that fewer calls do not trigger.
// Kills mutant: triggering at threshold-1 would be too aggressive.
func TestDoomLoopGuard_BelowThresholdNoHalt(t *testing.T) {
	t.Parallel()

	guard := harness.NewDoomLoopGuard(harness.DefaultWindowSize, harness.DefaultThreshold)
	calls := makeDoomToolCalls(doomToolName, doomToolArgs)

	for range harness.DefaultThreshold - 1 {
		_, halt, err := guard.Check(t.Context(), harness.NewIterationContext(nil), "", calls)
		require.NoError(t, err)
		assert.False(t, halt)
	}
}

// TestDoomLoopGuard_WindowEviction verifies old fingerprints are evicted.
// Kills mutant: not evicting would accumulate stale fingerprints.
func TestDoomLoopGuard_WindowEviction(t *testing.T) {
	t.Parallel()

	const windowSize = 4

	const threshold = 3

	guard := harness.NewDoomLoopGuard(windowSize, threshold)
	calls := makeDoomToolCalls(doomToolName, doomToolArgs)

	// Record 2 identical calls.
	for range 2 {
		_, halt, err := guard.Check(t.Context(), harness.NewIterationContext(nil), "", calls)
		require.NoError(t, err)
		assert.False(t, halt)
	}

	// Fill the window with different calls to push out the first two.
	for idx := range windowSize {
		differentCalls := makeDoomToolCalls("other_tool", `{"idx":"`+string(rune('a'+idx))+`"}`)

		_, halt, err := guard.Check(t.Context(), harness.NewIterationContext(nil), "", differentCalls)
		require.NoError(t, err)
		assert.False(t, halt)
	}

	// Now the original fingerprint should be evicted; calling twice more should not trigger.
	for range 2 {
		_, halt, err := guard.Check(t.Context(), harness.NewIterationContext(nil), "", calls)
		require.NoError(t, err)
		assert.False(t, halt)
	}
}

// TestDoomLoopGuard_DifferentFingerprintsNoInterference verifies independence.
// Kills mutant: mixing up fingerprints would cause false positives.
func TestDoomLoopGuard_DifferentFingerprintsNoInterference(t *testing.T) {
	t.Parallel()

	guard := harness.NewDoomLoopGuard(harness.DefaultWindowSize, harness.DefaultThreshold)

	// Alternate between two different tools, each called threshold-1 times.
	for range harness.DefaultThreshold - 1 {
		callsA := makeDoomToolCalls("tool_a", `{"a":1}`)
		callsB := makeDoomToolCalls("tool_b", `{"b":2}`)

		_, haltA, errA := guard.Check(t.Context(), harness.NewIterationContext(nil), "", callsA)
		require.NoError(t, errA)

		_, haltB, errB := guard.Check(t.Context(), harness.NewIterationContext(nil), "", callsB)
		require.NoError(t, errB)

		// Neither should trigger because each has fewer than threshold.
		if haltA || haltB {
			t.Fatal("different fingerprints should not interfere")
		}
	}
}

// TestDoomLoopGuard_EmptyToolCalls verifies no-op for empty tool calls.
// Kills mutant: processing empty slice would waste cycles or panic.
func TestDoomLoopGuard_EmptyToolCalls(t *testing.T) {
	t.Parallel()

	guard := harness.NewDoomLoopGuard(harness.DefaultWindowSize, harness.DefaultThreshold)

	injected, halt, err := guard.Check(t.Context(), harness.NewIterationContext(nil), "", nil)

	require.NoError(t, err)
	assert.False(t, halt)
	assert.Empty(t, injected)
}

// TestDoomLoopGuard_DefaultValues verifies that zero values use defaults.
// Kills mutant: zero windowSize would create unusable guard.
func TestDoomLoopGuard_DefaultValues(t *testing.T) {
	t.Parallel()

	guard := harness.NewDoomLoopGuard(0, 0)
	calls := makeDoomToolCalls(doomToolName, doomToolArgs)

	// Should behave with defaults: needs DefaultThreshold calls to trigger.
	for range harness.DefaultThreshold - 1 {
		_, halt, err := guard.Check(t.Context(), harness.NewIterationContext(nil), "", calls)
		require.NoError(t, err)
		assert.False(t, halt)
	}

	_, halt, err := guard.Check(t.Context(), harness.NewIterationContext(nil), "", calls)

	require.NoError(t, err)
	assert.True(t, halt, "should halt at default threshold")
}

// TestDoomLoopGuard_WarningContainsToolName verifies the warning message details.
// Kills mutant: generic warning without tool name would reduce debuggability.
func TestDoomLoopGuard_WarningContainsToolName(t *testing.T) {
	t.Parallel()

	guard := harness.NewDoomLoopGuard(harness.DefaultWindowSize, harness.DefaultThreshold)
	calls := makeDoomToolCalls(doomToolName, doomToolArgs)

	for range harness.DefaultThreshold - 1 {
		_, _, err := guard.Check(t.Context(), harness.NewIterationContext(nil), "", calls)
		require.NoError(t, err)
	}

	injected, halt, err := guard.Check(t.Context(), harness.NewIterationContext(nil), "", calls)

	require.NoError(t, err)
	require.True(t, halt)
	require.Len(t, injected, 1)
	assert.Contains(t, injected[0].Content, doomToolName)
}

// TestDoomLoopGuard_EmitsEvent verifies that doom-loop detection emits an event.
// Kills mutant: not emitting would leave metrics unaware of doom-loop detection.
func TestDoomLoopGuard_EmitsEvent(t *testing.T) {
	t.Parallel()

	emitter := events.NewEventEmitter(100)
	_, eventCh, err := emitter.Subscribe()
	require.NoError(t, err)

	guard := harness.NewDoomLoopGuard(harness.DefaultWindowSize, harness.DefaultThreshold)
	guard.SetEmitter(emitter)

	calls := makeDoomToolCalls(doomToolName, doomToolArgs)
	iterCtx := harness.NewIterationContext(nil)

	for range harness.DefaultThreshold {
		_, _, checkErr := guard.Check(t.Context(), iterCtx, "", calls)
		require.NoError(t, checkErr)
	}

	// Should receive a doom-loop detected event.
	timeout := time.After(time.Second)

	select {
	case ev := <-eventCh:
		require.Equal(t, events.EventDoomLoopDetected, ev.Type)

		data, ok := ev.DoomLoopDetectedData()
		require.True(t, ok)
		assert.Equal(t, doomToolName, data.ToolName)
		assert.Equal(t, harness.DefaultThreshold, data.Count)
		assert.NotEmpty(t, data.Fingerprint)
	case <-timeout:
		t.Fatal("timeout waiting for doom-loop event")
	}
}

// TestDoomLoopGuard_NilEmitter_NoPanic verifies no panic without emitter.
// Kills mutant: not checking nil emitter would cause panic.
func TestDoomLoopGuard_NilEmitter_NoPanic(t *testing.T) {
	t.Parallel()

	guard := harness.NewDoomLoopGuard(harness.DefaultWindowSize, harness.DefaultThreshold)
	calls := makeDoomToolCalls(doomToolName, doomToolArgs)

	for range harness.DefaultThreshold {
		_, _, err := guard.Check(t.Context(), harness.NewIterationContext(nil), "", calls)
		require.NoError(t, err)
	}
}
