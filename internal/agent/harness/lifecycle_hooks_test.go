package harness_test

// Journey: specs/journeys/JOURNEY-008-wire-every-defined-lifecycle-hook-event.md.

import (
	"context"
	"slices"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/dmytrogajewski/spin/internal/agent/harness"
	"github.com/dmytrogajewski/spin/internal/message"
	"github.com/dmytrogajewski/spin/internal/safety/hooks"
)

type recordHooks struct {
	mu     sync.Mutex
	events []hooks.Event
}

func (r *recordHooks) Execute(_ context.Context, event hooks.Event, _ hooks.EventContext) hooks.HookResult {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.events = append(r.events, event)

	return hooks.HookResult{}
}

func (r *recordHooks) saw(event hooks.Event) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	return slices.Contains(r.events, event)
}

type orderCompactor struct {
	hooks *recordHooks
	saw   bool
}

func (o *orderCompactor) Compact(_ context.Context, msgs []message.Message) ([]message.Message, bool, error) {
	o.saw = o.hooks.saw(hooks.EventPreCompact)

	out := append([]message.Message(nil), msgs...)
	if len(out) > 0 {
		out[0].Content = compactedMarker
	}

	return out, true, nil
}

func TestExecute_PreCompactBeforeRewrite(t *testing.T) {
	t.Parallel()

	rec := &recordHooks{}
	comp := &orderCompactor{hooks: rec}
	exec := newContextEngExecutor(t, &stubCaller{content: testOutput}, &stubDispatcher{},
		harness.WithCompactor(comp),
		harness.WithHookRunner(rec),
	)

	_, err := exec.Execute(t.Context(), testQuery, nil)
	require.NoError(t, err)
	require.True(t, comp.saw, "PRE_COMPACT must run before Compact rewrites history")
}

func TestExecute_FiresStopOnLoopEnd(t *testing.T) {
	t.Parallel()

	rec := &recordHooks{}
	exec := newContextEngExecutor(t, &stubCaller{content: testOutput}, &stubDispatcher{},
		harness.WithHookRunner(rec),
	)

	_, err := exec.Execute(t.Context(), testQuery, nil)
	require.NoError(t, err)
	require.True(t, rec.saw(hooks.EventStop), "STOP must fire when the parent loop ends")
}
