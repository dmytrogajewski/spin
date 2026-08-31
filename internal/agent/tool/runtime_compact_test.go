package tool

// Journey: specs/journeys/JOURNEY-011-apply-compact-to-shell-exec.md.

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/dmytrogajewski/spin/internal/contexteng/compact"
	"github.com/dmytrogajewski/spin/internal/events"
	"github.com/dmytrogajewski/spin/internal/message"
	"github.com/dmytrogajewski/spin/internal/tools"
)

var errLookPathMiss = errors.New("rtk missing")

func newCompactRuntime(t *testing.T, probe *captureTool, lookPath func(string) (string, error), enabled bool) *Runtime {
	t.Helper()

	registry := tools.NewRegistry()
	require.NoError(t, registry.Register(probe))

	return NewRuntime(RuntimeConfig{
		Registry:       registry,
		Emitter:        events.NewEventEmitter(100),
		WorkDir:        t.TempDir(),
		CompactEnabled: enabled,
		CompactBackend: compact.BackendRTK,
		LookPath:       lookPath,
	})
}

func TestRuntime_RewritesShellArgvWhenRTKPresent(t *testing.T) {
	t.Parallel()

	probe := &captureTool{name: "shell_command"}
	rt := newCompactRuntime(t, probe, func(string) (string, error) {
		return "/fake/rtk", nil
	}, true)

	_, err := rt.Execute(context.Background(), &message.ToolCall{
		ID:   "call_r11",
		Type: "function",
		Function: message.ToolCallFunction{
			Name:      "shell_command",
			Arguments: `{"command":"git status"}`,
		},
	})
	require.NoError(t, err)

	cmd, cmdErr := probe.got.GetString("command")
	require.NoError(t, cmdErr)
	require.Equal(t, "rtk git status", cmd)
}

func TestRuntime_NoRewriteWhenRTKMissing(t *testing.T) {
	t.Parallel()

	probe := &captureTool{name: "shell_command"}
	rt := newCompactRuntime(t, probe, func(string) (string, error) {
		return "", errLookPathMiss
	}, true)

	_, err := rt.Execute(context.Background(), &message.ToolCall{
		ID:   "call_miss",
		Type: "function",
		Function: message.ToolCallFunction{
			Name:      "shell_command",
			Arguments: `{"command":"git status"}`,
		},
	})
	require.NoError(t, err)

	cmd, cmdErr := probe.got.GetString("command")
	require.NoError(t, cmdErr)
	require.Equal(t, "git status", cmd)
}

func TestRuntime_NoRewriteWhenCompactDisabled(t *testing.T) {
	t.Parallel()

	probe := &captureTool{name: "shell_command"}
	rt := newCompactRuntime(t, probe, func(string) (string, error) {
		return "/fake/rtk", nil
	}, false)

	_, err := rt.Execute(context.Background(), &message.ToolCall{
		ID:   "call_off",
		Type: "function",
		Function: message.ToolCallFunction{
			Name:      "shell_command",
			Arguments: `{"command":"git status"}`,
		},
	})
	require.NoError(t, err)

	cmd, cmdErr := probe.got.GetString("command")
	require.NoError(t, cmdErr)
	require.Equal(t, "git status", cmd)
}

func TestRuntime_NoRewriteWhenEnvOff(t *testing.T) {
	t.Setenv(compact.EnvName, compact.EnvOff)

	probe := &captureTool{name: "shell_command"}
	rt := newCompactRuntime(t, probe, func(string) (string, error) {
		return "/fake/rtk", nil
	}, true)

	_, err := rt.Execute(context.Background(), &message.ToolCall{
		ID:   "call_env",
		Type: "function",
		Function: message.ToolCallFunction{
			Name:      "shell_command",
			Arguments: `{"command":"git status"}`,
		},
	})
	require.NoError(t, err)

	cmd, cmdErr := probe.got.GetString("command")
	require.NoError(t, cmdErr)
	require.Equal(t, "git status", cmd)
}

const gitStatusPorcelain = "M  staged.go\n M unstaged.go\n?? new.go\n"

func TestRuntime_GitStatusObservationIsCompact(t *testing.T) {
	t.Parallel()

	executor := &recordingExecutor{
		stdout: gitStatusPorcelain,
	}
	shell := tools.NewShellCommandTool(nil, nil, executor)
	registry := tools.NewRegistry()
	require.NoError(t, registry.Register(shell))

	rt := NewRuntime(RuntimeConfig{
		Registry: registry,
		Emitter:  events.NewEventEmitter(100),
		WorkDir:  t.TempDir(),
	})

	result, err := rt.Execute(context.Background(), &message.ToolCall{
		ID:   "call_obs",
		Type: "function",
		Function: message.ToolCallFunction{
			Name:      "shell_command",
			Arguments: `{"operation":"execute","command":"git status"}`,
		},
	})
	require.NoError(t, err)
	require.NotNil(t, result)

	want := string(compact.Default().Apply("git status", []byte(gitStatusPorcelain), nil, 0).Stdout)
	require.Equal(t, want, result.Output)
}

type recordingExecutor struct {
	stdout string
	raw    string
}

func (r *recordingExecutor) Execute(_ context.Context, cmd tools.CommandInfo, _ any) (tools.ExecutionResult, error) {
	r.raw = cmd.GetRaw()

	return &execResult{stdout: r.stdout}, nil
}

type execResult struct {
	stdout string
	code   int
}

func (e *execResult) GetStdout() string           { return e.stdout }
func (e *execResult) GetStderr() string           { return "" }
func (e *execResult) GetExitCode() int            { return e.code }
func (e *execResult) GetMetadata() map[string]any { return nil }

func TestRuntime_RTKFallbackUsesGoPipeline(t *testing.T) {
	t.Parallel()

	executor := &recordingExecutor{stdout: gitStatusPorcelain}
	shell := tools.NewShellCommandTool(nil, nil, executor)
	registry := tools.NewRegistry()
	require.NoError(t, registry.Register(shell))

	rt := NewRuntime(RuntimeConfig{
		Registry:       registry,
		Emitter:        events.NewEventEmitter(100),
		WorkDir:        t.TempDir(),
		CompactEnabled: true,
		CompactBackend: compact.BackendRTK,
		LookPath:       func(string) (string, error) { return "", errLookPathMiss },
	})

	result, err := rt.Execute(context.Background(), &message.ToolCall{
		ID:   "call_fb",
		Type: "function",
		Function: message.ToolCallFunction{
			Name:      "shell_command",
			Arguments: `{"operation":"execute","command":"git status"}`,
		},
	})
	require.NoError(t, err)
	require.Equal(t, "git status", executor.raw)

	want := string(compact.Default().Apply("git status", []byte(gitStatusPorcelain), nil, 0).Stdout)
	require.Equal(t, want, result.Output)
}

func TestRuntime_UpdatedInputThenRTKPrefix(t *testing.T) {
	t.Parallel()

	runner := writePreToolUse(t, `echo '{"updated_input":"{\"command\":\"git status\"}"}'
exit 0`)
	probe := &captureTool{name: "shell_command"}
	registry := tools.NewRegistry()
	require.NoError(t, registry.Register(probe))

	rt := NewRuntime(RuntimeConfig{
		Registry:       registry,
		Emitter:        events.NewEventEmitter(100),
		WorkDir:        t.TempDir(),
		HookRunner:     runner,
		CompactEnabled: true,
		CompactBackend: compact.BackendRTK,
		LookPath:       func(string) (string, error) { return "/fake/rtk", nil },
	})

	_, err := rt.Execute(context.Background(), &message.ToolCall{
		ID:   "call_hook",
		Type: "function",
		Function: message.ToolCallFunction{
			Name:      "shell_command",
			Arguments: `{"command":"rm -rf /"}`,
		},
	})
	require.NoError(t, err)

	cmd, cmdErr := probe.got.GetString("command")
	require.NoError(t, cmdErr)
	require.Equal(t, "rtk git status", cmd)
}
