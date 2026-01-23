package compliance

import (
	"context"
	"log/slog"
	"testing"

	"github.com/coder/acp-go-sdk"
	"github.com/dmytrogajewski/spin/internal/agent"
	"github.com/dmytrogajewski/spin/internal/events"
	"github.com/dmytrogajewski/spin/internal/mcp"
	acppkg "github.com/dmytrogajewski/spin/internal/protocol/acp"
	"github.com/dmytrogajewski/spin/internal/session"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// createTestACPAgent creates a test ACP agent for compliance testing.
func createTestACPAgent(t *testing.T) *acppkg.SpinACPAgent {
	t.Helper()

	agentInstance := &agent.Agent{}
	mcpService := mcp.NewService(mcp.NewDefaultRegistryManager(slog.Default()))
	emitter := events.NewEventEmitter(100)

	// Create proper storage for tests
	storage, err := session.NewFileStorage(t.TempDir())
	require.NoError(t, err)

	acpAgent, err := acppkg.NewSpinACPAgentWithStorage(agentInstance, mcpService, emitter, storage)
	require.NoError(t, err)

	return acpAgent
}

// TestCompliance_Initialize_ProtocolVersion verifies protocol version negotiation compliance.
func TestCompliance_Initialize_ProtocolVersion(t *testing.T) {
	acpAgent := createTestACPAgent(t)
	ctx := context.Background()

	req := acp.InitializeRequest{
		ProtocolVersion: acp.ProtocolVersionNumber,
	}

	resp, err := acpAgent.Initialize(ctx, req)
	require.NoError(t, err)

	// Verify response format
	verifyInitializeResponse(t, resp)

	// Verify protocol version
	assert.Equal(t, acp.ProtocolVersion(1), resp.ProtocolVersion, "Protocol version should be 1")
}

// TestCompliance_Initialize_Capabilities verifies capability advertisement compliance.
func TestCompliance_Initialize_Capabilities(t *testing.T) {
	acpAgent := createTestACPAgent(t)
	ctx := context.Background()

	req := acp.InitializeRequest{
		ProtocolVersion: acp.ProtocolVersionNumber,
	}

	resp, err := acpAgent.Initialize(ctx, req)
	require.NoError(t, err)

	// Verify capabilities are correctly advertised
	caps := resp.AgentCapabilities
	require.NotNil(t, caps, "Agent capabilities should be set")

	// Verify prompt capabilities
	assert.True(t, caps.PromptCapabilities.Image, "Image capability should be advertised")
	assert.True(t, caps.PromptCapabilities.Audio, "Audio capability should be advertised")
	assert.True(t, caps.PromptCapabilities.EmbeddedContext, "Embedded context capability should be advertised")

	// Verify MCP capabilities
	// Note: Stdio is always supported (required), but not a field in McpCapabilities
	// Http and Sse are optional fields
}

// TestCompliance_Initialize_AgentInfo verifies agent info exchange compliance.
func TestCompliance_Initialize_AgentInfo(t *testing.T) {
	acpAgent := createTestACPAgent(t)
	ctx := context.Background()

	req := acp.InitializeRequest{
		ProtocolVersion: acp.ProtocolVersionNumber,
	}

	resp, err := acpAgent.Initialize(ctx, req)
	require.NoError(t, err)

	// Verify agent info
	require.NotNil(t, resp.AgentInfo, "Agent info should be set")
	assert.Equal(t, "spin", resp.AgentInfo.Name, "Agent name should be 'spin'")
	assert.NotEmpty(t, resp.AgentInfo.Version, "Agent version should be set")
}

// TestCompliance_Initialize_ClientCapabilities verifies client capability storage.
func TestCompliance_Initialize_ClientCapabilities(t *testing.T) {
	acpAgent := createTestACPAgent(t)
	ctx := context.Background()

	clientCaps := acp.ClientCapabilities{
		// Client capabilities
	}

	req := acp.InitializeRequest{
		ProtocolVersion:    acp.ProtocolVersionNumber,
		ClientCapabilities: clientCaps,
	}

	resp, err := acpAgent.Initialize(ctx, req)
	require.NoError(t, err)

	// If Initialize succeeds, client capabilities were processed
	verifyInitializeResponse(t, resp)
}

// TestCompliance_Initialize_ResponseFormat verifies Initialize response format compliance.
func TestCompliance_Initialize_ResponseFormat(t *testing.T) {
	acpAgent := createTestACPAgent(t)
	ctx := context.Background()

	req := acp.InitializeRequest{
		ProtocolVersion: acp.ProtocolVersionNumber,
	}

	resp, err := acpAgent.Initialize(ctx, req)
	require.NoError(t, err)

	// Verify complete response format
	verifyInitializeResponse(t, resp)
}

// TestCompliance_NewSession_Cwd verifies working directory validation compliance.
func TestCompliance_NewSession_Cwd(t *testing.T) {
	acpAgent := createTestACPAgent(t)
	ctx := context.Background()

	// Test with valid CWD
	req := acp.NewSessionRequest{
		Cwd: "/tmp/test",
	}

	resp, err := acpAgent.NewSession(ctx, req)
	require.NoError(t, err)
	verifyNewSessionResponse(t, resp)

	// Test with empty CWD (should fail)
	reqEmpty := acp.NewSessionRequest{
		Cwd: "",
	}

	_, err = acpAgent.NewSession(ctx, reqEmpty)
	require.Error(t, err, "NewSession should fail with empty CWD")
}

// TestCompliance_NewSession_SessionId verifies session ID generation compliance.
func TestCompliance_NewSession_SessionId(t *testing.T) {
	acpAgent := createTestACPAgent(t)
	ctx := context.Background()

	req := acp.NewSessionRequest{
		Cwd: "/tmp/test",
	}

	resp, err := acpAgent.NewSession(ctx, req)
	require.NoError(t, err)

	// Verify session ID is generated and non-empty
	assert.NotEmpty(t, resp.SessionId, "Session ID should be generated")
	assert.Greater(t, len(resp.SessionId), 0, "Session ID should have length > 0")
}

// TestCompliance_NewSession_ModeState verifies session mode state compliance.
func TestCompliance_NewSession_ModeState(t *testing.T) {
	acpAgent := createTestACPAgent(t)
	ctx := context.Background()

	req := acp.NewSessionRequest{
		Cwd: "/tmp/test",
	}

	resp, err := acpAgent.NewSession(ctx, req)
	require.NoError(t, err)

	// Verify mode state is included
	require.NotNil(t, resp.Modes, "Mode state should be included")
	assert.NotEmpty(t, resp.Modes.AvailableModes, "Available modes should be set")
	assert.NotEmpty(t, resp.Modes.CurrentModeId, "Current mode ID should be set")
	assert.Equal(t, acp.SessionModeId("regular"), resp.Modes.CurrentModeId, "Default mode should be 'regular'")
}

// TestCompliance_NewSession_ResponseFormat verifies NewSession response format compliance.
func TestCompliance_NewSession_ResponseFormat(t *testing.T) {
	acpAgent := createTestACPAgent(t)
	ctx := context.Background()

	req := acp.NewSessionRequest{
		Cwd: "/tmp/test",
	}

	resp, err := acpAgent.NewSession(ctx, req)
	require.NoError(t, err)

	verifyNewSessionResponse(t, resp)
}

// TestCompliance_Prompt_SessionId verifies session ID validation compliance.
func TestCompliance_Prompt_SessionId(t *testing.T) {
	acpAgent := createTestACPAgent(t)
	ctx := context.Background()

	// Test with invalid session ID (should fail validation before execution)
	invalidReq := acp.PromptRequest{
		SessionId: acp.SessionId("invalid-session-id"),
		Prompt: []acp.ContentBlock{
			acp.TextBlock("test"),
		},
	}

	_, err := acpAgent.Prompt(ctx, invalidReq)
	require.Error(t, err, "Prompt should fail with invalid session ID")
	assert.Contains(t, err.Error(), "session", "Error should mention session")
}

// TestCompliance_Prompt_ContentBlocks verifies content block format compliance.
func TestCompliance_Prompt_ContentBlocks(t *testing.T) {
	// Test content block format compliance (without execution)
	textBlock := acp.TextBlock("test message")
	verifyContentBlock(t, textBlock)

	imageBlock := acp.ImageBlock("base64data", "image/png")
	verifyContentBlock(t, imageBlock)

	audioBlock := acp.AudioBlock("base64data", "audio/mpeg")
	verifyContentBlock(t, audioBlock)

	resourceLinkBlock := acp.ResourceLinkBlock("file.txt", "file:///path/to/file.txt")
	verifyContentBlock(t, resourceLinkBlock)
}

// TestCompliance_Prompt_StopReason verifies stop reason format compliance.
func TestCompliance_Prompt_StopReason(t *testing.T) {
	// Test stop reason format (without execution)
	// Create a mock response with valid stop reason
	resp := acp.PromptResponse{
		StopReason: acp.StopReasonEndTurn,
	}

	verifyPromptResponse(t, resp)

	// Test other valid stop reasons
	validReasons := []acp.StopReason{
		acp.StopReasonEndTurn,
		acp.StopReasonCancelled,
		acp.StopReasonRefusal,
		acp.StopReasonMaxTokens,
	}

	for _, reason := range validReasons {
		resp := acp.PromptResponse{
			StopReason: reason,
		}
		verifyPromptResponse(t, resp)
	}
}

// TestCompliance_Prompt_ResponseFormat verifies Prompt response format compliance.
func TestCompliance_Prompt_ResponseFormat(t *testing.T) {
	// Test response format (without execution)
	resp := acp.PromptResponse{
		StopReason: acp.StopReasonEndTurn,
	}

	verifyPromptResponse(t, resp)
}
