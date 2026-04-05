package hooks

// Journey: specs/journeys/JOURNEY-5.2.md.

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// writeScript creates a hook script file. The runner invokes scripts via
// /bin/sh so the file does not need the executable bit (avoids ETXTBSY).
func writeScript(t *testing.T, dir, name, content string) {
	t.Helper()

	target := filepath.Join(dir, name)

	err := os.WriteFile(target, []byte(content+"\n"), 0o600)
	require.NoError(t, err)
}

func TestAllEvents_Count(t *testing.T) {
	t.Parallel()

	events := AllEvents()

	assert.Len(t, events, eventCount)
}

func TestAllEvents_Unique(t *testing.T) {
	t.Parallel()

	seen := make(map[Event]bool)
	for _, e := range AllEvents() {
		assert.False(t, seen[e], "duplicate event: %s", e)

		seen[e] = true
	}
}

func TestEvent_IsBlocking(t *testing.T) {
	t.Parallel()

	blocking := []Event{
		EventPreToolUse,
		EventUserPromptSubmit,
		EventSubagentStart,
	}
	for _, e := range blocking {
		assert.True(t, e.IsBlocking(), "%s should be blocking", e)
	}

	nonBlocking := []Event{
		EventSessionStart,
		EventPostToolUse,
		EventPostToolUseFailure,
		EventSubagentStop,
		EventPreCompact,
		EventStop,
		EventSessionEnd,
	}
	for _, e := range nonBlocking {
		assert.False(t, e.IsBlocking(), "%s should not be blocking", e)
	}
}

func TestEvent_ScriptName(t *testing.T) {
	t.Parallel()

	cases := []struct {
		event    Event
		expected string
	}{
		{EventSessionStart, "session-start"},
		{EventPreToolUse, "pre-tool-use"},
		{EventPostToolUse, "post-tool-use"},
		{EventPostToolUseFailure, "post-tool-use-failure"},
		{EventSubagentStart, "subagent-start"},
		{EventSubagentStop, "subagent-stop"},
		{EventPreCompact, "pre-compact"},
		{EventStop, "stop"},
		{EventSessionEnd, "session-end"},
		{EventUserPromptSubmit, "user-prompt-submit"},
	}

	for _, tc := range cases {
		assert.Equal(t, tc.expected, tc.event.ScriptName())
	}
}

func TestEvent_AllHaveScriptNames(t *testing.T) {
	t.Parallel()

	for _, e := range AllEvents() {
		assert.NotEmpty(t, e.ScriptName(), "event %s missing script name", e)
	}
}

func TestNewRunner_DefaultTimeout(t *testing.T) {
	t.Parallel()

	runner := NewRunner(Config{})

	assert.Equal(t, defaultTimeout, runner.timeout)
}

func TestNewRunner_CustomTimeout(t *testing.T) {
	t.Parallel()

	customTimeout := 10 * time.Second
	runner := NewRunner(Config{Timeout: customTimeout})

	assert.Equal(t, customTimeout, runner.timeout)
}

func TestRunner_MissingScriptSilentlySkipped(t *testing.T) {
	t.Parallel()

	runner := NewRunner(Config{
		ProjectDir: t.TempDir(),
	})

	result := runner.Execute(
		context.Background(),
		EventPreToolUse,
		EventContext{SessionID: "test"},
	)

	assert.False(t, result.Blocked)
	assert.Empty(t, result.Reason)
}

func TestRunner_BlockingHookExitCode2Blocks(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeScript(t, dir, "pre-tool-use", `echo "dangerous operation blocked"
exit 2`)

	runner := NewRunner(Config{ProjectDir: dir})

	result := runner.Execute(
		context.Background(),
		EventPreToolUse,
		EventContext{SessionID: "test", ToolName: "exec"},
	)

	assert.True(t, result.Blocked)
	assert.Equal(t, "dangerous operation blocked", result.Reason)
}

func TestRunner_BlockingHookExitCode0Allows(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeScript(t, dir, "pre-tool-use", "exit 0")

	runner := NewRunner(Config{ProjectDir: dir})

	result := runner.Execute(
		context.Background(),
		EventPreToolUse,
		EventContext{SessionID: "test"},
	)

	assert.False(t, result.Blocked)
}

func TestRunner_BlockingHookJSONOutput(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeScript(t, dir, "pre-tool-use",
		`echo '{"reason":"policy violation","updated_input":"sanitized input"}'
exit 2`)

	runner := NewRunner(Config{ProjectDir: dir})

	result := runner.Execute(
		context.Background(),
		EventPreToolUse,
		EventContext{SessionID: "test"},
	)

	assert.True(t, result.Blocked)
	assert.Equal(t, "policy violation", result.Reason)
	assert.Equal(t, "sanitized input", result.UpdatedInput)
}

func TestRunner_NonBlockingHookFiresAsync(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	marker := filepath.Join(dir, "marker")
	writeScript(t, dir, "post-tool-use",
		"touch "+marker)

	runner := NewRunner(Config{ProjectDir: dir})

	result := runner.Execute(
		context.Background(),
		EventPostToolUse,
		EventContext{SessionID: "test"},
	)

	assert.False(t, result.Blocked)

	// Wait briefly for the async hook to complete.
	assert.Eventually(t, func() bool {
		_, err := os.Stat(marker)

		return err == nil
	}, 2*time.Second, 50*time.Millisecond)
}

func TestRunner_HookReceivesJSONContext(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	outputFile := filepath.Join(dir, "context.json")
	writeScript(t, dir, "pre-tool-use",
		"cat > "+outputFile+"\nexit 0")

	runner := NewRunner(Config{ProjectDir: dir})

	evtCtx := EventContext{
		SessionID: "sess-123",
		WorkDir:   "/tmp/work",
		ToolName:  "execute_command",
		ToolInput: `{"command":"ls"}`,
	}

	runner.Execute(context.Background(), EventPreToolUse, evtCtx)

	data, err := os.ReadFile(outputFile)
	require.NoError(t, err)

	var received EventContext

	err = json.Unmarshal(data, &received)
	require.NoError(t, err)

	assert.Equal(t, EventPreToolUse, received.Event)
	assert.Equal(t, "sess-123", received.SessionID)
	assert.Equal(t, "/tmp/work", received.WorkDir)
	assert.Equal(t, "execute_command", received.ToolName)
	assert.JSONEq(t, `{"command":"ls"}`, received.ToolInput)
}

func TestRunner_HookTimeoutTreatedAsNonBlocking(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeScript(t, dir, "pre-tool-use", "sleep 30")

	shortTimeout := 200 * time.Millisecond
	runner := NewRunner(Config{
		ProjectDir: dir,
		Timeout:    shortTimeout,
	})

	start := time.Now()

	result := runner.Execute(
		context.Background(),
		EventPreToolUse,
		EventContext{SessionID: "test"},
	)
	elapsed := time.Since(start)

	assert.False(t, result.Blocked, "timed-out hook should not block")
	assert.Less(t, elapsed, 2*time.Second, "should not wait for full sleep")
}

func TestRunner_ExitCode1TreatedAsError(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeScript(t, dir, "pre-tool-use", "exit 1")

	runner := NewRunner(Config{ProjectDir: dir})

	result := runner.Execute(
		context.Background(),
		EventPreToolUse,
		EventContext{SessionID: "test"},
	)

	// Exit code 1 is an error, not a block.
	assert.False(t, result.Blocked)
}

func TestRunner_GlobalAndProjectHooksMerge(t *testing.T) {
	t.Parallel()

	globalDir := t.TempDir()
	projectDir := t.TempDir()
	globalMarker := filepath.Join(globalDir, "global-ran")
	projectMarker := filepath.Join(projectDir, "project-ran")

	writeScript(t, globalDir, "pre-tool-use",
		"touch "+globalMarker+"\nexit 0")
	writeScript(t, projectDir, "pre-tool-use",
		"touch "+projectMarker+"\nexit 0")

	runner := NewRunner(Config{
		GlobalDir:  globalDir,
		ProjectDir: projectDir,
	})

	result := runner.Execute(
		context.Background(),
		EventPreToolUse,
		EventContext{SessionID: "test"},
	)

	assert.False(t, result.Blocked)

	_, err := os.Stat(globalMarker)
	require.NoError(t, err, "global hook should have run")

	_, err = os.Stat(projectMarker)
	require.NoError(t, err, "project hook should have run")
}

func TestRunner_GlobalBlocksBeforeProject(t *testing.T) {
	t.Parallel()

	globalDir := t.TempDir()
	projectDir := t.TempDir()

	projectMarker := filepath.Join(projectDir, "project-ran")

	writeScript(t, globalDir, "pre-tool-use",
		`echo "global policy block"
exit 2`)
	writeScript(t, projectDir, "pre-tool-use",
		"touch "+projectMarker+"\nexit 0")

	runner := NewRunner(Config{
		GlobalDir:  globalDir,
		ProjectDir: projectDir,
	})

	result := runner.Execute(
		context.Background(),
		EventPreToolUse,
		EventContext{SessionID: "test"},
	)

	assert.True(t, result.Blocked)
	assert.Equal(t, "global policy block", result.Reason)

	// Project hook should NOT have run since global blocked first.
	_, err := os.Stat(projectMarker)
	assert.True(t, os.IsNotExist(err), "project hook should not run after global block")
}

func TestRunner_EmptyDirsNoError(t *testing.T) {
	t.Parallel()

	runner := NewRunner(Config{})

	result := runner.Execute(
		context.Background(),
		EventPreToolUse,
		EventContext{SessionID: "test"},
	)

	assert.False(t, result.Blocked)
}

func TestRunner_PlainTextBlockReason(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeScript(t, dir, "user-prompt-submit",
		`echo "not allowed"
exit 2`)

	runner := NewRunner(Config{ProjectDir: dir})

	result := runner.Execute(
		context.Background(),
		EventUserPromptSubmit,
		EventContext{SessionID: "test"},
	)

	assert.True(t, result.Blocked)
	assert.Equal(t, "not allowed", result.Reason)
	assert.Empty(t, result.UpdatedInput)
}
