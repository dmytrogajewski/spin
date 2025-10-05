package jsonrpc

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
)

// Handler processes JSON-RPC requests
type Handler interface {
	HandleRequest(ctx context.Context, method string, params json.RawMessage) (interface{}, error)
}

// Server is a JSON-RPC 2.0 server
type Server struct {
	handler Handler
}

// NewServer creates a new JSON-RPC server
func NewServer(handler Handler) *Server {
	return &Server{handler: handler}
}

// Serve processes requests from reader and writes responses to writer
func (s *Server) Serve(ctx context.Context, r io.Reader, w io.Writer) error {
	scanner := bufio.NewScanner(r)
	// Set max buffer size to 10MB
	const maxMessageSize = 10 * 1024 * 1024
	scanner.Buffer(make([]byte, 4096), maxMessageSize)

	encoder := json.NewEncoder(w)

	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		line := scanner.Bytes()

		// Parse request
		var req Request
		if err := json.Unmarshal(line, &req); err != nil {
			// Send parse error response
			resp := Response{
				JSONRPC: "2.0",
				Error:   NewError(ParseError, "Parse error"),
			}
			encoder.Encode(resp)
			continue
		}

		// Handle request
		result, err := s.handler.HandleRequest(ctx, req.Method, req.Params)

		// Send response (if not notification)
		if req.ID != nil {
			resp := Response{
				JSONRPC: "2.0",
				ID:      *req.ID,
			}

			if err != nil {
				// Error response
				if rpcErr, ok := err.(*Error); ok {
					resp.Error = rpcErr
				} else {
					resp.Error = NewError(InternalError, err.Error())
				}
			} else {
				// Success response
				resultJSON, _ := json.Marshal(result)
				resp.Result = resultJSON
			}

			if err := encoder.Encode(resp); err != nil {
				return err
			}
		}
	}

	return scanner.Err()
}

// SendNotification sends a notification to the client
func (s *Server) SendNotification(w io.Writer, method string, params interface{}) error {
	paramsJSON, _ := json.Marshal(params)
	notif := Notification{
		JSONRPC: "2.0",
		Method:  method,
		Params:  paramsJSON,
	}
	return json.NewEncoder(w).Encode(notif)
}
