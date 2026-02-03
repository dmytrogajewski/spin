package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/dmytrogajewski/spin/internal/config"
	"github.com/dmytrogajewski/spin/internal/conversation"
	"github.com/dmytrogajewski/spin/internal/events"
	"github.com/dmytrogajewski/spin/internal/llm"
	"github.com/dmytrogajewski/spin/internal/security"
	"github.com/dmytrogajewski/spin/internal/session"
	"github.com/dmytrogajewski/spin/internal/tui"
	"github.com/dmytrogajewski/spin/internal/ui/adapters"
	"github.com/spf13/cobra"
)

// newTUICmd creates the TUI command for interactive terminal mode.
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

	// TUI-specific flags
	cmd.Flags().Int("max-turns", 50, "Maximum conversation turns")
	cmd.Flags().Bool("debug", false, "Enable debug mode with detailed logging")
	cmd.Flags().Bool("auto-approve", false, "Automatically approve all operations (DANGEROUS)")

	return cmd
}

// runTUI executes the TUI mode.
func runTUI(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	setupSignalHandling(cancel)

	// Configure logging for TUI mode based on debug flag
	debugFlag, _ := cmd.Flags().GetBool("debug")
	if debugFlag {
		// In debug mode, enable DEBUG level logs
		slog.SetLogLoggerLevel(slog.LevelDebug)
	} else {
		// In normal mode, suppress INFO/DEBUG logs to prevent stderr interference
		setupTUILogging()
	}

	// Get TUI-specific flags
	autoApprove, _ := cmd.Flags().GetBool("auto-approve")
	maxTurns, _ := cmd.Flags().GetInt("max-turns")

	// Load configuration using new unified API
	cfg, err := config.Load(config.Source{
		File: flagConfigFile,
		Flags: config.FlagOverrides{
			Provider: flagProvider,
			Model:    flagModel,
			MaxTurns: maxTurns,
			Debug:    debugFlag,
		},
		WorkDir: flagWorkDir,
	})
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	// Apply --agents-md flag override
	if flagAgentsMD != "" {
		cfg.AgentsMD.Path = flagAgentsMD
	}

	authMgr := createAuthManager()
	provider, err := buildProvider(ctx, cfg, authMgr)
	if err != nil {
		return fmt.Errorf("create provider: %w", err)
	}
	defer provider.Close()

	// Debug flag already applied via config.Load()
	ui, err := adapters.NewPureTTY(os.Stdout)
	if err != nil {
		return fmt.Errorf("create TUI: %w", err)
	}

	// Provide approval TTLs to UI for key preview/TTL hints.
	ui.SetApprovalPolicyTTLs(cfg.Security.SessionPolicyTTL, cfg.Security.GlobalPolicyTTL)

	// Determine the actual model being used (already merged in cfg)
	currentModel := cfg.LLM.Model

	// Set max tokens for context percentage display
	// Try to get actual context window from provider's models
	maxTokens := int64(128000) // Default fallback for modern models
	if models, err := provider.Models(ctx); err == nil && len(models) > 0 {
		// Find the current model
		for _, m := range models {
			if m.ID == currentModel {
				// openai.Model doesn't have ContextSize field
				// Use default maxTokens value
				break
			}
		}
	}
	ui.SetMaxTokens(maxTokens)

	// Start UI in background
	uiCtx, uiCancel := context.WithCancel(ctx)
	defer uiCancel()

	go func() {
		if err := ui.Run(uiCtx); err != nil && err != context.Canceled {
			fmt.Fprintf(os.Stderr, "TUI error: %v\n", err)
		}
	}()
	defer ui.Stop()

	conv, err := createConversationForTUI(ctx, provider, cfg, ui, autoApprove)
	if err != nil {
		return fmt.Errorf("create conversation: %w", err)
	}
	defer conv.Close()

	// Initialize UI with conversation metadata
	initializeUI(ui, conv, provider, currentModel)

	// Create event mapper for TUI
	// The runtime has its own internal mapper for notifications, but we need a separate one
	// for processing the conversation event stream and updating the UI
	mapper := tui.NewTUIMapper(ui)
	defer mapper.Close()

	// Start streaming channel
	streamCh := mapper.StartStreaming()
	streamDone := make(chan struct{})
	go func() {
		ui.PrintChunks(ctx, streamCh)
		close(streamDone)
	}()

	// Print welcome message
	ui.PrintLine("")
	ui.PrintLine(SpinLogo)
	ui.PrintLine("Type your prompt and press Enter.")
	ui.PrintLine("Commands: /mode [name], /help, /exit (or press Ctrl-D)\n")

	// Subscribe to conversation events
	eventStream := conv.Stream()

	// Start event processing loop
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
				if err := mapper.MapEvent(event); err != nil {
					ui.PrintLine(fmt.Sprintf("⚠ Mapper error: %v", err))
				}

				// Update token count from conversation history after each event
				// This ensures the status bar always shows current cumulative total
				if event.Type == events.EventTurnComplete ||
					event.Type == events.EventContentComplete ||
					event.Type == events.EventToolCallComplete {
					tokenCount := int64(conv.GetTokenCount())
					ui.SetTokenCount(tokenCount)
				}

				// Handle real-time token count updates during turn execution
				// This shows estimated tokens as the turn progresses (before history is updated)
				if event.Type == events.EventTurnProgress {
					if data, ok := event.Data.(events.TurnEventData); ok {
						if data.TokensUsed > 0 {
							// Use the estimated token count from the event
							ui.SetTokenCount(int64(data.TokensUsed))
						}
					}
				}
			}
		}
	}()

	// Main input loop
	inputCh := ui.RequestInput()
	for {
		select {
		case <-ctx.Done():
			// Wait for event processing to finish
			<-eventDone
			return ctx.Err()

		case line, ok := <-inputCh:
			if !ok {
				// UI closed (Ctrl-D)
				<-eventDone
				return nil
			}

			if line == "" {
				continue
			}

			// Check if input is a command
			cmdResult := parseCommand(line)

			if cmdResult.isCommand {
				// Handle command
				_, err := handleCommand(ui, conv, cmdResult.command, cmdResult.args)
				if err != nil {
					if err.Error() == "exit requested" {
						<-eventDone
						return nil
					}
					ui.PrintLine(fmt.Sprintf("Command error: %v\n", err))
				}
				// Skip conversation turn for commands
				continue
			}

			// Submit prompt to conversation
			turnCtx, turnCancel := context.WithCancel(ctx)
			defer turnCancel()

			// Send message and handle errors
			err := conv.RunTurn(turnCtx, line)

			// Stop streaming to close the channel (this triggers final newline in PrintChunks)
			mapper.StopStreaming()

			// Wait for streaming to complete
			<-streamDone

			// Reset streamDone for next turn
			streamDone = make(chan struct{})
			streamCh = mapper.StartStreaming()
			go func() {
				ui.PrintChunks(ctx, streamCh)
				close(streamDone)
			}()

			if err != nil {
				ui.PrintLine(fmt.Sprintf("✗ Error: %v\n", err))
			}
		}
	}
}

// createConversationForTUI creates a conversation configured for TUI mode using the runtime pattern.
func createConversationForTUI(ctx context.Context, provider llm.Provider, cfg *config.ConfigV2, ui *adapters.PureTTY, autoApprove bool) (*conversation.Conversation, error) {
	workDir := cfg.Agent.WorkDir
	logger := slog.Default()

	// 1. Create shared infrastructure
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
		sessionID = fmt.Sprintf("tui-%d", time.Now().UnixNano())
	}

	// 2. Create protocol services
	protocolServices, cleanup, err := createServices(ctx, cfg, workDir, logger)
	if err != nil {
		return nil, err
	}

	// 3. Create approval handler (TUI-specific)
	var approvalHandler security.ApprovalHandler
	if autoApprove {
		approvalHandler = createAutoApproveHandler()
	} else {
		approvalHandler = createTUIApprovalHandler(ui)
	}

	// 4. Create builtin runtime (complete, self-contained)
	builtinRuntime, err := createBuiltinRuntime(
		workDir,
		emitter,
		storage,
		sessionID,
		approvalHandler,
		protocolServices,
		ui,
		logger,
		cfg,
	)
	if err != nil {
		cleanup()
		return nil, fmt.Errorf("create builtin runtime: %w", err)
	}

	// 5. Build conversation with runtime
	builder := conversation.NewBuilder(cfg, workDir, builtinRuntime, emitter, provider)

	// Add optional services
	if protocolServices.Git != nil {
		builder = builder.WithGit(protocolServices.Git)
	}
	if protocolServices.Shell != nil {
		builder = builder.WithShell(protocolServices.Shell)
	}
	if protocolServices.MCP != nil {
		builder = builder.WithMCP(protocolServices.MCP)

		// Create dynamic tool selector if any registry has dynamic_loadout
		if toolSelector := createToolSelector(protocolServices.MCP, nil, emitter, cfg, slog.Default()); toolSelector != nil {
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

// setupSignalHandling sets up signal handling for graceful shutdown.
func setupSignalHandling(cancel context.CancelFunc) {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigCh
		cancel()
	}()
}

// setupTUILogging configures logging for TUI mode to prevent stderr interference.
func setupTUILogging() {
	// Set slog level to WARN to suppress INFO level logs that interfere with TUI
	// This prevents logs like "Shell integration initialized" from appearing in stderr
	// while the TUI is running, which causes formatting conflicts
	slog.SetLogLoggerLevel(slog.LevelWarn)
}

// initializeUI initializes the UI with conversation metadata.
func initializeUI(ui *adapters.PureTTY, conv *conversation.Conversation, provider llm.Provider, model string) {
	taskMode := conv.GetTaskMode()
	ui.SetTaskMode(taskMode)

	providerName := provider.Name()
	ui.SetProviderInfo(providerName, model)

	tokenCount := int64(conv.GetTokenCount())
	ui.SetTokenCount(tokenCount)

	sessionID := conv.GetSessionID()
	ui.SetConversationID(sessionID)
}
