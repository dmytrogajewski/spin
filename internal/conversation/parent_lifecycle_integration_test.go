package conversation

// Journey: specs/journeys/JOURNEY-008-wire-every-defined-lifecycle-hook-event.md.

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/dmytrogajewski/spin/internal/agent/harness"
	"github.com/dmytrogajewski/spin/internal/agent/scaffold"
	"github.com/dmytrogajewski/spin/internal/agent/subagent"
	"github.com/dmytrogajewski/spin/internal/agent/tool"
	"github.com/dmytrogajewski/spin/internal/contexteng/history"
	"github.com/dmytrogajewski/spin/internal/events"
	"github.com/dmytrogajewski/spin/internal/message"
	"github.com/dmytrogajewski/spin/internal/safety/hooks"
	"github.com/dmytrogajewski/spin/internal/tools"
)

var errParentLifecycleBoom = errors.New("parent lifecycle boom")

var parentSideHookEvents = []hooks.Event{
	hooks.EventSessionStart,
	hooks.EventUserPromptSubmit,
	hooks.EventPreToolUse,
	hooks.EventPostToolUse,
	hooks.EventPostToolUseFailure,
	hooks.EventPreCompact,
	hooks.EventStop,
	hooks.EventSessionEnd,
}

type stubTurnOK struct{}

func (stubTurnOK) Execute(_ context.Context, _ string, hist []message.Message) (string, []message.Message, error) {
	return "done", hist, nil
}

type finishCaller struct{}

func (finishCaller) Call(
	_ context.Context, _ []message.Message, _ []tools.ToolSchema, _ int,
) (content string, toolCalls []message.ToolCall, finishReason string, err error) {
	return "done", nil, "stop", nil
}

type noopDispatch struct{}

func (noopDispatch) Dispatch(
	_ context.Context, msgs []message.Message, _ string, _ []message.ToolCall,
) []message.Message {
	return msgs
}

type rewriteCompact struct{}

func (rewriteCompact) Compact(_ context.Context, msgs []message.Message) ([]message.Message, bool, error) {
	return msgs, true, nil
}

type okLifecycleTool struct{}

func (okLifecycleTool) Name() string        { return "ok_lifecycle" }
func (okLifecycleTool) Description() string { return "ok" }
func (okLifecycleTool) Schema() tools.ToolSchema {
	return tools.ToolSchema{Type: "function", Function: tools.FunctionSchema{Name: "ok_lifecycle"}}
}

func (okLifecycleTool) Execute(_ context.Context, _ tools.ToolParameters) (tools.ToolResult, error) {
	return tools.NewToolResult("ok"), nil
}

type failLifecycleTool struct{}

func (failLifecycleTool) Name() string        { return "fail_lifecycle" }
func (failLifecycleTool) Description() string { return "fail" }
func (failLifecycleTool) Schema() tools.ToolSchema {
	return tools.ToolSchema{Type: "function", Function: tools.FunctionSchema{Name: "fail_lifecycle"}}
}

func (failLifecycleTool) Execute(_ context.Context, _ tools.ToolParameters) (tools.ToolResult, error) {
	return tools.ToolResult{}, errParentLifecycleBoom
}

func writeRecordingScripts(t *testing.T, dir, logPath string) {
	t.Helper()

	require.Len(t, hooks.AllEvents(), 10)

	for _, evt := range hooks.AllEvents() {
		body := fmt.Sprintf("#!/bin/sh\necho %s >> %s\n", evt, logPath)
		require.NoError(t, os.WriteFile(filepath.Join(dir, evt.ScriptName()), []byte(body), 0o600))
	}
}

func waitLogContains(t *testing.T, logPath string, names []hooks.Event) {
	t.Helper()

	require.Eventually(t, func() bool {
		data, err := os.ReadFile(logPath)
		if err != nil {
			return false
		}

		text := string(data)
		for _, name := range names {
			if !strings.Contains(text, string(name)) {
				return false
			}
		}

		return true
	}, 2*time.Second, 20*time.Millisecond, "parent-side hook names")
}

func TestParentLifecycle_RecordingScriptsAssertParentSubset(t *testing.T) {
	t.Parallel()

	hookDir := t.TempDir()
	logPath := filepath.Join(hookDir, "events.log")
	writeRecordingScripts(t, hookDir, logPath)

	runner := hooks.NewRunner(hooks.Config{ProjectDir: hookDir})
	ctx := context.Background()

	(&Builder{workDir: hookDir}).fireSessionStartHook(ctx, runner, "sess-parent")

	conv := &Conversation{
		hookRunner:      runner,
		id:              "sess-parent",
		workDir:         hookDir,
		history:         history.NewHistoryWithDefaults(),
		emitter:         events.NewEventEmitter(8),
		harnessExecutor: stubTurnOK{},
		subagentManager: subagent.NewManager(nil, subagent.DefaultMaxConcurrent),
	}
	require.NoError(t, conv.RunTurn(ctx, "hello"))

	registry := tools.NewRegistry()
	require.NoError(t, registry.Register(okLifecycleTool{}))
	require.NoError(t, registry.Register(failLifecycleTool{}))

	rt := tool.NewRuntime(tool.RuntimeConfig{
		Registry:   registry,
		Emitter:    events.NewEventEmitter(8),
		WorkDir:    hookDir,
		HookRunner: runner,
	})

	_, err := rt.Execute(ctx, &message.ToolCall{
		ID: "ok1", Type: "function",
		Function: message.ToolCallFunction{Name: "ok_lifecycle", Arguments: "{}"},
	})
	require.NoError(t, err)

	_, err = rt.Execute(ctx, &message.ToolCall{
		ID: "fail1", Type: "function",
		Function: message.ToolCallFunction{Name: "fail_lifecycle", Arguments: "{}"},
	})
	require.NoError(t, err)

	exec, err := harness.NewExecutor(
		&scaffold.Spec{SystemPrompt: "test", Config: scaffold.SpecConfig{MaxTurns: 2}},
		finishCaller{}, noopDispatch{}, nil, nil, slog.Default(),
		harness.WithCompactor(rewriteCompact{}),
		harness.WithHookRunner(runner),
	)
	require.NoError(t, err)

	_, err = exec.Execute(ctx, "compact me", nil)
	require.NoError(t, err)

	require.NoError(t, conv.Close(ctx))

	waitLogContains(t, logPath, parentSideHookEvents)
}
