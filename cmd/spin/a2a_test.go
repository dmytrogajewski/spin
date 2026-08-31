package main

// Journey: specs/journeys/JOURNEY-017-local-a2a-server-process.md
// Journey: specs/journeys/JOURNEY-025-parent-shutdown-cancels-children.md.

import (
	"bytes"
	"context"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/dmytrogajewski/spin/internal/agent/child"
	"github.com/dmytrogajewski/spin/internal/agent/subagent"
	"github.com/dmytrogajewski/spin/internal/protocol/a2a"
)

const a2aSendText = "explore-please"

func TestA2ACommand_Registered(t *testing.T) {
	t.Parallel()

	cmd := newRootCmd()
	a2aCmd, _, err := cmd.Find([]string{"a2a"})
	require.NoError(t, err)
	require.Equal(t, "a2a", a2aCmd.Name())
	require.NotNil(t, a2aCmd.Flags().Lookup(flagA2ASpec))
	require.NotNil(t, a2aCmd.Flags().Lookup(flagA2AStdio))
	require.NotNil(t, a2aCmd.Flags().Lookup(flagA2AListen))
}

func TestA2ACommand_HelpDocumentsUnixListen(t *testing.T) {
	t.Parallel()

	cmd := newA2ACmd()
	require.Contains(t, strings.ToLower(cmd.Long+" "+cmd.Flags().Lookup(flagA2AListen).Usage), "unix://")
}

func TestA2ACommand_StdioCardThenSend(t *testing.T) {
	t.Parallel()

	root := newRootCmd()
	stdinR, stdinW := io.Pipe()
	stdoutR, stdoutW := io.Pipe()

	t.Cleanup(func() {
		_ = stdinR.Close()
		_ = stdinW.Close()
		_ = stdoutR.Close()
		_ = stdoutW.Close()
	})

	root.SetArgs([]string{"a2a", "--spec", subagent.NameExplorer, "--stdio"})
	root.SetIn(stdinR)
	root.SetOut(stdoutW)
	root.SetErr(&bytes.Buffer{})

	errCh := make(chan error, 1)

	go func() {
		errCh <- root.Execute()
	}()

	client, err := a2a.NewClient(stdoutR, stdinW)
	require.NoError(t, err)
	require.Equal(t, subagent.NameExplorer, client.Card().Name)

	task, err := client.SendMessage(context.Background(), a2a.Message{
		MessageID: "cli-1",
		Role:      a2a.RoleUser,
		Parts:     []a2a.Part{{Text: a2aSendText, MediaType: "text/plain"}},
	})
	require.NoError(t, err)
	require.NotEmpty(t, task.ID)

	_ = stdinW.Close()

	require.NoError(t, <-errCh)
}

func TestA2ACommand_UnknownSpec(t *testing.T) {
	t.Parallel()

	root := newRootCmd()
	root.SetArgs([]string{"a2a", "--spec", "nope", "--stdio"})
	root.SetIn(bytes.NewReader(nil))
	root.SetOut(&bytes.Buffer{})
	root.SetErr(&bytes.Buffer{})

	err := root.Execute()
	require.Error(t, err)
	require.Contains(t, err.Error(), "nope")
}

func TestInstallChildPid_WritesAndRemoves(t *testing.T) {
	base := t.TempDir()
	t.Setenv("XDG_RUNTIME_DIR", base)

	cleanup := installChildPid()
	path := child.PidPath(child.RuntimeDir(), os.Getpid())
	_, err := os.Stat(path)
	require.NoError(t, err)
	cleanup()

	_, err = os.Stat(path)
	require.ErrorIs(t, err, os.ErrNotExist)
}
