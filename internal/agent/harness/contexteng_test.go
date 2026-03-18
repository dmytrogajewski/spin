package harness_test

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dmytrogajewski/spin/internal/agent/harness"
	"github.com/dmytrogajewski/spin/internal/events"
	"github.com/dmytrogajewski/spin/internal/message"
	"github.com/dmytrogajewski/spin/internal/tools"
)

// Journey: specs/journeys/JOURNEY-2.7.md.

const (
	compactedMarker     = "[compacted]"
	summarizedMarker    = "[summarized]"
	reminderContent     = "reminder: take action"
	highPressureContent = "a]very long content that simulates high pressure"
)

// fakeCompactor simulates context compaction.
type fakeCompactor struct {
	compactCalled int
	shouldCompact bool
}

func (f *fakeCompactor) Compact(
	_ context.Context, msgs []message.Message,
) ([]message.Message, bool, error) {
	f.compactCalled++

	if !f.shouldCompact {
		return msgs, false, nil
	}

	// Replace first tool message content with compacted marker.
	result := make([]message.Message, len(msgs))
	copy(result, msgs)

	for idx := range result {
		if result[idx].Role == message.RoleTool {
			result[idx].Content = compactedMarker

			break
		}
	}

	return result, true, nil
}

// fakeReminderInjector simulates reminder injection.
type fakeReminderInjector struct {
	injectCalled int
	shouldInject bool
}

func (f *fakeReminderInjector) InjectReminders(
	_ context.Context, _ []message.Message, _ int,
) []message.Message {
	f.injectCalled++

	if !f.shouldInject {
		return nil
	}

	return []message.Message{{
		Role:    message.RoleUser,
		Content: reminderContent,
	}}
}

// fakeObservationSummarizer simulates observation summarization.
type fakeObservationSummarizer struct {
	summarizeCalled int
}

func (f *fakeObservationSummarizer) SummarizeToolResults(
	msgs []message.Message,
) []message.Message {
	f.summarizeCalled++

	result := make([]message.Message, len(msgs))
	copy(result, msgs)

	for idx := range result {
		if result[idx].Role == message.RoleTool {
			result[idx].Content = summarizedMarker
		}
	}

	return result
}

// newContextEngExecutor creates an executor with contexteng options.
func newContextEngExecutor(
	t *testing.T,
	caller harness.Caller,
	dispatcher harness.ToolDispatcher,
	opts ...harness.Option,
) *harness.Executor {
	t.Helper()

	exec, err := harness.NewExecutor(
		testSpec(testMaxTurns), caller, dispatcher, nil, nil,
		slog.Default(), opts...,
	)
	require.NoError(t, err)

	return exec
}

// TestExecute_WithCompactor_CompactsMessages verifies compaction runs each turn.
// Kills mutant: skipping compaction would let context grow unbounded.
func TestExecute_WithCompactor_CompactsMessages(t *testing.T) {
	t.Parallel()

	comp := &fakeCompactor{shouldCompact: true}
	caller := &multiTurnCaller{toolTurns: 2}
	exec := newContextEngExecutor(t, caller, &stubDispatcher{},
		harness.WithCompactor(comp),
	)

	resp, err := exec.Execute(t.Context(), testQuery, nil)

	require.NoError(t, err)
	assert.Equal(t, testOutput, resp.Output)

	// Compactor called once per turn: 2 tool turns + 1 completion turn = 3.
	expectedCompactCalls := 3
	assert.Equal(t, expectedCompactCalls, comp.compactCalled)
}

// TestExecute_NilCompactor_Works verifies loop works without compactor.
// Kills mutant: nil compactor causing panic would crash the loop.
func TestExecute_NilCompactor_Works(t *testing.T) {
	t.Parallel()

	caller := &stubCaller{content: testOutput}
	exec := newContextEngExecutor(t, caller, &stubDispatcher{})

	resp, err := exec.Execute(t.Context(), testQuery, nil)

	require.NoError(t, err)
	assert.Equal(t, testOutput, resp.Output)
	assert.Equal(t, harness.FinishReasonStop, resp.FinishReason)
}

// TestExecute_WithReminder_InjectsMessages verifies reminder injection each turn.
// Kills mutant: not calling injector would miss recovery reminders.
func TestExecute_WithReminder_InjectsMessages(t *testing.T) {
	t.Parallel()

	inj := &fakeReminderInjector{shouldInject: true}
	caller := &multiTurnCaller{toolTurns: 1}
	exec := newContextEngExecutor(t, caller, &stubDispatcher{},
		harness.WithReminderInjector(inj),
	)

	resp, err := exec.Execute(t.Context(), testQuery, nil)

	require.NoError(t, err)

	// Injector called once per turn: 1 tool turn + 1 completion turn = 2.
	expectedInjectCalls := 2
	assert.Equal(t, expectedInjectCalls, inj.injectCalled)

	// Reminder content should appear in messages.
	found := false

	for _, msg := range resp.Messages {
		if msg.Content == reminderContent {
			found = true

			break
		}
	}

	assert.True(t, found, "reminder message should be in response messages")
}

// TestExecute_NilReminder_Works verifies loop works without reminder injector.
// Kills mutant: nil injector causing panic would crash the loop.
func TestExecute_NilReminder_Works(t *testing.T) {
	t.Parallel()

	caller := &stubCaller{content: testOutput}
	exec := newContextEngExecutor(t, caller, &stubDispatcher{})

	resp, err := exec.Execute(t.Context(), testQuery, nil)

	require.NoError(t, err)
	assert.Equal(t, testOutput, resp.Output)
}

// TestExecute_WithObservation_SummarizesToolResults verifies observation summarization.
// Kills mutant: not summarizing would waste context on raw tool output.
func TestExecute_WithObservation_SummarizesToolResults(t *testing.T) {
	t.Parallel()

	obs := &fakeObservationSummarizer{}
	caller := &multiTurnCaller{toolTurns: 2}
	exec := newContextEngExecutor(t, caller, &stubDispatcher{},
		harness.WithObservationSummarizer(obs),
	)

	resp, err := exec.Execute(t.Context(), testQuery, nil)

	require.NoError(t, err)
	assert.Equal(t, testOutput, resp.Output)

	// Summarizer called once per dispatch: 2 tool turns.
	expectedSummarizeCalls := 2
	assert.Equal(t, expectedSummarizeCalls, obs.summarizeCalled)
}

// TestExecute_NilObservation_Works verifies loop works without observation summarizer.
// Kills mutant: nil summarizer causing panic would crash the loop.
func TestExecute_NilObservation_Works(t *testing.T) {
	t.Parallel()

	caller := &stubCaller{content: testOutput}
	exec := newContextEngExecutor(t, caller, &stubDispatcher{})

	resp, err := exec.Execute(t.Context(), testQuery, nil)

	require.NoError(t, err)
	assert.Equal(t, testOutput, resp.Output)
}

// TestExecute_AllContextEngComponents verifies full integration with all components.
// Kills mutant: component interaction bugs would be caught here.
func TestExecute_AllContextEngComponents(t *testing.T) {
	t.Parallel()

	comp := &fakeCompactor{shouldCompact: false}
	inj := &fakeReminderInjector{shouldInject: false}
	obs := &fakeObservationSummarizer{}

	caller := &multiTurnCaller{toolTurns: 2}
	exec := newContextEngExecutor(t, caller, &stubDispatcher{},
		harness.WithCompactor(comp),
		harness.WithReminderInjector(inj),
		harness.WithObservationSummarizer(obs),
	)

	resp, err := exec.Execute(t.Context(), testQuery, nil)

	require.NoError(t, err)
	assert.Equal(t, testOutput, resp.Output)
	assert.Equal(t, harness.FinishReasonStop, resp.FinishReason)

	// All components called appropriately.
	expectedTurns := 3
	assert.Equal(t, expectedTurns, comp.compactCalled)
	assert.Equal(t, expectedTurns, inj.injectCalled)

	expectedDispatches := 2
	assert.Equal(t, expectedDispatches, obs.summarizeCalled)
}

// TestExecute_CompactorReplacesMessages verifies compacted messages reach caller.
// Kills mutant: not updating messages after compaction would waste tokens.
func TestExecute_CompactorReplacesMessages(t *testing.T) {
	t.Parallel()

	comp := &fakeCompactor{shouldCompact: true}

	// Provide history with a tool message that compactor will replace.
	history := []message.Message{
		{Role: message.RoleTool, Content: "original tool output", ToolCallID: "tc_1"},
	}

	caller := &stubCaller{content: testOutput}
	exec := newContextEngExecutor(t, caller, &stubDispatcher{},
		harness.WithCompactor(comp),
	)

	resp, err := exec.Execute(t.Context(), testQuery, history)

	require.NoError(t, err)

	// The tool message in final messages should be compacted.
	found := false

	for _, msg := range resp.Messages {
		if msg.Role == message.RoleTool && msg.Content == compactedMarker {
			found = true

			break
		}
	}

	assert.True(t, found, "compacted marker should appear in response messages")
}

// TestExecute_ObservationModifiesDispatchedMessages verifies that tool results
// from earlier turns are summarized while the latest turn's results remain raw.
// Uses 2 tool turns: the first turn's tool result should be summarized before
// the second LLM call, while the second turn's result stays raw.
func TestExecute_ObservationModifiesDispatchedMessages(t *testing.T) {
	t.Parallel()

	obs := &fakeObservationSummarizer{}
	caller := &multiTurnCaller{toolTurns: 2}
	exec := newContextEngExecutor(t, caller, &stubDispatcher{},
		harness.WithObservationSummarizer(obs),
	)

	resp, err := exec.Execute(t.Context(), testQuery, nil)

	require.NoError(t, err)

	// The first tool result should be summarized (LLM already saw it).
	found := false

	for _, msg := range resp.Messages {
		if msg.Role == message.RoleTool && msg.Content == summarizedMarker {
			found = true

			break
		}
	}

	assert.True(t, found, "summarized marker should appear in earlier tool messages")
}

// TestExecute_ReminderBeforeLLMCall verifies reminders are in messages before the LLM sees them.
// Kills mutant: injecting after LLM call would not influence its output.
func TestExecute_ReminderBeforeLLMCall(t *testing.T) {
	t.Parallel()

	inj := &fakeReminderInjector{shouldInject: true}

	// Use a caller that checks its messages for the reminder.
	checker := &messageCheckCaller{
		checkContent: reminderContent,
		toolTurns:    1,
	}

	exec := newContextEngExecutor(t, checker, &stubDispatcher{},
		harness.WithReminderInjector(inj),
	)

	_, err := exec.Execute(t.Context(), testQuery, nil)

	require.NoError(t, err)

	// On the second call (after first tool turn + reminder injection),
	// the reminder should be visible in messages passed to the caller.
	assert.True(t, checker.foundContent, "reminder should be in messages before LLM call")
}

// waitForEvent drains an event channel until an event of the given type arrives.
// Returns the matched event or fails the test on timeout.
func waitForEvent(
	t *testing.T, ch <-chan events.Event, target events.EventType,
) events.Event {
	t.Helper()

	timeout := time.After(time.Second)

	for {
		select {
		case ev := <-ch:
			if ev.Type == target {
				return ev
			}
		case <-timeout:
			t.Fatalf("timeout waiting for event %s", target)
		}
	}
}

// runPhaseEmitTest creates an emitter-wired executor, runs it, and returns
// the first event matching the target type.
func runPhaseEmitTest(
	t *testing.T, target events.EventType, opts ...harness.Option,
) events.Event {
	t.Helper()

	emitter := events.NewEventEmitter(100)
	_, eventCh, subErr := emitter.Subscribe()
	require.NoError(t, subErr)

	allOpts := append([]harness.Option{harness.WithEmitter(emitter)}, opts...)
	exec := newContextEngExecutor(t,
		&multiTurnCaller{toolTurns: 1}, &stubDispatcher{}, allOpts...,
	)

	_, execErr := exec.Execute(t.Context(), testQuery, nil)
	require.NoError(t, execErr)

	return waitForEvent(t, eventCh, target)
}

// TestExecute_CompactionEmitsEvent verifies compaction emits EventCompactionTriggered.
// Kills mutant: not emitting would leave UI unaware of compaction.
func TestExecute_CompactionEmitsEvent(t *testing.T) {
	t.Parallel()

	ev := runPhaseEmitTest(t, events.EventCompactionTriggered,
		harness.WithCompactor(&fakeCompactor{shouldCompact: true}),
	)

	data, ok := ev.CompactionTriggeredData()
	require.True(t, ok)
	assert.Equal(t, "compacted", data.Stage)
}

// TestExecute_ReminderEmitsEvent verifies reminder injection emits EventReminderInjected.
// Kills mutant: not emitting would leave metrics unaware of injections.
func TestExecute_ReminderEmitsEvent(t *testing.T) {
	t.Parallel()

	ev := runPhaseEmitTest(t, events.EventReminderInjected,
		harness.WithReminderInjector(&fakeReminderInjector{shouldInject: true}),
	)

	data, ok := ev.ReminderInjectedData()
	require.True(t, ok)
	assert.Equal(t, 1, data.Count)
}

// TestExecute_NilEmitter_NoPanic verifies no panic when emitter is nil.
// Kills mutant: not checking nil emitter would cause panic.
func TestExecute_NilEmitter_NoPanic(t *testing.T) {
	t.Parallel()

	comp := &fakeCompactor{shouldCompact: true}
	inj := &fakeReminderInjector{shouldInject: true}
	caller := &multiTurnCaller{toolTurns: 1}
	exec := newContextEngExecutor(t, caller, &stubDispatcher{},
		harness.WithCompactor(comp),
		harness.WithReminderInjector(inj),
	)

	resp, err := exec.Execute(t.Context(), testQuery, nil)
	require.NoError(t, err)
	assert.Equal(t, testOutput, resp.Output)
}

// messageCheckCaller checks if a specific content appears in messages on call #2+.
type messageCheckCaller struct {
	checkContent string
	toolTurns    int
	callCount    int
	foundContent bool
}

func (m *messageCheckCaller) Call(
	_ context.Context, msgs []message.Message, _ []tools.ToolSchema, _ int,
) (string, []message.ToolCall, string, error) {
	m.callCount++

	// Check on second call onwards (after reminder could have been injected).
	if m.callCount > 1 {
		for _, msg := range msgs {
			if msg.Content == m.checkContent {
				m.foundContent = true

				break
			}
		}
	}

	if m.callCount <= m.toolTurns {
		return "", []message.ToolCall{{
			ID:   testToolCallID,
			Type: "function",
			Function: message.ToolCallFunction{
				Name:      testToolName,
				Arguments: "{}",
			},
		}}, "", nil
	}

	return testOutput, nil, "stop", nil
}
