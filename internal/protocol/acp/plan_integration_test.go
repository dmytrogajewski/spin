package acp

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/coder/acp-go-sdk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dmytrogajewski/spin/internal/agent"
	"github.com/dmytrogajewski/spin/internal/events"
	"github.com/dmytrogajewski/spin/internal/mcp"
)

// TestConvertOrchestrationPlanToACP tests conversion of Plan to ACP PlanEntry[].
func TestConvertOrchestrationPlanToACP(t *testing.T) {
	t.Parallel()

	plan := &Plan{
		ID:   "test-plan",
		Task: "Test task",
		Steps: []Step{
			{
				ID:                "step-1",
				Description:       "First step",
				Action:            "Do something",
				DependsOn:         []string{},
				Status:            StepStatusPending,
				EstimatedDuration: 5 * time.Minute,
			},
			{
				ID:                "step-2",
				Description:       "Second step",
				Action:            "Do something else",
				DependsOn:         []string{"step-1"},
				Status:            StepStatusPending,
				EstimatedDuration: 10 * time.Minute,
			},
		},
		Status: PlanStatusPending,
	}

	entries := convertOrchestrationPlanToACP(plan)

	require.Len(t, entries, 2, "should convert all steps")

	// Check first step.
	assert.Equal(t, "First step: Do something", entries[0].Content)
	assert.Equal(t, acp.PlanEntryStatusPending, entries[0].Status)
	assert.Equal(t, acp.PlanEntryPriorityHigh, entries[0].Priority, "step with no dependencies should be high priority")

	// Check second step.
	assert.Equal(t, "Second step: Do something else", entries[1].Content)
	assert.Equal(t, acp.PlanEntryStatusPending, entries[1].Status)
	assert.Equal(t, acp.PlanEntryPriorityMedium, entries[1].Priority, "step with dependencies should be medium priority")
}

// TestConvertOrchestrationPlanToACP_StatusMapping tests status mapping.
func TestConvertOrchestrationPlanToACP_StatusMapping(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		stepStatus     StepStatus
		expectedStatus acp.PlanEntryStatus
	}{
		{"pending", StepStatusPending, acp.PlanEntryStatusPending},
		{"ready", StepStatusReady, acp.PlanEntryStatusPending},
		{"running", StepStatusRunning, acp.PlanEntryStatus("in_progress")},
		{"completed", StepStatusCompleted, acp.PlanEntryStatus("completed")},
		{"failed", StepStatusFailed, acp.PlanEntryStatus("failed")},
		{"skipped", StepStatusSkipped, acp.PlanEntryStatus("canceled")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			plan := &Plan{
				ID:   "test-plan",
				Task: "Test task",
				Steps: []Step{
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
	t.Parallel()

	plan := &Plan{
		ID:   "test-plan",
		Task: "Test task",
		Steps: []Step{
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

// TestSendPlanNotifications_WithTextPlan tests plan notification with text-based plan detection.
func TestSendPlanNotifications_WithTextPlan(t *testing.T) {
	t.Parallel()

	acpAgent, err := NewSpinACPAgentWithStorage(
		&agent.Agent{},
		mcp.NewService(mcp.NewDefaultRegistryManager(slog.Default())),
		events.NewEventEmitter(100),
		nil,
	)
	require.NoError(t, err)

	mockConn := &mockConnectionForPlan{}
	acpAgent.SetNotificationSender(mockConn)

	// Create agent response with plan-like text output.
	agentResp := &agent.Response{
		Output: `Plan:
1. First step
`,
		Success: true,
	}

	// Send plan notifications (should detect plan from text).
	err = acpAgent.sendPlanNotifications(context.Background(), acp.SessionId("test-session"), agentResp)
	require.NoError(t, err)

	// Verify notification was sent.
	notifications := mockConn.GetNotifications()
	require.NotEmpty(t, notifications, "should send plan notification")

	// Find plan notification.
	found := false

	for _, notif := range notifications {
		if notif.Update.Plan != nil {
			found = true

			assert.Len(t, notif.Update.Plan.Entries, 1)
			assert.Equal(t, "First step", notif.Update.Plan.Entries[0].Content)

			break
		}
	}

	assert.True(t, found, "should send plan notification with text-based plan")
}

// TestSendPlanNotifications_FallbackToTextDetection tests fallback to text-based detection.
func TestSendPlanNotifications_FallbackToTextDetection(t *testing.T) {
	t.Parallel()

	acpAgent, err := NewSpinACPAgentWithStorage(
		&agent.Agent{},
		mcp.NewService(mcp.NewDefaultRegistryManager(slog.Default())),
		events.NewEventEmitter(100),
		nil,
	)
	require.NoError(t, err)

	mockConn := &mockConnectionForPlan{}
	acpAgent.SetNotificationSender(mockConn)

	// Agent response with plan-like text but no structured plan.
	agentResp := &agent.Response{
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

	// Verify notification was sent (text-based detection should work).
	notifications := mockConn.GetNotifications()
	require.NotEmpty(t, notifications, "should send plan notification via text detection")
}

// Note: mockConnectionForPlan is defined in plan_notifications_test.go.
