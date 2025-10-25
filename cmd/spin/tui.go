package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/dmytrogajewski/spin/internal/config"
	"github.com/dmytrogajewski/spin/internal/conversation"
	"github.com/dmytrogajewski/spin/internal/events"
	"github.com/dmytrogajewski/spin/internal/llm"
	"github.com/dmytrogajewski/spin/internal/manager"
	"github.com/dmytrogajewski/spin/internal/security"
	"github.com/dmytrogajewski/spin/internal/tools"
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
	if !debugFlag {
		// Suppress INFO level logs to prevent stderr interference (only when not in debug mode)
		setupTUILogging()
	} else {
		// In debug mode, suppress slog to stdout but keep structured logging via events
		// This prevents duplicate logs while maintaining detailed logging to JSONL file
		slog.SetLogLoggerLevel(slog.LevelError)
	}

	configLoader, err := loadConfig()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	authMgr := createAuthManager()
	provider, err := buildProvider(ctx, configLoader, authMgr)
	if err != nil {
		return fmt.Errorf("create provider: %w", err)
	}
	defer provider.Close()

	maxTurns, _ := cmd.Flags().GetInt("max-turns")
	autoApprove, _ := cmd.Flags().GetBool("auto-approve")

	ui, err := adapters.NewPureTTY(os.Stdout)
	if err != nil {
		return fmt.Errorf("create TUI: %w", err)
	}

	// Set max tokens for context percentage display
	// Try to get actual context window from provider's models
	maxTokens := int64(128000) // Default fallback for modern models
	if models, err := provider.Models(ctx); err == nil && len(models) > 0 {
		// Find the current model
		currentModel := flagModel
		if currentModel == "" {
			currentModel = configLoader.GetString("model")
		}
		for _, m := range models {
			if m.ID == currentModel || m.Name == currentModel {
				if m.ContextSize > 0 {
					maxTokens = int64(m.ContextSize)
					break
				}
			}
		}
	}
	ui.SetMaxTokens(maxTokens)

	mgr, err := createManagerForTUI(provider, maxTurns, configLoader, ui, debugFlag, autoApprove)
	if err != nil {
		return fmt.Errorf("create manager: %w", err)
	}

	// Start UI in background
	uiCtx, uiCancel := context.WithCancel(ctx)
	defer uiCancel()

	go func() {
		if err := ui.Run(uiCtx); err != nil && err != context.Canceled {
			fmt.Fprintf(os.Stderr, "TUI error: %v\n", err)
		}
	}()
	defer ui.Stop()

	workDir := getWorkingDirectory()
	conv, err := mgr.NewConversation(ctx, workDir)
	if err != nil {
		return fmt.Errorf("create conversation: %w", err)
	}
	defer mgr.Close()

	// Initialize UI with conversation metadata
	initializeUI(ui, conv, provider)

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
				_, err := handleCommand(ui, conv, cmdResult)
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

// createManagerForTUI creates a core.Manager configured for TUI mode.
func createManagerForTUI(provider llm.Provider, maxTurns int, configLoader *config.Loader, ui *adapters.PureTTY, debug bool, autoApprove bool) (*manager.Manager, error) {
	workDir := getWorkingDirectory()
	cfg := buildConfig(configLoader, maxTurns, workDir)

	// Apply debug flag to configuration
	applyDebugFlag(cfg, debug)

	// Create tool registry with simple tools (no dependencies)
	registry := tools.NewRegistry()

	// Register simple built-in tools (file I/O)
	registry.Register(tools.NewReadFileTool())
	registry.Register(tools.NewWriteFileTool())
	registry.Register(tools.NewListDirectoryTool())

	// Note: ExecuteCommandTool and GetContextTool are registered by Agent
	// as they require executor, validator, and context dependencies

	// Create approval handler based on auto-approve flag
	var approvalHandler security.ApprovalHandler
	if autoApprove {
		approvalHandler = func(req security.ApprovalRequest) security.ApprovalResponse {
			return security.ApprovalResponse{
				RequestID: req.ID,
				Approved:  true,
				Reason:    "auto-approved",
			}
		}
	} else {
		approvalHandler = func(req security.ApprovalRequest) security.ApprovalResponse {
			return ui.ShowApprovalDialog(req)
		}
	}

	// Create manager with options
	var opts []manager.ManagerOption
	opts = append(opts, manager.WithLLM(provider))
	opts = append(opts, manager.WithManagerToolRegistry(registry))
	opts = append(opts, manager.WithManagerApprovalHandler(approvalHandler))

	mgr, err := manager.NewManager(cfg, opts...)
	if err != nil {
		return nil, fmt.Errorf("create manager: %w", err)
	}

	return mgr, nil
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

// getWorkingDirectory returns the working directory for the conversation.
func getWorkingDirectory() string {
	workDir, err := os.Getwd()
	if err != nil {
		workDir = "."
	}
	if flagWorkDir != "" {
		workDir = flagWorkDir
	}
	return workDir
}

// initializeUI initializes the UI with conversation metadata.
func initializeUI(ui *adapters.PureTTY, conv *conversation.Conversation, provider llm.Provider) {
	taskMode := conv.GetTaskMode()
	ui.SetTaskMode(taskMode)

	providerName := provider.Name()
	modelName := flagModel
	ui.SetProviderInfo(providerName, modelName)

	tokenCount := int64(conv.GetTokenCount())
	ui.SetTokenCount(tokenCount)

	sessionID := conv.GetSessionID()
	ui.SetConversationID(sessionID)
}

// buildConfig builds the configuration from multiple sources.
func buildConfig(configLoader *config.Loader, maxTurns int, workDir string) *manager.Config {
	cfg := manager.DefaultConfig()
	cfg.WorkDir = workDir

	// Layer 1: Load from config file
	var fileCfg manager.Config
	if err := configLoader.Unmarshal(&fileCfg); err == nil {
		applyFileConfig(cfg, &fileCfg)
	}

	// Layer 2: Override with CLI flags
	applyCLIFlags(cfg, maxTurns)

	return cfg
}

// applyFileConfig applies configuration from file to the main config.
func applyFileConfig(cfg *manager.Config, fileCfg *manager.Config) {
	if fileCfg.Provider != "" {
		cfg.Provider = fileCfg.Provider
	}
	if fileCfg.Model != "" {
		cfg.Model = fileCfg.Model
	}
	if fileCfg.MaxTurns > 0 {
		cfg.MaxTurns = fileCfg.MaxTurns
	}
	if fileCfg.Timeout > 0 {
		cfg.Timeout = fileCfg.Timeout
	}
	if fileCfg.MaxTokens > 0 {
		cfg.MaxTokens = fileCfg.MaxTokens
	}
}

// applyCLIFlags applies CLI flags to the configuration.
func applyCLIFlags(cfg *manager.Config, maxTurns int) {
	if maxTurns > 0 {
		cfg.MaxTurns = maxTurns
	}
	if flagProvider != "" {
		cfg.Provider = flagProvider
	}
	if flagModel != "" {
		cfg.Model = flagModel
	}
}

// applyDebugFlag applies the debug flag to configuration.
func applyDebugFlag(cfg *manager.Config, debug bool) {
	if debug {
		cfg.Debug = true
		cfg.LogLevel = "debug"
		// Don't suppress INFO logs when debug is enabled
		// This allows debug logging to work properly
	}
}
