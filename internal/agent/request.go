package agent

import (
	"fmt"
	"time"

	"github.com/dmytrogajewski/spin/internal/orchestration"
	"github.com/dmytrogajewski/spin/internal/tools"
)

// AgentRequest represents a request to the agent.
type AgentRequest struct {
	// Input is the user's request
	Input string

	// Context is the environment context
	Context *Environment

	// Task is the task mode
	Task orchestration.Task

	// TaskName is the task name
	TaskName string

	// Timeout for this request
	Timeout time.Duration

	// History contains previous conversation messages for context
	History []Message
}

// AgentResponse represents the agent's response.
type AgentResponse struct {
	// Output is the agent's response
	Output string

	// Success indicates if the request was successful
	Success bool

	// Error if any occurred
	Error error

	// ToolCalls are the tool calls made
	ToolCalls []orchestration.ToolCall

	// FinishReason indicates why the conversation finished
	FinishReason string

	// Duration of the request
	Duration time.Duration
}

// Message represents a conversation message.
type Message struct {
	// Role is the message role (user, assistant, system)
	Role string

	// Content is the message content
	Content string

	// ToolCalls are tool calls in this message
	ToolCalls []orchestration.ToolCall

	// ToolCallID is the ID of the tool call this message responds to
	ToolCallID string

	// Timestamp when the message was created
	Timestamp time.Time
}

// Plan represents an execution plan.
type Plan struct {
	// ID is the plan identifier
	ID string

	// Steps are the plan steps
	Steps []Step

	// Status is the plan status
	Status string
}

// Step represents a plan step.
type Step struct {
	ID                string
	Description       string
	Action            string
	DependsOn         []string
	Status            StepStatus
	EstimatedDuration time.Duration
}

// StepStatus represents the status of a plan step.
type StepStatus int

const (
	StepStatusPending StepStatus = iota
	StepStatusRunning
	StepStatusCompleted
	StepStatusFailed
)

// NewPlan creates a new plan.
func NewPlan(task orchestration.Task) *Plan {
	return &Plan{
		ID:     fmt.Sprintf("plan-%d", time.Now().Unix()),
		Steps:  []Step{},
		Status: "pending",
	}
}

// ValidateStructure validates the plan structure.
func (p *Plan) ValidateStructure() error {
	if p.ID == "" {
		return fmt.Errorf("plan ID cannot be empty")
	}
	if len(p.Steps) == 0 {
		return fmt.Errorf("plan must have at least one step")
	}
	return nil
}

// EstimatedDuration returns the total estimated duration of all steps.
func (p *Plan) EstimatedDuration() time.Duration {
	var total time.Duration
	for _, step := range p.Steps {
		total += step.EstimatedDuration
	}
	return total
}

// EventType represents the category of event.
type EventType int

const (
	EventWarning EventType = iota
	EventTurnPaused
	EventToolCallStart
	EventToolCallComplete
)

// Event represents an event.
type Event struct {
	Type      EventType
	Timestamp time.Time
	Data      interface{}
}

// ToolCallCompleteData represents tool call completion data.
type ToolCallCompleteData struct {
	ToolID   string
	ToolName string
	Success  bool
	Error    string
	Output   string
}

// ToolCallStartData represents tool call start data.
type ToolCallStartData struct {
	ToolID     string
	ToolName   string
	Parameters tools.ToolParameters
}

// Role constants
const (
	RoleUser      = "user"
	RoleAssistant = "assistant"
	RoleTool      = "tool"
	RoleSystem    = "system"
)
