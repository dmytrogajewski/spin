package main

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"time"

	"github.com/dmytrogajewski/spin/internal/agent"
	"github.com/dmytrogajewski/spin/internal/auth"
	"github.com/dmytrogajewski/spin/internal/config"
	"github.com/dmytrogajewski/spin/internal/conversation"
	"github.com/dmytrogajewski/spin/internal/events"
	gitpkg "github.com/dmytrogajewski/spin/internal/git"
	"github.com/dmytrogajewski/spin/internal/llm"
	"github.com/dmytrogajewski/spin/internal/llm/builder"
	mcppkg "github.com/dmytrogajewski/spin/internal/mcp"
	"github.com/dmytrogajewski/spin/internal/security"
	shellpkg "github.com/dmytrogajewski/spin/internal/shell"
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
	cmd.Flags().Bool("simple", false, "Use simple agent mode (no Git/Shell/MCP integrations)")

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
	simpleMode, _ := cmd.Flags().GetBool("simple")

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

	// Check if using simple mode
	if simpleMode {
		// Simple mode: use agent.Builder.Build() directly (no integrations)
		workDir := getWorkingDirectory()
		cfg := buildConfig(configLoader, 0, workDir)

		_, err := createSimpleAgent(ctx, provider, cfg)
		if err != nil {
			return fmt.Errorf("create simple agent: %w", err)
		}

		// TODO: Implement full simple execution flow
		return fmt.Errorf("simple mode execution not yet fully implemented - agent created successfully but execution flow pending")
	}

	// Create conversation using builder pattern
	conv, err := createConversationForExec(ctx, provider, configLoader, autoApprove, debugFlag)
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

// loadConfig loads configuration from file or defaults.
func loadConfig() (*config.LoaderV2, error) {
	configLoader := config.NewLoaderV2()

	// Load from explicit config file if provided
	if flagConfigFile != "" {
		if _, err := configLoader.LoadFromFile(flagConfigFile); err != nil {
			return nil, err
		}
	} else {
		// Try to load from default locations (ignore error if not found)
		_, _ = configLoader.Load()
	}

	return configLoader, nil
}

// createAuthManager creates an auth manager with platform-specific keystore.
func createAuthManager() *auth.Manager {
	keystore := auth.NewKeystore()
	return auth.NewManager(keystore)
}

// buildProvider creates an LLM provider from configuration.
func buildProvider(ctx context.Context, configLoader *config.LoaderV2, authMgr *auth.Manager) (llm.Provider, error) {
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

// createSimpleAgent creates a simple agent without conversation overhead using agent.Builder.Build().
// This is useful for simple scripting scenarios where Git/Shell/MCP integrations aren't needed.
func createSimpleAgent(ctx context.Context, provider llm.Provider, cfg *config.ConfigV2) (*agent.Agent, error) {
	emitter := events.NewEventEmitter(100)
	workDir := getWorkingDirectory()

	// Use agent.Builder.Build() for simple agent creation
	return agent.NewBuilder().
		WithUnifiedConfig(cfg).
		WithProvider(provider).
		WithWorkingDir(workDir).
		WithEmitter(emitter).
		Build()
}

// createConversationForExec creates a conversation configured for exec mode.
func createConversationForExec(ctx context.Context, provider llm.Provider, configLoader *config.LoaderV2, autoApprove bool, debug bool) (*conversation.Conversation, error) {
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

	// Create services based on configuration
	logger := slog.Default()

	var gitSvc *gitpkg.Service
	var shellSvc *shellpkg.Service
	var mcpSvc *mcppkg.Service
	var err error

	if cfg.Protocol.EnableGit {
		gitSvc, err = gitpkg.NewService(true, workDir, logger)
		if err != nil {
			return nil, fmt.Errorf("create git service: %w", err)
		}
	}

	if cfg.Protocol.EnableShell {
		shellSvc, err = shellpkg.NewService(true, workDir, logger, cfg.Protocol.ShellTimeout)
		if err != nil {
			if gitSvc != nil {
				gitSvc.Close()
			}
			return nil, fmt.Errorf("create shell service: %w", err)
		}
	}

	if cfg.Protocol.EnableMCP && len(cfg.Protocol.MCPServers) > 0 {
		mcpCfg := &mcppkg.Config{
			EnableMCP:  true,
			MCPServers: make([]mcppkg.MCPServerConfig, len(cfg.Protocol.MCPServers)),
		}
		for i, srv := range cfg.Protocol.MCPServers {
			mcpCfg.MCPServers[i] = mcppkg.MCPServerConfig{
				Name:    srv.Name,
				Command: srv.Command,
				Args:    srv.Args,
				Env:     srv.Env,
			}
		}
		var err error
		mcpSvc, err = mcppkg.NewService(mcpCfg, logger)
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
		WithToolRegistry(registry).
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
		// Builder.Build() already cleans up services on error
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
