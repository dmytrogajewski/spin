package tools_test

// Journey: specs/journeys/JOURNEY-R3.2.md.

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/dmytrogajewski/spin/internal/tools"
)

// mockTaskManager implements [tools.TaskManager] for testing.
type mockTaskManager struct {
	tasks  []tools.TaskSnapshot
	output string
	err    error
}

func (m *mockTaskManager) List(_ context.Context) []tools.TaskSnapshot {
	return m.tasks
}

func (m *mockTaskManager) GetOutput(_ context.Context, _ string, _ int) (string, error) {
	return m.output, m.err
}

func (m *mockTaskManager) Kill(_ context.Context, _ string) error {
	return m.err
}

// mockTaskStarter implements [tools.TaskStarter] for testing.
type mockTaskStarter struct {
	taskID        string
	initialOutput string
	err           error
	lastCommand   string
	lastWorkDir   string
}

func (m *mockTaskStarter) Start(_ context.Context, command, workDir string) (taskID, initialOutput string, err error) {
	m.lastCommand = command
	m.lastWorkDir = workDir

	return m.taskID, m.initialOutput, m.err
}

// errTaskNotFound is a test sentinel for task-not-found scenarios.
var errTaskNotFound = errors.New("task not found")

// errTaskNotRunning is a test sentinel for kill-on-completed scenarios.
var errTaskNotRunning = errors.New("task is not running")

// errMaxConcurrent is a test sentinel for max-tasks scenarios.
var errMaxConcurrent = errors.New("maximum concurrent tasks reached")

const (
	testTaskID      = "abc1234"
	testTaskCommand = "sleep 300"
)

// ListProcessesTool tests follow.

func TestListProcessesTool_Name(t *testing.T) {
	t.Parallel()

	tool := tools.NewListProcessesTool(nil)

	require.Equal(t, "list_processes", tool.Name())
}

func TestListProcessesTool_Schema(t *testing.T) {
	t.Parallel()

	tool := tools.NewListProcessesTool(nil)
	schema := tool.Schema()

	require.Equal(t, "function", schema.Type)
	require.Equal(t, "list_processes", schema.Function.Name)
	require.Equal(t, "object", schema.Function.Parameters.Type)
}

func TestListProcessesTool_ReturnsRunningTasks(t *testing.T) {
	t.Parallel()

	mgr := &mockTaskManager{
		tasks: []tools.TaskSnapshot{
			{
				ID:        testTaskID,
				Command:   testTaskCommand,
				Status:    tools.TaskStatusRunning,
				StartedAt: time.Now(),
				ExitCode:  -1,
			},
		},
	}

	tool := tools.NewListProcessesTool(mgr)
	result, err := tool.Execute(context.Background(), tools.ToolParameters{})

	require.NoError(t, err)
	require.True(t, result.Success)
	require.Contains(t, result.Output, testTaskID)
	require.Contains(t, result.Output, "running")
}

func TestListProcessesTool_EmptyList(t *testing.T) {
	t.Parallel()

	mgr := &mockTaskManager{tasks: []tools.TaskSnapshot{}}

	tool := tools.NewListProcessesTool(mgr)
	result, err := tool.Execute(context.Background(), tools.ToolParameters{})

	require.NoError(t, err)
	require.True(t, result.Success)
	require.Contains(t, result.Output, "No background tasks.")
}

func TestListProcessesTool_NilManager(t *testing.T) {
	t.Parallel()

	tool := tools.NewListProcessesTool(nil)
	result, err := tool.Execute(context.Background(), tools.ToolParameters{})

	require.NoError(t, err)
	require.True(t, result.Success)
	require.Contains(t, result.Output, "task manager not available")
}

func TestListProcessesTool_MultipleTasks(t *testing.T) {
	t.Parallel()

	mgr := &mockTaskManager{
		tasks: []tools.TaskSnapshot{
			{
				ID:      testTaskID,
				Command: testTaskCommand,
				Status:  tools.TaskStatusRunning,
			},
			{
				ID:       "def5678",
				Command:  "echo done",
				Status:   tools.TaskStatusCompleted,
				ExitCode: 0,
			},
		},
	}

	tool := tools.NewListProcessesTool(mgr)
	result, err := tool.Execute(context.Background(), tools.ToolParameters{})

	require.NoError(t, err)
	require.True(t, result.Success)
	require.Contains(t, result.Output, testTaskID)
	require.Contains(t, result.Output, "def5678")
	require.Contains(t, result.Output, "running")
	require.Contains(t, result.Output, "completed")
}

// GetProcessOutputTool tests follow.

func TestGetProcessOutputTool_Name(t *testing.T) {
	t.Parallel()

	tool := tools.NewGetProcessOutputTool(nil)

	require.Equal(t, "get_process_output", tool.Name())
}

func TestGetProcessOutputTool_Schema(t *testing.T) {
	t.Parallel()

	tool := tools.NewGetProcessOutputTool(nil)
	schema := tool.Schema()

	require.Equal(t, "function", schema.Type)
	require.Equal(t, "get_process_output", schema.Function.Name)
	require.Contains(t, schema.Function.Parameters.Properties, "task_id")
	require.Contains(t, schema.Function.Parameters.Properties, "max_lines")
	require.Contains(t, schema.Function.Parameters.Required, "task_id")
}

func TestGetProcessOutputTool_ReturnsOutput(t *testing.T) {
	t.Parallel()

	mgr := &mockTaskManager{output: "hello\nworld\n"}

	tool := tools.NewGetProcessOutputTool(mgr)

	params, paramErr := tools.FromMap(map[string]any{
		"task_id": testTaskID,
	})
	require.NoError(t, paramErr)

	result, err := tool.Execute(context.Background(), params)

	require.NoError(t, err)
	require.True(t, result.Success)
	require.Contains(t, result.Output, "hello")
	require.Contains(t, result.Output, "world")
}

func TestGetProcessOutputTool_DefaultMaxLines(t *testing.T) {
	t.Parallel()

	mgr := &mockTaskManager{output: "line1\n"}

	tool := tools.NewGetProcessOutputTool(mgr)

	params, paramErr := tools.FromMap(map[string]any{
		"task_id": testTaskID,
	})
	require.NoError(t, paramErr)

	result, err := tool.Execute(context.Background(), params)

	require.NoError(t, err)
	require.True(t, result.Success)
	require.Contains(t, result.Output, "line1")
}

func TestGetProcessOutputTool_EmptyOutput(t *testing.T) {
	t.Parallel()

	mgr := &mockTaskManager{output: ""}

	tool := tools.NewGetProcessOutputTool(mgr)

	params, paramErr := tools.FromMap(map[string]any{
		"task_id": testTaskID,
	})
	require.NoError(t, paramErr)

	result, err := tool.Execute(context.Background(), params)

	require.NoError(t, err)
	require.True(t, result.Success)
	require.Contains(t, result.Output, "No output available.")
}

func TestGetProcessOutputTool_InvalidID(t *testing.T) {
	t.Parallel()

	mgr := &mockTaskManager{err: errTaskNotFound}

	tool := tools.NewGetProcessOutputTool(mgr)

	params, paramErr := tools.FromMap(map[string]any{
		"task_id": "invalid",
	})
	require.NoError(t, paramErr)

	result, err := tool.Execute(context.Background(), params)

	require.NoError(t, err)
	require.False(t, result.Success)
	require.Contains(t, result.Error, "task not found")
}

func TestGetProcessOutputTool_MissingTaskID(t *testing.T) {
	t.Parallel()

	mgr := &mockTaskManager{}

	tool := tools.NewGetProcessOutputTool(mgr)
	result, err := tool.Execute(context.Background(), tools.ToolParameters{})

	require.NoError(t, err)
	require.False(t, result.Success)
	require.Contains(t, result.Error, "task_id parameter is required")
}

func TestGetProcessOutputTool_NilManager(t *testing.T) {
	t.Parallel()

	tool := tools.NewGetProcessOutputTool(nil)

	params, paramErr := tools.FromMap(map[string]any{
		"task_id": testTaskID,
	})
	require.NoError(t, paramErr)

	result, err := tool.Execute(context.Background(), params)

	require.NoError(t, err)
	require.True(t, result.Success)
	require.Contains(t, result.Output, "task manager not available")
}

// KillProcessTool tests follow.

func TestKillProcessTool_Name(t *testing.T) {
	t.Parallel()

	tool := tools.NewKillProcessTool(nil)

	require.Equal(t, "kill_process", tool.Name())
}

func TestKillProcessTool_Schema(t *testing.T) {
	t.Parallel()

	tool := tools.NewKillProcessTool(nil)
	schema := tool.Schema()

	require.Equal(t, "function", schema.Type)
	require.Equal(t, "kill_process", schema.Function.Name)
	require.Contains(t, schema.Function.Parameters.Properties, "task_id")
	require.Contains(t, schema.Function.Parameters.Required, "task_id")
}

func TestKillProcessTool_KillsTask(t *testing.T) {
	t.Parallel()

	mgr := &mockTaskManager{}

	tool := tools.NewKillProcessTool(mgr)

	params, paramErr := tools.FromMap(map[string]any{
		"task_id": testTaskID,
	})
	require.NoError(t, paramErr)

	result, err := tool.Execute(context.Background(), params)

	require.NoError(t, err)
	require.True(t, result.Success)
	require.Contains(t, result.Output, testTaskID)
	require.Contains(t, result.Output, "killed successfully")
}

func TestKillProcessTool_InvalidID(t *testing.T) {
	t.Parallel()

	mgr := &mockTaskManager{err: errTaskNotFound}

	tool := tools.NewKillProcessTool(mgr)

	params, paramErr := tools.FromMap(map[string]any{
		"task_id": "invalid",
	})
	require.NoError(t, paramErr)

	result, err := tool.Execute(context.Background(), params)

	require.NoError(t, err)
	require.False(t, result.Success)
	require.Contains(t, result.Error, "task not found")
}

func TestKillProcessTool_NotRunning(t *testing.T) {
	t.Parallel()

	mgr := &mockTaskManager{err: errTaskNotRunning}

	tool := tools.NewKillProcessTool(mgr)

	params, paramErr := tools.FromMap(map[string]any{
		"task_id": testTaskID,
	})
	require.NoError(t, paramErr)

	result, err := tool.Execute(context.Background(), params)

	require.NoError(t, err)
	require.False(t, result.Success)
	require.Contains(t, result.Error, "task is not running")
}

func TestKillProcessTool_MissingTaskID(t *testing.T) {
	t.Parallel()

	mgr := &mockTaskManager{}

	tool := tools.NewKillProcessTool(mgr)
	result, err := tool.Execute(context.Background(), tools.ToolParameters{})

	require.NoError(t, err)
	require.False(t, result.Success)
	require.Contains(t, result.Error, "task_id parameter is required")
}

func TestKillProcessTool_NilManager(t *testing.T) {
	t.Parallel()

	tool := tools.NewKillProcessTool(nil)

	params, paramErr := tools.FromMap(map[string]any{
		"task_id": testTaskID,
	})
	require.NoError(t, paramErr)

	result, err := tool.Execute(context.Background(), params)

	require.NoError(t, err)
	require.True(t, result.Success)
	require.Contains(t, result.Output, "task manager not available")
}

// StartProcessTool tests follow.

func TestStartProcessTool_Name(t *testing.T) {
	t.Parallel()

	tool := tools.NewStartProcessTool(nil)

	require.Equal(t, "start_process", tool.Name())
}

func TestStartProcessTool_Schema(t *testing.T) {
	t.Parallel()

	tool := tools.NewStartProcessTool(nil)
	schema := tool.Schema()

	require.Equal(t, "function", schema.Type)
	require.Equal(t, "start_process", schema.Function.Name)
	require.Contains(t, schema.Function.Parameters.Properties, "command")
	require.Contains(t, schema.Function.Parameters.Properties, "working_directory")
	require.Contains(t, schema.Function.Parameters.Required, "command")
}

func TestStartProcessTool_StartsProcess(t *testing.T) {
	t.Parallel()

	starter := &mockTaskStarter{
		taskID:        "bg12345",
		initialOutput: "Server started on port 8080",
	}

	tool := tools.NewStartProcessTool(starter)

	params, paramErr := tools.FromMap(map[string]any{
		"command": "python3 server.py",
	})
	require.NoError(t, paramErr)

	result, err := tool.Execute(context.Background(), params)

	require.NoError(t, err)
	require.True(t, result.Success)
	require.Contains(t, result.Output, "bg12345")
	require.Contains(t, result.Output, "Background process started")
	require.Contains(t, result.Output, "Server started on port 8080")
	require.Equal(t, "python3 server.py", starter.lastCommand)
}

func TestStartProcessTool_WithWorkDir(t *testing.T) {
	t.Parallel()

	starter := &mockTaskStarter{
		taskID: "bg99999",
	}

	tool := tools.NewStartProcessTool(starter)

	params, paramErr := tools.FromMap(map[string]any{
		"command":           "npm start",
		"working_directory": "/home/user/project",
	})
	require.NoError(t, paramErr)

	result, err := tool.Execute(context.Background(), params)

	require.NoError(t, err)
	require.True(t, result.Success)
	require.Contains(t, result.Output, "bg99999")
	require.Equal(t, "/home/user/project", starter.lastWorkDir)
}

func TestStartProcessTool_NoInitialOutput(t *testing.T) {
	t.Parallel()

	starter := &mockTaskStarter{
		taskID:        "bg00001",
		initialOutput: "",
	}

	tool := tools.NewStartProcessTool(starter)

	params, paramErr := tools.FromMap(map[string]any{
		"command": "sleep 300",
	})
	require.NoError(t, paramErr)

	result, err := tool.Execute(context.Background(), params)

	require.NoError(t, err)
	require.True(t, result.Success)
	require.Contains(t, result.Output, "bg00001")
	require.NotContains(t, result.Output, "Initial output")
}

func TestStartProcessTool_StartError(t *testing.T) {
	t.Parallel()

	starter := &mockTaskStarter{
		err: errMaxConcurrent,
	}

	tool := tools.NewStartProcessTool(starter)

	params, paramErr := tools.FromMap(map[string]any{
		"command": "python3 server.py",
	})
	require.NoError(t, paramErr)

	result, err := tool.Execute(context.Background(), params)

	require.NoError(t, err)
	require.False(t, result.Success)
	require.Contains(t, result.Error, "maximum concurrent tasks reached")
}

func TestStartProcessTool_MissingCommand(t *testing.T) {
	t.Parallel()

	starter := &mockTaskStarter{taskID: "bg12345"}

	tool := tools.NewStartProcessTool(starter)
	result, err := tool.Execute(context.Background(), tools.ToolParameters{})

	require.NoError(t, err)
	require.False(t, result.Success)
	require.Contains(t, result.Error, "command parameter is required")
}

func TestStartProcessTool_NilManager(t *testing.T) {
	t.Parallel()

	tool := tools.NewStartProcessTool(nil)

	params, paramErr := tools.FromMap(map[string]any{
		"command": "python3 server.py",
	})
	require.NoError(t, paramErr)

	result, err := tool.Execute(context.Background(), params)

	require.NoError(t, err)
	require.True(t, result.Success)
	require.Contains(t, result.Output, "task manager not available")
}

// TaskStatus tests follow.

func TestTaskStatus_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		status tools.TaskStatus
		want   string
	}{
		{tools.TaskStatusRunning, "running"},
		{tools.TaskStatusCompleted, "completed"},
		{tools.TaskStatusFailed, "failed"},
		{tools.TaskStatusKilled, "killed"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			t.Parallel()

			require.Equal(t, tt.want, tt.status.String())
		})
	}
}
