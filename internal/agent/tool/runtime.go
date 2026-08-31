// Package tool provides tool execution runtime for the agent.
package tool

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"

	"github.com/dmytrogajewski/spin/internal/contexteng/compact"
	"github.com/dmytrogajewski/spin/internal/events"
	"github.com/dmytrogajewski/spin/internal/message"
	"github.com/dmytrogajewski/spin/internal/safety"
	"github.com/dmytrogajewski/spin/internal/safety/hooks"
	"github.com/dmytrogajewski/spin/internal/tools"
)

var (
	// ErrApprovalRequiredButNoApprovalHandler is a sentinel error.
	ErrApprovalRequiredButNoApprovalHandler = errors.New("approval required but no approval handler configured")
	// ErrOperationDenied is a sentinel error.
	ErrOperationDenied = errors.New("operation denied")
	// ErrBlockedByHook is a sentinel error for hook-blocked tool calls.
	ErrBlockedByHook = errors.New("blocked by hook")
)

// RuntimeConfig configures the tool runtime.
type RuntimeConfig struct {
	Registry        *tools.Registry
	Validator       *safety.Validator
	ApprovalService *safety.ApprovalService
	Emitter         *events.EventEmitter
	WorkDir         string
	HookRunner      *hooks.Runner
	CompactEnabled  bool
	CompactBackend  string
	LookPath        func(string) (string, error)
}

// Runtime executes tool calls with validation, approval, and event emission.
type Runtime struct {
	registry        *tools.Registry
	validator       *safety.Validator
	approvalService *safety.ApprovalService
	emitter         *events.EventEmitter
	workDir         string
	hookRunner      *hooks.Runner
	compactEnabled  bool
	compactBackend  string
	lookPath        func(string) (string, error)
}

// NewRuntime creates a new tool runtime from config.
func NewRuntime(cfg RuntimeConfig) *Runtime {
	return &Runtime{
		registry:        cfg.Registry,
		validator:       cfg.Validator,
		approvalService: cfg.ApprovalService,
		emitter:         cfg.Emitter,
		workDir:         cfg.WorkDir,
		hookRunner:      cfg.HookRunner,
		compactEnabled:  cfg.CompactEnabled,
		compactBackend:  cfg.CompactBackend,
		lookPath:        cfg.LookPath,
	}
}

// ApprovalService updates the approval service.
func (t *Runtime) ApprovalService(service *safety.ApprovalService) *safety.ApprovalService {
	if service != nil {
		t.approvalService = service
	}

	return t.approvalService
}

// Registry returns the underlying tool registry.
func (t *Runtime) Registry() *tools.Registry {
	return t.registry
}

// Execute executes a single tool call.
func (t *Runtime) Execute(ctx context.Context, call *message.ToolCall) (*tools.ToolResult, error) {
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
		t.emitToolComplete(call, tool.Name(), denied, nil)

		return denied, nil
	}

	// Run PRE_TOOL_USE hook — may block execution or replace arguments.
	if hookResult := t.runPreToolHook(ctx, call, &args); hookResult != nil {
		return hookResult, nil
	}

	t.rewriteShellArgv(call, &args)

	// Emit tool call start event.
	t.emitToolStart(call, args)

	toolResult, err := tool.Execute(ctx, args)
	if err != nil {
		result := tools.NewToolErrorWithID(call.ID, err)
		t.emitToolComplete(call, tool.Name(), &result, err)
		t.runPostToolHook(ctx, call, &result, hooks.EventPostToolUseFailure)

		return &result, nil
	}

	result := toolResult.WithID(call.ID)
	t.emitToolComplete(call, tool.Name(), &result, nil)
	t.runPostToolHook(ctx, call, &result, hooks.EventPostToolUse)

	return &result, nil
}

// emitToolStart emits an EventToolCallStart event.
func (t *Runtime) emitToolStart(call *message.ToolCall, args tools.ToolParameters) {
	if t.emitter == nil {
		return
	}

	t.emitter.Emit(events.Event{
		Type: events.EventToolCallStart,
		Data: events.ToolCallStartData{
			ToolName:   call.Function.Name,
			ToolID:     call.ID,
			Parameters: args,
		},
	})
}

// emitToolComplete emits an EventToolCallComplete event.
func (t *Runtime) emitToolComplete(call *message.ToolCall, toolName string, result *tools.ToolResult, err error) {
	if t.emitter == nil {
		return
	}

	data := events.ToolCallCompleteData{
		ToolID:   call.ID,
		ToolName: toolName,
		Success:  result != nil && result.Success,
	}

	if result != nil {
		data.Output = result.Output
		data.Metadata = result.Metadata
	}

	if err != nil {
		data.Error = err.Error()
	} else if result != nil && result.Error != "" {
		data.Error = result.Error
	}

	t.emitter.Emit(events.Event{
		Type: events.EventToolCallComplete,
		Data: data,
	})
}

// resolveTool looks up the tool by name and returns a helpful error if not found.
func (t *Runtime) resolveTool(call *message.ToolCall) (tools.Tool, error) {
	tool, err := t.registry.Get(call.Function.Name)
	if err != nil {
		available := t.registry.List()

		names := make([]string, len(available))
		for i, at := range available {
			names[i] = at.Name()
		}

		return nil, fmt.Errorf(
			"tool not found: %q is not a valid tool. Available tools: %v: %w",
			call.Function.Name, names, tools.ErrToolNotFound,
		)
	}

	return tool, nil
}

// checkToolApproval checks if a tool requires approval and requests it if needed.
// Returns a non-nil ToolResult if the operation was denied or approval failed.
func (t *Runtime) checkToolApproval(
	ctx context.Context, tool tools.Tool, args tools.ToolParameters, call *message.ToolCall,
) *tools.ToolResult {
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

	cmd := &safety.Command{
		Program: call.Function.Name,
		Args:    []string{needs.Reason},
		Raw:     fmt.Sprintf("%s: %s", call.Function.Name, needs.Reason),
		WorkDir: t.workDir,
	}

	operation := safety.NewOperationWithToolCallID(cmd, needs.Reason, t.workDir, call.ID)

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
func (t *Runtime) ExecuteBatch(ctx context.Context, calls []*message.ToolCall) ([]*tools.ToolResult, error) {
	if len(calls) == 0 {
		return []*tools.ToolResult{}, nil
	}

	results := make([]*tools.ToolResult, len(calls))
	errs := make([]error, len(calls))

	var wg sync.WaitGroup
	for i, call := range calls {
		wg.Add(1)

		go func(idx int, c *message.ToolCall) {
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

func (t *Runtime) validateToolCall(call *message.ToolCall) error {
	return tools.ValidateToolCall(call)
}

func (t *Runtime) parseToolArguments(call *message.ToolCall) (tools.ToolParameters, error) {
	parser := tools.NewStrictArgumentParser()

	return parser.Parse(call.Function.Arguments)
}

// runPreToolHook executes PRE_TOOL_USE hooks. Returns a ToolResult if blocked.
func (t *Runtime) runPreToolHook(ctx context.Context, call *message.ToolCall, args *tools.ToolParameters) *tools.ToolResult {
	if t.hookRunner == nil {
		return nil
	}

	evtCtx := hooks.EventContext{
		WorkDir:   t.workDir,
		ToolName:  call.Function.Name,
		ToolInput: call.Function.Arguments,
	}

	hookResult := t.hookRunner.Execute(ctx, hooks.EventPreToolUse, evtCtx)
	if hookResult.Blocked {
		result := tools.NewToolErrorWithID(call.ID,
			fmt.Errorf("%w: %s", ErrBlockedByHook, hookResult.Reason))

		return &result
	}

	applyUpdatedInput(call, args, hookResult.UpdatedInput)

	return nil
}

func applyUpdatedInput(call *message.ToolCall, args *tools.ToolParameters, raw string) {
	if raw == "" || args == nil {
		return
	}

	parsed, err := tools.NewStrictArgumentParser().Parse(raw)
	if err != nil {
		return
	}

	call.Function.Arguments = raw
	*args = parsed
}

func (t *Runtime) rewriteShellArgv(call *message.ToolCall, args *tools.ToolParameters) {
	if !t.compactEnabled || args == nil || call.Function.Name != "shell_command" {
		return
	}

	cmd := args.GetStringOr("command", "")

	rewritten, used := compact.RewriteArgv(cmd, t.compactBackend, t.lookPath)
	if !used || rewritten == cmd {
		return
	}

	mapped := args.ToMap()
	mapped["command"] = rewritten

	raw, err := json.Marshal(mapped)
	if err != nil {
		return
	}

	applyUpdatedInput(call, args, string(raw))
}

// runPostToolHook fires a post-tool lifecycle event asynchronously (non-blocking).
func (t *Runtime) runPostToolHook(ctx context.Context, call *message.ToolCall, result *tools.ToolResult, event hooks.Event) {
	if t.hookRunner == nil {
		return
	}

	evtCtx := hooks.EventContext{
		WorkDir:      t.workDir,
		ToolName:     call.Function.Name,
		ToolInput:    call.Function.Arguments,
		ToolResponse: result.Output,
	}

	t.hookRunner.Execute(ctx, event, evtCtx)
}
