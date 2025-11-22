package acp

import (
	"fmt"
	"strings"

	"github.com/coder/acp-go-sdk"
	"github.com/dmytrogajewski/spin/internal/planning"
)

// convertOrchestrationPlanToACP converts a planning.Plan to ACP PlanEntry[].
func convertOrchestrationPlanToACP(plan *planning.Plan) []acp.PlanEntry {
	if plan == nil || len(plan.Steps) == 0 {
		return nil
	}

	entries := make([]acp.PlanEntry, 0, len(plan.Steps))
	for _, step := range plan.Steps {
		entry := acp.PlanEntry{
			Content:  buildStepContent(step),
			Priority: mapStepPriority(step, plan),
			Status:   mapStepStatus(step.Status),
		}
		entries = append(entries, entry)
	}

	return entries
}

// buildStepContent builds the content string for a plan entry from a step.
func buildStepContent(step planning.Step) string {
	if step.Action != "" {
		return fmt.Sprintf("%s: %s", step.Description, step.Action)
	}
	return step.Description
}

// mapStepStatus maps planning.StepStatus to acp.PlanEntryStatus.
func mapStepStatus(status planning.StepStatus) acp.PlanEntryStatus {
	switch status {
	case planning.StepStatusPending:
		return acp.PlanEntryStatusPending
	case planning.StepStatusReady:
		return acp.PlanEntryStatusPending // Ready steps are still pending from ACP perspective
	case planning.StepStatusRunning:
		return acp.PlanEntryStatus("in_progress") // ACP uses "in_progress" for running steps
	case planning.StepStatusCompleted:
		return acp.PlanEntryStatus("completed")
	case planning.StepStatusFailed:
		return acp.PlanEntryStatus("failed")
	case planning.StepStatusSkipped:
		return acp.PlanEntryStatus("cancelled")
	default:
		return acp.PlanEntryStatusPending
	}
}

// mapStepPriority maps step priority based on dependencies.
// Steps with no dependencies are high priority (can start immediately).
// Steps with dependencies are medium priority.
func mapStepPriority(step planning.Step, plan *planning.Plan) acp.PlanEntryPriority {
	lowerDesc := strings.ToLower(step.Description)

	// Heuristics: check for explicit priority keywords
	if strings.Contains(lowerDesc, "critical") || strings.Contains(lowerDesc, "urgent") ||
		strings.Contains(lowerDesc, "important") || strings.Contains(lowerDesc, "priority") {
		return acp.PlanEntryPriorityHigh
	}
	if strings.Contains(lowerDesc, "optional") || strings.Contains(lowerDesc, "nice to have") {
		return acp.PlanEntryPriorityLow
	}

	// Steps with no dependencies are high priority (critical path start)
	if len(step.DependsOn) == 0 {
		return acp.PlanEntryPriorityHigh
	}

	// Steps with dependencies are medium priority
	return acp.PlanEntryPriorityMedium
}

