package child

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"github.com/dmytrogajewski/spin/internal/agent/subagent"
	"github.com/dmytrogajewski/spin/internal/protocol/a2a"
)

// ErrNilSpec indicates NewServer was called with a nil Spec.
var ErrNilSpec = errors.New("child: spec is nil")

// Server serves A2A over a stream from an isolated Spec harness.
type Server struct {
	handler a2a.Handler
	harness *Harness
}

// NewServer builds a child A2A server. Parent history is not accepted.
func NewServer(spec *subagent.Spec) (*Server, error) {
	if spec == nil {
		return nil, ErrNilSpec
	}

	childHarness := NewHarness(spec)

	return &Server{
		handler: &recordingHandler{
			inner:   a2a.NewMemoryHandler(CardFromSpec(spec)),
			harness: childHarness,
		},
		harness: childHarness,
	}, nil
}

// Serve publishes the Agent Card then answers NDJSON-RPC methods.
func (server *Server) Serve(ctx context.Context, reader io.Reader, writer io.Writer) error {
	if err := a2a.Serve(ctx, reader, writer, server.handler); err != nil {
		return fmt.Errorf("child serve: %w", err)
	}

	return nil
}

// Harness returns the isolated child harness.
func (server *Server) Harness() *Harness {
	return server.harness
}

type recordingHandler struct {
	inner   a2a.Handler
	harness *Harness
}

// Card returns the inner handler's Agent Card.
func (handler *recordingHandler) Card() a2a.AgentCard {
	return handler.inner.Card()
}

// Handle records message/send onto the isolated harness, then delegates.
func (handler *recordingHandler) Handle(
	ctx context.Context,
	method string,
	params json.RawMessage,
) (json.RawMessage, *a2a.RPCError) {
	if method == a2a.MethodMessageSend {
		var in a2a.SendMessageParams
		if err := json.Unmarshal(params, &in); err == nil {
			handler.harness.Record(in.Message)
		}
	}

	return handler.inner.Handle(ctx, method, params)
}
