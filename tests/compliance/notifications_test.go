package compliance

import (
	"testing"

	"github.com/coder/acp-go-sdk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCompliance_Notification_AgentMessageChunk verifies agent message chunk notification format.
func TestCompliance_Notification_AgentMessageChunk(t *testing.T) {
	t.Parallel()

	notif := acp.SessionNotification{
		SessionId: acp.SessionId("test-session"),
		Update: acp.SessionUpdate{
			AgentMessageChunk: &acp.SessionUpdateAgentMessageChunk{
				Content: acp.TextBlock("Hello"),
			},
		},
	}

	verifySessionNotification(t, notif)
	assert.NotNil(t, notif.Update.AgentMessageChunk, "Agent message chunk should be set")
	verifyContentBlock(t, notif.Update.AgentMessageChunk.Content)
}

// TestCompliance_Notification_ToolCall verifies tool call notification format.
func TestCompliance_Notification_ToolCall(t *testing.T) {
	t.Parallel()

	notif := acp.SessionNotification{
		SessionId: acp.SessionId("test-session"),
		Update: acp.SessionUpdate{
			ToolCall: &acp.SessionUpdateToolCall{
				ToolCallId: acp.ToolCallId("tool-123"),
				Title:      "read_file",
			},
		},
	}

	verifySessionNotification(t, notif)
	verifyToolCall(t, notif.Update.ToolCall)
}

// TestCompliance_Notification_ToolCallUpdate verifies tool call update notification format.
func TestCompliance_Notification_ToolCallUpdate(t *testing.T) {
	t.Parallel()

	status := acp.ToolCallStatusCompleted
	notif := acp.SessionNotification{
		SessionId: acp.SessionId("test-session"),
		Update: acp.SessionUpdate{
			ToolCallUpdate: &acp.SessionToolCallUpdate{
				ToolCallId: acp.ToolCallId("tool-123"),
				Status:     &status,
			},
		},
	}

	verifySessionNotification(t, notif)
	verifyToolCallUpdate(t, notif.Update.ToolCallUpdate)
}

// TestCompliance_Notification_Plan verifies plan notification format.
func TestCompliance_Notification_Plan(t *testing.T) {
	t.Parallel()

	notif := acp.SessionNotification{
		SessionId: acp.SessionId("test-session"),
		Update: acp.SessionUpdate{
			Plan: &acp.SessionUpdatePlan{
				Entries: []acp.PlanEntry{
					{
						Content:  "Step 1",
						Status:   acp.PlanEntryStatus("pending"),
						Priority: acp.PlanEntryPriority("normal"),
					},
				},
			},
		},
	}

	verifySessionNotification(t, notif)
	assert.NotNil(t, notif.Update.Plan, "Plan should be set")
	assert.NotEmpty(t, notif.Update.Plan.Entries, "Plan entries should not be empty")
}

// TestCompliance_Notification_Commands verifies available commands update notification format.
func TestCompliance_Notification_Commands(t *testing.T) {
	t.Parallel()

	notif := acp.SessionNotification{
		SessionId: acp.SessionId("test-session"),
		Update: acp.SessionUpdate{
			AvailableCommandsUpdate: &acp.SessionAvailableCommandsUpdate{
				AvailableCommands: []acp.AvailableCommand{
					{
						Name:        "mode",
						Description: "Set session mode",
					},
				},
			},
		},
	}

	verifySessionNotification(t, notif)
	assert.NotNil(t, notif.Update.AvailableCommandsUpdate, "Available commands update should be set")
	assert.NotEmpty(t, notif.Update.AvailableCommandsUpdate.AvailableCommands, "Commands should not be empty")
}

// TestCompliance_Notification_ContentBlocks verifies notification content block format.
func TestCompliance_Notification_ContentBlocks(t *testing.T) {
	t.Parallel()

	notif := acp.SessionNotification{
		SessionId: acp.SessionId("test-session"),
		Update: acp.SessionUpdate{
			AgentMessageChunk: &acp.SessionUpdateAgentMessageChunk{
				Content: acp.TextBlock("Hello"),
			},
		},
	}

	verifySessionNotification(t, notif)
	require.NotNil(t, notif.Update.AgentMessageChunk, "Agent message chunk should be set")

	// Verify content block.
	verifyContentBlock(t, notif.Update.AgentMessageChunk.Content)
}

// TestCompliance_Notification_Streaming verifies streaming notification support.
func TestCompliance_Notification_Streaming(t *testing.T) {
	t.Parallel()

	// Test incremental updates.
	notif1 := acp.SessionNotification{
		SessionId: acp.SessionId("test-session"),
		Update: acp.SessionUpdate{
			AgentMessageChunk: &acp.SessionUpdateAgentMessageChunk{
				Content: acp.TextBlock("Hello"),
			},
		},
	}

	notif2 := acp.SessionNotification{
		SessionId: acp.SessionId("test-session"),
		Update: acp.SessionUpdate{
			AgentMessageChunk: &acp.SessionUpdateAgentMessageChunk{
				Content: acp.TextBlock(" world"),
			},
		},
	}

	// Both notifications should be valid.
	verifySessionNotification(t, notif1)
	verifySessionNotification(t, notif2)
}
