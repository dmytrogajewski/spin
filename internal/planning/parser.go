// Package planning provides planning services and parsing.
package planning

import (
	"fmt"
	"strings"
)

// DetectPlanFromText detects plan-like structures in text output.
// This is a conservative implementation that ONLY detects plans after explicit headers.
func DetectPlanFromText(output string) *Plan {
	if output == "" {
		return nil
	}

	var steps []Step

	lines := strings.Split(output, "\n")

	// Only enter plan section after explicit headers.
	var (
		inPlanSection           bool
		consecutiveNonPlanLines int
	)

	for _, line := range lines {
		line = strings.TrimSpace(line)

		if line == "" {
			// Empty lines within plan section are OK, but count toward exit condition.
			if inPlanSection {
				consecutiveNonPlanLines++
				if consecutiveNonPlanLines >= 2 {
					inPlanSection = false
					consecutiveNonPlanLines = 0
				}
			}

			continue
		}

		// Check if this line starts an EXPLICIT plan section.
		lowerLine := strings.ToLower(line)
		isHeader := strings.HasPrefix(lowerLine, "plan:") || strings.HasPrefix(lowerLine, "steps:") ||
			strings.HasPrefix(lowerLine, "task:") || strings.HasPrefix(lowerLine, "tasks:") ||
			strings.HasPrefix(lowerLine, "## plan") || strings.HasPrefix(lowerLine, "## steps")

		// Also accept lines ending with "plan:" or "steps:" (e.g. "Here's the plan:").
		if !isHeader && strings.HasSuffix(lowerLine, ":") &&
			(strings.Contains(lowerLine, "plan") || strings.Contains(lowerLine, "step") || strings.Contains(lowerLine, "task")) {
			isHeader = true
		}

		if isHeader {
			inPlanSection = true
			consecutiveNonPlanLines = 0

			continue
		}

		// Only process lines if we're in an explicitly declared plan section.
		if !inPlanSection {
			continue
		}

		// Check if current line matches plan pattern (numbered list or bullet).
		isPlanPattern := matchesPlanPattern(line)

		// If in plan section but no plan pattern, check if we should exit.
		if !isPlanPattern {
			consecutiveNonPlanLines++
			// Exit plan section after 2 consecutive non-plan lines.
			if consecutiveNonPlanLines >= 2 {
				inPlanSection = false
				consecutiveNonPlanLines = 0
			}

			continue
		}

		// Reset counter - we saw a plan pattern.
		consecutiveNonPlanLines = 0

		// Extract plan step from line.
		step := extractPlanStep(line, len(steps)+1)
		if step != nil {
			// Chain steps sequentially: each step depends on the previous one.
			if len(steps) > 0 {
				prevStep := steps[len(steps)-1]
				step.DependsOn = []string{prevStep.ID}
			}

			steps = append(steps, *step)
		}
	}

	if len(steps) == 0 {
		return nil
	}

	plan := NewPlan()
	plan.Steps = steps

	return plan
}

// matchesPlanPattern checks if a line matches common plan patterns.
func matchesPlanPattern(line string) bool {
	line = strings.TrimSpace(line)
	if line == "" {
		return false
	}
	// Check for numbered list (1., 2., 3., etc.)
	if len(line) >= 2 && line[0] >= '1' && line[0] <= '9' && (line[1] == '.' || line[1] == ')') {
		return true
	}
	// Check for bullet points.
	if strings.HasPrefix(line, "- ") || strings.HasPrefix(line, "* ") {
		return true
	}

	return false
}

// extractPlanStep extracts a plan step from a line of text.
func extractPlanStep(line string, index int) *Step {
	// Remove common list prefixes.
	content := line
	content = strings.TrimPrefix(content, "- ")
	content = strings.TrimPrefix(content, "* ")
	// Remove numbered prefixes (1., 2., etc.)
	for i := 1; i <= 99; i++ {
		numberedPrefix := fmt.Sprintf("%d.", i)
		if after, ok := strings.CutPrefix(content, numberedPrefix); ok {
			content = after

			break
		}

		parenPrefix := fmt.Sprintf("%d)", i)
		if after, ok := strings.CutPrefix(content, parenPrefix); ok {
			content = after

			break
		}
	}

	content = strings.TrimSpace(content)

	if content == "" {
		return nil
	}

	step := &Step{
		ID:          fmt.Sprintf("step-%d", index),
		Description: content,
		Status:      StepStatusPending,
	}

	return step
}
