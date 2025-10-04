package core

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	coretesting "github.com/dmytrogajewski/spin/internal/core/testing"
	"github.com/dmytrogajewski/spin/internal/llm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewPlanner tests planner creation
func TestNewPlanner(t *testing.T) {
	mockLLM := llm.NewMockProvider("test", llm.WithResponse(""))

	planner := NewPlanner(mockLLM, DefaultPlannerConfig())

	assert.NotNil(t, planner)
	assert.NotNil(t, planner.llm)
	assert.NotNil(t, planner.config)
}

// TestPlanner_Plan_SimpleTask tests planning a simple task
func TestPlanner_Plan_SimpleTask(t *testing.T) {
	mockResponse := `{
		"steps": [
			{
				"id": "step-1",
				"description": "Analyze the code",
				"action": "Review existing code structure",
				"depends_on": [],
				"estimated_minutes": 10
			},
			{
				"id": "step-2",
				"description": "Implement changes",
				"action": "Make necessary modifications",
				"depends_on": ["step-1"],
				"estimated_minutes": 20
			}
		]
	}`

	mockLLM := llm.NewMockProvider("test", llm.WithResponse(mockResponse))
	planner := NewPlanner(mockLLM, DefaultPlannerConfig())

	plan, err := planner.Plan(context.Background(), "Refactor authentication")

	require.NoError(t, err)
	assert.NotNil(t, plan)
	assert.Equal(t, "Refactor authentication", plan.Task)
	assert.Equal(t, 2, len(plan.Steps))
	assert.Equal(t, "step-1", plan.Steps[0].ID)
	assert.Equal(t, "step-2", plan.Steps[1].ID)
	assert.Contains(t, plan.Steps[1].DependsOn, "step-1")
}

// TestPlanner_Plan_ComplexTask tests planning with multiple dependencies
func TestPlanner_Plan_ComplexTask(t *testing.T) {
	mockResponse := `{
		"steps": [
			{
				"id": "step-1",
				"description": "Setup foundation",
				"action": "Initialize project structure",
				"depends_on": [],
				"estimated_minutes": 15
			},
			{
				"id": "step-2",
				"description": "Build module A",
				"action": "Implement module A",
				"depends_on": ["step-1"],
				"estimated_minutes": 30
			},
			{
				"id": "step-3",
				"description": "Build module B",
				"action": "Implement module B",
				"depends_on": ["step-1"],
				"estimated_minutes": 25
			},
			{
				"id": "step-4",
				"description": "Integrate modules",
				"action": "Connect A and B",
				"depends_on": ["step-2", "step-3"],
				"estimated_minutes": 20
			}
		]
	}`

	mockLLM := llm.NewMockProvider("test", llm.WithResponse(mockResponse))
	planner := NewPlanner(mockLLM, DefaultPlannerConfig())

	plan, err := planner.Plan(context.Background(), "Build two-module system")

	require.NoError(t, err)
	assert.Equal(t, 4, len(plan.Steps))

	// Verify step-4 depends on both step-2 and step-3
	step4, _ := plan.GetStep("step-4")
	assert.ElementsMatch(t, []string{"step-2", "step-3"}, step4.DependsOn)

	// Verify no cycles
	assert.False(t, plan.HasCycles())

	// Verify topological sort works
	sorted, err := plan.TopologicalSort()
	require.NoError(t, err)
	assert.Equal(t, "step-1", sorted[0].ID) // First
	assert.Equal(t, "step-4", sorted[3].ID) // Last
}

// TestPlanner_Plan_EmptyTask tests empty task error
func TestPlanner_Plan_EmptyTask(t *testing.T) {
	mockLLM := llm.NewMockProvider("test", llm.WithResponse("{}"))
	planner := NewPlanner(mockLLM, DefaultPlannerConfig())

	_, err := planner.Plan(context.Background(), "")

	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrEmptyTask)
}

// TestPlanner_Plan_LLMError tests LLM failure handling
func TestPlanner_Plan_LLMError(t *testing.T) {
	mockLLM := llm.NewMockProvider("mock", llm.WithError(coretesting.ErrMockLLMFailed))
	planner := NewPlanner(mockLLM, DefaultPlannerConfig())

	_, err := planner.Plan(context.Background(), "Some task")

	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrLLMFailed)
}

// TestPlanner_Plan_InvalidJSON tests invalid JSON response
func TestPlanner_Plan_InvalidJSON(t *testing.T) {
	mockLLM := llm.NewMockProvider("test", llm.WithResponse("this is not valid JSON"))
	planner := NewPlanner(mockLLM, DefaultPlannerConfig())

	_, err := planner.Plan(context.Background(), "Some task")

	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidResponse)
}

// TestPlanner_Plan_MalformedResponse tests response with missing fields
func TestPlanner_Plan_MalformedResponse(t *testing.T) {
	tests := []struct {
		name     string
		response string
		wantErr  bool
	}{
		{
			name:     "missing steps array",
			response: `{}`,
			wantErr:  true,
		},
		{
			name:     "empty steps array",
			response: `{"steps": []}`,
			wantErr:  true,
		},
		{
			name: "step missing ID",
			response: `{
				"steps": [{
					"description": "Do something",
					"action": "action",
					"depends_on": [],
					"estimated_minutes": 10
				}]
			}`,
			wantErr: true,
		},
		{
			name: "step missing description",
			response: `{
				"steps": [{
					"id": "step-1",
					"action": "action",
					"depends_on": [],
					"estimated_minutes": 10
				}]
			}`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockLLM := llm.NewMockProvider("test", llm.WithResponse(tt.response))
			planner := NewPlanner(mockLLM, DefaultPlannerConfig())

			_, err := planner.Plan(context.Background(), "Task")

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestPlanner_Plan_CircularDependencies tests circular dependency detection
func TestPlanner_Plan_CircularDependencies(t *testing.T) {
	mockResponse := `{
		"steps": [
			{
				"id": "step-1",
				"description": "Step 1",
				"action": "action1",
				"depends_on": ["step-2"],
				"estimated_minutes": 10
			},
			{
				"id": "step-2",
				"description": "Step 2",
				"action": "action2",
				"depends_on": ["step-1"],
				"estimated_minutes": 10
			}
		]
	}`

	mockLLM := llm.NewMockProvider("test", llm.WithResponse(mockResponse))
	planner := NewPlanner(mockLLM, DefaultPlannerConfig())

	_, err := planner.Plan(context.Background(), "Task")

	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrCircularDeps)
}

// TestPlanner_Plan_InvalidDependency tests dependency on non-existent step
func TestPlanner_Plan_InvalidDependency(t *testing.T) {
	mockResponse := `{
		"steps": [
			{
				"id": "step-1",
				"description": "Step 1",
				"action": "action1",
				"depends_on": ["step-99"],
				"estimated_minutes": 10
			}
		]
	}`

	mockLLM := llm.NewMockProvider("test", llm.WithResponse(mockResponse))
	planner := NewPlanner(mockLLM, DefaultPlannerConfig())

	_, err := planner.Plan(context.Background(), "Task")

	assert.Error(t, err)
}

// TestPlanner_Plan_TooManySteps tests max steps validation
func TestPlanner_Plan_TooManySteps(t *testing.T) {
	// Generate response with 101 steps
	steps := make([]map[string]interface{}, 101)
	for i := 0; i < 101; i++ {
		steps[i] = map[string]interface{}{
			"id":                fmt.Sprintf("step-%d", i+1),
			"description":       fmt.Sprintf("Step %d", i+1),
			"action":            "action",
			"depends_on":        []string{},
			"estimated_minutes": 5,
		}
	}

	response := map[string]interface{}{"steps": steps}
	responseJSON, _ := json.Marshal(response)

	mockLLM := llm.NewMockProvider("test", llm.WithResponse(string(responseJSON)))
	config := PlannerConfig{
		MaxSteps:    100,
		Timeout:     10 * time.Second,
		Temperature: 0.7,
	}
	planner := NewPlanner(mockLLM, config)

	_, err := planner.Plan(context.Background(), "Complex task")

	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrTooManySteps)
}

// TestPlanner_Plan_DuplicateStepIDs tests duplicate ID detection
func TestPlanner_Plan_DuplicateStepIDs(t *testing.T) {
	mockResponse := `{
		"steps": [
			{
				"id": "step-1",
				"description": "First step",
				"action": "action1",
				"depends_on": [],
				"estimated_minutes": 10
			},
			{
				"id": "step-1",
				"description": "Duplicate ID",
				"action": "action2",
				"depends_on": [],
				"estimated_minutes": 10
			}
		]
	}`

	mockLLM := llm.NewMockProvider("test", llm.WithResponse(mockResponse))
	planner := NewPlanner(mockLLM, DefaultPlannerConfig())

	_, err := planner.Plan(context.Background(), "Task")

	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrDuplicateStepID)
}

// TestPlanner_Plan_EstimatedDuration tests duration calculation
func TestPlanner_Plan_EstimatedDuration(t *testing.T) {
	mockResponse := `{
		"steps": [
			{
				"id": "step-1",
				"description": "Step 1",
				"action": "action1",
				"depends_on": [],
				"estimated_minutes": 10
			},
			{
				"id": "step-2",
				"description": "Step 2",
				"action": "action2",
				"depends_on": ["step-1"],
				"estimated_minutes": 20
			},
			{
				"id": "step-3",
				"description": "Step 3",
				"action": "action3",
				"depends_on": ["step-2"],
				"estimated_minutes": 15
			}
		]
	}`

	mockLLM := llm.NewMockProvider("test", llm.WithResponse(mockResponse))
	planner := NewPlanner(mockLLM, DefaultPlannerConfig())

	plan, err := planner.Plan(context.Background(), "Task")

	require.NoError(t, err)

	// Linear dependencies: 10 + 20 + 15 = 45 minutes
	assert.Equal(t, 45*time.Minute, plan.EstimatedDuration)
}

// TestPlanner_Plan_ContextCancellation tests context cancellation
func TestPlanner_Plan_ContextCancellation(t *testing.T) {
	mockLLM := llm.NewMockProvider("test", llm.WithResponse("{}"))
	planner := NewPlanner(mockLLM, DefaultPlannerConfig())

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err := planner.Plan(ctx, "Task")

	assert.Error(t, err)
	assert.ErrorIs(t, err, context.Canceled)
}

// TestPlanner_Plan_Timeout tests planning timeout
func TestPlanner_Plan_Timeout(t *testing.T) {
	mockLLM := llm.NewMockProvider("test", llm.WithResponse("{}"))
	config := PlannerConfig{
		MaxSteps:    100,
		Timeout:     1 * time.Nanosecond, // Very short timeout
		Temperature: 0.7,
	}
	planner := NewPlanner(mockLLM, config)

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()

	time.Sleep(2 * time.Millisecond) // Ensure timeout

	_, err := planner.Plan(ctx, "Task")

	assert.Error(t, err)
	assert.ErrorIs(t, err, context.DeadlineExceeded)
}

// TestPlanner_ValidatePlan tests plan validation
func TestPlanner_ValidatePlan(t *testing.T) {
	tests := []struct {
		name    string
		plan    *Plan
		wantErr bool
		errType error
	}{
		{
			name: "valid plan",
			plan: &Plan{
				Task: "Test task",
				Steps: []Step{
					{ID: "step-1", Description: "Step 1", Action: "action1"},
					{ID: "step-2", Description: "Step 2", Action: "action2", DependsOn: []string{"step-1"}},
				},
			},
			wantErr: false,
		},
		{
			name: "circular dependencies",
			plan: &Plan{
				Task: "Test task",
				Steps: []Step{
					{ID: "step-1", Description: "Step 1", Action: "action1", DependsOn: []string{"step-2"}},
					{ID: "step-2", Description: "Step 2", Action: "action2", DependsOn: []string{"step-1"}},
				},
			},
			wantErr: true,
			errType: ErrCircularDeps,
		},
		{
			name: "too many steps",
			plan: &Plan{
				Task:  "Test task",
				Steps: make([]Step, 101),
			},
			wantErr: true,
			errType: ErrTooManySteps,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			planner := NewPlanner(nil, PlannerConfig{MaxSteps: 100})

			err := planner.ValidatePlan(tt.plan)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errType != nil {
					assert.ErrorIs(t, err, tt.errType)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestPlanner_parseLLMResponse tests JSON parsing
func TestPlanner_parseLLMResponse(t *testing.T) {
	tests := []struct {
		name     string
		response string
		task     string
		wantErr  bool
		check    func(*testing.T, *Plan)
	}{
		{
			name: "valid response",
			response: `{
				"steps": [
					{
						"id": "step-1",
						"description": "Test step",
						"action": "do something",
						"depends_on": [],
						"estimated_minutes": 10
					}
				]
			}`,
			task:    "Test task",
			wantErr: false,
			check: func(t *testing.T, plan *Plan) {
				assert.Equal(t, "Test task", plan.Task)
				assert.Equal(t, 1, len(plan.Steps))
				assert.Equal(t, "step-1", plan.Steps[0].ID)
				assert.Equal(t, 10*time.Minute, plan.Steps[0].EstimatedDuration)
			},
		},
		{
			name:     "invalid JSON",
			response: "not json",
			task:     "Test task",
			wantErr:  true,
		},
		{
			name:     "missing steps",
			response: `{}`,
			task:     "Test task",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			planner := NewPlanner(nil, DefaultPlannerConfig())

			plan, err := planner.parseLLMResponse(tt.response, tt.task)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				if tt.check != nil {
					tt.check(t, plan)
				}
			}
		})
	}
}

// TestDefaultPlannerConfig tests default configuration
func TestDefaultPlannerConfig(t *testing.T) {
	config := DefaultPlannerConfig()

	assert.Equal(t, 100, config.MaxSteps)
	assert.Equal(t, 30*time.Second, config.Timeout)
	assert.Equal(t, 0.7, config.Temperature)
	assert.False(t, config.EnableStreaming)
}

// TestPlanner_Integration tests end-to-end planning
func TestPlanner_Integration(t *testing.T) {
	// Create a realistic planning scenario
	mockResponse := `{
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
				"description": "Implement JWT generation",
				"action": "Create GenerateJWT() function",
				"depends_on": ["step-2"],
				"estimated_minutes": 20
			},
			{
				"id": "step-4",
				"description": "Implement JWT validation",
				"action": "Create ValidateJWT() middleware",
				"depends_on": ["step-2"],
				"estimated_minutes": 20
			},
			{
				"id": "step-5",
				"description": "Update login endpoint",
				"action": "Modify POST /login to return JWT",
				"depends_on": ["step-3"],
				"estimated_minutes": 10
			},
			{
				"id": "step-6",
				"description": "Update protected routes",
				"action": "Add ValidateJWT middleware",
				"depends_on": ["step-4"],
				"estimated_minutes": 15
			},
			{
				"id": "step-7",
				"description": "Write tests",
				"action": "Create test suite for JWT",
				"depends_on": ["step-3", "step-4"],
				"estimated_minutes": 30
			},
			{
				"id": "step-8",
				"description": "Update documentation",
				"action": "Document JWT authentication",
				"depends_on": ["step-5", "step-6"],
				"estimated_minutes": 15
			}
		]
	}`

	mockLLM := llm.NewMockProvider("test", llm.WithResponse(mockResponse))
	planner := NewPlanner(mockLLM, DefaultPlannerConfig())

	// Generate plan
	plan, err := planner.Plan(context.Background(), "Refactor authentication to use JWT")
	require.NoError(t, err)

	// Verify plan structure
	assert.Equal(t, 8, len(plan.Steps))
	assert.NotEmpty(t, plan.ID)
	assert.Equal(t, PlanStatusPending, plan.Status)
	assert.NotZero(t, plan.CreatedAt)

	// Verify dependencies are correct
	assert.False(t, plan.HasCycles())

	// Verify topological sort
	sorted, err := plan.TopologicalSort()
	require.NoError(t, err)
	assert.Equal(t, "step-1", sorted[0].ID)

	// Verify initial ready steps
	ready := plan.GetReadySteps()
	assert.Equal(t, 1, len(ready))
	assert.Equal(t, "step-1", ready[0].ID)

	// Simulate executing step-1
	err = plan.UpdateStepStatus("step-1", StepStatusCompleted)
	require.NoError(t, err)

	// Check new ready steps
	ready = plan.GetReadySteps()
	assert.Equal(t, 1, len(ready))
	assert.Equal(t, "step-2", ready[0].ID)

	// Verify progress
	progress := plan.Progress()
	assert.Equal(t, 12.5, progress) // 1 of 8 = 12.5%
}
