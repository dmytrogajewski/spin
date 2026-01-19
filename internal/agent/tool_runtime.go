package agent

import (
	"context"
	"fmt"
	"sync"

	"github.com/dmytrogajewski/spin/internal/events"
	"github.com/dmytrogajewski/spin/internal/message"
	"github.com/dmytrogajewski/spin/internal/security"
	"github.com/dmytrogajewski/spin/internal/tools"
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
	if err := t.validateToolCall(call); err != nil {
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

	tool, err := t.registry.Get(call.Function.Name)
	if err != nil {
		result := tools.NewToolErrorWithID(call.ID, fmt.Errorf("tool not found: %s: %w", call.Function.Name, err))
		return &result, nil
	}

	if toolWithApproval, ok := tool.(tools.ToolWithApproval); ok {
		needs := toolWithApproval.CheckApproval(args)
		if needs.Required {
			if t.approvalService == nil {
				result := tools.NewToolErrorWithID(call.ID, fmt.Errorf("approval required but no approval handler configured: %s (risk: %s)", needs.Reason, needs.Risk))
				return &result, nil
			}

			cmd := &security.Command{
				Program: call.Function.Name,
				Args:    []string{needs.Reason},
				Raw:     fmt.Sprintf("%s: %s", call.Function.Name, needs.Reason),
				WorkDir: t.workDir,
			}

			// Pass tool call ID to approval service so approval notifications
			// use the same tool call ID as the tool call events
			operation := security.NewOperationWithToolCallID(cmd, needs.Reason, t.workDir, call.ID)

			_, approved, err := t.approvalService.RequestApproval(ctx, operation)
			if err != nil {
				result := tools.NewToolErrorWithID(call.ID, fmt.Errorf("approval request failed: %w", err))
				return &result, nil
			}

			if !approved {
				result := tools.NewToolErrorWithID(call.ID, fmt.Errorf("operation denied: %s (risk: %s)", needs.Reason, needs.Risk))
				return &result, nil
			}
		}
	}

	toolResult, err := tool.Execute(ctx, args)
	if err != nil {
		result := tools.NewToolErrorWithID(call.ID, err)
		return &result, nil
	}

	// The tool already returned a tools.ToolResult, just add the ID
	result := toolResult.WithID(call.ID)
	return &result, nil
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
