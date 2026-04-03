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

	"github.com/dmytrogajewski/spin/internal/auth"
	"github.com/dmytrogajewski/spin/internal/config"
	"github.com/dmytrogajewski/spin/internal/conversation"
	"github.com/dmytrogajewski/spin/internal/llm"
	"github.com/dmytrogajewski/spin/internal/llm/builder"
	llmrecorder "github.com/dmytrogajewski/spin/internal/llm/recorder"
	"github.com/dmytrogajewski/spin/internal/tui"
	"github.com/dmytrogajewski/spin/internal/ui/adapters"
	spinterm "github.com/dmytrogajewski/spin/internal/ui/term"
)

// ErrNoPromptProvidedUseCommandLine is a sentinel error.
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
	cmd.Flags().String("format", formatText, "Output format (text, json)")
	cmd.Flags().Bool("no-stream", false, "Disable streaming output")
	cmd.Flags().Bool("exit-on-error", true, "Exit immediately on first error")
	cmd.Flags().Bool("debug", false, "Enable debug mode with detailed logging")
	cmd.Flags().String("record-fixture", "", "Record LLM responses to JSONL fixture file for test replay")

	return cmd
}

// runExec executes the exec mode using unified TUI logic but non-interactive.
func runExec(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	setupSignalHandling(cancel)

	// Parse prompt from args or stdin.
	prompt, err := parsePrompt(args, cmd.InOrStdin())
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

	// Wrap provider with recorder if --record-fixture is specified.
	if recordPath, _ := cmd.Flags().GetString("record-fixture"); recordPath != "" {
		rec, recErr := llmrecorder.New(provider, recordPath)
		if recErr != nil {
			return fmt.Errorf("create fixture recorder: %w", recErr)
		}

		provider = rec

		defer rec.Close()
	}

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

		duration, err = time.ParseDuration(timeout)
		if err != nil {
			return fmt.Errorf("invalid timeout: %w", err)
		}

		ctx, cancel = context.WithTimeout(ctx, duration)
		defer cancel()
	}

	// Create conversation using shared builder pattern.
	ui, uiErr := createExecUI(cmd.OutOrStdout())
	if uiErr != nil {
		return fmt.Errorf("create TUI: %w", uiErr)
	}

	var approvalHandler = createDenyHandler("exec mode requires --auto-approve for dangerous operations")
	if autoApprove {
		approvalHandler = createAutoApproveHandler()
	}

	const eventBufferSize = 100

	conv, err := createConversation(ctx, provider, cfg, conversationConfig{
		approvalHandler: approvalHandler,
		ui:              ui,
		sessionPrefix:   "exec",
		eventBufferSize: eventBufferSize,
	})
	if err != nil {
		return fmt.Errorf("create conversation: %w", err)
	}
	defer conv.Close(ctx)

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

// parsePrompt parses the prompt from command line args or the given reader.
// If no args are provided, it reads from r (typically [os.Stdin]).
func parsePrompt(args []string, r io.Reader) (string, error) {
	if len(args) > 0 {
		// Join all args as prompt.
		return args[0], nil
	}

	// Read from the provided reader.
	data, err := io.ReadAll(r)
	if err != nil {
		return "", fmt.Errorf("read stdin: %w", err)
	}

	prompt := string(data)
	if prompt == "" {
		return "", ErrNoPromptProvidedUseCommandLine
	}

	return prompt, nil
}

// createExecUI creates the UI adapter for exec mode.
// The out writer is used for all output; when nil it defaults to [os.Stdout].
func createExecUI(out io.Writer) (*adapters.PureTTY, error) {
	if out == nil {
		out = os.Stdout
	}

	opts := []adapters.PureTTYOption{adapters.WithExecMode()}

	// Check if the writer is a real terminal; if not, use a mock TTY.
	if f, ok := out.(*os.File); !ok || !termx.IsTerminal(spinterm.SafeFd(f.Fd())) {
		const (
			mockTermWidth  = 120
			mockTermHeight = 30
		)

		mockTty := &mockTTY{width: mockTermWidth, height: mockTermHeight}
		opts = append(opts, adapters.WithTTY(mockTty))
	}

	return adapters.NewPureTTY(out, opts...)
}

// mockTTY implements term.TerminalController for non-terminal environments.
type mockTTY struct {
	width, height int
}

func (m *mockTTY) Enter() error              { return nil }
func (m *mockTTY) Exit() error               { return nil }
func (m *mockTTY) Size() (width, height int) { return m.width, m.height }
func (m *mockTTY) OnResize(_ func(w, h int)) {}

// executePromptWithTUI executes a prompt non-interactively but shows TUI interface.
func executePromptWithTUI(ctx context.Context, conv *conversation.Conversation, prompt, _ string, _, exitOnError bool) error {
	pureTTY, err := createExecUI(nil)
	if err != nil {
		return fmt.Errorf("create TUI: %w", err)
	}

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

	eventsDone := startEventLoop(ctx, conv.Stream(), mapper, pureTTY, conv)

	errChan := make(chan error, 1)

	go func() {
		errChan <- conv.RunTurn(ctx, prompt)
	}()

	select {
	case <-ctx.Done():
		return fmt.Errorf("execution canceled: %w", ctx.Err())

	case err = <-errChan:
		// Close the emitter to flush all remaining events through the event loop,
		// then wait for the event loop to finish processing before stopping the stream.
		conv.GetEmitter().Close()
		<-eventsDone
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
