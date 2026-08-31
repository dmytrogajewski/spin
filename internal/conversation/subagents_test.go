package conversation

// Journey: specs/journeys/JOURNEY-018-spawn-process-children-from-the-parent.md
// Journey: specs/journeys/JOURNEY-019-subagent-hooks-and-no-silent-drop.md.

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/dmytrogajewski/spin/internal/agent/child"
	"github.com/dmytrogajewski/spin/internal/agent/subagent"
	"github.com/dmytrogajewski/spin/internal/config"
	"github.com/dmytrogajewski/spin/internal/events"
	"github.com/dmytrogajewski/spin/internal/safety/hooks"
)

const extraSpecName = "research"

func TestCreateSubagentManager_RegistersConfigSpecs(t *testing.T) {
	t.Parallel()

	cfg := testConfig()
	cfg.Subagents = map[string]config.SubagentConfigV2{
		extraSpecName: {Model: "tiny", MaxIterations: 2},
	}

	workDir := t.TempDir()
	rt, emitter, provider := createTestRuntime(t, workDir)
	b := NewBuilder(cfg, workDir, rt, emitter, provider)

	mgr := b.createSubagentManager(nil)
	spec := mgr.Spec(extraSpecName)
	require.NotNil(t, spec)
	require.Equal(t, "tiny", spec.ModelOverride)
	require.Equal(t, 2, spec.MaxIterations)
}

func TestCreateSubagentManager_OverlayKeepsBuiltinResidue(t *testing.T) {
	t.Parallel()

	cfg := testConfig()
	cfg.Subagents = map[string]config.SubagentConfigV2{
		subagent.NameExplorer: {Model: "tiny", MaxIterations: 2},
	}

	workDir := t.TempDir()
	rt, emitter, provider := createTestRuntime(t, workDir)
	b := NewBuilder(cfg, workDir, rt, emitter, provider)

	mgr := b.createSubagentManager(nil)
	spec := mgr.Spec(subagent.NameExplorer)
	require.NotNil(t, spec)
	require.Equal(t, "tiny", spec.ModelOverride)
	require.Equal(t, 2, spec.MaxIterations)
	require.NotEmpty(t, spec.SystemPrompt)
	require.True(t, spec.HasTool("read_file"))
}

func TestBuilder_SpawnProcess_NotStub(t *testing.T) {
	bin, ok := child.FindRepoBinary()
	require.True(t, ok, "build/bin/spin not found")
	t.Setenv("SPIN_BIN", bin)

	cfg := testConfig()
	workDir := t.TempDir()
	rt, emitter, provider := createTestRuntime(t, workDir)
	conv, err := NewBuilder(cfg, workDir, rt, emitter, provider).Build(context.Background())
	require.NoError(t, err)
	t.Cleanup(func() { _ = conv.Close(context.Background()) })

	_, evts, subErr := emitter.Subscribe()
	require.NoError(t, subErr)

	summary, spawnErr := conv.GetSubagentManager().Spawn(context.Background(), subagent.NameExplorer, spawnParentQuery)
	require.NoError(t, spawnErr)
	require.NotErrorIs(t, spawnErr, ErrSubagentSpawnNotSupported)
	require.Equal(t, spawnParentQuery, summary)

	got := drainConvEvents(evts)
	require.True(t, hasConvEvent(got, events.EventSubagentSpawn))
	require.True(t, hasConvEvent(got, events.EventSubagentComplete))
}

func TestBuilder_SpawnVetoNoProcess(t *testing.T) {
	bin, ok := child.FindRepoBinary()
	require.True(t, ok, "build/bin/spin not found")
	t.Setenv("SPIN_BIN", bin)

	workDir := t.TempDir()
	hookDir := filepath.Join(workDir, ".spin", "hooks")
	require.NoError(t, os.MkdirAll(hookDir, 0o750))
	require.NoError(t, os.WriteFile(
		filepath.Join(hookDir, hooks.EventSubagentStart.ScriptName()),
		[]byte("echo veto\nexit 2\n"),
		0o600,
	))

	cfg := testConfig()
	rt, emitter, provider := createTestRuntime(t, workDir)
	conv, err := NewBuilder(cfg, workDir, rt, emitter, provider).Build(context.Background())
	require.NoError(t, err)
	t.Cleanup(func() { _ = conv.Close(context.Background()) })

	_, spawnErr := conv.GetSubagentManager().Spawn(context.Background(), subagent.NameExplorer, spawnParentQuery)
	require.ErrorIs(t, spawnErr, child.ErrStartBlocked)
}

const spawnParentQuery = "parent-spawn-query"

func drainConvEvents(evts <-chan events.Event) []events.Event {
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

func hasConvEvent(got []events.Event, want events.EventType) bool {
	for _, ev := range got {
		if ev.Type == want {
			return true
		}
	}

	return false
}
