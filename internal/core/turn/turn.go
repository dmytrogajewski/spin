// Package turn provides turn state management for conversations.
//
// A turn represents a single user-AI interaction cycle, tracking
// execution state, tool calls, token usage, and results.
package turn

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Common errors
var (
	// ErrInvalidTransition indicates an invalid state transition was attempted
	ErrInvalidTransition = errors.New("invalid state transition")
	// ErrTurnNotStarted indicates operation requires turn to be started
	ErrTurnNotStarted = errors.New("turn not started")
	// ErrTurnAlreadyDone indicates operation cannot be performed on completed/failed/cancelled turn
	ErrTurnAlreadyDone = errors.New("turn already completed/failed/cancelled")
)

// Turn represents a single user-AI interaction cycle within a conversation.
//
// A Turn tracks the complete lifecycle of processing user input, including
// LLM interactions, tool executions, and state transitions. Turns support
// approval workflows for dangerous operations and comprehensive execution
// tracking.
//
// State Machine:
//
//	Pending → Running → WaitingApproval → Running → Completed
//	                 → Completed
//	                 → Failed
//	                 → Cancelled
//
// Thread Safety:
//
//	Turn methods are thread-safe and can be called concurrently.
type Turn struct {
	// Identity
	ID        string `json:"id"`         // UUID v4
	SessionID string `json:"session_id"` // Parent session ID

	// Content
	UserInput  string `json:"user_input"`  // User's input message
	AIResponse string `json:"ai_response"` // AI's accumulated response

	// Tool Execution
	ToolCalls   []ToolCall   `json:"tool_calls"`   // Tools invoked during turn
	ToolResults []ToolResult `json:"tool_results"` // Results from tool execution

	// State
	State TurnState `json:"state"`           // Current state
	Error error     `json:"error,omitempty"` // Error if State == Failed

	// Timing
	StartedAt   time.Time `json:"started_at"`   // When turn started
	CompletedAt time.Time `json:"completed_at"` // When turn completed/failed/cancelled

	// Metrics
	Tokens TokenUsage `json:"tokens"` // Token consumption tracking

	// Metadata
	Metadata map[string]interface{} `json:"metadata,omitempty"` // Extensible metadata

	// Thread safety
	mu sync.RWMutex
}

// ToolCall represents a tool invocation by the LLM.
type ToolCall struct {
	ID       string                 `json:"id"`        // Tool call ID from LLM
	Name     string                 `json:"name"`      // Tool name
	Args     map[string]interface{} `json:"args"`      // Tool arguments
	CallTime time.Time              `json:"call_time"` // When tool was called
}

// ToolResult represents the result of a tool execution.
type ToolResult struct {
	ToolCallID string        `json:"tool_call_id"`    // Matching ToolCall.ID
	Result     interface{}   `json:"result"`          // Tool execution result
	Error      error         `json:"error,omitempty"` // Error if tool failed
	Duration   time.Duration `json:"duration"`        // Execution time
}

// TokenUsage tracks token consumption for cost monitoring and context management.
type TokenUsage struct {
	PromptTokens     int `json:"prompt_tokens"`     // Tokens in prompt
	CompletionTokens int `json:"completion_tokens"` // Tokens in completion
	TotalTokens      int `json:"total_tokens"`      // Total tokens used
}

// NewTurn creates a new turn in the Pending state.
//
// Parameters:
//   - sessionID: The parent session ID
//   - userInput: The user's input message
//
// Returns a new Turn with a generated UUID and initialized collections.
func NewTurn(sessionID, userInput string) *Turn {
	return &Turn{
		ID:          uuid.New().String(),
		SessionID:   sessionID,
		UserInput:   userInput,
		State:       StatePending,
		ToolCalls:   make([]ToolCall, 0),
		ToolResults: make([]ToolResult, 0),
		Metadata:    make(map[string]interface{}),
	}
}

// Start transitions the turn from Pending to Running state.
// Sets the StartedAt timestamp.
//
// Returns an error if the turn is not in Pending state.
func (t *Turn) Start() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if !CanTransition(t.State, StateRunning) {
		return fmt.Errorf("%w: cannot start from %s", ErrInvalidTransition, t.State)
	}

	t.State = StateRunning
	t.StartedAt = time.Now()
	return nil
}

// Complete marks the turn as successfully completed.
// Stores the final AI response, token usage, and sets the CompletedAt timestamp.
//
// Parameters:
//   - response: The final AI response
//   - tokens: Token usage statistics
//
// Returns an error if the turn is not in Running state.
func (t *Turn) Complete(response string, tokens TokenUsage) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if !CanTransition(t.State, StateCompleted) {
		return fmt.Errorf("%w: cannot complete from %s", ErrInvalidTransition, t.State)
	}

	t.State = StateCompleted
	t.AIResponse = response
	t.Tokens = tokens
	t.CompletedAt = time.Now()
	return nil
}

// Fail marks the turn as failed with the given error.
// Sets the CompletedAt timestamp.
//
// Parameters:
//   - err: The error that caused the failure
//
// Returns an error if the turn is not in Running state.
func (t *Turn) Fail(err error) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if !CanTransition(t.State, StateFailed) {
		return fmt.Errorf("%w: cannot fail from %s", ErrInvalidTransition, t.State)
	}

	t.State = StateFailed
	t.Error = err
	t.CompletedAt = time.Now()
	return nil
}

// Cancel marks the turn as cancelled.
// Sets the CompletedAt timestamp.
//
// Returns an error if the turn is not in Running or WaitingApproval state.
func (t *Turn) Cancel() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if !CanTransition(t.State, StateCancelled) {
		return fmt.Errorf("%w: cannot cancel from %s", ErrInvalidTransition, t.State)
	}

	t.State = StateCancelled
	t.CompletedAt = time.Now()
	return nil
}

// RequestApproval transitions the turn to WaitingApproval state.
// This is used when the agent needs user approval for a potentially dangerous operation.
//
// Returns an error if the turn is not in Running state.
func (t *Turn) RequestApproval() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if !CanTransition(t.State, StateWaitingApproval) {
		return fmt.Errorf("%w: cannot request approval from %s", ErrInvalidTransition, t.State)
	}

	t.State = StateWaitingApproval
	return nil
}

// Approve transitions the turn from WaitingApproval back to Running.
// This is called after the user approves a requested operation.
//
// Returns an error if the turn is not in WaitingApproval state.
func (t *Turn) Approve() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.State != StateWaitingApproval {
		return fmt.Errorf("%w: cannot approve from %s", ErrInvalidTransition, t.State)
	}

	t.State = StateRunning
	return nil
}

// Deny transitions the turn from WaitingApproval to Cancelled.
// This is called when the user denies approval for a requested operation.
//
// Returns an error if the turn is not in WaitingApproval state.
func (t *Turn) Deny() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.State != StateWaitingApproval {
		return fmt.Errorf("%w: cannot deny from %s", ErrInvalidTransition, t.State)
	}

	t.State = StateCancelled
	t.CompletedAt = time.Now()
	return nil
}

// AddToolCall adds a tool call to the turn's history.
// This is thread-safe and can be called concurrently.
//
// Parameters:
//   - call: The tool call to add
func (t *Turn) AddToolCall(call ToolCall) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.ToolCalls = append(t.ToolCalls, call)
}

// AddToolResult adds a tool result to the turn's history.
// This is thread-safe and can be called concurrently.
//
// Parameters:
//   - result: The tool result to add
func (t *Turn) AddToolResult(result ToolResult) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.ToolResults = append(t.ToolResults, result)
}

// UpdateTokens updates the token usage statistics.
// This is thread-safe and can be called concurrently.
//
// Parameters:
//   - usage: The token usage to set
func (t *Turn) UpdateTokens(usage TokenUsage) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.Tokens = usage
}

// GetTotalTokens returns the total token count.
// This is thread-safe and can be called concurrently.
//
// Returns the total number of tokens used.
func (t *Turn) GetTotalTokens() int {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.Tokens.TotalTokens
}
