package child

import (
	"context"
	"errors"
	"fmt"

	"github.com/dmytrogajewski/spin/internal/agent/tasks"
	"github.com/dmytrogajewski/spin/internal/protocol/a2a"
)

// TaskHandle adapts a Process to tasks.Handle.
type TaskHandle struct {
	proc *Process
	id   string
}

// NewTaskHandle binds a child process to a task id.
func NewTaskHandle(proc *Process, id string) *TaskHandle {
	return &TaskHandle{proc: proc, id: id}
}

// Get maps tasks/get onto a registry state string.
func (h *TaskHandle) Get(ctx context.Context) (string, error) {
	task, err := h.proc.GetTask(ctx, h.id)
	if err != nil {
		return "", fmt.Errorf("child handle get: %w", err)
	}

	return mapTaskState(task.Status.State), nil
}

// Cancel maps to tasks/cancel.
func (h *TaskHandle) Cancel(ctx context.Context) error {
	if _, err := h.proc.CancelTask(ctx, h.id); err != nil {
		if isNotCancelable(err) {
			return nil
		}

		return fmt.Errorf("child handle cancel: %w", err)
	}

	return nil
}

func isNotCancelable(err error) bool {
	var rpcErr *a2a.RPCError

	return errors.As(err, &rpcErr) && rpcErr.Code == a2a.CodeTaskNotCancelable
}

// SignalTERM sends SIGTERM to the child.
func (h *TaskHandle) SignalTERM() error {
	return h.proc.SignalTERM()
}

func mapTaskState(state a2a.TaskState) string {
	switch state {
	case a2a.TaskStateCompleted:
		return tasks.StateCompleted
	case a2a.TaskStateFailed, a2a.TaskStateRejected:
		return tasks.StateFailed
	case a2a.TaskStateCanceled:
		return tasks.StateCanceled
	default:
		return tasks.StateWorking
	}
}
