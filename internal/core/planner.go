package core

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"text/template"
	"time"

	"github.com/dmytrogajewski/spin/internal/llm"
)

// Planner-related errors
var (
	ErrLLMFailed       = errors.New("LLM request failed")
	ErrInvalidResponse = errors.New("invalid LLM response format")
)

// Planner implements task planning and decomposition for complex multi-step operations.
// It uses an LLM to break down high-level tasks into concrete, executable steps with
// dependency tracking and status management.
//
// Example usage:
//
//	planner := NewPlanner(llmProvider, DefaultPlannerConfig())
//	plan, err := planner.Plan(ctx, "Refactor authentication module")
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	for _, step := range plan.Steps {
//	    fmt.Printf("%s: %s\n", step.ID, step.Description)
//	}
type Planner struct {
	llm     llm.Provider
	config  PlannerConfig
	prompts *planningPrompts
}

// PlannerConfig configures the planner behavior
type PlannerConfig struct {
	MaxSteps        int           // Maximum steps allowed per plan (default: 100)
	Timeout         time.Duration // Planning timeout (default: 30s)
	Temperature     float64       // LLM temperature for planning (default: 0.7)
	EnableStreaming bool          // Stream plan generation (future enhancement)
}

// planningPrompts contains prompt templates
type planningPrompts struct {
	systemPrompt *template.Template
}

// Default configuration values
const (
	defaultMaxSteps    = 100
	defaultTimeout     = 30 * time.Second
	defaultTemperature = 0.7
)

// Planning prompt template
const planningPromptTemplate = `You are a task planning assistant. Your job is to decompose a high-level task into concrete, executable steps.

Task: {{.Task}}

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

Now create a plan for the given task. Respond with ONLY the JSON, no additional text.`

// NewPlanner creates a new planner with the given LLM provider and configuration
func NewPlanner(provider llm.Provider, config PlannerConfig) *Planner {
	// Parse prompt template
	tmpl := template.Must(template.New("planning").Parse(planningPromptTemplate))

	return &Planner{
		llm:    provider,
		config: config,
		prompts: &planningPrompts{
			systemPrompt: tmpl,
		},
	}
}

// DefaultPlannerConfig returns default planner configuration
func DefaultPlannerConfig() PlannerConfig {
	return PlannerConfig{
		MaxSteps:        defaultMaxSteps,
		Timeout:         defaultTimeout,
		Temperature:     defaultTemperature,
		EnableStreaming: false,
	}
}

// Plan decomposes a task into executable steps using the LLM
func (p *Planner) Plan(ctx context.Context, task string) (*Plan, error) {
	// Validate input
	if task == "" {
		return nil, fmt.Errorf("%w", ErrEmptyTask)
	}

	// Apply timeout
	ctx, cancel := context.WithTimeout(ctx, p.config.Timeout)
	defer cancel()

	// Check context cancellation
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	// Build prompt
	prompt, err := p.buildPrompt(task)
	if err != nil {
		return nil, fmt.Errorf("failed to build prompt: %w", err)
	}

	// Call LLM
	req := llm.CompletionRequest{
		Messages: []llm.Message{
			{
				Role:    "user",
				Content: prompt,
			},
		},
		Temperature: p.config.Temperature,
		MaxTokens:   4096, // Enough for detailed plans
	}

	resp, err := p.llm.Complete(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrLLMFailed, err)
	}

	// Parse response
	plan, err := p.parseLLMResponse(resp.Content, task)
	if err != nil {
		return nil, err
	}

	// Validate plan
	if err := p.ValidatePlan(plan); err != nil {
		return nil, err
	}

	return plan, nil
}

// buildPrompt constructs the planning prompt from the template
func (p *Planner) buildPrompt(task string) (string, error) {
	var buf strings.Builder
	data := struct {
		Task string
	}{
		Task: task,
	}

	if err := p.prompts.systemPrompt.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	return buf.String(), nil
}

// llmResponse represents the JSON structure from the LLM
type llmResponse struct {
	Steps []llmStep `json:"steps"`
}

// llmStep represents a step in the LLM response
type llmStep struct {
	ID               string   `json:"id"`
	Description      string   `json:"description"`
	Action           string   `json:"action"`
	DependsOn        []string `json:"depends_on"`
	EstimatedMinutes int      `json:"estimated_minutes"`
}

// parseLLMResponse parses the LLM JSON response into a Plan
func (p *Planner) parseLLMResponse(response string, task string) (*Plan, error) {
	// Clean up response (remove markdown code blocks if present)
	response = strings.TrimSpace(response)
	response = strings.TrimPrefix(response, "```json")
	response = strings.TrimPrefix(response, "```")
	response = strings.TrimSuffix(response, "```")
	response = strings.TrimSpace(response)

	// Parse JSON
	var llmResp llmResponse
	if err := json.Unmarshal([]byte(response), &llmResp); err != nil {
		return nil, fmt.Errorf("%w: failed to parse JSON: %v", ErrInvalidResponse, err)
	}

	// Check we have steps
	if len(llmResp.Steps) == 0 {
		return nil, fmt.Errorf("%w: no steps in response", ErrInvalidResponse)
	}

	// Create plan
	plan := NewPlan(task)

	// Convert LLM steps to plan steps
	for _, llmStep := range llmResp.Steps {
		// Validate step has required fields
		if llmStep.ID == "" {
			return nil, fmt.Errorf("%w: step missing ID", ErrInvalidResponse)
		}
		if llmStep.Description == "" {
			return nil, fmt.Errorf("%w: step %s missing description", ErrInvalidResponse, llmStep.ID)
		}

		step := Step{
			ID:                llmStep.ID,
			Description:       llmStep.Description,
			Action:            llmStep.Action,
			DependsOn:         llmStep.DependsOn,
			Status:            StepStatusPending,
			EstimatedDuration: time.Duration(llmStep.EstimatedMinutes) * time.Minute,
			StartedAt:         nil,
			CompletedAt:       nil,
			Result:            nil,
		}

		plan.Steps = append(plan.Steps, step)
	}

	// Calculate total estimated duration
	plan.EstimatedDuration = plan.CalculateEstimatedDuration()

	return plan, nil
}

// ValidatePlan validates a plan structure
func (p *Planner) ValidatePlan(plan *Plan) error {
	// Check max steps
	if len(plan.Steps) > p.config.MaxSteps {
		return fmt.Errorf("%w: plan has %d steps (max: %d)", ErrTooManySteps, len(plan.Steps), p.config.MaxSteps)
	}

	// Use plan's built-in validation
	return plan.ValidateStructure()
}
