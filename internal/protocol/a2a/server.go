package a2a

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
)

// Handler serves A2A methods over the local JSON-RPC binding.
type Handler interface {
	Card() AgentCard
	Handle(ctx context.Context, method string, params json.RawMessage) (json.RawMessage, *RPCError)
}

// Serve reads NDJSON JSON-RPC requests from reader and writes responses to writer.
// The Agent Card is published as the first framed notification.
func Serve(ctx context.Context, reader io.Reader, writer io.Writer, handler Handler) error {
	if err := writeEnvelope(writer, envelope{
		JSONRPC: jsonRPCVersion,
		Method:  MethodAgentCard,
		Params:  mustMarshal(handler.Card()),
	}); err != nil {
		return fmt.Errorf("announce card: %w", err)
	}

	bufReader := bufio.NewReader(reader)

	for {
		if err := ctx.Err(); err != nil {
			return fmt.Errorf("serve: %w", err)
		}

		if err := serveOne(ctx, bufReader, writer, handler); err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}

			return err
		}
	}
}

func serveOne(ctx context.Context, reader *bufio.Reader, writer io.Writer, handler Handler) error {
	env, readErr := readEnvelope(reader)
	if errors.Is(readErr, errEmptyLine) {
		return nil
	}

	if errors.Is(readErr, errJSONParse) {
		return writeError(writer, jsonNull, CodeParseError, msgParseError)
	}

	if readErr != nil {
		return readErr
	}

	return dispatch(ctx, writer, handler, env)
}

func dispatch(ctx context.Context, writer io.Writer, handler Handler, env envelope) error {
	if env.JSONRPC != jsonRPCVersion || env.Method == "" {
		return writeError(writer, env.ID, CodeInvalidRequest, msgInvalidRequest)
	}

	if len(env.ID) == 0 {
		return nil
	}

	if !isA2AMethod(env.Method) {
		return writeError(writer, env.ID, CodeMethodNotFound, msgMethodNotFound)
	}

	result, rpcErr := handler.Handle(ctx, env.Method, env.Params)
	if rpcErr != nil {
		return writeError(writer, env.ID, rpcErr.Code, rpcErr.Message)
	}

	return writeResult(writer, env.ID, result)
}

func mustMarshal(value any) json.RawMessage {
	body, err := json.Marshal(value)
	if err != nil {
		return json.RawMessage("{}")
	}

	return body
}
