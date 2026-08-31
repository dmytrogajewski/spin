package a2a

// Journey: specs/journeys/JOURNEY-016-a2a-types-and-local-json-rpc-codec.md.

import (
	"bufio"
	"context"
	"encoding/json"
	"net"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestClientServer_SendAndGetOverPipe(t *testing.T) {
	t.Parallel()

	client := startPipeClient(t, fixtureCard())
	ctx := context.Background()

	sent, err := client.SendMessage(ctx, userMessage("msg-1", "hello"))
	require.NoError(t, err)
	require.NotEmpty(t, sent.ID)
	require.Equal(t, TaskStateCompleted, sent.Status.State)
	require.Equal(t, "hello", sent.Artifacts[0].Parts[0].Text)

	got, err := client.GetTask(ctx, sent.ID)
	require.NoError(t, err)
	require.Equal(t, sent.ID, got.ID)
}

func TestClient_CardAnnounceAndFetch(t *testing.T) {
	t.Parallel()

	client := startPipeClient(t, fixtureCard())
	require.Equal(t, "fixture-agent", client.Card().Name)
	require.Equal(t, "1.0.0", client.Card().Version)

	card, err := client.GetCard(context.Background())
	require.NoError(t, err)
	require.Equal(t, client.Card().Name, card.Name)
	require.Equal(t, client.Card().Version, card.Version)
}

func TestClient_ListTasks(t *testing.T) {
	t.Parallel()

	client := startPipeClient(t, fixtureCard())
	ctx := context.Background()

	sent, err := client.SendMessage(ctx, userMessage("msg-list", "listed"))
	require.NoError(t, err)

	listed, err := client.ListTasks(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, listed.TotalSize)
	require.Equal(t, sent.ID, listed.Tasks[0].ID)
	require.Empty(t, listed.Tasks[0].Artifacts)
}

func TestServe_ParseError(t *testing.T) {
	t.Parallel()

	conn := startRawPipe(t, fixtureCard())
	reader := bufio.NewReader(conn)
	_, err := reader.ReadBytes('\n')
	require.NoError(t, err)

	_, err = conn.Write([]byte("not-json\n"))
	require.NoError(t, err)

	line, err := reader.ReadBytes('\n')
	require.NoError(t, err)

	var env envelope
	require.NoError(t, json.Unmarshal(line, &env))
	require.Equal(t, CodeParseError, env.Error.Code)
}

func TestServe_InvalidRequest(t *testing.T) {
	t.Parallel()

	conn := startRawPipe(t, fixtureCard())
	discardCard(t, conn)

	_, err := conn.Write([]byte(`{"jsonrpc":"1.0","id":1,"method":"message/send"}` + "\n"))
	require.NoError(t, err)

	requireRPCCode(t, conn, CodeInvalidRequest)
}

func TestClient_MethodNotFound(t *testing.T) {
	t.Parallel()

	client := startPipeClient(t, fixtureCard())
	err := client.Call(context.Background(), "no/such", struct{}{}, nil)
	requireRPCErrorCode(t, err, CodeMethodNotFound)
}

func TestClient_TaskNotFound(t *testing.T) {
	t.Parallel()

	client := startPipeClient(t, fixtureCard())
	_, err := client.GetTask(context.Background(), "missing")
	requireRPCErrorCode(t, err, CodeTaskNotFound)
}

func TestClient_CancelTerminal(t *testing.T) {
	t.Parallel()

	client := startPipeClient(t, fixtureCard())
	ctx := context.Background()

	sent, err := client.SendMessage(ctx, userMessage("msg-done", "done"))
	require.NoError(t, err)

	_, err = client.CancelTask(ctx, sent.ID)
	requireRPCErrorCode(t, err, CodeTaskNotCancelable)
}

func TestClient_CancelWorking(t *testing.T) {
	t.Parallel()

	client := startPipeClient(t, fixtureCard())
	ctx := context.Background()

	sent, err := client.SendMessageImmediate(ctx, userMessage("msg-work", "work"))
	require.NoError(t, err)
	require.Equal(t, TaskStateWorking, sent.Status.State)

	canceled, err := client.CancelTask(ctx, sent.ID)
	require.NoError(t, err)
	require.Equal(t, TaskStateCanceled, canceled.Status.State)
}

func TestClient_StreamUnsupported(t *testing.T) {
	t.Parallel()

	client := startPipeClient(t, fixtureCard())
	_, err := client.StreamMessage(context.Background(), userMessage("msg-stream", "x"))
	requireRPCErrorCode(t, err, CodeUnsupportedOperation)
}

func TestClient_InvalidParams(t *testing.T) {
	t.Parallel()

	client := startPipeClient(t, fixtureCard())
	_, err := client.SendMessage(context.Background(), Message{MessageID: "empty"})
	requireRPCErrorCode(t, err, CodeInvalidParams)
}

func TestClient_StreamWhenEnabled(t *testing.T) {
	t.Parallel()

	card := fixtureCard()
	card.Capabilities.Streaming = true
	client := startPipeClient(t, card)

	sent, err := client.StreamMessage(context.Background(), userMessage("msg-ok", "streamed"))
	require.NoError(t, err)
	require.NotEmpty(t, sent.ID)
}

func TestRPCError_Error(t *testing.T) {
	t.Parallel()

	require.Empty(t, (*RPCError)(nil).Error())
	require.Contains(t, NewRPCError(CodeParseError, msgParseError).Error(), "32700")
}

func startPipeClient(t *testing.T, card AgentCard) *Client {
	t.Helper()

	clientConn, serverConn := net.Pipe()

	t.Cleanup(func() {
		_ = clientConn.Close()
		_ = serverConn.Close()
	})

	ctx, cancel := context.WithCancel(context.Background())

	t.Cleanup(cancel)

	go func() {
		_ = Serve(ctx, serverConn, serverConn, NewMemoryHandler(card))
	}()

	client, err := NewClient(clientConn, clientConn)
	require.NoError(t, err)

	return client
}

func startRawPipe(t *testing.T, card AgentCard) net.Conn {
	t.Helper()

	clientConn, serverConn := net.Pipe()

	t.Cleanup(func() {
		_ = clientConn.Close()
		_ = serverConn.Close()
	})

	ctx, cancel := context.WithCancel(context.Background())

	t.Cleanup(cancel)

	go func() {
		_ = Serve(ctx, serverConn, serverConn, NewMemoryHandler(card))
	}()

	return clientConn
}

func discardCard(t *testing.T, conn net.Conn) {
	t.Helper()

	reader := bufio.NewReader(conn)
	_, err := reader.ReadBytes('\n')
	require.NoError(t, err)
}

func requireRPCCode(t *testing.T, conn net.Conn, want int) {
	t.Helper()

	line, err := bufio.NewReader(conn).ReadBytes('\n')
	require.NoError(t, err)

	var env envelope
	require.NoError(t, json.Unmarshal(line, &env))
	require.NotNil(t, env.Error)
	require.Equal(t, want, env.Error.Code)
}

func requireRPCErrorCode(t *testing.T, err error, want int) {
	t.Helper()

	var rpcErr *RPCError
	require.ErrorAs(t, err, &rpcErr)
	require.Equal(t, want, rpcErr.Code)
}

func userMessage(id, text string) Message {
	return Message{
		MessageID: id,
		Role:      RoleUser,
		Parts:     []Part{{Text: text, MediaType: mediaTextPlain}},
	}
}
