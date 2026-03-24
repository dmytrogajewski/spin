package lsp_test

// Journey: specs/journeys/JOURNEY-R8.1.md.

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/dmytrogajewski/spin/internal/lsp"
)

const (
	testMethod     = "textDocument/definition"
	testTimeout    = 2 * time.Second
	testResultJSON = `{"uri":"file:///test.go","range":{"start":{"line":1,"character":0}}}`
)

// mockPipe creates an in-memory pipe pair for testing transport.
type mockPipe struct {
	reader *io.PipeReader
	writer *io.PipeWriter
}

func newMockPipe() *mockPipe {
	reader, writer := io.Pipe()

	return &mockPipe{reader: reader, writer: writer}
}

// mockJSONRPCServer simulates a language server that reads requests and sends responses.
func mockJSONRPCServer(clientToServer, serverToClient *mockPipe) {
	go func() {
		buf := make([]byte, 0, 4096)
		readBuf := make([]byte, 1024)

		for {
			bytesRead, readErr := clientToServer.reader.Read(readBuf)
			if readErr != nil {
				return
			}

			buf = append(buf, readBuf[:bytesRead]...)

			// Try to extract a complete message.
			reqID := extractRequestID(buf)
			if reqID < 0 {
				continue
			}

			// Reset buffer for next message.
			buf = buf[:0]

			resp := buildResponse(reqID, json.RawMessage(testResultJSON))
			writeJSONRPCMessage(serverToClient.writer, resp)
		}
	}()
}

func extractRequestID(data []byte) int64 {
	// Simple extraction: find "id": in the body.
	var req struct {
		ID *int64 `json:"id"`
	}

	// Find the start of JSON body (after \r\n\r\n).
	bodyStart := findBodyStart(data)
	if bodyStart < 0 {
		return -1
	}

	if unmarshalErr := json.Unmarshal(data[bodyStart:], &req); unmarshalErr != nil {
		return -1
	}

	if req.ID == nil {
		return -1
	}

	return *req.ID
}

func findBodyStart(data []byte) int {
	separator := []byte("\r\n\r\n")

	for i := range len(data) - len(separator) + 1 {
		if bytes.Equal(data[i:i+len(separator)], separator) {
			return i + len(separator)
		}
	}

	return -1
}

func buildResponse(reqID int64, result json.RawMessage) []byte {
	resp := struct {
		JSONRPC string          `json:"jsonrpc"`
		ID      int64           `json:"id"`
		Result  json.RawMessage `json:"result"`
	}{
		JSONRPC: "2.0",
		ID:      reqID,
		Result:  result,
	}

	data, marshalErr := json.Marshal(resp)
	if marshalErr != nil {
		return nil
	}

	return data
}

func writeJSONRPCMessage(writer io.Writer, body []byte) {
	header := "Content-Length: " + strconv.Itoa(len(body)) + "\r\n\r\n"

	_, _ = io.WriteString(writer, header)
	_, _ = writer.Write(body)
}

func TestStdioTransport_SendAndReceive(t *testing.T) {
	t.Parallel()

	clientToServer := newMockPipe()
	serverToClient := newMockPipe()

	mockJSONRPCServer(clientToServer, serverToClient)

	transport := lsp.NewStdioTransport(serverToClient.reader, clientToServer.writer)

	defer func() { _ = transport.Close() }()

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	result, sendErr := transport.Send(ctx, testMethod, nil)
	require.NoError(t, sendErr)
	require.NotNil(t, result)

	var loc lsp.Location
	require.NoError(t, json.Unmarshal(result, &loc))
	require.Equal(t, "file:///test.go", loc.URI)
}

func TestStdioTransport_Notify(t *testing.T) {
	t.Parallel()

	clientToServer := newMockPipe()
	serverToClient := newMockPipe()

	// Drain the pipe so writes don't block.
	go func() {
		buf := make([]byte, 4096)

		for {
			if _, readErr := clientToServer.reader.Read(buf); readErr != nil {
				return
			}
		}
	}()

	transport := lsp.NewStdioTransport(serverToClient.reader, clientToServer.writer)

	defer func() { _ = transport.Close() }()

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	notifyErr := transport.Notify(ctx, "initialized", struct{}{})
	require.NoError(t, notifyErr)
}

func TestStdioTransport_SendAfterClose(t *testing.T) {
	t.Parallel()

	clientToServer := newMockPipe()
	serverToClient := newMockPipe()

	transport := lsp.NewStdioTransport(serverToClient.reader, clientToServer.writer)
	require.NoError(t, transport.Close())

	ctx := context.Background()

	_, sendErr := transport.Send(ctx, testMethod, nil)
	require.ErrorIs(t, sendErr, lsp.ErrTransportClosed)
}

func TestStdioTransport_NotifyAfterClose(t *testing.T) {
	t.Parallel()

	clientToServer := newMockPipe()
	serverToClient := newMockPipe()

	transport := lsp.NewStdioTransport(serverToClient.reader, clientToServer.writer)
	require.NoError(t, transport.Close())

	ctx := context.Background()
	notifyErr := transport.Notify(ctx, "exit", nil)
	require.ErrorIs(t, notifyErr, lsp.ErrTransportClosed)
}

func TestStdioTransport_ContextCancelled(t *testing.T) {
	t.Parallel()

	clientToServer := newMockPipe()
	serverToClient := newMockPipe()

	// Drain the pipe so writes don't block.
	go func() {
		buf := make([]byte, 4096)

		for {
			if _, readErr := clientToServer.reader.Read(buf); readErr != nil {
				return
			}
		}
	}()

	// No mock server — response will never arrive.
	transport := lsp.NewStdioTransport(serverToClient.reader, clientToServer.writer)

	defer func() { _ = transport.Close() }()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, sendErr := transport.Send(ctx, testMethod, nil)
	require.Error(t, sendErr)
}

func TestStdioTransport_ConcurrentSends(t *testing.T) {
	t.Parallel()

	clientToServer := newMockPipe()
	serverToClient := newMockPipe()

	mockJSONRPCServer(clientToServer, serverToClient)

	transport := lsp.NewStdioTransport(serverToClient.reader, clientToServer.writer)

	defer func() { _ = transport.Close() }()

	const goroutineCount = 5

	var waitGroup sync.WaitGroup

	errs := make([]error, goroutineCount)

	for idx := range goroutineCount {
		waitGroup.Add(1)

		go func(index int) {
			defer waitGroup.Done()

			ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
			defer cancel()

			_, sendErr := transport.Send(ctx, testMethod, nil)
			errs[index] = sendErr
		}(idx)
	}

	waitGroup.Wait()

	for idx, sendErr := range errs {
		require.NoError(t, sendErr, "goroutine %d failed", idx)
	}
}

func TestStdioTransport_CloseIdempotent(t *testing.T) {
	t.Parallel()

	clientToServer := newMockPipe()
	serverToClient := newMockPipe()

	transport := lsp.NewStdioTransport(serverToClient.reader, clientToServer.writer)
	require.NoError(t, transport.Close())

	// Second close should be a no-op.
	require.NoError(t, transport.Close())
}

func TestStdioTransport_ServerCrash_UnblocksPendingSend(t *testing.T) {
	t.Parallel()

	clientToServer := newMockPipe()
	serverToClient := newMockPipe()

	// Drain client writes so they don't block.
	go func() {
		buf := make([]byte, 4096)

		for {
			if _, readErr := clientToServer.reader.Read(buf); readErr != nil {
				return
			}
		}
	}()

	transport := lsp.NewStdioTransport(serverToClient.reader, clientToServer.writer)

	defer func() { _ = transport.Close() }()

	// Start a Send that will block waiting for response.
	sendDone := make(chan error, 1)

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
		defer cancel()

		_, sendErr := transport.Send(ctx, testMethod, nil)
		sendDone <- sendErr
	}()

	// Give Send time to register the pending request.
	time.Sleep(50 * time.Millisecond)

	// Simulate server crash by closing the writer end of the server->client pipe.
	serverToClient.writer.Close()

	// Send should unblock quickly with an error (not wait for ctx timeout).
	select {
	case sendErr := <-sendDone:
		require.Error(t, sendErr)
		// Should NOT be a context error — it should be a transport read error.
		require.NotErrorIs(t, sendErr, context.DeadlineExceeded)
	case <-time.After(testTimeout):
		t.Fatal("Send did not unblock after server crash")
	}
}

func TestStdioTransport_CleanClose_ReturnsTransportClosed(t *testing.T) {
	t.Parallel()

	clientToServer := newMockPipe()
	serverToClient := newMockPipe()

	// Drain client writes so they don't block.
	go func() {
		buf := make([]byte, 4096)

		for {
			if _, readErr := clientToServer.reader.Read(buf); readErr != nil {
				return
			}
		}
	}()

	transport := lsp.NewStdioTransport(serverToClient.reader, clientToServer.writer)

	// Start a Send that will block waiting for response.
	sendDone := make(chan error, 1)

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
		defer cancel()

		_, sendErr := transport.Send(ctx, testMethod, nil)
		sendDone <- sendErr
	}()

	// Give Send time to register.
	time.Sleep(50 * time.Millisecond)

	// Clean close.
	require.NoError(t, transport.Close())

	select {
	case sendErr := <-sendDone:
		require.ErrorIs(t, sendErr, lsp.ErrTransportClosed)
	case <-time.After(testTimeout):
		t.Fatal("Send did not unblock after Close")
	}
}
