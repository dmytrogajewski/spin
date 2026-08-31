package a2a

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
)

const (
	jsonRPCVersion = "2.0"
	ndjsonNewline  = '\n'
)

// jsonNull is the JSON-RPC id for parse errors.
var jsonNull = json.RawMessage("null")

// ErrUnexpectedCard is returned when the first framed message is not a card announce.
var ErrUnexpectedCard = errors.New("a2a: first message is not an agent card")

type envelope struct {
	Error   *RPCError       `json:"error,omitempty"`
	ID      json.RawMessage `json:"id,omitempty"`
	JSONRPC string          `json:"jsonrpc"`
	Method  string          `json:"method,omitempty"`
	Params  json.RawMessage `json:"params,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
}

func writeEnvelope(writer io.Writer, env envelope) error {
	body, marshalErr := json.Marshal(env)
	if marshalErr != nil {
		return fmt.Errorf("marshal envelope: %w", marshalErr)
	}

	if _, writeErr := writer.Write(append(body, ndjsonNewline)); writeErr != nil {
		return fmt.Errorf("write envelope: %w", writeErr)
	}

	return nil
}

func readEnvelope(reader *bufio.Reader) (envelope, error) {
	line, readErr := reader.ReadBytes(ndjsonNewline)
	if readErr != nil && !errors.Is(readErr, io.EOF) {
		return envelope{}, fmt.Errorf("read envelope: %w", readErr)
	}

	line = bytes.TrimSpace(line)
	if len(line) == 0 {
		if readErr != nil {
			return envelope{}, fmt.Errorf("read envelope: %w", readErr)
		}

		return envelope{}, errEmptyLine
	}

	var env envelope
	if unmarshalErr := json.Unmarshal(line, &env); unmarshalErr != nil {
		return envelope{}, errJSONParse
	}

	return env, nil
}

var (
	errJSONParse = errors.New("invalid json payload")
	errEmptyLine = errors.New("empty ndjson line")
)

func writeError(writer io.Writer, id json.RawMessage, code int, message string) error {
	if len(id) == 0 {
		id = jsonNull
	}

	return writeEnvelope(writer, envelope{
		JSONRPC: jsonRPCVersion,
		ID:      id,
		Error:   NewRPCError(code, message),
	})
}

func writeResult(writer io.Writer, id, result json.RawMessage) error {
	return writeEnvelope(writer, envelope{
		JSONRPC: jsonRPCVersion,
		ID:      id,
		Result:  result,
	})
}

func isA2AMethod(method string) bool {
	switch method {
	case MethodMessageSend, MethodMessageStream, MethodTasksGet,
		MethodTasksList, MethodTasksCancel, MethodAgentGetCard:
		return true
	default:
		return false
	}
}
