package acp

import (
	"context"
	"testing"
	"time"

	"github.com/coder/acp-go-sdk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestACP_Initialize tests the initialization flow.
func TestACP_Initialize(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	// Start ACP agent
	cmd, stdin, stdout := startACPAgent(t, "--provider", "ollama", "--model", "qwen3:0.6b")
	defer cleanupAgent(t, cmd, stdin)

	// Create ACP client
	client := createACPClient(t, stdin, stdout)

	// Test initialization
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	req := acp.InitializeRequest{
		ProtocolVersion: acp.ProtocolVersionNumber,
		ClientCapabilities: acp.ClientCapabilities{},
		ClientInfo: &acp.Implementation{
			Name:    "test-client",
			Version: "1.0.0",
		},
	}

	resp, err := client.Initialize(ctx, req)
	require.NoError(t, err, "Initialize should succeed")

	// Verify response
	assert.Equal(t, acp.ProtocolVersion(acp.ProtocolVersionNumber), resp.ProtocolVersion, "Protocol version should match")
	assert.NotNil(t, resp.AgentCapabilities, "Agent capabilities should be set")
	assert.NotNil(t, resp.AgentInfo, "Agent info should be set")
	assert.Equal(t, "spin", resp.AgentInfo.Name, "Agent name should be 'spin'")
	assert.NotEmpty(t, resp.AgentInfo.Version, "Agent version should be set")

	// Verify capabilities
	caps := resp.AgentCapabilities
	assert.True(t, caps.PromptCapabilities.Image, "Should support images")
	assert.True(t, caps.PromptCapabilities.Audio, "Should support audio")
	assert.True(t, caps.PromptCapabilities.EmbeddedContext, "Should support embedded context")
}

// TestACP_Initialize_ProtocolVersion tests protocol version negotiation.
func TestACP_Initialize_ProtocolVersion(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	cmd, stdin, stdout := startACPAgent(t, "--provider", "ollama", "--model", "qwen3:0.6b")
	defer cleanupAgent(t, cmd, stdin)

	client := createACPClient(t, stdin, stdout)

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	tests := []struct {
		name           string
		clientVersion  acp.ProtocolVersion
		expectedVersion acp.ProtocolVersion
	}{
		{
			name:           "version 1 supported",
			clientVersion:  acp.ProtocolVersion(1),
			expectedVersion: acp.ProtocolVersion(1),
		},
		{
			name:           "unsupported version returns version 1",
			clientVersion:  acp.ProtocolVersion(2),
			expectedVersion: acp.ProtocolVersion(1), // Agent returns latest supported
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := acp.InitializeRequest{
				ProtocolVersion:    tt.clientVersion,
				ClientCapabilities: acp.ClientCapabilities{},
			}

			resp, err := client.Initialize(ctx, req)
			require.NoError(t, err)
			assert.Equal(t, tt.expectedVersion, resp.ProtocolVersion)
		})
	}
}

// TestACP_Initialize_AgentCapabilities tests agent capability advertisement.
func TestACP_Initialize_AgentCapabilities(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	cmd, stdin, stdout := startACPAgent(t, "--provider", "ollama", "--model", "qwen3:0.6b")
	defer cleanupAgent(t, cmd, stdin)

	client := createACPClient(t, stdin, stdout)

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	req := acp.InitializeRequest{
		ProtocolVersion: acp.ProtocolVersionNumber,
	}

	resp, err := client.Initialize(ctx, req)
	require.NoError(t, err)

	// Verify capabilities are advertised correctly
	caps := resp.AgentCapabilities
	assert.True(t, caps.PromptCapabilities.Image, "Should advertise image support")
	assert.True(t, caps.PromptCapabilities.Audio, "Should advertise audio support")
	assert.True(t, caps.PromptCapabilities.EmbeddedContext, "Should advertise embedded context support")
	assert.NotNil(t, caps.McpCapabilities, "MCP capabilities should be set")
}

// TestACP_Initialize_ClientCapabilitiesStorage tests that client capabilities are stored.
func TestACP_Initialize_ClientCapabilitiesStorage(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	cmd, stdin, stdout := startACPAgent(t, "--provider", "ollama", "--model", "qwen3:0.6b")
	defer cleanupAgent(t, cmd, stdin)

	client := createACPClient(t, stdin, stdout)

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	clientCaps := acp.ClientCapabilities{
		// Client capabilities (can be extended in future)
	}

	req := acp.InitializeRequest{
		ProtocolVersion:    acp.ProtocolVersionNumber,
		ClientCapabilities: clientCaps,
	}

	_, err := client.Initialize(ctx, req)
	require.NoError(t, err)

	// Note: We can't directly verify storage, but if Initialize succeeds,
	// the agent has processed the client capabilities
}

// TestACP_Initialize_Timeout tests that initialization times out correctly.
func TestACP_Initialize_Timeout(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	cmd, stdin, stdout := startACPAgent(t, "--provider", "ollama", "--model", "qwen3:0.6b")
	defer cleanupAgent(t, cmd, stdin)

	client := createACPClient(t, stdin, stdout)

	// Use a very short timeout to test timeout handling
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()

	// Wait a bit to ensure context is already expired
	time.Sleep(10 * time.Millisecond)

	req := acp.InitializeRequest{
		ProtocolVersion: acp.ProtocolVersionNumber,
	}

	_, err := client.Initialize(ctx, req)
	// Should fail due to timeout
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "context deadline exceeded")
}

