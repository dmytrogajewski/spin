package acp

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/coder/acp-go-sdk"
	"github.com/dmytrogajewski/spin/internal/agent"
	"github.com/dmytrogajewski/spin/internal/cycle"
	"github.com/dmytrogajewski/spin/internal/detection"
	"github.com/dmytrogajewski/spin/internal/events"
	"github.com/dmytrogajewski/spin/internal/llm"
	"github.com/dmytrogajewski/spin/internal/mcp"
	"github.com/dmytrogajewski/spin/internal/planning"
	"github.com/dmytrogajewski/spin/internal/security"
	"github.com/dmytrogajewski/spin/internal/tools"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestConvertOrchestrationPlanToACP tests conversion of planning.Plan to ACP PlanEntry[].
func TestConvertOrchestrationPlanToACP(t *testing.T) {
	plan := &planning.Plan{
		ID:   "test-plan",
		Task: "Test task",
		Steps: []planning.Step{
			{
				ID:                "step-1",
				Description:       "First step",
				Action:            "Do something",
				DependsOn:         []string{},
				Status:            planning.StepStatusPending,
				EstimatedDuration: 5 * time.Minute,
			},
			{
				ID:                "step-2",
				Description:       "Second step",
				Action:            "Do something else",
				DependsOn:         []string{"step-1"},
				Status:            planning.StepStatusPending,
				EstimatedDuration: 10 * time.Minute,
			},
		},
		Status: planning.PlanStatusPending,
	}

	entries := convertOrchestrationPlanToACP(plan)

	require.Len(t, entries, 2, "should convert all steps")

	// Check first step
	assert.Equal(t, "First step: Do something", entries[0].Content)
	assert.Equal(t, acp.PlanEntryStatusPending, entries[0].Status)
	assert.Equal(t, acp.PlanEntryPriorityHigh, entries[0].Priority, "step with no dependencies should be high priority")

	// Check second step
	assert.Equal(t, "Second step: Do something else", entries[1].Content)
	assert.Equal(t, acp.PlanEntryStatusPending, entries[1].Status)
	assert.Equal(t, acp.PlanEntryPriorityMedium, entries[1].Priority, "step with dependencies should be medium priority")
}

// TestConvertOrchestrationPlanToACP_StatusMapping tests status mapping.
func TestConvertOrchestrationPlanToACP_StatusMapping(t *testing.T) {
	tests := []struct {
		name           string
		stepStatus     planning.StepStatus
		expectedStatus acp.PlanEntryStatus
	}{
		{"pending", planning.StepStatusPending, acp.PlanEntryStatusPending},
		{"ready", planning.StepStatusReady, acp.PlanEntryStatusPending},
		{"running", planning.StepStatusRunning, acp.PlanEntryStatus("in_progress")},
		{"completed", planning.StepStatusCompleted, acp.PlanEntryStatus("completed")},
		{"failed", planning.StepStatusFailed, acp.PlanEntryStatus("failed")},
		{"skipped", planning.StepStatusSkipped, acp.PlanEntryStatus("cancelled")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plan := &planning.Plan{
				ID:   "test-plan",
				Task: "Test task",
				Steps: []planning.Step{
					{
						ID:          "step-1",
						Description: "Test step",
						Action:      "Test action",
						Status:      tt.stepStatus,
					},
				},
			}

			entries := convertOrchestrationPlanToACP(plan)
			require.Len(t, entries, 1)
			assert.Equal(t, tt.expectedStatus, entries[0].Status)
		})
	}
}

// TestConvertOrchestrationPlanToACP_PriorityMapping tests priority mapping based on dependencies.
func TestConvertOrchestrationPlanToACP_PriorityMapping(t *testing.T) {
	plan := &planning.Plan{
		ID:   "test-plan",
		Task: "Test task",
		Steps: []planning.Step{
			{
				ID:          "step-1",
				Description: "No dependencies",
				Action:      "Action 1",
				DependsOn:   []string{},
			},
			{
				ID:          "step-2",
				Description: "Has dependencies",
				Action:      "Action 2",
				DependsOn:   []string{"step-1"},
			},
			{
				ID:          "step-3",
				Description: "Also has dependencies",
				Action:      "Action 3",
				DependsOn:   []string{"step-1"},
			},
		},
	}

	entries := convertOrchestrationPlanToACP(plan)

	require.Len(t, entries, 3)
	assert.Equal(t, acp.PlanEntryPriorityHigh, entries[0].Priority, "step with no dependencies should be high priority")
	assert.Equal(t, acp.PlanEntryPriorityMedium, entries[1].Priority, "step with dependencies should be medium priority")
	assert.Equal(t, acp.PlanEntryPriorityMedium, entries[2].Priority, "step with dependencies should be medium priority")
}

// TestSendPlanNotifications_WithOrchestrationPlan tests plan notification with agent plan.
func TestSendPlanNotifications_WithOrchestrationPlan(t *testing.T) {
	// Create agent
	validator := security.NewValidator()
	emitter := events.NewEventEmitter(100)
	approvalService := security.NewApprovalServiceWithConfig(security.ApprovalServiceConfig{Handler: nil, Emitter: emitter, Validator: validator})
	securityService := security.NewSecurityService(validator, approvalService)
	detectionService := detection.NewDetectionService(cycle.NewDetector(cycle.Config{Enabled: false}), nil)
	toolRuntime := agent.NewToolRuntime(agent.ToolRuntimeConfig{
		Registry:        tools.NewRegistry(),
		Validator:       validator,
		ApprovalService: approvalService,
		Emitter:         emitter,
		WorkDir:         "/tmp",
	})

	provider := llm.NewMockProvider("test")
	agentInstance, err := agent.NewAgent(
		provider,
		securityService,
		detectionService,
		toolRuntime,
		planning.NewPlanningService(provider),
		&agent.Environment{WorkDir: "/tmp"},
		emitter,
	)
	require.NoError(t, err)

	// Set plan on agent
	plan := &planning.Plan{
		ID:   "test-plan",
		Task: "Test task",
		Steps: []planning.Step{
			{
				ID:          "step-1",
				Description: "First step",
				Action:      "Do something",
				Status:      planning.StepStatusPending,
			},
		},
	}
	agentInstance.SetPlanner(plan)

	acpAgent, err := NewSpinACPAgentWithStorage(
		agentInstance,
		mcp.NewService(mcp.NewDefaultRegistryManager(slog.Default())),
		emitter,
		nil,
	)
	require.NoError(t, err)

	mockConn := &mockConnectionForPlan{}
	acpAgent.SetNotificationSender(mockConn)

	// Create agent response
	agentResp := &agent.AgentResponse{
		Output:  "Plan created",
		Success: true,
	}

	// Send plan notifications (should detect agent plan)
	err = acpAgent.sendPlanNotifications(context.Background(), acp.SessionId("test-session"), agentResp)
	require.NoError(t, err)

	// Verify notification was sent
	notifications := mockConn.GetNotifications()
	require.Greater(t, len(notifications), 0, "should send plan notification")

	// Find plan notification
	found := false
	for _, notif := range notifications {
		if notif.Update.Plan != nil {
			found = true
			assert.Len(t, notif.Update.Plan.Entries, 1)
			assert.Equal(t, "First step: Do something", notif.Update.Plan.Entries[0].Content)
			break
		}
	}
	assert.True(t, found, "should send plan notification with agent plan")
}

// TestSendPlanNotifications_FallbackToTextDetection tests fallback to text-based detection.
func TestSendPlanNotifications_FallbackToTextDetection(t *testing.T) {
	acpAgent, err := NewSpinACPAgentWithStorage(
		&agent.Agent{},
		mcp.NewService(mcp.NewDefaultRegistryManager(slog.Default())),
		events.NewEventEmitter(100),
		nil,
	)
	require.NoError(t, err)

	mockConn := &mockConnectionForPlan{}
	acpAgent.SetNotificationSender(mockConn)

	// Agent response with plan-like text but no structured plan
	agentResp := &agent.AgentResponse{
		Output: `
Plan:
1. First step
2. Second step
3. Third step
`,
		Success: true,
	}

	err = acpAgent.sendPlanNotifications(context.Background(), acp.SessionId("test-session"), agentResp)
	require.NoError(t, err)

	// Verify notification was sent (text-based detection should work)
	notifications := mockConn.GetNotifications()
	require.Greater(t, len(notifications), 0, "should send plan notification via text detection")
}

// Note: mockConnectionForPlan is defined in plan_notifications_test.go
