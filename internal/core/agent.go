package core

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

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
	llm             llm.Provider                // LLM provider interface
	executor        *Executor                   // Command executor
	validator       *Validator                  // Command validator
	context         *Environment                    // Environment context
	emitter         *EventEmitter               // Event emitter
	config          *AgentConfig                // Agent configuration
	toolRegistry    *tools.Registry             // Tool registry
	approvalHandler func(*Command, string) bool // Approval handler for testing
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
}

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

	// WorkDir is the working directory
	WorkDir string
}

// AgentResponse represents the agent's response.
type AgentResponse struct {
	// Content is the response content
	Content string

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

	// Create agent with defaults
	agent := &Agent{
		llm:          provider,
		executor:     executor,
		validator:    validator,
		context:      context,
		emitter:      emitter,
		toolRegistry: registry,
		config: &AgentConfig{
			MaxTurns:        DefaultMaxTurns,
			Timeout:         DefaultAgentTimeout,
			Temperature:     DefaultTemperature,
			MaxTokens:       DefaultMaxTokens,
			RequireApproval: false,
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

// WithToolRegistry sets a custom tool registry for the agent.
func WithToolRegistry(registry *tools.Registry) AgentOption {
	return func(a *Agent) error {
		if registry == nil {
			return errors.New("tool registry cannot be nil")
		}
		a.toolRegistry = registry
		return nil
	}
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
	// Validate request
	if req == nil {
		return nil, ErrNilRequest
	}
	if req.Input == "" {
		return nil, ErrEmptyInput
	}

	// Apply timeout from config if not already set
	var cancel context.CancelFunc
	if _, ok := ctx.Deadline(); !ok {
		ctx, cancel = context.WithTimeout(ctx, a.config.Timeout)
		defer cancel()
	}

	// Initialize response
	resp := &AgentResponse{
		ToolCalls:   make([]*ToolCall, 0),
		ToolResults: make([]*ToolResult, 0),
		TurnsUsed:   0,
		TokensUsed:  0,
	}

	// Build initial prompt
	messages := a.buildPrompt(req)

	// Agent loop
	for turn := 0; turn < a.config.MaxTurns; turn++ {
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
			Data: map[string]interface{}{
				"turn": turn + 1,
			},
		})

		// Call LLM
		llmResp, err := a.callLLM(ctx, messages)
		if err != nil {
			resp.Error = fmt.Errorf("LLM call failed: %w", err)
			resp.FinishReason = "error"
			return resp, err
		}

		// Accumulate content
		resp.Content += llmResp.Content

		// Emit content event
		if llmResp.Content != "" {
			a.emitter.Emit(Event{
				Type:      EventContentDelta,
				Timestamp: time.Now(),
				Data: map[string]interface{}{
					"content": llmResp.Content,
				},
			})
		}

		// Check if we're done (no tool calls means completion)
		// For now, we consider the response complete after one turn
		// TODO: In Phase 6.2, add proper tool call processing
		// Note: This is a temporary break until multi-turn support is added
		resp.FinishReason = "stop"
		break // nolint:staticcheck // Placeholder for Feature 6.2
	}

	// Check if we hit max turns
	if resp.TurnsUsed >= a.config.MaxTurns {
		resp.FinishReason = "max_turns"
	}

	// Emit completion event
	a.emitter.Emit(Event{
		Type:      EventTurnComplete,
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"turns_used":    resp.TurnsUsed,
			"tokens_used":   resp.TokensUsed,
			"finish_reason": resp.FinishReason,
		},
	})

	return resp, nil
}

// callLLM calls the LLM provider with the given messages.
func (a *Agent) callLLM(ctx context.Context, messages []Message) (*llm.CompletionResponse, error) {
	// Convert messages to LLM format
	llmMessages := make([]llm.Message, len(messages))
	for i, msg := range messages {
		llmMessages[i] = llm.Message{
			Role:    string(msg.Role),
			Content: msg.Content,
		}
	}

	// Build LLM request
	req := llm.CompletionRequest{
		Messages:    llmMessages,
		Temperature: a.config.Temperature,
		MaxTokens:   a.config.MaxTokens,
	}

	// Call LLM
	return a.llm.Complete(ctx, req)
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
		// Default system prompt
		prompt = `You are a helpful AI coding assistant. You can help with:
- Reading and analyzing code
- Writing and modifying code
- Running commands
- Searching files
- Managing git operations

Always explain your reasoning and ask for clarification when needed.`
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
	a.emitter.Emit(Event{
		Type:      EventToolCallStart,
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"tool_id":   call.ID,
			"tool_name": call.Function.Name,
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
	a.emitter.Emit(Event{
		Type:      EventToolCallComplete,
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"tool_id": call.ID,
			"success": result.Success,
		},
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
func (a *Agent) requestApproval(ctx context.Context, cmd *Command, reason string) bool {
	// Emit approval request event
	a.emitter.Emit(Event{
		Type:      EventCommandApproval,
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"command": cmd.Raw,
			"reason":  reason,
		},
	})

	// Use approval handler if set (for testing)
	if a.approvalHandler != nil {
		return a.approvalHandler(cmd, reason)
	}

	// Default to deny if no handler
	return false
}
