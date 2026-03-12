package debug

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"time"

	"github.com/dmytrogajewski/spin/internal/agent"
	"github.com/dmytrogajewski/spin/internal/agent/runtime"
	"github.com/dmytrogajewski/spin/internal/config"
	"github.com/dmytrogajewski/spin/internal/conversation"
	"github.com/dmytrogajewski/spin/internal/events"
	gitpkg "github.com/dmytrogajewski/spin/internal/git"
	"github.com/dmytrogajewski/spin/internal/llm"
	mcppkg "github.com/dmytrogajewski/spin/internal/mcp"
	"github.com/dmytrogajewski/spin/internal/security"
	"github.com/dmytrogajewski/spin/internal/session"
	shellpkg "github.com/dmytrogajewski/spin/internal/shell"
)

// EventLogger captures and logs all core events for debugging.
type EventLogger struct {
	format string
	filter map[string]bool
	writer io.Writer
}

// NewEventLogger creates a new event logger.
//
// format: "text" or "json"
// filter: list of event types to log (empty = log all)
func NewEventLogger(format string, filter []string) *EventLogger {
	filterMap := make(map[string]bool)
	for _, f := range filter {
		filterMap[f] = true
	}

	return &EventLogger{
		format: format,
		filter: filterMap,
		writer: os.Stderr,
	}
}

// Run executes a task with event logging enabled.
func (el *EventLogger) Run(ctx context.Context, prompt string) error {
	if prompt == "" {
		return errors.New("prompt cannot be empty")
	}

	// Create conversation with default config using builder pattern.
	cfg := config.DefaultConfigV2()
	// Set required fields for validation.
	cfg.LLM.Provider = "mock"
	cfg.LLM.Model = "test-model"

	workDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}

	logger := slog.Default()

	// Create services based on configuration.
	var (
		gitSvc   *gitpkg.Service
		shellSvc *shellpkg.Service
		mcpSvc   *mcppkg.Service
	)

	if cfg.Protocol.EnableGit {
		gitSvc, err = gitpkg.NewService(true, workDir, logger)
		if err != nil {
			return fmt.Errorf("create git service: %w", err)
		}
		defer gitSvc.Close()
	}

	if cfg.Protocol.EnableShell {
		shellSvc, err = shellpkg.NewService(true, workDir, logger, cfg.Protocol.ShellTimeout)
		if err != nil {
			return fmt.Errorf("create shell service: %w", err)
		}
		defer shellSvc.Close()
	}

	if cfg.Protocol.EnableMCP && len(cfg.Protocol.MCPServers) > 0 {
		registryManager := mcppkg.NewDefaultRegistryManager(logger)
		for _, srv := range cfg.Protocol.MCPServers {
			registry, err := mcppkg.NewLocalRegistry(mcppkg.LocalRegistryConfig{
				Name:    srv.Name,
				Command: srv.Command,
				Args:    srv.Args,
				Env:     srv.Env,
				Logger:  logger,
			})
			if err != nil {
				logger.Warn("failed to create MCP registry", "name", srv.Name, "err", err)

				continue
			}

			err = registryManager.Register(registry)
			if err != nil {
				logger.Warn("failed to register MCP registry", "name", srv.Name, "err", err)

				continue
			}
		}

		for _, reg := range registryManager.All() {
			err := reg.Initialize(ctx)
			if err != nil {
				logger.Warn("failed to initialize MCP registry", "name", reg.Name(), "err", err)
			}
		}

		mcpSvc = mcppkg.NewService(registryManager)
		defer mcpSvc.Close()
	}

	// Create required dependencies for conversation.
	emitter := events.NewEventEmitter(100)
	provider := llm.NewMockProvider("debug")

	// Create builtin runtime for debug mode.
	var storage session.Storage
	if cfg.Agent.SessionDir != "" {
		storage, _ = session.NewFileStorage(cfg.Agent.SessionDir)
	}

	// Create auto-approve handler for debug (no approval needed for event logging).
	approvalHandler := func(ctx context.Context, req security.ApprovalRequest) security.ApprovalResponse {
		return security.ApprovalResponse{
			RequestID: req.ID,
			Approved:  true,
			Reason:    "debug mode auto-approve",
		}
	}

	// Build a minimal executor for debug.
	executor, _ := agent.NewExecutor(workDir)
	validator := security.NewValidator()

	builtinRuntime, err := runtime.NewBuiltinRuntime(runtime.BuiltinRuntimeConfig{
		WorkDir:         workDir,
		Emitter:         emitter,
		Storage:         storage,
		SessionID:       fmt.Sprintf("debug-%d", time.Now().UnixNano()),
		Executor:        agent.NewExecutorRuntimeAdapter(executor),
		Validator:       validator,
		ShellService:    shellSvc,
		GitService:      gitSvc,
		UI:              nil, // No UI needed for debug event logging.
		ApprovalHandler: approvalHandler,
		Logger:          logger,
	})
	if err != nil {
		return fmt.Errorf("create builtin runtime: %w", err)
	}

	// Build conversation with required dependencies.
	builder := conversation.NewBuilder(cfg, workDir, builtinRuntime, emitter, provider)

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
		return fmt.Errorf("failed to build conversation: %w", err)
	}
	defer conv.Close()

	// Get event stream.
	eventStream := conv.Stream()

	// Start turn in goroutine.
	errChan := make(chan error, 1)

	go func() {
		errChan <- conv.RunTurn(ctx, prompt)
	}()

	// Log all events.
	for event := range eventStream {
		if el.shouldLog(event) {
			el.logEvent(event)
		}

		// Check for errors.
		if event.Type == events.EventError {
			return fmt.Errorf("task failed: %v", event.Data)
		}

		// Stop on turn complete or failed.
		if event.Type == events.EventTurnComplete || event.Type == events.EventTurnFailed {
			break
		}
	}

	// Check for turn execution error.
	select {
	case err := <-errChan:
		if err != nil {
			return fmt.Errorf("turn execution failed: %w", err)
		}
	default:
	}

	return nil
}

// shouldLog checks if an event should be logged based on the filter.
func (el *EventLogger) shouldLog(event events.Event) bool {
	if len(el.filter) == 0 {
		return true // No filter = log all.
	}

	return el.filter[event.Type.String()]
}

// logEvent prints an event to the configured writer.
func (el *EventLogger) logEvent(event events.Event) {
	timestamp := time.Now().Format("2006-01-02 15:04:05")

	if el.format == "json" {
		el.logEventJSON(timestamp, event)
	} else {
		el.logEventText(timestamp, event)
	}
}

// EventLogOutput represents a structured event log entry.
type EventLogOutput struct {
	Timestamp string           `json:"timestamp"`
	Type      events.EventType `json:"type"`
	Data      json.RawMessage  `json:"data"`
}

// logEventJSON logs event in JSON format.
func (el *EventLogger) logEventJSON(timestamp string, event events.Event) {
	data, err := json.Marshal(event.Data)
	if err != nil {
		data = []byte("{}")
	}

	output := EventLogOutput{
		Timestamp: timestamp,
		Type:      event.Type,
		Data:      json.RawMessage(data),
	}

	encoded, err := json.Marshal(output)
	if err != nil {
		fmt.Fprintf(el.writer, `{"error": "failed to encode event"}`+"\n")

		return
	}

	fmt.Fprintf(el.writer, "%s\n", encoded)
}

// logEventText logs event in human-readable text format.
func (el *EventLogger) logEventText(timestamp string, event events.Event) {
	dataStr := fmt.Sprintf("%v", event.Data)
	fmt.Fprintf(el.writer, "[%s] %s %s\n", timestamp, event.Type, dataStr)
}
