package executor

import (
	"time"

	"github.com/dmytrogajewski/spin/internal/safety"
	"github.com/dmytrogajewski/spin/pkg/alg/concurrency"
)

// PipelineContext carries data through pipeline stages.
type PipelineContext struct {
	// Command is the command being executed.
	Command *safety.Command

	// WorkDir is the working directory.
	WorkDir string

	// Env contains environment variables.
	Env map[string]string

	// Timeout is the execution timeout.
	Timeout time.Duration

	// IsServer indicates the command was detected as a long-running server.
	IsServer bool

	// Halted indicates the pipeline was halted by a stage.
	Halted bool

	// HaltErr is the reason for halting (may be nil).
	HaltErr error

	// Result carries the execution result through stages.
	Result *CommandResult

	values map[string]any
}

// NewPipelineContext creates a context for pipeline execution.
func NewPipelineContext(cmd *safety.Command) *PipelineContext {
	return &PipelineContext{
		Command: cmd,
		values:  make(map[string]any),
	}
}

// Halt stops the pipeline with an optional reason.
func (pc *PipelineContext) Halt(err error) {
	pc.Halted = true
	pc.HaltErr = err
}

// SetValue stores a key-value pair for inter-stage communication.
func (pc *PipelineContext) SetValue(key string, val any) {
	pc.values[key] = val
}

// GetValue retrieves a value by key.
func (pc *PipelineContext) GetValue(key string) (any, bool) {
	val, ok := pc.values[key]

	return val, ok
}

// Stage processes a PipelineContext and may modify it.
type Stage = concurrency.PipelineStage[PipelineContext]

// Pipeline runs a sequence of stages.
type Pipeline = concurrency.Pipeline[PipelineContext]

// NewPipeline creates a pipeline with halt detection for the given stages.
func NewPipeline(stages ...Stage) *Pipeline {
	return concurrency.NewPipeline(concurrency.PipelineConfig[PipelineContext]{
		Halted: func(pc *PipelineContext) bool { return pc.Halted },
	}, stages...)
}
