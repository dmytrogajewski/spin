package exec

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/dmytrogajewski/spin/internal/core"
	"github.com/dmytrogajewski/spin/internal/llm"
)

// RunWithProvider executes the task with a provided LLM provider.
// This is the main entry point for exec mode with real providers.
func RunWithProvider(ctx context.Context, args *ExecArgs, provider llm.Provider) error {
	// Create core config with defaults
	coreConfig := core.DefaultConfig()

	// Set dummy provider/model to pass validation (actual provider passed via WithLLM)
	coreConfig.Provider = "provided"
	coreConfig.Model = "provided"

	// Override with command-line settings
	if args.AutoApprove {
		// --auto-approve disables command validation entirely
		// This allows dangerous commands to run (USE WITH CAUTION)
		coreConfig.AllowedCommands = []string{"*"} // Allow all commands
		slog.Warn("auto-approve enabled - all commands will execute without validation")
	}

	// Create manager with provided provider
	mgr, err := core.NewManager(coreConfig, core.WithLLM(provider))
	if err != nil {
		return fmt.Errorf("create manager: %w", err)
	}

	// Get working directory
	workDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("get working directory: %w", err)
	}

	// Create conversation
	conv, err := mgr.NewConversation(ctx, workDir)
	if err != nil {
		return fmt.Errorf("create conversation: %w", err)
	}
	defer conv.Stop(ctx)

	// Execute turn and stream output
	return executeTurn(ctx, conv, args.Prompt, args.AutoApprove)
}

// runTask executes the task using the core module (legacy, uses mock provider).
// Deprecated: Use RunWithProvider instead.
func runTask(ctx context.Context, args *ExecArgs) error {
	// Create mock provider for backward compatibility
	provider := llm.NewMockProvider("default")
	return RunWithProvider(ctx, args, provider)
}

// executeTurn runs a conversation turn and streams output to stdout.
func executeTurn(ctx context.Context, conv *core.Conversation, prompt string, autoApprove bool) error {
	// Run turn in background
	errChan := make(chan error, 1)
	go func() {
		errChan <- conv.RunTurn(ctx, prompt)
	}()

	// Create audit logger for approval decisions
	auditLogger := slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	// Stream events
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()

		case err := <-errChan:
			if err != nil {
				return err
			}
			// Turn completed successfully
			return nil

		case event, ok := <-conv.Stream():
			if !ok {
				// Stream closed
				return nil
			}

			// Handle events
			if err := handleEvent(event, auditLogger, autoApprove); err != nil {
				return err
			}
		}
	}
}

// handleEvent processes a single event.
func handleEvent(event core.Event, auditLogger *slog.Logger, autoApprove bool) error {
	switch event.Type {
	case core.EventContentDelta:
		// Stream content delta to stdout
		// Event.Data can be either *ContentDeltaData or map[string]interface{}
		if data, ok := event.Data.(*core.ContentDeltaData); ok {
			fmt.Print(data.Content)
		} else if dataMap, ok := event.Data.(map[string]interface{}); ok {
			if content, ok := dataMap["content"].(string); ok {
				fmt.Print(content)
			}
		}

	case core.EventCommandApproval:
		// In exec mode:
		// - If --auto-approve: this shouldn't fire (AllowedCommands = ["*"])
		// - Otherwise: deny and return error

		if !autoApprove {
			// Log the denial decision
			auditLogger.Info("command approval request denied in exec mode",
				"event_type", event.Type.String(),
				"reason", "exec mode requires --auto-approve flag for dangerous commands",
			)

			return fmt.Errorf("command requires approval (use --auto-approve to allow dangerous operations): %v", event.Data)
		}

	case core.EventError:
		// Error occurred
		if err, ok := event.Data.(error); ok {
			return err
		}
		return fmt.Errorf("unknown error: %v", event.Data)

	case core.EventTurnComplete:
		// Turn completed - add newline
		fmt.Println()

	default:
		// Log other events for debugging
		slog.Debug("event received", "type", event.Type.String())
	}

	return nil
}
