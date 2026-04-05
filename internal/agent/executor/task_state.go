package executor

import "time"

// TaskState represents the lifecycle state of a background task.
type TaskState int

const (
	// TaskRunning indicates the task process is actively executing.
	TaskRunning TaskState = iota
	// TaskCompleted indicates the task process exited with code 0.
	TaskCompleted
	// TaskFailed indicates the task process exited with a non-zero code.
	TaskFailed
	// TaskKilled indicates the task was terminated by the manager.
	TaskKilled
)

// String returns the string representation of [TaskState].
func (s TaskState) String() string {
	names := []string{
		"running",
		"completed",
		"failed",
		"killed",
	}

	if int(s) < len(names) {
		return names[s]
	}

	return "unknown"
}

// TaskInfo is a read-only snapshot of a background task for listing.
type TaskInfo struct {
	// ID is the 7-char hex task identifier.
	ID string
	// Command is the original command string.
	Command string
	// State is the current task lifecycle state.
	State TaskState
	// StartedAt is when the task was launched.
	StartedAt time.Time
	// ExitCode is the process exit code (-1 if still running or killed).
	ExitCode int
}
