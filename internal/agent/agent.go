package agent

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/dmytrogajewski/spin/internal/ace/bullet"
	"github.com/dmytrogajewski/spin/internal/ace/trajectory"
	"github.com/dmytrogajewski/spin/internal/agent/sanitizer"
	"github.com/dmytrogajewski/spin/internal/agentsmd"
	"github.com/dmytrogajewski/spin/internal/detection"
	spinerrors "github.com/dmytrogajewski/spin/internal/errors"
	"github.com/dmytrogajewski/spin/internal/events"
	"github.com/dmytrogajewski/spin/internal/llm"
	"github.com/dmytrogajewski/spin/internal/message"
	"github.com/dmytrogajewski/spin/internal/planning"
	"github.com/dmytrogajewski/spin/internal/security"
	"github.com/dmytrogajewski/spin/internal/task"
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
	ErrNilLLM         = errors.New("LLM provider cannot be nil")
	ErrNilSecurity    = errors.New("security service cannot be nil")
	ErrNilDetection   = errors.New("detection service cannot be nil")
	ErrNilToolRuntime = errors.New("tool runtime cannot be nil")
	ErrNilPlanning    = errors.New("planning service cannot be nil")
	ErrNilContext     = errors.New("context cannot be nil")
	ErrNilEmitter     = errors.New("event emitter cannot be nil")
	ErrNilRequest     = errors.New("agent request cannot be nil")
	ErrEmptyInput     = errors.New("agent request input cannot be empty")
	ErrMaxTurns       = errors.New("maximum turns reached")
)

// Agent implements the core agent logic and decision-making loop.
//
// The Agent orchestrates the interaction between the LLM, tools, and execution
// environment. It processes user requests through multiple turns of LLM calls
// and tool executions until the task is complete or limits are reached.

type Agent struct {
	// Core LLM interaction
	llm llm.Provider

	// Service layers
	security        *security.SecurityService
	detection       *detection.DetectionService
	toolRuntime     *ToolRuntime
	planningService *planning.PlanningService
	aceService      *ACEService       // ACE (Agentic Context Engineering) - optional
	agentsMD        *agentsmd.Service // AGENTS.md project instructions - optional
	toolSelector    *ToolSelector     // Dynamic tool selection - optional

	// Infrastructure
	context     *Environment
	emitter     *events.EventEmitter
	planner     *planning.Plan
	planTracker *PlanTracker

	// Configuration (options-based)
	maxTurns        int
	timeout         time.Duration
	temperature     float64
	maxTokens       int
	requireApproval bool
	aceConfig       *ACEConfig
	cycleDetection  bool
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

// WithACEConfig sets the ACE configuration on the agent.
func WithACEConfig(aceConfig *ACEConfig) AgentOption {
	return func(a *Agent) error {
		a.aceConfig = aceConfig
		return nil
	}
}

// WithAgentsMDService sets the AGENTS.md service for the agent.
// If not provided, agent will operate without project-specific instructions.
func WithAgentsMDService(svc *agentsmd.Service) AgentOption {
	return func(a *Agent) error {
		a.agentsMD = svc
		return nil
	}
}

// WithToolSelector sets the dynamic tool selector for the agent.
// If not provided, agent will only use statically configured tools.
func WithToolSelector(selector *ToolSelector) AgentOption {
	return func(a *Agent) error {
		a.toolSelector = selector
		return nil
	}
}

// WithMaxTurns sets the maximum number of agent turns.
func WithMaxTurns(maxTurns int) AgentOption {
	return func(a *Agent) error {
		if maxTurns <= 0 {
			return spinerrors.Newf(spinerrors.CodeValidation, "Agent.WithMaxTurns", nil, "max turns must be positive, got %d", maxTurns)
		}
		a.maxTurns = maxTurns
		return nil
	}
}

// WithAgentTimeout sets the agent execution timeout.
func WithAgentTimeout(timeout time.Duration) AgentOption {
	return func(a *Agent) error {
		if timeout <= 0 {
			return spinerrors.Newf(spinerrors.CodeValidation, "Agent.WithAgentTimeout", nil, "timeout must be positive, got %v", timeout)
		}
		a.timeout = timeout
		return nil
	}
}

// WithTemperature sets the LLM temperature.
func WithTemperature(temperature float64) AgentOption {
	return func(a *Agent) error {
		if temperature < 0 || temperature > 2 {
			return spinerrors.Newf(spinerrors.CodeValidation, "Agent.WithTemperature", nil, "temperature must be between 0 and 2, got %f", temperature)
		}
		a.temperature = temperature
		return nil
	}
}

// WithMaxTokens sets the maximum tokens per LLM call.
func WithMaxTokens(maxTokens int) AgentOption {
	return func(a *Agent) error {
		if maxTokens <= 0 {
			return spinerrors.Newf(spinerrors.CodeValidation, "Agent.WithMaxTokens", nil, "max tokens must be positive, got %d", maxTokens)
		}
		a.maxTokens = maxTokens
		return nil
	}
}

// WithRequireApproval sets whether dangerous commands require approval.
func WithRequireApproval(require bool) AgentOption {
	return func(a *Agent) error {
		a.requireApproval = require
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
	runtime *ToolRuntime,
	planning *planning.PlanningService,
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
	if runtime == nil {
		return nil, ErrNilToolRuntime
	}
	if planning == nil {
		return nil, ErrNilPlanning
	}
	if context == nil {
		return nil, ErrNilContext
	}
	if emitter == nil {
		return nil, ErrNilEmitter
	}

	// Create agent with services and reasonable defaults
	agent := &Agent{
		llm:             provider,
		security:        security,
		detection:       detection,
		toolRuntime:     runtime,
		planningService: planning,
		context:         context,
		emitter:         emitter,
		maxTurns:        50,               // Default: 50 turns
		timeout:         60 * time.Minute, // Default: 60 minutes
		temperature:     0.7,              // Default: 0.7
		maxTokens:       8192,             // Default: 8K tokens
	}

	// Apply options
	for _, opt := range opts {
		if err := opt(agent); err != nil {
			return nil, spinerrors.New(spinerrors.CodeValidation, "Agent.NewAgent", "applying option failed", err)
		}
	}

	return agent, nil
}

// GetSecurityService returns the agent's security service.
// This allows access to the security service for updating approval handlers.
func (a *Agent) GetSecurityService() *security.SecurityService {
	return a.security
}

// GetToolRuntime returns the agent's tool runtime.
// This allows access to the tool registry for dynamic tool registration.
func (a *Agent) GetToolRuntime() *ToolRuntime {
	return a.toolRuntime
}

// SetApprovalService updates the approval service on the underlying orchestration service.
// This is useful when the approval service needs to be configured after agent creation.
func (a *Agent) SetApprovalService(service *security.ApprovalService) {
	if a.toolRuntime != nil {
		a.toolRuntime.SetApprovalService(service)
	}
}

// resolveTask determines which task to use for this request.
//
// The task must be provided in req.Task. If nil, returns an error.
// This simplification removes the runtime registry pattern in favor of
// compile-time task creation via task.NewTask().
func (a *Agent) resolveTask(req *AgentRequest) (task.Task, error) {
	if req.Task == nil {
		return nil, spinerrors.New(spinerrors.CodeValidation, "Agent.resolveTask", "task is required", nil)
	}

	slog.Debug("task resolution: using task", "name", req.Task.Name())
	return req.Task, nil
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
	taskName := ""
	if req.Task != nil {
		taskName = req.Task.Name()
	}
	slog.Info("agent execution started", "input_len", len(req.Input), "task_mode", taskName)
	// Validate request and setup
	ctx, resp, err := a.executeSetup(ctx, req)
	if err != nil {
		slog.Error("agent setup failed", "error", err)
		return resp, err
	}

	// Resolve task mode
	task, err := a.resolveTask(req)
	if err != nil {
		slog.Error("task resolution failed", "task_name", taskName, "error", err)
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

			// Only show learning messages if ACE events are enabled
			// This prevents noise for users who don't want to see ACE internals
			if len(learnedBullets) > 0 && a.aceConfig != nil && a.aceConfig.Retrieval.ProgressiveContext.EmitACEEvents {
				// Convert bullets to BulletData for event
				bulletData := make([]events.BulletData, len(learnedBullets))
				for i, b := range learnedBullets {
					bulletData[i] = events.BulletData{
						Content: b.Content,
						// Category is optional and not present in bullet.Bullet
					}
				}

				// Emit ACE learning event to show learned insights as compact hint
				a.emitter.Emit(events.Event{
					Type:      events.EventACELearned,
					Timestamp: time.Now(),
					Data: events.ACELearningData{
						Success: resp.Success,
						Bullets: bulletData,
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
	return context.WithTimeout(ctx, a.timeout)
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

		// Extract new messages from this turn (excluding history)
		// This includes: user input, assistant messages with tool calls, tool results, final assistant
		resp.Messages = messages[historyLen:]
	}

	// Note: EventTurnComplete is NOT emitted here - it's emitted after ACE bullet
	// generation to ensure all post-execution events are emitted before signaling completion
}

// GetPlanner returns the current execution planner.
// Returns nil if no planner has been set.
func (a *Agent) GetPlanner() *planning.Plan {
	return a.planner
}

// SetPlanner sets the execution planner.
func (a *Agent) SetPlanner(planner *planning.Plan) {
	a.planner = planner

	// Initialize plan tracker to monitor execution
	if planner != nil && a.emitter != nil {
		a.planTracker = NewPlanTracker(planner, a.emitter)
	}
}

// BuildToolsForTask constructs the filtered tool list for the LLM request,
// based on the task mode's allowed tools.
//
// This method delegates to the orchestration service's tool registry.
func (a *Agent) BuildToolsForTask(task task.Task) ([]tools.Tool, error) {
	if a.toolRuntime == nil {
		return nil, nil
	}

	toolRegistry := a.toolRuntime.Registry()
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

			// Check Agent-level approval flag first
			if !a.requireApproval {
				return false
			}

			// Validate command to check if forbidden (forbidden commands are blocked, not approved)
			result, err := a.security.ValidateCommand(cmdStruct)
			if err != nil {
				// On validation error, require approval for safety (fail-safe behavior)
				return true
			}

			// Forbidden commands are blocked, not approved
			if result.Classification == security.CommandForbidden {
				return false
			}

			// Use SecurityService to check if approval is needed
			return a.security.NeedsApproval(cmdStruct)
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
func (a *Agent) processToolCallsInternal(ctx context.Context, messages []message.Message, content string, toolCalls []ToolCall, resp *AgentResponse) []message.Message {
	// Create assistant message with tool calls
	assistantMsg := message.Message{
		Role:      message.RoleAssistant,
		Content:   content,
		Timestamp: time.Now(),
	}

	// Add assistant message FIRST (before tool results)
	messages = append(messages, assistantMsg)
	assistantMsgIdx := len(messages) - 1 // Capture index before adding tool results

	// Convert and process each tool call
	for i := range toolCalls {
		coreToolCall := &toolCalls[i]

		// Add to assistant message (use captured index, not len-1 which changes as we add tool results)
		msgToolCall := message.ToolCall{
			ID:   coreToolCall.ID,
			Type: coreToolCall.Type,
			Function: message.ToolCallFunction{
				Name:      coreToolCall.Function.Name,
				Arguments: coreToolCall.Function.Arguments,
			},
		}
		messages[assistantMsgIdx].ToolCalls = append(messages[assistantMsgIdx].ToolCalls, msgToolCall)

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
func (a *Agent) ProcessToolCall(ctx context.Context, call *ToolCall) (*ToolResult, error) {
	// 1. Validate tool call
	if err := a.validateToolCall(call); err != nil {
		result := tools.NewToolErrorWithID(call.ID, err)
		return &result, nil // Return nil error so agent continues
	}

	// 2. Parse arguments
	args, err := a.parseToolArguments(call)
	if err != nil {
		result := tools.NewToolErrorWithID(call.ID, err)
		return &result, nil
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

	// 4. Execute tool via runtime
	result, err := a.toolRuntime.Execute(ctx, call)
	if err != nil {
		slog.Error("tool execution failed", "tool", call.Function.Name, "error", err)
		errResult := tools.NewToolErrorWithID(call.ID, err)
		result = &errResult
	}

	// 5. Emit completion event
	completion := events.ToolCallCompleteData{
		ToolID:   call.ID,
		ToolName: call.Function.Name,
		Success:  result.Success,
		Metadata: result.Metadata,
	}
	if result.Success {
		completion.Output = result.Output
		slog.Debug("tool execution succeeded", "tool", call.Function.Name, "output_len", len(result.Output))
	} else if result.Err != nil {
		completion.Error = result.Err.Error()
		slog.Warn("tool execution failed", "tool", call.Function.Name, "error", result.Err.Error())
	} else if result.Error != "" {
		completion.Error = result.Error
		slog.Warn("tool execution failed", "tool", call.Function.Name, "error", result.Error)
	}
	a.emitter.Emit(events.Event{
		Type:      events.EventToolCallComplete,
		Timestamp: time.Now(),
		Data:      completion,
	})

	// 6. Update plan status if tracker is active
	if a.planTracker != nil {
		event := events.Event{
			Type:      events.EventToolCallComplete,
			Timestamp: time.Now(),
			Data:      completion,
		}
		a.planTracker.OnToolCallComplete(event)
	}

	return result, nil // Always return nil error so agent continues
}

// validateToolCall validates the tool call structure.
func (a *Agent) validateToolCall(call *ToolCall) error {
	return tools.ValidateToolCall(call)
}

// parseToolArguments extracts and parses JSON arguments from tool call.
func (a *Agent) parseToolArguments(call *ToolCall) (tools.ToolParameters, error) {
	parser := tools.NewStrictArgumentParser()
	return parser.Parse(call.Function.Arguments)
}

// getToolResultContent returns the appropriate content to send to LLM based on tool result.
// If tool succeeded, returns output. If failed, returns error message.
func getToolResultContent(toolCall *ToolCall, result *ToolResult) string {
	if result.Success {
		return result.Output
	}

	// Tool failed - send error message to LLM so it knows what went wrong
	// Check Err first (error type), then Error (string type) for backward compatibility
	if result.Err != nil {
		errorMsg := fmt.Sprintf("Tool %s failed: %v", toolCall.Function.Name, result.Err)
		slog.Debug("Tool failed, sending error to LLM", "tool", toolCall.Function.Name, "error", result.Err)
		return errorMsg
	}
	if result.Error != "" {
		errorMsg := fmt.Sprintf("Tool %s failed: %s", toolCall.Function.Name, result.Error)
		slog.Debug("Tool failed, sending error to LLM", "tool", toolCall.Function.Name, "error", result.Error)
		return errorMsg
	}

	// Edge case: not successful but no error message
	return fmt.Sprintf("Tool %s failed with no error message", toolCall.Function.Name)
}

// callLLM calls the LLM provider with the given messages and filtered tools based on task mode.
// The task parameter controls both tool filtering and token budget:
//   - Tools: Only tools in task.AllowedTools() are included
//   - Tokens: Uses task.MaxTokens() if > 0, otherwise agent.config.MaxTokens
//
// The bullets parameter contains ACE bullets already retrieved for this turn.
// Note: ACE bullet display is now handled by EventACERetrieval emission in loop.go
func (a *Agent) callLLM(ctx context.Context, messages []message.Message, t task.Task, bullets []*bullet.Bullet) (*openai.ChatCompletion, error) {
	// Start with system message from task
	openaiMessages := make([]openai.ChatCompletionMessageParamUnion, 0, len(messages)+1)

	// Build system prompt with proper layering:
	// 1. AGENTS.md project instructions (if available)
	// 2. Task system prompt
	// 3. Thinking instructions
	// 4. ACE bullets (if enabled)
	var promptBuilder strings.Builder

	// 1. AGENTS.md project instructions (placed first for context)
	if a.agentsMD != nil && a.agentsMD.IsLoaded() {
		promptBuilder.WriteString("# Project Instructions\n\n")
		promptBuilder.WriteString(a.agentsMD.Content())
		promptBuilder.WriteString("\n\n---\n\n")
		slog.Debug("injected AGENTS.md into system prompt", "path", a.agentsMD.Path(), "size", len(a.agentsMD.Content()))
	}

	// 2. Task system prompt
	systemPrompt := t.SystemPrompt()
	if systemPrompt != "" {
		promptBuilder.WriteString(systemPrompt)
	}

	// 3. Thinking instructions
	if promptBuilder.Len() > 0 {
		promptBuilder.WriteString(`

IMPORTANT: When you need to think through a problem or reason about your approach, wrap your thinking process in <think> and </think> tags. This helps users understand your reasoning process. For example:

<think>
I need to analyze this code to understand what it does. Let me break down the function step by step...
</think>

Then provide your response after the thinking block.`)
	}

	enhancedSystemPrompt := promptBuilder.String()

	// 4. ACE: Enhance system prompt with retrieved bullets
	if enhancedSystemPrompt != "" {
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
	toolList, err := a.BuildToolsForTask(t)
	if err != nil {
		return nil, spinerrors.New(spinerrors.CodeInternal, "Agent.callLLM", "failed to build tools", err)
	}

	// Determine token budget: task overrides agent config
	maxTokens := a.maxTokens
	if t != nil {
		taskMaxTokens := t.MaxTokens()
		if taskMaxTokens > 0 {
			maxTokens = taskMaxTokens
		}
	}

	// Build OpenAI request params
	params := openai.ChatCompletionNewParams{
		Messages:    openai.F(openaiMessages),
		Temperature: openai.F(a.temperature),
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

	// Initialize sanitizer for content stream
	streamSanitizer := sanitizer.New()

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
		// Use sanitizer to filter out protocol artifacts and separate thoughts
		if delta.Content != "" {
			content, thought := streamSanitizer.Process(delta.Content)

			if content != "" {
				a.emitter.Emit(events.Event{
					Type:      events.EventContentDelta,
					Timestamp: time.Now(),
					Data: events.ContentDeltaData{
						Content: content,
						Role:    string(message.RoleAssistant),
					},
				})
				slog.Debug("received content chunk", "count", chunkCount, "content_len", len(content))
			}

			if thought != "" {
				a.emitter.Emit(events.Event{
					Type:      events.EventThinkingDelta,
					Timestamp: time.Now(),
					Data: events.ThinkingDeltaData{
						Content: thought,
					},
				})
				slog.Debug("received thinking chunk", "count", chunkCount, "content_len", len(thought))
			}
		}

		// Check if a tool call just finished being accumulated
		toolCall, finished := acc.JustFinishedToolCall()
		if !finished && choice.FinishReason == openai.ChatCompletionChunkChoicesFinishReasonToolCalls {
			finished = true
		}

		if finished {
			if toolCall.Name != "" {
				slog.Debug("tool call finished", "index", toolCall.Index, "name", toolCall.Name, "args_len", len(toolCall.Arguments))
			}
			// Plan detection logic
			// If we haven't set a planner yet, check the accumulated content for a plan.
			// This handles the "Thinking/Planning -> ToolCall" pattern synchronously.
			if a.planner == nil {
				if len(acc.Choices) > 0 {
					content := acc.Choices[0].Message.Content
					if content != "" {
						plan := planning.DetectPlanFromText(content)
						if plan != nil {
							a.SetPlanner(plan)
							// Manually emit EventPlanUpdate so ACP agent sees it
							a.emitter.Emit(events.Event{
								Type:      events.EventPlanUpdate,
								Timestamp: time.Now(),
								Data: events.PlanUpdateData{
									Plan: plan,
								},
							})
						}
					}
				}
			}

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
		if chunkCount == 0 {
			return nil, spinerrors.New(spinerrors.CodeLLM, "Agent.callLLM", "stream returned no chunks - possible connection error or empty response from LLM", nil)
		}
		return nil, spinerrors.New(spinerrors.CodeLLM, "Agent.callLLM", "no choices in response after processing chunks", nil)
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

	// XML Tool Call Recovery:
	// If no structured tool calls were detected, check if the LLM output tool calls as XML in content.
	// This handles models like Qwen that may fallback to XML output.
	if len(response.Choices[0].Message.ToolCalls) == 0 {
		content := response.Choices[0].Message.Content
		xmlToolCalls := parseToolCallsFromXML(content)
		if len(xmlToolCalls) > 0 {
			slog.Info("detected XML tool calls in content", "count", len(xmlToolCalls))
			response.Choices[0].Message.ToolCalls = xmlToolCalls
			// Force finish reason to tool_calls so the loop processes them
			response.Choices[0].FinishReason = openai.ChatCompletionChoicesFinishReasonToolCalls

			// Clean up content to remove XML tags (avoid duplication in history)
			// We reconstruct content preserving thoughts but removing function tags
			s := sanitizer.New()
			cleanContent, cleanThought := s.Process(content)

			var sb strings.Builder
			if cleanThought != "" {
				sb.WriteString("<think>")
				sb.WriteString(cleanThought)
				sb.WriteString("</think>\n")
			}
			sb.WriteString(cleanContent)
			response.Choices[0].Message.Content = sb.String()
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

// emitACERetrievalEvent emits an ACE retrieval event with calculated metrics
func (a *Agent) emitACERetrievalEvent(
	trajCtx *trajectory.TrajectoryContext,
	trigger trajectory.TriggerType,
	query string,
	bullets []*bullet.Bullet,
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

	// Convert bullets to BulletData for event
	bulletData := make([]events.BulletData, len(bullets))
	for i, b := range bullets {
		category := ""
		if b.Tags != nil {
			category = b.Tags["category"]
		}
		bulletData[i] = events.BulletData{
			Content:  b.Content,
			Category: category,
		}
	}

	a.emitter.Emit(events.Event{
		Type: events.EventACERetrieval,
		Data: events.ACERetrievalData{
			Turn:             turn,
			Trigger:          string(trigger),
			Query:            query,
			BulletsRetrieved: len(bullets),
			BulletsNew:       bulletsNew,
			CacheSize:        len(trajCtx.BulletCache),
			CacheHitRate:     hitRate,
			Bullets:          bulletData,
		},
	})
}
