package main

import (
	"testing"

	"github.com/dmytrogajewski/spin/internal/events"
)

// Journey: specs/bugs/BUG-tui-context-counter.md.

type fakeTokenSink struct {
	got []int64
}

func (f *fakeTokenSink) SetTokenCount(n int64) { f.got = append(f.got, n) }

type fakeTokenSource struct {
	estimate int
}

func (f *fakeTokenSource) GetTokenCount() int { return f.estimate }

func TestTokenCounter_RealUsageWinsOverEstimate(t *testing.T) {
	t.Parallel()

	sink := &fakeTokenSink{}
	source := &fakeTokenSource{estimate: 13}
	tc := &tokenCounter{}

	tc.update(events.Event{
		Type: events.EventTurnProgress,
		Data: events.TurnEventData{TokensUsed: 12345},
	}, sink, source)

	// Completion events must not stomp real usage with the tiny estimate.
	tc.update(events.Event{Type: events.EventContentComplete}, sink, source)
	tc.update(events.Event{Type: events.EventTurnComplete}, sink, source)

	if len(sink.got) != 1 || sink.got[0] != 12345 {
		t.Fatalf("token counts = %v, want exactly [12345]", sink.got)
	}
}

func TestTokenCounter_FallsBackToEstimateWithoutRealUsage(t *testing.T) {
	t.Parallel()

	sink := &fakeTokenSink{}
	source := &fakeTokenSource{estimate: 42}
	tc := &tokenCounter{}

	tc.update(events.Event{Type: events.EventTurnComplete}, sink, source)

	if len(sink.got) != 1 || sink.got[0] != 42 {
		t.Fatalf("token counts = %v, want [42]", sink.got)
	}
}

func TestTokenCounter_IgnoresZeroUsage(t *testing.T) {
	t.Parallel()

	sink := &fakeTokenSink{}
	source := &fakeTokenSource{estimate: 42}
	tc := &tokenCounter{}

	tc.update(events.Event{
		Type: events.EventTurnProgress,
		Data: events.TurnEventData{TokensUsed: 0},
	}, sink, source)

	if len(sink.got) != 0 {
		t.Fatalf("zero usage must not update the counter, got %v", sink.got)
	}
}
