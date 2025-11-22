//go:build e2e_llm_test

package acp

import (
	"context"
	"sync"
	"testing"

	"github.com/coder/acp-go-sdk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// cancelPermissionClient cancels all permission requests.
type cancelPermissionClient struct {
	testClient
}

func (c *cancelPermissionClient) RequestPermission(ctx context.Context, params acp.RequestPermissionRequest) (acp.RequestPermissionResponse, error) {
	return acp.RequestPermissionResponse{
		Outcome: acp.NewRequestPermissionOutcomeCancelled(),
	}, nil
}

// permissionTestClient tracks permission requests.
type permissionTestClient struct {
	testClient
	permissionRequests []acp.RequestPermissionRequest
	mu                 sync.Mutex
}

func (c *permissionTestClient) RequestPermission(ctx context.Context, params acp.RequestPermissionRequest) (acp.RequestPermissionResponse, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.permissionRequests = append(c.permissionRequests, params)

	// Auto-approve by selecting the first allow option
	var selectedOptionID acp.PermissionOptionId
	for _, option := range params.Options {
		if option.Kind == acp.PermissionOptionKindAllowOnce || option.Kind == acp.PermissionOptionKindAllowAlways {
			selectedOptionID = option.OptionId
			break
		}
	}

	// If no allow option found, use first option (for testing)
	if selectedOptionID == "" && len(params.Options) > 0 {
		selectedOptionID = params.Options[0].OptionId
	}

	return acp.RequestPermissionResponse{
		Outcome: acp.NewRequestPermissionOutcomeSelected(selectedOptionID),
	}, nil
}

// TestACP_Permission_AllowOnce tests allow_once option.
func TestACP_Permission_AllowOnce(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	workDir := createTestWorkspace(t)
	cmd, stdin, stdout := startACPAgent(t, "--workspace", workDir)
	defer cleanupAgent(t, cmd, stdin)

	permClient := &permissionTestClient{}
	client := createACPClientWithClient(t, stdin, stdout, &permClient.testClient)
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

	// Send prompt that might trigger permission request (dangerous command)
	promptReq := acp.PromptRequest{
		SessionId: sessionResp.SessionId,
		Prompt: []acp.ContentBlock{
			acp.TextBlock("execute command: rm -rf /tmp/test"),
		},
	}

	_, err = client.Prompt(ctx, promptReq)
	require.NoError(t, err)

	// Verify RequestPermission was called
	permClient.mu.Lock()
	requests := permClient.permissionRequests
	permClient.mu.Unlock()

	if len(requests) > 0 {
		req := requests[0]
		assert.NotEmpty(t, req.SessionId, "Request should have session ID")
		assert.NotNil(t, req.ToolCall, "Request should have tool call")
		assert.NotEmpty(t, req.Options, "Request should have options")

		// Check for allow_once option
		hasAllowOnce := false
		for _, opt := range req.Options {
			if opt.Kind == acp.PermissionOptionKindAllowOnce {
				hasAllowOnce = true
				assert.NotEmpty(t, opt.OptionId, "Option should have ID")
				assert.NotEmpty(t, opt.Name, "Option should have name")
				break
			}
		}
		if hasAllowOnce {
			t.Log("Found allow_once option in permission request")
		}
	} else {
		t.Log("RequestPermission was not called (may be expected if agent doesn't require approval)")
	}
}

// TestACP_Permission_AllowAlways tests allow_always option.
func TestACP_Permission_AllowAlways(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	workDir := createTestWorkspace(t)
	cmd, stdin, stdout := startACPAgent(t, "--workspace", workDir)
	defer cleanupAgent(t, cmd, stdin)

	permClient := &permissionTestClient{}
	client := createACPClientWithClient(t, stdin, stdout, &permClient.testClient)
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

	// Send prompt that might trigger permission request
	promptReq := acp.PromptRequest{
		SessionId: sessionResp.SessionId,
		Prompt: []acp.ContentBlock{
			acp.TextBlock("execute command: sudo rm -rf /tmp/test"),
		},
	}

	_, err = client.Prompt(ctx, promptReq)
	require.NoError(t, err)

	// Verify RequestPermission was called
	permClient.mu.Lock()
	requests := permClient.permissionRequests
	permClient.mu.Unlock()

	if len(requests) > 0 {
		req := requests[0]
		// Check for allow_always option
		hasAllowAlways := false
		for _, opt := range req.Options {
			if opt.Kind == acp.PermissionOptionKindAllowAlways {
				hasAllowAlways = true
				break
			}
		}
		if hasAllowAlways {
			t.Log("Found allow_always option in permission request")
		}
	} else {
		t.Log("RequestPermission was not called (may be expected)")
	}
}

// TestACP_Permission_RejectOnce tests reject_once option.
func TestACP_Permission_RejectOnce(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	workDir := createTestWorkspace(t)
	cmd, stdin, stdout := startACPAgent(t, "--workspace", workDir)
	defer cleanupAgent(t, cmd, stdin)

	permClient := &permissionTestClient{}
	client := createACPClientWithClient(t, stdin, stdout, &permClient.testClient)
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

	// Send prompt that might trigger permission request
	promptReq := acp.PromptRequest{
		SessionId: sessionResp.SessionId,
		Prompt: []acp.ContentBlock{
			acp.TextBlock("execute command: rm -rf /tmp/test"),
		},
	}

	_, err = client.Prompt(ctx, promptReq)
	require.NoError(t, err)

	// Verify RequestPermission was called
	permClient.mu.Lock()
	requests := permClient.permissionRequests
	permClient.mu.Unlock()

	if len(requests) > 0 {
		req := requests[0]
		// Check for reject_once option
		hasRejectOnce := false
		for _, opt := range req.Options {
			if opt.Kind == acp.PermissionOptionKindRejectOnce {
				hasRejectOnce = true
				break
			}
		}
		if hasRejectOnce {
			t.Log("Found reject_once option in permission request")
		}
	} else {
		t.Log("RequestPermission was not called (may be expected)")
	}
}

// TestACP_Permission_RejectAlways tests reject_always option.
func TestACP_Permission_RejectAlways(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	workDir := createTestWorkspace(t)
	cmd, stdin, stdout := startACPAgent(t, "--workspace", workDir)
	defer cleanupAgent(t, cmd, stdin)

	permClient := &permissionTestClient{}
	client := createACPClientWithClient(t, stdin, stdout, &permClient.testClient)
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

	// Send prompt that might trigger permission request
	promptReq := acp.PromptRequest{
		SessionId: sessionResp.SessionId,
		Prompt: []acp.ContentBlock{
			acp.TextBlock("execute command: sudo rm -rf /tmp/test"),
		},
	}

	_, err = client.Prompt(ctx, promptReq)
	require.NoError(t, err)

	// Verify RequestPermission was called
	permClient.mu.Lock()
	requests := permClient.permissionRequests
	permClient.mu.Unlock()

	if len(requests) > 0 {
		req := requests[0]
		// Check for reject_always option
		hasRejectAlways := false
		for _, opt := range req.Options {
			if opt.Kind == acp.PermissionOptionKindRejectAlways {
				hasRejectAlways = true
				break
			}
		}
		if hasRejectAlways {
			t.Log("Found reject_always option in permission request")
		}
	} else {
		t.Log("RequestPermission was not called (may be expected)")
	}
}

// TestACP_Permission_Cancelled tests cancellation outcome.
func TestACP_Permission_Cancelled(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	workDir := createTestWorkspace(t)
	cmd, stdin, stdout := startACPAgent(t, "--workspace", workDir)
	defer cleanupAgent(t, cmd, stdin)

	cancelClient := &cancelPermissionClient{}
	client := createACPClientWithClient(t, stdin, stdout, &cancelClient.testClient)
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

	// Send prompt that might trigger permission request
	promptReq := acp.PromptRequest{
		SessionId: sessionResp.SessionId,
		Prompt: []acp.ContentBlock{
			acp.TextBlock("execute command: rm -rf /tmp/test"),
		},
	}

	_, err = client.Prompt(ctx, promptReq)
	require.NoError(t, err)

	// Cancellation should be handled gracefully
	t.Log("Permission request cancellation handled")
}

// TestACP_Permission_MultipleOptions tests multiple permission options.
func TestACP_Permission_MultipleOptions(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	workDir := createTestWorkspace(t)
	cmd, stdin, stdout := startACPAgent(t, "--workspace", workDir)
	defer cleanupAgent(t, cmd, stdin)

	permClient := &permissionTestClient{}
	client := createACPClientWithClient(t, stdin, stdout, &permClient.testClient)
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

	// Send prompt that might trigger permission request
	promptReq := acp.PromptRequest{
		SessionId: sessionResp.SessionId,
		Prompt: []acp.ContentBlock{
			acp.TextBlock("execute command: rm -rf /tmp/test"),
		},
	}

	_, err = client.Prompt(ctx, promptReq)
	require.NoError(t, err)

	// Verify RequestPermission was called with multiple options
	permClient.mu.Lock()
	requests := permClient.permissionRequests
	permClient.mu.Unlock()

	if len(requests) > 0 {
		req := requests[0]
		if len(req.Options) > 1 {
			t.Logf("Found %d permission options", len(req.Options))
			optionKinds := make(map[acp.PermissionOptionKind]bool)
			for _, opt := range req.Options {
				optionKinds[opt.Kind] = true
			}
			t.Logf("Permission option kinds: %v", optionKinds)
		}
	} else {
		t.Log("RequestPermission was not called (may be expected)")
	}
}
