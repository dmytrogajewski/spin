package lsp

import (
	"io"

	"github.com/dmytrogajewski/spin/pkg/protocol/jsonrpc"
)

// Transport is an alias for [jsonrpc.Transport].
type Transport = jsonrpc.Transport

// StdioTransport is an alias for [jsonrpc.StdioTransport].
type StdioTransport = jsonrpc.StdioTransport

// NewStdioTransport creates a JSON-RPC 2.0 transport over stdio.
func NewStdioTransport(reader io.ReadCloser, writer io.WriteCloser) *StdioTransport {
	return jsonrpc.NewStdioTransport(reader, writer)
}
