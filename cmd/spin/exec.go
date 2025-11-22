package main

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"time"

	"github.com/dmytrogajewski/spin/internal/auth"
	"github.com/dmytrogajewski/spin/internal/config"
	"github.com/dmytrogajewski/spin/internal/conversation"
	"github.com/dmytrogajewski/spin/internal/events"
	"github.com/dmytrogajewski/spin/internal/llm"
	"github.com/dmytrogajewski/spin/internal/llm/builder"
	"github.com/dmytrogajewski/spin/internal/security"
	"github.com/dmytrogajewski/spin/internal/session"
	"github.com/dmytrogajewski/spin/internal/tui"
	"github.com/dmytrogajewski/spin/internal/ui/adapters"
	"github.com/dmytrogajewski/spin/internal/ui/ports"
	"github.com/spf13/cobra"
	termx "golang.org/x/term"
)

// newExecCmd creates the exec command for non-interactive execution.
func newExecCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "exec [prompt]",
		Short: "Non-interactive execution mode",
		Long: `Execute Spin in non-interactive mode for CI/CD and automation.

Examples:
  spin exec "run all tests and fix failures"
  echo "refactor authentication" | spin exec
  spin exec --timeout 5m "deploy to staging"
  spin exec --format json "analyze code" | jq`,
		RunE: runExec,
	}

	// Exec-specific flags
	cmd.Flags().Bool("auto-approve", false, "Automatically approve all operations (DANGEROUS)")
	cmd.Flags().String("timeout", "", "Maximum execution time (e.g., 5m, 1h)")
	cmd.Flags().String("format", "text", "Output format (text, json)")
	cmd.Flags().Bool("no-stream", false, "Disable streaming output")
	cmd.Flags().Bool("exit-on-error", true, "Exit immediately on first error")
	cmd.Flags().Bool("debug", false, "Enable debug mode with detailed logging")

	return cmd
}

// runExec executes the exec mode using unified TUI logic but non-interactive.
func runExec(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	setupSignalHandling(cancel)

	// Parse prompt from args or stdin
	prompt, err := parsePrompt(args)
	if err != nil {
		return err
	}

	// Get exec-specific flags
	debugFlag, _ := cmd.Flags().GetBool("debug")
	autoApprove, _ := cmd.Flags().GetBool("auto-approve")
	timeout, _ := cmd.Flags().GetString("timeout")
	format, _ := cmd.Flags().GetString("format")
	noStream, _ := cmd.Flags().GetBool("no-stream")
	exitOnError, _ := cmd.Flags().GetBool("exit-on-error")

	// Load configuration using new unified API
	cfg, err := config.Load(config.Source{
		File: flagConfigFile,
		Flags: config.FlagOverrides{
			Provider: flagProvider,
			Model:    flagModel,
			Debug:    debugFlag,
		},
		WorkDir: flagWorkDir,
	})
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	authMgr := createAuthManager()
	provider, err := buildProvider(ctx, cfg, authMgr)

	if err != nil {
		return fmt.Errorf("create provider: %w", err)
	}
	defer provider.Close()

	// Configure logging based on debug flag
	if debugFlag {
		// In debug mode, enable DEBUG level logs
		slog.SetLogLoggerLevel(slog.LevelDebug)
	} else {
		// In normal mode, suppress INFO/DEBUG logs (only show WARN and ERROR)
		slog.SetLogLoggerLevel(slog.LevelWarn)
	}

	// Apply timeout if specified
	if timeout != "" {
		duration, err := parseDuration(timeout)
		if err != nil {
			return fmt.Errorf("invalid timeout: %w", err)
		}
		ctx, cancel = context.WithTimeout(ctx, duration)
		defer cancel()
	}

	// Create conversation using builder pattern
	conv, err := createConversationForExec(ctx, provider, cfg, autoApprove, debugFlag)
	if err != nil {
		return fmt.Errorf("create conversation: %w", err)
	}
	defer conv.Close()

	// Execute the prompt non-interactively with TUI display
	err = executePromptWithTUI(ctx, conv, prompt, format, noStream, exitOnError)

	// Explicitly exit after execution completes
	if err != nil {
		return err
	}
	return nil
}

// createAuthManager creates an auth manager with platform-specific keystore.
func createAuthManager() *auth.Manager {
	keystore := auth.NewKeystore()
	return auth.NewManager(keystore)
}

// buildProvider creates an LLM provider from configuration.
func buildProvider(ctx context.Context, cfg *config.ConfigV2, authMgr *auth.Manager) (llm.Provider, error) {
	if extra, ok, err := createProviderForExecExtra(cfg.LLM.Provider); err != nil {
		return nil, err
	} else if ok {
		return extra, nil
	}

	b := builder.NewBuilder(cfg, authMgr)
	return b.Build(ctx)
}

// parsePrompt parses the prompt from command line args or stdin.
func parsePrompt(args []string) (string, error) {
	if len(args) > 0 {
		// Join all args as prompt
		return args[0], nil
	}

	// Read from stdin
	data, err := io.ReadAll(os.Stdin)
	if err != nil {
		return "", fmt.Errorf("read stdin: %w", err)
	}

	prompt := string(data)
	if len(prompt) == 0 {
		return "", fmt.Errorf("no prompt provided (use command line args or stdin)")
	}

	return prompt, nil
}

// parseDuration parses a duration string like "5m", "1h", etc.
func parseDuration(s string) (time.Duration, error) {
	return time.ParseDuration(s)
}

// createConversationForExec creates a conversation configured for exec mode using the runtime pattern.
func createConversationForExec(ctx context.Context, provider llm.Provider, cfg *config.ConfigV2, autoApprove bool, debug bool) (*conversation.Conversation, error) {
	workDir := cfg.Agent.WorkDir
	logger := slog.Default()
	emitter := events.NewEventEmitter(100)

	var storage session.Storage

	if cfg.Agent.SessionDir != "" {
		var err error

		storage, err = session.NewFileStorage(cfg.Agent.SessionDir)

		if err != nil {
			return nil, fmt.Errorf("create session storage: %w", err)
		}
	}

	sessionID := ""

	if storage != nil {
		sess := session.NewSession(workDir)
		sessionID = sess.ID
	} else {
		sessionID = fmt.Sprintf("exec-%d", time.Now().UnixNano())
	}

	services, cleanup, err := createServices(cfg, workDir, logger)

	if err != nil {
		return nil, err
	}

	var approvalHandler security.ApprovalHandler

	if autoApprove {
		approvalHandler = createAutoApproveHandler()
	} else {
		approvalHandler = createDenyHandler("exec mode requires --auto-approve for dangerous operations")
	}

	var ui ports.UI

	if termx.IsTerminal(int(os.Stdout.Fd())) && termx.IsTerminal(int(os.Stdin.Fd())) {
		var err error

		ui, err = adapters.NewPureTTY(os.Stdout, adapters.WithExecMode())

		if err != nil {
			cleanup()
			return nil, fmt.Errorf("create TUI: %w", err)
		}
	} else {
		mockTty := &mockTTY{width: 120, height: 30}

		var err error

		ui, err = adapters.NewPureTTY(os.Stdout, adapters.WithExecMode(), adapters.WithTTY(mockTty))

		if err != nil {
			cleanup()
			return nil, fmt.Errorf("create TUI: %w", err)
		}
	}

	builtinRuntime, err := createBuiltinRuntime(
		workDir,
		emitter,
		storage,
		sessionID,
		approvalHandler,
		services,
		ui,
		logger,
		cfg,
	)
	if err != nil {
		cleanup()
		return nil, fmt.Errorf("create builtin runtime: %w", err)
	}

	builder := conversation.NewBuilder(cfg, workDir, builtinRuntime, emitter, provider)

	if services.Git != nil {
		builder = builder.WithGit(services.Git)
	}

	if services.Shell != nil {
		builder = builder.WithShell(services.Shell)
	}

	if services.MCP != nil {
		builder = builder.WithMCP(services.MCP)
	}

	conv, err := builder.Build(ctx)

	if err != nil {
		cleanup()
		return nil, fmt.Errorf("build conversation: %w", err)
	}

	return conv, nil
}

// mockTTY implements term.TerminalController for non-terminal environments.
type mockTTY struct {
	width, height int
}

func (m *mockTTY) Enter() error               { return nil }
func (m *mockTTY) Exit() error                { return nil }
func (m *mockTTY) Size() (int, int)           { return m.width, m.height }
func (m *mockTTY) OnResize(cb func(w, h int)) {}

// executePromptWithTUI executes a prompt non-interactively but shows TUI interface.
func executePromptWithTUI(ctx context.Context, conv *conversation.Conversation, prompt, format string, noStream, exitOnError bool) error {
	// Create TUI adapter. Use real TTY when available; otherwise, mock one.
	opts := []adapters.PureTTYOption{adapters.WithExecMode()}
	if !(termx.IsTerminal(int(os.Stdout.Fd())) && termx.IsTerminal(int(os.Stdin.Fd()))) {
		mockTty := &mockTTY{width: 120, height: 30}
		opts = append(opts, adapters.WithTTY(mockTty))
	}
	ui, err := adapters.NewPureTTY(os.Stdout, opts...)
	if err != nil {
		return fmt.Errorf("create TUI: %w", err)
	}
	defer ui.Stop()

	// Initialize UI with conversation metadata
	ui.SetTaskMode(conv.GetTaskMode())

	// Create event mapper
	mapper := tui.NewTUIMapper(ui)
	defer mapper.Close()

	// Start streaming channel
	streamCh := mapper.StartStreaming()
	streamDone := make(chan struct{})
	go func() {
		ui.PrintChunks(ctx, streamCh)
		close(streamDone)
	}()

	// Subscribe to conversation events
	eventStream := conv.Stream()

	// Start turn in background
	errChan := make(chan error, 1)
	go func() {
		errChan <- conv.RunTurn(ctx, prompt)
	}()

	// Process events and map them to TUI
	// NOTE: In exec mode, we process events but don't wait for the stream to close
	// because the conversation's event stream stays open for potential future turns
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case event, ok := <-eventStream:
				if !ok {
					return
				}
				if err := mapper.MapEvent(event); err != nil {
					ui.PrintLine(fmt.Sprintf("⚠ Mapper error: %v", err))
				}

				// Update token count from conversation history after each event
				if event.Type == events.EventTurnComplete || event.Type == events.EventContentComplete {
					tokenCount := int64(conv.GetTokenCount())
					ui.SetTokenCount(tokenCount)
				}

				// Handle real-time token count updates during turn execution
				if event.Type == events.EventTurnProgress {
					if data, ok := event.Data.(events.TurnEventData); ok {
						if data.TokensUsed > 0 {
							ui.SetTokenCount(int64(data.TokensUsed))
						}
					}
				}
			}
		}
	}()

	// Wait for completion
	select {
	case <-ctx.Done():
		return ctx.Err()

	case err := <-errChan:
		mapper.StopStreaming()

		<-streamDone

		if err != nil {
			if exitOnError {
				return err
			}
			ui.PrintLine(fmt.Sprintf("✗ Error: %v\n", err))
		}

		return nil
	}
}
