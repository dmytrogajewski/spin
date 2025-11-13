package agent

import (
	"fmt"
	"time"

	"github.com/dmytrogajewski/spin/internal/ace/bullet"
	"github.com/dmytrogajewski/spin/internal/ace/trajectory"
	"github.com/dmytrogajewski/spin/internal/message"
	"github.com/dmytrogajewski/spin/internal/orchestration"
	"github.com/dmytrogajewski/spin/internal/tools"
)

// AgentRequest represents a request to the agent.
type AgentRequest struct {
	// Input is the user's request
	Input string

	// Context is the environment context
	Context *Environment

	// Task is the task mode (required)
	// Use task.NewTask(name) to create task instances
	Task orchestration.Task

	// Timeout for this request
	Timeout time.Duration

	// History contains previous conversation messages for context
	History []message.Message
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

	// Messages contains all messages from this turn (excluding input history)
	// This includes: user input, assistant messages with tool calls, tool results, final assistant message
	// The conversation layer should persist these to maintain proper OpenAI message format
	Messages []message.Message

	// RetrievedBullets contains ACE bullets retrieved during execution
	// Populated from TrajectoryContext or simple retrieval mode
	RetrievedBullets []*bullet.Bullet

	// TrajectoryContext contains progressive execution context (for Reflector)
	TrajectoryContext *trajectory.TrajectoryContext
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
