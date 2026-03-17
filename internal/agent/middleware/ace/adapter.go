package ace

import (
	"context"

	"github.com/dmytrogajewski/spin/internal/agent"
	"github.com/dmytrogajewski/spin/internal/agent/harness"
)

// HarnessAdapter adapts the ACE Middleware to the harness.Middleware interface.
// It bridges the type differences between harness and agent packages,
// reading the trajectory.Context from IterationContext.TrajectoryCtx.
type HarnessAdapter struct {
	inner *Middleware
}

// NewHarnessAdapter creates a harness.Middleware adapter wrapping an ACE Middleware.
func NewHarnessAdapter(mw *Middleware) *HarnessAdapter {
	return &HarnessAdapter{inner: mw}
}

// BeforeTurn implements harness.Middleware.
// Reads TrajectoryCtx from IterationContext, updates its turn, and delegates.
func (a *HarnessAdapter) BeforeTurn(
	ctx context.Context, iterCtx *harness.IterationContext,
) {
	trajCtx := iterCtx.TrajectoryCtx
	if trajCtx == nil {
		return
	}

	trajCtx.CurrentTurn = iterCtx.Turn
	a.inner.BeforeTurn(ctx, trajCtx, iterCtx.Turn)
}

// AfterExecution implements harness.Middleware.
// Converts harness.Response to agent.Response and delegates to the inner middleware.
func (a *HarnessAdapter) AfterExecution(
	ctx context.Context,
	iterCtx *harness.IterationContext,
	resp *harness.Response,
) {
	trajCtx := iterCtx.TrajectoryCtx
	if trajCtx == nil {
		return
	}

	agentResp := &agent.Response{
		Output:       resp.Output,
		Success:      resp.FinishReason == harness.FinishReasonStop,
		FinishReason: resp.FinishReason,
		Duration:     resp.Duration,
	}

	a.inner.AfterExecution(ctx, trajCtx, agentResp)
}
