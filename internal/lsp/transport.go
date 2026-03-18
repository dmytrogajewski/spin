package lsp

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
)

const (
	// headerContentLength is the Content-Length header prefix.
	headerContentLength = "Content-Length: "
	// headerTerminator is the double CRLF that terminates headers.
	headerTerminator = "\r\n\r\n"
	// singleCRLF separates header lines.
	singleCRLF = "\r\n"
	// pendingChannelBuffer is the buffer size for pending response channels.
	pendingChannelBuffer = 1
)

// Transport defines the interface for JSON-RPC 2.0 communication.
type Transport interface {
	// Send sends a request and waits for the response.
	Send(ctx context.Context, method string, params any) (json.RawMessage, error)

	// Notify sends a notification (no response expected).
	Notify(ctx context.Context, method string, params any) error

	// Close shuts down the transport.
	Close() error
}

// jsonrpcRequest is a JSON-RPC 2.0 request.
type jsonrpcRequest struct {
	JSONRPC string `json:"jsonrpc"`
	ID      *int64 `json:"id,omitempty"`
	Method  string `json:"method"`
	Params  any    `json:"params,omitempty"`
}

// jsonrpcResponse is a JSON-RPC 2.0 response.
type jsonrpcResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      *int64          `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *jsonrpcError   `json:"error,omitempty"`
}

// jsonrpcError is a JSON-RPC 2.0 error object.
type jsonrpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// Error returns the error message.
func (e *jsonrpcError) Error() string {
	return fmt.Sprintf("jsonrpc error %d: %s", e.Code, e.Message)
}

// StdioTransport implements [Transport] over stdin/stdout using Content-Length framing.
type StdioTransport struct {
	nextID    atomic.Int64
	writer    io.WriteCloser
	reader    io.ReadCloser
	writeMu   sync.Mutex
	pending   sync.Map
	closed    atomic.Bool
	done      chan struct{}
	closeOnce sync.Once
	readErr   atomic.Pointer[error]
}

// NewStdioTransport creates a transport over the given reader/writer pair.
// It starts a background goroutine to read responses.
func NewStdioTransport(reader io.ReadCloser, writer io.WriteCloser) *StdioTransport {
	transport := &StdioTransport{
		writer: writer,
		reader: reader,
		done:   make(chan struct{}),
	}

	go transport.readLoop()

	return transport
}

// Send sends a JSON-RPC request and waits for the corresponding response.
func (st *StdioTransport) Send(ctx context.Context, method string, params any) (json.RawMessage, error) {
	if st.closed.Load() {
		return nil, ErrTransportClosed
	}

	reqID := st.nextID.Add(1)
	responseCh := make(chan jsonrpcResponse, pendingChannelBuffer)

	st.pending.Store(reqID, responseCh)

	defer st.pending.Delete(reqID)

	req := jsonrpcRequest{
		JSONRPC: "2.0",
		ID:      &reqID,
		Method:  method,
		Params:  params,
	}

	if writeErr := st.writeMessage(req); writeErr != nil {
		return nil, fmt.Errorf("write request: %w", writeErr)
	}

	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("request %s: %w", method, ctx.Err())
	case resp := <-responseCh:
		if resp.Error != nil {
			return nil, resp.Error
		}

		return resp.Result, nil
	case <-st.done:
		return nil, st.doneError()
	}
}

// Notify sends a JSON-RPC notification (no ID, no response expected).
func (st *StdioTransport) Notify(ctx context.Context, method string, params any) error {
	if st.closed.Load() {
		return ErrTransportClosed
	}

	select {
	case <-ctx.Done():
		return fmt.Errorf("notify %s: %w", method, ctx.Err())
	default:
	}

	req := jsonrpcRequest{
		JSONRPC: "2.0",
		Method:  method,
		Params:  params,
	}

	if writeErr := st.writeMessage(req); writeErr != nil {
		return fmt.Errorf("write notification: %w", writeErr)
	}

	return nil
}

// Close shuts down the transport and releases resources.
func (st *StdioTransport) Close() error {
	if st.closed.Swap(true) {
		return nil
	}

	st.closeDone()

	var errs []error

	if closeErr := st.writer.Close(); closeErr != nil {
		errs = append(errs, fmt.Errorf("close writer: %w", closeErr))
	}

	if closeErr := st.reader.Close(); closeErr != nil {
		errs = append(errs, fmt.Errorf("close reader: %w", closeErr))
	}

	return errors.Join(errs...)
}

// writeMessage serializes and writes a JSON-RPC message with Content-Length framing.
func (st *StdioTransport) writeMessage(msg any) error {
	body, marshalErr := json.Marshal(msg)
	if marshalErr != nil {
		return fmt.Errorf("marshal message: %w", marshalErr)
	}

	header := headerContentLength + strconv.Itoa(len(body)) + headerTerminator

	st.writeMu.Lock()
	defer st.writeMu.Unlock()

	if _, writeErr := io.WriteString(st.writer, header); writeErr != nil {
		return fmt.Errorf("write header: %w", writeErr)
	}

	if _, writeErr := st.writer.Write(body); writeErr != nil {
		return fmt.Errorf("write body: %w", writeErr)
	}

	return nil
}

// readLoop reads JSON-RPC responses and dispatches them to pending request channels.
// On unexpected read error (not from [Close]), it stores the error and closes done
// to immediately unblock all pending callers.
func (st *StdioTransport) readLoop() {
	scanner := bufio.NewReader(st.reader)

	for !st.closed.Load() {
		contentLen, headerErr := readContentLength(scanner)
		if headerErr != nil {
			st.handleReadError(headerErr)

			return
		}

		body := make([]byte, contentLen)
		if _, readErr := io.ReadFull(scanner, body); readErr != nil {
			st.handleReadError(readErr)

			return
		}

		var resp jsonrpcResponse
		if unmarshalErr := json.Unmarshal(body, &resp); unmarshalErr != nil {
			continue
		}

		if resp.ID == nil {
			continue
		}

		if ch, ok := st.pending.Load(*resp.ID); ok {
			respCh, _ := ch.(chan jsonrpcResponse)
			respCh <- resp
		}
	}
}

// handleReadError processes a read error from readLoop.
// If the transport was already closed (clean shutdown), the error is ignored.
// Otherwise it is stored and done is closed to unblock pending callers.
func (st *StdioTransport) handleReadError(err error) {
	if st.closed.Load() {
		return
	}

	st.setReadErr(err)
}

// setReadErr stores the read error and closes done to unblock pending callers.
func (st *StdioTransport) setReadErr(err error) {
	st.readErr.Store(&err)
	st.closeDone()
}

// closeDone closes the done channel exactly once, safe for concurrent callers.
func (st *StdioTransport) closeDone() {
	st.closeOnce.Do(func() {
		close(st.done)
	})
}

// doneError returns the appropriate error when done is closed.
// If readLoop stored an error, it is returned wrapped.
// Otherwise returns [ErrTransportClosed] (clean shutdown).
func (st *StdioTransport) doneError() error {
	if errPtr := st.readErr.Load(); errPtr != nil {
		return fmt.Errorf("transport read: %w", *errPtr)
	}

	return ErrTransportClosed
}

// readContentLength reads the Content-Length header from the stream.
func readContentLength(reader *bufio.Reader) (int, error) {
	var contentLen int

	for {
		line, readErr := reader.ReadString('\n')
		if readErr != nil {
			return 0, fmt.Errorf("read header line: %w", readErr)
		}

		trimmed := strings.TrimRight(line, singleCRLF)
		if trimmed == "" {
			break
		}

		if val, found := strings.CutPrefix(trimmed, headerContentLength); found {
			parsed, parseErr := strconv.Atoi(val)
			if parseErr != nil {
				return 0, fmt.Errorf("parse content length: %w", parseErr)
			}

			contentLen = parsed
		}
	}

	return contentLen, nil
}
