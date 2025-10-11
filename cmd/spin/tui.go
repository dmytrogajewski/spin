package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/dmytrogajewski/spin/internal/core"
	"github.com/dmytrogajewski/spin/internal/llm"
	"github.com/dmytrogajewski/spin/internal/tools"
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

Examples:
  spin tui
  spin tui --model llama3.1
  spin tui --provider anthropic --model claude-3-5-sonnet-20241022`,
		RunE: runTUI,
	}

	// TUI-specific flags
	cmd.Flags().Int("max-turns", 50, "Maximum conversation turns")
	cmd.Flags().Bool("require-approval", false, "Require user approval for dangerous commands")

	return cmd
}

// runTUI executes the TUI mode.
func runTUI(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Setup signal handling
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigCh
		cancel()
	}()

	// Load configuration
	configLoader, err := loadConfig()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	// Create auth manager
	authMgr := createAuthManager()

	// Build LLM provider
	provider, err := buildProvider(ctx, configLoader, authMgr)
	if err != nil {
		return fmt.Errorf("create provider: %w", err)
	}
	defer provider.Close()

	// Get flags
	maxTurns, _ := cmd.Flags().GetInt("max-turns")
	requireApproval, _ := cmd.Flags().GetBool("require-approval")

	// Create core manager
	mgr, err := createManagerForTUI(provider, maxTurns, requireApproval)
	if err != nil {
		return fmt.Errorf("create manager: %w", err)
	}

	// Initialize TUI
	ui, err := adapters.NewPureTTY(os.Stdout)
	if err != nil {
		return fmt.Errorf("create TUI: %w", err)
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

	// Get working directory
	workDir, err := os.Getwd()
	if err != nil {
		workDir = "."
	}
	if flagWorkDir != "" {
		workDir = flagWorkDir
	}

	// Create conversation
	conv, err := mgr.NewConversation(ctx, workDir)
	if err != nil {
		return fmt.Errorf("create conversation: %w", err)
	}

	// Create event mapper
	mapper := core.NewTUIMapper(ui)
	defer mapper.Close()

	// Start streaming channel
	streamCh := mapper.StartStreaming()
	go func() {
		ui.PrintChunks(ctx, streamCh)
	}()

	// Print welcome message
	ui.PrintLine("Spin TUI - AI Coding Assistant")
	ui.PrintLine("Type your prompt and press Enter. Press Ctrl-D to exit.\n")

	// Subscribe to conversation events
	events := conv.Stream()

	// Start event processing loop
	eventDone := make(chan struct{})
	go func() {
		defer close(eventDone)
		for {
			select {
			case <-ctx.Done():
				return
			case event, ok := <-events:
				if !ok {
					return
				}
				if err := mapper.MapEvent(event); err != nil {
					ui.PrintLine(fmt.Sprintf("⚠ Mapper error: %v", err))
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

			// Submit prompt to conversation
			turnCtx, turnCancel := context.WithCancel(ctx)
			defer turnCancel()

			// Send message and handle errors
			err := conv.RunTurn(turnCtx, line)
			if err != nil {
				ui.PrintLine(fmt.Sprintf("✗ Error: %v\n", err))
			}
		}
	}
}

// createManagerForTUI creates a core.Manager configured for TUI mode.
func createManagerForTUI(provider llm.Provider, maxTurns int, requireApproval bool) (*core.Manager, error) {
	// Get current working directory
	workDir, err := os.Getwd()
	if err != nil {
		workDir = "."
	}

	// Override with flag if provided
	if flagWorkDir != "" {
		workDir = flagWorkDir
	}

	// Create core configuration with defaults
	cfg := core.DefaultConfig()
	cfg.MaxTurns = maxTurns
	cfg.WorkDir = workDir
	cfg.Provider = flagProvider
	cfg.Model = flagModel

	// Create tool registry with simple tools (no dependencies)
	registry := tools.NewRegistry()

	// Register simple built-in tools (file I/O)
	registry.Register(tools.NewReadFileTool())
	registry.Register(tools.NewWriteFileTool())
	registry.Register(tools.NewListDirectoryTool())

	// Note: ExecuteCommandTool and GetContextTool are registered by Agent
	// as they require executor, validator, and context dependencies

	// Create manager with options
	mgr, err := core.NewManager(cfg,
		core.WithLLM(provider),
		core.WithManagerToolRegistry(registry),
	)
	if err != nil {
		return nil, fmt.Errorf("create manager: %w", err)
	}

	return mgr, nil
}
