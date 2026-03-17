// Package retrieval provides a context assembly pipeline that gathers
// fragments from multiple sources into a unified retrieval result.
package retrieval

import (
	"context"

	"github.com/dmytrogajewski/spin/internal/ace/trajectory"
)

// Fragment represents a single piece of assembled context from a source.
type Fragment struct {
	Source  string `json:"source"`
	Content string `json:"content"`
}

// Request carries input needed for context retrieval.
type Request struct {
	Turn          int
	Query         string
	TrajectoryCtx *trajectory.Context
}

// Result holds the assembled context fragments from all sources.
type Result struct {
	Fragments []Fragment
}

// Source provides named context fragments for the assembly pipeline.
type Source interface {
	// Name returns the unique identifier for this source.
	Name() string

	// Retrieve gathers context fragments for the given request.
	Retrieve(ctx context.Context, req Request) ([]Fragment, error)
}

// Pipeline assembles context from multiple registered sources.
type Pipeline struct {
	sources []Source
}

// NewPipeline creates a Pipeline with the given sources.
func NewPipeline(sources ...Source) *Pipeline {
	return &Pipeline{sources: sources}
}

// Assemble gathers context fragments from all registered sources.
// Sources are queried in registration order. If a source returns an error,
// it is skipped and the pipeline continues with remaining sources.
func (p *Pipeline) Assemble(ctx context.Context, req Request) Result {
	var fragments []Fragment

	for _, src := range p.sources {
		frags, err := src.Retrieve(ctx, req)
		if err != nil {
			continue
		}

		fragments = append(fragments, frags...)
	}

	return Result{Fragments: fragments}
}
