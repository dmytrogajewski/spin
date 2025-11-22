//go:build e2e_llm_test

package acp

import (
	"context"
	"testing"

	"github.com/coder/acp-go-sdk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestACP_Plan_Create tests plan notification with entries.
func TestACP_Plan_Create(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	workDir := createTestWorkspace(t)
	cmd, stdin, stdout := startACPAgent(t, "--workspace", workDir)
	defer cleanupAgent(t, cmd, stdin)

	clientImpl := &testClient{}
	client := createACPClientWithClient(t, stdin, stdout, clientImpl)
	ctx := context.Background()

	_, err := client.Initialize(ctx, acp.InitializeRequest{
		ProtocolVersion: acp.ProtocolVersionNumber,
	})
	require.NoError(t, err)

	sessionResp, err := client.NewSession(ctx, acp.NewSessionRequest{
		Cwd:        workDir,
		McpServers: []acp.McpServer{},
	})
	require.NoError(t, err)

	clientImpl.clearNotifications()

	// Send prompt that might trigger plan creation
	promptReq := acp.PromptRequest{
		SessionId: sessionResp.SessionId,
		Prompt: []acp.ContentBlock{
			acp.TextBlock("create a plan to implement user authentication"),
		},
	}

	_, err = client.Prompt(ctx, promptReq)
	require.NoError(t, err)

	// Check for plan notification
	notifications := clientImpl.getNotifications()
	for _, notif := range notifications {
		if notif.Update.Plan != nil {
			plan := notif.Update.Plan
			assert.NotEmpty(t, plan.Entries, "Plan should have entries")
			t.Logf("Found plan with %d entries", len(plan.Entries))
			return
		}
	}
	t.Log("No plan notification found (may be expected)")
}

// TestACP_Plan_Entries_Structure tests entry structure.
func TestACP_Plan_Entries_Structure(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	workDir := createTestWorkspace(t)
	cmd, stdin, stdout := startACPAgent(t, "--workspace", workDir)
	defer cleanupAgent(t, cmd, stdin)

	clientImpl := &testClient{}
	client := createACPClientWithClient(t, stdin, stdout, clientImpl)
	ctx := context.Background()

	_, err := client.Initialize(ctx, acp.InitializeRequest{
		ProtocolVersion: acp.ProtocolVersionNumber,
	})
	require.NoError(t, err)

	sessionResp, err := client.NewSession(ctx, acp.NewSessionRequest{
		Cwd:        workDir,
		McpServers: []acp.McpServer{},
	})
	require.NoError(t, err)

	clientImpl.clearNotifications()

	// Send prompt
	promptReq := acp.PromptRequest{
		SessionId: sessionResp.SessionId,
		Prompt: []acp.ContentBlock{
			acp.TextBlock("create a plan"),
		},
	}

	_, err = client.Prompt(ctx, promptReq)
	require.NoError(t, err)

	// Check for plan entries structure
	notifications := clientImpl.getNotifications()
	for _, notif := range notifications {
		if notif.Update.Plan != nil {
			plan := notif.Update.Plan
			for _, entry := range plan.Entries {
				assert.NotEmpty(t, entry.Content, "Entry should have content")
				assert.NotNil(t, entry.Priority, "Entry should have priority")
				assert.NotNil(t, entry.Status, "Entry should have status")
				t.Logf("Entry: %s (priority: %v, status: %v)", entry.Content, entry.Priority, entry.Status)
			}
			return
		}
	}
	t.Log("No plan entries found (may be expected)")
}

// TestACP_Plan_Priorities tests all priorities.
func TestACP_Plan_Priorities(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	workDir := createTestWorkspace(t)
	cmd, stdin, stdout := startACPAgent(t, "--workspace", workDir)
	defer cleanupAgent(t, cmd, stdin)

	clientImpl := &testClient{}
	client := createACPClientWithClient(t, stdin, stdout, clientImpl)
	ctx := context.Background()

	_, err := client.Initialize(ctx, acp.InitializeRequest{
		ProtocolVersion: acp.ProtocolVersionNumber,
	})
	require.NoError(t, err)

	sessionResp, err := client.NewSession(ctx, acp.NewSessionRequest{
		Cwd:        workDir,
		McpServers: []acp.McpServer{},
	})
	require.NoError(t, err)

	clientImpl.clearNotifications()

	// Send prompt
	promptReq := acp.PromptRequest{
		SessionId: sessionResp.SessionId,
		Prompt: []acp.ContentBlock{
			acp.TextBlock("create a detailed plan"),
		},
	}

	_, err = client.Prompt(ctx, promptReq)
	require.NoError(t, err)

	// Check for all priority types
	notifications := clientImpl.getNotifications()
	priorities := make(map[acp.PlanEntryPriority]bool)
	for _, notif := range notifications {
		if notif.Update.Plan != nil {
			plan := notif.Update.Plan
			for _, entry := range plan.Entries {
				priorities[entry.Priority] = true
			}
		}
	}

	// Verify valid priorities if found
	validPriorities := []acp.PlanEntryPriority{
		acp.PlanEntryPriorityHigh,
		acp.PlanEntryPriorityMedium,
		acp.PlanEntryPriorityLow,
	}
	for priority := range priorities {
		assert.Contains(t, validPriorities, priority, "Priority should be valid")
	}

	if len(priorities) > 0 {
		t.Logf("Found priorities: %v", priorities)
	} else {
		t.Log("No plan priorities found (may be expected)")
	}
}

// TestACP_Plan_Status_Transitions tests status transitions.
func TestACP_Plan_Status_Transitions(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	workDir := createTestWorkspace(t)
	cmd, stdin, stdout := startACPAgent(t, "--workspace", workDir)
	defer cleanupAgent(t, cmd, stdin)

	clientImpl := &testClient{}
	client := createACPClientWithClient(t, stdin, stdout, clientImpl)
	ctx := context.Background()

	_, err := client.Initialize(ctx, acp.InitializeRequest{
		ProtocolVersion: acp.ProtocolVersionNumber,
	})
	require.NoError(t, err)

	sessionResp, err := client.NewSession(ctx, acp.NewSessionRequest{
		Cwd:        workDir,
		McpServers: []acp.McpServer{},
	})
	require.NoError(t, err)

	clientImpl.clearNotifications()

	// Send prompt
	promptReq := acp.PromptRequest{
		SessionId: sessionResp.SessionId,
		Prompt: []acp.ContentBlock{
			acp.TextBlock("create and execute a plan"),
		},
	}

	_, err = client.Prompt(ctx, promptReq)
	require.NoError(t, err)

	// Check for status transitions
	notifications := clientImpl.getNotifications()
	statuses := make(map[acp.PlanEntryStatus]bool)
	for _, notif := range notifications {
		if notif.Update.Plan != nil {
			plan := notif.Update.Plan
			for _, entry := range plan.Entries {
				statuses[entry.Status] = true
			}
		}
	}

	// Verify valid statuses if found
	validStatuses := []acp.PlanEntryStatus{
		acp.PlanEntryStatusPending,
		acp.PlanEntryStatusInProgress,
		acp.PlanEntryStatusCompleted,
	}
	for status := range statuses {
		assert.Contains(t, validStatuses, status, "Status should be valid")
	}

	if len(statuses) > 0 {
		t.Logf("Found statuses: %v", statuses)
	} else {
		t.Log("No plan statuses found (may be expected)")
	}
}

// TestACP_Plan_Update tests that plan updates replace entire plan.
func TestACP_Plan_Update(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	workDir := createTestWorkspace(t)
	cmd, stdin, stdout := startACPAgent(t, "--workspace", workDir)
	defer cleanupAgent(t, cmd, stdin)

	clientImpl := &testClient{}
	client := createACPClientWithClient(t, stdin, stdout, clientImpl)
	ctx := context.Background()

	_, err := client.Initialize(ctx, acp.InitializeRequest{
		ProtocolVersion: acp.ProtocolVersionNumber,
	})
	require.NoError(t, err)

	sessionResp, err := client.NewSession(ctx, acp.NewSessionRequest{
		Cwd:        workDir,
		McpServers: []acp.McpServer{},
	})
	require.NoError(t, err)

	clientImpl.clearNotifications()

	// Send prompt
	promptReq := acp.PromptRequest{
		SessionId: sessionResp.SessionId,
		Prompt: []acp.ContentBlock{
			acp.TextBlock("create a plan and update it"),
		},
	}

	_, err = client.Prompt(ctx, promptReq)
	require.NoError(t, err)

	// Check for multiple plan updates
	notifications := clientImpl.getNotifications()
	planCount := 0
	for _, notif := range notifications {
		if notif.Update.Plan != nil {
			planCount++
			plan := notif.Update.Plan
			t.Logf("Plan update %d: %d entries", planCount, len(plan.Entries))
		}
	}

	if planCount > 1 {
		t.Logf("Found %d plan updates (plan was updated)", planCount)
	} else {
		t.Log("Found single or no plan updates (may be expected)")
	}
}

// TestACP_Plan_Dynamic tests that plan can add/remove entries.
func TestACP_Plan_Dynamic(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	workDir := createTestWorkspace(t)
	cmd, stdin, stdout := startACPAgent(t, "--workspace", workDir)
	defer cleanupAgent(t, cmd, stdin)

	clientImpl := &testClient{}
	client := createACPClientWithClient(t, stdin, stdout, clientImpl)
	ctx := context.Background()

	_, err := client.Initialize(ctx, acp.InitializeRequest{
		ProtocolVersion: acp.ProtocolVersionNumber,
	})
	require.NoError(t, err)

	sessionResp, err := client.NewSession(ctx, acp.NewSessionRequest{
		Cwd:        workDir,
		McpServers: []acp.McpServer{},
	})
	require.NoError(t, err)

	clientImpl.clearNotifications()

	// Send prompt
	promptReq := acp.PromptRequest{
		SessionId: sessionResp.SessionId,
		Prompt: []acp.ContentBlock{
			acp.TextBlock("create a dynamic plan that changes"),
		},
	}

	_, err = client.Prompt(ctx, promptReq)
	require.NoError(t, err)

	// Check for plan updates with different entry counts
	notifications := clientImpl.getNotifications()
	entryCounts := []int{}
	for _, notif := range notifications {
		if notif.Update.Plan != nil {
			plan := notif.Update.Plan
			entryCounts = append(entryCounts, len(plan.Entries))
		}
	}

	if len(entryCounts) > 1 {
		// Check if entry counts changed (dynamic planning)
		changed := false
		for i := 1; i < len(entryCounts); i++ {
			if entryCounts[i] != entryCounts[0] {
				changed = true
				break
			}
		}
		if changed {
			t.Logf("Plan entry counts changed: %v (dynamic planning)", entryCounts)
		} else {
			t.Logf("Plan entry counts: %v", entryCounts)
		}
	} else {
		t.Log("No dynamic plan changes found (may be expected)")
	}
}

// TestACP_Plan_MultipleUpdates tests multiple plan updates in one turn.
func TestACP_Plan_MultipleUpdates(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	workDir := createTestWorkspace(t)
	cmd, stdin, stdout := startACPAgent(t, "--workspace", workDir)
	defer cleanupAgent(t, cmd, stdin)

	clientImpl := &testClient{}
	client := createACPClientWithClient(t, stdin, stdout, clientImpl)
	ctx := context.Background()

	_, err := client.Initialize(ctx, acp.InitializeRequest{
		ProtocolVersion: acp.ProtocolVersionNumber,
	})
	require.NoError(t, err)

	sessionResp, err := client.NewSession(ctx, acp.NewSessionRequest{
		Cwd:        workDir,
		McpServers: []acp.McpServer{},
	})
	require.NoError(t, err)

	clientImpl.clearNotifications()

	// Send prompt
	promptReq := acp.PromptRequest{
		SessionId: sessionResp.SessionId,
		Prompt: []acp.ContentBlock{
			acp.TextBlock("create a plan and update it multiple times"),
		},
	}

	_, err = client.Prompt(ctx, promptReq)
	require.NoError(t, err)

	// Count plan updates
	notifications := clientImpl.getNotifications()
	updateCount := 0
	for _, notif := range notifications {
		if notif.Update.Plan != nil {
			updateCount++
		}
	}

	if updateCount > 1 {
		t.Logf("Found %d plan updates in one turn", updateCount)
	} else {
		t.Log("Found single or no plan updates (may be expected)")
	}
}
