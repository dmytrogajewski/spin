package child

import (
	"context"
	"errors"
	"fmt"

	"github.com/dmytrogajewski/spin/internal/safety/hooks"
)

// ErrStartBlocked indicates SUBAGENT_START vetoed process start (exit 2).
var ErrStartBlocked = errors.New("child: SUBAGENT_START blocked")

// HookRunner executes lifecycle hooks. Nil is a no-op admit.
type HookRunner interface {
	Execute(ctx context.Context, event hooks.Event, evtCtx hooks.EventContext) hooks.HookResult
}

// StartIfAllowed runs blocking SUBAGENT_START, then StartSpec. Exit 2 yields no pid.
func StartIfAllowed(
	ctx context.Context,
	runner HookRunner,
	bin, specName, workDir string,
) (*Process, error) {
	if err := admitStart(ctx, runner, specName, workDir); err != nil {
		return nil, err
	}

	return StartSpec(ctx, bin, specName, workDir)
}

func admitStart(ctx context.Context, runner HookRunner, specName, workDir string) error {
	if runner == nil {
		return nil
	}

	result := runner.Execute(ctx, hooks.EventSubagentStart, hooks.EventContext{
		WorkDir:  workDir,
		ToolName: specName,
	})
	if result.Blocked {
		return fmt.Errorf("%w: %s", ErrStartBlocked, result.Reason)
	}

	return nil
}

func fireStop(ctx context.Context, runner HookRunner, specName, workDir string) {
	if runner == nil {
		return
	}

	runner.Execute(context.WithoutCancel(ctx), hooks.EventSubagentStop, hooks.EventContext{
		WorkDir:  workDir,
		ToolName: specName,
	})
}
