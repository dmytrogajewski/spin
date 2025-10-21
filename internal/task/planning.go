package task

import (
	"errors"
	"fmt"
)

// Constants for Planning mode configuration
const (
	// PlanningMaxTokens is the token budget for Planning mode
	// Sufficient for detailed multi-step plans with dependencies
	PlanningMaxTokens = 4096

	// PlanningMinSteps is the minimum number of steps for a plan
	PlanningMinSteps = 1

	// PlanningMaxSteps is the maximum number of steps for a plan
	PlanningMaxSteps = 100

	// PlanningMinEstimate is the minimum estimated duration per step (minutes)
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

// NewPlanning creates a new Planning task instance.
func NewPlanning() *Planning {
	return &Planning{
		name:         "planning",
		systemPrompt: "You are a planning assistant. Break down complex tasks into manageable steps with clear dependencies and estimates.",
		maxTokens:    PlanningMaxTokens,
	}
}

func (p *Planning) Name() string {
	return p.name
}

func (p *Planning) SystemPrompt() string {
	return p.systemPrompt
}

func (p *Planning) AllowedTools() []string {
	// Planning mode allows analysis tools (context-only, no file reading)
	return []string{"list_directory", "file_search", "git_context", "get_context"}
}

func (p *Planning) MaxTokens() int {
	return p.maxTokens
}

func (p *Planning) Validate() error {
	if p.maxTokens <= 0 {
		return errors.New("max tokens must be positive")
	}
	if p.maxTokens > MaxAllowedTokens {
		return fmt.Errorf("max tokens %d exceeds maximum allowed %d", p.maxTokens, MaxAllowedTokens)
	}
	return nil
}
