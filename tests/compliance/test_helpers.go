// Package compliance provides compliance test utilities.
package compliance

import (
	"encoding/json"
	"slices"
	"testing"

	"github.com/coder/acp-go-sdk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// verifyJSONRPCRequest verifies that a request matches JSON-RPC 2.0 format.
func verifyJSONRPCRequest(t *testing.T, data []byte) {
	t.Helper()

	var req struct {
		JSONRPC string          `json:"jsonrpc"`
		Method  string          `json:"method"`
		Params  json.RawMessage `json:"params"`
		ID      any             `json:"id"`
	}

	err := json.Unmarshal(data, &req)
	require.NoError(t, err, "Request should be valid JSON")

	assert.Equal(t, "2.0", req.JSONRPC, "jsonrpc field should be '2.0'")
	assert.NotEmpty(t, req.Method, "method field should be present")
	assert.NotNil(t, req.ID, "id field should be present")
}

// verifyJSONRPCResponse verifies that a response matches JSON-RPC 2.0 format.
func verifyJSONRPCResponse(t *testing.T, data []byte) {
	t.Helper()

	var resp struct {
		JSONRPC string          `json:"jsonrpc"`
		Result  json.RawMessage `json:"result,omitempty"`
		Error   json.RawMessage `json:"error,omitempty"`
		ID      any             `json:"id"`
	}

	err := json.Unmarshal(data, &resp)
	require.NoError(t, err, "Response should be valid JSON")

	assert.Equal(t, "2.0", resp.JSONRPC, "jsonrpc field should be '2.0'")
	assert.NotNil(t, resp.ID, "id field should be present")
	// Either result or error should be present, but not both.
	if resp.Result != nil {
		assert.Nil(t, resp.Error, "Response should not have both result and error")
	} else {
		assert.NotNil(t, resp.Error, "Response should have either result or error")
	}
}

// verifyJSONRPCNotification verifies that a notification matches JSON-RPC 2.0 format.
func verifyJSONRPCNotification(t *testing.T, data []byte) {
	t.Helper()

	var notif struct {
		JSONRPC string          `json:"jsonrpc"`
		Method  string          `json:"method"`
		Params  json.RawMessage `json:"params"`
		ID      any             `json:"id,omitempty"`
	}

	err := json.Unmarshal(data, &notif)
	require.NoError(t, err, "Notification should be valid JSON")

	assert.Equal(t, "2.0", notif.JSONRPC, "jsonrpc field should be '2.0'")
	assert.NotEmpty(t, notif.Method, "method field should be present")
	assert.Nil(t, notif.ID, "Notification should not have id field")
}

// verifyJSONRPCError verifies that an error response matches JSON-RPC 2.0 format.
func verifyJSONRPCError(t *testing.T, data []byte, expectedCode int) {
	t.Helper()

	var resp struct {
		JSONRPC string `json:"jsonrpc"`
		Error   struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
			Data    any    `json:"data,omitempty"`
		} `json:"error"`
		ID any `json:"id"`
	}

	err := json.Unmarshal(data, &resp)
	require.NoError(t, err, "Error response should be valid JSON")

	assert.Equal(t, "2.0", resp.JSONRPC, "jsonrpc field should be '2.0'")
	assert.Equal(t, expectedCode, resp.Error.Code, "Error code should match")
	assert.NotEmpty(t, resp.Error.Message, "Error message should be present")
	assert.NotNil(t, resp.ID, "Error response should have id field")
}

// verifyInitializeResponse verifies Initialize response format compliance.
func verifyInitializeResponse(t *testing.T, resp acp.InitializeResponse) {
	t.Helper()

	// Verify protocol version.
	assert.Equal(t, acp.ProtocolVersion(1), resp.ProtocolVersion, "Protocol version should be 1")

	// Verify agent capabilities.
	require.NotNil(t, resp.AgentCapabilities, "Agent capabilities should be set")
	caps := resp.AgentCapabilities

	// Verify prompt capabilities.
	require.NotNil(t, caps.PromptCapabilities, "Prompt capabilities should be set")
	assert.True(t, caps.PromptCapabilities.Image, "Image capability should be advertised")
	assert.True(t, caps.PromptCapabilities.Audio, "Audio capability should be advertised")
	assert.True(t, caps.PromptCapabilities.EmbeddedContext, "Embedded context capability should be advertised")

	// Verify MCP capabilities.
	require.NotNil(t, caps.McpCapabilities, "MCP capabilities should be set")
	// Note: Stdio is always supported (required), but not a field in McpCapabilities
	// Http and Sse are optional fields.

	// Verify agent info.
	require.NotNil(t, resp.AgentInfo, "Agent info should be set")
	assert.Equal(t, "spin", resp.AgentInfo.Name, "Agent name should be 'spin'")
	assert.NotEmpty(t, resp.AgentInfo.Version, "Agent version should be set")
}

// verifyNewSessionResponse verifies NewSession response format compliance.
func verifyNewSessionResponse(t *testing.T, resp acp.NewSessionResponse) {
	t.Helper()

	// Verify session ID.
	assert.NotEmpty(t, resp.SessionId, "Session ID should be generated")

	// Verify mode state (if present).
	if resp.Modes != nil {
		assert.NotEmpty(t, resp.Modes.AvailableModes, "Available modes should be set")
		assert.NotEmpty(t, resp.Modes.CurrentModeId, "Current mode ID should be set")
	}
}

// verifyPromptResponse verifies Prompt response format compliance.
func verifyPromptResponse(t *testing.T, resp acp.PromptResponse) {
	t.Helper()

	// Verify stop reason is set.
	assert.NotNil(t, resp.StopReason, "Stop reason should be set")

	// Verify stop reason is valid.
	validReasons := []acp.StopReason{
		acp.StopReasonEndTurn,
		acp.StopReasonCancelled,
		acp.StopReasonRefusal,
		acp.StopReasonMaxTokens,
	}

	found := slices.Contains(validReasons, resp.StopReason)

	assert.True(t, found, "Stop reason should be a valid value: %v", resp.StopReason)
}

// verifyContentBlock verifies content block format compliance.
func verifyContentBlock(t *testing.T, block acp.ContentBlock) {
	t.Helper()

	// At least one content type should be set.
	hasContent := block.Text != nil ||
		block.Image != nil ||
		block.Audio != nil ||
		block.ResourceLink != nil ||
		block.Resource != nil

	assert.True(t, hasContent, "Content block should have at least one content type")

	// Verify text block format.
	if block.Text != nil {
		assert.NotEmpty(t, block.Text.Text, "Text block should have text content")
	}

	// Verify image block format.
	if block.Image != nil {
		assert.NotEmpty(t, block.Image.Data, "Image block should have data")
		assert.NotEmpty(t, block.Image.MimeType, "Image block should have MIME type")
	}

	// Verify audio block format.
	if block.Audio != nil {
		assert.NotEmpty(t, block.Audio.Data, "Audio block should have data")
		assert.NotEmpty(t, block.Audio.MimeType, "Audio block should have MIME type")
	}

	// Verify resource link format.
	if block.ResourceLink != nil {
		assert.NotEmpty(t, block.ResourceLink.Uri, "Resource link should have URI")
	}

	// Verify resource format.
	if block.Resource != nil {
		hasResourceContent := block.Resource.Resource.TextResourceContents != nil ||
			block.Resource.Resource.BlobResourceContents != nil
		assert.True(t, hasResourceContent, "Resource block should have content")
	}
}

// verifySessionNotification verifies session notification format compliance.
func verifySessionNotification(t *testing.T, notif acp.SessionNotification) {
	t.Helper()

	// Verify session ID.
	assert.NotEmpty(t, notif.SessionId, "Notification should have session ID")

	// Verify update is set.
	require.NotNil(t, notif.Update, "Notification should have update")

	// At least one update type should be set.
	hasUpdate := notif.Update.AgentMessageChunk != nil ||
		notif.Update.UserMessageChunk != nil ||
		notif.Update.ToolCall != nil ||
		notif.Update.ToolCallUpdate != nil ||
		notif.Update.Plan != nil ||
		notif.Update.AvailableCommandsUpdate != nil ||
		notif.Update.CurrentModeUpdate != nil ||
		notif.Update.AgentThoughtChunk != nil

	assert.True(t, hasUpdate, "Notification update should have at least one update type")
}

// verifyToolCall verifies tool call notification format compliance.
func verifyToolCall(t *testing.T, toolCall *acp.SessionUpdateToolCall) {
	t.Helper()

	require.NotNil(t, toolCall, "Tool call should not be nil")
	assert.NotEmpty(t, toolCall.ToolCallId, "Tool call should have ID")
	assert.NotEmpty(t, toolCall.Title, "Tool call should have title")
}

// verifyToolCallUpdate verifies tool call update notification format compliance.
func verifyToolCallUpdate(t *testing.T, update *acp.SessionToolCallUpdate) {
	t.Helper()

	require.NotNil(t, update, "Tool call update should not be nil")
	assert.NotEmpty(t, update.ToolCallId, "Tool call update should have ID")
	assert.NotNil(t, update.Status, "Tool call update should have status")
}
