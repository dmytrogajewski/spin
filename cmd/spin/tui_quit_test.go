package main

import (
	"context"
	"testing"
	"time"

	"github.com/dmytrogajewski/spin/internal/events"
)

// Journey: specs/bugs/BUG-tui-exit-clear.md
// Journey: specs/journeys/JOURNEY-025-parent-shutdown-cancels-children.md.
//
// Ctrl+C closes the prompt/input channel but used to leave the conversation
// event loop running. runTUI then blocked on eventDone until a second SIGINT
// canceled ctx — so one Ctrl+C cleared the screen and did not exit.

func TestStopTUILoop_UnblocksEventLoopWithoutSecondSignal(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	eventCh := make(chan events.Event)
	eventDone := startEventLoop(ctx, eventCh, nil, nil, nil)

	quit := make(chan struct{})

	go func() {
		stopTUILoop(cancel, eventDone, nil)
		close(quit)
	}()

	select {
	case <-quit:
	case <-time.After(time.Second):
		t.Fatal("single quit must cancel the event loop; hung waiting for events like Ctrl+C without SIGINT")
	}
}

func TestStopTUILoop_RunsCloserForSessionEnd(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	eventCh := make(chan events.Event)
	eventDone := startEventLoop(ctx, eventCh, nil, nil, nil)

	closed := false

	stopTUILoop(cancel, eventDone, func() { closed = true })

	if !closed {
		t.Fatal("teardown must Close conversation so SESSION_END runs on Ctrl-C")
	}
}

func TestStartEventLoop_StaysUpUntilCanceled(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	eventCh := make(chan events.Event)
	eventDone := startEventLoop(ctx, eventCh, nil, nil, nil)

	select {
	case <-eventDone:
		t.Fatal("event loop exited while conversation stream is still open")
	case <-time.After(50 * time.Millisecond):
	}
}
