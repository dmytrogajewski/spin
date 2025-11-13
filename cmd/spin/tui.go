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
	"github.com/dmytrogajewski/spin/internal/git"
	"github.com/dmytrogajewski/spin/internal/llm"
	"github.com/dmytrogajewski/spin/internal/mcp"
	"github.com/dmytrogajewski/spin/internal/security"
	"github.com/dmytrogajewski/spin/internal/shell"
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

	// Determine the actual model being used (flag takes precedence over config)
	currentModel := flagModel
	if currentModel == "" {
		currentModel = configLoader.GetString("model")
	}

	// Set max tokens for context percentage display
	// Try to get actual context window from provider's models
	maxTokens := int64(128000) // Default fallback for modern models
	if models, err := provider.Models(ctx); err == nil && len(models) > 0 {
		// Find the current model
		for _, m := range models {
			if m.ID == currentModel {
				// openai.Model doesn't have ContextSize field
				// Use a default or fetch from model details if needed
				// For now, keep the default maxTokens
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

	conv, err := createConversationForTUI(ctx, provider, maxTurns, configLoader, ui, debugFlag, autoApprove)
	if err != nil {
		return fmt.Errorf("create conversation: %w", err)
	}
	defer conv.Close()

	// Initialize UI with conversation metadata
	initializeUI(ui, conv, provider, currentModel)

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

// createConversationForTUI creates a conversation configured for TUI mode using the new builder pattern.
func createConversationForTUI(ctx context.Context, provider llm.Provider, maxTurns int, configLoader *config.LoaderV2, ui *adapters.PureTTY, debug bool, autoApprove bool) (*conversation.Conversation, error) {
	workDir := getWorkingDirectory()
	cfg := buildConfig(configLoader, maxTurns, workDir)

	applyDebugFlag(cfg, debug)

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

	// Create services based on configuration
	logger := slog.Default()

	var gitSvc *git.Service
	var shellSvc *shell.Service
	var mcpSvc *mcp.Service

	if cfg.Protocol.EnableGit {
		var err error
		gitSvc, err = git.NewService(true, workDir, logger)
		if err != nil {
			return nil, fmt.Errorf("create git service: %w", err)
		}
	}

	if cfg.Protocol.EnableShell {
		var err error
		shellSvc, err = shell.NewService(true, workDir, logger, cfg.Protocol.ShellTimeout)
		if err != nil {
			if gitSvc != nil {
				gitSvc.Close()
			}
			return nil, fmt.Errorf("create shell service: %w", err)
		}
	}

	if cfg.Protocol.EnableMCP && len(cfg.Protocol.MCPServers) > 0 {
		mcpCfg := &mcp.Config{
			EnableMCP:  true,
			MCPServers: make([]mcp.MCPServerConfig, len(cfg.Protocol.MCPServers)),
		}
		for i, srv := range cfg.Protocol.MCPServers {
			mcpCfg.MCPServers[i] = mcp.MCPServerConfig{
				Name:    srv.Name,
				Command: srv.Command,
				Args:    srv.Args,
				Env:     srv.Env,
			}
		}
		var err error
		mcpSvc, err = mcp.NewService(mcpCfg, logger)
		if err != nil {
			if gitSvc != nil {
				gitSvc.Close()
			}
			if shellSvc != nil {
				shellSvc.Close()
			}
			return nil, fmt.Errorf("create mcp service: %w", err)
		}
	}

	// Build conversation with services
	builder := conversation.NewBuilder(cfg, workDir).
		WithLLM(provider).
		WithApprovalHandler(approvalHandler)

	if gitSvc != nil {
		builder = builder.WithGit(gitSvc)
	}
	if shellSvc != nil {
		builder = builder.WithShell(shellSvc)
	}
	if mcpSvc != nil {
		builder = builder.WithMCP(mcpSvc)
	}

	conv, err := builder.Build(ctx)
	if err != nil {
		// Clean up services on error
		if gitSvc != nil {
			gitSvc.Close()
		}
		if shellSvc != nil {
			shellSvc.Close()
		}
		if mcpSvc != nil {
			mcpSvc.Close()
		}
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

// buildConfig builds the configuration from multiple sources.
func buildConfig(configLoader *config.LoaderV2, maxTurns int, workDir string) *config.ConfigV2 {
	cfg := config.DefaultConfigV2()
	cfg.Agent.WorkDir = workDir

	// Layer 1: Load from config file
	var fileCfg config.ConfigV2
	if err := configLoader.Unmarshal(&fileCfg); err == nil {
		applyFileConfig(cfg, &fileCfg)
	}

	// Layer 2: Override with CLI flags
	applyCLIFlags(cfg, maxTurns)

	return cfg
}

// applyFileConfig applies configuration from file to the main config.
func applyFileConfig(cfg *config.ConfigV2, fileCfg *config.ConfigV2) {
	if fileCfg.LLM.Provider != "" {
		cfg.LLM.Provider = fileCfg.LLM.Provider
	}
	if fileCfg.LLM.Model != "" {
		cfg.LLM.Model = fileCfg.LLM.Model
	}
	if fileCfg.Agent.MaxTurns > 0 {
		cfg.Agent.MaxTurns = fileCfg.Agent.MaxTurns
	}
	if fileCfg.Agent.Timeout > 0 {
		cfg.Agent.Timeout = fileCfg.Agent.Timeout
	}
	if fileCfg.LLM.MaxTokens > 0 {
		cfg.LLM.MaxTokens = fileCfg.LLM.MaxTokens
	}
}

// applyCLIFlags applies CLI flags to the configuration.
func applyCLIFlags(cfg *config.ConfigV2, maxTurns int) {
	if maxTurns > 0 {
		cfg.Agent.MaxTurns = maxTurns
	}
	if flagProvider != "" {
		cfg.LLM.Provider = flagProvider
	}
	if flagModel != "" {
		cfg.LLM.Model = flagModel
	}
}

// applyDebugFlag applies the debug flag to configuration.
func applyDebugFlag(cfg *config.ConfigV2, debug bool) {
	if debug {
		cfg.Agent.Debug = true
		cfg.Agent.LogLevel = "debug"
		// Don't suppress INFO logs when debug is enabled
		// This allows debug logging to work properly
	}
}
