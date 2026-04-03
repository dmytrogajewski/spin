package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/dmytrogajewski/spin/internal/config"
	"github.com/dmytrogajewski/spin/internal/conversation"
	"github.com/dmytrogajewski/spin/internal/events"
	"github.com/dmytrogajewski/spin/internal/llm"
	"github.com/dmytrogajewski/spin/internal/tui"
	"github.com/dmytrogajewski/spin/internal/ui/adapters"
)

// newTUICmd creates the TUI command for interactive terminal mode.
const (
	defaultMaxTurns  = 50
	defaultMaxTokens = 128000
)

func newTUICmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tui",
		Short: "Interactive terminal UI mode",
		Long: `Launch Spin with an interactive terminal user interface.

The TUI provides a native-scrollback interface with:
  • Block-based timeline (EXECUTE, READ, APPLY_PATCH, etc.)
  • Real-time LLM streaming
  • Keyboard-first navigation (PgUp/PgDn, g/G)
  • Command palette (Ctrl-P)
  • Timeline filtering (/)
  • Approval dialogs for dangerous commands

Examples:
  spin tui
  spin tui --model llama3.1
  spin tui --provider anthropic --model claude-3-5-sonnet-20241022
  spin tui --auto-approve  # Automatically approve all operations`,
		RunE: runTUI,
	}

	// TUI-specific flags.
	cmd.Flags().Int("max-turns", defaultMaxTurns, "Maximum conversation turns")
	cmd.Flags().Bool("debug", false, "Enable debug mode with detailed logging")
	cmd.Flags().Bool("auto-approve", false, "Automatically approve all operations (DANGEROUS)")

	return cmd
}

// tuiFlags holds parsed TUI command flags.
type tuiFlags struct {
	debug       bool
	autoApprove bool
	maxTurns    int
}

// parseTUIFlags extracts TUI-specific flags from the command.
func parseTUIFlags(cmd *cobra.Command) tuiFlags {
	debugFlag, _ := cmd.Flags().GetBool("debug")
	autoApprove, _ := cmd.Flags().GetBool("auto-approve")
	maxTurns, _ := cmd.Flags().GetInt("max-turns")

	return tuiFlags{
		debug:       debugFlag,
		autoApprove: autoApprove,
		maxTurns:    maxTurns,
	}
}

// configureTUILogging sets up logging based on the debug flag.
// In TUI mode, logs must never go to stderr as they break the terminal display.
// Debug mode writes logs to ~/.spin/spin.log; normal mode discards them.
func configureTUILogging(debug bool) {
	if debug {
		logFile := openTUILogFile()
		handler := slog.NewTextHandler(logFile, &slog.HandlerOptions{Level: slog.LevelDebug})
		slog.SetDefault(slog.New(handler))
	} else {
		slog.SetDefault(slog.New(slog.DiscardHandler))
	}
}

// openTUILogFile opens the log file for debug mode, falling back to discard on error.
func openTUILogFile() io.Writer {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return io.Discard
	}

	spinDir := filepath.Join(homeDir, ".spin")
	if mkdirErr := os.MkdirAll(spinDir, 0o700); mkdirErr != nil {
		return io.Discard
	}

	f, err := os.OpenFile(filepath.Join(spinDir, "spin.log"), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return io.Discard
	}

	return f
}

// setupTUIProvider creates and configures the TUI provider and UI components.
// The out writer is used for TUI output; when nil it defaults to
// [os.Stdout].
func setupTUIProvider(
	ctx context.Context, cmd *cobra.Command, flags tuiFlags, out io.Writer,
) (*config.V2, llm.Provider, *adapters.PureTTY, error) {
	cfg, err := config.Load(config.Source{
		File: flagConfigFile(cmd),
		Flags: config.FlagOverrides{
			Provider: flagProvider(cmd),
			Model:    flagModel(cmd),
			MaxTurns: flags.maxTurns,
			Debug:    flags.debug,
		},
		WorkDir: flagWorkDir(cmd),
	})
	if err != nil {
		return nil, nil, nil, fmt.Errorf("load config: %w", err)
	}

	if agentsMD := flagAgentsMD(cmd); agentsMD != "" {
		cfg.AgentsMD.Path = agentsMD
	}

	authMgr := createAuthManager()

	provider, err := buildProvider(ctx, cfg, authMgr)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("create provider: %w", err)
	}

	if out == nil {
		out = os.Stdout
	}

	ui, err := adapters.NewPureTTY(out)
	if err != nil {
		provider.Close()

		return nil, nil, nil, fmt.Errorf("create TUI: %w", err)
	}

	ui.SetApprovalPolicyTTLs(cfg.Security.SessionPolicyTTL, cfg.Security.GlobalPolicyTTL)
	configureMaxTokens(ui)

	return cfg, provider, ui, nil
}

// configureMaxTokens sets max tokens on the UI.
func configureMaxTokens(ui *adapters.PureTTY) {
	ui.SetMaxTokens(int64(defaultMaxTokens))
}

// startTUIBackground starts the UI and streaming in the background.
// The errOut writer receives error messages; when nil it defaults to [os.Stderr].
func startTUIBackground(ctx context.Context, ui *adapters.PureTTY, errOut io.Writer) context.CancelFunc {
	if errOut == nil {
		errOut = os.Stderr
	}

	uiCtx, uiCancel := context.WithCancel(ctx)

	go func() {
		runErr := ui.Run(uiCtx)
		if runErr != nil && !errors.Is(runErr, context.Canceled) {
			fmt.Fprintf(errOut, "TUI error: %v\n", runErr)
		}
	}()

	return uiCancel
}

// processEvent handles a single event from the conversation stream.
func processEvent(ctx context.Context, event events.Event, mapper *tui.Mapper, ui *adapters.PureTTY, conv *conversation.Conversation) {
	mapErr := mapper.MapEvent(ctx, event)
	if mapErr != nil {
		_ = ui.PrintLine(fmt.Sprintf("⚠ Mapper error: %v", mapErr))
	}

	updateTokensFromEvent(event, ui, conv)
}

// updateTokensFromEvent updates token counts based on event type.
func updateTokensFromEvent(event events.Event, ui *adapters.PureTTY, conv *conversation.Conversation) {
	switch event.Type {
	case events.EventTurnComplete, events.EventContentComplete, events.EventToolCallComplete:
		ui.SetTokenCount(int64(conv.GetTokenCount()))
	case events.EventTurnProgress:
		if data, ok := event.Data.(events.TurnEventData); ok && data.TokensUsed > 0 {
			ui.SetTokenCount(int64(data.TokensUsed))
		}
	default:
		// Other event types don't require token count updates.
	}
}

// startEventLoop starts the event processing goroutine.
func startEventLoop(
	ctx context.Context, eventStream <-chan events.Event,
	mapper *tui.Mapper, ui *adapters.PureTTY,
	conv *conversation.Conversation,
) <-chan struct{} {
	eventDone := make(chan struct{})

	go func() {
		defer close(eventDone)

		for {
			select {
			case <-ctx.Done():
				return
			case event, ok := <-eventStream:
				if !ok {
					return
				}

				processEvent(ctx, event, mapper, ui, conv)
			}
		}
	}()

	return eventDone
}

// handleTUIInput processes a single line of TUI input.
// Returns whether to exit and the new streamDone channel (unchanged if command).
func handleTUIInput(
	ctx context.Context, line string, ui *adapters.PureTTY,
	conv *conversation.Conversation, mapper *tui.Mapper,
	streamDone chan struct{},
) (shouldExit bool, newStreamDone chan struct{}) {
	cmdResult := parseCommand(line)
	if cmdResult.isCommand {
		exit := handleTUICommand(ctx, ui, conv, cmdResult)

		return exit, streamDone
	}

	newDone := executeTurn(ctx, line, conv, mapper, ui, streamDone)

	return false, newDone
}

// handleTUICommand handles a parsed command input. Returns true if exit is requested.
func handleTUICommand(ctx context.Context, ui *adapters.PureTTY, conv *conversation.Conversation, cmdResult commandResult) bool {
	cmdErr := handleCommand(ctx, ui, conv, cmdResult.command, cmdResult.args)
	if cmdErr == nil {
		return false
	}

	if errors.Is(cmdErr, ErrExitRequested) {
		return true
	}

	_ = ui.PrintLine(fmt.Sprintf("Command error: %v\n", cmdErr))

	return false
}

// executeTurn runs a conversation turn and resets streaming.
// Returns the new streamDone channel for the next turn.
func executeTurn(
	ctx context.Context, line string,
	conv *conversation.Conversation, mapper *tui.Mapper,
	ui *adapters.PureTTY, streamDone chan struct{},
) chan struct{} {
	turnCtx, turnCancel := context.WithCancel(ctx)
	turnErr := conv.RunTurn(turnCtx, line)

	turnCancel()

	mapper.StopStreaming()
	<-streamDone

	newDone := make(chan struct{})
	streamCh := mapper.StartStreaming()

	go func() {
		_ = ui.PrintChunks(ctx, streamCh)

		close(newDone)
	}()

	if turnErr != nil {
		_ = ui.PrintLine(fmt.Sprintf("✗ Error: %v\n", turnErr))
	}

	return newDone
}

// runTUI executes the TUI mode.
const tuiEventBuffer = 100

func runTUI(cmd *cobra.Command, _ []string) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	setupSignalHandling(cancel)

	flags := parseTUIFlags(cmd)
	configureTUILogging(flags.debug)

	cfg, provider, ui, err := setupTUIProvider(ctx, cmd, flags, cmd.OutOrStdout())
	if err != nil {
		return err
	}
	defer provider.Close()

	uiCancel := startTUIBackground(ctx, ui, cmd.ErrOrStderr())
	defer uiCancel()
	defer func() { _ = ui.Stop() }()

	var approvalHandler = createTUIApprovalHandler(ui)
	if flags.autoApprove {
		approvalHandler = createAutoApproveHandler()
	}

	conv, err := createConversation(ctx, provider, cfg, conversationConfig{
		approvalHandler: approvalHandler,
		ui:              ui,
		sessionPrefix:   "tui",
		eventBufferSize: tuiEventBuffer,
	})
	if err != nil {
		return fmt.Errorf("create conversation: %w", err)
	}
	defer conv.Close(ctx)

	initializeUI(ui, conv, provider, cfg.LLM.Model)

	mapper := tui.NewMapper(ui)
	defer mapper.Close()

	streamCh := mapper.StartStreaming()
	streamDone := make(chan struct{})

	go func() {
		_ = ui.PrintChunks(ctx, streamCh)

		close(streamDone)
	}()

	_ = ui.PrintLine("")
	_ = ui.PrintLine(SpinLogo)
	_ = ui.PrintLine("Type your prompt and press Enter.")
	_ = ui.PrintLine("Commands: /mode [name], /help, /exit (or press Ctrl-D)\n")

	eventDone := startEventLoop(ctx, conv.Stream(), mapper, ui, conv)
	inputCh := ui.RequestInput()

	for {
		select {
		case <-ctx.Done():
			<-eventDone

			return fmt.Errorf("TUI loop canceled: %w", ctx.Err())

		case line, ok := <-inputCh:
			if !ok {
				<-eventDone

				return nil
			}

			if line == "" {
				continue
			}

			shouldExit, newDone := handleTUIInput(ctx, line, ui, conv, mapper, streamDone)
			streamDone = newDone

			if shouldExit {
				<-eventDone

				return nil
			}
		}
	}
}

// setupSignalHandling sets up signal handling for graceful shutdown.
// An optional shutdownMsg is printed to stderr when a signal is received.
func setupSignalHandling(cancel context.CancelFunc, shutdownMsg ...string) {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigCh

		if len(shutdownMsg) > 0 && shutdownMsg[0] != "" {
			fmt.Fprintln(os.Stderr, shutdownMsg[0])
		}

		cancel()
	}()
}

// initializeUI initializes the UI with conversation metadata.
func initializeUI(ui *adapters.PureTTY, conv *conversation.Conversation, provider llm.Provider, model string) {
	taskMode := conv.GetTaskMode()
	ui.SetTaskMode(taskMode)

	providerName := provider.Name()
	ui.SetProviderInfo(providerName, model)

	tokenCount := int64(conv.GetTokenCount())
	ui.SetTokenCount(tokenCount)

	sessionID := conv.ID()
	ui.SetConversationID(sessionID)
}
