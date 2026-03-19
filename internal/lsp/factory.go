package lsp

import (
	"context"
	"fmt"
	"os/exec"
)

// DefaultServerFactory creates a language server by spawning the configured
// command via StdioTransport and performing the LSP initialization handshake.
func DefaultServerFactory(ctx context.Context, lang LanguageConfig, rootURI string) (*Server, error) {
	if lang.ServerCommand == "" {
		return nil, fmt.Errorf("%w: no server command for language %q", ErrServerNotFound, lang.ID)
	}

	// Check that the server binary is available.
	if _, lookupErr := exec.LookPath(lang.ServerCommand); lookupErr != nil {
		return nil, fmt.Errorf("%w: %s: %s", ErrServerNotFound, lang.ServerCommand, lookupErr)
	}

	cmd := exec.CommandContext(ctx, lang.ServerCommand, lang.ServerArgs...)

	stdin, stdinErr := cmd.StdinPipe()
	if stdinErr != nil {
		return nil, fmt.Errorf("stdin pipe for %s: %w", lang.ServerCommand, stdinErr)
	}

	stdout, stdoutErr := cmd.StdoutPipe()
	if stdoutErr != nil {
		return nil, fmt.Errorf("stdout pipe for %s: %w", lang.ServerCommand, stdoutErr)
	}

	if startErr := cmd.Start(); startErr != nil {
		return nil, fmt.Errorf("start %s: %w", lang.ServerCommand, startErr)
	}

	transport := NewStdioTransport(stdout, stdin)
	server := NewServer(transport, lang)

	if initErr := server.Initialize(ctx, rootURI); initErr != nil {
		_ = server.Close(ctx)

		return nil, fmt.Errorf("initialize language server %s: %w", lang.ServerCommand, initErr)
	}

	return server, nil
}
