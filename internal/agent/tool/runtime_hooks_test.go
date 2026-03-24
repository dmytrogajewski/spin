package tool

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

// echoTool is a minimal tool that records if Execute was called.
type echoTool struct {
	called bool
}

func (e *echoTool) Name() string        { return "echo_test" }
func (e *echoTool) Description() string { return "test tool" }
func (e *echoTool) Schema() tools.ToolSchema {
	return tools.ToolSchema{
		Type: "function",
		Function: tools.FunctionSchema{
			Name: "echo_test",
			Parameters: tools.ParameterSchema{
				Type:       "object",
				Properties: map[string]tools.PropertyDefinition{},
			},
		},
	}
}

func (e *echoTool) Execute(_ context.Context, _ tools.ToolParameters) (tools.ToolResult, error) {
	e.called = true

	return tools.NewToolResult("ok"), nil
}

func newRuntimeWithHooks(t *testing.T, runner *hooks.Runner) (*Runtime, *echoTool) {
	t.Helper()

	registry := tools.NewRegistry()
	tool := &echoTool{}

	err := registry.Register(tool)
	require.NoError(t, err)

	emitter := events.NewEventEmitter(100)

	rt := NewRuntime(RuntimeConfig{
		Registry:   registry,
		Emitter:    emitter,
		WorkDir:    "/tmp",
		HookRunner: runner,
	})

	return rt, tool
}

func makeToolCall() *message.ToolCall {
	return &message.ToolCall{
		ID:   "call_hook_test",
		Type: "function",
		Function: message.ToolCallFunction{
			Name:      "echo_test",
			Arguments: "{}",
		},
	}
}

func TestRuntime_NilHookRunnerNoOp(t *testing.T) {
	t.Parallel()

	// Mutant killed: "nil hook runner panics".
	rt, tool := newRuntimeWithHooks(t, nil)

	result, err := rt.Execute(context.Background(), makeToolCall())

	require.NoError(t, err)
	require.True(t, result.Success)
	require.True(t, tool.called, "tool should execute when hook runner is nil")
}

func TestRuntime_CallsPreToolHookBeforeExecution(t *testing.T) {
	t.Parallel()

	// Mutant killed: "PRE_TOOL_USE hook not called".
	// Create a hook dir with a pre-tool-use script that creates a marker file.
	hookDir := t.TempDir()
	markerFile := filepath.Join(hookDir, "pre-tool-marker")

	scriptPath := filepath.Join(hookDir, "pre-tool-use")
	script := "#!/bin/sh\ntouch " + markerFile + "\n"

	err := os.WriteFile(scriptPath, []byte(script), 0o600)
	require.NoError(t, err)
	require.NoError(t, os.Chmod(scriptPath, 0o700))

	runner := hooks.NewRunner(hooks.Config{
		ProjectDir: hookDir,
	})

	rt, tool := newRuntimeWithHooks(t, runner)

	result, err := rt.Execute(context.Background(), makeToolCall())

	require.NoError(t, err)
	require.True(t, result.Success)
	require.True(t, tool.called)

	// Verify the hook actually ran by checking the marker file.
	_, statErr := os.Stat(markerFile)
	require.NoError(t, statErr, "pre-tool-use hook should have created marker file")
}

func TestRuntime_BlockedHookPreventsExecution(t *testing.T) {
	t.Parallel()

	// Mutant killed: "blocked hook still allows tool execution".
	hookDir := t.TempDir()

	scriptPath := filepath.Join(hookDir, "pre-tool-use")
	// Exit code 2 = block, with JSON reason on stdout.
	script := `#!/bin/sh
echo '{"reason":"policy violation"}'
exit 2
`

	err := os.WriteFile(scriptPath, []byte(script), 0o600)
	require.NoError(t, err)
	require.NoError(t, os.Chmod(scriptPath, 0o700))

	runner := hooks.NewRunner(hooks.Config{
		ProjectDir: hookDir,
	})

	rt, tool := newRuntimeWithHooks(t, runner)

	result, execErr := rt.Execute(context.Background(), makeToolCall())

	require.NoError(t, execErr)
	require.False(t, result.Success, "result should indicate failure when hook blocks")
	require.Contains(t, result.Error, "blocked by hook")
	require.False(t, tool.called, "tool should NOT execute when hook blocks")
}
