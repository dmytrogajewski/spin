package child

// Journey: specs/journeys/JOURNEY-018-spawn-process-children-from-the-parent.md.

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/dmytrogajewski/spin/internal/agent/subagent"
	"github.com/dmytrogajewski/spin/internal/events"
)

func TestNewExecutor_EmitsSpawnAndComplete(t *testing.T) {
	t.Parallel()

	emitter := events.NewEventEmitter(16)
	_, evts, err := emitter.Subscribe()
	require.NoError(t, err)

	spec, lookupErr := subagent.Lookup(subagent.NameExplorer)
	require.NoError(t, lookupErr)

	exec := NewExecutor(requireSpin(t), "", emitter, nil)
	summary, runErr := exec(testCtx(t), spec, spawnQuery)
	require.NoError(t, runErr)
	require.Equal(t, spawnQuery, summary)

	got := drainEvents(evts)
	spawn := findEvent(t, got, events.EventSubagentSpawn)
	data, ok := spawn.SubagentSpawnData()
	require.True(t, ok)
	require.Equal(t, spec.Name, data.AgentType)
	require.Equal(t, spawnQuery, data.Query)

	complete := findEvent(t, got, events.EventSubagentComplete)
	done, doneOK := complete.SubagentCompleteData()
	require.True(t, doneOK)
	require.Equal(t, spec.Name, done.AgentType)
	require.Equal(t, spawnQuery, done.Summary)
}

func drainEvents(evts <-chan events.Event) []events.Event {
	var got []events.Event

	for {
		select {
		case ev := <-evts:
			got = append(got, ev)
		default:
			return got
		}
	}
}

func findEvent(t *testing.T, got []events.Event, want events.EventType) events.Event {
	t.Helper()

	for _, ev := range got {
		if ev.Type == want {
			return ev
		}
	}

	t.Fatalf("missing event %s", want)

	return events.Event{}
}
