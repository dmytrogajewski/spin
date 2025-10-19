package core

import (
	"context"
	"fmt"
	"time"
)

// ApprovalRequest represents a command approval request sent to the approval handler.
type ApprovalRequest struct {
	// ID is a unique identifier for this approval request (UUID)
	ID string

	// Command is the command requiring approval
	Command *Command

	// Reason explains why approval is needed (from Validator)
	Reason string

	// WorkDir is the working directory where the command will execute
	WorkDir string

	// Timestamp is when the request was created
	Timestamp time.Time
}

// ApprovalResponse represents the user's approval decision.
type ApprovalResponse struct {
	// RequestID must match the ApprovalRequest.ID
	RequestID string

	// Approved indicates whether the command was approved (true) or denied (false)
	Approved bool

	// Reason is an optional user-provided reason for the decision
	Reason string

	// ModifiedCommand is an optional modified version of the command.
	// If provided, the original command will be replaced and re-validated.
	// If empty, the original command is used as-is.
	ModifiedCommand string

	// Timestamp is when the response was created
	Timestamp time.Time
}

// ApprovalHandler is a callback function for handling approval requests.
// It receives an ApprovalRequest and must return an ApprovalResponse.
// The handler should block until the user makes a decision or timeout occurs.
// If the handler is nil, commands requiring approval are automatically denied.
type ApprovalHandler func(ApprovalRequest) ApprovalResponse

// AgentRequest represents a request to the agent.
type AgentRequest struct {
	// Input is the user's request
	Input string

	// History is the conversation history
	History []Message

	// Context is the environment context (optional, will use agent's context if nil)
	Context *Environment

	// Task is the task mode (optional, uses regular mode if nil)
	Task Task

	// TaskName is the name of the task mode to use (optional, resolved from registry).
	// Takes precedence over default but is overridden by explicit Task field.
	// If both Task and TaskName are provided, Task takes precedence.
	TaskName string

	// WorkDir is the working directory
	WorkDir string
}

// AgentResponse represents the agent's response.
type AgentResponse struct {
	// Content is the response content
	Content string

	// Messages contains all messages generated during turn execution.
	// This includes assistant messages with tool calls and tool result messages.
	// These should be added to conversation history to maintain context.
	Messages []Message

	// ToolCalls are the tools that were called
	ToolCalls []*ToolCall

	// ToolResults are the tool execution results
	ToolResults []*ToolResult

	// TurnsUsed is the number of turns used
	TurnsUsed int

	// TokensUsed is the tokens consumed
	TokensUsed int

	// FinishReason is the reason for completion
	FinishReason string

	// Error is any error that occurred
	Error error
}

// executeSetup validates the request and initializes tracing and response.
func (a *Agent) executeSetup(ctx context.Context, req *AgentRequest) (context.Context, *AgentResponse, error) {
	if req == nil {
		return ctx, nil, ErrNilRequest
	}

	if req.Input == "" {
		return ctx, nil, ErrEmptyInput
	}

	resp := &AgentResponse{
		Messages:    make([]Message, 0),
		ToolCalls:   make([]*ToolCall, 0),
		ToolResults: make([]*ToolResult, 0),
		TurnsUsed:   0,
		TokensUsed:  0,
	}

	return ctx, resp, nil
}

// applyTimeout applies timeout from config if not already set.
func (a *Agent) applyTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	if _, ok := ctx.Deadline(); !ok {
		return context.WithTimeout(ctx, a.config.Timeout)
	}
	return ctx, func() {}
}

// finalizeResponse completes the response with final messages and events.
func (a *Agent) finalizeResponse(resp *AgentResponse, messages []Message, historyLen int) {
	// Capture all new messages generated during this turn
	if len(messages) > historyLen {
		resp.Messages = messages[historyLen:]
	}

	// Emit completion event
	a.emitter.Emit(Event{
		Type:      EventTurnComplete,
		Timestamp: time.Now(),
		Data: TurnEventData{
			TurnsUsed:  resp.TurnsUsed,
			TokensUsed: resp.TokensUsed,
			Status:     "complete",
			Message:    resp.FinishReason,
			MaxTurns:   a.config.MaxTurns,
		},
	})

}

// buildPrompt constructs the LLM prompt with context and history.
func (a *Agent) buildPrompt(req *AgentRequest) []Message {
	messages := make([]Message, 0)

	// Add system message
	systemContent := a.buildSystemMessage(req)
	messages = append(messages, Message{
		Role:      RoleSystem,
		Content:   systemContent,
		Timestamp: time.Now(),
	})

	// Add conversation history
	if req.History != nil {
		messages = append(messages, req.History...)
	}

	// Add current user input
	messages = append(messages, Message{
		Role:      RoleUser,
		Content:   req.Input,
		Timestamp: time.Now(),
	})

	return messages
}

// buildSystemMessage constructs the system message with context.
func (a *Agent) buildSystemMessage(req *AgentRequest) string {
	// Start with task-specific prompt if provided
	var prompt string
	if req.Task != nil {
		prompt = req.Task.SystemPrompt()
	} else {
		// Default system prompt (agentic, action-oriented)
		prompt = `You are a decisive AI coding agent.

	CAPABILITIES:
	- Read/modify files, run commands (with safety), search code, use Git

	BEHAVIOR:
	- Make decisions and proceed; state assumptions briefly when unsure
	- Prefer applying edits via tools over suggesting snippets only
	- Validate after changes (tests, lints, or checks) and iterate
	- Keep explanations concise; focus on actions and code

	OUTPUT:
	- Provide concrete edits and commands, then execute with tools when appropriate
	- Summarize impact at the end`
	}

	// Add environment context
	ctx := req.Context
	if ctx == nil {
		ctx = a.context
	}

	if ctx != nil {
		prompt += "\n\nEnvironment:\n"
		prompt += fmt.Sprintf("- OS: %s (%s)\n", ctx.OS.OS, ctx.OS.Arch)
		prompt += fmt.Sprintf("- Working Directory: %s\n", ctx.WorkDir)

		if ctx.Git != nil {
			prompt += fmt.Sprintf("- Git Branch: %s\n", ctx.Git.Branch)
			if ctx.Git.HasChanges {
				prompt += "- Git Status: Uncommitted changes present\n"
			}
		}

		if len(ctx.Languages) > 0 {
			prompt += fmt.Sprintf("- Languages: %v\n", ctx.Languages)
		}
	}

	// Add safety guidelines
	prompt += "\n\nSafety Guidelines:\n"
	prompt += "- Always verify commands before execution\n"
	prompt += "- Be careful with file modifications\n"
	prompt += "- Ask for confirmation for dangerous operations\n"

	return prompt
}
