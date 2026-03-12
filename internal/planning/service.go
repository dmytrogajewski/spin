package planning

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/openai/openai-go"

	spinerrors "github.com/dmytrogajewski/spin/internal/errors"
	"github.com/dmytrogajewski/spin/internal/llm"
)

// PlanningService handles task decomposition into execution plans.
type PlanningService struct {
	llm llm.Provider
}

// NewPlanningService creates a new planning service with the given LLM provider.
func NewPlanningService(provider llm.Provider) *PlanningService {
	return &PlanningService{llm: provider}
}

// ErrEmptyInput is returned when task name is empty.
var ErrEmptyInput = errors.New("task name cannot be empty")

// CreatePlan creates a new execution plan for the given task.
// This method uses the LLM to decompose complex tasks into manageable steps.
func (s *PlanningService) CreatePlan(ctx context.Context, taskName string) (*Plan, error) {
	if taskName == "" {
		return nil, ErrEmptyInput
	}

	// Create a new plan.
	plan := NewPlan()

	// Use LLM to decompose the task into steps.
	decompositionPrompt := s.buildDecompositionPrompt(taskName)

	// Call LLM for task decomposition.
	params := openai.ChatCompletionNewParams{
		Messages: openai.F([]openai.ChatCompletionMessageParamUnion{
			openai.UserMessage(decompositionPrompt),
		}),
		MaxTokens:   openai.F(int64(1000)),
		Temperature: openai.F(0.3), // Lower temperature for more consistent planning.
	}

	resp, err := s.llm.Complete(ctx, params)
	if err != nil {
		return nil, spinerrors.New(spinerrors.CodeLLM, "PlanningService.CreatePlan", "llm completion failed", err)
	}

	// Parse the response and create steps.
	responseContent := getContent(resp)

	decomposition, err := s.parseDecompositionResponse(responseContent)
	if err != nil {
		return nil, spinerrors.New(spinerrors.CodeLLM, "PlanningService.CreatePlan", "failed to parse LLM response", err)
	}

	// Create steps from parsed data.
	steps, err := s.createStepsFromData(decomposition)
	if err != nil {
		return nil, spinerrors.New(spinerrors.CodeValidation, "PlanningService.CreatePlan", "failed to create steps", err)
	}

	plan.Steps = steps

	// Validate the plan structure.
	err = plan.ValidateStructure()
	if err != nil {
		return nil, spinerrors.New(spinerrors.CodeValidation, "PlanningService.CreatePlan", "plan validation failed", err)
	}

	return plan, nil
}

// buildDecompositionPrompt creates the prompt for task decomposition.
func (s *PlanningService) buildDecompositionPrompt(taskName string) string {
	return fmt.Sprintf(`
Decompose the following task into specific, actionable steps:

Task: %s

Please provide a JSON response with the following structure:
{
  "steps": [
    {
      "id": "step_1",
      "description": "Clear description of what to do",
      "action": "Specific action to perform",
      "depends_on": [],
      "estimated_duration": "5m"
    }
  ]
}

Guidelines:
- Each step should be atomic and testable
- Include dependencies between steps
- Provide realistic time estimates
- Focus on concrete actions, not abstract concepts
`, taskName)
}

// decompositionData represents the structure of the LLM response.
type decompositionData struct {
	Steps []stepData `json:"steps"`
}

// stepData represents a single step in the decomposition response.
type stepData struct {
	ID                string   `json:"id"`
	Description       string   `json:"description"`
	Action            string   `json:"action"`
	DependsOn         []string `json:"depends_on"`
	EstimatedDuration string   `json:"estimated_duration"`
}

// parseDecompositionResponse parses the JSON response from the LLM.
func (s *PlanningService) parseDecompositionResponse(content string) (*decompositionData, error) {
	var decomposition decompositionData
	err := json.Unmarshal([]byte(content), &decomposition)
	if err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	return &decomposition, nil
}

// createStepsFromData creates Step instances from parsed decomposition data.
func (s *PlanningService) createStepsFromData(data *decompositionData) ([]Step, error) {
	steps := make([]Step, 0, len(data.Steps))
	for _, stepData := range data.Steps {
		duration, _ := time.ParseDuration(stepData.EstimatedDuration)
		step := Step{
			ID:                stepData.ID,
			Description:       stepData.Description,
			Action:            stepData.Action,
			DependsOn:         stepData.DependsOn,
			Status:            StepStatusPending,
			EstimatedDuration: duration,
		}
		steps = append(steps, step)
	}

	return steps, nil
}

// Plan represents an execution plan.
type Plan struct {
	ID                string
	Task              string
	Steps             []Step
	CreatedAt         time.Time
	EstimatedDuration time.Duration
	Status            PlanStatus
}

// PlanStatus represents overall plan state.
type PlanStatus int

const (
	PlanStatusPending PlanStatus = iota
	PlanStatusInProgress
	PlanStatusCompleted
	PlanStatusFailed
	PlanStatusCancelled
)

// Step represents a plan step.
type Step struct {
	ID                string
	Description       string
	Action            string
	DependsOn         []string
	Status            StepStatus
	EstimatedDuration time.Duration
	StartedAt         *time.Time
	CompletedAt       *time.Time
}

// StepStatus represents the status of a plan step.
type StepStatus int

const (
	StepStatusPending StepStatus = iota
	StepStatusReady
	StepStatusRunning
	StepStatusCompleted
	StepStatusFailed
	StepStatusSkipped
)

// NewPlan creates a new plan.
func NewPlan() *Plan {
	return &Plan{
		ID:        fmt.Sprintf("plan-%d", time.Now().Unix()),
		Steps:     []Step{},
		Status:    PlanStatusPending,
		CreatedAt: time.Now(),
	}
}

// ValidateStructure validates the plan structure.
func (p *Plan) ValidateStructure() error {
	if p.ID == "" {
		return errors.New("plan ID cannot be empty")
	}

	if len(p.Steps) == 0 {
		return errors.New("plan must have at least one step")
	}

	return nil
}

// CalculateEstimatedDuration returns the total estimated duration of all steps.
func (p *Plan) CalculateEstimatedDuration() time.Duration {
	var total time.Duration
	for _, step := range p.Steps {
		total += step.EstimatedDuration
	}

	return total
}

// getContent extracts the content from the first choice in a ChatCompletion.
func getContent(completion *openai.ChatCompletion) string {
	if completion == nil || len(completion.Choices) == 0 {
		return ""
	}

	return completion.Choices[0].Message.Content
}
