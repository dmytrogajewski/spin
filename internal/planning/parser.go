// Package planning provides planning services and parsing.
package planning

import (
	"fmt"
	"strings"
)

const maxConsecutiveNonPlanLines = 2

// planScanner tracks state while scanning text for plan sections.
type planScanner struct {
	inPlanSection           bool
	consecutiveNonPlanLines int
	steps                   []Step
}

// DetectPlanFromText detects plan-like structures in text output.
// This is a conservative implementation that ONLY detects plans after explicit headers.
func DetectPlanFromText(output string) *Plan {
	if output == "" {
		return nil
	}

	scanner := &planScanner{}

	for _, line := range strings.Split(output, "\n") {
		scanner.processLine(strings.TrimSpace(line))
	}

	if len(scanner.steps) == 0 {
		return nil
	}

	plan := NewPlan()
	plan.Steps = scanner.steps

	return plan
}

// processLine handles a single line during plan scanning.
func (ps *planScanner) processLine(line string) {
	if line == "" {
		ps.handleEmptyLine()

		return
	}

	if isPlanHeader(line) {
		ps.inPlanSection = true
		ps.consecutiveNonPlanLines = 0

		return
	}

	if !ps.inPlanSection {
		return
	}

	if !matchesPlanPattern(line) {
		ps.handleNonPlanLine()

		return
	}

	ps.consecutiveNonPlanLines = 0
	ps.addStep(line)
}

// handleEmptyLine processes an empty line within or outside a plan section.
func (ps *planScanner) handleEmptyLine() {
	if !ps.inPlanSection {
		return
	}

	ps.consecutiveNonPlanLines++
	if ps.consecutiveNonPlanLines >= maxConsecutiveNonPlanLines {
		ps.inPlanSection = false
		ps.consecutiveNonPlanLines = 0
	}
}

// handleNonPlanLine processes a non-plan-pattern line within a plan section.
func (ps *planScanner) handleNonPlanLine() {
	ps.consecutiveNonPlanLines++
	if ps.consecutiveNonPlanLines >= maxConsecutiveNonPlanLines {
		ps.inPlanSection = false
		ps.consecutiveNonPlanLines = 0
	}
}

// addStep extracts and appends a plan step from the given line.
func (ps *planScanner) addStep(line string) {
	step := extractPlanStep(line, len(ps.steps)+1)
	if step == nil {
		return
	}

	if len(ps.steps) > 0 {
		prevStep := ps.steps[len(ps.steps)-1]
		step.DependsOn = []string{prevStep.ID}
	}

	ps.steps = append(ps.steps, *step)
}

// isPlanHeader checks if a line is an explicit plan section header.
func isPlanHeader(line string) bool {
	lowerLine := strings.ToLower(line)

	headerPrefixes := []string{
		"plan:", "steps:", "task:", "tasks:", "## plan", "## steps",
	}

	for _, prefix := range headerPrefixes {
		if strings.HasPrefix(lowerLine, prefix) {
			return true
		}
	}

	// Also accept lines ending with ":" containing plan/step/task keywords.
	if strings.HasSuffix(lowerLine, ":") {
		return strings.Contains(lowerLine, "plan") ||
			strings.Contains(lowerLine, "step") ||
			strings.Contains(lowerLine, "task")
	}

	return false
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
