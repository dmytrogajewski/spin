package debug

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/dmytrogajewski/spin/internal/core"
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

	// Create core manager with default config
	cfg := core.DefaultConfig()
	mgr, err := core.NewManager(cfg)
	if err != nil {
		return fmt.Errorf("failed to create manager: %w", err)
	}

	// Start conversation (empty workDir uses default from config)
	conv, err := mgr.NewConversation(ctx, "")
	if err != nil {
		return fmt.Errorf("failed to create conversation: %w", err)
	}
	defer conv.Stop(ctx)

	// Get event stream
	events := conv.Stream()

	// Start turn in goroutine
	errChan := make(chan error, 1)
	go func() {
		errChan <- conv.RunTurn(ctx, prompt)
	}()

	// Log all events
	for event := range events {
		if el.shouldLog(event) {
			el.logEvent(event)
		}

		// Check for errors
		if event.Type == core.EventError {
			return fmt.Errorf("task failed: %v", event.Data)
		}

		// Stop on turn complete or failed
		if event.Type == core.EventTurnComplete || event.Type == core.EventTurnFailed {
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
func (el *EventLogger) shouldLog(event core.Event) bool {
	if len(el.filter) == 0 {
		return true // No filter = log all
	}
	return el.filter[event.Type.String()]
}

// logEvent prints an event to the configured writer.
func (el *EventLogger) logEvent(event core.Event) {
	timestamp := time.Now().Format("2006-01-02 15:04:05")

	if el.format == "json" {
		el.logEventJSON(timestamp, event)
	} else {
		el.logEventText(timestamp, event)
	}
}

// logEventJSON logs event in JSON format.
func (el *EventLogger) logEventJSON(timestamp string, event core.Event) {
	data, err := json.Marshal(event.Data)
	if err != nil {
		data = []byte("{}")
	}

	output := map[string]interface{}{
		"timestamp": timestamp,
		"type":      event.Type,
		"data":      json.RawMessage(data),
	}

	encoded, err := json.Marshal(output)
	if err != nil {
		fmt.Fprintf(el.writer, `{"error": "failed to encode event"}`+"\n")
		return
	}

	fmt.Fprintf(el.writer, "%s\n", encoded)
}

// logEventText logs event in human-readable text format.
func (el *EventLogger) logEventText(timestamp string, event core.Event) {
	dataStr := fmt.Sprintf("%v", event.Data)
	fmt.Fprintf(el.writer, "[%s] %s %s\n", timestamp, event.Type, dataStr)
}
