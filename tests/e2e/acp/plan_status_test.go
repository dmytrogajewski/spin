//go:build e2e_llm_test

package acp

import (
	"context"
	"testing"
	"time"

	"github.com/coder/acp-go-sdk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestACP_Plan_Status_Updates tests that plan steps transition from pending to completed
// when corresponding tools are executed.
func TestACP_Plan_Status_Updates(t *testing.T) {
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
		ClientCapabilities: acp.ClientCapabilities{
			Terminal: true,
		},
	})
	require.NoError(t, err)

	sessionResp, err := client.NewSession(ctx, acp.NewSessionRequest{
		Cwd:        workDir,
		McpServers: []acp.McpServer{},
	})
	require.NoError(t, err)

	clientImpl.clearNotifications()

	// Send prompt that triggers the "execute plan test" scenario in test-llm
	// This will return a plan "Run echo command" and a tool call "shell_command"
	promptReq := acp.PromptRequest{
		SessionId: sessionResp.SessionId,
		Prompt: []acp.ContentBlock{
			acp.TextBlock("execute plan test"),
		},
	}

	_, err = client.Prompt(ctx, promptReq)
	require.NoError(t, err)

	// Wait for plan updates
	// We expect:
	// 1. Plan created (pending)
	// 2. Step running (when tool starts)
	// 3. Step completed (when tool finishes)

	// Wait for notifications to arrive
	require.Eventually(t, func() bool {
		notifications := clientImpl.getNotifications()
		for _, notif := range notifications {
			if notif.Update.Plan != nil {
				plan := notif.Update.Plan
				// Log plan for debugging
				// t.Logf("Received plan update: %+v", plan)
				for _, entry := range plan.Entries {
					if entry.Status == acp.PlanEntryStatusCompleted {
						return true
					}
				}
			}
		}
		return false
	}, 5*time.Second, 100*time.Millisecond, "Should receive plan update with completed step")

	// detailed verification
	notifications := clientImpl.getNotifications()
	var finalPlan *acp.SessionUpdatePlan

	for _, notif := range notifications {
		if notif.Update.Plan != nil {
			finalPlan = notif.Update.Plan
			t.Logf("Plan entries: %+v", finalPlan.Entries)
		}
	}

	require.NotNil(t, finalPlan, "Should have received plan update")
	require.NotEmpty(t, finalPlan.Entries, "Plan should have entries")

	entry := finalPlan.Entries[0]
	assert.Equal(t, "Run echo command", entry.Content)
	assert.Equal(t, acp.PlanEntryStatusCompleted, entry.Status)
}
