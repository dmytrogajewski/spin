package core

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/dmytrogajewski/spin/internal/core/cycle"
	"github.com/dmytrogajewski/spin/internal/core/task"
	"github.com/dmytrogajewski/spin/internal/llm"
	"github.com/dmytrogajewski/spin/internal/tools"
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
	ErrNilLLM       = errors.New("LLM provider cannot be nil")
	ErrNilExecutor  = errors.New("executor cannot be nil")
	ErrNilValidator = errors.New("validator cannot be nil")
	ErrNilContext   = errors.New("context cannot be nil")
	ErrNilEmitter   = errors.New("event emitter cannot be nil")
	ErrNilRequest   = errors.New("agent request cannot be nil")
	ErrEmptyInput   = errors.New("agent request input cannot be empty")
	ErrMaxTurns     = errors.New("maximum turns reached")
)

// Agent implements the core agent logic and decision-making loop.
//
// The Agent orchestrates the interaction between the LLM, tools, and execution
// environment. It processes user requests through multiple turns of LLM calls
// and tool executions until the task is complete or limits are reached.
type Agent struct {
	llm             llm.Provider           // LLM provider interface
	executor        *Executor              // Command executor (deprecated - use toolExecutor)
	validator       *Validator             // Command validator
	context         *Environment           // Environment context
	emitter         *EventEmitter          // Event emitter
	config          *Config                // Agent configuration
	toolRegistry    *tools.Registry        // Tool registry
	taskRegistry    *task.Registry         // Task registry for execution modes
	approvalHandler ApprovalHandler        // Approval handler for user approval requests (deprecated - use approvalService)
	approvalService *ApprovalService       // Centralized approval service
	toolExecutor    *ToolExecutor          // Centralized tool execution service
	cycleDetector   *cycle.Detector        // Cycle detection and intervention
	patternDetector *cycle.PatternDetector // Advanced pattern detection
	planner         *Plan                  // Task planning and decomposition
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

// AgentConfig is now unified with core.Config for consistency.
// Use core.Config instead of this type.
type AgentConfig = Config

// NewAgent creates a new agent with the given dependencies and options.
//
// The agent requires an LLM provider, executor, validator, context, and event
// emitter. Optional configuration can be provided via functional options.
func NewAgent(
	provider llm.Provider,
	executor *Executor,
	validator *Validator,
	context *Environment,
	emitter *EventEmitter,
	opts ...AgentOption,
) (*Agent, error) {
	// Validate required dependencies
	if provider == nil {
		return nil, ErrNilLLM
	}
	if executor == nil {
		return nil, ErrNilExecutor
	}
	if validator == nil {
		return nil, ErrNilValidator
	}
	if context == nil {
		return nil, ErrNilContext
	}
	if emitter == nil {
		return nil, ErrNilEmitter
	}

	// Create default tool registry with built-in tools
	registry := tools.NewRegistry()
	_ = registry.Register(tools.NewReadFileTool())
	_ = registry.Register(tools.NewWriteFileTool())
	_ = registry.Register(tools.NewListDirectoryTool())
	_ = registry.Register(tools.NewExecuteCommandTool(executor, validator))
	_ = registry.Register(tools.NewGetContextTool(context))
	_ = registry.Register(tools.NewApplyPatchTool(context.WorkDir))
	_ = registry.Register(tools.NewFileSearchTool(context.WorkDir))
	_ = registry.Register(tools.NewGitContextTool(context.WorkDir))

	// Create default task registry with built-in modes
	taskRegistry := task.NewRegistry()
	if err := taskRegistry.Register("regular", task.NewRegular()); err != nil {
		return nil, fmt.Errorf("failed to register regular task: %w", err)
	}
	if err := taskRegistry.Register("review", task.NewReview()); err != nil {
		return nil, fmt.Errorf("failed to register review task: %w", err)
	}
	if err := taskRegistry.Register("compact", task.NewCompact()); err != nil {
		return nil, fmt.Errorf("failed to register compact task: %w", err)
	}
	if err := taskRegistry.Register("planning", task.NewPlanning()); err != nil {
		return nil, fmt.Errorf("failed to register planning task: %w", err)
	}
	if err := taskRegistry.SetDefault("regular"); err != nil {
		return nil, fmt.Errorf("failed to set default task: %w", err)
	}

	// Create default config
	defaultConfig := DefaultConfig()

	// Create cycle detection config from default config
	cycleConfig := cycle.Config{
		WindowSize:       defaultConfig.CycleDetection.WindowSize,
		SimilarityThresh: defaultConfig.CycleDetection.SimilarityThresh,
		ToolRepeatLimit:  defaultConfig.CycleDetection.ToolRepeatLimit,
		ErrorRepeatLimit: defaultConfig.CycleDetection.ErrorRepeatLimit,
		Enabled:          defaultConfig.CycleDetection.Enabled,
	}

	// Create agent with defaults
	agent := &Agent{
		llm:           provider,
		executor:      executor,
		validator:     validator,
		context:       context,
		emitter:       emitter,
		toolRegistry:  registry,
		taskRegistry:  taskRegistry,
		cycleDetector: cycle.NewDetector(cycleConfig),
		config:        defaultConfig,
	}

	// Apply options
	for _, opt := range opts {
		if err := opt(agent); err != nil {
			return nil, fmt.Errorf("applying option: %w", err)
		}
	}

	// Initialize approval service with agent's dependencies
	// This must come after applying options so approvalHandler is set
	agent.approvalService = NewApprovalServiceWithConfig(ApprovalServiceConfig{
		Handler:         agent.approvalHandler,
		Emitter:         agent.emitter,
		Validator:       agent.validator,
		ApprovalTimeout: agent.config.ApprovalTimeout,
	})

	// Initialize tool executor with agent's dependencies
	agent.toolExecutor = NewToolExecutor(ToolExecutorConfig{
		Registry:        agent.toolRegistry,
		Validator:       agent.validator,
		ApprovalService: agent.approvalService,
		Emitter:         agent.emitter,
		WorkDir:         agent.context.WorkDir,
	})

	return agent, nil
}

// resolveTask determines which task to use for this request.
//
// Precedence order:
//  1. Explicit req.Task object (if non-nil)
//  2. Task by name req.TaskName (if non-empty, looked up in registry)
//  3. Default task from registry
//
// Returns an error if:
//   - TaskName is provided but not found in registry
//   - No default task is configured in registry (should never happen with default initialization)
func (a *Agent) resolveTask(req *AgentRequest) (Task, error) {
	// Priority 1: Explicit task object provided
	if req.Task != nil {
		slog.Debug("task resolution: using explicit task", "name", req.Task.Name())
		return req.Task, nil
	}

	// Priority 2: Task name provided - look up in registry
	if req.TaskName != "" {
		task, err := a.taskRegistry.Get(req.TaskName)
		if err != nil {
			return nil, fmt.Errorf("task resolution failed: task '%s' not found in registry", req.TaskName)
		}
		slog.Debug("task resolution: resolved by name", "name", req.TaskName)
		return task, nil
	}

	// Priority 3: Use default task from registry
	task, err := a.taskRegistry.GetDefault()
	if err != nil {
		return nil, fmt.Errorf("task resolution failed: no default task configured: %w", err)
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
		return resp, err
	}

	// Finalize response
	a.finalizeResponse(resp, messages, historyLen)
	return resp, nil
}

// BuildToolsForTask constructs the filtered tool list for the LLM request,
// based on the task mode's allowed tools.
//
// Algorithm:
//  1. Get all tool schemas from tool registry
//  2. Get allowed tool names from task.AllowedTools()
//  3. Build allowed tool set for O(1) lookup
//  4. Filter tool schemas, keeping only allowed tools
//  5. Convert to LLM tool format
//
// Returns nil if tool registry is nil (no tools available).
// Returns empty slice if no tools are allowed (not an error).
//
// This method is exported primarily for testing purposes to verify
// that tool filtering is working correctly for each task mode.
func (a *Agent) BuildToolsForTask(task Task) ([]llm.Tool, error) {
	if a.toolRegistry == nil {
		return nil, nil
	}

	// Get all available tools
	allSchemas := a.toolRegistry.ListSchemas()
	if len(allSchemas) == 0 {
		return nil, nil
	}

	// Get allowed tools for this mode
	allowedTools := task.AllowedTools()
	if len(allowedTools) == 0 {
		slog.Debug("task allows no tools", "task", task.Name())
		return []llm.Tool{}, nil
	}

	// Build allowed tool set for O(1) lookup
	allowedSet := make(map[string]bool, len(allowedTools))
	for _, name := range allowedTools {
		allowedSet[name] = true
	}

	// Filter tools
	filtered := make([]llm.Tool, 0, len(allSchemas))
	for _, schema := range allSchemas {
		// Check if tool is allowed in this mode
		if !allowedSet[schema.Function.Name] {
			continue
		}

		// Convert ParameterSchema struct to map[string]interface{}
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

// GetTaskRegistry returns the agent's task registry.
// This is useful for testing and introspection of registered task modes.
// The registry is thread-safe for concurrent access.
func (a *Agent) GetTaskRegistry() *task.Registry {
	return a.taskRegistry
}

// ListTaskModes returns all registered task mode names in sorted order.
// Returns nil if the task registry is not initialized.
// This is a convenience method that wraps the registry's List() method.
func (a *Agent) ListTaskModes() []string {
	if a.taskRegistry == nil {
		return nil
	}
	return a.taskRegistry.List()
}

// CreatePlan creates a new execution plan for the given task.
// This method uses the LLM to decompose complex tasks into manageable steps.
func (a *Agent) CreatePlan(ctx context.Context, task string) (*Plan, error) {
	if task == "" {
		return nil, ErrEmptyInput
	}

	// Create a new plan
	plan := NewPlan(task)

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
`, task)

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
	plan.EstimatedDuration = plan.CalculateEstimatedDuration()

	return plan, nil
}
