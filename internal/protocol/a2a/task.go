package a2a

// TaskState is an A2A task lifecycle state (proto enum string).
type TaskState string

const (
	// TaskStateUnspecified is an unknown or indeterminate state.
	TaskStateUnspecified TaskState = "TASK_STATE_UNSPECIFIED"
	// TaskStateSubmitted means the task was accepted.
	TaskStateSubmitted TaskState = "TASK_STATE_SUBMITTED"
	// TaskStateWorking means the agent is processing the task.
	TaskStateWorking TaskState = "TASK_STATE_WORKING"
	// TaskStateCompleted is a successful terminal state.
	TaskStateCompleted TaskState = "TASK_STATE_COMPLETED"
	// TaskStateFailed is a failed terminal state.
	TaskStateFailed TaskState = "TASK_STATE_FAILED"
	// TaskStateCanceled is a canceled terminal state.
	TaskStateCanceled TaskState = "TASK_STATE_CANCELED"
	// TaskStateInputRequired means the agent needs more user input.
	TaskStateInputRequired TaskState = "TASK_STATE_INPUT_REQUIRED"
	// TaskStateRejected is a terminal refusal to perform the task.
	TaskStateRejected TaskState = "TASK_STATE_REJECTED"
	// TaskStateAuthRequired means authentication is required to continue.
	TaskStateAuthRequired TaskState = "TASK_STATE_AUTH_REQUIRED"
)

// Terminal reports whether the state cannot accept further work.
func (state TaskState) Terminal() bool {
	switch state {
	case TaskStateCompleted, TaskStateFailed, TaskStateCanceled, TaskStateRejected:
		return true
	case TaskStateUnspecified, TaskStateSubmitted, TaskStateWorking,
		TaskStateInputRequired, TaskStateAuthRequired:
		return false
	default:
		return false
	}
}

// Task is the A2A unit of work.
type Task struct {
	Artifacts []Artifact `json:"artifacts,omitempty"`
	ContextID string     `json:"contextId,omitempty"`
	History   []Message  `json:"history,omitempty"`
	ID        string     `json:"id"`
	Status    TaskStatus `json:"status"`
}

// TaskStatus is the current state of a Task.
type TaskStatus struct {
	Message   *Message  `json:"message,omitempty"`
	State     TaskState `json:"state"`
	Timestamp string    `json:"timestamp,omitempty"`
}
