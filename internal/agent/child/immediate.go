package child

import (
	"context"

	"github.com/dmytrogajewski/spin/internal/agent/subagent"
	"github.com/dmytrogajewski/spin/internal/agent/tasks"
)

// ImmediateStarter starts a child and returns after non-blocking message/send.
func ImmediateStarter(bin, workDir string, runner HookRunner) subagent.BackgroundStarter {
	return func(ctx context.Context, spec *subagent.Spec, query string) (string, tasks.Handle, error) {
		proc, err := StartIfAllowed(ctx, runner, bin, specName(spec), workDir)
		if err != nil {
			return "", nil, err
		}

		task, sendErr := proc.SendImmediate(ctx, query)
		if sendErr != nil {
			_ = proc.Close()

			return "", nil, sendErr
		}

		return task.ID, NewTaskHandle(proc, task.ID), nil
	}
}
