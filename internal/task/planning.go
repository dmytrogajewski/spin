package task

import (
	"fmt"
)

// Constants for Planning mode configuration.
const (
	// PlanningMaxTokens is the token budget for Planning mode
	// Sufficient for detailed multi-step plans with dependencies.
	PlanningMaxTokens = 4096

	// PlanningMinSteps is the minimum number of steps for a plan.
	PlanningMinSteps = 1

	// PlanningMaxSteps is the maximum number of steps for a plan.
	PlanningMaxSteps = 100

	// PlanningMinEstimate is the minimum estimated duration per step (minutes).
	PlanningMinEstimate = 1
)

// Planning implements a specialized task mode for breaking down complex tasks
// into executable steps with dependency tracking.
//
// Planning mode is designed for:
//   - Task decomposition and breakdown
//   - Multi-step plan generation
//   - Dependency analysis
type Planning struct {
	name         string
	systemPrompt string
	maxTokens    int
}

// PlanningSystemPrompt is the system prompt for planning mode.
// Focused on task decomposition and analysis without execution.
const PlanningSystemPrompt = `You are a technical planning assistant. ` +
	`Your role is to analyze codebases and break down complex tasks into clear, actionable implementation plans.

# Your Role

You can:
- Explore directory structures with list_directory
- Search for patterns and code with file_search
- Understand project context with get_context and git_context

You cannot:
- Read file contents directly (use file_search to find relevant code)
- Modify files or execute commands
- Implement the plan yourself

# Planning Guidelines

1. Start by exploring the codebase structure to understand the project
2. Identify relevant files and components affected by the changes
3. Break down the task into specific, ordered steps
4. Note dependencies between steps
5. Highlight potential risks or areas needing clarification

# Output Format

Provide plans as numbered steps with:
- Clear action description
- Files/components involved
- Dependencies on other steps
- Any assumptions or open questions`

// NewPlanning creates a new Planning task instance.
func NewPlanning() *Planning {
	return &Planning{
		name:         TaskNamePlanning,
		systemPrompt: PlanningSystemPrompt,
		maxTokens:    PlanningMaxTokens,
	}
}

// Name implements the Name operation.
func (p *Planning) Name() string {
	return p.name
}

// SystemPrompt implements the SystemPrompt operation.
func (p *Planning) SystemPrompt() string {
	return p.systemPrompt
}

// AllowedTools implements the AllowedTools operation.
func (p *Planning) AllowedTools() []string {
	// Planning mode allows analysis tools (context-only, no file reading).
	return []string{"list_directory", "file_search", "git_context", "get_context"}
}

// MaxTokens implements the MaxTokens operation.
func (p *Planning) MaxTokens() int {
	return p.maxTokens
}

// Validate implements the Validate operation.
func (p *Planning) Validate() error {
	if p.maxTokens <= 0 {
		return ErrMaxTokensMustBePositive
	}

	if p.maxTokens > MaxAllowedTokens {
		return fmt.Errorf("max tokens %d exceeds maximum allowed %d: %w", p.maxTokens, MaxAllowedTokens, ErrMaxTokensExceedsMaximumAllowed)
	}

	return nil
}
