package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/dmytrogajewski/spin/internal/config"
	"github.com/dmytrogajewski/spin/internal/conversation"
	"github.com/dmytrogajewski/spin/internal/events"
	"github.com/dmytrogajewski/spin/internal/llm"
	"github.com/dmytrogajewski/spin/internal/safety"
	"github.com/dmytrogajewski/spin/internal/session"
	"github.com/dmytrogajewski/spin/internal/ui/ports"
)

// conversationConfig holds mode-specific options for building a conversation.
type conversationConfig struct {
	// approvalHandler controls how tool-call approvals are handled.
	approvalHandler safety.ApprovalHandler
	// ui is the UI adapter for the conversation runtime.
	ui ports.UI
	// sessionPrefix is the prefix used when no session storage is available (e.g. "tui", "exec").
	sessionPrefix string
	// eventBufferSize is the capacity of the event emitter channel.
	eventBufferSize int
}

// createConversation creates a conversation using a shared builder pattern.
// Mode-specific behavior is controlled through [conversationConfig].
func createConversation(
	ctx context.Context, provider llm.Provider,
	cfg *config.V2, opts conversationConfig,
) (*conversation.Conversation, error) {
	workDir := cfg.Agent.WorkDir
	logger := slog.Default()
	emitter := events.NewEventEmitter(opts.eventBufferSize)

	storage, err := createSessionStorage(cfg.Agent.SessionDir)
	if err != nil && !errors.Is(err, ErrNoSessionDir) {
		return nil, err
	}

	sessionID := resolveSessionID(storage, workDir, opts.sessionPrefix)

	services, cleanup, err := createServices(ctx, cfg, workDir, logger)
	if err != nil {
		return nil, err
	}

	builtinRuntime, err := createBuiltinRuntime(
		ctx, workDir, emitter, storage, sessionID,
		opts.approvalHandler, services, opts.ui, logger, cfg,
	)
	if err != nil {
		cleanup()

		return nil, fmt.Errorf("create builtin runtime: %w", err)
	}

	convBuilder := conversation.NewBuilder(cfg, workDir, builtinRuntime, emitter, provider)

	if services.Git != nil {
		convBuilder = convBuilder.WithGit(services.Git)
	}

	if services.Shell != nil {
		convBuilder = convBuilder.WithShell(services.Shell)
	}

	if services.MCP != nil {
		convBuilder = convBuilder.WithMCP(services.MCP)

		if toolSelector := createToolSelector(
			ctx, services.MCP, nil, emitter, cfg, slog.Default(),
		); toolSelector != nil {
			convBuilder = convBuilder.WithToolSelector(toolSelector)
		}
	}

	conv, convErr := convBuilder.Build(ctx)
	if convErr != nil {
		cleanup()

		return nil, fmt.Errorf("build conversation: %w", convErr)
	}

	return conv, nil
}

// resolveSessionID determines the session ID based on storage availability.
func resolveSessionID(storage session.Storage, workDir, prefix string) string {
	if storage != nil {
		sess := session.NewSession(workDir)

		return sess.ID
	}

	return fmt.Sprintf("%s-%d", prefix, time.Now().UnixNano())
}
