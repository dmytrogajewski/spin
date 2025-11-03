package agent

import (
	"strings"

	"github.com/dmytrogajewski/spin/internal/ace/trajectory"
)

// buildQueryFromContext constructs a retrieval query based on trajectory state and trigger.
//
// Query composition strategy:
// - TriggerInitial: base query only
// - TriggerError: base query + error patterns (last N steps)
// - TriggerToolChange: base query + tool names (last N steps)
// - TriggerInterval: base query + concepts (last N steps)
//
// Returns space-separated string compatible with embedding search.
//
// Requires: ctx != nil, trigger != ""
// Ensures: result != ""
func (a *Agent) buildQueryFromContext(
	ctx *trajectory.TrajectoryContext,
	trigger trajectory.TriggerType,
) string {
	// Start with base query
	parts := []string{ctx.Query}

	// Add context based on trigger type
	switch trigger {
	case trajectory.TriggerInitial:
		// Base query only
		return ctx.Query

	case trajectory.TriggerError:
		// Add error patterns from recent steps
		errorPatterns := trajectory.ExtractErrorPatterns(ctx.Steps, a.config.ACE.Retrieval.ProgressiveContext.ErrorLookback)
		parts = append(parts, errorPatterns...)

	case trajectory.TriggerToolChange:
		// Add tool names from recent steps
		tools := ctx.GetRecentTools(a.config.ACE.Retrieval.ProgressiveContext.ToolChangeLookback)
		parts = append(parts, tools...)

	case trajectory.TriggerInterval:
		// Add concepts from recent steps (fixed lookback of 5 for now)
		concepts := trajectory.ExtractConcepts(ctx.Steps, 5)
		parts = append(parts, concepts...)
	}

	return strings.Join(parts, " ")
}
