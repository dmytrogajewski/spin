package child

// Journey: specs/journeys/JOURNEY-017-local-a2a-server-process.md.

import (
	"context"
	"net"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/dmytrogajewski/spin/internal/agent/subagent"
	"github.com/dmytrogajewski/spin/internal/protocol/a2a"
)

func TestServer_ServeCardThenMessageSend(t *testing.T) {
	t.Parallel()

	spec, err := subagent.Lookup(subagent.NameExplorer)
	require.NoError(t, err)

	client, server := startChildClient(t, spec)
	require.Equal(t, subagent.NameExplorer, client.Card().Name)

	task, sendErr := client.SendMessage(context.Background(), a2a.Message{
		MessageID: "msg-1",
		Role:      a2a.RoleUser,
		Parts:     []a2a.Part{{Text: "explore", MediaType: mediaTextPlain}},
	})
	require.NoError(t, sendErr)
	require.NotEmpty(t, task.ID)
	require.Equal(t, a2a.TaskStateCompleted, task.Status.State)
	require.Contains(t, fmtHistory(server.Harness()), "explore")
	require.NotContains(t, fmtHistory(server.Harness()), parentHistorySentinel)
}

func TestServer_ChildHistoryIsOwnConversation(t *testing.T) {
	t.Parallel()

	spec, err := subagent.Lookup(subagent.NameExplorer)
	require.NoError(t, err)

	server, err := NewServer(spec)
	require.NoError(t, err)
	require.Empty(t, server.Harness().History())
	require.NotContains(t, fmtHistory(server.Harness()), parentHistorySentinel)
}

func TestNewServer_NilSpec(t *testing.T) {
	t.Parallel()

	_, err := NewServer(nil)
	require.ErrorIs(t, err, ErrNilSpec)
}

func TestServe_FirstLineIsJSONRPC(t *testing.T) {
	t.Parallel()

	spec, err := subagent.Lookup(subagent.NameExplorer)
	require.NoError(t, err)

	client, _ := startChildClient(t, spec)
	require.NotEmpty(t, client.Card().Name)
	require.Equal(t, a2a.ProtocolBindingNDJSON, client.Card().SupportedInterfaces[0].ProtocolBinding)
}

func fmtHistory(childHarness *Harness) string {
	var raw strings.Builder

	for _, msg := range childHarness.History() {
		for _, part := range msg.Parts {
			raw.WriteString(part.Text)
		}
	}

	return raw.String()
}

func startChildClient(t *testing.T, spec *subagent.Spec) (*a2a.Client, *Server) {
	t.Helper()

	clientConn, serverConn := net.Pipe()

	t.Cleanup(func() {
		_ = clientConn.Close()
		_ = serverConn.Close()
	})

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	server, err := NewServer(spec)
	require.NoError(t, err)

	go func() {
		_ = server.Serve(ctx, serverConn, serverConn)
	}()

	client, err := a2a.NewClient(clientConn, clientConn)
	require.NoError(t, err)

	return client, server
}
