package reflector

import (
	"fmt"
	"strings"

	"github.com/dmytrogajewski/spin/internal/ace/generator"
	"github.com/dmytrogajewski/spin/internal/ace/trajectory"
)

// PromptBuilder constructs reflection prompts for trajectories.
type PromptBuilder struct {
	// Future: configuration options can go here.
}

// NewPromptBuilder creates a new prompt builder.
func NewPromptBuilder() *PromptBuilder {
	return &PromptBuilder{}
}

// BuildSingleTrajectory creates a reflection prompt for one trajectory.
// Aligned with ACE paper Figure 12 (Reflector prompt).
func (pb *PromptBuilder) BuildSingleTrajectory(traj *generator.Trajectory) string {
	var sb strings.Builder

	sb.WriteString("You are an expert analyst and educator. Your job is to diagnose the execution trajectory ")
	sb.WriteString("and extract actionable insights for improving future coding tasks.\n\n")

	sb.WriteString("**Instructions:**\n")
	sb.WriteString("- Carefully analyze the execution trace to identify what worked well and what went wrong\n")
	sb.WriteString("- If execution failed, identify specific errors, root causes, or misapplied strategies\n")
	sb.WriteString("- If execution succeeded, identify patterns and strategies that led to success\n")
	sb.WriteString("- Provide actionable insights that could help avoid mistakes or replicate success in the future\n")
	sb.WriteString("- Focus on the root cause, not just surface-level errors\n")
	sb.WriteString("- Be specific about what should be done differently or what pattern should be remembered\n\n")

	sb.WriteString("**Trajectory:**\n")
	sb.WriteString(fmt.Sprintf("Task: %s\n", traj.Query))
	sb.WriteString(fmt.Sprintf("Success: %t\n\n", traj.Success))

	// Include detailed execution steps if available.
	if len(traj.Steps) > 0 {
		sb.WriteString("**Execution Steps:**\n")

		for _, step := range traj.Steps {
			sb.WriteString(fmt.Sprintf("%d. [%s] %s\n", step.StepNumber+1, step.Type, step.Content))
		}

		sb.WriteString("\n")
	}

	// Include retrieval events if available (progressive context).
	if traj.Metadata.RetrievalEvents != nil {
		if events, ok := traj.Metadata.RetrievalEvents.([]trajectory.RetrievalEvent); ok && len(events) > 0 {
			sb.WriteString("**Retrieval Events:**\n")
			sb.WriteString("(Shows when and why bullets were retrieved during execution)\n")

			for _, event := range events {
				sb.WriteString(fmt.Sprintf("Turn %d [%s]: Query=\"%s\" → Retrieved %d bullets\n",
					event.Turn, event.Trigger, event.Query, len(event.BulletsAdded)))
			}

			sb.WriteString("\n")
		}
	}

	// Include retrieved bullets if available.
	if len(traj.RetrievedBullets) > 0 {
		sb.WriteString("**Retrieved Playbook Bullets:**\n")

		for _, bullet := range traj.RetrievedBullets {
			sb.WriteString(fmt.Sprintf("- [%s] %s\n", bullet.ID, bullet.Content))
		}

		sb.WriteString("\n")
	}

	sb.WriteString(fmt.Sprintf("**Final Output:**\n%s\n\n", traj.Output))

	sb.WriteString("**Your output should be a JSON object with the following fields:**\n")
	sb.WriteString("- reasoning: Your chain of thought, detailed analysis of what happened\n")
	sb.WriteString("- error_identification: What specifically went wrong (or \"N/A\" if successful)\n")
	sb.WriteString("- root_cause_analysis: Why did this error occur? What was misunderstood? (or success factors)\n")
	sb.WriteString("- correct_approach: What should be done instead? (or what was done right)\n")
	sb.WriteString("- key_insight: What strategy, pattern, or principle should be remembered?\n")
	sb.WriteString("- category: success_pattern | error_mode | optimization | anti_pattern\n")
	sb.WriteString("- confidence: Your confidence in this analysis (0.0 to 1.0)\n\n")

	sb.WriteString("**Answer in this exact JSON format:**\n")
	sb.WriteString("{\n")
	sb.WriteString("  \"reasoning\": \"[Your detailed analysis and thinking process]\",\n")
	sb.WriteString("  \"error_identification\": \"[What went wrong, or N/A if successful]\",\n")
	sb.WriteString("  \"root_cause_analysis\": \"[Why this happened]\",\n")
	sb.WriteString("  \"correct_approach\": \"[What should be done instead]\",\n")
	sb.WriteString("  \"key_insight\": \"[Strategy or principle to remember]\",\n")
	sb.WriteString("  \"category\": \"success_pattern\",\n")
	sb.WriteString("  \"confidence\": 0.95\n")
	sb.WriteString("}\n")

	return sb.String()
}

// BuildWithGroundTruth creates a reflection prompt with ground truth comparison.
// Aligned with ACE paper Figure 12 - includes bullet tagging for itemized learning.
func (pb *PromptBuilder) BuildWithGroundTruth(traj *generator.Trajectory, groundTruth string, usedBullets []string) string {
	var sb strings.Builder

	sb.WriteString("You are an expert analyst and educator. Your job is to diagnose why the execution went wrong ")
	sb.WriteString("by analyzing the gap between the actual outcome and the expected outcome.\n\n")

	sb.WriteString("**Instructions:**\n")
	sb.WriteString("- Carefully analyze the execution trace to identify where it went wrong\n")
	sb.WriteString("- Compare the actual outcome with the expected outcome to understand the gap\n")
	sb.WriteString("- Identify specific conceptual errors, execution mistakes, or misapplied strategies\n")
	sb.WriteString("- Provide actionable insights that could help avoid this mistake in the future\n")
	sb.WriteString("- Focus on the root cause, not just surface-level errors\n")
	sb.WriteString("- Be specific about what should have been done differently\n")
	sb.WriteString("- Analyze which playbook bullets were helpful, harmful, or neutral\n\n")

	sb.WriteString("**Task:**\n")
	sb.WriteString(fmt.Sprintf("%s\n\n", traj.Query))

	sb.WriteString("**Execution Trace:**\n")
	sb.WriteString(fmt.Sprintf("%s\n\n", traj.Output))

	sb.WriteString("**Actual Outcome:**\n")
	sb.WriteString(fmt.Sprintf("Success: %t\n\n", traj.Success))

	if groundTruth != "" {
		sb.WriteString("**Expected Outcome:**\n")
		sb.WriteString(fmt.Sprintf("%s\n\n", groundTruth))
	}

	if len(usedBullets) > 0 {
		sb.WriteString("**Playbook Bullets Used:**\n")

		for _, bulletID := range usedBullets {
			sb.WriteString(fmt.Sprintf("- %s\n", bulletID))
		}

		sb.WriteString("\n")
	}

	sb.WriteString("**Your output should be a JSON object with the following fields:**\n")
	sb.WriteString("- reasoning: Your chain of thought, detailed analysis and calculations\n")
	sb.WriteString("- error_identification: What specifically went wrong in the execution?\n")
	sb.WriteString("- root_cause_analysis: Why did this error occur? What was misunderstood?\n")
	sb.WriteString("- correct_approach: What should have been done instead?\n")
	sb.WriteString("- key_insight: What strategy, formula, or principle should be remembered to avoid this error?\n")

	if len(usedBullets) > 0 {
		sb.WriteString("- bullet_tags: A list of JSON objects with bullet_id and tag for each bullet used\n")
		sb.WriteString("  - tag can be: 'helpful' (aided correct solution), 'harmful' (led to error), 'neutral' (no impact)\n")
	}

	sb.WriteString("\n**Answer in this exact JSON format:**\n")
	sb.WriteString("{\n")
	sb.WriteString("  \"reasoning\": \"[Your chain of thought / reasoning / thinking process, detailed analysis and calculations]\",\n")
	sb.WriteString("  \"error_identification\": \"[What specifically went wrong in the reasoning?]\",\n")
	sb.WriteString("  \"root_cause_analysis\": \"[Why did this error occur? What concept was misunderstood?]\",\n")
	sb.WriteString("  \"correct_approach\": \"[What should the model have done instead?]\",\n")
	sb.WriteString("  \"key_insight\": \"[What strategy, formula, or principle should be remembered to avoid this error?]\"")

	if len(usedBullets) > 0 {
		sb.WriteString(",\n  \"bullet_tags\": [\n")
		sb.WriteString("    {\"id\": \"B001\", \"tag\": \"helpful\"},\n")
		sb.WriteString("    {\"id\": \"B002\", \"tag\": \"harmful\"}\n")
		sb.WriteString("  ]")
	}

	sb.WriteString("\n}\n")

	return sb.String()
}

// BuildRefinementPrompt creates a prompt for refining existing insights.
func (pb *PromptBuilder) BuildRefinementPrompt(insights []*Insight) string {
	var sb strings.Builder

	sb.WriteString("You are refining actionable coding insights to make them more specific and actionable.\n\n")
	sb.WriteString("# Current Insights\n")

	for i, insight := range insights {
		sb.WriteString(fmt.Sprintf("%d. %s\n", i+1, insight.Content))
		sb.WriteString(fmt.Sprintf("   Category: %s\n", insight.Category))
		sb.WriteString(fmt.Sprintf("   Confidence: %.2f\n", insight.Confidence))

		if len(insight.Evidence) > 0 {
			sb.WriteString(fmt.Sprintf("   Evidence: %s\n", strings.Join(insight.Evidence, "; ")))
		}

		sb.WriteString("\n")
	}

	sb.WriteString("# Task\n")
	sb.WriteString("Refine each insight to make it:\n")
	sb.WriteString("1. More specific and concrete\n")
	sb.WriteString("2. More actionable (clear when/how to apply)\n")
	sb.WriteString("3. More confident if evidence supports it\n\n")
	sb.WriteString("For each refined insight, provide:\n")
	sb.WriteString("1. content: The improved insight (50-500 chars)\n")
	sb.WriteString("2. evidence: Array of supporting evidence quotes (e.g., [\"quote1\", \"quote2\"])\n")
	sb.WriteString("3. confidence: Updated confidence (0.0 to 1.0)\n")
	sb.WriteString("4. category: success_pattern | error_mode | optimization | anti_pattern\n\n")
	sb.WriteString("Return the same number of insights in the same order as JSON array.\n")

	return sb.String()
}

// BuildBatchTrajectory creates a prompt for analyzing multiple trajectories to find patterns.
func (pb *PromptBuilder) BuildBatchTrajectory(trajs []*generator.Trajectory) string {
	var sb strings.Builder

	sb.WriteString("You are analyzing multiple execution trajectories to extract patterns and insights.\n\n")
	sb.WriteString("# Trajectories\n")

	for i, traj := range trajs {
		sb.WriteString(fmt.Sprintf("## Trajectory %d (ID: %s)\n", i+1, traj.ID))
		sb.WriteString(fmt.Sprintf("Query: %s\n", traj.Query))
		sb.WriteString(fmt.Sprintf("Success: %t\n", traj.Success))
		sb.WriteString(fmt.Sprintf("Output: %s\n\n", traj.Output))
	}

	sb.WriteString("# Task\n")
	sb.WriteString("Analyze these trajectories to extract actionable insights.\n\n")
	sb.WriteString("Look for:\n")
	sb.WriteString("1. Common patterns across successful trajectories\n")
	sb.WriteString("2. Recurring errors or failure modes\n")
	sb.WriteString("3. Best practices that appear multiple times\n")
	sb.WriteString("4. Anti-patterns that lead to failures\n\n")
	sb.WriteString("For each insight, provide:\n")
	sb.WriteString("1. content: A specific lesson learned (50-500 chars)\n")
	sb.WriteString("2. evidence: Array of quotes from trajectories supporting this (e.g., [\"quote1\", \"quote2\"])\n")
	sb.WriteString("3. confidence: Your confidence in this insight (0.0 to 1.0)\n")
	sb.WriteString("4. category: success_pattern | error_mode | optimization | anti_pattern\n\n")
	sb.WriteString("Prioritize insights that appear across multiple trajectories.\n")
	sb.WriteString("Format as JSON array.\n")

	return sb.String()
}
