package concurrency

import "context"

// PipelineStage processes a state and may modify it.
// Returning an error stops the pipeline.
type PipelineStage[State any] func(ctx context.Context, state *State) error

// PipelineConfig configures a [Pipeline].
type PipelineConfig[State any] struct {
	// Halted is an optional predicate that checks if the pipeline should stop
	// before running the next stage. If nil, the pipeline runs all stages.
	Halted func(*State) bool
}

// Pipeline runs a sequence of stages in order.
// Stops on the first error or when the halted predicate returns true.
type Pipeline[State any] struct {
	stages []PipelineStage[State]
	config PipelineConfig[State]
}

// NewPipeline creates a pipeline with the given configuration and stages.
func NewPipeline[State any](config PipelineConfig[State], stages ...PipelineStage[State]) *Pipeline[State] {
	return &Pipeline[State]{
		stages: stages,
		config: config,
	}
}

// Run executes all stages in order.
// Stops on the first error or when the halted predicate returns true.
func (p *Pipeline[State]) Run(ctx context.Context, state *State) error {
	for _, stage := range p.stages {
		if p.config.Halted != nil && p.config.Halted(state) {
			return nil
		}

		if err := stage(ctx, state); err != nil {
			return err
		}
	}

	return nil
}
