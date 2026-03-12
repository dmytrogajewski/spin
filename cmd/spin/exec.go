package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"time"

	"github.com/spf13/cobra"
	termx "golang.org/x/term"

	"github.com/dmytrogajewski/spin/internal/agent/executor"
	spinterm "github.com/dmytrogajewski/spin/internal/ui/term"
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
)

var ErrNoPromptProvidedUseCommandLine = errors.New("no prompt provided (use command line args or stdin)")

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

	// Exec-specific flags.
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

	// Parse prompt from args or stdin.
	prompt, err := parsePrompt(args)
	if err != nil {
		return err
	}

	// Get exec-specific flags.
	debugFlag, _ := cmd.Flags().GetBool("debug")
	autoApprove, _ := cmd.Flags().GetBool("auto-approve")
	timeout, _ := cmd.Flags().GetString("timeout")
	format, _ := cmd.Flags().GetString("format")
	noStream, _ := cmd.Flags().GetBool("no-stream")
	exitOnError, _ := cmd.Flags().GetBool("exit-on-error")

	// Load configuration using new unified API.
	cfg, err := config.Load(config.Source{
		File: flagConfigFile(cmd),
		Flags: config.FlagOverrides{
			Provider: flagProvider(cmd),
			Model:    flagModel(cmd),
			Debug:    debugFlag,
		},
		WorkDir: flagWorkDir(cmd),
	})
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	// Apply --agents-md flag override.
	if agentsMD := flagAgentsMD(cmd); agentsMD != "" {
		cfg.AgentsMD.Path = agentsMD
	}

	authMgr := createAuthManager()

	provider, err := buildProvider(ctx, cfg, authMgr)
	if err != nil {
		return fmt.Errorf("create provider: %w", err)
	}

	defer provider.Close()

	// Configure logging based on debug flag.
	if debugFlag {
		// In debug mode, enable DEBUG level logs.
		slog.SetLogLoggerLevel(slog.LevelDebug)
	} else {
		// In normal mode, suppress INFO/DEBUG logs (only show WARN and ERROR).
		slog.SetLogLoggerLevel(slog.LevelWarn)
	}

	// Apply timeout if specified.
	if timeout != "" {
		var duration time.Duration
		duration, err = parseDuration(timeout)
		if err != nil {
			return fmt.Errorf("invalid timeout: %w", err)
		}

		ctx, cancel = context.WithTimeout(ctx, duration)
		defer cancel()
	}

	// Create conversation using builder pattern.
	conv, err := createConversationForExec(ctx, provider, cfg, autoApprove, debugFlag)
	if err != nil {
		return fmt.Errorf("create conversation: %w", err)
	}
	defer conv.Close()

	// Execute the prompt non-interactively with TUI display.
	err = executePromptWithTUI(ctx, conv, prompt, format, noStream, exitOnError)

	// Explicitly exit after execution completes.
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
func buildProvider(ctx context.Context, cfg *config.V2, authMgr *auth.Manager) (llm.Provider, error) {
	extra, ok, err := createProviderForExecExtra(cfg.LLM.Provider)
	if err != nil {
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
		// Join all args as prompt.
		return args[0], nil
	}

	// Read from stdin.
	data, err := io.ReadAll(os.Stdin)
	if err != nil {
		return "", fmt.Errorf("read stdin: %w", err)
	}

	prompt := string(data)
	if len(prompt) == 0 {
		return "", ErrNoPromptProvidedUseCommandLine
	}

	return prompt, nil
}

// parseDuration parses a duration string like "5m", "1h", etc.
func parseDuration(s string) (time.Duration, error) {
	d, err := time.ParseDuration(s)
	if err != nil {
		return 0, fmt.Errorf("parsing duration %q: %w", s, err)
	}

	return d, nil
}

// resolveSessionID determines the session ID based on storage availability.
func resolveSessionID(storage session.Storage, workDir, prefix string) string {
	if storage != nil {
		sess := session.NewSession(workDir)
		return sess.ID
	}

	return fmt.Sprintf("%s-%d", prefix, time.Now().UnixNano())
}

// createExecUI creates the UI adapter for exec mode.
func createExecUI() (ports.UI, error) {
	opts := []adapters.PureTTYOption{adapters.WithExecMode()}

	if !termx.IsTerminal(spinterm.SafeFd(os.Stdout.Fd())) || !termx.IsTerminal(spinterm.SafeFd(os.Stdin.Fd())) {
		mockTty := &mockTTY{width: 120, height: 30}
		opts = append(opts, adapters.WithTTY(mockTty))
	}

	return adapters.NewPureTTY(os.Stdout, opts...)
}

// buildConversation wires services into a conversation builder and builds the conversation.
func buildConversation(ctx context.Context, cfg *config.V2, workDir string, builtinRuntime *executor.BuiltinRuntime, emitter *events.EventEmitter, provider llm.Provider, services *ProtocolServices, cleanup func()) (*conversation.Conversation, error) {
	builder := conversation.NewBuilder(cfg, workDir, builtinRuntime, emitter, provider)

	if services.Git != nil {
		builder = builder.WithGit(services.Git)
	}

	if services.Shell != nil {
		builder = builder.WithShell(services.Shell)
	}

	if services.MCP != nil {
		builder = builder.WithMCP(services.MCP)

		if toolSelector := createToolSelector(ctx, services.MCP, nil, emitter, cfg, slog.Default()); toolSelector != nil {
			builder = builder.WithToolSelector(toolSelector)
		}
	}

	conv, err := builder.Build(ctx)
	if err != nil {
		cleanup()
		return nil, fmt.Errorf("build conversation: %w", err)
	}

	return conv, nil
}

// createConversationForExec creates a conversation configured for exec mode using the runtime pattern.
func createConversationForExec(ctx context.Context, provider llm.Provider, cfg *config.V2, autoApprove bool, _ bool) (*conversation.Conversation, error) {
	workDir := cfg.Agent.WorkDir
	logger := slog.Default()
	emitter := events.NewEventEmitter(100)

	storage, err := createSessionStorage(cfg.Agent.SessionDir)
	if err != nil {
		return nil, err
	}

	sessionID := resolveSessionID(storage, workDir, "exec")

	services, cleanup, err := createServices(ctx, cfg, workDir, logger)
	if err != nil {
		return nil, err
	}

	var approvalHandler security.ApprovalHandler
	if autoApprove {
		approvalHandler = createAutoApproveHandler()
	} else {
		approvalHandler = createDenyHandler("exec mode requires --auto-approve for dangerous operations")
	}

	ui, err := createExecUI()
	if err != nil {
		cleanup()
		return nil, fmt.Errorf("create TUI: %w", err)
	}

	builtinRuntime, err := createBuiltinRuntime(workDir, emitter, storage, sessionID, approvalHandler, services, ui, logger, cfg)
	if err != nil {
		cleanup()
		return nil, fmt.Errorf("create builtin runtime: %w", err)
	}

	return buildConversation(ctx, cfg, workDir, builtinRuntime, emitter, provider, services, cleanup)
}

// mockTTY implements term.TerminalController for non-terminal environments.
type mockTTY struct {
	width, height int
}

func (m *mockTTY) Enter() error               { return nil }
func (m *mockTTY) Exit() error                { return nil }
func (m *mockTTY) Size() (int, int)           { return m.width, m.height }
func (m *mockTTY) OnResize(_ func(w, h int)) {}

// processExecEvent handles a single event in exec mode.
func processExecEvent(ctx context.Context, event events.Event, mapper *tui.Mapper, ui *adapters.PureTTY, conv *conversation.Conversation) {
	mapErr := mapper.MapEvent(ctx, event)
	if mapErr != nil {
		_ = ui.PrintLine(fmt.Sprintf("⚠ Mapper error: %v", mapErr))
	}

	switch event.Type {
	case events.EventTurnComplete, events.EventContentComplete:
		ui.SetTokenCount(int64(conv.GetTokenCount()))
	case events.EventTurnProgress:
		if data, ok := event.Data.(events.TurnEventData); ok && data.TokensUsed > 0 {
			ui.SetTokenCount(int64(data.TokensUsed))
		}
	}
}

// startExecEventLoop starts the event processing goroutine for exec mode.
func startExecEventLoop(ctx context.Context, eventStream <-chan events.Event, mapper *tui.Mapper, ui *adapters.PureTTY, conv *conversation.Conversation) {
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case event, ok := <-eventStream:
				if !ok {
					return
				}
				processExecEvent(ctx, event, mapper, ui, conv)
			}
		}
	}()
}

// executePromptWithTUI executes a prompt non-interactively but shows TUI interface.
func executePromptWithTUI(ctx context.Context, conv *conversation.Conversation, prompt, _ string, _, exitOnError bool) error {
	ui, err := createExecUI()
	if err != nil {
		return fmt.Errorf("create TUI: %w", err)
	}

	pureTTY := ui.(*adapters.PureTTY)
	defer func() { _ = pureTTY.Stop() }()

	pureTTY.SetTaskMode(conv.GetTaskMode())

	mapper := tui.NewMapper(pureTTY)
	defer mapper.Close()

	streamCh := mapper.StartStreaming()
	streamDone := make(chan struct{})

	go func() {
		_ = pureTTY.PrintChunks(ctx, streamCh)
		close(streamDone)
	}()

	startExecEventLoop(ctx, conv.Stream(), mapper, pureTTY, conv)

	errChan := make(chan error, 1)
	go func() {
		errChan <- conv.RunTurn(ctx, prompt)
	}()

	select {
	case <-ctx.Done():
		return fmt.Errorf("execution canceled: %w", ctx.Err())

	case err = <-errChan:
		mapper.StopStreaming()
		<-streamDone

		if err != nil && exitOnError {
			return err
		}

		if err != nil {
			_ = pureTTY.PrintLine(fmt.Sprintf("✗ Error: %v\n", err))
		}

		return nil
	}
}
