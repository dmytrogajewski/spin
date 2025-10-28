package orchestration

import (
	"context"
	"fmt"
	"sync"

	"github.com/dmytrogajewski/spin/internal/events"
	"github.com/dmytrogajewski/spin/internal/security"
	"github.com/dmytrogajewski/spin/internal/tools"
)

// ToolExecutor handles execution of tool calls with validation and approval.
// It centralizes all tool execution logic that was previously scattered in Agent.
//
// GOROUTINE LIFECYCLE:
// - ExecuteBatch() spawns one goroutine per tool call that:
//   - Executes the tool via Execute()
//   - Stores the result at the appropriate index
//   - Lives until tool execution completes or context is cancelled
//   - Automatically cleaned up when all tools complete (via WaitGroup)
//
// - All goroutines terminate when WaitGroup completes
//
// CONCURRENCY:
// - Execute() is safe to call concurrently (each call is independent)
// - ExecuteBatch() creates isolated goroutines with no shared state
// - Results are stored at predetermined indices (no race conditions)
// - WaitGroup ensures all goroutines complete before returning
type ToolExecutor struct {
	registry        *tools.Registry
	validator       *security.Validator
	approvalService *security.ApprovalService
	emitter         *events.EventEmitter
	workDir         string
}

// ToolExecutorConfig configures the tool executor.
type ToolExecutorConfig struct {
	Registry        *tools.Registry
	Validator       *security.Validator
	ApprovalService *security.ApprovalService
	Emitter         *events.EventEmitter
	WorkDir         string
}

// NewToolExecutor creates a new tool executor with the given configuration.
func NewToolExecutor(cfg ToolExecutorConfig) *ToolExecutor {
	return &ToolExecutor{
		registry:        cfg.Registry,
		validator:       cfg.Validator,
		approvalService: cfg.ApprovalService,
		emitter:         cfg.Emitter,
		workDir:         cfg.WorkDir,
	}
}

// Execute executes a tool call and returns the result.
// This method handles:
// - Tool validation
// - Argument parsing
// - Command execution with approval
// - Event emission
func (t *ToolExecutor) Execute(ctx context.Context, call *ToolCall) (*ToolResult, error) {
	// Validate tool call
	if err := t.validateToolCall(call); err != nil {
		// Handle nil call case
		callID := ""
		if call != nil {
			callID = call.ID
		}
		return &ToolResult{
			ID:    callID,
			Error: err,
		}, nil
	}

	// Parse arguments
	args, err := t.parseToolArguments(call)
	if err != nil {
		return &ToolResult{
			ID:      call.ID,
			Success: false,
			Error:   fmt.Errorf("invalid arguments: %w", err),
		}, nil
	}

	// Get tool from registry
	tool, err := t.registry.Get(call.Function.Name)
	if err != nil {
		return &ToolResult{
			ID:      call.ID,
			Success: false,
			Error:   fmt.Errorf("tool not found: %s: %w", call.Function.Name, err),
		}, nil
	}

	// Check if tool requires approval
	if toolWithApproval, ok := tool.(tools.ToolWithApproval); ok {
		needs := toolWithApproval.CheckApproval(args)
		if needs.Required {
			return &ToolResult{
				ID:      call.ID,
				Success: false,
				Error:   fmt.Errorf("approval required: %s (risk: %s)", needs.Reason, needs.Risk),
			}, nil
		}
	}

	// Execute the tool
	toolResult, err := tool.Execute(ctx, args)
	if err != nil {
		return &ToolResult{
			ID:    call.ID,
			Error: err,
		}, nil
	}

	// Convert tools.ToolResult to ToolResult
	result := &ToolResult{
		ID:      call.ID,
		Success: toolResult.Success,
		Output:  toolResult.Output,
	}

	// Convert string error to error type if present
	if toolResult.Error != "" {
		result.Error = fmt.Errorf("%s", toolResult.Error)
	}

	return result, nil
}

// validateToolCall validates a tool call structure.
func (t *ToolExecutor) validateToolCall(call *ToolCall) error {
	if call == nil {
		return fmt.Errorf("tool call is nil")
	}
	if call.ID == "" {
		return fmt.Errorf("tool call ID is empty")
	}
	if call.Function.Name == "" {
		return fmt.Errorf("tool call function name is empty")
	}
	return nil
}

// parseToolArguments parses tool call arguments from JSON.
func (t *ToolExecutor) parseToolArguments(call *ToolCall) (tools.ToolParameters, error) {
	parser := tools.NewArgumentParser()
	return parser.Parse(call.Function.Arguments)
}

// ExecuteBatch executes multiple tool calls concurrently.
// Results are returned in the same order as the input calls.
func (t *ToolExecutor) ExecuteBatch(ctx context.Context, calls []*ToolCall) ([]*ToolResult, error) {
	if len(calls) == 0 {
		return []*ToolResult{}, nil
	}

	results := make([]*ToolResult, len(calls))
	errs := make([]error, len(calls))

	var wg sync.WaitGroup

	// Execute all calls concurrently
	for i, call := range calls {
		wg.Add(1)
		go func(idx int, c *ToolCall) {
			defer wg.Done()

			// Execute tool
			result, err := t.Execute(ctx, c)

			// Store result and error at index
			results[idx] = result
			errs[idx] = err
		}(i, call)
	}

	// Wait for all goroutines to complete
	wg.Wait()

	// Check for errors (fail fast on first error)
	for i, err := range errs {
		if err != nil {
			return nil, fmt.Errorf("executing tool call %d: %w", i, err)
		}
	}

	return results, nil
}
