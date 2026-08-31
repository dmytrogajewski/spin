package tool

// Journey: specs/journeys/JOURNEY-007-finish-the-hook-runner-contract.md.

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/dmytrogajewski/spin/internal/events"
	"github.com/dmytrogajewski/spin/internal/message"
	"github.com/dmytrogajewski/spin/internal/safety/hooks"
	"github.com/dmytrogajewski/spin/internal/tools"
)

type captureTool struct {
	name string
	got  tools.ToolParameters
}

func (c *captureTool) Name() string        { return c.name }
func (c *captureTool) Description() string { return "capture" }
func (c *captureTool) Schema() tools.ToolSchema {
	return tools.ToolSchema{
		Type: "function",
		Function: tools.FunctionSchema{
			Name: c.name,
			Parameters: tools.ParameterSchema{
				Type:       "object",
				Properties: map[string]tools.PropertyDefinition{},
			},
		},
	}
}

func (c *captureTool) Execute(_ context.Context, params tools.ToolParameters) (tools.ToolResult, error) {
	c.got = params

	return tools.NewToolResult("ok"), nil
}

func newRuntimeWithCapture(t *testing.T, runner *hooks.Runner, name string) (*Runtime, *captureTool) {
	t.Helper()

	probe := &captureTool{name: name}
	registry := tools.NewRegistry()
	require.NoError(t, registry.Register(probe))

	return NewRuntime(RuntimeConfig{
		Registry:   registry,
		Emitter:    events.NewEventEmitter(100),
		WorkDir:    t.TempDir(),
		HookRunner: runner,
	}), probe
}

func writePreToolUse(t *testing.T, body string) *hooks.Runner {
	t.Helper()

	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "pre-tool-use"), []byte(body+"\n"), 0o600))

	return hooks.NewRunner(hooks.Config{ProjectDir: dir})
}

func TestRuntime_UpdatedInputReplacesStructuredArgs(t *testing.T) {
	t.Parallel()

	runner := writePreToolUse(t, `echo '{"updated_input":"{\"path\":\"safe.txt\"}"}'
exit 0`)
	rt, probe := newRuntimeWithCapture(t, runner, "read_probe")

	_, err := rt.Execute(context.Background(), &message.ToolCall{
		ID:   "call_rewrite",
		Type: "function",
		Function: message.ToolCallFunction{
			Name:      "read_probe",
			Arguments: `{"path":"secret.txt"}`,
		},
	})
	require.NoError(t, err)

	path, pathErr := probe.got.GetString("path")
	require.NoError(t, pathErr)
	require.Equal(t, "safe.txt", path)
}

func TestRuntime_EmptyUpdatedInputKeepsOriginalArgs(t *testing.T) {
	t.Parallel()

	runner := writePreToolUse(t, "exit 0")
	rt, probe := newRuntimeWithCapture(t, runner, "read_probe")

	_, err := rt.Execute(context.Background(), &message.ToolCall{
		ID:   "call_keep",
		Type: "function",
		Function: message.ToolCallFunction{
			Name:      "read_probe",
			Arguments: `{"path":"original.txt"}`,
		},
	})
	require.NoError(t, err)

	path, pathErr := probe.got.GetString("path")
	require.NoError(t, pathErr)
	require.Equal(t, "original.txt", path)
}

func TestRuntime_UpdatedInputReplacesShellArgs(t *testing.T) {
	t.Parallel()

	runner := writePreToolUse(t, `echo '{"updated_input":"{\"command\":\"echo safe\"}"}'
exit 0`)
	rt, probe := newRuntimeWithCapture(t, runner, "shell_probe")

	_, err := rt.Execute(context.Background(), &message.ToolCall{
		ID:   "call_shell",
		Type: "function",
		Function: message.ToolCallFunction{
			Name:      "shell_probe",
			Arguments: `{"command":"rm -rf /"}`,
		},
	})
	require.NoError(t, err)

	cmd, cmdErr := probe.got.GetString("command")
	require.NoError(t, cmdErr)
	require.Equal(t, "echo safe", cmd)
}

func TestRuntime_NonObjectUpdatedInputKeepsOriginalArgs(t *testing.T) {
	t.Parallel()

	runner := writePreToolUse(t, `echo '{"updated_input":"sanitized"}'
exit 0`)
	rt, probe := newRuntimeWithCapture(t, runner, "read_probe")

	_, err := rt.Execute(context.Background(), &message.ToolCall{
		ID:   "call_plain",
		Type: "function",
		Function: message.ToolCallFunction{
			Name:      "read_probe",
			Arguments: `{"path":"keep.txt"}`,
		},
	})
	require.NoError(t, err)

	path, pathErr := probe.got.GetString("path")
	require.NoError(t, pathErr)
	require.Equal(t, "keep.txt", path)
}

func TestRuntime_UpdatedInputJSONObjectReplacesArgs(t *testing.T) {
	t.Parallel()

	runner := writePreToolUse(t, `echo '{"updated_input":{"path":"from-object"}}'
exit 0`)
	rt, probe := newRuntimeWithCapture(t, runner, "read_probe")

	_, err := rt.Execute(context.Background(), &message.ToolCall{
		ID:   "call_object",
		Type: "function",
		Function: message.ToolCallFunction{
			Name:      "read_probe",
			Arguments: `{"path":"before.txt"}`,
		},
	})
	require.NoError(t, err)

	path, pathErr := probe.got.GetString("path")
	require.NoError(t, pathErr)
	require.Equal(t, "from-object", path)
}
