package agent

import (
	"strings"
	"sync"
	"time"

	"github.com/dmytrogajewski/spin/internal/events"
	"github.com/dmytrogajewski/spin/internal/planning"
)

// PlanTracker monitors tool execution and updates plan step statuses.
type PlanTracker struct {
	plan    *planning.Plan
	emitter *events.EventEmitter
	mu      sync.RWMutex

	// Track which steps are currently running.
	runningSteps map[string]bool
}

// NewPlanTracker creates a new plan tracker.
func NewPlanTracker(plan *planning.Plan, emitter *events.EventEmitter) *PlanTracker {
	return &PlanTracker{
		plan:         plan,
		emitter:      emitter,
		runningSteps: make(map[string]bool),
	}
}

// OnToolCallComplete handles tool call completion events.
func (t *PlanTracker) OnToolCallComplete(event events.Event) {
	data, ok := event.ToolCallCompleteData()
	if !ok {
		return
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	// Find matching step using fuzzy description match.
	for i := range t.plan.Steps {
		step := &t.plan.Steps[i]

		if !t.matchesStep(step, data.ToolName) {
			continue
		}

		t.transitionStepOnToolComplete(step, data.Success)

		break // Only match first step.
	}
}

// transitionStepOnToolComplete updates step status based on tool completion.
func (t *PlanTracker) transitionStepOnToolComplete(step *planning.Step, success bool) {
	if step.Status == planning.StepStatusPending {
		step.Status = planning.StepStatusRunning
		now := time.Now()
		step.StartedAt = &now
		t.runningSteps[step.ID] = true
		t.emitPlanUpdate()
	}

	if !t.runningSteps[step.ID] {
		return
	}

	if success {
		step.Status = planning.StepStatusCompleted
		now := time.Now()
		step.CompletedAt = &now
	} else {
		step.Status = planning.StepStatusFailed
	}

	delete(t.runningSteps, step.ID)
	t.emitPlanUpdate()
}

// matchesStep performs fuzzy matching between step and tool name.
// Returns true if the tool name appears in step description/action.
func (t *PlanTracker) matchesStep(step *planning.Step, toolName string) bool {
	lowerTool := strings.ToLower(toolName)
	lowerDesc := strings.ToLower(step.Description)
	lowerAction := strings.ToLower(step.Action)

	// Direct tool name mention in description or action.
	if strings.Contains(lowerDesc, lowerTool) || strings.Contains(lowerAction, lowerTool) {
		return true
	}

	// Semantic matching for common patterns.
	toolPatterns := map[string][]string{
		"read_file":       {"read", "open", "load", "get"},
		"write_file":      {"write", "save", "create", "update"},
		"list_directory":  {"list", "find", "search", "explore"},
		"execute_command": {"run", "execute", "command"},
		"shell_command":   {"run", "execute", "command", "echo"},
		"terminal":        {"run", "execute", "command", "echo"},
		"apply_patch":     {"apply", "patch", "modify", "change"},
	}

	if patterns, exists := toolPatterns[lowerTool]; exists {
		for _, pattern := range patterns {
			if strings.Contains(lowerDesc, pattern) || strings.Contains(lowerAction, pattern) {
				return true
			}
		}
	}

	return false
}

// emitPlanUpdate emits an EventPlanUpdate with the current plan state.
func (t *PlanTracker) emitPlanUpdate() {
	t.emitter.Emit(events.Event{
		Type:      events.EventPlanUpdate,
		Timestamp: time.Now(),
		Data: events.PlanUpdateData{
			Plan: t.plan,
		},
	})
}

// UpdatePlanStatus updates the overall plan status based on step statuses.
func (t *PlanTracker) UpdatePlanStatus() {
	t.mu.Lock()
	defer t.mu.Unlock()

	allCompleted := true
	anyFailed := false
	anyRunning := false

	for _, step := range t.plan.Steps {
		switch step.Status {
		case planning.StepStatusRunning:
			anyRunning = true
			allCompleted = false
		case planning.StepStatusPending:
			allCompleted = false
		case planning.StepStatusFailed:
			anyFailed = true
			allCompleted = false
		}
	}

	oldStatus := t.plan.Status
	if anyFailed {
		t.plan.Status = planning.PlanStatusFailed
	} else if allCompleted {
		t.plan.Status = planning.PlanStatusCompleted
	} else if anyRunning {
		t.plan.Status = planning.PlanStatusInProgress
	}

	// Emit update if status changed.
	if oldStatus != t.plan.Status {
		t.emitPlanUpdate()
	}
}
