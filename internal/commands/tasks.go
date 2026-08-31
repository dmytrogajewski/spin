package commands

import (
	"context"
	"errors"
	"fmt"

	"github.com/dmytrogajewski/spin/internal/agent/tasks"
	"github.com/dmytrogajewski/spin/internal/tools"
)

// ErrTaskRegistryUnavailable is returned when CommandContext is not a TaskSource.
var ErrTaskRegistryUnavailable = errors.New("task registry is not available")

// TaskSource is implemented by contexts that own the A2A registry.
type TaskSource interface {
	AgentTasks() *tasks.Registry
}

// ShellTaskSource is implemented by contexts that can list shell background tasks.
type ShellTaskSource interface {
	ShellTasks() *tools.ShellAdapter
}

func registryFrom(cmdCtx CommandContext) (*tasks.Registry, error) {
	src, ok := cmdCtx.(TaskSource)
	if !ok || src.AgentTasks() == nil {
		return nil, ErrTaskRegistryUnavailable
	}

	return src.AgentTasks(), nil
}

// TasksCommand lists agent and shell tasks in one view (kind=agent|shell).
type TasksCommand struct{}

// Name implements Command.
func (c *TasksCommand) Name() string {
	return "/tasks"
}

// Description implements Command.
func (c *TasksCommand) Description() string {
	return "List agent and shell tasks (kind=agent|shell)"
}

// Execute lists registry rows.
func (c *TasksCommand) Execute(ctx context.Context, _ []string, cmdCtx CommandContext) (string, error) {
	reg, err := registryFrom(cmdCtx)
	if err != nil {
		return "", err
	}

	return tasks.FormatView(tasks.Merge(reg.List(), shellsFrom(ctx, cmdCtx))), nil
}

func shellsFrom(ctx context.Context, cmdCtx CommandContext) []tasks.ShellSnapshot {
	src := shellSourceFrom(cmdCtx)
	if src == nil {
		return nil
	}

	return src.List(ctx)
}

func shellSourceFrom(cmdCtx CommandContext) *tools.ShellAdapter {
	src, ok := cmdCtx.(ShellTaskSource)
	if !ok {
		return nil
	}

	return src.ShellTasks()
}

// TaskCommand handles /task wait <id> and /task cancel <id>.
type TaskCommand struct{}

// Name implements Command.
func (c *TaskCommand) Name() string {
	return "/task"
}

// Description implements Command.
func (c *TaskCommand) Description() string {
	return "Wait or cancel a task: /task wait <id> | /task cancel <id>"
}

// Execute waits or cancels.
func (c *TaskCommand) Execute(ctx context.Context, args []string, cmdCtx CommandContext) (string, error) {
	if len(args) < taskSubcommandArgs {
		return "", errTaskUsage
	}

	reg, err := registryFrom(cmdCtx)
	if err != nil {
		return "", err
	}

	switch args[0] {
	case "wait":
		kind, raw := tasks.SplitID(args[1])
		if kind == tasks.KindShell {
			return "", tasks.ErrNotFound
		}

		rec, waitErr := reg.Wait(ctx, raw)
		if waitErr != nil {
			return "", waitErr
		}

		return tasks.Format([]tasks.Record{rec}), nil
	case "cancel":
		if cancelErr := tasks.CancelView(ctx, args[1], reg, shellSourceFrom(cmdCtx)); cancelErr != nil {
			return "", cancelErr
		}

		return fmt.Sprintf("canceled %s", args[1]), nil
	default:
		return "", errTaskUsage
	}
}

const taskSubcommandArgs = 2

var errTaskUsage = errors.New("usage: /task wait <id> | /task cancel <id>")
