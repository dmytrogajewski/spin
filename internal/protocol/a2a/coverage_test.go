package a2a

// Journey: specs/journeys/JOURNEY-016-a2a-types-and-local-json-rpc-codec.md.

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"net"
	"testing"

	"github.com/stretchr/testify/require"
)

type stubHandler struct {
	card   AgentCard
	result json.RawMessage
	rpcErr *RPCError
}

func (stub stubHandler) Card() AgentCard {
	return stub.card
}

func (stub stubHandler) Handle(
	_ context.Context,
	_ string,
	_ json.RawMessage,
) (json.RawMessage, *RPCError) {
	return stub.result, stub.rpcErr
}

func TestNewClient_UnexpectedCard(t *testing.T) {
	t.Parallel()

	reader, writer := io.Pipe()

	t.Cleanup(func() {
		_ = reader.Close()
		_ = writer.Close()
	})

	go func() {
		_ = writeEnvelope(writer, envelope{
			JSONRPC: jsonRPCVersion,
			Method:  "nope",
			Params:  json.RawMessage("{}"),
		})
		_ = writer.Close()
	}()

	_, err := NewClient(reader, io.Discard)
	require.ErrorIs(t, err, ErrUnexpectedCard)
}

func TestNewClient_ReadError(t *testing.T) {
	t.Parallel()

	reader, writer := io.Pipe()
	require.NoError(t, writer.Close())

	_, err := NewClient(reader, io.Discard)
	require.Error(t, err)
}

func TestNewClient_BadCardJSON(t *testing.T) {
	t.Parallel()

	reader, writer := io.Pipe()

	t.Cleanup(func() {
		_ = reader.Close()
		_ = writer.Close()
	})

	go func() {
		_ = writeEnvelope(writer, envelope{
			JSONRPC: jsonRPCVersion,
			Method:  MethodAgentCard,
			Params:  json.RawMessage("[]"),
		})
		_ = writer.Close()
	}()

	_, err := NewClient(reader, io.Discard)
	require.Error(t, err)
}

func TestCall_CanceledContext(t *testing.T) {
	t.Parallel()

	client := startPipeClient(t, fixtureCard())
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := client.Call(ctx, MethodTasksList, struct{}{}, nil)
	require.Error(t, err)
}

func TestHandle_CanceledContext(t *testing.T) {
	t.Parallel()

	handler := NewMemoryHandler(fixtureCard())
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, rpcErr := handler.Handle(ctx, MethodTasksList, nil)
	require.Equal(t, CodeInternalError, rpcErr.Code)
}

func TestHandle_UnknownMethod(t *testing.T) {
	t.Parallel()

	handler := NewMemoryHandler(fixtureCard())
	_, rpcErr := handler.Handle(context.Background(), "nope", nil)
	require.Equal(t, CodeMethodNotFound, rpcErr.Code)
}

func TestGetCard_Unsupported(t *testing.T) {
	t.Parallel()

	card := fixtureCard()
	card.Capabilities.ExtendedAgentCard = false
	client := startPipeClient(t, card)

	_, err := client.GetCard(context.Background())
	requireRPCErrorCode(t, err, CodeUnsupportedOperation)
}

func TestCancel_MissingAndInvalid(t *testing.T) {
	t.Parallel()

	client := startPipeClient(t, fixtureCard())
	ctx := context.Background()

	_, err := client.CancelTask(ctx, "missing")
	requireRPCErrorCode(t, err, CodeTaskNotFound)

	err = client.Call(ctx, MethodTasksCancel, struct{}{}, nil)
	requireRPCErrorCode(t, err, CodeInvalidParams)

	err = client.Call(ctx, MethodTasksGet, struct{}{}, nil)
	requireRPCErrorCode(t, err, CodeInvalidParams)
}

func TestFirstText_Empty(t *testing.T) {
	t.Parallel()

	require.Empty(t, firstText(Message{Parts: []Part{{URL: "x"}}}))
}

func TestSendMessage_InvalidAgentResponse(t *testing.T) {
	t.Parallel()

	client := startStubClient(t, stubHandler{
		card:   fixtureCard(),
		result: json.RawMessage("{}"),
	})

	_, err := client.SendMessage(context.Background(), userMessage("msg-empty", "x"))
	requireRPCErrorCode(t, err, CodeInvalidAgentResponse)
}

func TestStreamMessage_InvalidAgentResponse(t *testing.T) {
	t.Parallel()

	card := fixtureCard()
	card.Capabilities.Streaming = true
	client := startStubClient(t, stubHandler{
		card:   card,
		result: json.RawMessage("{}"),
	})

	_, err := client.StreamMessage(context.Background(), userMessage("msg-empty", "x"))
	requireRPCErrorCode(t, err, CodeInvalidAgentResponse)
}

func TestServe_NotificationThenSend(t *testing.T) {
	t.Parallel()

	conn := startRawPipe(t, fixtureCard())
	discardCard(t, conn)

	_, err := conn.Write([]byte(`{"jsonrpc":"2.0","method":"agent/card","params":{}}` + "\n"))
	require.NoError(t, err)

	_, err = conn.Write([]byte("\n"))
	require.NoError(t, err)

	payload, err := json.Marshal(envelope{
		JSONRPC: jsonRPCVersion,
		ID:      json.RawMessage("7"),
		Method:  MethodTasksList,
		Params:  json.RawMessage("{}"),
	})
	require.NoError(t, err)

	_, err = conn.Write(append(payload, '\n'))
	require.NoError(t, err)

	line, err := bufio.NewReader(conn).ReadBytes('\n')
	require.NoError(t, err)

	var env envelope
	require.NoError(t, json.Unmarshal(line, &env))
	require.Nil(t, env.Error)
	require.NotEmpty(t, env.Result)
}

func startStubClient(t *testing.T, handler Handler) *Client {
	t.Helper()

	clientConn, serverConn := net.Pipe()

	t.Cleanup(func() {
		_ = clientConn.Close()
		_ = serverConn.Close()
	})

	ctx, cancel := context.WithCancel(context.Background())

	t.Cleanup(cancel)

	go func() {
		_ = Serve(ctx, serverConn, serverConn, handler)
	}()

	client, err := NewClient(clientConn, clientConn)
	require.NoError(t, err)

	return client
}
