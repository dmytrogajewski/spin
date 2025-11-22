package conversation

import (
	"context"
	"fmt"
	"sync"

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
	agent    *agent.Agent
	history  *history.History
	emitter  *events.EventEmitter
	taskMode string // Current task mode (regular, review, compact, planning)
	id       string // Unified conversation ID (for both session and protocol)
	workDir  string // Working directory for this conversation

	// Protocol-specific fields (optional, for protocol use)
	turnID     string             // Current turn ID
	cancel     context.CancelFunc // Cancellation context
	protocolMu sync.RWMutex       // Protects protocol fields (turnID, cancel)
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

	// Execute agent (user message is in resp.Messages)
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

// SetTaskMode sets the task mode for the conversation.
func (c *Conversation) SetTaskMode(mode string) error {
	if err := task.ValidateMode(mode); err != nil {
		return err
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

// GetHistoryMessages returns the conversation history formatted for LLM use.
func (c *Conversation) GetHistoryMessages() []message.Message {
	return c.history.MessagesForLLM()
}

// AddHistoryMessage adds a message to the conversation history.
func (c *Conversation) AddHistoryMessage(msg message.Message) error {
	return c.history.AddMessage(msg)
}

// ID returns the conversation ID as a string.
func (c *Conversation) ID() string {
	return c.id
}

// GetSessionID returns the session identifier for this conversation.
func (c *Conversation) GetSessionID() string {
	return c.id
}

// GetAgent returns the underlying agent instance.
// This is useful for modes like ACP that need direct access to the agent.
func (c *Conversation) GetAgent() *agent.Agent {
	return c.agent
}

// GetEmitter returns the event emitter for this conversation.
// This is useful for modes like ACP that need direct access to the emitter.
func (c *Conversation) GetEmitter() *events.EventEmitter {
	return c.emitter
}

// SetID sets the conversation ID.
func (c *Conversation) SetID(id string) {
	c.id = id
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

// GetTurnID returns the current turn ID.
func (c *Conversation) GetTurnID() string {
	c.protocolMu.RLock()
	defer c.protocolMu.RUnlock()
	return c.turnID
}

// SetTurnID sets the current turn ID (thread-safe).
func (c *Conversation) SetTurnID(turnID string) {
	c.protocolMu.Lock()
	defer c.protocolMu.Unlock()
	c.turnID = turnID
}

// GetCancel returns the cancellation context function.
func (c *Conversation) GetCancel() context.CancelFunc {
	c.protocolMu.RLock()
	defer c.protocolMu.RUnlock()
	return c.cancel
}

// SetCancel sets the cancellation context function (thread-safe).
func (c *Conversation) SetCancel(cancel context.CancelFunc) {
	c.protocolMu.Lock()
	defer c.protocolMu.Unlock()
	c.cancel = cancel
}

// Cancel cancels the current turn (thread-safe).
func (c *Conversation) Cancel() {
	c.protocolMu.RLock()
	cancel := c.cancel
	c.protocolMu.RUnlock()

	if cancel != nil {
		cancel()
	}
}
