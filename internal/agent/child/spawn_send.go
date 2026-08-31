package child

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/dmytrogajewski/spin/internal/protocol/a2a"
)

const artifactNameStderr = "stderr"

// Send blocks until the Task is completed or failed and returns artifact text.
func (proc *Process) Send(ctx context.Context, query string) (string, error) {
	if proc.client == nil {
		return proc.recordCrash()
	}

	task, err := proc.client.SendMessage(ctx, userMessage(query))
	if err != nil {
		return proc.recordCrash()
	}

	task, err = proc.waitTerminal(ctx, task)
	if err != nil {
		return proc.recordCrash()
	}

	proc.task = task

	return firstArtifactText(task), nil
}

// SendImmediate calls message/send with returnImmediately and returns the task id now.
func (proc *Process) SendImmediate(ctx context.Context, query string) (*a2a.Task, error) {
	if proc.client == nil {
		_, err := proc.recordCrash()

		return proc.task, err
	}

	task, err := proc.client.SendMessageImmediate(ctx, userMessage(query))
	if err != nil {
		_, crashErr := proc.recordCrash()

		return proc.task, crashErr
	}

	proc.task = task

	return task, nil
}

// GetTask calls tasks/get on the child.
func (proc *Process) GetTask(ctx context.Context, id string) (*a2a.Task, error) {
	if proc.client == nil {
		return nil, ErrChildCrashed
	}

	return proc.client.GetTask(ctx, id)
}

// CancelTask calls tasks/cancel on the child.
func (proc *Process) CancelTask(ctx context.Context, id string) (*a2a.Task, error) {
	if proc.client == nil {
		return nil, ErrChildCrashed
	}

	return proc.client.CancelTask(ctx, id)
}

func (proc *Process) waitTerminal(ctx context.Context, task *a2a.Task) (*a2a.Task, error) {
	current := task
	for current != nil && !current.Status.State.Terminal() {
		if err := ctx.Err(); err != nil {
			return nil, fmt.Errorf("child wait: %w", err)
		}

		next, getErr := proc.client.GetTask(ctx, current.ID)
		if getErr != nil {
			return nil, fmt.Errorf("child tasks/get: %w", getErr)
		}

		current = next
	}

	return current, nil
}

func (proc *Process) recordCrash() (string, error) {
	proc.reap()
	text := proc.stderr.String()
	proc.task = failedTask(text)

	return text, fmt.Errorf("%w: %s", ErrChildCrashed, text)
}

func userMessage(query string) a2a.Message {
	return a2a.Message{
		MessageID: uuid.NewString(),
		Role:      a2a.RoleUser,
		Parts:     []a2a.Part{{Text: query, MediaType: mediaTextPlain}},
	}
}

func firstArtifactText(task *a2a.Task) string {
	if task == nil {
		return ""
	}

	for _, art := range task.Artifacts {
		for _, part := range art.Parts {
			if part.Text != "" {
				return part.Text
			}
		}
	}

	return ""
}

func failedTask(stderr string) *a2a.Task {
	return &a2a.Task{
		Status: a2a.TaskStatus{State: a2a.TaskStateFailed},
		Artifacts: []a2a.Artifact{{
			ArtifactID: artifactNameStderr,
			Name:       artifactNameStderr,
			Parts:      []a2a.Part{{Text: stderr, MediaType: mediaTextPlain}},
		}},
	}
}
