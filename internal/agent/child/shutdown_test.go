package child

// Journey: specs/journeys/JOURNEY-025-parent-shutdown-cancels-children.md.

import (
	"bytes"
	"context"
	"io"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/dmytrogajewski/spin/internal/agent/subagent"
	"github.com/dmytrogajewski/spin/internal/protocol/a2a"
)

const stdinEOFChild = `printf '%s\n' '{"jsonrpc":"2.0","method":"agent/card","params":{"name":"x"}}'; cat >/dev/null`

func TestServe_ReturnsAfterReaderEOF(t *testing.T) {
	t.Parallel()

	spec, err := subagent.Lookup(subagent.NameExplorer)
	require.NoError(t, err)

	server, err := NewServer(spec)
	require.NoError(t, err)

	reader, writer := io.Pipe()
	errCh := make(chan error, 1)

	go func() {
		errCh <- server.Serve(t.Context(), reader, io.Discard)
	}()

	require.NoError(t, writer.Close())

	select {
	case serveErr := <-errCh:
		require.NoError(t, serveErr)
	case <-time.After(2 * time.Second):
		t.Fatal("child serve must return after stdin/reader close")
	}
}

func TestProcess_ExitsAfterStdinClose(t *testing.T) {
	t.Parallel()

	proc, err := Start(testCtx(t), "/bin/sh", []string{"-c", stdinEOFChild}, "")
	require.NoError(t, err)
	require.NotNil(t, proc)
	t.Cleanup(func() { _ = proc.Close() })

	require.NoError(t, proc.stdin.Close())

	done := make(chan error, 1)

	go func() { done <- proc.cmd.Wait() }()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("child must exit after stdin close")
	}
}

func TestListenAndServe_ReturnsAfterClientClose(t *testing.T) {
	t.Parallel()

	spec, err := subagent.Lookup(subagent.NameExplorer)
	require.NoError(t, err)

	server, err := NewServer(spec)
	require.NoError(t, err)

	sock := t.TempDir() + "/a2a.sock"
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	errCh := make(chan error, 1)

	go func() {
		errCh <- server.ListenAndServe(ctx, ListenUnixPrefix+sock)
	}()

	conn := dialUnix(t, sock)
	_, err = a2a.NewClient(conn, conn)
	require.NoError(t, err)
	require.NoError(t, conn.Close())

	select {
	case serveErr := <-errCh:
		require.NoError(t, serveErr)
	case <-time.After(2 * time.Second):
		t.Fatal("child must exit after socket close")
	}
}

func TestServe_WritesCardBeforeEOF(t *testing.T) {
	t.Parallel()

	spec, err := subagent.Lookup(subagent.NameExplorer)
	require.NoError(t, err)

	server, err := NewServer(spec)
	require.NoError(t, err)

	var out bytes.Buffer
	require.NoError(t, server.Serve(t.Context(), bytes.NewReader(nil), &out))
	require.Contains(t, out.String(), a2a.MethodAgentCard)
}
