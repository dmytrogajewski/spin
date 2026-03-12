package curator

import (
	"fmt"
	"strings"

	"github.com/dmytrogajewski/spin/internal/ace/reflector"
)

// PromptBuilder constructs curator prompts for integrating insights into playbook.
// Aligned with ACE paper Figures 10, 11, 13, 14 (Curator prompts).
type PromptBuilder struct {
	// Future: configuration options can go here.
}

// NewPromptBuilder creates a new curator prompt builder.
func NewPromptBuilder() *PromptBuilder {
	return &PromptBuilder{}
}

// BuildCurationPrompt creates a prompt for curating insights into playbook deltas.
// This is the core Curator prompt from the ACE paper.
func (pb *PromptBuilder) BuildCurationPrompt(req CurationRequest) string {
	var sb strings.Builder

	sb.WriteString("You are a master curator of knowledge. Your job is to identify what new insights should be ")
	sb.WriteString("added to an existing playbook based on a reflection from a previous execution.\n\n")

	sb.WriteString("**Context:**\n")
	sb.WriteString("- The playbook you create helps solve similar coding tasks\n")
	sb.WriteString("- The reflection is generated from execution traces that are NOT available when the playbook is used\n")
	sb.WriteString("- You need to come up with content that can aid future executions to achieve better outcomes\n\n")

	sb.WriteString("**CRITICAL: You MUST respond with valid JSON only. Do not use markdown formatting or code blocks.**\n\n")

	sb.WriteString("**Instructions:**\n")
	sb.WriteString("- Review the existing playbook and the reflection from the previous execution\n")
	sb.WriteString("- Identify ONLY the NEW insights, strategies, or mistakes that are MISSING from the current playbook\n")
	sb.WriteString("- Avoid redundancy - if similar advice already exists, only add new content that is a perfect complement\n")
	sb.WriteString("- Do NOT regenerate the entire playbook - only provide the additions needed\n")
	sb.WriteString("- Focus on quality over quantity - a focused, well-organized playbook is better than an exhaustive one\n")
	sb.WriteString("- Format your response as a PURE JSON object with specific sections\n")
	sb.WriteString("- If no new content to add, return an empty list for the operations field\n")
	sb.WriteString("- Be concise and specific - each addition should be actionable\n")
	sb.WriteString("- For coding tasks, explicitly curate API schemas, error patterns, and best practices\n\n")

	sb.WriteString("**Task Context (the actual task instruction):**\n")
	sb.WriteString(fmt.Sprintf("%s\n\n", req.TaskContext))

	sb.WriteString("**Current Playbook:**\n")

	if req.CurrentPlaybook != "" {
		sb.WriteString(fmt.Sprintf("%s\n\n", req.CurrentPlaybook))
	} else {
		sb.WriteString("[Empty playbook - this is the first entry]\n\n")
	}

	sb.WriteString("**Current Reflection (insights from the execution):**\n")
	sb.WriteString(fmt.Sprintf("%s\n\n", req.Reflection))

	// Add examples.
	sb.WriteString("**Examples:**\n\n")
	sb.WriteString("**Example 1:**\n")
	sb.WriteString("Task Context: \"Fix null pointer exception in user authentication\"\n\n")
	sb.WriteString("Current Playbook: [Basic error handling guidelines]\n\n")
	sb.WriteString("Reflection: \"The agent failed because it didn't check if the user object was nil before accessing ")
	sb.WriteString("its properties, leading to a null pointer exception. This is a common pattern in authentication flows.\"\n\n")
	sb.WriteString("Response:\n")
	sb.WriteString("```\n")
	sb.WriteString("{\n")
	sb.WriteString("  \"reasoning\": \"The reflection shows a critical error pattern where nil checks were skipped before ")
	sb.WriteString("property access. This is a fundamental defensive programming principle that should be captured in the ")
	sb.WriteString("playbook to prevent similar failures in authentication and user data handling.\",\n")
	sb.WriteString("  \"operations\": [\n")
	sb.WriteString("    {\n")
	sb.WriteString("      \"type\": \"ADD\",\n")
	sb.WriteString("      \"section\": \"error_handling\",\n")
	sb.WriteString("      \"content\": \"Always check if objects are nil before accessing properties\\n- In authentication flows, ")
	sb.WriteString("verify user object is not nil before accessing user.Email, user.ID, etc.\\n- Use early returns with error ")
	sb.WriteString("checks to avoid nested nil checks\\n- Pattern: if user == nil { return ErrUserNotFound }\"\n")
	sb.WriteString("    }\n")
	sb.WriteString("  ]\n")
	sb.WriteString("}\n")
	sb.WriteString("```\n\n")

	sb.WriteString("**Example 2:**\n")
	sb.WriteString("Task Context: \"Implement pagination for large database queries\"\n\n")
	sb.WriteString("Current Playbook: [Basic database query examples]\n\n")
	sb.WriteString("Reflection: \"The agent used a fixed LIMIT 100 instead of proper cursor-based pagination, ")
	sb.WriteString("causing inconsistent results when data changed between page requests.\"\n\n")
	sb.WriteString("Response:\n")
	sb.WriteString("```\n")
	sb.WriteString("{\n")
	sb.WriteString("  \"reasoning\": \"The reflection identifies a pagination anti-pattern where offset-based ")
	sb.WriteString("pagination was used instead of cursor-based. This is crucial for large datasets and should be ")
	sb.WriteString("documented as a best practice.\",\n")
	sb.WriteString("  \"operations\": [\n")
	sb.WriteString("    {\n")
	sb.WriteString("      \"type\": \"ADD\",\n")
	sb.WriteString("      \"section\": \"database_patterns\",\n")
	sb.WriteString("      \"content\": \"Use cursor-based pagination for large datasets\\n- Avoid OFFSET/LIMIT as ")
	sb.WriteString("it's slow and inconsistent when data changes\\n- Use WHERE id > lastSeenId ORDER BY id LIMIT 100\\n- ")
	sb.WriteString("Return the last ID as cursor for next page\\n- This ensures consistent results and better performance\"\n")
	sb.WriteString("    }\n")
	sb.WriteString("  ]\n")
	sb.WriteString("}\n")
	sb.WriteString("```\n\n")

	sb.WriteString("**Your Task:**\n")
	sb.WriteString("Output ONLY a valid JSON object with these exact fields:\n")
	sb.WriteString("- reasoning: Your chain of thought / reasoning / thinking process, detailed analysis\n")
	sb.WriteString("- operations: A list of operations to be performed on the playbook\n")
	sb.WriteString("  - type: The type of operation (\"ADD\" or \"UPDATE\")\n")
	sb.WriteString("  - section: The section to add/update the bullet in (e.g., \"error_handling\", \"api_patterns\", ")
	sb.WriteString("\"optimization\", \"anti_patterns\")\n")
	sb.WriteString("  - content: The new content of the bullet. Note: no need to include bullet_id, the system adds it automatically\n\n")

	sb.WriteString("**Available Operations:**\n")
	sb.WriteString("1. ADD: Create new bullet points with fresh IDs\n")
	sb.WriteString("   - section: The section to add the new bullet to\n")
	sb.WriteString("   - content: The new content of the bullet\n\n")

	sb.WriteString("**RESPONSE FORMAT - Output ONLY this JSON structure (no markdown, no code blocks):**\n")
	sb.WriteString("{\n")
	sb.WriteString("  \"reasoning\": \"[Your chain of thought / reasoning / thinking process, detailed analysis here]\",\n")
	sb.WriteString("  \"operations\": [\n")
	sb.WriteString("    {\n")
	sb.WriteString("      \"type\": \"ADD\",\n")
	sb.WriteString("      \"section\": \"strategies_and_patterns\",\n")
	sb.WriteString("      \"content\": \"[New strategy or pattern...]\"\n")
	sb.WriteString("    }\n")
	sb.WriteString("  ]\n")
	sb.WriteString("}\n")

	return sb.String()
}

// BuildRefinementPrompt creates a prompt for refining playbook quality.
// This prompts the curator to identify redundant or low-quality bullets.
func (pb *PromptBuilder) BuildRefinementPrompt(currentPlaybook string, stats PlaybookStats) string {
	var sb strings.Builder

	sb.WriteString("You are a master curator of knowledge. Your job is to analyze an existing playbook ")
	sb.WriteString("and identify bullets that should be removed or merged.\n\n")

	sb.WriteString("**Context:**\n")
	sb.WriteString("- The playbook has grown over time and may contain redundant or low-quality entries\n")
	sb.WriteString("- Your job is to identify bullets that:\n")
	sb.WriteString("  1. Are semantically redundant (duplicates with different wording)\n")
	sb.WriteString("  2. Have low utility (harmful > helpful)\n")
	sb.WriteString("  3. Are too vague or not actionable\n")
	sb.WriteString("  4. Contradict other bullets\n\n")

	sb.WriteString("**Playbook Statistics:**\n")
	sb.WriteString(fmt.Sprintf("- Total bullets: %d\n", stats.TotalBullets))
	sb.WriteString(fmt.Sprintf("- Average helpful count: %.2f\n", stats.AvgHelpfulCount))
	sb.WriteString(fmt.Sprintf("- Average harmful count: %.2f\n\n", stats.AvgHarmfulCount))

	sb.WriteString("**Current Playbook:**\n")
	sb.WriteString(fmt.Sprintf("%s\n\n", currentPlaybook))

	sb.WriteString("**Your Task:**\n")
	sb.WriteString("Analyze the playbook and identify bullets that should be removed or merged.\n\n")

	sb.WriteString("Output ONLY a valid JSON object with these exact fields:\n")
	sb.WriteString("- reasoning: Your analysis of redundancy and quality issues\n")
	sb.WriteString("- operations: A list of refinement operations\n")
	sb.WriteString("  - type: \"REMOVE\" or \"MERGE\"\n")
	sb.WriteString("  - bullet_ids: List of bullet IDs to remove or merge\n")
	sb.WriteString("  - reason: Why these bullets should be removed/merged\n\n")

	sb.WriteString("**RESPONSE FORMAT:**\n")
	sb.WriteString("{\n")
	sb.WriteString("  \"reasoning\": \"[Your analysis]\",\n")
	sb.WriteString("  \"operations\": [\n")
	sb.WriteString("    {\n")
	sb.WriteString("      \"type\": \"REMOVE\",\n")
	sb.WriteString("      \"bullet_ids\": [\"B001\", \"B002\"],\n")
	sb.WriteString("      \"reason\": \"Redundant - both say the same thing\"\n")
	sb.WriteString("    }\n")
	sb.WriteString("  ]\n")
	sb.WriteString("}\n")

	return sb.String()
}

// CurationRequest contains all the data needed for the curator prompt.
type CurationRequest struct {
	TaskContext     string // The original task/query.
	CurrentPlaybook string // The current playbook content.
	Reflection      string // The reflection from the Reflector.
}

// PlaybookStats contains statistics about the playbook for refinement.
type PlaybookStats struct {
	TotalBullets     int
	AvgHelpfulCount  float64
	AvgHarmfulCount  float64
	LowUtilityCount  int // Bullets with harmful > helpful.
	HighUtilityCount int // Bullets with helpful > 10.
}

// CurationResponse represents the LLM response from the curator prompt.
type CurationResponse struct {
	Reasoning  string              `json:"reasoning"`
	Operations []CurationOperation `json:"operations"`
}

// CurationOperation represents a single playbook modification operation.
type CurationOperation struct {
	Type    string `json:"type"`    // "ADD" or "UPDATE".
	Section string `json:"section"` // Section to add/update.
	Content string `json:"content"` // Bullet content.
}

// RefinementResponse represents the LLM response from the refinement prompt.
type RefinementResponse struct {
	Reasoning  string                `json:"reasoning"`
	Operations []RefinementOperation `json:"operations"`
}

// RefinementOperation represents a refinement operation (remove/merge).
type RefinementOperation struct {
	Type      string   `json:"type"`       // "REMOVE" or "MERGE".
	BulletIDs []string `json:"bullet_ids"` // IDs to remove/merge.
	Reason    string   `json:"reason"`     // Why this operation is needed.
}

// FormatReflectionForCurator converts a Reflector insight into text for the curator.
func FormatReflectionForCurator(insight *reflector.Insight) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("**Category:** %s\n", insight.Category))
	sb.WriteString(fmt.Sprintf("**Confidence:** %.2f\n\n", insight.Confidence))

	// Main insight content.
	sb.WriteString(fmt.Sprintf("**Insight:** %s\n\n", insight.Content))

	// Supporting evidence if available.
	if len(insight.Evidence) > 0 {
		sb.WriteString("**Evidence:**\n")

		for _, evidence := range insight.Evidence {
			sb.WriteString(fmt.Sprintf("- %s\n", evidence))
		}

		sb.WriteString("\n")
	}

	return sb.String()
}
