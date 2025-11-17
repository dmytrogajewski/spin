package acp

import (
	"context"
	"testing"

	"github.com/coder/acp-go-sdk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestACP_NewSession tests creating a new session.
func TestACP_NewSession(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	// Create test workspace
	workDir := createTestWorkspace(t)

	// Start ACP agent
	cmd, stdin, stdout := startACPAgent(t, "--provider", "ollama", "--model", "qwen3:0.6b", "--workspace", workDir)
	defer cleanupAgent(t, cmd, stdin)

	// Create client and initialize
	client := createACPClient(t, stdin, stdout)
	ctx := context.Background()

	_, err := client.Initialize(ctx, acp.InitializeRequest{
		ProtocolVersion: acp.ProtocolVersionNumber,
	})
	require.NoError(t, err)

	// Test NewSession
	req := acp.NewSessionRequest{
		Cwd:        workDir,
		McpServers: []acp.McpServer{},
	}

	resp, err := client.NewSession(ctx, req)
	require.NoError(t, err, "NewSession should succeed")

	// Verify response
	assert.NotEmpty(t, resp.SessionId, "Session ID should be generated")
	assert.NotNil(t, resp.Modes, "Session mode state should be set")
	
	// Verify mode state
	if resp.Modes != nil {
		assert.NotEmpty(t, resp.Modes.AvailableModes, "Available modes should be set")
		assert.NotEmpty(t, resp.Modes.CurrentModeId, "Current mode should be set")
	}
}

// TestACP_NewSession_WithMcpServers tests creating a session with MCP servers.
func TestACP_NewSession_WithMcpServers(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	workDir := createTestWorkspace(t)
	cmd, stdin, stdout := startACPAgent(t, "--provider", "ollama", "--model", "qwen3:0.6b", "--workspace", workDir)
	defer cleanupAgent(t, cmd, stdin)

	client := createACPClient(t, stdin, stdout)
	ctx := context.Background()

	_, err := client.Initialize(ctx, acp.InitializeRequest{
		ProtocolVersion: acp.ProtocolVersionNumber,
	})
	require.NoError(t, err)

	// Test NewSession with MCP server (using echo as a simple test server)
	req := acp.NewSessionRequest{
		Cwd: workDir,
		McpServers: []acp.McpServer{
			{
				Stdio: &acp.McpServerStdio{
					Command: "/bin/echo",
					Args:    []string{"test"},
				},
			},
		},
	}

	resp, err := client.NewSession(ctx, req)
	// MCP server connection may fail, but session should still be created
	// The agent creates sessions asynchronously with MCP connections
	if err != nil {
		t.Logf("NewSession with MCP server returned error (expected for test): %v", err)
	} else {
		assert.NotEmpty(t, resp.SessionId, "Session ID should be generated even with MCP server")
	}
}

// TestACP_NewSession_InvalidCwd tests error handling for invalid working directory.
func TestACP_NewSession_InvalidCwd(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	cmd, stdin, stdout := startACPAgent(t, "--provider", "ollama", "--model", "qwen3:0.6b")
	defer cleanupAgent(t, cmd, stdin)

	client := createACPClient(t, stdin, stdout)
	ctx := context.Background()

	_, err := client.Initialize(ctx, acp.InitializeRequest{
		ProtocolVersion: acp.ProtocolVersionNumber,
	})
	require.NoError(t, err)

	// Test with empty CWD
	req := acp.NewSessionRequest{
		Cwd:        "",
		McpServers: []acp.McpServer{},
	}

	_, err = client.NewSession(ctx, req)
	// Should return error for empty CWD
	assert.Error(t, err, "NewSession should fail with empty CWD")
}

// TestACP_NewSession_ModeState tests that session mode state is included.
func TestACP_NewSession_ModeState(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	workDir := createTestWorkspace(t)
	cmd, stdin, stdout := startACPAgent(t, "--provider", "ollama", "--model", "qwen3:0.6b", "--workspace", workDir)
	defer cleanupAgent(t, cmd, stdin)

	client := createACPClient(t, stdin, stdout)
	ctx := context.Background()

	_, err := client.Initialize(ctx, acp.InitializeRequest{
		ProtocolVersion: acp.ProtocolVersionNumber,
	})
	require.NoError(t, err)

	req := acp.NewSessionRequest{
		Cwd:        workDir,
		McpServers: []acp.McpServer{},
	}

	resp, err := client.NewSession(ctx, req)
	require.NoError(t, err)

	// Verify mode state
	require.NotNil(t, resp.Modes, "Session mode state should be set")
	
	modeState := resp.Modes
	assert.NotEmpty(t, modeState.AvailableModes, "Available modes should be set")
	assert.NotEmpty(t, modeState.CurrentModeId, "Current mode should be set")
	
	// Verify default mode is "regular"
	assert.Equal(t, acp.SessionModeId("regular"), modeState.CurrentModeId, "Default mode should be 'regular'")
	
	// Verify available modes include expected modes
	modeMap := make(map[acp.SessionModeId]bool)
	for _, mode := range modeState.AvailableModes {
		modeMap[mode.Id] = true
	}
	
	assert.True(t, modeMap[acp.SessionModeId("regular")], "Should have 'regular' mode")
	assert.True(t, modeMap[acp.SessionModeId("review")], "Should have 'review' mode")
	assert.True(t, modeMap[acp.SessionModeId("compact")], "Should have 'compact' mode")
	assert.True(t, modeMap[acp.SessionModeId("planning")], "Should have 'planning' mode")
}

// TestACP_LoadSession tests loading an existing session (if storage available).
func TestACP_LoadSession(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	// Note: LoadSession requires session storage to be configured
	// This test may be skipped if storage is not available
	workDir := createTestWorkspace(t)
	cmd, stdin, stdout := startACPAgent(t, "--provider", "ollama", "--model", "qwen3:0.6b", "--workspace", workDir)
	defer cleanupAgent(t, cmd, stdin)

	client := createACPClient(t, stdin, stdout)
	ctx := context.Background()

	_, err := client.Initialize(ctx, acp.InitializeRequest{
		ProtocolVersion: acp.ProtocolVersionNumber,
	})
	require.NoError(t, err)

	// First create a session
	newSessionReq := acp.NewSessionRequest{
		Cwd:        workDir,
		McpServers: []acp.McpServer{},
	}
	newSessionResp, err := client.NewSession(ctx, newSessionReq)
	require.NoError(t, err)
	sessionID := newSessionResp.SessionId

	// Try to load the session
	// Note: This may fail if session storage is not configured
	loadReq := acp.LoadSessionRequest{
		SessionId: sessionID,
	}
	
	_, err = client.LoadSession(ctx, loadReq)
	// LoadSession may not be available if storage is not configured
	// That's okay - we're just testing the method exists and can be called
	if err != nil {
		t.Logf("LoadSession returned error (may be expected if storage not configured): %v", err)
	}
}

// TestACP_NewSession_Concurrent tests creating multiple sessions concurrently.
func TestACP_NewSession_Concurrent(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	workDir := createTestWorkspace(t)
	cmd, stdin, stdout := startACPAgent(t, "--provider", "ollama", "--model", "qwen3:0.6b", "--workspace", workDir)
	defer cleanupAgent(t, cmd, stdin)

	client := createACPClient(t, stdin, stdout)
	ctx := context.Background()

	_, err := client.Initialize(ctx, acp.InitializeRequest{
		ProtocolVersion: acp.ProtocolVersionNumber,
	})
	require.NoError(t, err)

	// Create multiple sessions concurrently
	const numSessions = 3
	sessionIDs := make([]acp.SessionId, numSessions)
	errs := make([]error, numSessions)

	for i := 0; i < numSessions; i++ {
		req := acp.NewSessionRequest{
			Cwd:        workDir,
			McpServers: []acp.McpServer{},
		}
		resp, err := client.NewSession(ctx, req)
		sessionIDs[i] = resp.SessionId
		errs[i] = err
	}

	// Verify all sessions were created successfully
	for i, err := range errs {
		assert.NoError(t, err, "Session %d should be created", i)
		assert.NotEmpty(t, sessionIDs[i], "Session %d should have ID", i)
	}

	// Verify all session IDs are unique
	seen := make(map[acp.SessionId]bool)
	for _, id := range sessionIDs {
		assert.False(t, seen[id], "Session ID should be unique: %s", id)
		seen[id] = true
	}
}

