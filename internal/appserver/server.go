package appserver

import (
	"context"
	"io"

	"github.com/dmytrogajewski/spin/internal/core"
	"github.com/dmytrogajewski/spin/internal/llm"
	"github.com/dmytrogajewski/spin/internal/protocol/jsonrpc"
)

// Server is the app-server implementation
type Server struct {
	workspacePath string
	processor     *Processor
	jsonrpcServer *jsonrpc.Server
}

// Config contains configuration for the app-server
type Config struct {
	WorkspacePath string
	Version       string
	Provider      llm.Provider      // LLM provider
	Executor      *core.Executor    // Command executor (optional)
	Validator     *core.Validator   // Command validator (optional)
	Environment   *core.Environment // Environment context (optional)
}

// New creates a new app-server
func New(config Config) (*Server, error) {
	processor, err := NewProcessor(ProcessorConfig{
		WorkspacePath: config.WorkspacePath,
		Version:       config.Version,
		Provider:      config.Provider,
		Executor:      config.Executor,
		Validator:     config.Validator,
		Environment:   config.Environment,
	})
	if err != nil {
		return nil, err
	}

	handler := &Handler{processor: processor}
	jsonrpcServer := jsonrpc.NewServer(handler)

	return &Server{
		workspacePath: config.WorkspacePath,
		processor:     processor,
		jsonrpcServer: jsonrpcServer,
	}, nil
}

// Serve starts the server and processes requests
func (s *Server) Serve(ctx context.Context, r io.Reader, w io.Writer) error {
	// Set output writer for notifications
	s.processor.SetOutput(w)

	// Start JSON-RPC server
	return s.jsonrpcServer.Serve(ctx, r, w)
}
