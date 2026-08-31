package conversation

// Journey: specs/journeys/JOURNEY-008-wire-every-defined-lifecycle-hook-event.md.

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/dmytrogajewski/spin/internal/safety/hooks"
)

type recordHooks struct {
	events []hooks.Event
}

func (r *recordHooks) Execute(_ context.Context, event hooks.Event, _ hooks.EventContext) hooks.HookResult {
	r.events = append(r.events, event)

	return hooks.HookResult{}
}

func TestClose_FiresStopThenSessionEnd(t *testing.T) {
	t.Parallel()

	rec := &recordHooks{}
	conv := &Conversation{hookRunner: rec, id: "sess-close", workDir: t.TempDir()}

	require.NoError(t, conv.Close(context.Background()))
	require.Equal(t, []hooks.Event{hooks.EventStop, hooks.EventSessionEnd}, rec.events)
}

func TestClose_CanceledContextStillFiresSessionEnd(t *testing.T) {
	t.Parallel()

	rec := &recordHooks{}
	conv := &Conversation{hookRunner: rec, id: "sess-cancel", workDir: t.TempDir()}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	require.NoError(t, conv.Close(ctx))
	require.Equal(t, []hooks.Event{hooks.EventStop, hooks.EventSessionEnd}, rec.events)
}

func TestClose_IdempotentHooks(t *testing.T) {
	t.Parallel()

	rec := &recordHooks{}
	conv := &Conversation{hookRunner: rec, id: "sess-once", workDir: t.TempDir()}

	require.NoError(t, conv.Close(context.Background()))
	require.NoError(t, conv.Close(context.Background()))
	require.Equal(t, []hooks.Event{hooks.EventStop, hooks.EventSessionEnd}, rec.events)
}
