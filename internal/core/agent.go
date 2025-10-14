package core

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/dmytrogajewski/spin/internal/core/cycle"
	"github.com/dmytrogajewski/spin/internal/core/task"
	"github.com/dmytrogajewski/spin/internal/llm"
	"github.com/dmytrogajewski/spin/internal/tools"
	"github.com/dmytrogajewski/spin/internal/types"
	"github.com/google/uuid"
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
	llm             llm.Provider    // LLM provider interface
	executor        *Executor       // Command executor
	validator       *Validator      // Command validator
	context         *Environment    // Environment context
	emitter         *EventEmitter   // Event emitter
	config          *AgentConfig    // Agent configuration
	toolRegistry    *tools.Registry // Tool registry
	taskRegistry    *task.Registry  // Task registry for execution modes
	approvalHandler ApprovalHandler // Approval handler for user approval requests
	cycleDetector   *cycle.Detector // Cycle detection and intervention
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

// AgentConfig contains agent configuration options.
type AgentConfig struct {
	// MaxTurns is the maximum number of agent turns (default: 50)
	MaxTurns int

	// Timeout is the execution timeout (default: 5min)
	Timeout time.Duration

	// Temperature is the LLM temperature (default: 0.7)
	Temperature float64

	// MaxTokens is the max tokens per LLM call (default: 4096)
	MaxTokens int

	// RequireApproval determines if dangerous commands need approval
	RequireApproval bool

	// ApprovalTimeout is the maximum time to wait for approval response (default: 60s)
	ApprovalTimeout time.Duration

	// CycleDetection configures automatic cycle detection and intervention
	CycleDetection struct {
		// Enabled controls whether cycle detection is active (default: true)
		Enabled bool

		// WindowSize is the number of snapshots to compare for pattern detection (default: 3)
		WindowSize int

		// SimilarityThresh is the threshold for response similarity detection (default: 0.8)
		SimilarityThresh float64

		// ToolRepeatLimit is the max identical tool calls before triggering cycle (default: 3)
		ToolRepeatLimit int

		// ErrorRepeatLimit is the max identical errors before triggering cycle (default: 3)
		ErrorRepeatLimit int
	}
}

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

// AgentOption is a functional option for configuring the Agent.
type AgentOption func(*Agent) error

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

	// Create cycle detection config
	cycleConfig := cycle.Config{
		WindowSize:       3,
		SimilarityThresh: 0.8,
		ToolRepeatLimit:  3,
		ErrorRepeatLimit: 3,
		Enabled:          true,
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
		config: &AgentConfig{
			MaxTurns:        DefaultMaxTurns,
			Timeout:         DefaultAgentTimeout,
			Temperature:     DefaultTemperature,
			MaxTokens:       DefaultMaxTokens,
			RequireApproval: false,
			CycleDetection: struct {
				Enabled          bool
				WindowSize       int
				SimilarityThresh float64
				ToolRepeatLimit  int
				ErrorRepeatLimit int
			}{
				Enabled:          true,
				WindowSize:       3,
				SimilarityThresh: 0.8,
				ToolRepeatLimit:  3,
				ErrorRepeatLimit: 3,
			},
		},
	}

	// Apply options
	for _, opt := range opts {
		if err := opt(agent); err != nil {
			return nil, fmt.Errorf("applying option: %w", err)
		}
	}

	return agent, nil
}

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

// WithApprovalHandler sets the approval handler for the agent.
// The handler is called when a command requires user approval.
// If no handler is set, commands requiring approval are automatically denied.
func WithApprovalHandler(handler ApprovalHandler) AgentOption {
	return func(a *Agent) error {
		a.approvalHandler = handler
		return nil
	}
}

// WithToolRegistry merges a custom tool registry with the agent's default tools.
// Custom tools will override default tools with the same name.
// This ensures default tools (execute_command, get_context, etc.) are always available.
func WithToolRegistry(registry *tools.Registry) AgentOption {
	return func(a *Agent) error {
		if registry == nil {
			return errors.New("tool registry cannot be nil")
		}

		// Merge custom tools into the agent's existing registry
		// Custom tools override defaults with the same name
		for _, tool := range registry.List() {
			if err := a.toolRegistry.RegisterOrReplace(tool); err != nil {
				return fmt.Errorf("failed to register tool %s: %w", tool.Name(), err)
			}
		}

		return nil
	}
}

// WithTaskRegistry sets a custom task registry for the agent.
// This replaces the default registry with all built-in modes (regular, review, compact, planning).
// Use this option to provide custom task modes or override default behavior.
//
// Example:
//
//	customRegistry := task.NewRegistry()
//	customRegistry.Register("custom", myCustomTask)
//	agent := NewAgent(llm, exec, val, ctx, emitter, WithTaskRegistry(customRegistry))
func WithTaskRegistry(registry *task.Registry) AgentOption {
	return func(a *Agent) error {
		if registry == nil {
			return errors.New("task registry cannot be nil")
		}
		a.taskRegistry = registry
		return nil
	}
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
	// Validate request first (before accessing req fields)
	if req == nil {
		return nil, ErrNilRequest
	}

	// Start tracing span (after validation)
	ctx, span := StartSpan(ctx, "Agent.Execute",
		StringAttr("input_length", fmt.Sprintf("%d", len(req.Input))),
		IntAttr("max_turns", a.config.MaxTurns),
	)
	defer span.End()

	// Validate input
	if req.Input == "" {
		span.SetError(ErrEmptyInput)
		return nil, ErrEmptyInput
	}

	// Resolve task mode for this request
	task, err := a.resolveTask(req)
	if err != nil {
		span.SetError(err)
		return nil, fmt.Errorf("failed to resolve task mode: %w", err)
	}
	span.SetAttribute("task_mode", task.Name())

	// Apply timeout from config if not already set
	var cancel context.CancelFunc
	if _, ok := ctx.Deadline(); !ok {
		ctx, cancel = context.WithTimeout(ctx, a.config.Timeout)
		defer cancel()
	}

	// Initialize response
	resp := &AgentResponse{
		Messages:    make([]Message, 0),
		ToolCalls:   make([]*ToolCall, 0),
		ToolResults: make([]*ToolResult, 0),
		TurnsUsed:   0,
		TokensUsed:  0,
	}

	// Build initial prompt
	messages := a.buildPrompt(req)
	historyLen := len(messages)

	// Agent loop
	maxTurns := a.config.MaxTurns
	for turn := 0; turn < maxTurns; turn++ {
		// Check context cancellation
		select {
		case <-ctx.Done():
			resp.FinishReason = "timeout"
			return resp, ctx.Err()
		default:
		}

		// Increment turn counter
		resp.TurnsUsed = turn + 1

		// Emit turn start event
		a.emitter.Emit(Event{
			Type:      EventTurnStart,
			Timestamp: time.Now(),
			Data: TurnEventData{
				Turn: turn + 1,
			},
		})

		llmResp, err := a.callLLM(ctx, messages, task)
		if err != nil {
			resp.Error = fmt.Errorf("LLM call failed: %w", err)
			resp.FinishReason = "error"
			return resp, err
		}

		resp.Content += llmResp.Content
		resp.TokensUsed += llmResp.Usage.TotalTokens

		// Record snapshot for cycle detection (before processing tool calls)
		if a.config.CycleDetection.Enabled && a.cycleDetector != nil {
			snapshot := cycle.Snapshot{
				Turn:      turn + 1,
				Response:  llmResp.Content,
				ToolCalls: extractToolNames(llmResp.ToolCalls),
				Error:     "", // No error at this point
				Timestamp: time.Now(),
			}
			a.cycleDetector.Record(snapshot)

			// Check for cycles and apply interventions
			if cycleResult, err := a.cycleDetector.Check(); err == nil && cycleResult.Type != cycle.CycleNone {
				// Apply intervention
				intervention := a.selectIntervention(cycleResult.Type, turn+1)
				if intervention != nil {
					// Convert messages to cycle.Message interface
					cycleMessages := make([]cycle.Message, len(messages))
					for i, msg := range messages {
						cycleMessages[i] = msg
					}

					modifiedCycleMessages, err := intervention.Apply(ctx, cycleMessages)
					if err != nil {
						slog.Warn("cycle intervention failed", "error", err, "cycle_type", cycleResult.Type)
					} else {
						// Convert back to core.Message slice
						messages = make([]Message, len(modifiedCycleMessages))
						for i, msg := range modifiedCycleMessages {
							// Convert interface back to concrete Message type
							messages[i] = Message{
								Role:      Role(msg.GetRole()),
								Content:   msg.GetContent(),
								Timestamp: msg.GetTimestamp(),
							}
						}

						// Emit cycle detection event
						a.emitter.Emit(Event{
							Type:      EventWarning,
							Timestamp: time.Now(),
							Data: SystemEventData{
								Level:   "warning",
								Message: fmt.Sprintf("Cycle detected: %s. Applied intervention: %s", cycleResult.Type, intervention.Name()),
								Details: cycleResult.Details,
							},
						})

						// If this was an escalation intervention, pause the agent
						if intervention.Severity() >= 3 {
							resp.FinishReason = "cycle_intervention"
							return resp, nil
						}
					}
				}
			}
		}

		if len(llmResp.ToolCalls) > 0 {
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

	// Check if we hit max turns
	if resp.TurnsUsed >= a.config.MaxTurns {
		resp.FinishReason = "max_turns"
	}

	// Capture all new messages generated during this turn (everything after history)
	// This includes assistant messages with tool calls, tool result messages, and final assistant message
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

	// Add span attributes before returning
	span.SetAttribute("finish_reason", resp.FinishReason)
	span.SetAttribute("turns_used", resp.TurnsUsed)
	span.SetAttribute("tokens_used", resp.TokensUsed)

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

// callLLM calls the LLM provider with the given messages and filtered tools based on task mode.
// The task parameter controls both tool filtering and token budget:
//   - Tools: Only tools in task.AllowedTools() are included
//   - Tokens: Uses task.MaxTokens() if > 0, otherwise agent.config.MaxTokens
func (a *Agent) callLLM(ctx context.Context, messages []Message, task Task) (*llm.CompletionResponse, error) {
	// Convert messages to LLM format
	llmMessages := make([]llm.Message, len(messages))
	for i, msg := range messages {
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

		llmMessages[i] = llmMsg
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
			a.emitter.Emit(Event{
				Type:      EventContentDelta,
				Timestamp: time.Now(),
				Data: ContentDeltaData{
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

// ShouldApprove determines if a command needs user approval.
//
// Returns:
//   - needsApproval: true if the command requires approval
//   - reason: explanation of why approval is needed
func (a *Agent) ShouldApprove(cmd *Command) (bool, string) {
	// If approval is disabled, never require approval
	if !a.config.RequireApproval {
		return false, ""
	}

	// Classify the command
	result, err := a.validator.Classify(cmd)
	if err != nil {
		// On error, require approval for safety
		return true, fmt.Sprintf("Classification error: %v", err)
	}

	switch result.Classification {
	case CommandSafe:
		return false, ""

	case CommandInteractive:
		return true, "This command may modify files or system state"

	case CommandDangerous:
		return true, fmt.Sprintf("WARNING: Dangerous operation - %s", result.Reason)

	case CommandForbidden:
		// Forbidden commands should never be executed, even with approval
		// This will be handled by the executor
		return false, fmt.Sprintf("BLOCKED: %s", result.Reason)

	case CommandUnverified:
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

		// Convert llm.ToolCall to core.ToolCall
		coreToolCall := &ToolCall{
			ID:   toolCall.ID,
			Type: toolCall.Type,
			Function: ToolCallFunction{
				Name:      toolCall.Function.Name,
				Arguments: toolCall.Function.Arguments,
			},
		}

		// Add to assistant message (note: message already appended above)
		messages[len(messages)-1].ToolCalls = append(messages[len(messages)-1].ToolCalls, *coreToolCall)

		// Process the tool call (ProcessToolCall will emit EventToolCallStart)
		toolResult, err := a.ProcessToolCall(ctx, coreToolCall)
		if err != nil {
			// Emit tool error event
			a.emitter.Emit(Event{
				Type:      EventToolCallComplete,
				Timestamp: time.Now(),
				Data: ToolCallCompleteData{
					ToolID:   coreToolCall.ID,
					ToolName: coreToolCall.Function.Name,
					Success:  false,
					Error:    err.Error(),
				},
			})

			// Add error message to conversation (after assistant message)
			messages = append(messages, Message{
				Role: RoleTool,
				Content: fmt.Sprintf("Tool %s failed: %v",
					coreToolCall.Function.Name, err),
				ToolCallID: coreToolCall.ID,
				Timestamp:  time.Now(),
			})
		} else {
			// Emit tool completion event
			completion := ToolCallCompleteData{
				ToolID:   coreToolCall.ID,
				ToolName: coreToolCall.Function.Name,
				Success:  toolResult.Success,
			}
			if toolResult.Success {
				completion.Output = toolResult.Output
			} else if toolResult.Error != nil {
				completion.Error = toolResult.Error.Error()
			}
			a.emitter.Emit(Event{
				Type:      EventToolCallComplete,
				Timestamp: time.Now(),
				Data:      completion,
			})

			// Add tool result to conversation (after assistant message)
			slog.Debug("Agent tool result", "tool", coreToolCall.Function.Name, "output_len", len(toolResult.Output), "success", toolResult.Success)
			messages = append(messages, Message{
				Role:       RoleTool,
				Content:    toolResult.Output,
				ToolCallID: coreToolCall.ID,
				Timestamp:  time.Now(),
			})

			// Track tool call in response
			resp.ToolCalls = append(resp.ToolCalls, coreToolCall)
		}
	}

	return messages
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
		return &ToolResult{
			ID:      call.ID,
			Success: false,
			Error:   err,
		}, nil // Return nil error so agent continues
	}

	// 2. Parse arguments
	args, err := a.parseToolArguments(call)
	if err != nil {
		return &ToolResult{
			ID:      call.ID,
			Success: false,
			Error:   err,
		}, nil
	}

	// 3. Emit tool start event
	// Convert args to ToolCallArguments
	toolArgs, _ := types.FromMap(args)

	a.emitter.Emit(Event{
		Type:      EventToolCallStart,
		Timestamp: time.Now(),
		Data: ToolCallStartData{
			ToolID:     call.ID,
			ToolName:   call.Function.Name,
			Parameters: toolArgs,
		},
	})

	// 4. Execute tool
	var result *ToolResult

	// execute_command needs special handling for approval workflow
	if call.Function.Name == "execute_command" {
		result, _ = a.executeCommand(ctx, call.ID, args)
	} else {
		// Use tool registry for other tools
		toolResult, err := a.toolRegistry.Execute(ctx, call.Function.Name, args)
		if err != nil {
			result = &ToolResult{
				ID:      call.ID,
				Success: false,
				Error:   err,
			}
		} else {
			// Convert tools.ToolResult to core.ToolResult
			result = &ToolResult{
				ID:      call.ID,
				Success: toolResult.Success,
				Output:  toolResult.Output,
			}
			if toolResult.Error != "" {
				result.Error = errors.New(toolResult.Error)
			}
		}
	}

	// 5. Emit completion event
	completion := ToolCallCompleteData{
		ToolID:   call.ID,
		ToolName: call.Function.Name,
		Success:  result.Success,
	}
	if result.Success {
		completion.Output = result.Output
	} else if result.Error != nil {
		completion.Error = result.Error.Error()
	}
	a.emitter.Emit(Event{
		Type:      EventToolCallComplete,
		Timestamp: time.Now(),
		Data:      completion,
	})

	return result, nil // Always return nil error so agent continues
}

// validateToolCall validates the tool call structure.
func (a *Agent) validateToolCall(call *ToolCall) error {
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
func (a *Agent) parseToolArguments(call *ToolCall) (map[string]interface{}, error) {
	if call.Function.Arguments == "" {
		return nil, errors.New("tool arguments cannot be empty")
	}

	var args map[string]interface{}
	if err := json.Unmarshal([]byte(call.Function.Arguments), &args); err != nil {
		return nil, fmt.Errorf("failed to parse tool arguments: %w", err)
	}

	return args, nil
}

// executeCommand executes a shell command with approval workflow.
func (a *Agent) executeCommand(ctx context.Context, id string, args map[string]interface{}) (*ToolResult, error) {
	// Extract command string
	cmdStr, ok := args["command"].(string)
	if !ok {
		return &ToolResult{
			ID:      id,
			Success: false,
			Error:   errors.New("command argument must be a string"),
		}, nil
	}

	// Parse command into parts
	parts := strings.Fields(cmdStr)
	if len(parts) == 0 {
		return &ToolResult{
			ID:      id,
			Success: false,
			Error:   errors.New("command cannot be empty"),
		}, nil
	}

	// Create Command struct
	cmd := &Command{
		Program: parts[0],
		Args:    parts[1:],
		Raw:     cmdStr,
	}

	// Set working directory if provided
	if workDir, ok := args["workdir"].(string); ok {
		cmd.WorkDir = workDir
	} else {
		cmd.WorkDir = a.context.WorkDir
	}

	// Check if command needs approval
	if needsApproval, reason := a.ShouldApprove(cmd); needsApproval {
		// Request approval
		approved := a.requestApproval(ctx, cmd, reason)
		if !approved {
			return &ToolResult{
				ID:      id,
				Success: false,
				Error:   errors.New("command denied by user"),
			}, nil
		}
	}

	// Execute command (use default options)
	result, err := a.executor.Execute(ctx, cmd, nil)
	if err != nil {
		return &ToolResult{
			ID:       id,
			Success:  false,
			Output:   result.Stderr,
			Error:    err,
			ExitCode: result.ExitCode,
		}, nil
	}

	return &ToolResult{
		ID:       id,
		Success:  result.ExitCode == 0,
		Output:   result.Stdout,
		ExitCode: result.ExitCode,
	}, nil
}

// requestApproval requests user approval for a command.
// It emits an EventCommandApproval event, invokes the approval handler (if set),
// and emits the result (EventCommandApproved or EventCommandDenied).
func (a *Agent) requestApproval(ctx context.Context, cmd *Command, reason string) bool {
	// Generate unique request ID
	reqID := uuid.New().String()

	// Create approval request
	req := ApprovalRequest{
		ID:        reqID,
		Command:   cmd,
		Reason:    reason,
		WorkDir:   a.context.WorkDir,
		Timestamp: time.Now(),
	}

	// Emit approval request event
	a.emitter.Emit(Event{
		Type:      EventCommandApproval,
		Timestamp: req.Timestamp,
		Data: ApprovalEventData{
			RequestID: req.ID,
			Command:   cmd.Raw,
			WorkDir:   req.WorkDir,
			Reason:    req.Reason,
			Status:    "pending",
			Timestamp: req.Timestamp,
		},
	})

	// If no handler, auto-deny
	if a.approvalHandler == nil {
		a.emitter.Emit(Event{
			Type:      EventCommandDenied,
			Timestamp: time.Now(),
			Data: ApprovalEventData{
				RequestID: reqID,
				Command:   cmd.Raw,
				WorkDir:   cmd.WorkDir,
				Reason:    "no approval handler configured",
				Status:    "denied",
				Timestamp: time.Now(),
			},
		})
		return false
	}

	// Determine approval timeout
	timeout := a.config.ApprovalTimeout
	if timeout == 0 {
		timeout = 60 * time.Second // default 60 seconds
	}

	// Create timeout context
	approvalCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Invoke handler in goroutine with timeout
	respChan := make(chan ApprovalResponse, 1)
	go func() {
		resp := a.approvalHandler(req)
		respChan <- resp
	}()

	// Wait for response or timeout
	var resp ApprovalResponse
	select {
	case resp = <-respChan:
		// Got response
	case <-approvalCtx.Done():
		// Timeout or cancellation
		a.emitter.Emit(Event{
			Type:      EventCommandDenied,
			Timestamp: time.Now(),
			Data: ApprovalEventData{
				RequestID: reqID,
				Command:   cmd.Raw,
				WorkDir:   cmd.WorkDir,
				Reason:    "approval timeout or context cancelled",
				Status:    "denied",
				Timestamp: time.Now(),
			},
		})
		return false
	}

	// Validate response request ID
	if resp.RequestID != reqID {
		a.emitter.Emit(Event{
			Type:      EventCommandDenied,
			Timestamp: time.Now(),
			Data: ApprovalEventData{
				RequestID: reqID,
				Command:   cmd.Raw,
				WorkDir:   cmd.WorkDir,
				Reason:    "response request ID mismatch",
				Status:    "denied",
				Timestamp: time.Now(),
			},
		})
		return false
	}

	// Handle command modification
	if resp.Approved && resp.ModifiedCommand != "" {
		// Parse modified command
		modCmd, err := ParseCommand(resp.ModifiedCommand)
		if err != nil {
			a.emitter.Emit(Event{
				Type:      EventCommandDenied,
				Timestamp: time.Now(),
				Data: ApprovalEventData{
					RequestID: reqID,
					Command:   cmd.Raw,
					WorkDir:   cmd.WorkDir,
					Reason:    "modified command parse error: " + err.Error(),
					Status:    "denied",
					Timestamp: time.Now(),
				},
			})
			return false
		}

		// Re-validate modified command
		result, err := a.validator.Classify(modCmd)
		if err != nil {
			a.emitter.Emit(Event{
				Type:      EventCommandDenied,
				Timestamp: time.Now(),
				Data: ApprovalEventData{
					RequestID: reqID,
					Command:   cmd.Raw,
					WorkDir:   cmd.WorkDir,
					Reason:    "modified command validation error: " + err.Error(),
					Status:    "denied",
					Timestamp: time.Now(),
				},
			})
			return false
		}
		if result.Classification != CommandSafe {
			a.emitter.Emit(Event{
				Type:      EventCommandDenied,
				Timestamp: time.Now(),
				Data: ApprovalEventData{
					RequestID: reqID,
					Command:   cmd.Raw,
					WorkDir:   cmd.WorkDir,
					Reason:    fmt.Sprintf("modified command failed validation: %s", result.Classification.String()),
					Status:    "denied",
					Timestamp: time.Now(),
				},
			})
			return false
		}

		// Update command with modified version
		*cmd = *modCmd
	}

	// Emit result event
	if resp.Approved {
		a.emitter.Emit(Event{
			Type:      EventCommandApproved,
			Timestamp: time.Now(),
			Data: ApprovalEventData{
				RequestID:       reqID,
				Command:         cmd.Raw,
				WorkDir:         cmd.WorkDir,
				Reason:          resp.Reason,
				Status:          "approved",
				ModifiedCommand: resp.ModifiedCommand,
				Timestamp:       time.Now(),
			},
		})
		return true
	}

	// Denied
	a.emitter.Emit(Event{
		Type:      EventCommandDenied,
		Timestamp: time.Now(),
		Data: ApprovalEventData{
			RequestID: reqID,
			Command:   cmd.Raw,
			WorkDir:   cmd.WorkDir,
			Reason:    resp.Reason,
			Status:    "denied",
			Timestamp: time.Now(),
		},
	})
	return false
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

// eventEmitterAdapter adapts core.EventEmitter to cycle.EventEmitter interface
type eventEmitterAdapter struct {
	emitter *EventEmitter
}

func (a *eventEmitterAdapter) Emit(event cycle.Event) {
	// Convert cycle.Event to core.Event
	// Map event type based on string value
	var eventType EventType
	switch event.GetType() {
	case "turn_paused":
		eventType = EventTurnPaused
	default:
		eventType = EventWarning // fallback
	}

	coreEvent := Event{
		Type:      eventType,
		Timestamp: event.GetTimestamp(),
		Data:      event.GetData(),
	}
	a.emitter.Emit(coreEvent)
}

// extractToolNames extracts tool names from LLM tool calls for cycle detection
func extractToolNames(toolCalls []llm.ToolCall) []string {
	names := make([]string, len(toolCalls))
	for i, tc := range toolCalls {
		names[i] = tc.Function.Name
	}
	return names
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
