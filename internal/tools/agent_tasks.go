package tools

import (
	"context"
	"fmt"

	"github.com/dmytrogajewski/spin/internal/agent/tasks"
)

const (
	listAgentTasksName   = "list_agent_tasks"
	waitAgentTaskName    = "wait_agent_task"
	cancelAgentTaskName  = "cancel_agent_task"
	agentTaskIDParam     = "task_id"
	errAgentTasksMissing = "agent task registry is not available"
)

// AgentTaskStore is the A2A registry surface used by tools.
type AgentTaskStore interface {
	List() []tasks.Record
	Wait(ctx context.Context, id string) (tasks.Record, error)
	Cancel(ctx context.Context, id string) error
}

// ListAgentTasksTool lists A2A registry rows (not shell processes).
type ListAgentTasksTool struct {
	store AgentTaskStore
}

// NewListAgentTasksTool creates list_agent_tasks.
func NewListAgentTasksTool(store AgentTaskStore) *ListAgentTasksTool {
	return &ListAgentTasksTool{store: store}
}

// Name implements Tool.
func (t *ListAgentTasksTool) Name() string { return listAgentTasksName }

// Description implements Tool.
func (t *ListAgentTasksTool) Description() string {
	return "List A2A agent tasks (id, spec, state). Does not include shell start_process rows."
}

// Schema implements Tool.
func (t *ListAgentTasksTool) Schema() ToolSchema {
	return ToolSchema{Type: "function", Function: FunctionSchema{
		Name: t.Name(), Description: t.Description(),
		Parameters: ParameterSchema{Type: "object", Properties: map[string]PropertyDefinition{}},
	}}
}

// Execute lists A2A tasks.
func (t *ListAgentTasksTool) Execute(_ context.Context, _ ToolParameters) (ToolResult, error) {
	if t.store == nil {
		return NewToolResult(errAgentTasksMissing), nil
	}

	return NewToolResult(tasks.Format(t.store.List())), nil
}

// WaitAgentTaskTool blocks until an A2A task is terminal or ctx is canceled.
type WaitAgentTaskTool struct {
	store AgentTaskStore
}

// NewWaitAgentTaskTool creates wait_agent_task.
func NewWaitAgentTaskTool(store AgentTaskStore) *WaitAgentTaskTool {
	return &WaitAgentTaskTool{store: store}
}

// Name implements Tool.
func (t *WaitAgentTaskTool) Name() string { return waitAgentTaskName }

// Description implements Tool.
func (t *WaitAgentTaskTool) Description() string {
	return "Wait for an A2A agent task until completed, failed, or canceled."
}

// Schema implements Tool.
func (t *WaitAgentTaskTool) Schema() ToolSchema {
	return agentTaskIDSchema(t.Name(), t.Description())
}

// Execute waits on a task id.
func (t *WaitAgentTaskTool) Execute(ctx context.Context, params ToolParameters) (ToolResult, error) {
	return runAgentTaskID(t.store, params, func(id string) (string, error) {
		rec, err := t.store.Wait(ctx, id)
		if err != nil {
			return "", err
		}

		return tasks.Format([]tasks.Record{rec}), nil
	})
}

// CancelAgentTaskTool maps to tasks/cancel then SIGTERM.
type CancelAgentTaskTool struct {
	store AgentTaskStore
}

// NewCancelAgentTaskTool creates cancel_agent_task.
func NewCancelAgentTaskTool(store AgentTaskStore) *CancelAgentTaskTool {
	return &CancelAgentTaskTool{store: store}
}

// Name implements Tool.
func (t *CancelAgentTaskTool) Name() string { return cancelAgentTaskName }

// Description implements Tool.
func (t *CancelAgentTaskTool) Description() string {
	return "Cancel an A2A agent task (tasks/cancel then SIGTERM)."
}

// Schema implements Tool.
func (t *CancelAgentTaskTool) Schema() ToolSchema {
	return agentTaskIDSchema(t.Name(), t.Description())
}

// Execute cancels a task id.
func (t *CancelAgentTaskTool) Execute(ctx context.Context, params ToolParameters) (ToolResult, error) {
	return runAgentTaskID(t.store, params, func(id string) (string, error) {
		if err := t.store.Cancel(ctx, id); err != nil {
			return "", err
		}

		return fmt.Sprintf("canceled %s", id), nil
	})
}

func agentTaskIDSchema(name, desc string) ToolSchema {
	return ToolSchema{Type: "function", Function: FunctionSchema{
		Name: name, Description: desc,
		Parameters: ParameterSchema{
			Type: "object",
			Properties: map[string]PropertyDefinition{
				agentTaskIDParam: {Type: "string", Description: "A2A task id"},
			},
			Required: []string{agentTaskIDParam},
		},
	}}
}

func runAgentTaskID(store AgentTaskStore, params ToolParameters, fn func(id string) (string, error)) (ToolResult, error) {
	if store == nil {
		return NewToolResult(errAgentTasksMissing), nil
	}

	id := params.GetStringOr(agentTaskIDParam, "")
	if id == "" {
		return NewToolError(errTaskIDParameterRequired), nil
	}

	out, err := fn(id)
	if err != nil {
		return NewToolError(err), nil
	}

	return NewToolResult(out), nil
}

// RegisterAgentTaskTools registers list/wait/cancel against the A2A registry.
func RegisterAgentTaskTools(registry *Registry, store AgentTaskStore) {
	if registry == nil {
		return
	}

	_ = registry.Register(NewListAgentTasksTool(store))
	_ = registry.Register(NewWaitAgentTaskTool(store))
	_ = registry.Register(NewCancelAgentTaskTool(store))
}
