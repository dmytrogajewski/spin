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
	"github.com/dmytrogajewski/spin/internal/ui/banner"
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

	printTUIWelcome(ui)

	configureMaxTokens(ui, cfg, provider)

	return cfg, provider, ui, nil
}

// configureMaxTokens sets the status-bar context window from config or the provider.
func configureMaxTokens(ui *adapters.PureTTY, cfg *config.V2, provider llm.Provider) {
	ui.SetMaxTokens(int64(resolveUIContextWindow(cfg, provider)))
}

// resolveUIContextWindow returns the token window shown in the TUI status bar.
// Config override wins, then provider-detected capabilities, then defaultMaxTokens.
func resolveUIContextWindow(cfg *config.V2, provider llm.Provider) int {
	if cfg != nil && cfg.LLM.ContextWindow > 0 {
		return cfg.LLM.ContextWindow
	}

	if provider != nil {
		if n := provider.Capabilities().ContextWindow; n > 0 {
			return n
		}
	}

	return defaultMaxTokens
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
func processEvent(
	ctx context.Context, event events.Event, mapper *tui.Mapper,
	ui *adapters.PureTTY, conv *conversation.Conversation, tokens *tokenCounter,
) {
	mapErr := mapper.MapEvent(ctx, event)
	if mapErr != nil {
		_ = ui.PrintLine(fmt.Sprintf("⚠ Mapper error: %v", mapErr))
	}

	tokens.update(event, ui, conv)
}

// tokenSink receives context token counts for display.
type tokenSink interface {
	SetTokenCount(tokenCount int64)
}

// tokenSource estimates conversation tokens from history.
type tokenSource interface {
	GetTokenCount() int
}

// tokenCounter tracks the context counter shown in the status bar.
// Providers report real usage (prompt = full context actually processed)
// via EventTurnProgress; once seen, that value wins over the history
// estimate, which undercounts by excluding system prompt and tool schemas.
type tokenCounter struct {
	sawRealUsage bool
}

// update refreshes the token count from a conversation event.
// It is called from the single event-loop goroutine, so no locking is needed.
func (tc *tokenCounter) update(event events.Event, sink tokenSink, source tokenSource) {
	switch event.Type {
	case events.EventTurnProgress:
		if data, ok := event.Data.(events.TurnEventData); ok && data.TokensUsed > 0 {
			tc.sawRealUsage = true

			sink.SetTokenCount(int64(data.TokensUsed))
		}
	case events.EventTurnComplete, events.EventContentComplete, events.EventToolCallComplete:
		if !tc.sawRealUsage {
			sink.SetTokenCount(int64(source.GetTokenCount()))
		}
	default:
		// Other event types don't affect the token count.
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

		tokens := &tokenCounter{}

		for {
			select {
			case <-ctx.Done():
				return
			case event, ok := <-eventStream:
				if !ok {
					return
				}

				processEvent(ctx, event, mapper, ui, conv, tokens)
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
	ui.SetAgentState(ctx, "Starting")

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

	stopBlink := startCatBlink(ctx, ui)
	defer stopBlink()

	ui.SetTranscriptStartHook(stopBlink)

	var approvalHandler = createTUIApprovalHandler(ui)
	if flags.autoApprove {
		approvalHandler = createAutoApproveHandler()

		ui.SetApprovalMode(adapters.ApprovalModeYolo)
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

	initializeUI(ui, conv, provider, cfg)

	mapper := tui.NewMapper(ui)
	defer mapper.Close()

	streamCh := mapper.StartStreaming()
	streamDone := make(chan struct{})

	go func() {
		_ = ui.PrintChunks(ctx, streamCh)

		close(streamDone)
	}()

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

// Welcome banner layout constants.
const (
	// termClearHome homes the cursor, clears the screen, and purges the
	// terminal scrollback (ED3, same as clear(1)). The clear gives the
	// banner a deterministic row (required by the idle blink overlay);
	// the purge removes stale frames from previous runs that terminals
	// like xterm.js would otherwise keep forever at the top of scrollback.
	termClearHome = "\x1b[H\x1b[2J\x1b[3J"
	// bannerBaseRow is the terminal row of the banner's first line after clearing.
	bannerBaseRow = 1
	// welcomeFooterLines is the number of lines printed below the banner.
	welcomeFooterLines = 5
	// statusReserveLines is the bottom area reserved for the status bar and prompt.
	statusReserveLines = 2
	// minBlinkTermWidth guards against banner line wrapping which would
	// break the blink overlay row math.
	minBlinkTermWidth = 60
)

func printTUIWelcome(ui *adapters.PureTTY) {
	_, _ = io.WriteString(ui.Out(), termClearHome)
	_ = banner.Play(ui.Out(), banner.PlayOptions{})
	_, _ = io.WriteString(ui.Out(),
		"\nType your prompt and press Enter.\n"+
			"Commands: /mode [name], /resume, /help, /exit (or press Ctrl-D)\n"+
			"Shift+Tab: cycle approvals (ask / yolo)\n")
}

// startCatBlink keeps the welcome mascot blinking until the returned stop
// function is called. It is a no-op on terminals too small to hold the
// whole welcome screen without scrolling. The stop function is idempotent
// and blocks until the blink goroutine has fully exited.
func startCatBlink(ctx context.Context, ui *adapters.PureTTY) func() {
	width, height := ui.TermSize()
	if height < banner.Height()+welcomeFooterLines+statusReserveLines || width < minBlinkTermWidth {
		return func() {}
	}

	blinkCtx, cancel := context.WithCancel(ctx)
	done := make(chan struct{})

	go func() {
		defer close(done)

		_ = banner.Blink(blinkCtx, ui, banner.BlinkOptions{
			BaseRow: bannerBaseRow,
			Active: func() bool {
				w, h := ui.TermSize()

				return w == width && h == height
			},
		})
	}()

	return func() {
		cancel()
		<-done
	}
}

// initializeUI initializes the UI with conversation metadata.
func initializeUI(ui *adapters.PureTTY, conv *conversation.Conversation, provider llm.Provider, cfg *config.V2) {
	taskMode := conv.GetTaskMode()
	ui.SetTaskMode(taskMode)

	providerName := provider.Name()
	ui.SetProviderInfo(providerName, cfg.LLM.Model)
	configureMaxTokens(ui, cfg, provider)

	tokenCount := int64(conv.GetTokenCount())
	ui.SetTokenCount(tokenCount)

	sessionID := conv.ID()
	ui.SetConversationID(sessionID)
}
