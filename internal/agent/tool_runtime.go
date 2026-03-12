package agent

import (
	"errors"
	"context"
	"fmt"
	"sync"

	"github.com/dmytrogajewski/spin/internal/events"
	"github.com/dmytrogajewski/spin/internal/message"
	"github.com/dmytrogajewski/spin/internal/security"
	"github.com/dmytrogajewski/spin/internal/tools"
)

var (
	ErrToolNotFound                        = errors.New("tool not found")
	ErrApprovalRequiredButNoApprovalHandler = errors.New("approval required but no approval handler configured")
	ErrOperationDenied                     = errors.New("operation denied")
)

// ToolCall is an alias for message.ToolCall to avoid duplication.
type ToolCall = message.ToolCall

// ToolCallFunction is an alias for message.ToolCallFunction to avoid duplication.
type ToolCallFunction = message.ToolCallFunction

// ToolResult is an alias for tools.ToolResult to provide a unified type.
// This consolidates the previously duplicate ToolResult definitions.
type ToolResult = tools.ToolResult

// ToolRuntimeConfig configures the tool runtime.
type ToolRuntimeConfig struct {
	Registry        *tools.Registry
	Validator       *security.Validator
	ApprovalService *security.ApprovalService
	Emitter         *events.EventEmitter
	WorkDir         string
}

// ToolRuntime executes tool calls with validation, approval, and event emission.
type ToolRuntime struct {
	registry        *tools.Registry
	validator       *security.Validator
	approvalService *security.ApprovalService
	emitter         *events.EventEmitter
	workDir         string
}

// NewToolRuntime creates a new tool runtime from config.
func NewToolRuntime(cfg ToolRuntimeConfig) *ToolRuntime {
	return &ToolRuntime{
		registry:        cfg.Registry,
		validator:       cfg.Validator,
		approvalService: cfg.ApprovalService,
		emitter:         cfg.Emitter,
		workDir:         cfg.WorkDir,
	}
}

// SetApprovalService updates the approval service.
func (t *ToolRuntime) SetApprovalService(service *security.ApprovalService) {
	t.approvalService = service
}

// Registry returns the underlying tool registry.
func (t *ToolRuntime) Registry() *tools.Registry {
	return t.registry
}

// Execute executes a single tool call.
func (t *ToolRuntime) Execute(ctx context.Context, call *ToolCall) (*ToolResult, error) {
	err := t.validateToolCall(call)
	if err != nil {
		callID := ""
		if call != nil {
			callID = call.ID
		}

		result := tools.NewToolErrorWithID(callID, err)

		return &result, nil
	}

	args, err := t.parseToolArguments(call)
	if err != nil {
		result := tools.NewToolErrorWithID(call.ID, fmt.Errorf("invalid arguments: %w", err))

		return &result, nil
	}

	tool, err := t.resolveTool(call)
	if err != nil {
		result := tools.NewToolErrorWithID(call.ID, err)

		return &result, nil
	}

	if denied := t.checkToolApproval(ctx, tool, args, call); denied != nil {
		return denied, nil
	}

	toolResult, err := tool.Execute(ctx, args)
	if err != nil {
		result := tools.NewToolErrorWithID(call.ID, err)

		return &result, nil
	}

	// The tool already returned a tools.ToolResult, just add the ID.
	result := toolResult.WithID(call.ID)

	return &result, nil
}

// resolveTool looks up the tool by name and returns a helpful error if not found.
func (t *ToolRuntime) resolveTool(call *ToolCall) (tools.Tool, error) {
	tool, err := t.registry.Get(call.Function.Name)
	if err != nil {
		available := t.registry.List()
		names := make([]string, len(available))
		for i, at := range available {
			names[i] = at.Name()
		}

		return nil, fmt.Errorf(
			"tool not found: %q is not a valid tool. Available tools: %v: %w",
			call.Function.Name, names, ErrToolNotFound,
		)
	}

	return tool, nil
}

// checkToolApproval checks if a tool requires approval and requests it if needed.
// Returns a non-nil ToolResult if the operation was denied or approval failed.
func (t *ToolRuntime) checkToolApproval(ctx context.Context, tool tools.Tool, args tools.ToolParameters, call *ToolCall) *ToolResult {
	toolWithApproval, ok := tool.(tools.ToolWithApproval)
	if !ok {
		return nil
	}

	needs := toolWithApproval.CheckApproval(args)
	if !needs.Required {
		return nil
	}

	if t.approvalService == nil {
		result := tools.NewToolErrorWithID(call.ID, fmt.Errorf(
			"approval required but no approval handler configured: %s (risk: %s): %w",
			needs.Reason, needs.Risk, ErrApprovalRequiredButNoApprovalHandler,
		))

		return &result
	}

	cmd := &security.Command{
		Program: call.Function.Name,
		Args:    []string{needs.Reason},
		Raw:     fmt.Sprintf("%s: %s", call.Function.Name, needs.Reason),
		WorkDir: t.workDir,
	}

	operation := security.NewOperationWithToolCallID(cmd, needs.Reason, t.workDir, call.ID)

	_, approved, approvalErr := t.approvalService.RequestApproval(ctx, operation)
	if approvalErr != nil {
		result := tools.NewToolErrorWithID(call.ID, fmt.Errorf("approval request failed: %w", approvalErr))

		return &result
	}

	if !approved {
		result := tools.NewToolErrorWithID(call.ID, fmt.Errorf(
			"operation denied: %s (risk: %s): %w",
			needs.Reason, needs.Risk, ErrOperationDenied,
		))

		return &result
	}

	return nil
}

// ExecuteBatch runs multiple tool calls concurrently.
func (t *ToolRuntime) ExecuteBatch(ctx context.Context, calls []*ToolCall) ([]*ToolResult, error) {
	if len(calls) == 0 {
		return []*ToolResult{}, nil
	}

	results := make([]*ToolResult, len(calls))
	errs := make([]error, len(calls))

	var wg sync.WaitGroup
	for i, call := range calls {
		wg.Add(1)

		go func(idx int, c *ToolCall) {
			defer wg.Done()

			res, err := t.Execute(ctx, c)
			results[idx] = res
			errs[idx] = err
		}(i, call)
	}

	wg.Wait()

	for i, err := range errs {
		if err != nil {
			return nil, fmt.Errorf("executing tool call %d: %w", i, err)
		}
	}

	return results, nil
}

func (t *ToolRuntime) validateToolCall(call *ToolCall) error {
	return tools.ValidateToolCall(call)
}

func (t *ToolRuntime) parseToolArguments(call *ToolCall) (tools.ToolParameters, error) {
	parser := tools.NewStrictArgumentParser()

	return parser.Parse(call.Function.Arguments)
}
