package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/dmytrogajewski/spin/internal/ace/bullet"
	"github.com/dmytrogajewski/spin/internal/ace/trajectory"
	"github.com/dmytrogajewski/spin/internal/detection"
	spinerrors "github.com/dmytrogajewski/spin/internal/errors"
	"github.com/dmytrogajewski/spin/internal/events"
	"github.com/dmytrogajewski/spin/internal/llm"
	"github.com/dmytrogajewski/spin/internal/message"
	"github.com/dmytrogajewski/spin/internal/orchestration"
	"github.com/dmytrogajewski/spin/internal/security"
	"github.com/dmytrogajewski/spin/internal/tools"
	"github.com/openai/openai-go"
)

// Default agent configuration values
const (
	DefaultMaxTurns        = 500
	DefaultAgentTimeout    = 60 * time.Minute
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
	aceService    *ACEService // ACE (Agentic Context Engineering) - optional

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

// AgentOption is a functional option for configuring an Agent.
type AgentOption func(*Agent) error

// WithACEService sets the ACE service for the agent.
// If not provided, agent will operate without ACE (no persistent learning).
func WithACEService(aceService *ACEService) AgentOption {
	return func(a *Agent) error {
		a.aceService = aceService
		return nil
	}
}

// WithMaxTurns sets the maximum number of agent turns.
func WithMaxTurns(maxTurns int) AgentOption {
	return func(a *Agent) error {
		if maxTurns <= 0 {
			return spinerrors.Newf(spinerrors.CodeValidation, "Agent.WithMaxTurns", nil, "max turns must be positive, got %d", maxTurns)
		}
		a.config.MaxTurns = maxTurns
		return nil
	}
}

// WithAgentTimeout sets the agent execution timeout.
func WithAgentTimeout(timeout time.Duration) AgentOption {
	return func(a *Agent) error {
		if timeout <= 0 {
			return spinerrors.Newf(spinerrors.CodeValidation, "Agent.WithAgentTimeout", nil, "timeout must be positive, got %v", timeout)
		}
		a.config.Timeout = timeout
		return nil
	}
}

// WithTemperature sets the LLM temperature.
func WithTemperature(temperature float64) AgentOption {
	return func(a *Agent) error {
		if temperature < 0 || temperature > 2 {
			return spinerrors.Newf(spinerrors.CodeValidation, "Agent.WithTemperature", nil, "temperature must be between 0 and 2, got %f", temperature)
		}
		a.config.Temperature = temperature
		return nil
	}
}

// WithMaxTokens sets the maximum tokens per LLM call.
func WithMaxTokens(maxTokens int) AgentOption {
	return func(a *Agent) error {
		if maxTokens <= 0 {
			return spinerrors.Newf(spinerrors.CodeValidation, "Agent.WithMaxTokens", nil, "max tokens must be positive, got %d", maxTokens)
		}
		a.config.MaxTokens = maxTokens
		return nil
	}
}

// WithRequireApproval sets whether dangerous commands require approval.
func WithRequireApproval(require bool) AgentOption {
	return func(a *Agent) error {
		a.config.RequireApproval = require
		return nil
	}
}

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
			return nil, spinerrors.New(spinerrors.CodeValidation, "Agent.NewAgent", "applying option failed", err)
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
			return nil, spinerrors.New(spinerrors.CodeNotFound, "Agent.resolveTask", "task resolution failed", err)
		}
		slog.Debug("task resolution: resolved by name", "name", req.TaskName)
		return task, nil
	}

	// Priority 3: Use default task from orchestration
	task, err := a.orchestration.GetDefaultTask()
	if err != nil {
		return nil, spinerrors.New(spinerrors.CodeNotFound, "Agent.resolveTask", "task resolution failed", err)
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
	slog.Info("agent execution started", "input_len", len(req.Input), "task_mode", req.TaskName)
	// Validate request and setup
	ctx, resp, err := a.executeSetup(ctx, req)
	if err != nil {
		slog.Error("agent setup failed", "error", err)
		return resp, err
	}

	// Resolve task mode
	task, err := a.resolveTask(req)
	if err != nil {
		slog.Error("task resolution failed", "task_name", req.TaskName, "error", err)
		return nil, spinerrors.New(spinerrors.CodeNotFound, "Agent.Execute", "failed to resolve task mode", err)
	}
	slog.Debug("task resolved", "task_name", task.Name(), "max_tokens", task.MaxTokens())

	// Apply timeout if needed
	ctx, cancel := a.applyTimeout(ctx)
	defer cancel()

	// Build initial prompt and execute agent loop
	messages := a.buildPrompt(req)
	historyLen := len(messages)

	// Initialize trajectory context for progressive retrieval
	initialQuery := extractInitialQuery(messages)
	trajCtx := trajectory.NewTrajectoryContext(initialQuery)

	messages, resp, err = a.executeAgentLoop(ctx, messages, task, resp, trajCtx)
	if err != nil {
		slog.Error("agent loop failed", "error", err, "finish_reason", resp.FinishReason)
		// Emit turn failed event
		a.emitter.Emit(events.Event{
			Type:      events.EventTurnFailed,
			Timestamp: time.Now(),
			Data:      events.TurnEventData{},
		})
		return resp, err
	}

	// Store trajectory context in response
	resp.TrajectoryContext = trajCtx

	// Finalize response
	a.finalizeResponse(resp, messages, historyLen)

	// ACE: Generate bullets from execution (both success AND failure) if enabled
	// Run synchronously to ensure bullets are learned before returning
	if a.aceService != nil {
		slog.Info("Starting bullet generation from execution", "success", resp.Success)

		// Use TrajectoryContext.ToTrajectory() instead of buildExecutionTrajectory()
		trajectory := trajCtx.ToTrajectory()
		trajectory.Success = resp.Success // Update success status from response
		slog.Debug("Execution trajectory built", "steps", len(trajectory.Steps), "success", trajectory.Success)

		// Use Reflector+Curator pipeline if AutoReflect is enabled, otherwise use simple generator
		var learnedBullets []*bullet.Bullet
		var err error
		if a.aceService.config.Generation.AutoReflect {
			learnedBullets, err = a.aceService.GenerateBulletsWithReflectionFromTrajectory(ctx, trajectory)
		} else {
			// For simple generation without reflection, convert trajectory to string
			var summaryBuilder strings.Builder
			summaryBuilder.WriteString("Task: ")
			summaryBuilder.WriteString(trajectory.Query)
			summaryBuilder.WriteString("\n\nExecution Steps:\n")
			for _, step := range trajectory.Steps {
				summaryBuilder.WriteString(fmt.Sprintf("- [%s] %s\n", step.Type, step.Content))
			}
			summaryBuilder.WriteString("\nResult: ")
			summaryBuilder.WriteString(trajectory.Output)

			learnedBullets, err = a.aceService.GenerateBullets(ctx, summaryBuilder.String(), "trajectory")
		}

		if err != nil {
			slog.Warn("ACE bullet generation failed", "error", err)
			// Don't fail the entire execution if bullet generation fails
		} else {
			slog.Info("Successfully generated bullets from execution", "count", len(learnedBullets))

			if len(learnedBullets) == 0 {
				slog.Debug("No bullets to display (empty result)")
			}

			if len(learnedBullets) > 0 {
				// Build bullet list for display
				bulletList := ""
				for i, b := range learnedBullets {
					bulletList += fmt.Sprintf("  %d. %s\n", i+1, b.Content)
				}

				successStr := "successful"
				if !resp.Success {
					successStr = "failed"
				}
				learnMsg := fmt.Sprintf("Learned %d new insight%s from this %s execution:\n%s",
					len(learnedBullets), pluralize(len(learnedBullets)), successStr, bulletList)

				// Emit content complete event to show ACE learning activity as a chat block
				a.emitter.Emit(events.Event{
					Type:      events.EventContentComplete,
					Timestamp: time.Now(),
					Data: events.ContentDeltaData{
						Content: learnMsg,
						Role:    string(message.RoleAssistant),
					},
				})
			}
		}
	} else {
		slog.Debug("Bullet generation skipped", "ace_service_nil", a.aceService == nil)
	}

	// Emit turn complete event after all processing (including ACE) is done
	// This ensures clients waiting for completion get all events before the signal
	a.emitter.Emit(events.Event{
		Type:      events.EventTurnComplete,
		Timestamp: time.Now(),
		Data:      events.TurnEventData{},
	})

	slog.Info("agent execution completed", "finish_reason", resp.FinishReason, "success", resp.Success)
	return resp, nil
}

// executeSetup validates the request and sets up the execution context.
func (a *Agent) executeSetup(ctx context.Context, req *AgentRequest) (context.Context, *AgentResponse, error) {
	if req == nil {
		return ctx, nil, spinerrors.New(spinerrors.CodeValidation, "Agent.Execute", "request cannot be nil", nil)
	}
	if req.Input == "" {
		return ctx, nil, spinerrors.New(spinerrors.CodeValidation, "Agent.Execute", "request input cannot be empty", nil)
	}

	// Create response
	resp := &AgentResponse{
		Success: true,
	}

	return ctx, resp, nil
}

// applyTimeout applies timeout to the context if needed.
func (a *Agent) applyTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	// If context already has a deadline, respect it
	if _, hasDeadline := ctx.Deadline(); hasDeadline {
		return ctx, func() {} // Return no-op cancel function
	}
	// Use config timeout (initialized from DefaultConfig: 60 minutes)
	return context.WithTimeout(ctx, a.config.Timeout)
}

// buildPrompt builds the initial prompt for the agent.
func (a *Agent) buildPrompt(req *AgentRequest) []message.Message {
	// Start with history messages if provided
	messages := make([]message.Message, 0, len(req.History)+1)
	if len(req.History) > 0 {
		messages = append(messages, req.History...)
	}

	// Add current user input
	messages = append(messages, message.Message{
		Role:    message.RoleUser,
		Content: req.Input,
	})

	return messages
}

// finalizeResponse finalizes the agent response.
func (a *Agent) finalizeResponse(resp *AgentResponse, messages []message.Message, historyLen int) {
	if len(messages) > historyLen {
		// Get the last assistant message
		for i := len(messages) - 1; i >= historyLen; i-- {
			if messages[i].Role == "assistant" {
				resp.Output = messages[i].Content
				break
			}
		}
	}

	// Note: EventTurnComplete is NOT emitted here - it's emitted after ACE bullet
	// generation to ensure all post-execution events are emitted before signaling completion
}

// BuildToolsForTask constructs the filtered tool list for the LLM request,
// based on the task mode's allowed tools.
//
// This method delegates to the orchestration service's tool registry.
func (a *Agent) BuildToolsForTask(task Task) ([]tools.Tool, error) {
	if a.orchestration == nil {
		return nil, nil
	}

	toolRegistry := a.orchestration.GetToolRegistry()
	if toolRegistry == nil {
		return nil, nil
	}

	// Get all available tools from registry
	allTools := toolRegistry.List()
	if len(allTools) == 0 {
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
	filtered := make([]tools.Tool, 0, len(allTools))
	for _, tool := range allTools {
		// Check if tool is allowed in this mode
		if !allowAllTools && !allowedSet[tool.Name()] {
			continue
		}

		filtered = append(filtered, tool)
	}

	slog.Debug("filtered tools for task",
		"task", task.Name(),
		"total", len(allTools),
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
	params := openai.ChatCompletionNewParams{
		Messages: openai.F([]openai.ChatCompletionMessageParamUnion{
			openai.UserMessage(decompositionPrompt),
		}),
		MaxTokens:   openai.F(int64(1000)),
		Temperature: openai.F(0.3), // Lower temperature for more consistent planning
	}

	resp, err := a.llm.Complete(ctx, params)
	if err != nil {
		return nil, spinerrors.New(spinerrors.CodeLLM, "Agent.CreatePlan", "llm completion failed", err)
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

	responseContent := getContent(resp)
	if err := json.Unmarshal([]byte(responseContent), &decomposition); err != nil {
		return nil, spinerrors.New(spinerrors.CodeLLM, "Agent.CreatePlan", "failed to parse LLM response", err)
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
		return nil, spinerrors.New(spinerrors.CodeValidation, "Agent.CreatePlan", "plan validation failed", err)
	}

	// Calculate total estimated duration
	// plan.EstimatedDuration is calculated by the method

	return plan, nil
}

// ShouldApprove determines if a command needs user approval.
//
// Returns:
//   - needsApproval: true if the command requires approval
//   - reason: explanation of why approval is needed
func (a *Agent) ShouldApprove(cmd *security.Command) (bool, string) {
	// If approval is disabled, never require approval
	if !a.config.RequireApproval {
		return false, ""
	}

	// Classify the command via security service
	result, err := a.security.ValidateCommand(cmd)
	if err != nil {
		// On error, require approval for safety
		return true, fmt.Sprintf("Classification error: %v", err)
	}

	switch result.Classification {
	case security.CommandSafe:
		return false, ""

	case security.CommandInteractive:
		return true, "This command may modify files or system state"

	case security.CommandDangerous:
		return true, fmt.Sprintf("WARNING: Dangerous operation - %s", result.Reason)

	case security.CommandForbidden:
		// Forbidden commands should never be executed, even with approval
		// This will be handled by the executor
		return false, fmt.Sprintf("BLOCKED: %s", result.Reason)

	case security.CommandUnverified:
		return true, "Unknown command, approval required for safety"

	default:
		return true, "Unknown command classification, approval required"
	}
}

// determineRequiresApproval determines if a tool call requires approval based on tool name and arguments.
// This is used to populate the RequiresApproval field in ToolCallStartData.
func (a *Agent) determineRequiresApproval(toolName string, args map[string]interface{}) bool {
	// Tools that always require approval
	requiresApprovalTools := map[string]bool{
		"execute_command": true,
		"write_file":      true,
		"apply_patch":     true,
	}

	if requiresApprovalTools[toolName] {
		return true
	}

	// For execute_command, also check if the command itself requires approval
	if toolName == "execute_command" {
		if cmd, ok := args["command"].(string); ok && cmd != "" {
			cmdStruct := &security.Command{Program: cmd}
			needsApproval, _ := a.ShouldApprove(cmdStruct)
			return needsApproval
		}
	}

	return false
}

// processToolCalls handles all tool calls from an LLM response.
// It adds the assistant message with tool calls, executes each tool,
// and adds tool result messages to the conversation.
// processToolCallsFromCompletion is a wrapper that extracts data from openai.ChatCompletion
// and delegates to processToolCalls.
func (a *Agent) processToolCallsFromCompletion(ctx context.Context, messages []message.Message, completion *openai.ChatCompletion, resp *AgentResponse) []message.Message {
	content := getContent(completion)
	toolCalls := getToolCalls(completion)
	return a.processToolCallsInternal(ctx, messages, content, toolCalls, resp)
}

// processToolCallsInternal contains the actual logic for processing tool calls.
func (a *Agent) processToolCallsInternal(ctx context.Context, messages []message.Message, content string, toolCalls []orchestration.ToolCall, resp *AgentResponse) []message.Message {
	// Create assistant message with tool calls
	assistantMsg := message.Message{
		Role:      message.RoleAssistant,
		Content:   content,
		Timestamp: time.Now(),
	}

	// Add assistant message FIRST (before tool results)
	messages = append(messages, assistantMsg)

	// Convert and process each tool call
	for i := range toolCalls {
		coreToolCall := &toolCalls[i]

		// Add to assistant message (note: message already appended above)
		msgToolCall := message.ToolCall{
			ID:   coreToolCall.ID,
			Type: coreToolCall.Type,
			Function: message.FunctionCall{
				Name:      coreToolCall.Function.Name,
				Arguments: coreToolCall.Function.Arguments,
			},
		}
		messages[len(messages)-1].ToolCalls = append(messages[len(messages)-1].ToolCalls, msgToolCall)

		// Process the tool call (ProcessToolCall will emit EventToolCallStart)
		toolResult, err := a.ProcessToolCall(ctx, coreToolCall)
		if err != nil {

			// Add error message to conversation (after assistant message)
			messages = append(messages, message.Message{
				Role: message.RoleTool,
				Content: fmt.Sprintf("Tool %s failed: %v",
					coreToolCall.Function.Name, err),
				ToolCallID: coreToolCall.ID,
				Timestamp:  time.Now(),
			})
		} else {

			// Add tool result to conversation (after assistant message)
			slog.Debug("Agent tool result", "tool", coreToolCall.Function.Name, "output_len", len(toolResult.Output), "success", toolResult.Success)
			messages = append(messages, message.Message{
				Role:       message.RoleTool,
				Content:    getToolResultContent(coreToolCall, toolResult),
				ToolCallID: coreToolCall.ID,
				Timestamp:  time.Now(),
			})

			// Track tool call in response
			resp.ToolCalls = append(resp.ToolCalls, *coreToolCall)
		}
	}

	return messages
}

// ProcessToolCall processes a single tool call from the LLM.
//
// This method validates the tool call, parses arguments, executes the appropriate
// tool based on the function name, and returns the result. It handles:
// - Command execution with approval workflow
// - File operations (read, write, list)
// - Event emission for tool lifecycle
// - Error handling and recovery
func (a *Agent) ProcessToolCall(ctx context.Context, call *orchestration.ToolCall) (*orchestration.ToolResult, error) {
	// 1. Validate tool call
	if err := a.validateToolCall(call); err != nil {
		return &orchestration.ToolResult{
			ID:      call.ID,
			Success: false,
			Error:   err,
		}, nil // Return nil error so agent continues
	}

	// 2. Parse arguments
	args, err := a.parseToolArguments(call)
	if err != nil {
		return &orchestration.ToolResult{
			ID:      call.ID,
			Success: false,
			Error:   err,
		}, nil
	}

	// 3. Emit tool start event
	// Determine if this tool requires approval
	requiresApproval := a.determineRequiresApproval(call.Function.Name, args.ToMap())

	a.emitter.Emit(events.Event{
		Type:      events.EventToolCallStart,
		Timestamp: time.Now(),
		Data: events.ToolCallStartData{
			ToolID:           call.ID,
			ToolName:         call.Function.Name,
			Parameters:       args,
			RequiresApproval: requiresApproval,
		},
	})

	// 4. Execute tool via orchestration service
	result, err := a.orchestration.ExecuteTool(ctx, call)
	if err != nil {
		slog.Error("tool execution failed via orchestration", "tool", call.Function.Name, "error", err)
		// If orchestration returns an error, create error result
		result = &orchestration.ToolResult{
			ID:      call.ID,
			Success: false,
			Error:   err,
		}
	}

	// 5. Emit completion event
	completion := events.ToolCallCompleteData{
		ToolID:   call.ID,
		ToolName: call.Function.Name,
		Success:  result.Success,
	}
	if result.Success {
		completion.Output = result.Output
		slog.Debug("tool execution succeeded", "tool", call.Function.Name, "output_len", len(result.Output))
	} else if result.Error != nil {
		completion.Error = result.Error.Error()
		slog.Warn("tool execution failed", "tool", call.Function.Name, "error", result.Error.Error())
	}
	a.emitter.Emit(events.Event{
		Type:      events.EventToolCallComplete,
		Timestamp: time.Now(),
		Data:      completion,
	})

	return result, nil // Always return nil error so agent continues
}

// validateToolCall validates the tool call structure.
func (a *Agent) validateToolCall(call *orchestration.ToolCall) error {
	if call == nil {
		return errors.New("tool call cannot be nil")
	}
	if call.ID == "" {
		return errors.New("tool call ID cannot be empty")
	}
	if call.Function.Name == "" {
		return errors.New("tool function name cannot be empty")
	}
	return nil
}

// parseToolArguments extracts and parses JSON arguments from tool call.
func (a *Agent) parseToolArguments(call *orchestration.ToolCall) (tools.ToolParameters, error) {
	parser := tools.NewStrictArgumentParser()
	return parser.Parse(call.Function.Arguments)
}

// getToolResultContent returns the appropriate content to send to LLM based on tool result.
// If tool succeeded, returns output. If failed, returns error message.
func getToolResultContent(toolCall *orchestration.ToolCall, result *orchestration.ToolResult) string {
	if result.Success {
		return result.Output
	}

	// Tool failed - send error message to LLM so it knows what went wrong
	if result.Error != nil {
		errorMsg := fmt.Sprintf("Tool %s failed: %v", toolCall.Function.Name, result.Error)
		slog.Debug("Tool failed, sending error to LLM", "tool", toolCall.Function.Name, "error", result.Error)
		return errorMsg
	}

	// Edge case: not successful but no error message
	return fmt.Sprintf("Tool %s failed with no error message", toolCall.Function.Name)
}

// extractQueryFromMessages extracts a query string from messages for ACE retrieval.
// Uses the most recent user message as the retrieval query.
func extractQueryFromMessages(messages []message.Message) string {
	// Search backwards for the most recent user message
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Role == "user" {
			return messages[i].Content
		}
	}
	return ""
}

// callLLM calls the LLM provider with the given messages and filtered tools based on task mode.
// The task parameter controls both tool filtering and token budget:
//   - Tools: Only tools in task.AllowedTools() are included
//   - Tokens: Uses task.MaxTokens() if > 0, otherwise agent.config.MaxTokens
//
// The bullets parameter contains ACE bullets already retrieved for this turn.
func (a *Agent) callLLM(ctx context.Context, messages []message.Message, task Task, bullets []*bullet.Bullet) (*openai.ChatCompletion, error) {
	// ACE: Emit event to show ACE activity if bullets were provided
	if a.aceService != nil && len(bullets) > 0 {
		// Build bullet list for display
		bulletList := ""
		for i, b := range bullets {
			bulletList += fmt.Sprintf("  %d. %s\n", i+1, b.Content)
		}

		aceMsg := fmt.Sprintf("ACE: Retrieved %d relevant bullet%s from playbook:\n%s", len(bullets), pluralize(len(bullets)), bulletList)

		a.emitter.Emit(events.Event{
			Type:      events.EventContentComplete,
			Timestamp: time.Now(),
			Data: events.ContentDeltaData{
				Content: aceMsg,
				Role:    string(message.RoleAssistant),
			},
		})
	}

	// Start with system message from task
	openaiMessages := make([]openai.ChatCompletionMessageParamUnion, 0, len(messages)+1)

	// Add system prompt with thinking instructions
	systemPrompt := task.SystemPrompt()
	if systemPrompt != "" {
		// Add thinking instructions to the system prompt
		enhancedSystemPrompt := systemPrompt + `

IMPORTANT: When you need to think through a problem or reason about your approach, wrap your thinking process in <think> and </think> tags. This helps users understand your reasoning process. For example:

<think>
I need to analyze this code to understand what it does. Let me break down the function step by step...
</think>

Then provide your response after the thinking block.`

		// ACE: Enhance system prompt with retrieved bullets
		if a.aceService != nil {
			acePrompt, err := a.aceService.BuildPrompt(ctx, enhancedSystemPrompt, bullets)
			if err != nil {
				slog.Warn("ACE prompt building failed", "error", err)
				// Fall back to non-ACE prompt
			} else {
				enhancedSystemPrompt = acePrompt
				slog.Debug("ACE enhanced system prompt", "bullets_count", len(bullets))
			}
		}

		openaiMessages = append(openaiMessages, openai.SystemMessage(enhancedSystemPrompt))
	}

	// Convert conversation messages to OpenAI format
	for _, msg := range messages {
		openaiMessages = append(openaiMessages, convertMessageToOpenAI(msg))
	}

	// Build filtered tool list for this task mode
	toolList, err := a.BuildToolsForTask(task)
	if err != nil {
		return nil, spinerrors.New(spinerrors.CodeInternal, "Agent.callLLM", "failed to build tools", err)
	}

	// Determine token budget: task overrides agent config
	maxTokens := a.config.MaxTokens
	if task != nil {
		taskMaxTokens := task.MaxTokens()
		if taskMaxTokens > 0 {
			maxTokens = taskMaxTokens
		}
	}

	// Build OpenAI request params
	params := openai.ChatCompletionNewParams{
		Messages:    openai.F(openaiMessages),
		Temperature: openai.F(a.config.Temperature),
		MaxTokens:   openai.F(int64(maxTokens)),
	}

	// Add tools if present
	if len(toolList) > 0 {
		params.Tools = openai.F(convertToolsToOpenAI(toolList))
	}

	slog.Debug("calling LLM", "tool_count", len(toolList), "message_count", len(openaiMessages))

	// Call LLM with streaming
	chunks, err := a.llm.Stream(ctx, params)
	if err != nil {
		return nil, spinerrors.New(spinerrors.CodeLLM, "Agent.callLLM", "failed to start LLM stream", err)
	}

	// Use ChatCompletionAccumulator to properly handle streaming chunks
	// This handles tool call accumulation by index correctly
	acc := openai.ChatCompletionAccumulator{}

	chunkCount := 0
	for chunk := range chunks {
		chunkCount++

		// Check context cancellation
		if err := ctx.Err(); err != nil {
			return nil, spinerrors.New(spinerrors.CodeTimeout, "Agent.callLLM", "context cancelled", err)
		}

		// Add chunk to accumulator - this handles proper merging of deltas
		acc.AddChunk(chunk)

		// Handle empty chunk (shouldn't happen but be safe)
		if len(chunk.Choices) == 0 {
			continue
		}

		choice := chunk.Choices[0]
		delta := choice.Delta

		// Emit content delta immediately for real-time streaming
		if delta.Content != "" {
			a.emitter.Emit(events.Event{
				Type:      events.EventContentDelta,
				Timestamp: time.Now(),
				Data: events.ContentDeltaData{
					Content: delta.Content,
					Role:    string(message.RoleAssistant),
				},
			})
			slog.Debug("received content chunk", "count", chunkCount, "content_len", len(delta.Content))
		}

		// Check if a tool call just finished being accumulated
		if toolCall, ok := acc.JustFinishedToolCall(); ok {
			slog.Debug("tool call finished", "index", toolCall.Index, "name", toolCall.Name, "args_len", len(toolCall.Arguments))
		}

		// Log finish reason when received
		if choice.FinishReason != "" {
			slog.Debug("received finish chunk", "finish_reason", choice.FinishReason, "total_chunks", chunkCount)
		}
	}

	// Get the accumulated response
	response := &acc.ChatCompletion

	// Check if we have any choices (may be empty on timeout/error)
	if len(response.Choices) == 0 {
		slog.Warn("stream ended with no choices", "total_chunks", chunkCount)
		return nil, spinerrors.New(spinerrors.CodeLLM, "Agent.callLLM", "no choices in response", nil)
	}

	slog.Debug("stream ended",
		"total_chunks", chunkCount,
		"content_len", len(response.Choices[0].Message.Content),
		"tool_calls", len(response.Choices[0].Message.ToolCalls))

	// Check if context was cancelled after stream ended
	if err := ctx.Err(); err != nil {
		return nil, spinerrors.New(spinerrors.CodeTimeout, "Agent.callLLM", "context cancelled", err)
	}

	// Fallback: if no finish reason was provided, set a default one
	if response.Choices[0].FinishReason == "" {
		if len(response.Choices[0].Message.ToolCalls) > 0 {
			response.Choices[0].FinishReason = openai.ChatCompletionChoicesFinishReasonToolCalls
		} else {
			response.Choices[0].FinishReason = openai.ChatCompletionChoicesFinishReasonStop
		}
	}

	// ACE: Parse feedback and update bullets if retrieved any
	if a.aceService != nil && len(bullets) > 0 {
		responseContent := response.Choices[0].Message.Content
		feedback, err := a.aceService.ParseFeedback(responseContent)
		if err != nil {
			slog.Warn("ACE feedback parsing failed", "error", err)
		} else if feedback != nil {
			// Update bullets asynchronously (based on config)
			if err := a.aceService.UpdateBullets(ctx, bullets, feedback); err != nil {
				slog.Warn("ACE bullet update failed", "error", err)
			} else {
				slog.Debug("ACE updated bullets", "helpful_count", len(feedback.HelpfulBullets), "harmful_count", len(feedback.HarmfulBullets))
			}
		}
	}

	return response, nil
}

// messageAdapter adapts message.Message to detection.Message interface
type messageAdapter struct {
	msg message.Message
}

func (m *messageAdapter) GetRole() string {
	return string(m.msg.Role)
}

func (m *messageAdapter) GetContent() string {
	return m.msg.Content
}

func (m *messageAdapter) GetTimestamp() time.Time {
	return m.msg.Timestamp
}

// pluralize returns "s" if count is not 1, otherwise returns empty string
func pluralize(count int) string {
	if count == 1 {
		return ""
	}
	return "s"
}

// emitACERetrievalEvent emits an ACE retrieval event with calculated metrics
func (a *Agent) emitACERetrievalEvent(
	trajCtx *trajectory.TrajectoryContext,
	trigger trajectory.TriggerType,
	query string,
	bulletsRetrieved int,
	turn int,
) {
	// Calculate cache hit rate
	total := trajCtx.CacheHits + trajCtx.CacheMisses
	hitRate := 0.0
	if total > 0 {
		hitRate = float64(trajCtx.CacheHits) / float64(total)
	}

	// BulletsNew is approximated by cache misses
	bulletsNew := trajCtx.CacheMisses

	a.emitter.Emit(events.Event{
		Type: events.EventACERetrieval,
		Data: events.ACERetrievalData{
			Turn:             turn,
			Trigger:          string(trigger),
			Query:            query,
			BulletsRetrieved: bulletsRetrieved,
			BulletsNew:       bulletsNew,
			CacheSize:        len(trajCtx.BulletCache),
			CacheHitRate:     hitRate,
		},
	})
}
