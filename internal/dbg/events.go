package dbg

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
	agentexec "github.com/dmytrogajewski/spin/internal/agent/executor"
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

var (
	ErrPromptCannotBeEmpty = errors.New("prompt cannot be empty")
	ErrTaskFailed          = errors.New("task failed")
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

// debugServices holds the services created during debug setup.
type debugServices struct {
	gitSvc   *gitpkg.Service
	shellSvc *shellpkg.Service
	mcpSvc   *mcppkg.Service
}

// createGitService creates a git service if enabled by config.
func createGitService(cfg *config.V2, workDir string, logger *slog.Logger) (*gitpkg.Service, error) {
	if !cfg.Protocol.EnableGit {
		return nil, nil
	}

	svc, err := gitpkg.NewService(true, workDir, logger)
	if err != nil {
		return nil, fmt.Errorf("create git service: %w", err)
	}

	return svc, nil
}

// createShellService creates a shell service if enabled by config.
func createShellService(cfg *config.V2, workDir string, logger *slog.Logger) (*shellpkg.Service, error) {
	if !cfg.Protocol.EnableShell {
		return nil, nil
	}

	svc, err := shellpkg.NewService(true, workDir, logger, cfg.Protocol.ShellTimeout)
	if err != nil {
		return nil, fmt.Errorf("create shell service: %w", err)
	}

	return svc, nil
}

// createMCPService creates an MCP service if enabled and configured.
func createMCPService(ctx context.Context, cfg *config.V2, logger *slog.Logger) *mcppkg.Service {
	if !cfg.Protocol.EnableMCP || len(cfg.Protocol.MCPServers) == 0 {
		return nil
	}

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
			logger.WarnContext(ctx, "failed to create MCP registry", "name", srv.Name, "err", err)

			continue
		}

		if regErr := registryManager.Register(registry); regErr != nil {
			logger.WarnContext(ctx, "failed to register MCP registry", "name", srv.Name, "err", regErr)

			continue
		}
	}

	for _, reg := range registryManager.All() {
		if err := reg.Initialize(ctx); err != nil {
			logger.WarnContext(ctx, "failed to initialize MCP registry", "name", reg.Name(), "err", err)
		}
	}

	return mcppkg.NewService(registryManager)
}

// initServices creates all required services based on configuration.
func initServices(ctx context.Context, cfg *config.V2, workDir string, logger *slog.Logger) (*debugServices, error) {
	gitSvc, err := createGitService(cfg, workDir, logger)
	if err != nil {
		return nil, err
	}

	shellSvc, err := createShellService(cfg, workDir, logger)
	if err != nil {
		if gitSvc != nil {
			gitSvc.Close()
		}

		return nil, err
	}

	mcpSvc := createMCPService(ctx, cfg, logger)

	return &debugServices{
		gitSvc:   gitSvc,
		shellSvc: shellSvc,
		mcpSvc:   mcpSvc,
	}, nil
}

// close closes all services.
func (ds *debugServices) close() {
	if ds.gitSvc != nil {
		ds.gitSvc.Close()
	}

	if ds.shellSvc != nil {
		ds.shellSvc.Close()
	}

	if ds.mcpSvc != nil {
		ds.mcpSvc.Close()
	}
}

// createBuiltinRuntime builds a minimal runtime for debug mode.
func createBuiltinRuntime(
	workDir string,
	emitter *events.EventEmitter,
	cfg *config.V2,
	svcs *debugServices,
	logger *slog.Logger,
) (*agentexec.BuiltinRuntime, error) {
	var storage session.Storage
	if cfg.Agent.SessionDir != "" {
		storage, _ = session.NewFileStorage(cfg.Agent.SessionDir)
	}

	approvalHandler := func(_ context.Context, req security.ApprovalRequest) security.ApprovalResponse {
		return security.ApprovalResponse{
			RequestID: req.ID,
			Approved:  true,
			Reason:    "debug mode auto-approve",
		}
	}

	executor, _ := agent.NewExecutor(workDir)
	validator := security.NewValidator()

	return agentexec.NewBuiltinRuntime(agentexec.BuiltinRuntimeConfig{
		WorkDir:         workDir,
		Emitter:         emitter,
		Storage:         storage,
		SessionID:       fmt.Sprintf("debug-%d", time.Now().UnixNano()),
		Executor:        agent.NewExecutorRuntimeAdapter(executor),
		Validator:       validator,
		ShellService:    svcs.shellSvc,
		GitService:      svcs.gitSvc,
		UI:              nil,
		ApprovalHandler: approvalHandler,
		Logger:          logger,
	})
}

// buildConversation creates and configures a conversation using the builder pattern.
func buildConversation(
	ctx context.Context,
	cfg *config.V2,
	workDir string,
	runtime *agentexec.BuiltinRuntime,
	emitter *events.EventEmitter,
	provider llm.Provider,
	svcs *debugServices,
) (*conversation.Conversation, error) {
	builder := conversation.NewBuilder(cfg, workDir, runtime, emitter, provider)

	if svcs.gitSvc != nil {
		builder = builder.WithGit(svcs.gitSvc)
	}

	if svcs.shellSvc != nil {
		builder = builder.WithShell(svcs.shellSvc)
	}

	if svcs.mcpSvc != nil {
		builder = builder.WithMCP(svcs.mcpSvc)
	}

	conv, err := builder.Build(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to build conversation: %w", err)
	}

	return conv, nil
}

// processEvents reads and logs events from the stream, returning any error.
func (el *EventLogger) processEvents(eventStream <-chan events.Event) error {
	for event := range eventStream {
		if el.shouldLog(event) {
			el.logEvent(event)
		}

		if event.Type == events.EventError {
			return fmt.Errorf("task failed: %v: %w", event.Data, ErrTaskFailed)
		}

		if event.Type == events.EventTurnComplete || event.Type == events.EventTurnFailed {
			break
		}
	}

	return nil
}

// Run executes a task with event logging enabled.
func (el *EventLogger) Run(ctx context.Context, prompt string) error {
	if prompt == "" {
		return ErrPromptCannotBeEmpty
	}

	cfg := config.DefaultV2()
	cfg.LLM.Provider = "mock"
	cfg.LLM.Model = "test-model"

	workDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}

	logger := slog.Default()

	svcs, err := initServices(ctx, cfg, workDir, logger)
	if err != nil {
		return err
	}
	defer svcs.close()

	emitter := events.NewEventEmitter(100)
	provider := llm.NewMockProvider("debug")

	runtime, err := createBuiltinRuntime(workDir, emitter, cfg, svcs, logger)
	if err != nil {
		return fmt.Errorf("create builtin runtime: %w", err)
	}

	conv, err := buildConversation(ctx, cfg, workDir, runtime, emitter, provider, svcs)
	if err != nil {
		return err
	}
	defer conv.Close()

	eventStream := conv.Stream()
	errChan := make(chan error, 1)

	go func() {
		errChan <- conv.RunTurn(ctx, prompt)
	}()

	if evtErr := el.processEvents(eventStream); evtErr != nil {
		return evtErr
	}

	select {
	case turnErr := <-errChan:
		if turnErr != nil {
			return fmt.Errorf("turn execution failed: %w", turnErr)
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
