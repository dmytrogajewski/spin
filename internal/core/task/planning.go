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
//   - Duration estimation
//
// The Planning mode uses a structured prompt to guide the LLM to produce
// well-formatted JSON plans with steps, dependencies, and time estimates.
//
// Example usage:
//
//	planning := task.NewPlanning()
//	req := &core.AgentRequest{
//	    Input: "Refactor authentication module to use JWT",
//	    Task: planning,
//	}
//	resp, err := agent.Execute(ctx, req)
type Planning struct {
	config *PlanningConfig
}

// PlanningConfig contains configuration options for Planning mode.
type PlanningConfig struct {
	// MaxSteps limits the number of steps in a plan
	// If 0, uses PlanningMaxSteps (100)
	MaxSteps int

	// MinSteps requires a minimum number of steps
	// If 0, uses PlanningMinSteps (1)
	MinSteps int

	// CustomPrompt optionally overrides the default planning prompt
	// Empty string uses the default prompt
	CustomPrompt string
}

// NewPlanning creates a new Planning task mode with default configuration.
func NewPlanning() *Planning {
	return &Planning{
		config: nil,
	}
}

// NewPlanningWithConfig creates a new Planning task mode with custom configuration.
func NewPlanningWithConfig(config *PlanningConfig) *Planning {
	return &Planning{
		config: config,
	}
}

// Name returns the unique identifier for this task mode.
func (p *Planning) Name() string {
	return "planning"
}

// SystemPrompt returns the planning-specific system prompt that instructs
// the LLM on how to decompose tasks into structured plans.
func (p *Planning) SystemPrompt() string {
	if p.config != nil && p.config.CustomPrompt != "" {
		return p.config.CustomPrompt
	}
	return planningSystemPrompt
}

// planningSystemPrompt is the comprehensive system prompt for Planning mode
const planningSystemPrompt = `You are a task planning assistant. Your job is to decompose a high-level task into concrete, executable steps.

Requirements:
1. Break the task into 3-10 clear steps
2. Each step should be concrete and actionable
3. Identify dependencies between steps (which steps must complete before others)
4. Estimate duration for each step (in minutes)
5. Use clear, imperative language for step descriptions

Format your response as JSON:
{
  "steps": [
    {
      "id": "step-1",
      "description": "Clear description of what to do",
      "action": "Specific command or action",
      "depends_on": [],
      "estimated_minutes": 5
    }
  ]
}

Example for "Refactor authentication to use JWT":
{
  "steps": [
    {
      "id": "step-1",
      "description": "Analyze current authentication implementation",
      "action": "Review auth.go and identify current token mechanism",
      "depends_on": [],
      "estimated_minutes": 10
    },
    {
      "id": "step-2",
      "description": "Design JWT token structure",
      "action": "Define JWT claims and expiration policy",
      "depends_on": ["step-1"],
      "estimated_minutes": 15
    },
    {
      "id": "step-3",
      "description": "Implement JWT generation function",
      "action": "Create GenerateJWT() function with signing",
      "depends_on": ["step-2"],
      "estimated_minutes": 20
    },
    {
      "id": "step-4",
      "description": "Implement JWT validation middleware",
      "action": "Create ValidateJWT() middleware function",
      "depends_on": ["step-2"],
      "estimated_minutes": 20
    },
    {
      "id": "step-5",
      "description": "Update login endpoint to issue JWT",
      "action": "Modify /login handler to return JWT token",
      "depends_on": ["step-3"],
      "estimated_minutes": 10
    },
    {
      "id": "step-6",
      "description": "Update protected routes to use JWT middleware",
      "action": "Add ValidateJWT middleware to protected endpoints",
      "depends_on": ["step-4"],
      "estimated_minutes": 15
    },
    {
      "id": "step-7",
      "description": "Write unit tests for JWT functions",
      "action": "Create test cases for token generation and validation",
      "depends_on": ["step-3", "step-4"],
      "estimated_minutes": 25
    },
    {
      "id": "step-8",
      "description": "Update API documentation",
      "action": "Document new JWT authentication in README and API docs",
      "depends_on": ["step-5", "step-6"],
      "estimated_minutes": 10
    }
  ]
}

Now create a plan for the user's task. Respond with ONLY the JSON, no additional text.`

// AllowedTools returns the tools available in Planning mode.
// Planning mode has restricted tool access since it focuses on
// plan generation rather than execution.
func (p *Planning) AllowedTools() []string {
	// Planning mode uses minimal tools - primarily for context gathering
	return []string{
		"get_context", // Gather environment and project context
		"read_file",   // Read files to understand current state
		"list_dir",    // List directory contents
	}
}

// MaxTokens returns the token budget for Planning mode.
func (p *Planning) MaxTokens() int {
	return PlanningMaxTokens
}

// Validate validates the Planning task configuration.
func (p *Planning) Validate() error {
	if p.config == nil {
		return nil // Default config is always valid
	}

	var errs []error

	// Validate max steps
	if p.config.MaxSteps < 0 {
		errs = append(errs, fmt.Errorf("max steps cannot be negative: %d", p.config.MaxSteps))
	}
	if p.config.MaxSteps > 0 && p.config.MaxSteps < p.getMinSteps() {
		errs = append(errs, fmt.Errorf("max steps (%d) must be >= min steps (%d)", p.config.MaxSteps, p.getMinSteps()))
	}

	// Validate min steps
	if p.config.MinSteps < 0 {
		errs = append(errs, fmt.Errorf("min steps cannot be negative: %d", p.config.MinSteps))
	}

	// Validate custom prompt
	if p.config.CustomPrompt != "" && len(p.config.CustomPrompt) < MinPromptLength {
		errs = append(errs, fmt.Errorf("custom prompt too short (%d characters, minimum %d)", len(p.config.CustomPrompt), MinPromptLength))
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}

	return nil
}

// getMaxSteps returns the effective max steps setting
func (p *Planning) getMaxSteps() int {
	if p.config != nil && p.config.MaxSteps > 0 {
		return p.config.MaxSteps
	}
	return PlanningMaxSteps
}

// getMinSteps returns the effective min steps setting
func (p *Planning) getMinSteps() int {
	if p.config != nil && p.config.MinSteps > 0 {
		return p.config.MinSteps
	}
	return PlanningMinSteps
}
