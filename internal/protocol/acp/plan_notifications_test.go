package acp

import (
	"context"
	"log/slog"
	"sync"
	"testing"

	"github.com/coder/acp-go-sdk"
	"github.com/dmytrogajewski/spin/internal/agent"
	"github.com/dmytrogajewski/spin/internal/events"
	"github.com/dmytrogajewski/spin/internal/mcp"
	"github.com/dmytrogajewski/spin/internal/session"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockConnectionForPlan is a mock connection for testing plan notifications.
type mockConnectionForPlan struct {
	mu            sync.Mutex
	notifications []acp.SessionNotification
}

func (m *mockConnectionForPlan) SessionUpdate(ctx context.Context, notification acp.SessionNotification) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.notifications = append(m.notifications, notification)
	return nil
}

func (m *mockConnectionForPlan) RequestPermission(ctx context.Context, params acp.RequestPermissionRequest) (acp.RequestPermissionResponse, error) {
	// Auto-approve for testing by selecting the first allow option
	for _, opt := range params.Options {
		if opt.Kind == acp.PermissionOptionKindAllowOnce || opt.Kind == acp.PermissionOptionKindAllowAlways {
			return acp.RequestPermissionResponse{
				Outcome: acp.NewRequestPermissionOutcomeSelected(opt.OptionId),
			}, nil
		}
	}
	// No allow option found, return cancelled
	return acp.RequestPermissionResponse{
		Outcome: acp.NewRequestPermissionOutcomeCancelled(),
	}, nil
}

func (m *mockConnectionForPlan) GetNotifications() []acp.SessionNotification {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make([]acp.SessionNotification, len(m.notifications))
	copy(result, m.notifications)
	return result
}

func (m *mockConnectionForPlan) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.notifications = nil
}

// createTestACPAgentWithMock creates a SpinACPAgent with a mock connection.
func createTestACPAgentWithMock(t *testing.T) (*SpinACPAgent, *mockConnectionForPlan) {
	t.Helper()
	agentInstance := &agent.Agent{}
	mcpManager := mcp.NewMCPManager(&mcp.Config{EnableMCP: false}, slog.Default())
	emitter := events.NewEventEmitter(100)

	storage, err := session.NewFileStorage(t.TempDir())
	require.NoError(t, err)
	acpAgent, err := NewSpinACPAgentWithStorage(agentInstance, mcpManager, emitter, storage)
	require.NoError(t, err)

	mockConn := &mockConnectionForPlan{}
	acpAgent.SetNotificationSender(mockConn)

	return acpAgent, mockConn
}

func TestDetectPlanFromOutput_NumberedList(t *testing.T) {
	output := `Here's my plan:
1. First step
2. Second step
3. Third step`

	entries := detectPlanFromOutput(output)
	require.Len(t, entries, 3)
	assert.Equal(t, "First step", entries[0].Content)
	assert.Equal(t, "Second step", entries[1].Content)
	assert.Equal(t, "Third step", entries[2].Content)
	assert.Equal(t, acp.PlanEntryPriorityMedium, entries[0].Priority)
	assert.Equal(t, acp.PlanEntryStatusPending, entries[0].Status)
}

func TestDetectPlanFromOutput_PlanHeader(t *testing.T) {
	output := `Plan:
1. Do this
2. Do that
3. Finish`

	entries := detectPlanFromOutput(output)
	require.Len(t, entries, 3)
	assert.Equal(t, "Do this", entries[0].Content)
	assert.Equal(t, "Do that", entries[1].Content)
	assert.Equal(t, "Finish", entries[2].Content)
}

func TestDetectPlanFromOutput_StepsHeader(t *testing.T) {
	output := `Steps:
- Step one
- Step two
- Step three`

	entries := detectPlanFromOutput(output)
	require.Len(t, entries, 3)
	assert.Equal(t, "Step one", entries[0].Content)
	assert.Equal(t, "Step two", entries[1].Content)
	assert.Equal(t, "Step three", entries[2].Content)
}

func TestDetectPlanFromOutput_BulletPoints(t *testing.T) {
	output := `- Task 1
- Task 2
- Task 3`

	entries := detectPlanFromOutput(output)
	require.Len(t, entries, 3)
	assert.Equal(t, "Task 1", entries[0].Content)
	assert.Equal(t, "Task 2", entries[1].Content)
	assert.Equal(t, "Task 3", entries[2].Content)
}

func TestDetectPlanFromOutput_PriorityHigh(t *testing.T) {
	output := `1. Critical task - urgent
2. Important task - high priority
3. Regular task`

	entries := detectPlanFromOutput(output)
	require.Len(t, entries, 3)
	assert.Equal(t, acp.PlanEntryPriorityHigh, entries[0].Priority)
	assert.Equal(t, acp.PlanEntryPriorityHigh, entries[1].Priority)
	assert.Equal(t, acp.PlanEntryPriorityMedium, entries[2].Priority)
}

func TestDetectPlanFromOutput_PriorityLow(t *testing.T) {
	output := `1. Required task
2. Optional task
3. Nice to have task`

	entries := detectPlanFromOutput(output)
	require.Len(t, entries, 3)
	assert.Equal(t, acp.PlanEntryPriorityMedium, entries[0].Priority)
	assert.Equal(t, acp.PlanEntryPriorityLow, entries[1].Priority)
	assert.Equal(t, acp.PlanEntryPriorityLow, entries[2].Priority)
}

func TestDetectPlanFromOutput_NoPlan(t *testing.T) {
	output := `This is just regular text without any plan structure.`

	entries := detectPlanFromOutput(output)
	assert.Empty(t, entries)
}

func TestDetectPlanFromOutput_EmptyOutput(t *testing.T) {
	entries := detectPlanFromOutput("")
	assert.Empty(t, entries)
}

func TestMatchesPlanPattern_NumberedList(t *testing.T) {
	tests := []struct {
		name     string
		line     string
		expected bool
	}{
		{"numbered with dot", "1. Task", true},
		{"numbered with paren", "1) Task", true},
		{"bullet dash", "- Task", true},
		{"bullet star", "* Task", true},
		{"regular text", "Regular text", false},
		{"empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := matchesPlanPattern(tt.line)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExtractPlanEntry(t *testing.T) {
	tests := []struct {
		name     string
		line     string
		prefix   string
		expected *acp.PlanEntry
	}{
		{
			name:   "numbered with dot",
			line:   "1. Task description",
			prefix: "",
			expected: &acp.PlanEntry{
				Content:  "Task description",
				Priority: acp.PlanEntryPriorityMedium,
				Status:   acp.PlanEntryStatusPending,
			},
		},
		{
			name:   "bullet dash",
			line:   "- Task description",
			prefix: "",
			expected: &acp.PlanEntry{
				Content:  "Task description",
				Priority: acp.PlanEntryPriorityMedium,
				Status:   acp.PlanEntryStatusPending,
			},
		},
		{
			name:   "with prefix",
			line:   "1. Task",
			prefix: "Step",
			expected: &acp.PlanEntry{
				Content:  "Step Task",
				Priority: acp.PlanEntryPriorityMedium,
				Status:   acp.PlanEntryStatusPending,
			},
		},
		{
			name:     "empty line",
			line:     "",
			prefix:   "",
			expected: nil,
		},
		{
			name:   "critical priority",
			line:   "1. Critical task",
			prefix: "",
			expected: &acp.PlanEntry{
				Content:  "Critical task",
				Priority: acp.PlanEntryPriorityHigh,
				Status:   acp.PlanEntryStatusPending,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractPlanEntry(tt.line, tt.prefix)
			if tt.expected == nil {
				assert.Nil(t, result)
			} else {
				require.NotNil(t, result)
				assert.Equal(t, tt.expected.Content, result.Content)
				assert.Equal(t, tt.expected.Priority, result.Priority)
				assert.Equal(t, tt.expected.Status, result.Status)
			}
		})
	}
}

func TestSendPlanNotifications_NoConnection(t *testing.T) {
	acpAgent, _ := createTestACPAgentWithMock(t)
	acpAgent.SetNotificationSender(nil) // No connection

	agentResp := &agent.AgentResponse{
		Output: "Plan:\n1. Step one\n2. Step two",
	}

	err := acpAgent.sendPlanNotifications(context.Background(), "session-1", agentResp)
	assert.NoError(t, err) // Should return nil when no connection
}

func TestSendPlanNotifications_NoPlan(t *testing.T) {
	acpAgent, mockConn := createTestACPAgentWithMock(t)

	agentResp := &agent.AgentResponse{
		Output: "This is just regular text without any plan.",
	}

	err := acpAgent.sendPlanNotifications(context.Background(), "session-1", agentResp)
	assert.NoError(t, err)

	// Verify no notifications were sent
	notifications := mockConn.GetNotifications()
	assert.Empty(t, notifications)
}

func TestSendPlanNotifications_WithPlan(t *testing.T) {
	acpAgent, mockConn := createTestACPAgentWithMock(t)

	agentResp := &agent.AgentResponse{
		Output: "Plan:\n1. Step one\n2. Step two\n3. Step three",
	}

	err := acpAgent.sendPlanNotifications(context.Background(), "session-1", agentResp)
	assert.NoError(t, err)

	// Verify plan notification was sent
	notifications := mockConn.GetNotifications()
	require.Len(t, notifications, 1)
	notification := notifications[0]
	assert.Equal(t, "session-1", string(notification.SessionId))
	// Check that Plan is set (it's a union type)
	// The Plan field should be non-nil
	require.NotNil(t, notification.Update.Plan)
	assert.Len(t, notification.Update.Plan.Entries, 3)
	assert.Equal(t, "Step one", notification.Update.Plan.Entries[0].Content)
	assert.Equal(t, "Step two", notification.Update.Plan.Entries[1].Content)
	assert.Equal(t, "Step three", notification.Update.Plan.Entries[2].Content)
}

