package tool

// Journey: specs/journeys/JOURNEY-008-wire-every-defined-lifecycle-hook-event.md.

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/dmytrogajewski/spin/internal/events"
	"github.com/dmytrogajewski/spin/internal/message"
	"github.com/dmytrogajewski/spin/internal/safety/hooks"
	"github.com/dmytrogajewski/spin/internal/tools"
)

var errToolBoom = errors.New("tool boom")

type failTool struct{}

func (f *failTool) Name() string        { return "fail_test" }
func (f *failTool) Description() string { return "fails" }
func (f *failTool) Schema() tools.ToolSchema {
	return tools.ToolSchema{
		Type: "function",
		Function: tools.FunctionSchema{
			Name: "fail_test",
			Parameters: tools.ParameterSchema{
				Type:       "object",
				Properties: map[string]tools.PropertyDefinition{},
			},
		},
	}
}

func (f *failTool) Execute(_ context.Context, _ tools.ToolParameters) (tools.ToolResult, error) {
	return tools.ToolResult{}, errToolBoom
}

func writeLifecycleScript(t *testing.T, dir, name, marker string) {
	t.Helper()

	body := "#!/bin/sh\ntouch " + marker + "\n"
	require.NoError(t, os.WriteFile(filepath.Join(dir, name), []byte(body), 0o600))
}

func waitMarker(t *testing.T, path string) {
	t.Helper()

	require.Eventually(t, func() bool {
		_, err := os.Stat(path)

		return err == nil
	}, 2*time.Second, 20*time.Millisecond, "hook marker %s", path)
}

func TestRuntime_ToolErrorFiresPostToolUseFailure(t *testing.T) {
	t.Parallel()

	hookDir := t.TempDir()
	failMarker := filepath.Join(hookDir, "failure")
	okMarker := filepath.Join(hookDir, "success")
	writeLifecycleScript(t, hookDir, "post-tool-use-failure", failMarker)
	writeLifecycleScript(t, hookDir, "post-tool-use", okMarker)

	registry := tools.NewRegistry()
	require.NoError(t, registry.Register(&failTool{}))

	rt := NewRuntime(RuntimeConfig{
		Registry:   registry,
		Emitter:    events.NewEventEmitter(8),
		WorkDir:    t.TempDir(),
		HookRunner: hooks.NewRunner(hooks.Config{ProjectDir: hookDir}),
	})

	result, err := rt.Execute(context.Background(), &message.ToolCall{
		ID:   "call_fail",
		Type: "function",
		Function: message.ToolCallFunction{
			Name:      "fail_test",
			Arguments: "{}",
		},
	})

	require.NoError(t, err)
	require.False(t, result.Success)
	waitMarker(t, failMarker)

	_, statErr := os.Stat(okMarker)
	require.Error(t, statErr, "POST_TOOL_USE must not fire on tool error")
}

func TestRuntime_ToolSuccessStillFiresPostToolUse(t *testing.T) {
	t.Parallel()

	hookDir := t.TempDir()
	okMarker := filepath.Join(hookDir, "success")
	failMarker := filepath.Join(hookDir, "failure")
	writeLifecycleScript(t, hookDir, "post-tool-use", okMarker)
	writeLifecycleScript(t, hookDir, "post-tool-use-failure", failMarker)

	rt, _ := newRuntimeWithHooks(t, hooks.NewRunner(hooks.Config{ProjectDir: hookDir}))
	result, err := rt.Execute(context.Background(), makeToolCall())

	require.NoError(t, err)
	require.True(t, result.Success)
	waitMarker(t, okMarker)

	_, statErr := os.Stat(failMarker)
	require.Error(t, statErr, "POST_TOOL_USE_FAILURE must not fire on success")
}
