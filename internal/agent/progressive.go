package agent

import (
	"github.com/dmytrogajewski/spin/internal/ace/trajectory"
)

// shouldRetrieveProgressive determines if retrieval should be triggered based on trajectory state.
// Returns (shouldRetrieve, triggerType).
//
// Trigger priority (first match wins):
// 1. TriggerInitial - Always on turn 0
// 2. TriggerError - Recent error detected
// 3. TriggerToolChange - Tool usage changed
// 4. TriggerInterval - Cache TTL expired
//
// Requires: ctx != nil
// Ensures: if shouldRetrieve, then triggerType is valid (not empty).
func (a *Agent) shouldRetrieveProgressive(ctx *trajectory.TrajectoryContext) (bool, trajectory.TriggerType) {
	// Check if progressive context is enabled.
	if a.aceConfig == nil || !a.aceConfig.Retrieval.ProgressiveContext.Enabled {
		return false, ""
	}

	// Trigger 1: Initial (Turn 0) - always retrieve on first turn.
	if ctx.CurrentTurn == 0 {
		return true, trajectory.TriggerInitial
	}

	cfg := a.aceConfig.Retrieval.ProgressiveContext

	// Trigger 2: Error - recent error detected.
	if ctx.HasRecentError(cfg.ErrorLookback) {
		return true, trajectory.TriggerError
	}

	// Trigger 3: Tool Change - different tools used recently.
	tools := ctx.GetRecentTools(cfg.ToolChangeLookback)
	if len(tools) > 1 {
		return true, trajectory.TriggerToolChange
	}

	// Trigger 4: Interval - cache TTL expired.
	if ctx.CurrentTurn-ctx.LastRetrievalTurn >= cfg.CacheTTL {
		return true, trajectory.TriggerInterval
	}

	// No triggers activated.
	return false, ""
}
