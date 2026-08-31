package child

// Journey: specs/journeys/JOURNEY-017-local-a2a-server-process.md.

import (
	"context"
	"net"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/dmytrogajewski/spin/internal/agent/subagent"
	"github.com/dmytrogajewski/spin/internal/protocol/a2a"
)

const (
	unixDialTimeout = 2 * time.Second
	unixDialPause   = 5 * time.Millisecond
)

func TestParseListen_Unix(t *testing.T) {
	t.Parallel()

	network, path, err := ParseListen("unix:///tmp/spin-a2a.sock")
	require.NoError(t, err)
	require.Equal(t, networkUnix, network)
	require.Equal(t, "/tmp/spin-a2a.sock", path)
}

func TestParseListen_RejectsNonUnix(t *testing.T) {
	t.Parallel()

	_, _, err := ParseListen("tcp://127.0.0.1:9")
	require.ErrorIs(t, err, ErrUnsupportedListen)
}

func TestListenAndServe_UnixCard(t *testing.T) {
	t.Parallel()

	spec, err := subagent.Lookup(subagent.NameExplorer)
	require.NoError(t, err)

	server, err := NewServer(spec)
	require.NoError(t, err)

	sock := filepath.Join(t.TempDir(), "a2a.sock")
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	errCh := make(chan error, 1)

	go func() {
		errCh <- server.ListenAndServe(ctx, ListenUnixPrefix+sock)
	}()

	conn := dialUnix(t, sock)
	t.Cleanup(func() { _ = conn.Close() })

	client, err := a2a.NewClient(conn, conn)
	require.NoError(t, err)
	require.Equal(t, subagent.NameExplorer, client.Card().Name)
	cancel()
}

func dialUnix(t *testing.T, path string) net.Conn {
	t.Helper()

	deadline := time.Now().Add(unixDialTimeout)
	dialer := net.Dialer{}

	for time.Now().Before(deadline) {
		conn, err := dialer.DialContext(t.Context(), networkUnix, path)
		if err == nil {
			return conn
		}

		time.Sleep(unixDialPause)
	}

	t.Fatalf("dial unix %s: timeout", path)

	return nil
}
