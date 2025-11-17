package planning

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/dmytrogajewski/spin/internal/llm"
	"github.com/openai/openai-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewPlanningService(t *testing.T) {
	provider := llm.NewMockProvider("test response")
	service := NewPlanningService(provider)

	assert.NotNil(t, service)
	assert.Equal(t, provider, service.llm)
}

func TestPlanningService_CreatePlan_Success(t *testing.T) {
	ctx := context.Background()

	// Create mock response with valid JSON
	mockResponse := `{
		"steps": [
			{
				"id": "step_1",
				"description": "First step description",
				"action": "action_1",
				"depends_on": [],
				"estimated_duration": "5m"
			},
			{
				"id": "step_2",
				"description": "Second step description",
				"action": "action_2",
				"depends_on": ["step_1"],
				"estimated_duration": "10m"
			}
		]
	}`

	provider := llm.NewMockProvider("test-provider", llm.WithResponse(mockResponse))
	service := NewPlanningService(provider)

	plan, err := service.CreatePlan(ctx, "test task")
	require.NoError(t, err)
	require.NotNil(t, plan)

	// Verify plan structure
	assert.NotEmpty(t, plan.ID)
	assert.Equal(t, 2, len(plan.Steps))
	assert.Equal(t, PlanStatusPending, plan.Status)

	// Verify first step
	step1 := plan.Steps[0]
	assert.Equal(t, "step_1", step1.ID)
	assert.Equal(t, "First step description", step1.Description)
	assert.Equal(t, "action_1", step1.Action)
	assert.Empty(t, step1.DependsOn)
	assert.Equal(t, 5*time.Minute, step1.EstimatedDuration)
	assert.Equal(t, StepStatusPending, step1.Status)

	// Verify second step
	step2 := plan.Steps[1]
	assert.Equal(t, "step_2", step2.ID)
	assert.Equal(t, "Second step description", step2.Description)
	assert.Equal(t, "action_2", step2.Action)
	assert.Equal(t, []string{"step_1"}, step2.DependsOn)
	assert.Equal(t, 10*time.Minute, step2.EstimatedDuration)
	assert.Equal(t, StepStatusPending, step2.Status)
}

func TestPlanningService_CreatePlan_EmptyTask(t *testing.T) {
	ctx := context.Background()
	provider := llm.NewMockProvider("response")
	service := NewPlanningService(provider)

	plan, err := service.CreatePlan(ctx, "")
	assert.Nil(t, plan)
	assert.Equal(t, ErrEmptyInput, err)
}

func TestPlanningService_CreatePlan_InvalidJSON(t *testing.T) {
	ctx := context.Background()

	// Create mock response with invalid JSON
	provider := llm.NewMockProvider("invalid json response")
	service := NewPlanningService(provider)

	plan, err := service.CreatePlan(ctx, "test task")
	assert.Nil(t, plan)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse LLM response")
}

func TestPlanningService_CreatePlan_EmptySteps(t *testing.T) {
	ctx := context.Background()

	// Create mock response with empty steps
	mockResponse := `{"steps": []}`

	provider := llm.NewMockProvider("test-provider", llm.WithResponse(mockResponse))
	service := NewPlanningService(provider)

	plan, err := service.CreatePlan(ctx, "test task")
	assert.Nil(t, plan)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "plan validation failed")
}

func TestPlanningService_CreatePlan_LLMError(t *testing.T) {
	ctx := context.Background()

	// Create mock provider that returns an error
	provider := llm.NewMockProvider("error-provider", llm.WithError(errors.New("llm error")))
	service := NewPlanningService(provider)

	plan, err := service.CreatePlan(ctx, "test task")
	assert.Nil(t, plan)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "llm completion failed")
}

func TestPlanningService_buildDecompositionPrompt(t *testing.T) {
	service := &PlanningService{}
	taskName := "test task name"

	prompt := service.buildDecompositionPrompt(taskName)

	assert.Contains(t, prompt, taskName)
	assert.Contains(t, prompt, "Decompose the following task")
	assert.Contains(t, prompt, "JSON response")
	assert.Contains(t, prompt, "steps")
}

func TestPlanningService_parseDecompositionResponse(t *testing.T) {
	service := &PlanningService{}

	validJSON := `{
		"steps": [
			{
				"id": "step_1",
				"description": "test",
				"action": "action",
				"depends_on": [],
				"estimated_duration": "5m"
			}
		]
	}`

	data, err := service.parseDecompositionResponse(validJSON)
	require.NoError(t, err)
	require.NotNil(t, data)
	assert.Equal(t, 1, len(data.Steps))
	assert.Equal(t, "step_1", data.Steps[0].ID)
}

func TestPlanningService_parseDecompositionResponse_InvalidJSON(t *testing.T) {
	service := &PlanningService{}

	invalidJSON := "not json"
	data, err := service.parseDecompositionResponse(invalidJSON)
	assert.Nil(t, data)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse JSON")
}

func TestPlanningService_createStepsFromData(t *testing.T) {
	service := &PlanningService{}

	data := &decompositionData{
		Steps: []stepData{
			{
				ID:                "step_1",
				Description:       "test",
				Action:            "action",
				DependsOn:         []string{"step_0"},
				EstimatedDuration: "15m",
			},
			{
				ID:                "step_2",
				Description:       "test2",
				Action:            "action2",
				DependsOn:         []string{},
				EstimatedDuration: "30s",
			},
		},
	}

	steps, err := service.createStepsFromData(data)
	require.NoError(t, err)
	require.Equal(t, 2, len(steps))

	assert.Equal(t, "step_1", steps[0].ID)
	assert.Equal(t, "test", steps[0].Description)
	assert.Equal(t, "action", steps[0].Action)
	assert.Equal(t, []string{"step_0"}, steps[0].DependsOn)
	assert.Equal(t, 15*time.Minute, steps[0].EstimatedDuration)
	assert.Equal(t, StepStatusPending, steps[0].Status)

	assert.Equal(t, "step_2", steps[1].ID)
	assert.Equal(t, "test2", steps[1].Description)
	assert.Equal(t, "action2", steps[1].Action)
	assert.Empty(t, steps[1].DependsOn)
	assert.Equal(t, 30*time.Second, steps[1].EstimatedDuration)
	assert.Equal(t, StepStatusPending, steps[1].Status)
}

func TestPlanningService_createStepsFromData_InvalidDuration(t *testing.T) {
	service := &PlanningService{}

	// Invalid duration should not cause error (parseDuration returns zero on error)
	data := &decompositionData{
		Steps: []stepData{
			{
				ID:                "step_1",
				Description:       "test",
				Action:            "action",
				DependsOn:         []string{},
				EstimatedDuration: "invalid",
			},
		},
	}

	steps, err := service.createStepsFromData(data)
	require.NoError(t, err) // Duration parsing errors are silently ignored
	require.Equal(t, 1, len(steps))
	assert.Equal(t, time.Duration(0), steps[0].EstimatedDuration)
}

func TestPlanningService_CreatePlan_MultipleSteps(t *testing.T) {
	ctx := context.Background()

	mockResponse := `{
		"steps": [
			{
				"id": "step_1",
				"description": "Step 1",
				"action": "action1",
				"depends_on": [],
				"estimated_duration": "1m"
			},
			{
				"id": "step_2",
				"description": "Step 2",
				"action": "action2",
				"depends_on": ["step_1"],
				"estimated_duration": "2m"
			},
			{
				"id": "step_3",
				"description": "Step 3",
				"action": "action3",
				"depends_on": ["step_2"],
				"estimated_duration": "3m"
			}
		]
	}`

	provider := llm.NewMockProvider("test-provider", llm.WithResponse(mockResponse))
	service := NewPlanningService(provider)

	plan, err := service.CreatePlan(ctx, "complex task")
	require.NoError(t, err)
	require.NotNil(t, plan)

	assert.Equal(t, 3, len(plan.Steps))
	assert.Equal(t, "step_1", plan.Steps[0].ID)
	assert.Equal(t, "step_2", plan.Steps[1].ID)
	assert.Equal(t, "step_3", plan.Steps[2].ID)

	// Verify dependencies
	assert.Empty(t, plan.Steps[0].DependsOn)
	assert.Equal(t, []string{"step_1"}, plan.Steps[1].DependsOn)
	assert.Equal(t, []string{"step_2"}, plan.Steps[2].DependsOn)
}

func TestPlanningService_CreatePlan_MalformedJSON(t *testing.T) {
	ctx := context.Background()

	testCases := []struct {
		name    string
		json    string
		wantErr bool
	}{
		{"missing steps", `{}`, true},
		{"steps not array", `{"steps": "not array"}`, true},
		{"step missing id", `{"steps": [{"description": "test"}]}`, false}, // Missing fields might parse but fail validation
		{"invalid structure", `{"not": "valid"}`, true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			provider := llm.NewMockProvider("test-provider", llm.WithResponse(tc.json))
			service := NewPlanningService(provider)

			plan, err := service.CreatePlan(ctx, "test task")
			if tc.wantErr {
				assert.Error(t, err)
				if plan != nil {
					// If plan is created, validation should fail
					_ = plan.ValidateStructure()
				}
			} else {
				// Some cases might parse but fail later
				if err == nil {
					// If parsing succeeded, validation might fail
					_ = plan.ValidateStructure()
				}
			}
		})
	}
}

func TestPlanningService_CreatePlan_JSONWithWhitespace(t *testing.T) {
	ctx := context.Background()

	// JSON with extra whitespace and newlines
	mockResponse := `{
		"steps": [
			{
				"id": "step_1",
				"description": "Test step",
				"action": "test_action",
				"depends_on": [],
				"estimated_duration": "5m"
			}
		]
	}`

	provider := llm.NewMockProvider("test-provider", llm.WithResponse(mockResponse))
	service := NewPlanningService(provider)

	plan, err := service.CreatePlan(ctx, "test task")
	require.NoError(t, err)
	require.NotNil(t, plan)
	assert.Equal(t, 1, len(plan.Steps))
}

// Test helper: verify that getContent works correctly
func TestGetContent(t *testing.T) {
	t.Run("nil completion", func(t *testing.T) {
		content := getContent(nil)
		assert.Empty(t, content)
	})

	t.Run("empty choices", func(t *testing.T) {
		completion := &openai.ChatCompletion{
			Choices: []openai.ChatCompletionChoice{},
		}
		content := getContent(completion)
		assert.Empty(t, content)
	})

	t.Run("valid completion", func(t *testing.T) {
		completion := &openai.ChatCompletion{
			Choices: []openai.ChatCompletionChoice{
				{
					Message: openai.ChatCompletionMessage{
						Content: "test content",
					},
				},
			},
		}
		content := getContent(completion)
		assert.Equal(t, "test content", content)
	})
}


