package orchestration

import (
	"context"
	"testing"

	"github.com/dmytrogajewski/spin/internal/task"
	"github.com/dmytrogajewski/spin/internal/tools"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewOrchestrationService(t *testing.T) {
	toolRegistry := tools.NewRegistry()
	taskRegistry := NewRegistry()
	toolExecutor := NewToolExecutor(ToolExecutorConfig{
		Registry: toolRegistry,
	})

	tests := []struct {
		name         string
		toolExecutor *ToolExecutor
		toolRegistry *tools.Registry
		taskRegistry *Registry
		wantNil      bool
	}{
		{
			name:         "with all dependencies",
			toolExecutor: toolExecutor,
			toolRegistry: toolRegistry,
			taskRegistry: taskRegistry,
			wantNil:      false,
		},
		{
			name:         "with nil tool executor",
			toolExecutor: nil,
			toolRegistry: toolRegistry,
			taskRegistry: taskRegistry,
			wantNil:      false, // Service allows nil
		},
		{
			name:         "with nil tool registry",
			toolExecutor: toolExecutor,
			toolRegistry: nil,
			taskRegistry: taskRegistry,
			wantNil:      false, // Service allows nil
		},
		{
			name:         "with nil task registry",
			toolExecutor: toolExecutor,
			toolRegistry: toolRegistry,
			taskRegistry: nil,
			wantNil:      false, // Service allows nil
		},
		{
			name:         "with all nil",
			toolExecutor: nil,
			toolRegistry: nil,
			taskRegistry: nil,
			wantNil:      false, // Service allows nil
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewOrchestrationService(tt.toolExecutor, tt.toolRegistry, tt.taskRegistry)

			if tt.wantNil {
				assert.Nil(t, svc)
			} else {
				assert.NotNil(t, svc)
			}
		})
	}
}

func TestOrchestrationService_ExecuteTool(t *testing.T) {
	registry := tools.NewRegistry()
	_ = registry.Register(tools.NewReadFileTool())

	toolExecutor := NewToolExecutor(ToolExecutorConfig{
		Registry: registry,
	})

	svc := NewOrchestrationService(toolExecutor, registry, nil)

	tests := []struct {
		name    string
		call    *ToolCall
		wantErr bool
	}{
		{
			name: "valid tool call",
			call: &ToolCall{
				ID: "call-1",
				Function: ToolCallFunction{
					Name:      "read_file",
					Arguments: `{"path": "test.txt"}`,
				},
			},
			wantErr: false,
		},
		{
			name: "invalid tool name",
			call: &ToolCall{
				ID: "call-2",
				Function: ToolCallFunction{
					Name:      "nonexistent_tool",
					Arguments: `{}`,
				},
			},
			wantErr: false, // Returns result with error, not Go error
		},
		{
			name:    "nil call",
			call:    nil,
			wantErr: false, // Returns result with error
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			result, err := svc.ExecuteTool(ctx, tt.call)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.NotNil(t, result)
		})
	}
}

func TestOrchestrationService_ExecuteTool_NilExecutor(t *testing.T) {
	svc := NewOrchestrationService(nil, nil, nil)

	call := &ToolCall{
		ID: "call-1",
		Function: ToolCallFunction{
			Name:      "read_file",
			Arguments: `{}`,
		},
	}

	ctx := context.Background()
	result, err := svc.ExecuteTool(ctx, call)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "tool executor not configured")
}

func TestOrchestrationService_ExecuteBatch(t *testing.T) {
	registry := tools.NewRegistry()
	_ = registry.Register(tools.NewReadFileTool())

	toolExecutor := NewToolExecutor(ToolExecutorConfig{
		Registry: registry,
	})

	svc := NewOrchestrationService(toolExecutor, registry, nil)

	tests := []struct {
		name      string
		calls     []*ToolCall
		wantCount int
		wantErr   bool
	}{
		{
			name:      "empty batch",
			calls:     []*ToolCall{},
			wantCount: 0,
			wantErr:   false,
		},
		{
			name: "single call",
			calls: []*ToolCall{
				{
					ID: "call-1",
					Function: ToolCallFunction{
						Name:      "read_file",
						Arguments: `{"path": "test.txt"}`,
					},
				},
			},
			wantCount: 1,
			wantErr:   false,
		},
		{
			name: "multiple calls",
			calls: []*ToolCall{
				{
					ID: "call-1",
					Function: ToolCallFunction{
						Name:      "read_file",
						Arguments: `{"path": "test1.txt"}`,
					},
				},
				{
					ID: "call-2",
					Function: ToolCallFunction{
						Name:      "read_file",
						Arguments: `{"path": "test2.txt"}`,
					},
				},
			},
			wantCount: 2,
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			results, err := svc.ExecuteBatch(ctx, tt.calls)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Len(t, results, tt.wantCount)
		})
	}
}

func TestOrchestrationService_ExecuteBatch_NilExecutor(t *testing.T) {
	svc := NewOrchestrationService(nil, nil, nil)

	calls := []*ToolCall{
		{
			ID: "call-1",
			Function: ToolCallFunction{
				Name:      "read_file",
				Arguments: `{}`,
			},
		},
	}

	ctx := context.Background()
	results, err := svc.ExecuteBatch(ctx, calls)

	assert.Error(t, err)
	assert.Nil(t, results)
}

func TestOrchestrationService_GetTask(t *testing.T) {
	taskRegistry := NewRegistry()
	_ = taskRegistry.Register("test-task", task.NewRegular())

	svc := NewOrchestrationService(nil, nil, taskRegistry)

	tests := []struct {
		name     string
		taskName string
		wantErr  bool
	}{
		{
			name:     "existing task",
			taskName: "test-task",
			wantErr:  false,
		},
		{
			name:     "nonexistent task",
			taskName: "nonexistent",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := svc.GetTask(tt.taskName)

			if tt.wantErr {
				require.Error(t, err)
				assert.Nil(t, result)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, result)
			}
		})
	}
}

func TestOrchestrationService_GetTask_NilRegistry(t *testing.T) {
	svc := NewOrchestrationService(nil, nil, nil)

	task, err := svc.GetTask("regular")

	assert.Error(t, err)
	assert.Nil(t, task)
	assert.Contains(t, err.Error(), "task registry not configured")
}

func TestOrchestrationService_GetDefaultTask(t *testing.T) {
	taskRegistry := NewRegistry()
	_ = taskRegistry.Register("default-task", task.NewRegular())
	_ = taskRegistry.SetDefault("default-task")

	svc := NewOrchestrationService(nil, nil, taskRegistry)

	result, err := svc.GetDefaultTask()

	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestOrchestrationService_GetDefaultTask_NilRegistry(t *testing.T) {
	svc := NewOrchestrationService(nil, nil, nil)

	task, err := svc.GetDefaultTask()

	assert.Error(t, err)
	assert.Nil(t, task)
	assert.Contains(t, err.Error(), "task registry not configured")
}

func TestOrchestrationService_ListTasks(t *testing.T) {
	taskRegistry := NewRegistry()
	_ = taskRegistry.Register("task1", task.NewRegular())
	_ = taskRegistry.Register("task2", task.NewCompact())

	svc := NewOrchestrationService(nil, nil, taskRegistry)

	tasks := svc.ListTasks()

	assert.Len(t, tasks, 2)
	assert.Contains(t, tasks, "task1")
	assert.Contains(t, tasks, "task2")
}

func TestOrchestrationService_ListTasks_NilRegistry(t *testing.T) {
	svc := NewOrchestrationService(nil, nil, nil)

	tasks := svc.ListTasks()

	assert.Len(t, tasks, 0)
}

func TestOrchestrationService_GetPlanner(t *testing.T) {
	svc := NewOrchestrationService(nil, nil, nil)

	planner := svc.GetPlanner()
	assert.Nil(t, planner)
}

func TestOrchestrationService_SetPlanner(t *testing.T) {
	svc := NewOrchestrationService(nil, nil, nil)

	plan := NewPlan("test task")
	svc.SetPlanner(plan)

	retrieved := svc.GetPlanner()
	assert.NotNil(t, retrieved)
	assert.Equal(t, "test task", retrieved.Task)
}

// Benchmark tests
func BenchmarkOrchestrationService_ExecuteTool(b *testing.B) {
	registry := tools.NewRegistry()
	_ = registry.Register(tools.NewReadFileTool())

	toolExecutor := NewToolExecutor(ToolExecutorConfig{
		Registry: registry,
	})

	svc := NewOrchestrationService(toolExecutor, registry, nil)

	call := &ToolCall{
		ID: "call-1",
		Function: ToolCallFunction{
			Name:      "read_file",
			Arguments: `{"path": "test.txt"}`,
		},
	}

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = svc.ExecuteTool(ctx, call)
	}
}

func BenchmarkOrchestrationService_GetTask(b *testing.B) {
	taskRegistry := NewRegistry()
	_ = taskRegistry.Register("test-task", task.NewRegular())

	svc := NewOrchestrationService(nil, nil, taskRegistry)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = svc.GetTask("test-task")
	}
}
