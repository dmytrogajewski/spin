package agent

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dmytrogajewski/spin/internal/cycle"
	"github.com/dmytrogajewski/spin/internal/detection"
	"github.com/dmytrogajewski/spin/internal/events"
	"github.com/dmytrogajewski/spin/internal/llm"
	"github.com/dmytrogajewski/spin/internal/planning"
	"github.com/dmytrogajewski/spin/internal/security"
	"github.com/dmytrogajewski/spin/internal/tools"
)

// TestAgent_GetPlanner tests the GetPlanner method.
func TestAgent_GetPlanner(t *testing.T) {
	// Create agent.
	validator := security.NewValidator()
	emitter := events.NewEventEmitter(100)
	approvalService := security.NewApprovalServiceWithConfig(security.ApprovalServiceConfig{Handler: nil, Emitter: emitter, Validator: validator})
	securityService := security.NewSecurityService(validator, approvalService)
	detectionService := detection.NewDetectionService(cycle.NewDetector(cycle.Config{Enabled: false}), nil)
	toolRuntime := NewToolRuntime(ToolRuntimeConfig{
		Registry:        tools.NewRegistry(),
		Validator:       validator,
		ApprovalService: approvalService,
		Emitter:         emitter,
		WorkDir:         "/tmp",
	})

	agent, err := NewAgent(
		llm.NewMockProvider("test"),
		securityService,
		detectionService,
		toolRuntime,
		planning.NewPlanningService(llm.NewMockProvider("test")),
		&Environment{WorkDir: "/tmp"},
		emitter,
	)
	require.NoError(t, err)

	// Set plan on agent.
	plan := &planning.Plan{
		ID:   "test-plan",
		Task: "Test task",
		Steps: []planning.Step{
			{
				ID:          "step-1",
				Description: "Test step",
				Status:      planning.StepStatusPending,
			},
		},
	}
	agent.SetPlanner(plan)

	// Get planner.
	retrievedPlan := agent.GetPlanner()
	require.NotNil(t, retrievedPlan)
	assert.Equal(t, "test-plan", retrievedPlan.ID)
	assert.Len(t, retrievedPlan.Steps, 1)
}

// TestAgent_GetPlanner_NoPlanner tests GetPlanner when no planner is set.
func TestAgent_GetPlanner_NoPlanner(t *testing.T) {
	validator := security.NewValidator()
	emitter := events.NewEventEmitter(100)
	approvalService := security.NewApprovalServiceWithConfig(security.ApprovalServiceConfig{Handler: nil, Emitter: emitter, Validator: validator})
	securityService := security.NewSecurityService(validator, approvalService)
	detectionService := detection.NewDetectionService(cycle.NewDetector(cycle.Config{Enabled: false}), nil)
	toolRuntime := NewToolRuntime(ToolRuntimeConfig{
		Registry:        tools.NewRegistry(),
		Validator:       validator,
		ApprovalService: approvalService,
		Emitter:         emitter,
		WorkDir:         "/tmp",
	})

	agent, err := NewAgent(
		llm.NewMockProvider("test"),
		securityService,
		detectionService,
		toolRuntime,
		planning.NewPlanningService(llm.NewMockProvider("test")),
		&Environment{WorkDir: "/tmp"},
		emitter,
	)
	require.NoError(t, err)

	// Get planner (should be nil).
	retrievedPlan := agent.GetPlanner()
	assert.Nil(t, retrievedPlan)
}
