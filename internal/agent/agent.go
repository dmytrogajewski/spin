package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/dmytrogajewski/spin/internal/cycle"
	"github.com/dmytrogajewski/spin/internal/detection"
	"github.com/dmytrogajewski/spin/internal/events"
	"github.com/dmytrogajewski/spin/internal/llm"
	"github.com/dmytrogajewski/spin/internal/orchestration"
	"github.com/dmytrogajewski/spin/internal/security"
	"github.com/dmytrogajewski/spin/internal/tools"
	"github.com/dmytrogajewski/spin/internal/types"
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

// WithMaxTurns sets the maximum number of agent turns.
func WithMaxTurns(maxTurns int) AgentOption {
	return func(a *Agent) error {
		if maxTurns <= 0 {
			return fmt.Errorf("max turns must be positive, got %d", maxTurns)
		}
		a.config.MaxTurns = maxTurns
		return nil
	}
}

// WithAgentTimeout sets the agent execution timeout.
func WithAgentTimeout(timeout time.Duration) AgentOption {
	return func(a *Agent) error {
		if timeout <= 0 {
			return fmt.Errorf("timeout must be positive, got %v", timeout)
		}
		a.config.Timeout = timeout
		return nil
	}
}

// WithTemperature sets the LLM temperature.
func WithTemperature(temperature float64) AgentOption {
	return func(a *Agent) error {
		if temperature < 0 || temperature > 2 {
			return fmt.Errorf("temperature must be between 0 and 2, got %f", temperature)
		}
		a.config.Temperature = temperature
		return nil
	}
}

// WithMaxTokens sets the maximum tokens per LLM call.
func WithMaxTokens(maxTokens int) AgentOption {
	return func(a *Agent) error {
		if maxTokens <= 0 {
			return fmt.Errorf("max tokens must be positive, got %d", maxTokens)
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

		// Convert ParameterSchema to map (defined below)
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

// processToolCalls handles all tool calls from an LLM response.
// It adds the assistant message with tool calls, executes each tool,
// and adds tool result messages to the conversation.
func (a *Agent) processToolCalls(ctx context.Context, messages []Message, llmResp *llm.CompletionResponse, resp *AgentResponse) []Message {
	// Create assistant message with tool calls
	assistantMsg := Message{
		Role:      RoleAssistant,
		Content:   llmResp.Content,
		Timestamp: time.Now(),
	}

	// Add assistant message FIRST (before tool results)
	messages = append(messages, assistantMsg)

	// Convert and process each tool call
	for i := range llmResp.ToolCalls {
		toolCall := &llmResp.ToolCalls[i]

		// Convert llm.ToolCall to orchestration.ToolCall
		coreToolCall := &orchestration.ToolCall{
			ID:   toolCall.ID,
			Type: toolCall.Type,
			Function: orchestration.ToolCallFunction{
				Name:      toolCall.Function.Name,
				Arguments: toolCall.Function.Arguments,
			},
		}

		// Add to assistant message (note: message already appended above)
		messages[len(messages)-1].ToolCalls = append(messages[len(messages)-1].ToolCalls, *coreToolCall)

		// Process the tool call (ProcessToolCall will emit EventToolCallStart)
		toolResult, err := a.ProcessToolCall(ctx, coreToolCall)
		if err != nil {

			// Add error message to conversation (after assistant message)
			messages = append(messages, Message{
				Role: RoleTool,
				Content: fmt.Sprintf("Tool %s failed: %v",
					coreToolCall.Function.Name, err),
				ToolCallID: coreToolCall.ID,
				Timestamp:  time.Now(),
			})
		} else {

			// Add tool result to conversation (after assistant message)
			slog.Debug("Agent tool result", "tool", coreToolCall.Function.Name, "output_len", len(toolResult.Output), "success", toolResult.Success)
			messages = append(messages, Message{
				Role:       RoleTool,
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
	// Convert args to ToolCallArguments
	toolArgs, _ := types.FromMap(args)

	a.emitter.Emit(events.Event{
		Type:      events.EventToolCallStart,
		Timestamp: time.Now(),
		Data: events.ToolCallStartData{
			ToolID:     call.ID,
			ToolName:   call.Function.Name,
			Parameters: toolArgs,
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
func (a *Agent) parseToolArguments(call *orchestration.ToolCall) (map[string]interface{}, error) {
	if call.Function.Arguments == "" {
		return nil, errors.New("tool arguments cannot be empty")
	}

	var args map[string]interface{}
	if err := json.Unmarshal([]byte(call.Function.Arguments), &args); err != nil {
		return nil, fmt.Errorf("failed to parse tool arguments: %w", err)
	}

	return args, nil
}

// convertParameterSchemaToMap converts a ParameterSchema struct to map[string]interface{}.
// This is needed because LLM providers expect parameters as a JSON-compatible map.
func convertParameterSchemaToMap(params tools.ParameterSchema) map[string]interface{} {
	// Convert struct to map via JSON marshaling
	data, err := json.Marshal(params)
	if err != nil {
		// Fallback to empty object if marshaling fails
		return map[string]interface{}{"type": "object"}
	}

	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		// Fallback to empty object if unmarshaling fails
		return map[string]interface{}{"type": "object"}
	}

	return result
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

// selectIntervention chooses the appropriate intervention based on cycle type and turn count
func (a *Agent) selectIntervention(cycleType cycle.CycleType, turnCount int) cycle.Intervention {
	// Escalation ladder based on turn count
	switch {
	case turnCount < 10:
		// Early cycles: Use soft intervention (reflection)
		return &cycle.ReflectionIntervention{}

	case turnCount < 30:
		// Mid-stage cycles: Use medium intervention (context summarization)
		// For now, use reflection as fallback since summarization needs compressor integration
		return &cycle.ReflectionIntervention{}

	default:
		// Late-stage/persistent cycles: Escalate to user
		return &cycle.EscalateIntervention{
			Emitter: &eventEmitterAdapter{emitter: a.emitter},
		}
	}
}

// eventEmitterAdapter adapts events.EventEmitter to cycle.EventEmitter interface
type eventEmitterAdapter struct {
	emitter *events.EventEmitter
}

func (a *eventEmitterAdapter) Emit(event cycle.Event) {
	// Convert cycle.Event to events.Event
	// Map event type based on string value
	var eventType events.EventType
	switch event.GetType() {
	case "turn_paused":
		eventType = events.EventTurnPaused
	default:
		eventType = events.EventWarning // fallback
	}

	coreEvent := events.Event{
		Type:      eventType,
		Timestamp: event.GetTimestamp(),
		Data:      event.GetData(),
	}
	a.emitter.Emit(coreEvent)
}

// extractToolNames extracts tool calls with parameters from LLM tool calls for cycle detection
// Returns strings in format "tool_name(arguments_json)" to enable parameter-aware cycle detection
func extractToolNames(toolCalls []llm.ToolCall) []string {
	calls := make([]string, len(toolCalls))
	for i, tc := range toolCalls {
		// Include both name and arguments for accurate cycle detection
		// This prevents false positives when same tool is called with different params
		// e.g., "list_directory(.)" vs "list_directory(advanced-features-20251012)"
		calls[i] = tc.Function.Name + "(" + tc.Function.Arguments + ")"
	}
	return calls
}

// executeAgentLoop runs the main agent execution loop.
func (a *Agent) executeAgentLoop(ctx context.Context, messages []Message, task Task, resp *AgentResponse) ([]Message, *AgentResponse, error) {
	maxTurns := a.config.MaxTurns

	for turn := 0; turn < maxTurns; turn++ {
		// Check context cancellation
		if err := ctx.Err(); err != nil {
			resp.FinishReason = "timeout"
			return messages, resp, err
		}

		a.emitTurnStart(turn + 1)

		// Call LLM
		llmResp, err := a.callLLM(ctx, messages, task)
		if err != nil {
			resp.Error = fmt.Errorf("LLM call failed: %w", err)
			resp.FinishReason = "error"
			return messages, resp, err
		}

		// Handle cycle detection via detection service
		if a.config.CycleDetection.Enabled {
			var shouldStop bool
			var err error
			messages, shouldStop, err = a.handleCycleDetection(ctx, messages, llmResp, turn+1, resp)
			if err != nil {
				return messages, resp, err
			}
			if shouldStop {
				return messages, resp, nil
			}
		}

		// Process tool calls or finish
		if len(llmResp.ToolCalls) > 0 {
			slog.Debug("processing tool calls", "count", len(llmResp.ToolCalls), "turn", turn+1)
			messages = a.processToolCalls(ctx, messages, llmResp, resp)
			continue
		}

		messages = a.addFinalMessage(messages, llmResp.Content)
		resp.FinishReason = llmResp.FinishReason
		if resp.FinishReason == "" {
			resp.FinishReason = "stop"
		}
		break
	}

	return messages, resp, nil
}

// handleCycleDetection processes cycle detection and interventions via detection service.
// Returns the modified messages (with intervention added if applicable), whether to stop, and any error.
func (a *Agent) handleCycleDetection(ctx context.Context, messages []Message, llmResp *llm.CompletionResponse, turn int, resp *AgentResponse) ([]Message, bool, error) {
	snapshot := cycle.Snapshot{
		Turn:      turn,
		Response:  llmResp.Content,
		ToolCalls: extractToolNames(llmResp.ToolCalls),
		Error:     "",
		Timestamp: time.Now(),
	}
	a.detection.RecordSnapshot(snapshot)

	cycleResult, err := a.detection.CheckCycle()
	if err != nil || cycleResult.Type == cycle.CycleNone {
		return messages, false, nil
	}

	intervention := a.selectIntervention(cycleResult.Type, turn)
	if intervention == nil {
		return messages, false, nil
	}

	// Convert messages to cycle.Message interface
	cycleMessages := make([]cycle.Message, len(messages))
	for i, msg := range messages {
		cycleMessages[i] = &messageAdapter{msg: msg}
	}

	modifiedCycleMessages, err := intervention.Apply(ctx, cycleMessages)
	if err != nil {
		slog.Warn("cycle intervention failed", "error", err, "cycle_type", cycleResult.Type)
		return messages, false, nil
	}

	// Convert back to Message slice
	messages = make([]Message, len(modifiedCycleMessages))
	for i, msg := range modifiedCycleMessages {
		messages[i] = Message{
			Role:      msg.GetRole(),
			Content:   msg.GetContent(),
			Timestamp: msg.GetTimestamp(),
		}
	}

	// Emit cycle detection event
	a.emitter.Emit(events.Event{
		Type:      events.EventWarning,
		Timestamp: time.Now(),
		Data: events.SystemEventData{
			Level:   "warning",
			Message: fmt.Sprintf("Cycle detected: %s. Applied intervention: %s", cycleResult.Type, intervention.Name()),
			Details: cycleResult.Details,
		},
	})

	// If this was an escalation intervention, pause the agent
	if intervention.Severity() >= 3 {
		resp.FinishReason = "cycle_intervention"
		return messages, true, nil
	}

	return messages, false, nil
}

// emitTurnStart emits a turn start event.
func (a *Agent) emitTurnStart(turn int) {
	a.emitter.Emit(events.Event{
		Type:      events.EventTurnStart,
		Timestamp: time.Now(),
		Data: events.TurnEventData{
			Turn: turn,
		},
	})
}

// callLLM calls the LLM provider with the given messages and filtered tools based on task mode.
// The task parameter controls both tool filtering and token budget:
//   - Tools: Only tools in task.AllowedTools() are included
//   - Tokens: Uses task.MaxTokens() if > 0, otherwise agent.config.MaxTokens
func (a *Agent) callLLM(ctx context.Context, messages []Message, task Task) (*llm.CompletionResponse, error) {
	// Start with system message from task
	llmMessages := make([]llm.Message, 0, len(messages)+1)

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

		llmMessages = append(llmMessages, llm.Message{
			Role:    "system",
			Content: enhancedSystemPrompt,
		})
	}

	// Convert conversation messages to LLM format
	for _, msg := range messages {
		llmMsg := llm.Message{
			Role:       string(msg.Role),
			Content:    msg.Content,
			ToolCallID: msg.ToolCallID,
		}

		// Convert tool calls if present
		if len(msg.ToolCalls) > 0 {
			llmMsg.ToolCalls = make([]llm.ToolCall, len(msg.ToolCalls))
			for j, tc := range msg.ToolCalls {
				llmMsg.ToolCalls[j] = llm.ToolCall{
					ID:   tc.ID,
					Type: tc.Type,
					Function: llm.FunctionCall{
						Name:      tc.Function.Name,
						Arguments: tc.Function.Arguments,
					},
				}
			}
		}

		llmMessages = append(llmMessages, llmMsg)
	}

	// Build filtered tool list for this task mode
	tools, err := a.BuildToolsForTask(task)
	if err != nil {
		return nil, fmt.Errorf("failed to build tools: %w", err)
	}

	// Determine token budget: task overrides agent config
	maxTokens := a.config.MaxTokens
	if task != nil {
		taskMaxTokens := task.MaxTokens()
		if taskMaxTokens > 0 {
			maxTokens = taskMaxTokens
		}
	}

	// Build LLM request with filtered tools
	req := llm.CompletionRequest{
		Messages:    llmMessages,
		Temperature: a.config.Temperature,
		MaxTokens:   maxTokens,
		Tools:       tools,
	}
	slog.Debug("calling LLM", "tool_count", len(tools), "message_count", len(llmMessages))

	// Call LLM with streaming
	chunks, err := a.llm.Stream(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to start LLM stream: %w", err)
	}

	// Accumulate response from streaming chunks
	response := &llm.CompletionResponse{
		Content:      "",
		ToolCalls:    []llm.ToolCall{},
		Usage:        llm.Usage{},
		FinishReason: "",
	}

	for chunk := range chunks {
		if chunk.Error != nil {
			return nil, fmt.Errorf("stream error: %w", chunk.Error)
		}

		// Accumulate content
		response.Content += chunk.Content

		// Emit content delta immediately for real-time streaming
		if chunk.Content != "" {
			a.emitter.Emit(events.Event{
				Type:      events.EventContentDelta,
				Timestamp: time.Now(),
				Data: events.ContentDeltaData{
					Content: chunk.Content,
					Role:    "assistant",
				},
			})
		}

		// Accumulate tool calls
		if chunk.ToolCall != nil {
			response.ToolCalls = append(response.ToolCalls, *chunk.ToolCall)
		}

		// Update finish reason
		if chunk.FinishReason != "" {
			response.FinishReason = chunk.FinishReason
		}
	}

	return response, nil
}

// addFinalMessage adds the final assistant message to the messages array.
// This is called when the agent is done (no more tool calls).
func (a *Agent) addFinalMessage(messages []Message, content string) []Message {
	if content != "" {
		messages = append(messages, Message{
			Role:      RoleAssistant,
			Content:   content,
			Timestamp: time.Now(),
		})
	}
	return messages
}

// messageAdapter adapts agent.Message to cycle.Message interface
type messageAdapter struct {
	msg Message
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
