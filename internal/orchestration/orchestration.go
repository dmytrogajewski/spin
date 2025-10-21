package orchestration

import (
	"context"
	"fmt"

	"github.com/dmytrogajewski/spin/internal/tools"
)

// OrchestrationService handles tool execution, task management, and planning.
//
// This service centralizes orchestration logic that was previously embedded in Agent.
// It provides a clean interface for executing tools, managing task modes, and
// coordinating execution plans.
type OrchestrationService struct {
	toolExecutor *ToolExecutor
	toolRegistry *tools.Registry
	taskRegistry *Registry
	planner      *Plan
}

// NewOrchestrationService creates a new orchestration service with the given dependencies.
//
// All dependencies can be nil. When nil, methods that require these dependencies
// will return appropriate errors.
func NewOrchestrationService(
	toolExecutor *ToolExecutor,
	toolRegistry *tools.Registry,
	taskRegistry *Registry,
) *OrchestrationService {
	return &OrchestrationService{
		toolExecutor: toolExecutor,
		toolRegistry: toolRegistry,
		taskRegistry: taskRegistry,
		planner:      nil,
	}
}

// ExecuteTool executes a single tool call.
//
// This delegates to the underlying ToolExecutor. Returns an error if the
// tool executor is not configured.
func (s *OrchestrationService) ExecuteTool(ctx context.Context, call *ToolCall) (*ToolResult, error) {
	if s.toolExecutor == nil {
		return nil, fmt.Errorf("tool executor not configured")
	}

	return s.toolExecutor.Execute(ctx, call)
}

// ExecuteBatch executes multiple tool calls concurrently.
//
// Results are returned in the same order as the input calls. Returns an error
// if the tool executor is not configured.
func (s *OrchestrationService) ExecuteBatch(ctx context.Context, calls []*ToolCall) ([]*ToolResult, error) {
	if s.toolExecutor == nil {
		return nil, fmt.Errorf("tool executor not configured")
	}

	return s.toolExecutor.ExecuteBatch(ctx, calls)
}

// GetTask retrieves a task by name from the registry.
//
// Returns an error if the task registry is not configured or if the task
// is not found.
func (s *OrchestrationService) GetTask(name string) (Task, error) {
	if s.taskRegistry == nil {
		return nil, fmt.Errorf("task registry not configured")
	}

	task, err := s.taskRegistry.Get(name)
	if err != nil {
		return nil, fmt.Errorf("failed to get task '%s': %w", name, err)
	}

	return task, nil
}

// GetDefaultTask returns the default task mode.
//
// Returns an error if the task registry is not configured or if no default
// task is set.
func (s *OrchestrationService) GetDefaultTask() (Task, error) {
	if s.taskRegistry == nil {
		return nil, fmt.Errorf("task registry not configured")
	}

	task, err := s.taskRegistry.GetDefault()
	if err != nil {
		return nil, fmt.Errorf("failed to get default task: %w", err)
	}

	return task, nil
}

// ListTasks returns all registered task names in sorted order.
//
// Returns an empty slice if the task registry is not configured.
func (s *OrchestrationService) ListTasks() []string {
	if s.taskRegistry == nil {
		return []string{}
	}

	return s.taskRegistry.List()
}

// SetPlanner sets the execution planner.
//
// The planner is used for task decomposition and execution planning.
func (s *OrchestrationService) SetPlanner(planner *Plan) {
	s.planner = planner
}

// GetPlanner returns the current execution planner.
//
// Returns nil if no planner has been set.
func (s *OrchestrationService) GetPlanner() *Plan {
	return s.planner
}

// GetTaskRegistry returns the task registry.
//
// This allows callers to access the task registry for advanced operations.
// Returns nil if no task registry is configured.
func (s *OrchestrationService) GetTaskRegistry() *Registry {
	return s.taskRegistry
}

// GetToolRegistry returns the tool registry.
//
// This allows callers to access the tool registry for advanced operations.
// Returns nil if no tool registry is configured.
func (s *OrchestrationService) GetToolRegistry() *tools.Registry {
	return s.toolRegistry
}
