package orchestration

import (
	"context"
	"fmt"

	"github.com/dmytrogajewski/spin/internal/tools"
)

// OrchestrationService handles tool execution and planning.
//
// This service centralizes orchestration logic that was previously embedded in Agent.
// It provides a clean interface for executing tools and coordinating execution plans.
//
// Note: Task management has been removed in favor of compile-time task creation
// via task.NewTask(). This eliminates the runtime registry pattern.
type OrchestrationService struct {
	toolExecutor *ToolExecutor
	toolRegistry *tools.Registry
	planner      *Plan
}

// NewOrchestrationService creates a new orchestration service with the given dependencies.
//
// All dependencies can be nil. When nil, methods that require these dependencies
// will return appropriate errors.
func NewOrchestrationService(
	toolExecutor *ToolExecutor,
	toolRegistry *tools.Registry,
) *OrchestrationService {
	return &OrchestrationService{
		toolExecutor: toolExecutor,
		toolRegistry: toolRegistry,
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

// GetToolRegistry returns the tool registry.
//
// This allows callers to access the tool registry for advanced operations.
// Returns nil if no tool registry is configured.
func (s *OrchestrationService) GetToolRegistry() *tools.Registry {
	return s.toolRegistry
}
