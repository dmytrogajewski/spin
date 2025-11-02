package debug

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"time"

	"github.com/dmytrogajewski/spin/internal/config"
	"github.com/dmytrogajewski/spin/internal/conversation"
	"github.com/dmytrogajewski/spin/internal/events"
	gitpkg "github.com/dmytrogajewski/spin/internal/git"
	mcppkg "github.com/dmytrogajewski/spin/internal/mcp"
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
		return fmt.Errorf("prompt cannot be empty")
	}

	// Create conversation with default config using builder pattern
	cfg := config.DefaultConfig()
	// Set required fields for validation
	cfg.Provider = "mock"
	cfg.Model = "test-model"

	workDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}

	logger := slog.Default()

	// Create services based on configuration
	var gitSvc *gitpkg.Service
	var shellSvc *shellpkg.Service
	var mcpSvc *mcppkg.Service

	if cfg.EnableGit {
		gitSvc, err = gitpkg.NewService(true, workDir, logger)
		if err != nil {
			return fmt.Errorf("create git service: %w", err)
		}
		defer gitSvc.Close()
	}

	if cfg.EnableShell {
		shellSvc, err = shellpkg.NewService(true, workDir, logger, cfg.ShellTimeout)
		if err != nil {
			return fmt.Errorf("create shell service: %w", err)
		}
		defer shellSvc.Close()
	}

	if cfg.EnableMCP && len(cfg.MCPServers) > 0 {
		mcpCfg := &mcppkg.Config{
			EnableMCP:  true,
			MCPServers: make([]mcppkg.MCPServerConfig, len(cfg.MCPServers)),
		}
		for i, srv := range cfg.MCPServers {
			mcpCfg.MCPServers[i] = mcppkg.MCPServerConfig{
				Name:    srv.Name,
				Command: srv.Command,
				Args:    srv.Args,
				Env:     srv.Env,
			}
		}
		mcpSvc, err = mcppkg.NewService(mcpCfg, logger)
		if err != nil {
			return fmt.Errorf("create mcp service: %w", err)
		}
		defer mcpSvc.Close()
	}

	// Build conversation with services
	builder := conversation.NewBuilder(cfg, workDir)

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

	// Get event stream
	eventStream := conv.Stream()

	// Start turn in goroutine
	errChan := make(chan error, 1)
	go func() {
		errChan <- conv.RunTurn(ctx, prompt)
	}()

	// Log all events
	for event := range eventStream {
		if el.shouldLog(event) {
			el.logEvent(event)
		}

		// Check for errors
		if event.Type == events.EventError {
			return fmt.Errorf("task failed: %v", event.Data)
		}

		// Stop on turn complete or failed
		if event.Type == events.EventTurnComplete || event.Type == events.EventTurnFailed {
			break
		}
	}

	// Check for turn execution error
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
		return true // No filter = log all
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
