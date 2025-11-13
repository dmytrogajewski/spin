package conversation

import (
	"context"
	"fmt"

	"github.com/dmytrogajewski/spin/internal/agent"
	"github.com/dmytrogajewski/spin/internal/events"
	gitpkg "github.com/dmytrogajewski/spin/internal/git"
	"github.com/dmytrogajewski/spin/internal/history"
	mcppkg "github.com/dmytrogajewski/spin/internal/mcp"
	"github.com/dmytrogajewski/spin/internal/message"
	shellpkg "github.com/dmytrogajewski/spin/internal/shell"
	"github.com/dmytrogajewski/spin/internal/task"
)

// Conversation represents an active conversation instance.
type Conversation struct {
	// Services (optional, can be nil)
	gitService   *gitpkg.Service
	shellService *shellpkg.Service
	mcpService   *mcppkg.Service

	// Core components
	agent     *agent.Agent
	history   *history.History
	emitter   *events.EventEmitter
	taskMode  string // Current task mode (regular, review, compact, planning)
	sessionID string // Session identifier for tracking and persistence
	workDir   string // Working directory for this conversation
}

// RunTurn executes a single turn in the conversation.
func (c *Conversation) RunTurn(ctx context.Context, input string) error {
	// Get conversation history for context
	historyMessages := c.history.MessagesForLLM()

	// Create task instance from task mode (use default if not set)
	var taskInstance task.Task
	var err error
	if c.taskMode == "" {
		taskInstance = task.DefaultTask()
	} else {
		taskInstance, err = task.NewTask(c.taskMode)
	}
	if err != nil {
		return fmt.Errorf("invalid task mode %q: %w", c.taskMode, err)
	}

	// Create agent request with task and history
	req := &agent.AgentRequest{
		Input:   input,
		Task:    taskInstance,
		History: historyMessages,
	}

	// Execute agent (user message will be in resp.Messages)
	resp, err := c.agent.Execute(ctx, req)
	if err != nil {
		// Add user message first (since it wasn't added before execution)
		if err := c.history.AddUserMessage(input); err != nil {
			return fmt.Errorf("failed to add user message: %w", err)
		}
		// Add error message to history so it's preserved
		errorMsg := message.Message{
			Role:    message.RoleAssistant,
			Content: fmt.Sprintf("Error: %v", err),
		}
		_ = c.history.AddMessage(errorMsg)
		return fmt.Errorf("agent execution failed: %w", err)
	}

	// Add all messages from the agent's execution to history
	// This includes: user input, assistant messages with tool calls, tool results, final assistant
	// This maintains proper OpenAI message format and ensures accurate token counting
	for _, msg := range resp.Messages {
		if err := c.history.AddMessage(msg); err != nil {
			return fmt.Errorf("failed to add message to history: %w", err)
		}
	}

	return nil
}

// validTaskModes defines the valid task modes
var validTaskModes = map[string]bool{
	"regular":  true,
	"review":   true,
	"compact":  true,
	"planning": true,
}

// SetTaskMode sets the task mode for the conversation.
func (c *Conversation) SetTaskMode(mode string) error {
	if !validTaskModes[mode] {
		return fmt.Errorf("invalid task mode: %s (must be one of: regular, review, compact, planning)", mode)
	}
	c.taskMode = mode

	// Emit mode switch event
	if c.emitter != nil {
		c.emitter.Emit(events.Event{
			Type: events.EventInfo,
			Data: events.SystemEventData{
				Message: fmt.Sprintf("Switched to %s mode", mode),
			},
		})
	}

	return nil
}

// GetTaskMode returns the current task mode for the conversation.
func (c *Conversation) GetTaskMode() string {
	return c.taskMode
}

// GetTokenCount returns the total number of tokens used in the conversation history.
func (c *Conversation) GetTokenCount() int {
	return c.history.TokenCount()
}

// GetSessionID returns the session identifier for this conversation.
func (c *Conversation) GetSessionID() string {
	return c.sessionID
}

// Stream returns the event stream for this conversation.
func (c *Conversation) Stream() <-chan events.Event {
	return c.emitter.Events()
}

// Close closes the conversation and cleans up resources.
// Note: Services (git, shell, mcp) are owned by the application layer
// and are NOT closed here - they can be shared across conversations.
func (c *Conversation) Close() error {
	// Close conversation-specific resources only
	// Services are managed by the application layer
	return nil
}
