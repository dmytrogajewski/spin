package agent

import (
	"testing"

	"github.com/dmytrogajewski/spin/internal/ace/generator"
	"github.com/dmytrogajewski/spin/internal/ace/trajectory"
)

func TestProgressiveContextConfig_Defaults(t *testing.T) {
	cfg := DefaultProgressiveContextConfig()

	if !cfg.Enabled {
		t.Error("expected Enabled=true by default (enabled by default per user requirement)")
	}

	if cfg.CacheTTL != 10 {
		t.Errorf("expected CacheTTL=10, got %d", cfg.CacheTTL)
	}

	if cfg.ErrorLookback != 5 {
		t.Errorf("expected ErrorLookback=5, got %d", cfg.ErrorLookback)
	}

	if cfg.ToolChangeLookback != 3 {
		t.Errorf("expected ToolChangeLookback=3, got %d", cfg.ToolChangeLookback)
	}

	if len(cfg.EnabledTriggers) != 4 {
		t.Errorf("expected 4 default triggers, got %d", len(cfg.EnabledTriggers))
	}
}

func TestShouldRetrieveProgressive_Disabled(t *testing.T) {
	agent := &Agent{
		aceConfig: &ACEConfig{
			Retrieval: ACERetrievalConfig{
				ProgressiveContext: ProgressiveContextConfig{
					Enabled: false, // Disabled.
				},
			},
		},
	}

	ctx := trajectory.NewTrajectoryContext("test query")
	ctx.CurrentTurn = 5

	shouldRetrieve, trigger := agent.shouldRetrieveProgressive(ctx)

	if shouldRetrieve {
		t.Error("expected false when progressive context disabled")
	}

	if trigger != "" {
		t.Errorf("expected empty trigger, got %q", trigger)
	}
}

func TestShouldRetrieveProgressive_TriggerPriority(t *testing.T) {
	agent := &Agent{
		aceConfig: &ACEConfig{
			Retrieval: ACERetrievalConfig{
				ProgressiveContext: ProgressiveContextConfig{
					Enabled:       true,
					CacheTTL:      10,
					ErrorLookback: 5,
				},
			},
		},
	}

	// Test 1: Turn 0 with error - should return TriggerInitial (higher priority).
	ctx := trajectory.NewTrajectoryContext("test query")
	ctx.CurrentTurn = 0
	ctx.AppendSteps([]generator.TrajectoryStep{
		{StepNumber: 0, Content: "error occurred"},
	})

	shouldRetrieve, trigger := agent.shouldRetrieveProgressive(ctx)

	if !shouldRetrieve {
		t.Error("expected true when turn 0")
	}

	if trigger != trajectory.TriggerInitial {
		t.Errorf("expected TriggerInitial (highest priority), got %q", trigger)
	}

	// Test 2: Error + TTL expired - should return TriggerError (higher than interval).
	ctx2 := trajectory.NewTrajectoryContext("test query")
	ctx2.CurrentTurn = 20
	ctx2.LastRetrievalTurn = 0 // 20 turns ago, TTL expired.
	ctx2.AppendSteps([]generator.TrajectoryStep{
		{StepNumber: 19, Content: "error occurred"},
	})

	shouldRetrieve2, trigger2 := agent.shouldRetrieveProgressive(ctx2)

	if !shouldRetrieve2 {
		t.Error("expected true when error")
	}

	if trigger2 != trajectory.TriggerError {
		t.Errorf("expected TriggerError (higher than interval), got %q", trigger2)
	}
}

func TestShouldRetrieveProgressive_TurnZero(t *testing.T) {
	agent := &Agent{
		aceConfig: &ACEConfig{
			Retrieval: ACERetrievalConfig{
				ProgressiveContext: ProgressiveContextConfig{
					Enabled: true,
				},
			},
		},
	}

	ctx := trajectory.NewTrajectoryContext("test query")
	ctx.CurrentTurn = 0

	shouldRetrieve, trigger := agent.shouldRetrieveProgressive(ctx)

	if !shouldRetrieve {
		t.Error("expected true on turn 0")
	}

	if trigger != trajectory.TriggerInitial {
		t.Errorf("expected TriggerInitial, got %q", trigger)
	}
}

func TestShouldRetrieveProgressive_RecentError(t *testing.T) {
	agent := &Agent{
		aceConfig: &ACEConfig{
			Retrieval: ACERetrievalConfig{
				ProgressiveContext: ProgressiveContextConfig{
					Enabled:       true,
					ErrorLookback: 5,
				},
			},
		},
	}

	ctx := trajectory.NewTrajectoryContext("test query")
	ctx.CurrentTurn = 10
	ctx.LastRetrievalTurn = 5

	// Add steps with error.
	ctx.AppendSteps([]generator.TrajectoryStep{
		{StepNumber: 8, Content: "running command"},
		{StepNumber: 9, Content: "error: command failed"},
	})

	shouldRetrieve, trigger := agent.shouldRetrieveProgressive(ctx)

	if !shouldRetrieve {
		t.Error("expected true when recent error detected")
	}

	if trigger != trajectory.TriggerError {
		t.Errorf("expected TriggerError, got %q", trigger)
	}
}

func TestShouldRetrieveProgressive_ToolChange(t *testing.T) {
	agent := &Agent{
		aceConfig: &ACEConfig{
			Retrieval: ACERetrievalConfig{
				ProgressiveContext: ProgressiveContextConfig{
					Enabled:            true,
					ToolChangeLookback: 3,
				},
			},
		},
	}

	ctx := trajectory.NewTrajectoryContext("test query")
	ctx.CurrentTurn = 10
	ctx.LastRetrievalTurn = 5

	// Add steps with tool change.
	ctx.AppendSteps([]generator.TrajectoryStep{
		{StepNumber: 8, Content: "Tool: bash"},
		{StepNumber: 9, Content: "Tool: grep"}, // Different tool.
	})

	shouldRetrieve, trigger := agent.shouldRetrieveProgressive(ctx)

	if !shouldRetrieve {
		t.Error("expected true when tool change detected")
	}

	if trigger != trajectory.TriggerToolChange {
		t.Errorf("expected TriggerToolChange, got %q", trigger)
	}
}

func TestShouldRetrieveProgressive_Interval(t *testing.T) {
	agent := &Agent{
		aceConfig: &ACEConfig{
			Retrieval: ACERetrievalConfig{
				ProgressiveContext: ProgressiveContextConfig{
					Enabled:  true,
					CacheTTL: 10,
				},
			},
		},
	}

	ctx := trajectory.NewTrajectoryContext("test query")
	ctx.CurrentTurn = 15
	ctx.LastRetrievalTurn = 0 // 15 turns ago, exceeds TTL.

	shouldRetrieve, trigger := agent.shouldRetrieveProgressive(ctx)

	if !shouldRetrieve {
		t.Error("expected true when cache TTL expired")
	}

	if trigger != trajectory.TriggerInterval {
		t.Errorf("expected TriggerInterval, got %q", trigger)
	}
}

func TestShouldRetrieveProgressive_NoTrigger(t *testing.T) {
	agent := &Agent{
		aceConfig: &ACEConfig{
			Retrieval: ACERetrievalConfig{
				ProgressiveContext: ProgressiveContextConfig{
					Enabled:  true,
					CacheTTL: 10,
				},
			},
		},
	}

	ctx := trajectory.NewTrajectoryContext("test query")
	ctx.CurrentTurn = 8
	ctx.LastRetrievalTurn = 5 // 3 turns ago, within TTL.
	// No errors, no tool changes.

	shouldRetrieve, trigger := agent.shouldRetrieveProgressive(ctx)

	if shouldRetrieve {
		t.Error("expected false when no triggers activated")
	}

	if trigger != "" {
		t.Errorf("expected empty trigger, got %q", trigger)
	}
}
