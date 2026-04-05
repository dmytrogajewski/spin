package tools

import (
	"context"
	"time"
)

// TaskStatus represents the lifecycle state of a background task.
// This mirrors executor.TaskState to avoid import cycles.
type TaskStatus int

const (
	// TaskStatusRunning indicates the task process is actively executing.
	TaskStatusRunning TaskStatus = iota
	// TaskStatusCompleted indicates the task process exited with code 0.
	TaskStatusCompleted
	// TaskStatusFailed indicates the task process exited with a non-zero code.
	TaskStatusFailed
	// TaskStatusKilled indicates the task was terminated by the manager.
	TaskStatusKilled
)

// taskStatusNames maps [TaskStatus] values to human-readable strings.
var taskStatusNames = []string{
	"running",
	"completed",
	"failed",
	"killed",
}

// String returns the string representation of [TaskStatus].
func (s TaskStatus) String() string {
	if int(s) < len(taskStatusNames) {
		return taskStatusNames[s]
	}

	return unknownStatus
}

// TaskSnapshot is a read-only snapshot of a background task for listing.
// This mirrors executor.TaskInfo to avoid import cycles between tools and executor.
type TaskSnapshot struct {
	// ID is the unique task identifier.
	ID string
	// Command is the original command string.
	Command string
	// Status is the current task lifecycle state.
	Status TaskStatus
	// StartedAt is when the task was launched.
	StartedAt time.Time
	// ExitCode is the process exit code (-1 if still running or killed).
	ExitCode int
}

// TaskManager manages background tasks (to avoid import cycle with executor package).
// The concrete implementation is [executor.BackgroundTaskManager].
type TaskManager interface {
	// List returns snapshots of all managed tasks.
	List(ctx context.Context) []TaskSnapshot
	// GetOutput returns the last maxLines of output for a task.
	GetOutput(ctx context.Context, taskID string, maxLines int) (string, error)
	// Kill terminates a running task.
	Kill(ctx context.Context, taskID string) error
}

// TaskStarter starts background tasks. Separated from [TaskManager] because
// starting a process requires shell parsing and env setup that read-only
// management tools do not need.
type TaskStarter interface {
	// Start launches a command in the background and returns the task ID
	// and any initial output captured during startup.
	Start(ctx context.Context, command string, workDir string) (taskID string, initialOutput string, err error)
}
