package acp

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/coder/acp-go-sdk"
	"github.com/dmytrogajewski/spin/internal/agent"
	"github.com/dmytrogajewski/spin/internal/events"
	"github.com/dmytrogajewski/spin/internal/mcp"
	"github.com/dmytrogajewski/spin/internal/session"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestThinkingBlockTracker_IncrementalDeltas tests that only incremental deltas are sent.
func TestThinkingBlockTracker_IncrementalDeltas(t *testing.T) {
	tracker := newThinkingBlockTracker()

	// Simulate incremental content deltas
	deltas := []string{
		"Hello",
		"!",
		" How",
		" can",
		" I",
		" help",
		" you",
		" today",
		"?",
	}

	var allSentContent []string

	for _, delta := range deltas {
		_, messageUpdate, hasUpdate := tracker.processContent(delta)

		if hasUpdate && messageUpdate.AgentMessageChunk != nil {
			// Extract text content from the content block
			if messageUpdate.AgentMessageChunk.Content.Text != nil {
				sentText := messageUpdate.AgentMessageChunk.Content.Text.Text
				allSentContent = append(allSentContent, sentText)
			}
		}
	}

	// Verify that each sent chunk is incremental (not accumulated)
	// The first chunk should be "Hello", second should be "!", etc.
	require.Greater(t, len(allSentContent), 0, "Should have sent at least one chunk")

	// Verify chunks are incremental (each should be a small delta, not full accumulated content)
	for i, sent := range allSentContent {
		// Each chunk should be relatively small (incremental)
		// Not the full accumulated content like "HelloHello!Hello! How..."
		if i > 0 {
			// Chunks should not contain previous chunks (no duplication)
			for j := 0; j < i; j++ {
				assert.NotContains(t, sent, allSentContent[j], "Chunk %d should not contain previous chunk %d", i, j)
			}
		}
	}

	// Verify total sent content matches expected
	totalSent := ""
	for _, chunk := range allSentContent {
		totalSent += chunk
	}
	expectedTotal := "Hello! How can I help you today?"
	assert.Equal(t, expectedTotal, totalSent, "Total sent content should match expected")
}

// TestThinkingBlockTracker_NoDuplication tests that content is not duplicated.
func TestThinkingBlockTracker_NoDuplication(t *testing.T) {
	agentInstance := &agent.Agent{}
	mcpManager := mcp.NewMCPManager(&mcp.Config{EnableMCP: false}, slog.Default())
	emitter := events.NewEventEmitter(100)

	storage, err := session.NewFileStorage(t.TempDir())
	require.NoError(t, err)
	acpAgent, err := NewSpinACPAgentWithStorage(agentInstance, mcpManager, emitter, storage)
	require.NoError(t, err)

	// Use the existing mockConnection from notifications_integration_test.go
	mockConn := &mockConnection{}
	acpAgent.SetNotificationSender(mockConn)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sessionID := acp.SessionId("test-session")
	eventCh := make(chan events.Event, 10)

	// Start event processing
	go acpAgent.processEvents(ctx, sessionID, eventCh)

	// Send incremental content deltas
	deltas := []string{"Hello", "!", " How", " can", " I", " help"}
	for _, delta := range deltas {
		eventCh <- events.Event{
			Type:      events.EventContentDelta,
			Timestamp: time.Now(),
			Data: events.ContentDeltaData{
				Content: delta,
				Role:    "assistant",
			},
		}
		time.Sleep(10 * time.Millisecond) // Small delay to allow processing
	}

	close(eventCh)
	time.Sleep(100 * time.Millisecond) // Allow processing to complete

	// Collect all agent message chunks
	notifications := mockConn.GetNotifications()
	var messageChunks []string
	for _, notif := range notifications {
		if notif.Update.AgentMessageChunk != nil {
			if notif.Update.AgentMessageChunk.Content.Text != nil {
				messageChunks = append(messageChunks, notif.Update.AgentMessageChunk.Content.Text.Text)
			}
		}
	}

	// Verify no duplication - each chunk should be unique and incremental
	if len(messageChunks) > 0 {
		// Check that chunks don't duplicate previous content
		for i := 1; i < len(messageChunks); i++ {
			current := messageChunks[i]
			// Current chunk should not contain all previous chunks concatenated
			for j := 0; j < i; j++ {
				assert.NotEqual(t, current, messageChunks[j], "Chunk %d should not duplicate chunk %d", i, j)
			}
		}

		// Verify total content is correct (no duplication)
		totalContent := ""
		for _, chunk := range messageChunks {
			totalContent += chunk
		}
		// Should be "Hello! How can I help" (no duplication)
		assert.Equal(t, "Hello! How can I help", totalContent, "Total content should not be duplicated")
	}
}
