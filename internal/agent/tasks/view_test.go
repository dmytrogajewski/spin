package tasks

// Journey: specs/journeys/JOURNEY-021-unified-task-view.md.

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTypedID_PrefixesKind(t *testing.T) {
	t.Parallel()

	require.Equal(t, "agent:abc1234", TypedID(KindAgent, "abc1234"))
}

func TestSplitID_TypedAndUntyped(t *testing.T) {
	t.Parallel()

	kind, raw := SplitID("shell:abc1234")
	require.Equal(t, KindShell, kind)
	require.Equal(t, "abc1234", raw)

	kind, raw = SplitID("task-1")
	require.Empty(t, kind)
	require.Equal(t, "task-1", raw)
}

func TestMerge_Empty(t *testing.T) {
	t.Parallel()

	require.Empty(t, Merge(nil, nil))
}

func TestMerge_AgentOnly(t *testing.T) {
	t.Parallel()

	got := Merge([]Record{{ID: "task-1", Spec: testSpecExplorer, State: StateWorking}}, nil)
	require.Equal(t, []Row{{
		Kind: KindAgent, ID: TypedID(KindAgent, "task-1"),
		Spec: testSpecExplorer, State: StateWorking,
	}}, got)
}

func TestMerge_ShellOnly(t *testing.T) {
	t.Parallel()

	got := Merge(nil, []ShellSnapshot{{ID: "abc1234", Command: "sleep 300", State: "running"}})
	require.Equal(t, []Row{{
		Kind: KindShell, ID: TypedID(KindShell, "abc1234"),
		Spec: "sleep 300", State: "running",
	}}, got)
}

func TestMerge_MixedCollisionIDs(t *testing.T) {
	t.Parallel()

	got := Merge(
		[]Record{{ID: "abc1234", Spec: testSpecExplorer, State: StateWorking}},
		[]ShellSnapshot{{ID: "abc1234", Command: "sleep 300", State: "running"}},
	)
	require.Len(t, got, 2)
	require.Equal(t, KindAgent, got[0].Kind)
	require.Equal(t, TypedID(KindAgent, "abc1234"), got[0].ID)
	require.Equal(t, KindShell, got[1].Kind)
	require.Equal(t, TypedID(KindShell, "abc1234"), got[1].ID)
}

func TestFormatView_Empty(t *testing.T) {
	t.Parallel()

	require.Equal(t, "No tasks.", FormatView(nil))
}

func TestCancelView_TypedShellCallsKill(t *testing.T) {
	t.Parallel()

	src := &killRecorder{}
	err := CancelView(t.Context(), TypedID(KindShell, "abc1234"), New(), src)
	require.NoError(t, err)
	require.Equal(t, []string{"abc1234"}, src.killed)
}

type killRecorder struct {
	killed []string
	listed []ShellSnapshot
}

func (k *killRecorder) List(context.Context) []ShellSnapshot { return k.listed }

func (k *killRecorder) Kill(_ context.Context, id string) error {
	k.killed = append(k.killed, id)

	return nil
}

func TestCancelView_TypedAgentUsesRegistry(t *testing.T) {
	t.Parallel()

	h := &orderHandle{}
	reg := New()
	reg.Register("task-1", testSpecExplorer, StateWorking, h)

	err := CancelView(t.Context(), TypedID(KindAgent, "task-1"), reg, &killRecorder{})
	require.NoError(t, err)
	require.Equal(t, []string{"cancel", "sigterm"}, h.calls)
}

func TestCancelView_UntypedCollisionRejected(t *testing.T) {
	t.Parallel()

	reg := New()
	reg.Register("abc1234", testSpecExplorer, StateWorking, nil)

	src := &killRecorder{listed: []ShellSnapshot{{ID: "abc1234", Command: "sleep 300", State: "running"}}}

	err := CancelView(t.Context(), "abc1234", reg, src)
	require.ErrorIs(t, err, ErrAmbiguous)
	require.Empty(t, src.killed)
}

func TestCancelView_UntypedShellOnlyKills(t *testing.T) {
	t.Parallel()

	src := &killRecorder{listed: []ShellSnapshot{{ID: "abc1234", Command: "sleep 300", State: "running"}}}
	err := CancelView(t.Context(), "abc1234", New(), src)
	require.NoError(t, err)
	require.Equal(t, []string{"abc1234"}, src.killed)
}
