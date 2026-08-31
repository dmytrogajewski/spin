package child

import (
	"context"
	"errors"

	"github.com/dmytrogajewski/spin/internal/agent/subagent"
	"github.com/dmytrogajewski/spin/internal/events"
)

// NewExecutor returns a production subagent.Executor that starts a child process.
func NewExecutor(bin, workDir string, emitter *events.EventEmitter, runner HookRunner) subagent.Executor {
	return func(ctx context.Context, spec *subagent.Spec, query string) (string, error) {
		return runChild(ctx, bin, workDir, emitter, runner, spec, query)
	}
}

func runChild(
	ctx context.Context,
	bin, workDir string,
	emitter *events.EventEmitter,
	runner HookRunner,
	spec *subagent.Spec,
	query string,
) (string, error) {
	proc, err := StartIfAllowed(ctx, runner, bin, specName(spec), workDir)
	if errors.Is(err, ErrStartBlocked) {
		emitHookVeto(emitter, spec, err)

		return "", err
	}

	defer fireStop(ctx, runner, specName(spec), workDir)

	emitSpawn(emitter, spec, query)

	if proc != nil {
		defer proc.Close()
	}

	if err != nil {
		summary := crashSummary(proc)
		emitComplete(emitter, spec, summary)

		return summary, err
	}

	summary, sendErr := proc.Send(ctx, query)
	emitComplete(emitter, spec, summary)

	return summary, sendErr
}

func specName(spec *subagent.Spec) string {
	if spec == nil {
		return ""
	}

	return spec.Name
}

func crashSummary(proc *Process) string {
	if proc == nil {
		return ""
	}

	return firstArtifactText(proc.Task())
}

func emitSpawn(emitter *events.EventEmitter, spec *subagent.Spec, query string) {
	if emitter == nil || spec == nil {
		return
	}

	emitter.Emit(events.Event{
		Type: events.EventSubagentSpawn,
		Data: events.SubagentSpawnData{AgentType: spec.Name, Query: query},
	})
}

func emitHookVeto(emitter *events.EventEmitter, spec *subagent.Spec, err error) {
	if emitter == nil {
		return
	}

	emitter.Emit(events.Event{
		Type: events.EventHookVeto,
		Data: events.HookVetoData{
			Event:  "SUBAGENT_START",
			Reason: err.Error(),
			Spec:   specName(spec),
		},
	})
}

func emitComplete(emitter *events.EventEmitter, spec *subagent.Spec, summary string) {
	if emitter == nil || spec == nil {
		return
	}

	emitter.Emit(events.Event{
		Type: events.EventSubagentComplete,
		Data: events.SubagentCompleteData{AgentType: spec.Name, Summary: summary},
	})
}
