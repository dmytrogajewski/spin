package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/dmytrogajewski/spin/internal/detection"
	"github.com/dmytrogajewski/spin/internal/events"
	"github.com/dmytrogajewski/spin/internal/llm"
	"github.com/dmytrogajewski/spin/internal/orchestration"
	"github.com/dmytrogajewski/spin/internal/security"
)

// Default agent configuration values
const (
	DefaultMaxTurns        = 50
	DefaultAgentTimeout    = 5 * time.Minute
	DefaultTemperature     = 0.7
	DefaultMaxTokens       = 4096
	DefaultEventBufferSize = 100
)

// Common agent errors
var (
	ErrNilLLM           = errors.New("LLM provider cannot be nil")
	ErrNilSecurity      = errors.New("security service cannot be nil")
	ErrNilDetection     = errors.New("detection service cannot be nil")
	ErrNilOrchestration = errors.New("orchestration service cannot be nil")
	ErrNilContext       = errors.New("context cannot be nil")
	ErrNilEmitter       = errors.New("event emitter cannot be nil")
	ErrNilRequest       = errors.New("agent request cannot be nil")
	ErrEmptyInput       = errors.New("agent request input cannot be empty")
	ErrMaxTurns         = errors.New("maximum turns reached")
)

// Agent implements the core agent logic and decision-making loop.
//
// The Agent orchestrates the interaction between the LLM, tools, and execution
// environment. It processes user requests through multiple turns of LLM calls
// and tool executions until the task is complete or limits are reached.
//
// REFACTORED: Agent now uses service-based architecture (2025-10-19)
// - SecurityService handles validation and approval
// - DetectionService handles cycle and pattern detection
// - OrchestrationService handles tool execution and task management
type Agent struct {
	// Core LLM interaction
	llm llm.Provider

	// Service layers
	security      *security.SecurityService
	detection     *detection.DetectionService
	orchestration *orchestration.OrchestrationService

	// Infrastructure
	context *Environment
	emitter *events.EventEmitter
	config  *Config
}

// Task defines the interface for different execution modes.
// This interface is implemented by types in the task subpackage.
type Task interface {
	// Name returns the unique identifier for this task mode
	Name() string

	// SystemPrompt returns the mode-specific system prompt
	SystemPrompt() string

	// AllowedTools returns the list of tool names permitted in this mode
	AllowedTools() []string

	// MaxTokens returns the token budget for this mode
	MaxTokens() int

	// Validate validates the task configuration
	Validate() error
}

// AgentConfig is now unified with manager.Config for consistency.
// Use manager.Config instead of this type.
type AgentConfig = Config

// NewAgent creates a new agent with service-based architecture.
//
// The agent requires an LLM provider and three service layers:
// - SecurityService: handles command validation and approval
// - DetectionService: handles cycle and pattern detection
// - OrchestrationService: handles tool execution and task management
//
// Optional configuration can be provided via functional options.
//
// REFACTORED: This constructor now uses services instead of individual dependencies.
// The old constructor signature is no longer supported - callers must build services first.
func NewAgent(
	provider llm.Provider,
	security *security.SecurityService,
	detection *detection.DetectionService,
	orchestration *orchestration.OrchestrationService,
	context *Environment,
	emitter *events.EventEmitter,
	opts ...AgentOption,
) (*Agent, error) {
	// Validate required dependencies
	if provider == nil {
		return nil, ErrNilLLM
	}
	if security == nil {
		return nil, ErrNilSecurity
	}
	if detection == nil {
		return nil, ErrNilDetection
	}
	if orchestration == nil {
		return nil, ErrNilOrchestration
	}
	if context == nil {
		return nil, ErrNilContext
	}
	if emitter == nil {
		return nil, ErrNilEmitter
	}

	// Create agent with services
	agent := &Agent{
		llm:           provider,
		security:      security,
		detection:     detection,
		orchestration: orchestration,
		context:       context,
		emitter:       emitter,
		config:        DefaultConfig(),
	}

	// Apply options
	for _, opt := range opts {
		if err := opt(agent); err != nil {
			return nil, fmt.Errorf("applying option: %w", err)
		}
	}

	return agent, nil
}

// resolveTask determines which task to use for this request.
//
// Precedence order:
//  1. Explicit req.Task object (if non-nil)
//  2. Task by name req.TaskName (if non-empty, looked up in orchestration service)
//  3. Default task from orchestration service
//
// Returns an error if:
//   - TaskName is provided but not found
//   - No default task is configured
func (a *Agent) resolveTask(req *AgentRequest) (Task, error) {
	// Priority 1: Explicit task object provided
	if req.Task != nil {
		slog.Debug("task resolution: using explicit task", "name", req.Task.Name())
		return req.Task, nil
	}

	// Priority 2: Task name provided - look up via orchestration
	if req.TaskName != "" {
		task, err := a.orchestration.GetTask(req.TaskName)
		if err != nil {
			return nil, fmt.Errorf("task resolution failed: %w", err)
		}
		slog.Debug("task resolution: resolved by name", "name", req.TaskName)
		return task, nil
	}

	// Priority 3: Use default task from orchestration
	task, err := a.orchestration.GetDefaultTask()
	if err != nil {
		return nil, fmt.Errorf("task resolution failed: %w", err)
	}
	slog.Debug("task resolution: using default task", "name", task.Name())
	return task, nil
}

// Execute runs the agent loop for a request.
//
// The agent will:
// 1. Build a prompt with context and history
// 2. Call the LLM to get a response
// 3. Process any tool calls in the response
// 4. Continue until the task is complete or limits are reached
//
// The agent respects the context timeout and max turns limit.
func (a *Agent) Execute(ctx context.Context, req *AgentRequest) (*AgentResponse, error) {
	// Validate request and setup
	ctx, resp, err := a.executeSetup(ctx, req)
	if err != nil {
		return resp, err
	}

	// Resolve task mode
	task, err := a.resolveTask(req)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve task mode: %w", err)
	}

	// Apply timeout if needed
	ctx, cancel := a.applyTimeout(ctx)
	defer cancel()

	// Build initial prompt and execute agent loop
	messages := a.buildPrompt(req)
	historyLen := len(messages)

	messages, resp, err = a.executeAgentLoop(ctx, messages, task, resp)
	if err != nil {
		// Emit turn failed event
		a.emitter.Emit(events.Event{
			Type:      events.EventTurnFailed,
			Timestamp: time.Now(),
			Data:      events.TurnEventData{},
		})
		return resp, err
	}

	// Finalize response
	a.finalizeResponse(resp, messages, historyLen)
	return resp, nil
}

// executeSetup validates the request and sets up the execution context.
func (a *Agent) executeSetup(ctx context.Context, req *AgentRequest) (context.Context, *AgentResponse, error) {
	if req == nil {
		return ctx, nil, fmt.Errorf("request cannot be nil")
	}
	if req.Input == "" {
		return ctx, nil, fmt.Errorf("request input cannot be empty")
	}

	// Create response
	resp := &AgentResponse{
		Success: true,
	}

	return ctx, resp, nil
}

// applyTimeout applies timeout to the context if needed.
func (a *Agent) applyTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	timeout := a.config.Timeout
	if timeout == 0 {
		timeout = DefaultAgentTimeout
	}
	return context.WithTimeout(ctx, timeout)
}

// buildPrompt builds the initial prompt for the agent.
func (a *Agent) buildPrompt(req *AgentRequest) []Message {
	// Start with history messages if provided
	messages := make([]Message, 0, len(req.History)+1)
	if len(req.History) > 0 {
		messages = append(messages, req.History...)
	}

	// Add current user input
	messages = append(messages, Message{
		Role:    "user",
		Content: req.Input,
	})

	return messages
}

// finalizeResponse finalizes the agent response.
func (a *Agent) finalizeResponse(resp *AgentResponse, messages []Message, historyLen int) {
	if len(messages) > historyLen {
		// Get the last assistant message
		for i := len(messages) - 1; i >= historyLen; i-- {
			if messages[i].Role == "assistant" {
				resp.Output = messages[i].Content
				break
			}
		}
	}

	// Emit turn complete event to signal agent is done
	a.emitter.Emit(events.Event{
		Type:      events.EventTurnComplete,
		Timestamp: time.Now(),
		Data:      events.TurnEventData{},
	})
}

// BuildToolsForTask constructs the filtered tool list for the LLM request,
// based on the task mode's allowed tools.
//
// This method delegates to the orchestration service's tool registry.
func (a *Agent) BuildToolsForTask(task Task) ([]llm.Tool, error) {
	if a.orchestration == nil {
		return nil, nil
	}

	toolRegistry := a.orchestration.GetToolRegistry()
	if toolRegistry == nil {
		return nil, nil
	}

	// Get all available tools
	allSchemas := toolRegistry.ListSchemas()
	if len(allSchemas) == 0 {
		return nil, nil
	}

	// Get allowed tools for this mode
	allowedTools := task.AllowedTools()

	// Empty list means all tools are allowed (no filtering)
	allowAllTools := len(allowedTools) == 0

	// Build allowed tool set for O(1) lookup (if filtering is needed)
	var allowedSet map[string]bool
	if !allowAllTools {
		allowedSet = make(map[string]bool, len(allowedTools))
		for _, name := range allowedTools {
			allowedSet[name] = true
		}
	}

	// Filter tools
	filtered := make([]llm.Tool, 0, len(allSchemas))
	for _, schema := range allSchemas {
		// Check if tool is allowed in this mode
		if !allowAllTools && !allowedSet[schema.Function.Name] {
			continue
		}

		// Convert ParameterSchema to map (defined in agent_tools.go)
		params := convertParameterSchemaToMap(schema.Function.Parameters)

		filtered = append(filtered, llm.Tool{
			Type: schema.Type,
			Function: llm.Function{
				Name:        schema.Function.Name,
				Description: schema.Function.Description,
				Parameters:  params,
			},
		})
	}

	slog.Debug("filtered tools for task",
		"task", task.Name(),
		"total", len(allSchemas),
		"allowed", len(filtered))

	return filtered, nil
}

// GetTaskRegistry returns the agent's task registry via orchestration service.
// This is useful for testing and introspection of registered task modes.
//
// This method delegates to the orchestration service to access the task registry.
// Returns nil if the orchestration service doesn't have a task registry configured.
func (a *Agent) GetTaskRegistry() *orchestration.Registry {
	if a.orchestration == nil {
		return nil
	}

	return a.orchestration.GetTaskRegistry()
}

// ListTaskModes returns all registered task mode names in sorted order.
// This delegates to the orchestration service.
func (a *Agent) ListTaskModes() []string {
	return a.orchestration.ListTasks()
}

// CreatePlan creates a new execution plan for the given task.
// This method uses the LLM to decompose complex tasks into manageable steps.
func (a *Agent) CreatePlan(ctx context.Context, taskName string) (*Plan, error) {
	if taskName == "" {
		return nil, ErrEmptyInput
	}

	// Create a new plan
	plan := NewPlan(nil)

	// Use LLM to decompose the task into steps
	decompositionPrompt := fmt.Sprintf(`
Decompose the following task into specific, actionable steps:

Task: %s

Please provide a JSON response with the following structure:
{
  "steps": [
    {
      "id": "step_1",
      "description": "Clear description of what to do",
      "action": "Specific action to perform",
      "depends_on": [],
      "estimated_duration": "5m"
    }
  ]
}

Guidelines:
- Each step should be atomic and testable
- Include dependencies between steps
- Provide realistic time estimates
- Focus on concrete actions, not abstract concepts
`, taskName)

	// Call LLM for task decomposition
	req := llm.CompletionRequest{
		Messages: []llm.Message{
			{
				Role:    "user",
				Content: decompositionPrompt,
			},
		},
		MaxTokens:   1000,
		Temperature: 0.3, // Lower temperature for more consistent planning
	}

	resp, err := a.llm.Complete(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("LLM completion failed: %w", err)
	}

	// Parse the response and create steps
	var decomposition struct {
		Steps []struct {
			ID                string   `json:"id"`
			Description       string   `json:"description"`
			Action            string   `json:"action"`
			DependsOn         []string `json:"depends_on"`
			EstimatedDuration string   `json:"estimated_duration"`
		} `json:"steps"`
	}

	if err := json.Unmarshal([]byte(resp.Content), &decomposition); err != nil {
		return nil, fmt.Errorf("failed to parse LLM response: %w", err)
	}

	// Create steps
	for _, stepData := range decomposition.Steps {
		duration, _ := time.ParseDuration(stepData.EstimatedDuration)
		step := Step{
			ID:                stepData.ID,
			Description:       stepData.Description,
			Action:            stepData.Action,
			DependsOn:         stepData.DependsOn,
			Status:            StepStatusPending,
			EstimatedDuration: duration,
		}
		plan.Steps = append(plan.Steps, step)
	}

	// Validate the plan structure
	if err := plan.ValidateStructure(); err != nil {
		return nil, fmt.Errorf("plan validation failed: %w", err)
	}

	// Calculate total estimated duration
	// plan.EstimatedDuration is calculated by the method

	return plan, nil
}
