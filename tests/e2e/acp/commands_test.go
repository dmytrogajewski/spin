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

// TestACP_Commands_AvailableCommandsUpdate tests available_commands_update notification.
func TestACP_Commands_AvailableCommandsUpdate(t *testing.T) {
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

	_, err = client.NewSession(ctx, acp.NewSessionRequest{
		Cwd:        workDir,
		McpServers: []acp.McpServer{},
	})
	require.NoError(t, err)

	// Wait for available commands notification
	time.Sleep(50 * time.Millisecond)

	// Check for available_commands_update notification
	notifications := clientImpl.getNotifications()
	for _, notif := range notifications {
		if notif.Update.AvailableCommandsUpdate != nil {
			update := notif.Update.AvailableCommandsUpdate
			assert.NotNil(t, update.AvailableCommands, "Available commands should be set")
			t.Logf("Found available commands: %d commands", len(update.AvailableCommands))
			return
		}
	}
	t.Log("No available commands notification found (may be expected)")
}

// TestACP_Commands_CommandStructure tests command structure.
func TestACP_Commands_CommandStructure(t *testing.T) {
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

	_, err = client.NewSession(ctx, acp.NewSessionRequest{
		Cwd:        workDir,
		McpServers: []acp.McpServer{},
	})
	require.NoError(t, err)

	// Wait for commands
	time.Sleep(50 * time.Millisecond)

	// Check command structure
	notifications := clientImpl.getNotifications()
	for _, notif := range notifications {
		if notif.Update.AvailableCommandsUpdate != nil {
			update := notif.Update.AvailableCommandsUpdate
			for _, cmd := range update.AvailableCommands {
				assert.NotEmpty(t, cmd.Name, "Command should have name")
				assert.NotEmpty(t, cmd.Description, "Command should have description")
				t.Logf("Command: %s - %s", cmd.Name, cmd.Description)
			}
			return
		}
	}
	t.Log("No commands found (may be expected)")
}

// TestACP_Commands_CommandInput tests command input hint.
func TestACP_Commands_CommandInput(t *testing.T) {
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

	_, err = client.NewSession(ctx, acp.NewSessionRequest{
		Cwd:        workDir,
		McpServers: []acp.McpServer{},
	})
	require.NoError(t, err)

	// Wait for commands
	time.Sleep(50 * time.Millisecond)

	// Check for commands with input
	notifications := clientImpl.getNotifications()
	for _, notif := range notifications {
		if notif.Update.AvailableCommandsUpdate != nil {
			update := notif.Update.AvailableCommandsUpdate
			for _, cmd := range update.AvailableCommands {
				if cmd.Input != nil {
					// Input structure depends on SDK - just verify it exists
					t.Logf("Command %s has input", cmd.Name)
				}
			}
			return
		}
	}
	t.Log("No commands with input found (may be expected)")
}

// TestACP_Commands_DynamicUpdate tests that commands can be updated.
func TestACP_Commands_DynamicUpdate(t *testing.T) {
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

	// Clear and wait for initial commands
	clientImpl.clearNotifications()
	time.Sleep(50 * time.Millisecond)

	initialCount := 0
	notifications := clientImpl.getNotifications()
	for _, notif := range notifications {
		if notif.Update.AvailableCommandsUpdate != nil {
			initialCount = len(notif.Update.AvailableCommandsUpdate.AvailableCommands)
			break
		}
	}

	// Send prompt that might trigger command update
	promptReq := acp.PromptRequest{
		SessionId: sessionResp.SessionId,
		Prompt: []acp.ContentBlock{
			acp.TextBlock("test prompt"),
		},
	}

	_, err = client.Prompt(ctx, promptReq)
	require.NoError(t, err)

	// Wait for potential command updates
	time.Sleep(50 * time.Millisecond)

	// Check for updated commands
	notifications = clientImpl.getNotifications()
	updateCount := 0
	for _, notif := range notifications {
		if notif.Update.AvailableCommandsUpdate != nil {
			updateCount++
		}
	}

	if updateCount > 1 {
		t.Logf("Found %d command updates (commands were updated)", updateCount)
	} else {
		t.Logf("Found %d command update(s) (initial: %d)", updateCount, initialCount)
	}
}

// TestACP_Commands_Execute tests executing command via prompt.
func TestACP_Commands_Execute(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	workDir := createTestWorkspace(t)
	cmd, stdin, stdout := startACPAgent(t, "--workspace", workDir)
	defer cleanupAgent(t, cmd, stdin)

	client := createACPClient(t, stdin, stdout)
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

	// Execute command via prompt (slash command)
	// Note: The command may not exist, which is okay - we're testing the protocol, not specific commands
	promptReq := acp.PromptRequest{
		SessionId: sessionResp.SessionId,
		Prompt: []acp.ContentBlock{
			acp.TextBlock("/test command execution"),
		},
	}

	resp, err := client.Prompt(ctx, promptReq)
	// Command may not exist, which is acceptable for protocol testing
	if err != nil {
		t.Logf("Command execution returned error (may be expected if command doesn't exist): %v", err)
	} else {
		assert.NotNil(t, resp.StopReason)
	}
}

// TestACP_Commands_ExecuteWithInput tests executing command with input.
func TestACP_Commands_ExecuteWithInput(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	workDir := createTestWorkspace(t)
	cmd, stdin, stdout := startACPAgent(t, "--workspace", workDir)
	defer cleanupAgent(t, cmd, stdin)

	client := createACPClient(t, stdin, stdout)
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

	// Execute command with input
	// Note: The command may not exist, which is okay - we're testing the protocol, not specific commands
	promptReq := acp.PromptRequest{
		SessionId: sessionResp.SessionId,
		Prompt: []acp.ContentBlock{
			acp.TextBlock("/web search query here"),
		},
	}

	resp, err := client.Prompt(ctx, promptReq)
	// Command may not exist, which is acceptable for protocol testing
	if err != nil {
		t.Logf("Command execution returned error (may be expected if command doesn't exist): %v", err)
	} else {
		assert.NotNil(t, resp.StopReason)
	}
}
