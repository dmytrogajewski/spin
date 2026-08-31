package main

// Journey: specs/journeys/JOURNEY-tui-resume.md.

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/dmytrogajewski/spin/internal/agent"
	"github.com/dmytrogajewski/spin/internal/conversation"
	"github.com/dmytrogajewski/spin/internal/events"
	"github.com/dmytrogajewski/spin/internal/message"
)

type resumeHarnessStub struct{}

func (resumeHarnessStub) Execute(
	_ context.Context, _ string, hist []message.Message,
) (string, []message.Message, error) {
	return "", hist, nil
}

func newResumeTestConv(t *testing.T, workDir string) *conversation.Conversation {
	t.Helper()

	conv, err := conversation.NewFromAgent(conversation.NewFromAgentConfig{
		Agent:           &agent.Agent{},
		HarnessExecutor: resumeHarnessStub{},
		Emitter:         events.NewEventEmitter(8),
		WorkDir:         workDir,
	})
	require.NoError(t, err)

	return conv
}

func TestTuiCommandContext_ResumeCatalogEmpty(t *testing.T) {
	t.Parallel()

	ctx := &tuiCommandContext{conv: newResumeTestConv(t, t.TempDir())}
	got := ctx.ResumeCatalog(context.Background())
	require.Contains(t, got, "No resumable sessions")
}

func TestTuiCommandContext_GetWorkDir(t *testing.T) {
	t.Parallel()

	workDir := t.TempDir()
	ctx := &tuiCommandContext{conv: newResumeTestConv(t, workDir)}
	require.Equal(t, workDir, ctx.GetWorkDir())
}

func TestTuiCommandContext_AgentTasks(t *testing.T) {
	t.Parallel()

	ctx := &tuiCommandContext{conv: newResumeTestConv(t, t.TempDir())}
	require.NotNil(t, ctx.AgentTasks())
}

func TestTuiCommandContext_ShellTasks(t *testing.T) {
	t.Parallel()

	ctx := &tuiCommandContext{conv: newResumeTestConv(t, t.TempDir())}
	_ = ctx.ShellTasks()
}
