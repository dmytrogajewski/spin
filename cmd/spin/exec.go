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
	"github.com/dmytrogajewski/spin/internal/manager"
	"github.com/dmytrogajewski/spin/internal/security"
	"github.com/dmytrogajewski/spin/internal/tools"
	"github.com/dmytrogajewski/spin/internal/tui"
	"github.com/dmytrogajewski/spin/internal/ui/adapters"
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

	// Get exec-specific flags
	autoApprove, _ := cmd.Flags().GetBool("auto-approve")
	timeout, _ := cmd.Flags().GetString("timeout")
	format, _ := cmd.Flags().GetString("format")
	noStream, _ := cmd.Flags().GetBool("no-stream")
	exitOnError, _ := cmd.Flags().GetBool("exit-on-error")
	debugFlag, _ := cmd.Flags().GetBool("debug")

	// Enable debug logging if requested
	if debugFlag {
		slog.SetLogLoggerLevel(slog.LevelDebug)
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

	// Create manager using same logic as TUI
	mgr, err := createManagerForExec(provider, configLoader, autoApprove, debugFlag)
	if err != nil {
		return fmt.Errorf("create manager: %w", err)
	}

	workDir := getWorkingDirectory()
	conv, err := mgr.NewConversation(ctx, workDir)
	if err != nil {
		return fmt.Errorf("create conversation: %w", err)
	}
	defer mgr.Close()

	// Execute the prompt non-interactively with TUI display
	err = executePromptWithTUI(ctx, conv, prompt, format, noStream, exitOnError)

	// Explicitly exit after execution completes
	if err != nil {
		return err
	}
	return nil
}

// loadConfig loads configuration from file or defaults.
func loadConfig() (*config.Loader, error) {
	configLoader := config.NewLoader()

	// Load from explicit config file if provided
	if flagConfigFile != "" {
		if err := configLoader.LoadFromFile(flagConfigFile); err != nil {
			return nil, err
		}
	} else {
		// Try to load from default locations (ignore error if not found)
		_ = configLoader.Load("")
	}

	return configLoader, nil
}

// createAuthManager creates an auth manager with platform-specific keystore.
func createAuthManager() *auth.Manager {
	keystore := auth.NewKeystore()
	return auth.NewManager(keystore)
}

// buildProvider creates an LLM provider from configuration.
func buildProvider(ctx context.Context, configLoader *config.Loader, authMgr *auth.Manager) (llm.Provider, error) {
	// Create builder
	b := builder.NewBuilder(configLoader, authMgr)

	// Build provider with flags as overrides (only if flags are set)
	cfg := builder.Config{}
	if flagProvider != "" {
		cfg.Provider = flagProvider
	}
	if flagModel != "" {
		cfg.Model = flagModel
	}
	// Additional flags can be added here in the future
	// BaseURL:  flagBaseURL,
	// KeyName:  flagKeyName,

	return b.Build(ctx, cfg)
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

// createManagerForExec creates a core.Manager configured for exec mode.
func createManagerForExec(provider llm.Provider, configLoader *config.Loader, autoApprove bool, debug bool) (*manager.Manager, error) {
	workDir := getWorkingDirectory()
	cfg := buildConfig(configLoader, 0, workDir) // No max turns limit for exec

	// Apply debug flag to configuration
	applyDebugFlag(cfg, debug)

	// Create tool registry with same tools as TUI
	registry := tools.NewRegistry()
	registry.Register(tools.NewReadFileTool())
	registry.Register(tools.NewWriteFileTool())
	registry.Register(tools.NewListDirectoryTool())

	// Create approval handler for exec mode
	var approvalHandler func(security.ApprovalRequest) security.ApprovalResponse
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
			// In exec mode without auto-approve, deny dangerous commands
			return security.ApprovalResponse{
				RequestID: req.ID,
				Approved:  false,
				Reason:    "exec mode requires --auto-approve for dangerous operations",
			}
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
			}
		}
	}()

	// Wait for completion
	select {
	case <-ctx.Done():
		return ctx.Err()

	case err := <-errChan:
		// Stop streaming to close the channel
		mapper.StopStreaming()

		// Wait for streaming to complete
		<-streamDone

		// EventTurnComplete is now emitted after all post-execution processing
		// (including ACE bullet generation), so we don't need to wait here

		if err != nil {
			if exitOnError {
				return err
			}
			ui.PrintLine(fmt.Sprintf("✗ Error: %v\n", err))
		}

		return nil
	}
}
